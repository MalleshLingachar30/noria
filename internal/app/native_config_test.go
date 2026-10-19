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

// These real installed clients read only persistent config files. No wrapper
// flags or provider environment credentials can make the test pass accidentally.
func TestInstalledNativePersistentConfiguration(t *testing.T) {
	if os.Getenv("NORIA_NATIVE_COMPAT") != "1" {
		t.Skip("opt-in installed-client test")
	}
	dir := t.TempDir()
	helper := filepath.Join(dir, "noria")
	build := exec.Command("go", "build", "-o", helper, "./cmd/noria")
	build.Dir = "../.."
	if b, e := build.CombinedOutput(); e != nil {
		t.Fatalf("build credential helper: %v %s", e, b)
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
			h := t.TempDir()
			private := filepath.Join(h, "noria")
			profile := filepath.Join(h, "native")
			_ = os.MkdirAll(profile, 0700)
			if e = config.Init(private); e != nil {
				t.Fatal(e)
			}
			secret, _ := config.Secret(private)
			var attempts atomic.Int32
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempts.Add(1)
				limited := r.Header.Get("Authorization") == "Bearer fixture-upstream-a"
				if websocket.IsWebSocketUpgrade(r) {
					c, e := (&websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}).Upgrade(w, r, nil)
					if e != nil {
						return
					}
					defer c.Close()
					for {
						_, _, e = c.ReadMessage()
						if e != nil {
							return
						}
						if limited {
							_ = c.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","status":429,"error":{"code":"usage_limit_reached"}}`))
							return
						}
						for _, v := range codexFixtureEvents() {
							if c.WriteMessage(websocket.TextMessage, []byte(v)) != nil {
								return
							}
						}
					}
				}
				_, _ = io.Copy(io.Discard, r.Body)
				if limited {
					w.WriteHeader(429)
					fmt.Fprint(w, `{"error":{"type":"rate_limit_error","message":"five-hour usage limit reached"}}`)
					return
				}
				w.Header().Set("Content-Type", "text/event-stream")
				events := codexFixtureEvents()
				if kind == "claude" {
					events = claudeFixtureEvents()
				}
				for _, event := range events {
					var v struct {
						Type string `json:"type"`
					}
					_ = json.Unmarshal([]byte(event), &v)
					fmt.Fprintf(w, "event: %s\ndata: %s\n\n", v.Type, event)
					w.(http.Flusher).Flush()
				}
			}))
			defer upstream.Close()
			scheduler, e := core.New("", []model.Account{{ID: "a", Provider: kind, Enabled: true, Endpoint: upstream.URL}, {ID: "b", Provider: kind, Enabled: true, Endpoint: upstream.URL}})
			if e != nil {
				t.Fatal(e)
			}
			gateway := proxy.New(scheduler, nativeFixtureBackend{}, secret)
			local := httptest.NewServer(gateway)
			defer local.Close()
			defer gateway.Shutdown(context.Background())
			if e = integration.Install(private, kind, profile, helper, strings.TrimPrefix(local.URL, "http://")); e != nil {
				t.Fatal(e)
			}
			env := []string{}
			for _, value := range isolatedEnv(profile, kind) {
				key, _, _ := strings.Cut(value, "=")
				if key != "HOME" && key != "XDG_CONFIG_HOME" && key != "XDG_CACHE_HOME" {
					env = append(env, value)
				}
			}
			env = append(env, "HOME="+h, "XDG_CONFIG_HOME="+h, "XDG_CACHE_HOME="+h, "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1")
			args := []string{"exec", "--skip-git-repo-check", "--json", "-m", "gpt-5.4", "Reply with fixture-ok."}
			if kind == "claude" {
				args = []string{"--setting-sources", "user", "--no-session-persistence", "-p", "Reply with fixture-ok.", "--output-format", "text", "--max-turns", "1"}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, executable, args...)
			cmd.Dir = h
			cmd.Env = env
			output, e := cmd.CombinedOutput()
			if e != nil {
				t.Fatalf("%s config-only native client failed: %v\n%s", kind, e, output)
			}
			if attempts.Load() != 2 || !strings.Contains(string(output), "fixture-ok") {
				t.Fatalf("persistent routing did not failover A to B: attempts=%d output=%s", attempts.Load(), output)
			}
			if e = integration.SetActive(private, helper, strings.TrimPrefix(local.URL, "http://"), false); e != nil {
				t.Fatal(e)
			}
			t.Logf("installed %s used persistent settings and the Noria auth helper, with no wrapper/provider env, including forced account failover", kind)
		})
	}
}
