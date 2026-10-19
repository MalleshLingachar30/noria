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

type nativeFixtureBackend struct{}

func (nativeFixtureBackend) Credentials(_ context.Context, account model.Account) (model.Credential, error) {
	return model.Credential{Token: "fixture-upstream-" + account.ID}, nil
}
func (nativeFixtureBackend) Refresh(context.Context, model.Account) (model.Snapshot, error) {
	return model.Snapshot{}, fmt.Errorf("fixture telemetry unknown")
}

// Opt-in checks execute installed native clients only against httptest servers,
// with all homes and working directories isolated. They never use live quota.
func TestInstalledNativeClients(t *testing.T) {
	if os.Getenv("NORIA_NATIVE_COMPAT") != "1" {
		t.Skip("set NORIA_NATIVE_COMPAT=1 for installed-client local compatibility checks")
	}
	for _, kind := range []string{"codex", "claude"} {
		t.Run(kind, func(t *testing.T) {
			executable, e := exec.LookPath(kind)
			if override := os.Getenv("NORIA_TEST_" + strings.ToUpper(kind) + "_BIN"); override != "" {
				executable = override
				e = nil
			}
			if e != nil {
				t.Skip(e)
			}
			var calls atomic.Int32
			var wsCalls atomic.Int32
			remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer fixture-upstream-") {
					t.Error("incorrect upstream credential")
				}
				calls.Add(1)
				if websocket.IsWebSocketUpgrade(r) {
					wsCalls.Add(1)
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
						if r.Header.Get("Authorization") == "Bearer fixture-upstream-a" {
							_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","status":429,"error":{"code":"usage_limit_reached"}}`))
							return
						}
						for _, event := range codexFixtureEvents() {
							if conn.WriteMessage(websocket.TextMessage, []byte(event)) != nil {
								return
							}
						}
					}
				}
				_, _ = io.Copy(io.Discard, r.Body)
				if r.Header.Get("Authorization") == "Bearer fixture-upstream-a" {
					w.WriteHeader(429)
					fmt.Fprint(w, `{"type":"error","error":{"type":"rate_limit_error","message":"five-hour usage limit reached"}}`)
					return
				}
				if strings.HasSuffix(r.URL.Path, "count_tokens") {
					fmt.Fprint(w, `{"input_tokens":1}`)
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
					if f, ok := w.(http.Flusher); ok {
						f.Flush()
					}
				}
			}))
			defer remote.Close()
			scheduler, e := core.New("", []model.Account{{ID: "a", Provider: kind, Enabled: true, Endpoint: remote.URL}, {ID: "b", Provider: kind, Enabled: true, Endpoint: remote.URL}})
			if e != nil {
				t.Fatal(e)
			}
			gateway := proxy.New(scheduler, nativeFixtureBackend{}, "fixture-local")
			local := httptest.NewServer(gateway)
			defer local.Close()
			defer gateway.Shutdown(context.Background())
			h := t.TempDir()
			profile := filepath.Join(h, kind)
			if e = os.Mkdir(profile, 0700); e != nil {
				t.Fatal(e)
			}
			env := isolatedEnv(profile, kind)
			// HOME and XDG directories are also temporary so incidental native state
			// cannot touch the real user's config, keychain profile, or history.
			filtered := []string{}
			for _, v := range env {
				k, _, _ := strings.Cut(v, "=")
				if k == "HOME" || k == "XDG_CONFIG_HOME" || k == "XDG_CACHE_HOME" {
					continue
				}
				filtered = append(filtered, v)
			}
			env = append(filtered, "HOME="+h, "XDG_CONFIG_HOME="+h, "XDG_CACHE_HOME="+h, "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1")
			var args []string
			if kind == "codex" {
				env = append(env, "NORIA_GATEWAY_TOKEN=fixture-local")
				args = append(codexOverrides(strings.TrimPrefix(local.URL, "http://")), "exec", "--skip-git-repo-check", "--json", "-m", "gpt-5.4", "Reply with fixture-ok.")
			} else {
				env = append(env, "ANTHROPIC_BASE_URL="+local.URL+"/claude", "ANTHROPIC_AUTH_TOKEN=fixture-local", "ENABLE_TOOL_SEARCH=true")
				args = []string{"--setting-sources", "", "--no-session-persistence", "-p", "Reply with fixture-ok.", "--output-format", "text", "--max-turns", "1"}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, executable, args...)
			cmd.Dir = h
			cmd.Env = env
			output, e := cmd.CombinedOutput()
			if e != nil {
				t.Fatalf("native %s compatibility failed: %v\n%s", kind, e, output)
			}
			if calls.Load() != 2 || !strings.Contains(string(output), "fixture-ok") {
				t.Fatalf("native %s did not complete through gateway (%d calls): %s", kind, calls.Load(), output)
			}
			var affinity struct {
				Bindings int `json:"bindings"`
			}
			saved, _ := json.Marshal(scheduler.Status())
			_ = json.Unmarshal(saved, &affinity)
			if affinity.Bindings == 0 {
				t.Fatalf("native %s supplied no recognized conversation identity", kind)
			}
			t.Logf("installed %s completed through local gateway with forced quota failover (%d upstream connection/request(s), %d WebSocket, recognized session identity)", kind, calls.Load(), wsCalls.Load())
		})
	}
}

func codexFixtureEvents() []string {
	item := `{"id":"msg_fixture","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"fixture-ok","annotations":[]}]}`
	return []string{
		`{"type":"response.created","response":{"id":"resp_fixture","status":"in_progress","output":[]}}`,
		`{"type":"response.output_text.delta","item_id":"msg_fixture","output_index":0,"content_index":0,"delta":"fixture-ok"}`,
		`{"type":"response.output_item.done","output_index":0,"item":` + item + `}`,
		`{"type":"response.completed","response":{"id":"resp_fixture","status":"completed","output":[` + item + `],"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2,"input_tokens_details":{"cached_tokens":0},"output_tokens_details":{"reasoning_tokens":0}}}}`,
	}
}
func claudeFixtureEvents() []string {
	return []string{
		`{"type":"message_start","message":{"id":"msg_fixture","type":"message","role":"assistant","model":"claude-sonnet-4-6","content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":1,"output_tokens":0}}}`,
		`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"fixture-ok"}}`,
		`{"type":"content_block_stop","index":0}`,
		`{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":1}}`,
		`{"type":"message_stop"}`,
	}
}
