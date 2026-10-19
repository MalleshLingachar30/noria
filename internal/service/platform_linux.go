//go:build !darwin && !windows

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/amxv/noria/internal/config"
)

func configureDetached(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }
func systemdAvailable() bool          { return command("systemctl", "--user", "show-environment") == nil }
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
		return command("systemctl", "--user", "start", Label(home)+".service")
	}
	return detach(home, exe, c)
}
func stopPlatform(home string) error {
	if systemdAvailable() {
		return command("systemctl", "--user", "stop", Label(home)+".service")
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
