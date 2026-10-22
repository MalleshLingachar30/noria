//go:build windows

package app

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func trayLogin(home, exe string, enabled bool) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	const name = "Noria Tray"
	if !enabled {
		err := key.DeleteValue(name)
		if err == registry.ErrNotExist {
			return nil
		}
		return err
	}
	tray := trayBinary(exe)
	if _, err := os.Stat(tray); err != nil {
		return fmt.Errorf("install Noria tray before enabling login startup: %w", err)
	}
	if strings.ContainsAny(tray+home, "\"\r\n") {
		return fmt.Errorf("tray startup path contains unsupported characters")
	}
	return key.SetStringValue(name, `"`+tray+`" --home "`+home+`"`)
}
