//go:build !darwin && !windows

package service

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/amxv/noria/internal/config"
)

func configureDetached(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }
func systemdAvailable() bool {
	actual, e := user.Current()
	if e != nil {
		return false
	}
	home, e := os.UserHomeDir()
	if e != nil {
		return false
	}
	actualHome, _ := filepath.EvalSymlinks(actual.HomeDir)
	selectedHome, _ := filepath.EvalSymlinks(home)
	// The user manager cannot resolve units in an arbitrary overridden HOME.
	if actualHome != "" && selectedHome != "" && actualHome != selectedHome {
		return false
	}
	return command("systemctl", "--user", "show-environment") == nil
}
func unitPath(home string) string {
	u, _ := os.UserHomeDir()
	return filepath.Join(u, ".config", "systemd", "user", Label(home)+".service")
}
func systemdQuote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, `%`, `%%`, `$`, `$$`, `\n`, ` `).Replace(s) + `"`
}
func installUnit(home, exe string, c config.Config) error {
	u, _ := os.UserHomeDir()
	content := fmt.Sprintf("[Unit]\nDescription=Noria local account router\nAfter=network.target\n[Service]\nType=simple\nExecStart=%s __daemon --home %s\nWorkingDirectory=%s\nEnvironment=%s\nEnvironment=%s\nRestart=on-failure\nRestartSec=3\n[Install]\nWantedBy=default.target\n", systemdQuote(exe), systemdQuote(home), systemdQuote(home), systemdQuote("HOME="+u), systemdQuote("PATH="+c.ToolPath))
	if e := userFile(unitPath(home), []byte(content)); e != nil {
		return e
	}
	return command("systemctl", "--user", "daemon-reload")
}
func startPlatform(home, exe string, c config.Config) error {
	if systemdAvailable() {
		if e := installUnit(home, exe, c); e != nil {
			return e
		}
		if e := command("systemctl", "--user", "start", Label(home)+".service"); e != nil {
			return detach(home, exe, c)
		}
		return nil
	}
	return detach(home, exe, c)
}
func stopPlatform(home string) error {
	if systemdAvailable() {
		if command("systemctl", "--user", "show", Label(home)+".service") == nil {
			return command("systemctl", "--user", "stop", Label(home)+".service")
		}
		return nil
	}
	return nil
}
func loginPlatform(home, exe string, enabled bool) error {
	if !systemdAvailable() {
		return fmt.Errorf("login startup needs a systemd user session on this Linux host; start/stop still work")
	}
	c, e := config.Load(home)
	if e != nil {
		return e
	}
	if e = installUnit(home, exe, c); e != nil {
		return e
	}
	action := "disable"
	if enabled {
		action = "enable"
	}
	return command("systemctl", "--user", action, Label(home)+".service")
}

func killManagedChild(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
