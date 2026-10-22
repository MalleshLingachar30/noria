package tray

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type CLI struct {
	Home string
	Exe  string
}

func newCLI(home string) (CLI, error) {
	if home == "" {
		home = os.Getenv("NORIA_HOME")
	}
	if home == "" {
		user, err := os.UserHomeDir()
		if err != nil {
			return CLI{}, err
		}
		home = filepath.Join(user, ".noria")
	}
	var err error
	home, err = filepath.Abs(home)
	if err != nil {
		return CLI{}, err
	}
	exe, err := os.Executable()
	if err != nil {
		return CLI{}, err
	}
	name := "noria"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	exe = filepath.Join(filepath.Dir(exe), name)
	if info, err := os.Stat(exe); err != nil || !info.Mode().IsRegular() {
		return CLI{}, fmt.Errorf("Noria CLI is not installed beside the tray executable")
	}
	return CLI{Home: home, Exe: exe}, nil
}

// Enforce a bounded command lifetime and no stdin; CLI remains the sole source
// of settings and secrets. Keep launch-environment PATH usable outside shells.
func (c CLI) run(timeout time.Duration, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.runContext(ctx, args...)
}

func (c CLI) runContext(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, c.Exe, append(args, "--home", c.Home)...)
	cmd.Dir = c.Home
	cmd.Stdin = nil
	cmd.Env = append(os.Environ(), "PATH="+guiPath())
	hideConsole(cmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &limitWriter{Buffer: &stdout, Max: 4 << 20}, &limitWriter{Buffer: &stderr, Max: 32 << 10}
	err := cmd.Run()
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("Noria command timed out")
		}
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		if len(detail) > 500 {
			detail = detail[:500]
		}
		return nil, fmt.Errorf("%s", detail)
	}
	return stdout.Bytes(), nil
}

func guiPath() string {
	user, _ := os.UserHomeDir()
	env := os.Getenv("PATH")
	entries := []string{filepath.Join(user, ".local", "bin"), filepath.Join(user, ".bun", "bin"), filepath.Join(user, ".cargo", "bin")}
	if runtime.GOOS == "windows" {
		entries = append(entries, filepath.Join(os.Getenv("APPDATA"), "npm"), filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "nodejs"))
	} else {
		entries = append(entries, "/usr/local/bin", "/usr/bin", "/bin")
	}
	return strings.Join(append(entries, env), string(os.PathListSeparator))
}

type limitWriter struct {
	Buffer *bytes.Buffer
	Max    int
}

func (w *limitWriter) Write(p []byte) (int, error) {
	if w.Buffer.Len() < w.Max {
		_, _ = w.Buffer.Write(p[:min(len(p), w.Max-w.Buffer.Len())])
	}
	return len(p), nil
}
