package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amxv/noria/internal/model"
)

func expiredClaudeFixture(t *testing.T) (*Manager, string, string) {
	t.Helper()
	m, a := fixtureProfile(t, "claude")
	path := filepath.Join(a.Profile, ".credentials.json")
	original := `{"claudeAiOauth":{"accessToken":"expired-access","refreshToken":"previous-refresh","expiresAt":1,"scopes":["user:inference"]},"otherSettings":{"preserved":true}}`
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	return m, a.Profile, original
}

func TestClaudeOAuthRotatesExpiredCredentialsOnceAcrossManagers(t *testing.T) {
	m, profile, _ := expiredClaudeFixture(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body map[string]string
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Authorization") != "" || json.NewDecoder(r.Body).Decode(&body) != nil || body["refresh_token"] != "previous-refresh" || body["client_id"] != claudeOAuthClientID || body["grant_type"] != "refresh_token" || body["scope"] != "user:inference" {
			t.Error("Claude rotation request was incorrect")
		}
		_, _ = io.WriteString(w, `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":28800}`)
	}))
	defer server.Close()
	m.claudeTokenURL = server.URL
	m2 := New(m.home)
	m2.claudeTokenURL = server.URL
	a := modelAccount(profile)
	var workers sync.WaitGroup
	for i := 0; i < 16; i++ {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			manager := m
			if index%2 == 0 {
				manager = m2
			}
			credential, err := manager.Credentials(context.Background(), a)
			if err != nil || credential.Token != "new-access" {
				t.Errorf("expired Claude credential was not automatically refreshed: %v", err)
			}
		}(i)
	}
	workers.Wait()
	if calls.Load() != 1 {
		t.Fatalf("expected one refresh under native profile lock, got %d", calls.Load())
	}
	data, err := os.ReadFile(filepath.Join(profile, ".credentials.json"))
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]json.RawMessage
	var oauth struct {
		RefreshToken string   `json:"refreshToken"`
		ExpiresAt    int64    `json:"expiresAt"`
		Scopes       []string `json:"scopes"`
	}
	if json.Unmarshal(data, &root) != nil || json.Unmarshal(root["claudeAiOauth"], &oauth) != nil || oauth.RefreshToken != "new-refresh" || oauth.ExpiresAt < time.Now().Add(7*time.Hour).UnixMilli() || len(oauth.Scopes) != 1 || string(root["otherSettings"]) != `{"preserved":true}` {
		t.Fatal("OAuth rotation lost refresh token, expiry or other native credential fields")
	}
	if st, err := os.Stat(filepath.Join(profile, ".credentials.json")); err != nil || st.Mode().Perm() != 0600 {
		t.Fatal("rotated credentials are not private")
	}
}

func modelAccount(profile string) model.Account {
	return model.Account{ID: "fixture", Provider: "claude", Profile: profile, Enabled: true}
}

func TestClaudeOAuthRefreshFailsClosedWithoutChangingNativeCredential(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"rejected", 400, `{"error":"private-server-secret"}`},
		{"missing token", 200, `{"expires_in":28800}`},
		{"invalid expiry", 200, `{"access_token":"bad","expires_in":0}`},
		{"invalid response", 200, `{not-json}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, profile, original := expiredClaudeFixture(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()
			m.claudeTokenURL = server.URL
			credential, err := m.Credentials(context.Background(), modelAccount(profile))
			if err == nil || credential.Token != "" || strings.Contains(err.Error(), "private-server-secret") || strings.Contains(err.Error(), "previous-refresh") {
				t.Fatalf("Claude returned unsafe or successful expired credential: %+v, %v", credential, err)
			}
			data, err := os.ReadFile(filepath.Join(profile, ".credentials.json"))
			if err != nil || string(data) != original {
				t.Fatal("failed OAuth refresh mutated native credentials")
			}
		})
	}
}
