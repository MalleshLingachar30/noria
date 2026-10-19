package app

import (
	"bytes"
	"strings"
	"testing"

	"github.com/amxv/noria/internal/config"
	"github.com/amxv/noria/internal/model"
	"path/filepath"
)

func TestRunRootHelp(t *testing.T) {
	var out bytes.Buffer
	var errBuf bytes.Buffer

	err := Run([]string{"--help"}, &out, &errBuf)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.Contains(out.String(), "Usage:") {
		t.Fatalf("expected help output, got: %q", out.String())
	}
	if !strings.Contains(out.String(), "--version") {
		t.Fatalf("expected --version in help output, got: %q", out.String())
	}
}

func TestRunVersion(t *testing.T) {
	var out bytes.Buffer
	var errBuf bytes.Buffer

	err := Run([]string{"--version"}, &out, &errBuf)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.Contains(out.String(), "noria ") {
		t.Fatalf("unexpected version output: %q", out.String())
	}
}

func TestAccountSetupAndClientIsolation(t *testing.T) {
	t.Setenv("NORIA_HOME", t.TempDir())
	var out bytes.Buffer
	var errBuf bytes.Buffer
	for _, args := range [][]string{{"init"}} {
		if err := Run(args, &out, &errBuf); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	home, err := config.Home()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"work", "personal"} {
		p := filepath.Join(home, "accounts", id)
		if err := config.PrivateDir(p); err != nil {
			t.Fatal(err)
		}
	}
	if err := config.Mutate(home, func(c *config.Config) error {
		c.Accounts = []model.Account{
			{ID: "work", Email: "work@example.com", Provider: "codex", Profile: filepath.Join(home, "accounts", "work"), Enabled: true, Authenticated: true},
			{ID: "personal", Email: "personal@example.com", Provider: "claude", Profile: filepath.Join(home, "accounts", "personal"), Enabled: true, Authenticated: true},
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"account", "disable", "personal"}, {"account", "list"}} {
		if err := Run(args, &out, &errBuf); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	if !strings.Contains(out.String(), "personal@example.com\tclaude\tenabled=false") {
		t.Fatal(out.String())
	}
	out.Reset()
	if err := Run([]string{"env", "codex"}, &out, &errBuf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "NORIA_GATEWAY_TOKEN=") || !strings.Contains(out.String(), "codex() {") || strings.Contains(out.String(), "export CODEX_HOME=") {
		t.Fatal("missing command-local client config or replaced client home")
	}
	out.Reset()
	if err := Run([]string{"env", "claude"}, &out, &errBuf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "ENABLE_TOOL_SEARCH='true'") || !strings.Contains(out.String(), "/claude'") {
		t.Fatal(out.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var out bytes.Buffer
	var errBuf bytes.Buffer

	err := Run([]string{"unknown"}, &out, &errBuf)
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestRunVersionSubcommandUnknown(t *testing.T) {
	var out bytes.Buffer
	var errBuf bytes.Buffer

	err := Run([]string{"version"}, &out, &errBuf)
	if err == nil {
		t.Fatal("expected error for version subcommand")
	}
}
