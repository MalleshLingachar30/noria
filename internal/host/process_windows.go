//go:build windows

package host

import (
	"os/exec"
	"strconv"
	"syscall"
)

func Detach(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000200 | 0x00000008, HideWindow: true}
}
func Terminate(c *exec.Cmd) { Kill(c) }
func Kill(c *exec.Cmd) {
	if c.Process != nil {
		_ = exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(c.Process.Pid)).Run()
		_ = c.Process.Kill()
	}
}
