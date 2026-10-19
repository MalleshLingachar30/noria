package app

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/amxv/noria/internal/config"
)

// Opt-in OS integration smoke. It registers no login item, logs into no provider,
// uses a random loopback port and changes only temporary native configuration.
func TestBackgroundServiceAndDirectRouting(t *testing.T) {
	if os.Getenv("NORIA_SERVICE_COMPAT") != "1" {
		t.Skip("opt-in real background service test")
	}
	binary := os.Getenv("NORIA_TEST_NORIA_BIN")
	if binary == "" {
		t.Fatal("NORIA_TEST_NORIA_BIN is required")
	}
	binary, e := filepath.Abs(binary)
	if e != nil {
		t.Fatal(e)
	}
	root := t.TempDir()
	home := filepath.Join(root, "noria")
	if e = config.Init(home); e != nil {
		t.Fatal(e)
	}
	socket, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	address := socket.Addr().String()
	socket.Close()
	if e = config.Change(home, func(c *config.Config) error { c.Listen = address; c.AutoUpdate = false; return nil }); e != nil {
		t.Fatal(e)
	}
	env := []string{}
	for _, pair := range os.Environ() {
		key, _, _ := strings.Cut(pair, "=")
		switch key {
		case "HOME", "USERPROFILE", "NORIA_HOME", "CODEX_HOME", "CLAUDE_CONFIG_DIR", "XDG_CONFIG_HOME", "XDG_CACHE_HOME":
			continue
		}
		env = append(env, pair)
	}
	env = append(env, "HOME="+root, "USERPROFILE="+root, "NORIA_HOME="+home, "XDG_CONFIG_HOME="+root, "XDG_CACHE_HOME="+root)
	command := func(args ...string) ([]byte, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Env = env
		cmd.Dir = root
		return cmd.CombinedOutput()
	}
	cleanup := func() {
		_, _ = command("stop", "--json")
		if unlock, e := config.Lock(home, "supervisor"); e == nil {
			unlock()
		}
	}
	t.Cleanup(cleanup)
	run := func(args ...string) []byte {
		t.Helper()
		b, e := command(args...)
		if e != nil {
			t.Fatalf("%v failed: %v\n%s", args, e, b)
		}
		return b
	}
	codexRoot := filepath.Join(root, "codex-client")
	claudeRoot := filepath.Join(root, "claude-client")
	_ = os.MkdirAll(codexRoot, 0700)
	_ = os.MkdirAll(claudeRoot, 0700)
	_ = os.WriteFile(filepath.Join(codexRoot, "config.toml"), []byte("model_provider = \"openai\"\nmodel = \"fixture\"\n"), 0600)
	_ = os.WriteFile(filepath.Join(claudeRoot, "settings.json"), []byte(`{"env":{"KEEP":"yes"}}`), 0600)
	run("start", "--json")
	run("start", "--json")
	run("integrate", "codex", "--client-home", codexRoot, "--yes", "--json")
	run("integrate", "claude", "--client-home", claudeRoot, "--yes", "--json")
	var active struct {
		Running bool `json:"running"`
	}
	if e = json.Unmarshal(run("status", "--json"), &active); e != nil || !active.Running {
		t.Fatal("service not healthy", e)
	}
	run("stop", "--json")
	b, _ := os.ReadFile(filepath.Join(codexRoot, "config.toml"))
	if !strings.Contains(string(b), `model_provider = "openai"`) || strings.Contains(string(b), "model_providers.noria") {
		t.Fatal("Codex direct settings not restored", string(b))
	}
	b, _ = os.ReadFile(filepath.Join(claudeRoot, "settings.json"))
	if strings.Contains(string(b), "ANTHROPIC_BASE_URL") || !strings.Contains(string(b), "KEEP") {
		t.Fatal("Claude direct settings not restored", string(b))
	}
	run("restart", "--json")
	run("health", "--json")
	run("stop", "--json")
	t.Log("Real user-scoped service started, repeated safely, stopped, restored both settings, restarted and stopped cleanly")
}
