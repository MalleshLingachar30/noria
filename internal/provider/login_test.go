package provider

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Self-hosted fixture executable replaces both native providers. It grants no
// real credentials, opens no browser, and fails if a model/user turn is sent.
func nativeLoginFixture() {
	mode := os.Getenv("GO_LOGIN_KIND")
	profile := os.Getenv("CODEX_HOME")
	if mode == "claude" {
		profile = os.Getenv("CLAUDE_CONFIG_DIR")
	}
	if profile == "" || os.Getenv("OPENAI_API_KEY") != "" || os.Getenv("ANTHROPIC_AUTH_TOKEN") != "" || os.Getenv("NORIA_GATEWAY_TOKEN") != "" {
		os.Exit(4)
	}
	enc := json.NewEncoder(os.Stdout)
	if mode == "claude" {
		args := strings.Join(os.Args[1:], " ")
		if args == "auth login --claudeai" {
			_ = os.WriteFile(filepath.Join(profile, "login-ok"), []byte("native-subscription-login"), 0600)
			return
		}
		if args == "auth status --json" {
			if _, e := os.Stat(filepath.Join(profile, "login-ok")); e != nil {
				os.Exit(5)
			}
			_ = enc.Encode(map[string]any{"loggedIn": true, "email": "claude@example.com", "accountUuid": "fixture-only"})
			return
		}
		os.Exit(6)
	}
	if !strings.Contains(strings.Join(os.Args[1:], " "), `cli_auth_credentials_store="file"`) {
		os.Exit(7)
	}
	reader := bufio.NewScanner(os.Stdin)
	for reader.Scan() {
		var request struct {
			ID     int            `json:"id"`
			Method string         `json:"method"`
			Params map[string]any `json:"params"`
		}
		if json.Unmarshal(reader.Bytes(), &request) != nil {
			os.Exit(8)
		}
		switch request.Method {
		case "initialize":
			_ = enc.Encode(map[string]any{"id": request.ID, "result": map[string]any{}})
		case "initialized":
		case "account/login/start":
			if request.Params["type"] != "chatgpt" {
				os.Exit(9)
			}
			address := "https://auth.openai.com/authorize?state=fixture-secret-state"
			if os.Getenv("GO_LOGIN_BAD_URL") == "1" {
				address = "https://evil.invalid/authorize?state=fixture-secret-state"
			}
			_ = enc.Encode(map[string]any{"id": request.ID, "result": map[string]any{"authUrl": address, "loginId": "fixture-login"}})
			if os.Getenv("GO_LOGIN_HANG") == "1" {
				time.Sleep(time.Hour)
			}
			token := "fixture." + base64.RawURLEncoding.EncodeToString([]byte(`{"email":"codex@example.com"}`)) + ".fixture"
			credentials, _ := json.Marshal(map[string]any{"tokens": map[string]any{"access_token": "fake-access-token", "id_token": token, "account_id": "fake-account"}})
			_ = os.WriteFile(filepath.Join(profile, "auth.json"), credentials, 0600)
			_ = enc.Encode(map[string]any{"method": "account/login/completed", "params": map[string]any{"loginId": "unrelated", "success": true}})
			_ = enc.Encode(map[string]any{"method": "account/login/completed", "params": map[string]any{"loginId": "fixture-login", "success": true}})
		default:
			os.Exit(10)
		}
	}
}
func TestBrowserLoginUsesNativeIsolatedAuthAndReturnsEmail(t *testing.T) {
	for _, kind := range []string{"codex", "claude"} {
		t.Run(kind, func(t *testing.T) {
			m, a := fixtureProfile(t, kind)
			m.codexCommand = os.Args[0]
			m.claudeCommand = os.Args[0]
			t.Setenv("GO_PROVIDER_FIXTURE", "native-login")
			t.Setenv("GO_LOGIN_KIND", kind)
			t.Setenv("ANTHROPIC_AUTH_TOKEN", "must-not-leak")
			t.Setenv("OPENAI_API_KEY", "must-not-leak")
			old := openLoginURL
			defer func() { openLoginURL = old }()
			opened := 0
			openLoginURL = func(url string) error {
				opened++
				if !strings.HasPrefix(url, "https://auth.openai.com/") {
					t.Error("unexpected browser origin")
				}
				return nil
			}
			if e := m.Login(context.Background(), a); e != nil {
				t.Fatal(e)
			}
			identity, e := m.Identity(context.Background(), a)
			if e != nil || !identity.Authenticated || identity.Email != kind+"@example.com" {
				t.Fatalf("native identity lost: %+v %v", identity, e)
			}
			if kind == "codex" && opened != 1 {
				t.Fatal("sign-in page was not opened exactly once")
			}
		})
	}
}
func TestLoginRejectsUntrustedURLsAndRedactsOpenErrors(t *testing.T) {
	for _, mode := range []string{"origin", "open-error"} {
		t.Run(mode, func(t *testing.T) {
			m, a := fixtureProfile(t, "codex")
			m.codexCommand = os.Args[0]
			t.Setenv("GO_PROVIDER_FIXTURE", "native-login")
			t.Setenv("GO_LOGIN_KIND", "codex")
			if mode == "origin" {
				t.Setenv("GO_LOGIN_BAD_URL", "1")
			}
			old := openLoginURL
			defer func() { openLoginURL = old }()
			opened := false
			openLoginURL = func(url string) error { opened = true; return fmt.Errorf("failed %s", url) }
			e := m.Login(context.Background(), a)
			if e == nil || strings.Contains(e.Error(), "fixture-secret-state") {
				t.Fatal("unsafe login error", e)
			}
			if mode == "origin" && opened {
				t.Fatal("opened untrusted sign-in URL")
			}
		})
	}
}
func TestLoginCancellationReleasesProfile(t *testing.T) {
	m, a := fixtureProfile(t, "codex")
	m.codexCommand = os.Args[0]
	t.Setenv("GO_PROVIDER_FIXTURE", "native-login")
	t.Setenv("GO_LOGIN_KIND", "codex")
	t.Setenv("GO_LOGIN_HANG", "1")
	old := openLoginURL
	defer func() { openLoginURL = old }()
	openLoginURL = func(string) error { return nil }
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if e := m.Login(ctx, a); e == nil {
		t.Fatal("cancelled login succeeded")
	}
	unlock, e := m.lock(context.Background(), a.Profile)
	if e != nil {
		t.Fatal(e)
	}
	unlock()
}
func TestDisplayIdentityRejectsControlCharacters(t *testing.T) {
	for _, value := range []string{"bad\n@example.com", "Name <a@example.com>", "not-email"} {
		if validEmail(value) != "" {
			t.Fatal(value)
		}
	}
	if validEmail("person@example.com") != "person@example.com" {
		t.Fatal("valid email rejected")
	}
}
