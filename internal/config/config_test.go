package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/amxv/noria/internal/model"
)

func TestPrivatePersistentConfiguration(t *testing.T) {
	h := t.TempDir()
	if e := Init(h); e != nil {
		t.Fatal(e)
	}
	s, e := Secret(h)
	if e != nil {
		t.Fatal(e)
	}
	if e = Init(h); e != nil {
		t.Fatal(e)
	}
	s2, _ := Secret(h)
	if s != s2 {
		t.Fatal("init rotated existing secret")
	}
	c, e := Load(h)
	if e != nil {
		t.Fatal(e)
	}
	if c.Listen != DefaultListen {
		t.Fatal(c.Listen)
	}
	for _, path := range []string{h, filepath.Join(h, "config.json"), filepath.Join(h, "secret")} {
		st, e := os.Stat(path)
		if e != nil {
			t.Fatal(e)
		}
		if runtime.GOOS != "windows" && st.Mode().Perm()&0077 != 0 {
			t.Fatalf("unsafe mode %s", path)
		}
	}
	c.Accounts = []model.Account{{ID: "safe", Provider: "codex", Profile: "/outside", Enabled: true}}
	if e = Save(h, c); e != nil {
		t.Fatal(e)
	}
	if _, e = Load(h); e == nil {
		t.Fatal("accepted external account profile")
	}
}

func TestRejectTraversalAndRemoteListen(t *testing.T) {
	for _, v := range []string{"../evil", ".", "/absolute", "a/b", ""} {
		if ValidateID(v) == nil {
			t.Fatalf("accepted %q", v)
		}
	}
	for _, v := range []string{"0.0.0.0:43119", "localhost:43119", "example.com:80", ":43119"} {
		if ValidateListen(v) == nil {
			t.Fatalf("accepted %q", v)
		}
	}
	for _, v := range []string{"127.0.0.1:43119", "[::1]:43119"} {
		if e := ValidateListen(v); e != nil {
			t.Fatal(e)
		}
	}
}

func TestRefuseSymlinkAtomicTargetDirectory(t *testing.T) {
	h := t.TempDir()
	target := t.TempDir()
	p := filepath.Join(h, "linked")
	if e := os.Symlink(target, p); e != nil {
		t.Skip(e)
	}
	if e := Write(filepath.Join(p, "secret"), []byte("x")); e == nil {
		t.Fatal("followed symlink private directory")
	}
}
