//go:build windows

package provider

import (
	"os/exec"
	"strconv"
)

func configureProcess(cmd *exec.Cmd) { cmd.Cancel = func() error { killProcess(cmd); return nil } }
func killProcess(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
		_ = cmd.Process.Kill()
	}
}
