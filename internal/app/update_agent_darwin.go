//go:build darwin

package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/amxv/noria/internal/config"
	"github.com/amxv/noria/internal/service"
)

// launchd runs updates independently of the menu bar and router. The CLI is
// still the only owner of preferences, signature checks, and safe installation.
const updateIntervalSeconds = 3600

func updateAgentLabel(home string) string { return service.Label(home) + ".update" }

func updateAgentPath(userHome, home string) string {
	return filepath.Join(userHome, "Library", "LaunchAgents", updateAgentLabel(home)+".plist")
}

func updateAgentPlist(home, exe, userHome string, c config.Config) []byte {
	esc := func(s string) string { return htmlEscape(s) }
	log := filepath.Join(home, "logs", "updates.log")
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>%s</string>
<key>ProgramArguments</key><array><string>%s</string><string>upgrade</string><string>--automatic</string><string>--json</string><string>--home</string><string>%s</string></array>
<key>RunAtLoad</key><true/>
<key>StartInterval</key><integer>%d</integer>
<key>EnvironmentVariables</key><dict><key>HOME</key><string>%s</string><key>NORIA_HOME</key><string>%s</string><key>PATH</key><string>%s</string></dict>
<key>StandardOutPath</key><string>%s</string><key>StandardErrorPath</key><string>%s</string>
</dict></plist>
`, esc(updateAgentLabel(home)), esc(exe), esc(home), updateIntervalSeconds,
		esc(userHome), esc(home), esc(c.ToolPath), esc(log), esc(log)))
}

func writeUpdatePlist(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("refusing nonregular update agent registration")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".noria-update-agent-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

// Commands are injected into the low-level reconciler to test login behavior
// without modifying the developer's real launchd jobs.
type launchctlRunner func(args ...string) error

func systemLaunchctl(args ...string) error {
	cmd := exec.Command("/bin/launchctl", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl %s: %w: %s", args[0], err, strings.TrimSpace(string(output)))
	}
	return nil
}

func reconcileUpdateAgent(home, exe, userHome string, c config.Config, run launchctlRunner) error {
	path := updateAgentPath(userHome, home)
	domain := "gui/" + strconv.Itoa(os.Getuid())
	label := domain + "/" + updateAgentLabel(home)
	loaded := run("print", label) == nil
	if !c.AutoUpdate {
		// Persist the disabled preference before this call. Even if an update
		// is underway, its next invocation will exit without installing.
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		if loaded {
			// launchctl bootout kills a running job. Never interrupt an
			// updater midway through a verified binary/app swap. The disabled
			// preference makes any later invocation a harmless no-op, and the
			// removed plist prevents the job from returning at login.
			unlock, err := service.Guard(home, "upgrade")
			if err != nil {
				return nil
			}
			defer unlock()
			return run("bootout", label)
		}
		return nil
	}
	if !filepath.IsAbs(exe) {
		return fmt.Errorf("updater executable path must be absolute")
	}
	if err := config.PrivateDir(filepath.Join(home, "logs")); err != nil {
		return err
	}
	if err := writeUpdatePlist(path, updateAgentPlist(home, exe, userHome, c)); err != nil {
		return err
	}
	if loaded {
		return nil // An update may be running; don't kill it just to refresh its plist.
	}
	// A headless SSH session has no GUI launchd domain; the saved plist will
	// be loaded at the next GUI login. Menu bar sessions always have one.
	if run("print", domain) != nil {
		return nil
	}
	return run("bootstrap", domain, path)
}

func syncUpdateAgent(home, exe string) error {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	c, err := config.Load(home)
	if err != nil {
		return err
	}
	return reconcileUpdateAgent(home, exe, userHome, c, systemLaunchctl)
}
