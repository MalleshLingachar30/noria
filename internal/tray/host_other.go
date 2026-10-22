//go:build !linux

package tray

func prepareDesktop() error { return nil }
func CheckHost() error      { return nil }
