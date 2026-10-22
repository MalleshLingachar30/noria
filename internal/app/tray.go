package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
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
		if runtime.GOOS == "linux" && (os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" ||
			(os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "")) {
			return errors.New("the tray requires a graphical desktop session and a D-Bus StatusNotifierItem host")
		}
		binary := trayBinary(exe)
		if info, err := os.Stat(binary); err != nil || !info.Mode().IsRegular() {
			return errors.New("Noria tray is not installed beside the CLI; install or upgrade Noria")
		}
		cmd := exec.Command(binary, "--home", home)
		cmd.Dir = home
		cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, io.Discard, io.Discard
		host.Detach(cmd)
		if err := cmd.Start(); err != nil {
			return err
		}
		if err := cmd.Process.Release(); err != nil {
			return err
		}
		fmt.Fprintln(out, "Opened Noria tray")
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
