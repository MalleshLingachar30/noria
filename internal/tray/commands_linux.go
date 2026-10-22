//go:build linux

package tray

import "os/exec"

func hideConsole(*exec.Cmd) {}
