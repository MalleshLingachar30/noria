package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amxv/noria/internal/config"
)

func TestStatusDoesNotExposeSecret(t *testing.T) {
	h := t.TempDir()
	t.Setenv("NORIA_HOME", h)
	if e := config.Init(h); e != nil {
		t.Fatal(e)
	}
	c, _ := config.Load(h)
	c.Listen = "127.0.0.1:1"
	_ = config.Save(h, c)
	secret, _ := config.Secret(h)
	var out, errs bytes.Buffer
	if e := Run([]string{"status", "--json"}, &out, &errs); e != nil {
		t.Fatal(e)
	}
	if strings.Contains(out.String(), secret) {
		t.Fatal("status exposed secret")
	}
}

func TestNativeEnvironmentStripsGatewayAndForeignAuth(t *testing.T) {
	t.Setenv("CODEX_HOME", "/must-not-use")
	t.Setenv("CLAUDE_CONFIG_DIR", "/must-not-use")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "foreign")
	t.Setenv("OPENAI_API_KEY", "foreign")
	t.Setenv("NORIA_GATEWAY_TOKEN", "foreign")
	env := strings.Join(isolatedEnv("/isolated", "codex"), "\n")
	if strings.Contains(env, "foreign") || strings.Contains(env, "/must-not-use") {
		t.Fatal("foreign gateway or profile environment leaked")
	}
	if !strings.Contains(env, "CODEX_HOME=/isolated") {
		t.Fatal(env)
	}
}

func TestAccountRemoveRetainsProfileAndConfigRejectsTraversal(t *testing.T) {
	h := t.TempDir()
	t.Setenv("NORIA_HOME", h)
	var out, errs bytes.Buffer
	for _, args := range [][]string{{"init"}, {"account", "add", "a", "--provider", "codex"}, {"account", "remove", "a"}} {
		if e := Run(args, &out, &errs); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := os.Stat(filepath.Join(h, "accounts", "a")); e != nil {
		t.Fatal("remove erased credentials/history")
	}
	if e := Run([]string{"account", "add", "../bad", "--provider", "codex"}, &out, &errs); e == nil {
		t.Fatal("accepted traversal")
	}
}

func TestAllCommandHelp(t *testing.T) {
	t.Setenv("NORIA_HOME", t.TempDir())
	for _, command := range [][]string{{"init"}, {"account"}, {"account", "add"}, {"serve"}, {"status"}, {"quota"}, {"quota", "refresh"}, {"doctor"}, {"env"}, {"run"}} {
		args := append(append([]string{}, command...), "--help")
		var out bytes.Buffer
		if e := Run(args, &out, &out); e != nil {
			t.Fatalf("%v: %v", args, e)
		}
		if !strings.Contains(out.String(), "Usage") && !strings.Contains(out.String(), "usage") {
			t.Fatalf("missing help for %v: %s", args, out.String())
		}
	}
}
