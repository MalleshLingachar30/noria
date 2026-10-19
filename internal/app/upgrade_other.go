//go:build !windows

package app

import (
	"errors"
	"io"
)

func scheduleWindowsUpgrade(string, string, string, bool, string) error {
	return errors.New("Windows upgrade helper is unavailable on this platform")
}

func applyWindowsUpgrade(string, []string, io.Writer) error {
	return errors.New("Windows upgrade helper is unavailable on this platform")
}
