//go:build darwin

package tray

import "os/exec"

// The Windows/Linux tray code is not shipped on macOS. This keeps its data
// contract tests runnable there alongside the native Swift menu fixtures.
func hideConsole(*exec.Cmd) {}
