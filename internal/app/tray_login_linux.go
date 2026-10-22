//go:build linux

package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Freedesktop XDG autostart works across common GNOME, KDE, Xfce, Cinnamon
// and MATE sessions. The system tray itself needs an SNI/AppIndicator host.
func trayLogin(home, exe string, enabled bool) error {
	user, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	xdgHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgHome == "" {
		xdgHome = filepath.Join(user, ".config")
	}
	if !filepath.IsAbs(xdgHome) {
		return fmt.Errorf("XDG_CONFIG_HOME must be absolute")
	}
	path := filepath.Join(xdgHome, "autostart", "noria-tray.desktop")
	if !enabled {
		err := os.Remove(path)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	tray := trayBinary(exe)
	if _, err := os.Stat(tray); err != nil {
		return fmt.Errorf("install Noria tray before enabling login startup: %w", err)
	}
	// The Desktop Entry spec escapes the four characters that matter inside
	// quoted Exec arguments, with a second escaping pass for percent symbols.
	quote := func(s string) string {
		s = strings.ReplaceAll(s, "%", "%%")
		s = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", "$", "\\$").Replace(s)
		return `"` + s + `"`
	}
	content := "[Desktop Entry]\nType=Application\nName=Noria Tray\nComment=Local account router\nExec=" + quote(tray) + " --home " + quote(home) + "\nTerminal=false\nX-GNOME-Autostart-enabled=true\n"
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0600)
}
