package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureTrayAlreadyInstalledDoesNotDownload(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(root, "noria")
	if err := os.WriteFile(trayBinary(exe), []byte("fixture"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := ensureTrayInstalled(home, exe); err != nil {
		t.Fatal(err)
	}
}
