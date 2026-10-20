package distribution

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseVersions(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want bool
	}{{"v0.2.0", "0.1.1", true}, {"v1.0.0", "1.0.0", false}, {"v0.1.0", "0.2.0", false}, {"not-a-release", "0.2.0", false}} {
		if Newer(tc.a, tc.b) != tc.want {
			t.Fatal(tc)
		}
	}
}
func TestOnlyCanonicalAssetOrigins(t *testing.T) {
	for _, u := range []string{"http://github.com/amxv/noria/releases/download/v0.2.0/file", "https://github.com/evil/repo/releases/download/v0.2.0/file", "https://example.com/file"} {
		r := Release{Tag: "v0.2.0", Assets: []Asset{{Name: "file", URL: u}}}
		if _, e := assetURL(r, "file"); e == nil {
			t.Fatal("untrusted asset URL accepted")
		}
	}
}
func TestTarRejectsPathsAndLinks(t *testing.T) {
	for _, header := range []tar.Header{{Name: "../escape", Typeflag: tar.TypeReg, Size: 1}, {Name: "/absolute", Typeflag: tar.TypeReg, Size: 1}, {Name: "\\volume-rooted", Typeflag: tar.TypeReg, Size: 1}, {Name: `C:/windows/system32/evil`, Typeflag: tar.TypeReg, Size: 1}, {Name: `Noria.app\\Contents\\evil`, Typeflag: tar.TypeReg, Size: 1}, {Name: `Noria.app/Contents/file:stream`, Typeflag: tar.TypeReg, Size: 1}, {Name: "Noria.app/link", Typeflag: tar.TypeSymlink, Linkname: "/tmp"}, {Name: "safe", Typeflag: tar.TypeLink, Linkname: "../../escape"}} {
		t.Run(header.Name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "archive.gz")
			f, _ := os.Create(path)
			gz := gzip.NewWriter(f)
			tw := tar.NewWriter(gz)
			_ = tw.WriteHeader(&header)
			if header.Size > 0 {
				_, _ = tw.Write([]byte("x"))
			}
			tw.Close()
			gz.Close()
			f.Close()
			if e := Extract(path, filepath.Join(root, "out")); e == nil {
				t.Fatal("unsafe archive accepted")
			}
		})
	}
}
func TestAtomicFileReplacementKeepsRollback(t *testing.T) {
	root := t.TempDir()
	source, target := filepath.Join(root, "new"), filepath.Join(root, "noria")
	_ = os.WriteFile(source, []byte("new"), 0755)
	_ = os.WriteFile(target, []byte("old"), 0755)
	if e := Replace(source, target, 0755); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(target)
	old, _ := os.ReadFile(target + ".previous")
	if string(b) != "new" || string(old) != "old" {
		t.Fatal("replacement lost rollback")
	}
}
func TestInvalidReleaseVersionRejectedBeforeNetwork(t *testing.T) {
	if _, e := Tagged(context.Background(), "../other"); e == nil {
		t.Fatal("unsafe tag accepted")
	}
}

func TestPublicLatestRedirectIsStrictlyValidated(t *testing.T) {
	const page = "https://github.com/amxv/noria/releases/latest"
	for _, location := range []string{
		"https://github.com/amxv/noria/releases/tag/v0.3.4",
		"/amxv/noria/releases/tag/v0.3.4",
	} {
		tag, err := latestTagFromLocation(page, location)
		if err != nil || tag != "v0.3.4" {
			t.Fatalf("valid redirect %q: %q %v", location, tag, err)
		}
	}
	for _, location := range []string{
		"", "https://evil.example/amxv/noria/releases/tag/v0.3.4",
		"https://github.com.evil.example/amxv/noria/releases/tag/v0.3.4",
		"http://github.com/amxv/noria/releases/tag/v0.3.4",
		"https://github.com/other/noria/releases/tag/v0.3.4",
		"https://github.com/amxv/noria/releases/tag/v0.3.4/extra",
		"https://github.com/amxv/noria/releases/tag/unstable",
		"https://github.com/amxv/noria/releases/tag/v0.3.4?download=1",
	} {
		if _, err := latestTagFromLocation(page, location); err == nil {
			t.Errorf("accepted untrusted latest redirect: %q", location)
		}
	}
}

func TestPublicReleaseAssetsStayCanonical(t *testing.T) {
	r := releaseForTag("v0.3.4")
	if r.Tag != "v0.3.4" || r.Draft || r.Prerelease || len(r.Assets) != 3 {
		t.Fatalf("unexpected release metadata: %+v", r)
	}
	for _, name := range []string{"SHA256SUMS", BinaryName(), ArchiveName()} {
		url, err := assetURL(r, name)
		if err != nil || url != "https://github.com/amxv/noria/releases/download/v0.3.4/"+name {
			t.Fatalf("invalid canonical asset URL %s: %s %v", name, url, err)
		}
		if strings.Contains(url, "api.github.com") {
			t.Fatal("release asset unexpectedly depends on REST API")
		}
	}
}
