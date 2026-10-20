// Package distribution verifies release assets before installing executable code.
package distribution

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}
type Release struct {
	Tag        string  `json:"tag_name"`
	Draft      bool    `json:"draft"`
	Prerelease bool    `json:"prerelease"`
	Assets     []Asset `json:"assets"`
}

var validVersion = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)$`)

func Newer(target, current string) bool {
	a := validVersion.FindStringSubmatch(target)
	b := validVersion.FindStringSubmatch(current)
	if a == nil {
		return false
	}
	if b == nil {
		return true
	}
	for i := 1; i <= 3; i++ {
		x, _ := strconv.Atoi(a[i])
		y, _ := strconv.Atoi(b[i])
		if x != y {
			return x > y
		}
	}
	return false
}
func Latest(ctx context.Context) (Release, error) {
	// GitHub's anonymous REST API has a shared per-IP rate limit. CI runners
	// and machines behind NAT can receive HTTP 403 even for public releases.
	// The public /releases/latest redirect is not subject to the REST quota.
	// We still checksum-verify every asset before executing it.
	tag, err := latestTagByRedirect(ctx)
	if err == nil {
		return releaseForTag(tag), nil
	}
	release, apiErr := Fetch(ctx, "https://api.github.com/repos/amxv/noria/releases/latest")
	if apiErr != nil {
		return Release{}, fmt.Errorf("could not locate latest release (GitHub redirect: %v; API: %w)", err, apiErr)
	}
	return release, nil
}
func Tagged(ctx context.Context, version string) (Release, error) {
	if !validVersion.MatchString(version) {
		return Release{}, errors.New("invalid release version")
	}
	tag := "v" + strings.TrimPrefix(version, "v")
	// The checksum and requested archive provide stronger validation than
	// relying on the rate-limited release-metadata API for known tags.
	return releaseForTag(tag), nil
}

// releaseForTag produces only the fixed, known asset URLs on our GitHub
// repository. It never accepts download links from a redirect response.
func releaseForTag(tag string) Release {
	base := "https://github.com/amxv/noria/releases/download/" + tag + "/"
	names := []string{"SHA256SUMS", BinaryName(), ArchiveName()}
	assets := make([]Asset, 0, len(names))
	for _, name := range names {
		assets = append(assets, Asset{Name: name, URL: base + name})
	}
	return Release{Tag: tag, Assets: assets}
}

func latestTagFromLocation(base, location string) (string, error) {
	page, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	ref, err := url.Parse(location)
	if err != nil {
		return "", err
	}
	u := page.ResolveReference(ref)
	const prefix = "/amxv/noria/releases/tag/"
	if u.Scheme != "https" || u.Host != "github.com" || u.User != nil ||
		u.RawQuery != "" || u.Fragment != "" || !strings.HasPrefix(u.Path, prefix) {
		return "", errors.New("untrusted latest-release redirect")
	}
	tag := strings.TrimPrefix(u.Path, prefix)
	if !strings.HasPrefix(tag, "v") || !validVersion.MatchString(tag) {
		return "", errors.New("invalid latest release tag")
	}
	return tag, nil
}

func latestTagByRedirect(ctx context.Context) (string, error) {
	const page = "https://github.com/amxv/noria/releases/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, page, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Noria-updater")
	client := &http.Client{
		Timeout:       20 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusMovedPermanently &&
		resp.StatusCode != http.StatusTemporaryRedirect && resp.StatusCode != http.StatusPermanentRedirect {
		return "", fmt.Errorf("latest-release page returned HTTP %d", resp.StatusCode)
	}
	return latestTagFromLocation(page, resp.Header.Get("Location"))
}
func Fetch(ctx context.Context, endpoint string) (Release, error) {
	var r Release
	b, e := get(ctx, endpoint, 2<<20)
	if e != nil {
		return r, e
	}
	if e = json.Unmarshal(b, &r); e != nil {
		return r, e
	}
	if r.Draft || r.Prerelease || !validVersion.MatchString(r.Tag) {
		return r, errors.New("release is not a stable published version")
	}
	return r, nil
}
func assetURL(r Release, name string) (string, error) {
	for _, a := range r.Assets {
		if a.Name == name {
			u, e := url.Parse(a.URL)
			if e != nil || u.Scheme != "https" || u.Host != "github.com" || !strings.HasPrefix(u.Path, "/amxv/noria/releases/download/"+r.Tag+"/") {
				return "", errors.New("untrusted release download origin")
			}
			return a.URL, nil
		}
	}
	return "", fmt.Errorf("release %s is missing %s", r.Tag, name)
}
func get(ctx context.Context, address string, limit int64) ([]byte, error) {
	if !trustedDownloadURL(address) {
		return nil, errors.New("untrusted release endpoint")
	}
	request, e := http.NewRequestWithContext(ctx, "GET", address, nil)
	if e != nil {
		return nil, e
	}
	request.Header.Set("User-Agent", "Noria-updater")
	request.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 90 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 5 || !trustedDownloadURL(req.URL.String()) {
			return errors.New("unsafe release redirect")
		}
		return nil
	}}
	response, e := client.Do(request)
	if e != nil {
		return nil, e
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("release server returned HTTP %d", response.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if e != nil {
		return nil, e
	}
	if int64(len(b)) > limit {
		return nil, errors.New("release asset exceeds size limit")
	}
	return b, nil
}
func BinaryName() string {
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	return "noria_" + runtime.GOOS + "_" + runtime.GOARCH + suffix
}
func ArchiveName() string { return "noria_" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz" }
func Download(ctx context.Context, r Release, name, dest string) error {
	url, e := assetURL(r, "SHA256SUMS")
	if e != nil {
		return e
	}
	sums, e := get(ctx, url, 1<<20)
	if e != nil {
		return e
	}
	checksum := ""
	for _, line := range strings.Split(string(sums), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 && strings.TrimPrefix(parts[1], "*") == name {
			if checksum != "" {
				return errors.New("duplicate checksum entry")
			}
			checksum = parts[0]
		}
	}
	if len(checksum) != 64 {
		return errors.New("release checksum missing")
	}
	if _, e = hex.DecodeString(checksum); e != nil {
		return errors.New("invalid checksum")
	}
	url, e = assetURL(r, name)
	if e != nil {
		return e
	}
	b, e := get(ctx, url, 128<<20)
	if e != nil {
		return e
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != strings.ToLower(checksum) {
		return errors.New("release checksum mismatch")
	}
	return os.WriteFile(dest, b, 0600)
}

// Extract permits regular files/directories only, and rejects symlink traversal.
func Extract(archive, dest string) error {
	file, e := os.Open(archive)
	if e != nil {
		return e
	}
	defer file.Close()
	gzipReader, e := gzip.NewReader(file)
	if e != nil {
		return e
	}
	defer gzipReader.Close()
	reader := tar.NewReader(gzipReader)
	var size int64
	for count := 0; count < 2048; count++ {
		header, e := reader.Next()
		if e == io.EOF {
			return nil
		}
		if e != nil {
			return e
		}
		// Tar names use POSIX '/' separators on *every* target OS. filepath.IsAbs
		// alone is insufficient on Windows: it does not consider a volume-rooted
		// path such as "/absolute" fully qualified, even though it must never be
		// accepted from an archive. Reject Windows drive/ADS syntax and backslashes
		// as well so extraction has the same safety rules on every platform.
		raw := header.Name
		clean := path.Clean(raw)
		if path.IsAbs(raw) || clean == ".." || strings.HasPrefix(clean, "../") ||
			strings.ContainsAny(raw, "\\:") || strings.HasPrefix(raw, "\\") ||
			strings.HasPrefix(raw, "//") || strings.IndexByte(raw, 0) >= 0 {
			return errors.New("unsafe release archive path")
		}
		name := filepath.FromSlash(clean)
		if filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
			return errors.New("unsafe release archive path")
		}
		target := filepath.Join(dest, name)
		relative, e := filepath.Rel(dest, target)
		if e != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return errors.New("archive escapes target")
		}
		if header.Size < 0 || header.Size > 128<<20 {
			return errors.New("archive file too large")
		}
		size += header.Size
		if size > 256<<20 {
			return errors.New("archive exceeds extracted size limit")
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if e = os.MkdirAll(target, 0755); e != nil {
				return e
			}
		case tar.TypeReg, tar.TypeRegA:
			if e = os.MkdirAll(filepath.Dir(target), 0755); e != nil {
				return e
			}
			mode := os.FileMode(0644)
			if header.Mode&0111 != 0 {
				mode = 0755
			}
			f, e := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
			if e != nil {
				return e
			}
			_, e = io.CopyN(f, reader, header.Size)
			ce := f.Close()
			if e != nil {
				return e
			}
			if ce != nil {
				return ce
			}
		default:
			return errors.New("release archive contains a link or unsupported entry")
		}
	}
	return errors.New("too many archive entries")
}

// Replace stages on the destination filesystem, then renames; rollback restores
// the previous executable if the final move fails (including Windows locks).
func Replace(source, target string, mode os.FileMode) error {
	input, e := os.Open(source)
	if e != nil {
		return e
	}
	defer input.Close()
	stage, e := os.CreateTemp(filepath.Dir(target), ".noria-update-*")
	if e != nil {
		return e
	}
	defer os.Remove(stage.Name())
	defer stage.Close()
	if e = stage.Chmod(mode); e != nil {
		return e
	}
	if _, e = io.Copy(stage, input); e != nil {
		return e
	}
	if e = stage.Sync(); e != nil {
		return e
	}
	if e = stage.Close(); e != nil {
		return e
	}
	previous := target + ".previous"
	_ = os.Remove(previous)
	had := false
	if _, e = os.Stat(target); e == nil {
		if e = os.Rename(target, previous); e != nil {
			return fmt.Errorf("cannot replace running binary: %w", e)
		}
		had = true
	}
	if e = os.Rename(stage.Name(), target); e != nil {
		if had {
			_ = os.Rename(previous, target)
		}
		return e
	}
	return nil
}

func trustedDownloadURL(address string) bool {
	u, e := url.Parse(address)
	if e != nil || u.Scheme != "https" || u.User != nil {
		return false
	}
	switch u.Hostname() {
	case "github.com", "api.github.com", "objects.githubusercontent.com", "release-assets.githubusercontent.com":
		return true
	}
	return false
}

// Swap retains a rollback copy until the caller commits the entire CLI/UI pair.
// Both copies live beside the installed executable, including cross-volume installs.
func Swap(source, target string, mode os.FileMode) (rollback func() error, finish func(), err error) {
	input, e := os.Open(source)
	if e != nil {
		return nil, nil, e
	}
	defer input.Close()
	stage, e := os.CreateTemp(filepath.Dir(target), ".noria-swap-*")
	if e != nil {
		return nil, nil, e
	}
	name := stage.Name()
	defer os.Remove(name)
	if e = stage.Chmod(mode); e == nil {
		_, e = io.Copy(stage, input)
	}
	if e == nil {
		e = stage.Sync()
	}
	if ce := stage.Close(); e == nil {
		e = ce
	}
	if e != nil {
		return nil, nil, e
	}
	backup := name + ".previous"
	had := false
	if info, e := os.Lstat(target); e == nil {
		if !info.Mode().IsRegular() {
			return nil, nil, errors.New("installed binary is not a regular file")
		}
		if e = os.Rename(target, backup); e != nil {
			return nil, nil, e
		}
		had = true
	} else if !os.IsNotExist(e) {
		return nil, nil, e
	}
	if e = os.Rename(name, target); e != nil {
		if had {
			_ = os.Rename(backup, target)
		}
		return nil, nil, e
	}
	rollback = func() error {
		if e := os.Remove(target); e != nil && !os.IsNotExist(e) {
			return e
		}
		if had {
			return os.Rename(backup, target)
		}
		return nil
	}
	finish = func() {
		if had {
			_ = os.Remove(backup)
		}
	}
	return rollback, finish, nil
}
