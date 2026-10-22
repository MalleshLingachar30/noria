//go:build linux

package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTrayAutostartUsesIsolatedHomeAndQuotedExec(t *testing.T) {
	userHome := t.TempDir()
	t.Setenv("HOME", userHome)
	t.Setenv("XDG_CONFIG_HOME", "")
	install := filepath.Join(userHome, "bin with spaces")
	if err := os.MkdirAll(install, 0700); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(install, "noria")
	tray := filepath.Join(install, "noria-tray")
	if err := os.WriteFile(tray, []byte("fixture"), 0700); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(userHome, "noria % custom")
	if err := trayLogin(home, exe, true); err != nil {
		t.Fatal(err)
	}
	startup := filepath.Join(userHome, ".config", "autostart", "noria-tray.desktop")
	b, err := os.ReadFile(startup)
	if err != nil {
		t.Fatal(err)
	}
	content := string(b)
	if !strings.Contains(content, `Exec="`+tray+`" --home "`+strings.ReplaceAll(home, "%", "%%")+`"`) {
		t.Fatalf("unsafe or incorrect XDG Exec line: %s", content)
	}
	if err := trayLogin(home, exe, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(startup); !os.IsNotExist(err) {
		t.Fatalf("autostart entry still present: %v", err)
	}
}

func TestTrayAutostartRequiresInstalledExecutable(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	if err := trayLogin(t.TempDir(), filepath.Join(t.TempDir(), "noria"), true); err == nil {
		t.Fatal("autostart accepted a missing tray executable")
	}
}

func TestTrayAutostartHonorsXDGConfigHome(t *testing.T) {
	userHome := t.TempDir()
	t.Setenv("HOME", userHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(userHome, "custom-config"))
	exe := filepath.Join(userHome, "noria")
	if err := os.WriteFile(filepath.Join(userHome, "noria-tray"), []byte("fixture"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := trayLogin(filepath.Join(userHome, "state"), exe, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(userHome, "custom-config", "autostart", "noria-tray.desktop")); err != nil {
		t.Fatal(err)
	}
}
