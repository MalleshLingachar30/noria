package distribution

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionOrdering(t *testing.T) {
	for _, v := range []struct {
		target, current string
		want            bool
	}{{"v0.2.0", "0.1.1", true}, {"0.2.0", "0.2.0", false}, {"0.1.0", "0.2.0", false}, {"v1.0.0", "0.10.0", true}, {"v0.2.1-evil", "0.2.0", false}} {
		if Newer(v.target, v.current) != v.want {
			t.Fatal(v)
		}
	}
}
func TestReleaseOriginPinned(t *testing.T) {
	r := Release{Tag: "v0.2.0", Assets: []Asset{{Name: "SHA256SUMS", URL: "https://evil.example/sums"}}}
	if _, e := assetURL(r, "SHA256SUMS"); e == nil {
		t.Fatal("untrusted checksum origin")
	}
}
func TestArchiveRefusesTraversalAndLinks(t *testing.T) {
	for _, h := range []*tar.Header{{Name: "../outside", Typeflag: tar.TypeReg, Size: 1}, {Name: "Noria.app/evil", Typeflag: tar.TypeSymlink, Linkname: "/tmp/elsewhere"}} {
		dir := t.TempDir()
		archive := filepath.Join(dir, "asset.tar.gz")
		f, _ := os.Create(archive)
		gz := gzip.NewWriter(f)
		tw := tar.NewWriter(gz)
		_ = tw.WriteHeader(h)
		if h.Size > 0 {
			_, _ = tw.Write([]byte("x"))
		}
		tw.Close()
		gz.Close()
		f.Close()
		if e := Extract(archive, filepath.Join(dir, "out")); e == nil {
			t.Fatal("unsafe archive accepted")
		}
	}
}
func TestFileReplacementStagesBeforeChangingTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "noria")
	_ = os.WriteFile(target, []byte("old"), 0755)
	if e := Replace(filepath.Join(dir, "missing"), target, 0755); e == nil {
		t.Fatal("missing source succeeded")
	}
	b, _ := os.ReadFile(target)
	if string(b) != "old" {
		t.Fatal("failed update changed target")
	}
	source := filepath.Join(dir, "new")
	_ = os.WriteFile(source, []byte("new"), 0755)
	if e := Replace(source, target, 0755); e != nil {
		t.Fatal(e)
	}
	b, _ = os.ReadFile(target)
	if strings.TrimSpace(string(b)) != "new" {
		t.Fatal("replacement missing")
	}
}
