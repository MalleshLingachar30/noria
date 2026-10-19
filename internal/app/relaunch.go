package app

import (
	"errors"
	"io"
	"os/exec"
	"strconv"
	"time"
)

func delayedMenuHome(home string, args []string) error {
	if len(args) != 2 || args[0] != "--wait-pid" {
		return errors.New("invalid menu bar relaunch request")
	}
	pid, err := strconv.Atoi(args[1])
	if err != nil || pid < 0 {
		return errors.New("invalid menu bar process ID")
	}
	if pid == 0 {
		time.Sleep(1500 * time.Millisecond) // CLI-requested relaunch, no GUI PID
	} else {
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			if exec.Command("/bin/kill", "-0", strconv.Itoa(pid)).Run() != nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if exec.Command("/bin/kill", "-0", strconv.Itoa(pid)).Run() == nil {
			return errors.New("previous menu bar instance did not exit")
		}
	}
	return menubar(home, []string{"open"}, io.Discard, io.Discard)
}
