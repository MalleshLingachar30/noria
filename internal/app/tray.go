package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/amxv/noria/internal/config"
	"github.com/amxv/noria/internal/host"
	"github.com/amxv/noria/internal/service"
)

type trayControl struct {
	Action      string    `json:"action"`
	RequestedAt time.Time `json:"requested_at"`
}

func trayBinary(exe string) string {
	name := "noria-tray"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(filepath.Dir(exe), name)
}

func trayCommand(home string, args []string, out io.Writer) error {
	if len(args) == 0 || isHelp(args[0]) {
		fmt.Fprintln(out, "Usage: noria tray open|quit|reopen\nWindows/Linux desktop tray wrapper. Quitting the tray does not stop the Noria router.")
		return nil
	}
	if runtime.GOOS == "darwin" {
		return errors.New("use noria menubar on macOS")
	}
	if err := config.Init(home); err != nil {
		return err
	}
	exe, err := service.Executable()
	if err != nil {
		return err
	}
	switch args[0] {
	case "open":
		if len(args) != 1 {
			return errors.New("usage: noria tray open")
		}
		env, err := desktopTrayEnvironment()
		if err != nil {
			return err
		}
		binary := trayBinary(exe)
		if err := ensureTrayInstalled(home, exe); err != nil {
			return err
		}
		// A missing GNOME tray extension should not prevent Noria from running:
		// gogpu/systray will register as soon as a tray host appears. Warn instead
		// of claiming the icon is visible when no host is currently available.
		checkCtx, checkCancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer checkCancel()
		check := exec.CommandContext(checkCtx, binary, "--check-host")
		check.Env = env
		if note, err := check.CombinedOutput(); err != nil {
			detail := strings.TrimSpace(string(note))
			if detail == "" {
				detail = "the desktop's status icon service did not respond"
			}
			fmt.Fprintln(out, "Desktop tray host not ready:", detail)
		}
		cmd := exec.Command(binary, "--home", home)
		cmd.Dir = home
		cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, io.Discard, io.Discard
		cmd.Env = env
		host.Detach(cmd)
		if err := cmd.Start(); err != nil {
			return err
		}
		if err := cmd.Process.Release(); err != nil {
			return err
		}
		fmt.Fprintln(out, "Started Noria tray")
		return nil
	case "quit":
		if len(args) != 1 {
			return errors.New("usage: noria tray quit")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		unlock, busyErr := config.LockContext(ctx, home, "tray")
		cancel()
		if busyErr == nil {
			unlock()
			fmt.Fprintln(out, "Noria tray is not running (router unchanged)")
			return nil
		}
		if !strings.Contains(busyErr.Error(), "tray is busy") {
			return busyErr
		}
		b, err := json.Marshal(trayControl{Action: "quit", RequestedAt: time.Now()})
		if err != nil {
			return err
		}
		if err := config.Write(filepath.Join(home, "tray-control.json"), b); err != nil {
			return err
		}
		fmt.Fprintln(out, "Requested Noria tray exit (router keeps running)")
		return nil
	case "reopen":
		if len(args) != 1 {
			return errors.New("usage: noria tray reopen")
		}
		cmd := exec.Command(exe, "__tray-reopen", "--home", home)
		cmd.Dir = home
		cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, io.Discard, io.Discard
		host.Detach(cmd)
		if err := cmd.Start(); err != nil {
			return err
		}
		return cmd.Process.Release()
	default:
		return errors.New("unknown tray command")
	}
}

// The launcher waits for the previous tray's OS lock instead of sleeping for
// an arbitrary time. This also avoids two tray icons during a GUI upgrade.
func delayedTrayHome(home string, args []string) error {
	if runtime.GOOS == "darwin" || len(args) != 0 {
		return errors.New("invalid tray relaunch")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	unlock, err := config.LockContext(ctx, home, "tray")
	if err != nil {
		return err
	}
	unlock()
	return trayCommand(home, []string{"open"}, io.Discard)
}
