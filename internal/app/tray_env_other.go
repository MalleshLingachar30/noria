//go:build !linux

package app

import "os"

func desktopTrayEnvironment() ([]string, error) { return os.Environ(), nil }
