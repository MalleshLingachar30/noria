// Package service owns user-scoped background lifetime, independently of any UI.
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/amxv/noria/internal/config"
)

type Intent struct {
	Wanted     bool      `json:"wanted"`
	Executable string    `json:"executable"`
	Updated    time.Time `json:"updated_at"`
}

func Label(home string) string {
	s := sha256.Sum256([]byte(home))
	return "xyz.ashray.noria." + hex.EncodeToString(s[:6])
}
func Executable() (string, error) {
	p, e := os.Executable()
	if e != nil {
		return "", e
	}
	return filepath.EvalSymlinks(p)
}
func SetIntent(home, exe string, wanted bool) error {
	b, _ := json.Marshal(Intent{wanted, exe, time.Now()})
	return config.Write(filepath.Join(home, "service.json"), b)
}
func ReadIntent(home string) Intent {
	var v Intent
	b, _ := os.ReadFile(filepath.Join(home, "service.json"))
	_ = json.Unmarshal(b, &v)
	return v
}
func Guard(home, name string) (func(), error) { return config.Lock(home, name) }
func Start(home, exe string) error {
	if e := SetIntent(home, exe, true); e != nil {
		return e
	}
	c, e := config.Load(home)
	if e != nil {
		return e
	}
	// Keep only the native tool search path; never copy credentials from a shell.
	if c.ToolPath == "" {
		c.ToolPath = os.Getenv("PATH")
		if e = config.Change(home, func(current *config.Config) error { current.ToolPath = c.ToolPath; return nil }); e != nil {
			return e
		}
	}
	if e = startPlatform(home, exe, c); e != nil {
		_ = SetIntent(home, exe, false)
		return e
	}
	return nil
}
func Stop(home, exe string) error {
	if e := SetIntent(home, exe, false); e != nil {
		return e
	}
	return stopPlatform(home)
}
func Login(home, exe string, enabled bool) error { return loginPlatform(home, exe, enabled) }
func environment(home string, c config.Config) []string {
	u, _ := os.UserHomeDir()
	env := []string{}
	for _, key := range []string{"SystemRoot", "WINDIR", "TEMP", "TMP", "USERPROFILE", "LOCALAPPDATA", "APPDATA", "USER", "LOGNAME", "LANG", "LC_ALL", "XDG_RUNTIME_DIR", "DBUS_SESSION_BUS_ADDRESS"} {
		if v := os.Getenv(key); v != "" {
			env = append(env, key+"="+v)
		}
	}
	env = append(env, "HOME="+u, "NORIA_HOME="+home, "PATH="+c.ToolPath)
	return env
}
func logFile(home string) (*os.File, error) {
	dir := filepath.Join(home, "logs")
	if e := config.PrivateDir(dir); e != nil {
		return nil, e
	}
	path := filepath.Join(dir, "service.log")
	if info, e := os.Stat(path); e == nil && info.Size() > 2<<20 {
		_ = os.Rename(path, path+".1")
	}
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
}
func detach(home, exe string, c config.Config) error {
	output, e := logFile(home)
	if e != nil {
		return e
	}
	defer output.Close()
	cmd := exec.Command(exe, "__supervise", "--home", home)
	cmd.Env = environment(home, c)
	cmd.Dir = home
	cmd.Stdin = nil
	cmd.Stdout = output
	cmd.Stderr = output
	configureDetached(cmd)
	if e = cmd.Start(); e != nil {
		return e
	}
	return cmd.Process.Release()
}

// Supervise is the Linux-without-systemd/Windows backend. OS login registration
// invokes this same executable; there is no shell or detached coding agent.
func Supervise(home string) error {
	unlock, e := Guard(home, "supervisor")
	if e != nil {
		return e
	}
	defer unlock()
	exe, e := Executable()
	if e != nil {
		return e
	}
	for ReadIntent(home).Wanted {
		c, e := config.Load(home)
		if e != nil {
			return e
		}
		cmd := exec.Command(exe, "__daemon", "--home", home)
		cmd.Env = environment(home, c)
		cmd.Dir = home
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = nil
		// The supervisor inherits no TTY. Keep the child's process group independent.
		configureDetached(cmd)
		if e = cmd.Start(); e != nil {
			return e
		}
		exited := make(chan error, 1)
		go func() { exited <- cmd.Wait() }()
		ticker := time.NewTicker(150 * time.Millisecond)
		finished := false
		for !finished {
			select {
			case e = <-exited:
				finished = true
			case <-ticker.C:
				if !ReadIntent(home).Wanted {
					// Stop already requested a graceful HTTP shutdown. Enforce a
					// bound when an unhealthy child cannot receive that request.
					select {
					case <-exited:
					case <-time.After(3 * time.Second):
						killManagedChild(cmd)
						<-exited
					}
					ticker.Stop()
					return nil
				}
			}
		}
		ticker.Stop()
		if !ReadIntent(home).Wanted {
			return nil
		}
		if e != nil {
			fmt.Fprintln(os.Stderr, "Noria service restarting after an unexpected exit")
		}

		time.Sleep(2 * time.Second)
	}
	return nil
}
func command(name string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if e := cmd.Run(); e != nil {
		return fmt.Errorf("%s failed; check user service permissions: %w", name, e)
	}
	return nil
}
func quoteXML(s string) string {
	b, _ := json.Marshal(s)
	_ = b
	var buf bytesXML
	return buf.escape(s)
}

type bytesXML struct{}

func (bytesXML) escape(s string) string {
	var result string
	for _, r := range s {
		switch r {
		case '&':
			result += "&amp;"
		case '<':
			result += "&lt;"
		case '>':
			result += "&gt;"
		case '"':
			result += "&quot;"
		case '\'':
			result += "&apos;"
		default:
			result += string(r)
		}
	}
	return result
}
func userFile(path string, b []byte) error {
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	if st, e := os.Lstat(path); e == nil && !st.Mode().IsRegular() {
		return errors.New("refusing nonregular service registration")
	}
	return os.WriteFile(path, b, 0600)
}
func Backend() string {
	switch runtime.GOOS {
	case "darwin":
		return "launchd"
	case "windows":
		return "user supervisor / Task Scheduler"
	default:
		return "systemd user service / user supervisor"
	}
}
