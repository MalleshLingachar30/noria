package app

import (
	"github.com/amxv/noria/internal/host"
	"os/exec"
	"time"
)

func relaunchMenu(exe, path string) error {
	c := exec.Command(exe, "__open-menu", path)
	host.Detach(c)
	if e := c.Start(); e != nil {
		return e
	}
	return c.Process.Release()
}
func delayedMenu(path string) error { time.Sleep(1500 * time.Millisecond); return host.OpenURL(path) }

func delayedMenuHome(home string) error {
	time.Sleep(1500 * time.Millisecond)
	return menubar(home, []string{"open"}, nil, nil)
}
