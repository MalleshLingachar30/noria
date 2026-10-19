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

	"github.com/amxv/noria/internal/core"
	"github.com/amxv/noria/internal/model"
	"github.com/amxv/noria/internal/proxy"
	"github.com/gorilla/websocket"
)

// This is native-client compatibility testing, not live model usage: both
// clients see only temporary profiles and an httptest provider returning text.
// First complete a turn on A, then resume that exact conversation and reject
// A's next request. B must receive the pending continuation with its history.
func TestInstalledNativeResumeFailover(t *testing.T) {
	if os.Getenv("NORIA_NATIVE_COMPAT") != "1" {
		t.Skip("opt-in installed-client local fixtures")
	}
	for _, kind := range []string{"codex", "claude"} {
		t.Run(kind, func(t *testing.T) {
			executable, err := exec.LookPath(kind)
			if override := os.Getenv("NORIA_TEST_" + strings.ToUpper(kind) + "_BIN"); override != "" {
				executable, err = override, nil
			}
			if err != nil {
				t.Skip(err)
			}
			var resumed atomic.Bool
			var rejected, acceptedA, acceptedB atomic.Int32
			var carriedHistory atomic.Bool
			response := func(account string, body []byte) (bool, []string) {
				var request struct {
					Generate *bool `json:"generate"`
				}
				_ = json.Unmarshal(body, &request)
				if request.Generate != nil && !*request.Generate {
					return false, codexFixtureEvents()
				}
				if resumed.Load() && account == "a" {
					rejected.Add(1)
					return true, []string{`{"type":"error","status":429,"error":{"code":"usage_limit_reached","message":"five-hour usage limit reached"}}`}
				}
				if account == "a" {
					acceptedA.Add(1)
				} else {
					acceptedB.Add(1)
					carriedHistory.Store(strings.Contains(string(body), "noria-resume-marker") && strings.Contains(string(body), "fixture-ok"))
				}
				if kind == "claude" {
					return false, claudeFixtureEvents()
				}
				return false, codexFixtureEvents()
			}
			remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer fixture-upstream-")
				if token != "a" && token != "b" {
					t.Error("fixture credential not substituted")
					w.WriteHeader(401)
					return
				}
				if websocket.IsWebSocketUpgrade(r) {
					conn, err := (&websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}).Upgrade(w, r, nil)
					if err != nil {
						return
					}
					defer conn.Close()
					for {
						_, body, err := conn.ReadMessage()
						if err != nil {
							return
						}
						limited, events := response(token, body)
						for _, event := range events {
							if conn.WriteMessage(websocket.TextMessage, []byte(event)) != nil {
								return
							}
						}
						if limited {
							return
						}
					}
				}
				body, _ := io.ReadAll(r.Body)
				if strings.HasSuffix(r.URL.Path, "count_tokens") {
					fmt.Fprint(w, `{"input_tokens":1}`)
					return
				}
				limited, events := response(token, body)
				if limited {
					w.WriteHeader(429)
					fmt.Fprint(w, events[0])
					return
				}
				w.Header().Set("Content-Type", "text/event-stream")
				for _, event := range events {
					var value struct {
						Type string `json:"type"`
					}
					_ = json.Unmarshal([]byte(event), &value)
					fmt.Fprintf(w, "event: %s\ndata: %s\n\n", value.Type, event)
					w.(http.Flusher).Flush()
				}
			}))
			defer remote.Close()
			scheduler, err := core.New("", []model.Account{
				{ID: "a", Provider: kind, Enabled: true, Endpoint: remote.URL},
				{ID: "b", Provider: kind, Enabled: true, Endpoint: remote.URL},
			})
			if err != nil {
				t.Fatal(err)
			}
			gateway := proxy.New(scheduler, nativeFixtureBackend{}, "fixture-local")
			local := httptest.NewServer(gateway)
			defer local.Close()
			defer gateway.Shutdown(context.Background())
			home := t.TempDir()
			profile := filepath.Join(home, kind)
			if err := os.Mkdir(profile, 0700); err != nil {
				t.Fatal(err)
			}
			var env []string
			for _, value := range isolatedEnv(profile, kind) {
				key, _, _ := strings.Cut(value, "=")
				if key != "HOME" && key != "XDG_CONFIG_HOME" && key != "XDG_CACHE_HOME" {
					env = append(env, value)
				}
			}
			env = append(env, "HOME="+home, "XDG_CONFIG_HOME="+home, "XDG_CACHE_HOME="+home, "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1")
			address := strings.TrimPrefix(local.URL, "http://")
			if kind == "codex" {
				env = append(env, "NORIA_GATEWAY_TOKEN=fixture-local")
			} else {
				env = append(env, "ANTHROPIC_BASE_URL="+local.URL+"/claude", "ANTHROPIC_AUTH_TOKEN=fixture-local", "ENABLE_TOOL_SEARCH=true")
			}
			run := func(args []string) []byte {
				t.Helper()
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, executable, args...)
				cmd.Dir, cmd.Env = home, env
				output, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("native %s continuation fixture failed: %v\n%s", kind, err, output)
				}
				if !strings.Contains(string(output), "fixture-ok") {
					t.Fatalf("missing fixture text: %s", output)
				}
				return output
			}
			const prompt = "Remember noria-resume-marker. Reply with fixture-ok."
			sessionID := "1d0a082d-56e8-4fb9-9852-bad5030c2cc6"
			if kind == "codex" {
				output := run(append(codexOverrides(address), "exec", "--skip-git-repo-check", "--json", "-m", "gpt-5.4", prompt))
				sessionID = ""
				for _, line := range strings.Split(string(output), "\n") {
					var event struct {
						Type   string `json:"type"`
						Thread string `json:"thread_id"`
					}
					if json.Unmarshal([]byte(line), &event) == nil && event.Type == "thread.started" {
						sessionID = event.Thread
					}
				}
				if sessionID == "" {
					t.Fatal("native Codex did not expose a resumable thread ID")
				}
			} else {
				run([]string{"--setting-sources", "", "--session-id", sessionID, "-p", prompt, "--output-format", "text", "--max-turns", "1"})
			}
			if acceptedA.Load() != 1 || acceptedB.Load() != 0 {
				t.Fatalf("first turn not sticky on A: A=%d B=%d", acceptedA.Load(), acceptedB.Load())
			}
			resumed.Store(true)
			if kind == "codex" {
				run(append(codexOverrides(address), "exec", "resume", "--skip-git-repo-check", "--json", sessionID, "Continue this conversation. Reply with fixture-ok."))
			} else {
				run([]string{"--setting-sources", "", "--resume", sessionID, "-p", "Continue this conversation. Reply with fixture-ok.", "--output-format", "text", "--max-turns", "1"})
			}
			if rejected.Load() != 1 || acceptedB.Load() != 1 || !carriedHistory.Load() {
				t.Fatalf("resume failover failed: rejected=%d acceptedB=%d history=%t", rejected.Load(), acceptedB.Load(), carriedHistory.Load())
			}
			t.Logf("installed %s resumed its saved conversation, retried A's quota rejection on B, and retained both prior user/assistant turns", kind)
		})
	}
}
