package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

func TestInstalledCodexResumesPreIntegrationSession(t *testing.T) {
	if os.Getenv("NORIA_NATIVE_COMPAT") != "1" {
		t.Skip("opt-in native client fixture")
	}
	executable := os.Getenv("NORIA_TEST_CODEX_BIN")
	if executable == "" {
		t.Skip("explicit native Codex executable required")
	}
	h := t.TempDir()
	pool := filepath.Join(h, "pool")
	profile := filepath.Join(h, "client")
	_ = os.MkdirAll(profile, 0700)
	if e := config.Init(pool); e != nil {
		t.Fatal(e)
	}
	secret, _ := config.Secret(pool)
	helper := filepath.Join(h, "noria")
	build := exec.Command("go", "build", "-o", helper, "./cmd/noria")
	build.Dir = "../.."
	if b, e := build.CombinedOutput(); e != nil {
		t.Fatalf("helper build: %v %s", e, b)
	}
	var direct, pooled atomic.Int32
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		isDirect := token == "Bearer direct-fixture-only"
		if isDirect {
			direct.Add(1)
		} else {
			pooled.Add(1)
		}
		send := func(body []byte) (bool, []string) {
			var warm struct {
				Generate *bool `json:"generate"`
			}
			_ = json.Unmarshal(body, &warm)
			if token == "Bearer fixture-upstream-a" && (warm.Generate == nil || *warm.Generate) {
				return true, []string{`{"type":"error","status":429,"error":{"code":"usage_limit_reached"}}`}
			}
			return false, codexFixtureEvents()
		}
		if websocket.IsWebSocketUpgrade(r) {
			c, e := (&websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}).Upgrade(w, r, nil)
			if e != nil {
				return
			}
			defer c.Close()
			for {
				_, body, e := c.ReadMessage()
				if e != nil {
					return
				}
				limited, events := send(body)
				for _, v := range events {
					if c.WriteMessage(websocket.TextMessage, []byte(v)) != nil {
						return
					}
				}
				if limited {
					return
				}
			}
		}
		body, _ := io.ReadAll(r.Body)
		limited, events := send(body)
		if limited {
			w.WriteHeader(429)
			fmt.Fprint(w, events[0])
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, v := range events {
			fmt.Fprintf(w, "data: %s\n\n", v)
			w.(http.Flusher).Flush()
		}
	}))
	defer remote.Close()
	initial := `model_provider = "direct_fixture"
[model_providers.direct_fixture]
name = "Direct fixture"
base_url = ` + strconv.Quote(remote.URL) + `
wire_api = "responses"
env_key = "DIRECT_FIXTURE_TOKEN"
supports_websockets = true
`
	if e := os.WriteFile(filepath.Join(profile, "config.toml"), []byte(initial), 0600); e != nil {
		t.Fatal(e)
	}
	env := []string{}
	for _, v := range isolatedEnv(profile, "codex") {
		k, _, _ := strings.Cut(v, "=")
		if k != "HOME" && k != "XDG_CONFIG_HOME" && k != "XDG_CACHE_HOME" {
			env = append(env, v)
		}
	}
	env = append(env, "HOME="+h, "XDG_CONFIG_HOME="+h, "XDG_CACHE_HOME="+h, "DIRECT_FIXTURE_TOKEN=direct-fixture-only")
	run := func(args ...string) []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		c := exec.CommandContext(ctx, executable, args...)
		c.Env, c.Dir = env, h
		b, e := c.CombinedOutput()
		if e != nil {
			t.Fatalf("native resume failed: %v\n%s", e, b)
		}
		return b
	}
	first := run("exec", "--skip-git-repo-check", "--json", "-m", "gpt-5.4", "Remember fixture-marker. Reply with fixture-ok.")
	id := ""
	for _, line := range strings.Split(string(first), "\n") {
		var v struct {
			Type   string `json:"type"`
			Thread string `json:"thread_id"`
		}
		if json.Unmarshal([]byte(line), &v) == nil && v.Type == "thread.started" {
			id = v.Thread
		}
	}
	if id == "" || direct.Load() == 0 {
		t.Fatal("direct fixture did not create a prior session")
	}
	oldDirect := direct.Load()
	scheduler, e := core.New("", []model.Account{{ID: "a", Provider: "codex", Enabled: true, Endpoint: remote.URL}, {ID: "b", Provider: "codex", Enabled: true, Endpoint: remote.URL}})
	if e != nil {
		t.Fatal(e)
	}
	gateway := proxy.New(scheduler, nativeFixtureBackend{}, secret)
	local := httptest.NewServer(gateway)
	defer local.Close()
	defer gateway.Shutdown(context.Background())
	if e = integration.Install(pool, "codex", profile, helper, strings.TrimPrefix(local.URL, "http://")); e != nil {
		t.Fatal(e)
	}
	// App-server differs from `codex exec resume`: it restores the provider
	// from the saved rollout even when the current config says Noria. Verify
	// the opt-in stdio adapter overrides it before the *first* thread/resume,
	// without sending a model turn or changing the saved thread history.
	if e = config.Change(pool, func(c *config.Config) error {
		c.Listen = strings.TrimPrefix(local.URL, "http://")
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		appServer := exec.CommandContext(ctx, helper, "run", "codex", "--", "app-server")
		appServer.Env = append(append([]string{}, env...), "NORIA_HOME="+pool)
		appServer.Dir = h
		stdin, err := appServer.StdinPipe()
		if err != nil {
			t.Fatal(err)
		}
		stdout, err := appServer.StdoutPipe()
		if err != nil {
			t.Fatal(err)
		}
		if err = appServer.Start(); err != nil {
			t.Fatal(err)
		}
		defer func() {
			// The native app-server owns a session writer. Let it close naturally
			// after stdin EOF before a second Codex process resumes that thread.
			_ = stdin.Close()
			if err := appServer.Wait(); err != nil {
				t.Fatalf("native app-server did not exit cleanly: %v", err)
			}
		}()
		encoder := json.NewEncoder(stdin)
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 4096), 4<<20)
		if err = encoder.Encode(map[string]any{"id": 1, "method": "initialize", "params": map[string]any{"clientInfo": map[string]string{"name": "noria-compat", "title": "Noria", "version": "0.1.0"}}}); err != nil {
			t.Fatal(err)
		}
		for scanner.Scan() {
			var reply struct {
				ID     int `json:"id"`
				Result struct {
					ModelProvider string `json:"modelProvider"`
				} `json:"result"`
				Error json.RawMessage `json:"error"`
			}
			if json.Unmarshal(scanner.Bytes(), &reply) != nil {
				t.Fatal("invalid app-server JSON")
			}
			if reply.ID == 1 {
				if err = encoder.Encode(map[string]any{"method": "initialized", "params": map[string]any{}}); err != nil {
					t.Fatal(err)
				}
				if err = encoder.Encode(map[string]any{"id": 2, "method": "thread/resume", "params": map[string]any{"threadId": id, "excludeTurns": true}}); err != nil {
					t.Fatal(err)
				}
			}
			if reply.ID == 2 {
				if reply.Result.ModelProvider != "noria_runner" {
					t.Fatalf("legacy app-server thread did not select Noria: provider=%q error=%s", reply.Result.ModelProvider, string(reply.Error)[:min(380, len(reply.Error))])
				}
				return
			}
		}
		t.Fatalf("app-server closed before legacy thread/resume: %v", scanner.Err())
	}()
	second := run("exec", "resume", "--skip-git-repo-check", "--json", id, "Continue. Reply with fixture-ok.")
	if !strings.Contains(string(second), "fixture-ok") || direct.Load() != oldDirect || pooled.Load() < 2 {
		t.Fatalf("prior session bypassed persistent routing: old-direct=%d now=%d pooled=%d\n%s", oldDirect, direct.Load(), pooled.Load(), second)
	}
	t.Log("A session created before integration resumed through the configured Noria provider and failed over to another pool account")
}
