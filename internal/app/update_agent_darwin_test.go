//go:build darwin

package app

import (
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amxv/noria/internal/config"
)

type fakeLaunchd struct {
	loaded bool
	gui    bool
	calls  []string
}

func (f *fakeLaunchd) run(args ...string) error {
	f.calls = append(f.calls, strings.Join(args, " "))
	switch args[0] {
	case "print":
		if strings.Contains(args[1], "/xyz.ashray.noria.") && strings.HasSuffix(args[1], ".update") {
			if f.loaded {
				return nil
			}
			return errors.New("not loaded")
		}
		if f.gui {
			return nil
		}
		return errors.New("no GUI domain")
	case "bootstrap":
		f.loaded = true
	case "bootout":
		f.loaded = false
	}
	return nil
}

func TestMacAutomaticUpdateAgentLifecycle(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "noria & <private>")
	userHome := filepath.Join(root, "user home")
	exe := filepath.Join(root, "install & tools", "noria")
	settings := config.Config{AutoUpdate: true, ToolPath: "/usr/local/bin:/usr/bin"}
	launchd := &fakeLaunchd{gui: true}

	if err := reconcileUpdateAgent(home, exe, userHome, settings, launchd.run); err != nil {
		t.Fatal(err)
	}
	path := updateAgentPath(userHome, home)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var plist any
	if err := xml.Unmarshal(data, &plist); err != nil {
		t.Fatalf("invalid plist XML: %v", err)
	}
	for _, required := range []string{"<key>RunAtLoad</key><true/>", "<key>StartInterval</key><integer>3600</integer>",
		"<string>upgrade</string><string>--automatic</string><string>--json</string>", "noria &amp; &lt;private&gt;", "install &amp; tools"} {
		if !strings.Contains(string(data), required) {
			t.Fatalf("missing %q in update agent plist: %s", required, data)
		}
	}
	if len(launchd.calls) != 3 || !strings.HasPrefix(launchd.calls[2], "bootstrap gui/") {
		t.Fatalf("expected a single bootstrap, got %v", launchd.calls)
	}
	if err := reconcileUpdateAgent(home, exe, userHome, settings, launchd.run); err != nil {
		t.Fatal(err)
	}
	if len(launchd.calls) != 4 || strings.Contains(launchd.calls[3], "bootout") {
		t.Fatalf("reconciliation must not interrupt a loaded updater: %v", launchd.calls)
	}

	settings.AutoUpdate = false
	if err := reconcileUpdateAgent(home, exe, userHome, settings, launchd.run); err != nil {
		t.Fatal(err)
	}
	if launchd.loaded {
		t.Fatal("disabled updater remains loaded")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("disabled updater plist remains: %v", err)
	}
	if err := reconcileUpdateAgent(home, exe, userHome, settings, launchd.run); err != nil {
		t.Fatal(err)
	}
	settings.AutoUpdate = true
	if err := reconcileUpdateAgent(home, exe, userHome, settings, launchd.run); err != nil || !launchd.loaded {
		t.Fatalf("reenabling automatic updates failed: %v", err)
	}
}

func TestMacAutomaticUpdatesOfflineAndUntrustedPlist(t *testing.T) {
	root := t.TempDir()
	home, userHome := filepath.Join(root, "pool"), filepath.Join(root, "user")
	exe := filepath.Join(root, "noria")
	c := config.Config{AutoUpdate: true, ToolPath: "/usr/bin"}
	launchd := &fakeLaunchd{gui: false}
	if err := reconcileUpdateAgent(home, exe, userHome, c, launchd.run); err != nil {
		t.Fatal(err)
	}
	if launchd.loaded {
		t.Fatal("headless updates must wait for GUI login")
	}
	if _, err := os.Stat(updateAgentPath(userHome, home)); err != nil {
		t.Fatalf("login registration must persist for next login: %v", err)
	}
	launchd.gui = true
	if err := reconcileUpdateAgent(home, exe, userHome, c, launchd.run); err != nil || !launchd.loaded {
		t.Fatalf("updater not loaded when GUI session became available: %v", err)
	}
	launchd.loaded = false
	path := updateAgentPath(userHome, home)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "private-file"), path); err != nil {
		t.Fatal(err)
	}
	if err := reconcileUpdateAgent(home, exe, userHome, c, launchd.run); err == nil {
		t.Fatal("must reject a symlink in LaunchAgents")
	}
}
