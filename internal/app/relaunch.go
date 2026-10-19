package app

import (
	"time"
)

func delayedMenuHome(home string) error {
	time.Sleep(1500 * time.Millisecond)
	return menubar(home, []string{"open"}, nil, nil)
}
