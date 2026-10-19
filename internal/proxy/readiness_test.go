package proxy

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/amxv/noria/internal/model"
)

type stalledQuotaBackend struct {
	fakeBackend
	started chan struct{}
	done    chan struct{}
}

func (b stalledQuotaBackend) Refresh(ctx context.Context, _ model.Account) (model.Snapshot, error) {
	close(b.started)
	<-ctx.Done()
	close(b.done)
	return model.Snapshot{}, ctx.Err()
}

func TestQuotaLookupDoesNotDelayFailoverAndStopsOnShutdown(t *testing.T) {
	server, local := fixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer token-a" {
			w.WriteHeader(429)
			_, _ = w.Write([]byte(`{"error":{"code":"usage_limit_reached"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"accepted-by-b"}`))
	}), "codex", 2)
	backend := stalledQuotaBackend{started: make(chan struct{}), done: make(chan struct{})}
	server.backend = backend
	done := make(chan error, 1)
	go func() {
		req, _ := http.NewRequest("POST", local.URL+"/responses", strings.NewReader(`{"model":"test","session_id":"one"}`))
		req.Header.Set("Authorization", "Bearer local-secret")
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Errorf("failover returned %d", resp.StatusCode)
			}
		}
		done <- err
	}()
	select {
	case <-backend.started:
	case <-time.After(2 * time.Second):
		t.Fatal("no rejection-triggered quota refresh")
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("pending request waited for blocked quota lookup")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-backend.done:
	default:
		t.Fatal("shutdown retained quota worker")
	}
}

func TestOAuthBetaAppendedWithoutDroppingNativeFlags(t *testing.T) {
	original := http.Header{
		"Authorization":     []string{"Bearer local-secret"},
		"Anthropic-Beta":    []string{"native-tools-2026-01-01", "another-native-beta"},
		"Anthropic-Version": []string{"2023-06-01"},
	}
	h := upstreamHeaders(original, "claude", model.Credential{Token: "upstream-token"})
	for _, beta := range []string{"native-tools-2026-01-01", "another-native-beta", "oauth-2025-04-20"} {
		if !strings.Contains(h.Get("Anthropic-Beta"), beta) {
			t.Fatalf("lost beta %s", beta)
		}
	}
	if h.Get("Authorization") != "Bearer upstream-token" || original.Get("Authorization") != "Bearer local-secret" {
		t.Fatal("auth substitution mutated source")
	}
	h = upstreamHeaders(h, "claude", model.Credential{Token: "upstream-token"})
	if strings.Count(h.Get("Anthropic-Beta"), "oauth-2025-04-20") != 1 {
		t.Fatal("duplicated OAuth marker")
	}
}

func TestRemoteEndpointsRequireTLS(t *testing.T) {
	for _, raw := range []string{"http://example.com", "http://example.com:8080"} {
		if _, err := endpoint(model.Account{Endpoint: raw}, "/responses", ""); err == nil {
			t.Fatal("accepted cleartext remote token destination")
		}
	}
	for _, raw := range []string{"http://127.0.0.1:9000", "http://[::1]:9000", "https://example.com"} {
		if _, err := endpoint(model.Account{Endpoint: raw}, "/responses", ""); err != nil {
			t.Fatal(err)
		}
	}
}
