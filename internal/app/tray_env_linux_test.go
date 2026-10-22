//go:build linux

package app

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTrayEnvironmentRequiresGraphicalSession(t *testing.T) {
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/example/bus")
	if _, err := desktopTrayEnvironment(); err == nil {
		t.Fatal("headless session accepted")
	}
}

func TestTrayEnvironmentUsesExistingSessionBus(t *testing.T) {
	t.Setenv("DISPLAY", ":0")
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/example/bus")
	env, err := desktopTrayEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(env, "\n"), "DBUS_SESSION_BUS_ADDRESS=unix:path=/example/bus") {
		t.Fatalf("missing bus: %v", env)
	}
}

func TestTrayEnvironmentDerivesSocketAddress(t *testing.T) {
	root := t.TempDir()
	socket := filepath.Join(root, "bus")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")
	t.Setenv("DISPLAY", "")
	t.Setenv("XDG_RUNTIME_DIR", root)
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "")
	env, err := desktopTrayEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	expected := "DBUS_SESSION_BUS_ADDRESS=unix:path=" + socket
	if !strings.Contains(strings.Join(env, "\n"), expected) {
		t.Fatalf("missing derived bus %s", expected)
	}
	if _, err := os.Stat(socket); err != nil {
		t.Fatal(err)
	}
}

func TestTrayEnvironmentRejectsMissingSessionBus(t *testing.T) {
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")
	t.Setenv("DISPLAY", "")
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "")
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	if _, err := desktopTrayEnvironment(); err == nil {
		t.Fatal("nonexistent session bus accepted")
	}
}
