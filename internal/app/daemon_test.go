package app

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/amxv/noria/internal/config"
)

type readyWriter struct{ ready chan struct{} }

func (w readyWriter) Write(b []byte) (int, error) {
	select {
	case w.ready <- struct{}{}:
	default:
	}
	return len(b), nil
}

func TestDaemonLifecycleControlAndRestart(t *testing.T) {
	home := t.TempDir()
	if e := config.Init(home); e != nil {
		t.Fatal(e)
	}
	for range 2 {
		ctx, cancel := context.WithCancel(context.Background())
		ready := make(chan struct{}, 1)
		done := make(chan error, 1)
		go func() {
			done <- serveWithContext(ctx, home, []string{"--listen", "127.0.0.1:0"}, readyWriter{ready}, io.Discard)
		}()
		select {
		case <-ready:
		case e := <-done:
			cancel()
			t.Fatal("daemon stopped early", e)
		case <-time.After(3 * time.Second):
			cancel()
			t.Fatal("daemon startup timed out")
		}
		if b, e := controlRequest(home, "GET", "/_noria/status", nil); e != nil || !strings.Contains(string(b), `"accounts":[]`) {
			cancel()
			t.Fatalf("live status: %s %v", b, e)
		}
		if b, e := controlRequest(home, "POST", "/_noria/refresh", map[string]string{"account": ""}); e != nil || string(b) != "[]\n" {
			cancel()
			t.Fatalf("refresh: %s %v", b, e)
		}
		if _, e := controlRequest(home, "POST", "/_noria/reload", nil); e != nil {
			cancel()
			t.Fatal(e)
		}
		secret, _ := config.Secret(home)
		addr := daemonAddress(home, "")
		for _, origin := range []string{"", "https://evil.example"} {
			req, _ := http.NewRequest("POST", "http://"+addr+"/_noria/refresh", strings.NewReader(`{}`))
			if origin != "" {
				req.Header.Set("Authorization", "Bearer "+secret)
				req.Header.Set("Origin", origin)
			}
			resp, e := http.DefaultClient.Do(req)
			if e != nil {
				cancel()
				t.Fatal(e)
			}
			resp.Body.Close()
			if resp.StatusCode != 401 && resp.StatusCode != 403 {
				cancel()
				t.Fatalf("control endpoint accepted unauthorized request: %d", resp.StatusCode)
			}
		}
		cancel()
		select {
		case e := <-done:
			if e != nil {
				t.Fatal(e)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("daemon did not shut down")
		}
		release, lockErr := config.Lock(home, "daemon")
		if lockErr != nil {
			t.Fatalf("daemon retained OS lock: %v", lockErr)
		}
		release()
		for _, name := range []string{"daemon.json"} {
			if _, e := os.Stat(filepath.Join(home, name)); !os.IsNotExist(e) {
				t.Fatalf("runtime file retained %s: %v", name, e)
			}
		}
	}
}

func TestClientSetupRetainsNormalHistoryAndSettings(t *testing.T) {
	home := t.TempDir()
	profile := t.TempDir()
	t.Setenv("NORIA_HOME", home)
	t.Setenv("CODEX_HOME", profile)
	t.Setenv("CLAUDE_CONFIG_DIR", profile)
	if e := config.Init(home); e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"config.toml", "history.jsonl", "settings.json"} {
		if e := os.WriteFile(filepath.Join(profile, name), []byte("sentinel"), 0600); e != nil {
			t.Fatal(e)
		}
	}
	for _, kind := range []string{"codex", "claude"} {
		var out bytes.Buffer
		if e := Run([]string{"env", kind}, &out, io.Discard); e != nil {
			t.Fatal(e)
		}
		if strings.Contains(out.String(), "export CODEX_HOME=") || strings.Contains(out.String(), "export CLAUDE_CONFIG_DIR=") {
			t.Fatal("client home replaced")
		}
	}
	env := strings.Join(clientEnv(), "\n")
	if !strings.Contains(env, "CODEX_HOME="+profile) || !strings.Contains(env, "CLAUDE_CONFIG_DIR="+profile) {
		t.Fatal("client homes not preserved")
	}
	for _, name := range []string{"config.toml", "history.jsonl", "settings.json"} {
		b, e := os.ReadFile(filepath.Join(profile, name))
		if e != nil || string(b) != "sentinel" {
			t.Fatal("client config or history changed")
		}
	}
}
