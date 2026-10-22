//go:build linux

package tray

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/godbus/dbus/v5"
)

func prepareDesktop() error {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return errors.New("a graphical Linux desktop session is required")
	}
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") != "" {
		return nil
	}
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if !filepath.IsAbs(runtimeDir) || runtimeDir == "" {
		return errors.New("missing user D-Bus session address")
	}
	bus := filepath.Join(runtimeDir, "bus")
	info, err := os.Stat(bus)
	if err != nil || info.Mode()&os.ModeSocket == 0 {
		return errors.New("user D-Bus session socket is unavailable")
	}
	return os.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+strings.ReplaceAll(bus, ",", "%2C"))
}

// CheckHost diagnoses missing status icon support without starting a GUI.
// The actual tray does not require a watcher at launch: systray registers
// once the desktop's tray host becomes available later in the login session.
func CheckHost() error {
	if err := prepareDesktop(); err != nil {
		return err
	}
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("connect to the desktop D-Bus: %w", err)
	}
	defer conn.Close()
	var owned bool
	if err := conn.BusObject().Call("org.freedesktop.DBus.NameHasOwner", 0, "org.kde.StatusNotifierWatcher").Store(&owned); err != nil {
		return fmt.Errorf("inspect the desktop status notifier: %w", err)
	}
	if owned {
		return nil
	}
	if strings.Contains(strings.ToUpper(os.Getenv("XDG_CURRENT_DESKTOP")), "GNOME") {
		return errors.New("GNOME's AppIndicator extension is not active. Ubuntu Desktop includes it; on other GNOME desktops, enable 'AppIndicator and KStatusNotifierItem Support' in Extensions")
	}
	return errors.New("this desktop has no active StatusNotifierItem tray host; enable its tray/panel applet")
}
