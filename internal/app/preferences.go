package app

import (
	"errors"
	"fmt"
	"github.com/amxv/noria/internal/config"
	"github.com/amxv/noria/internal/host"
	"github.com/amxv/noria/internal/service"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

func preferencesCommand(home, op string, args []string, out io.Writer) error {
	asJSON := false
	filtered := []string{}
	for _, v := range args {
		if v == "--json" {
			asJSON = true
		} else {
			filtered = append(filtered, v)
		}
	}
	args = filtered
	if len(args) > 0 && isHelp(args[0]) {
		fmt.Fprintln(out, "Usage: noria preferences [start-at-login|menubar-at-login|auto-update on|off] [--json]\n       noria autostart on|off\nAll preferences are owned by the CLI. Quitting the menu bar does not stop routing.")
		return nil
	}
	if op == "autostart" {
		if len(args) != 1 {
			return errors.New("usage: noria autostart on|off")
		}
		args = append([]string{"start-at-login"}, args...)
	}
	if e := config.Init(home); e != nil {
		return e
	}
	if len(args) > 0 {
		if len(args) != 2 || args[1] != "on" && args[1] != "off" {
			return errors.New("specify a preference followed by on or off")
		}
		enabled := args[1] == "on"
		exe, e := os.Executable()
		if e != nil {
			return e
		}
		switch args[0] {
		case "start-at-login":
			if e = service.Login(home, exe, enabled); e != nil {
				return e
			}
		case "menubar-at-login":
			if runtime.GOOS != "darwin" {
				return errors.New("the native menu bar is available on macOS")
			}
			arg := "--unregister-login-item"
			if enabled {
				arg = "--register-login-item"
			}
			if e = host.Run(filepath.Join(menuPath(exe), "Contents", "MacOS", "noria-menubar"), arg); e != nil {
				return e
			}
		case "auto-update":
		default:
			return errors.New("unknown preference")
		}
		if e = config.Change(home, func(c *config.Config) error {
			switch args[0] {
			case "start-at-login":
				c.StartAtLogin = enabled
			case "menubar-at-login":
				c.MenuBarAtLogin = enabled
			case "auto-update":
				c.AutoUpdate = enabled
			}
			return nil
		}); e != nil {
			return e
		}
	}
	c, e := config.Load(home)
	if e != nil {
		return e
	}
	v := map[string]bool{"start_at_login": c.StartAtLogin, "menubar_at_login": c.MenuBarAtLogin, "auto_update": c.AutoUpdate}
	if asJSON {
		return printJSON(out, v)
	}
	fmt.Fprintf(out, "Start at login: %t\nMenu bar at login: %t\nAutomatic updates: %t\n", c.StartAtLogin, c.MenuBarAtLogin, c.AutoUpdate)
	return nil
}
func menuPath(exe string) string { return filepath.Join(filepath.Dir(exe), "Noria.app") }
