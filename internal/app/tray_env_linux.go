//go:build linux

package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// A desktop launched via a user service does not always inherit the session
// bus address. Use the standard per-user socket when the address is missing.
// Never synthesize a graphical session on an SSH/headless machine.
func desktopTrayEnvironment() ([]string, error) {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return nil, errors.New("a graphical Linux desktop session is required for the tray")
	}
	env := os.Environ()
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") != "" {
		return env, nil
	}
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" || !filepath.IsAbs(runtimeDir) {
		return nil, errors.New("a desktop session D-Bus connection is required to show the tray")
	}
	bus := filepath.Join(runtimeDir, "bus")
	info, err := os.Stat(bus)
	if err != nil || info.Mode()&os.ModeSocket == 0 {
		return nil, errors.New("the user D-Bus session socket is unavailable; start Noria from your desktop session")
	}
	env = append(env, "DBUS_SESSION_BUS_ADDRESS=unix:path="+strings.ReplaceAll(bus, ",", "%2C"))
	return env, nil
}
