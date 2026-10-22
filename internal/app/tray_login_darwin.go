//go:build darwin

package app

import "errors"

func trayLogin(string, string, bool) error { return errors.New("use macOS menu bar login settings") }
