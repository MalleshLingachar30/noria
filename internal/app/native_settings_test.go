package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amxv/noria/internal/config"
	"github.com/amxv/noria/internal/core"
	"github.com/amxv/noria/internal/integration"
	"github.com/amxv/noria/internal/model"
	"github.com/amxv/noria/internal/proxy"
	"github.com/gorilla/websocket"
)

// Exercise saved user-level settings and command-backed auth, without a wrapper,
// exported gateway variables, real native credentials, or real model requests.
func TestInstalledClientsUsePersistentSettings(t *testing.T) {
	if os.Getenv("NORIA_NATIVE_COMPAT") != "1" {
		t.Skip("opt-in native settings test")
	}
	noria := os.Getenv("NORIA_TEST_NORIA_BIN")
	if noria == "" {
		t.Skip("set NORIA_TEST_NORIA_BIN to the built CLI")
	}
	for _, kind := range []string{"codex", "claude"} {
		t.Run(kind, func(t *testing.T) {
			executable, e := exec.LookPath(kind)
			if v := os.Getenv("NORIA_TEST_" + strings.ToUpper(kind) + "_BIN"); v != "" {
				executable, e = v, nil
			}
			if e != nil {
				t.Skip(e)
			}
			var requests atomic.Int32
			remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer fixture-upstream-") {
					t.Error("native helper auth was not substituted")
				}
				limited := r.Header.Get("Authorization") == "Bearer fixture-upstream-a"
				if websocket.IsWebSocketUpgrade(r) {
					conn, e := (&websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}).Upgrade(w, r, nil)
					if e != nil {
						return
					}
					defer conn.Close()
					for {
						_, _, e = conn.ReadMessage()
						if e != nil {
							return
						}
						requests.Add(1)
						if limited {
							_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","status":429,"error":{"code":"usage_limit_reached"}}`))
							return
						}
						for _, event := range codexFixtureEvents() {
							if e = conn.WriteMessage(websocket.TextMessage, []byte(event)); e != nil {
								return
							}
						}
					}
				}
				_, _ = io.Copy(io.Discard, r.Body)
				if strings.HasSuffix(r.URL.Path, "count_tokens") {
					fmt.Fprint(w, `{"input_tokens":1}`)
					return
				}
				requests.Add(1)
				if limited {
					w.WriteHeader(429)
					fmt.Fprint(w, `{"error":{"type":"rate_limit_error","message":"five-hour usage limit reached"}}`)
					return
				}
				events := codexFixtureEvents()
				if kind == "claude" {
					events = claudeFixtureEvents()
				}
				w.Header().Set("Content-Type", "text/event-stream")
				for _, event := range events {
					var v struct {
						Type string `json:"type"`
					}
					_ = json.Unmarshal([]byte(event), &v)
					fmt.Fprintf(w, "event: %s\ndata: %s\n\n", v.Type, event)
					w.(http.Flusher).Flush()
				}
			}))
			defer remote.Close()
			scheduler, e := core.New("", []model.Account{{ID: "a", Provider: kind, Enabled: true, Endpoint: remote.URL}, {ID: "b", Provider: kind, Enabled: true, Endpoint: remote.URL}})
			if e != nil {
				t.Fatal(e)
			}
			home := t.TempDir()
			native := filepath.Join(home, kind)
			if e = os.Mkdir(native, 0700); e != nil {
				t.Fatal(e)
			}
			noriaHome := filepath.Join(home, "noria")
			if e = config.Init(noriaHome); e != nil {
				t.Fatal(e)
			}
			secret, _ := config.Secret(noriaHome)
			server := proxy.New(scheduler, nativeFixtureBackend{}, secret)
			gateway := httptest.NewServer(server)
			defer gateway.Close()
			defer server.Shutdown(context.Background())
			if e = integration.Install(noriaHome, kind, native, noria, strings.TrimPrefix(gateway.URL, "http://")); e != nil {
				t.Fatal(e)
			}
			env := []string{}
			for _, pair := range isolatedEnv(native, kind) {
				key, _, _ := strings.Cut(pair, "=")
				if key != "HOME" && key != "XDG_CONFIG_HOME" && key != "XDG_CACHE_HOME" {
					env = append(env, pair)
				}
			}
			env = append(env, "HOME="+home, "XDG_CONFIG_HOME="+home, "XDG_CACHE_HOME="+home, "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1")
			args := []string{"exec", "--skip-git-repo-check", "--json", "-m", "gpt-5.4", "Reply with fixture-ok."}
			if kind == "claude" {
				args = []string{"--setting-sources=user", "--no-session-persistence", "-p", "Reply with fixture-ok.", "--max-turns", "1", "--output-format", "text"}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, executable, args...)
			cmd.Env = env
			cmd.Dir = home
			output, e := cmd.CombinedOutput()
			if e != nil || !strings.Contains(string(output), "fixture-ok") || requests.Load() < 2 {
				t.Fatalf("persistent %s config not honored: %v requests=%d\n%s", kind, e, requests.Load(), output)
			}
			if e = integration.SetActive(noriaHome, noria, strings.TrimPrefix(gateway.URL, "http://"), false); e != nil {
				t.Fatal(e)
			}
			t.Logf("%s used saved user settings and native token helper, failed over, then restored direct config", kind)
		})
	}
}
