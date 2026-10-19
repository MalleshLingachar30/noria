//go:build darwin

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/amxv/noria/internal/config"
)

func configureDetached(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }
func launchPlist(home, exe string, c config.Config) []byte {
	u, _ := os.UserHomeDir()
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict><key>Label</key><string>%s</string>
<key>ProgramArguments</key><array><string>%s</string><string>__daemon</string><string>--home</string><string>%s</string></array>
<key>RunAtLoad</key><true/><key>KeepAlive</key><true/><key>ThrottleInterval</key><integer>3</integer>
<key>ProcessType</key><string>Interactive</string><key>WorkingDirectory</key><string>%s</string>
<key>EnvironmentVariables</key><dict><key>HOME</key><string>%s</string><key>NORIA_HOME</key><string>%s</string><key>PATH</key><string>%s</string></dict>
<key>StandardOutPath</key><string>%s</string><key>StandardErrorPath</key><string>%s</string></dict></plist>
`, Label(home), quoteXML(exe), quoteXML(home), quoteXML(home), quoteXML(u), quoteXML(home), quoteXML(c.ToolPath), quoteXML(filepath.Join(home, "logs", "service.log")), quoteXML(filepath.Join(home, "logs", "service.log"))))
}
func domain() string { return "gui/" + strconv.Itoa(os.Getuid()) }
func startPlatform(home, exe string, c config.Config) error {
	if command("/bin/launchctl", "print", domain()) != nil {
		return detach(home, exe, c)
	}
	f, e := logFile(home)
	if e != nil {
		return e
	}
	f.Close()
	p := filepath.Join(home, "service.plist")
	if e = config.Write(p, launchPlist(home, exe, c)); e != nil {
		return e
	}
	if e = command("/bin/launchctl", "print", domain()+"/"+Label(home)); e == nil {
		return command("/bin/launchctl", "kickstart", domain()+"/"+Label(home))
	}
	return command("/bin/launchctl", "bootstrap", domain(), p)
}
func stopPlatform(home string) error {
	if e := command("/bin/launchctl", "print", domain()+"/"+Label(home)); e != nil {
		return nil
	}
	return command("/bin/launchctl", "bootout", domain()+"/"+Label(home))
}
func loginPlatform(home, exe string, enabled bool) error {
	u, e := os.UserHomeDir()
	if e != nil {
		return e
	}
	path := filepath.Join(u, "Library", "LaunchAgents", Label(home)+".plist")
	if !enabled {
		if e = os.Remove(path); e != nil && !os.IsNotExist(e) {
			return e
		}
		return nil
	}
	c, e := config.Load(home)
	if e != nil {
		return e
	}
	f, e := logFile(home)
	if e != nil {
		return e
	}
	f.Close()
	return userFile(path, launchPlist(home, exe, c))
}

func killManagedChild(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
