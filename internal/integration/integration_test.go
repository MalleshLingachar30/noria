package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amxv/noria/internal/config"
	"github.com/pelletier/go-toml/v2"
)

func fixture(t *testing.T, provider, body string) (string, string, string) {
	t.Helper()
	home := t.TempDir()
	root := t.TempDir()
	if e := config.Init(home); e != nil {
		t.Fatal(e)
	}
	path, e := TargetPath(provider, root)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(path, []byte(body), 0600); e != nil {
		t.Fatal(e)
	}
	return home, root, path
}
func TestCodexSettingsRoundTripRetainsCommentsLoginAndUnrelatedEdits(t *testing.T) {
	original := "# my settings\nmodel_provider = \"openai\" # preferred\nmodel = \"fixture-model\"\n[features]\nsome_feature = true\n"
	h, r, p := fixture(t, "codex", original)
	auth := filepath.Join(r, "auth.json")
	_ = os.WriteFile(auth, []byte("login-sentinel"), 0600)
	if _, e := Preview(h, "codex", r, "/safe/noria", "127.0.0.1:43119"); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(p)
	if string(b) != original {
		t.Fatal("preview changed file")
	}
	if e := Install(h, "codex", r, "/safe/noria", "127.0.0.1:43119"); e != nil {
		t.Fatal(e)
	}
	b, _ = os.ReadFile(p)
	var m map[string]any
	if e := toml.Unmarshal(b, &m); e != nil {
		t.Fatal(e)
	}
	if m["model_provider"] != "noria" || !bytes.Contains(b, []byte("# my settings")) {
		t.Fatal("integration lost routing/comments")
	}
	b = bytes.Replace(b, []byte("some_feature = true"), []byte("some_feature = false # edited while using Noria"), 1)
	_ = os.WriteFile(p, b, 0600)
	if e := SetActive(h, "/safe/noria", "127.0.0.1:43119", false); e != nil {
		t.Fatal(e)
	}
	b, _ = os.ReadFile(p)
	if bytes.Contains(b, []byte("model_providers.noria")) || !bytes.Contains(b, []byte(`model_provider = "openai" # preferred`)) || !bytes.Contains(b, []byte("false # edited while using Noria")) {
		t.Fatalf("direct settings restoration lost user data: %s", b)
	}
	login, _ := os.ReadFile(auth)
	if string(login) != "login-sentinel" {
		t.Fatal("native login changed")
	}
	if e := SetActive(h, "/safe/noria", "127.0.0.1:43119", true); e != nil {
		t.Fatal(e)
	}
	if e := Remove(h, "codex", r); e != nil {
		t.Fatal(e)
	}
	s, _ := Load(h)
	if len(s.Entries) != 0 {
		t.Fatal("remove did not unregister")
	}
}
func TestClaudePreservesAndRestoresCredentialOverrides(t *testing.T) {
	original := `{"apiKeyHelper":"original-helper","env":{"ANTHROPIC_BASE_URL":"https://original.invalid","ANTHROPIC_AUTH_TOKEN":"old-token","ANTHROPIC_API_KEY":"old-key","CLAUDE_CODE_OAUTH_TOKEN":"old-oauth","OTHER":"sentinel"},"theme":"dark"}`
	h, r, p := fixture(t, "claude", original)
	if e := Install(h, "claude", r, "/safe/noria", "127.0.0.1:43119"); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(p)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	env := m["env"].(map[string]any)
	for _, key := range []string{"ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_API_KEY", "CLAUDE_CODE_OAUTH_TOKEN"} {
		if env[key] != "" {
			t.Fatal("native credential override could bypass local auth helper")
		}
	}
	m["theme"] = "light"
	b, _ = json.Marshal(m)
	_ = os.WriteFile(p, b, 0600)
	if e := SetActive(h, "/safe/noria", "127.0.0.1:43119", false); e != nil {
		t.Fatal(e)
	}
	b, _ = os.ReadFile(p)
	_ = json.Unmarshal(b, &m)
	env = m["env"].(map[string]any)
	if env["ANTHROPIC_AUTH_TOKEN"] != "old-token" || m["apiKeyHelper"] != "original-helper" || m["theme"] != "light" || env["OTHER"] != "sentinel" {
		t.Fatalf("restoration clobbered original or newer settings: %s", b)
	}
}
func TestExternalRoutingEditsAreNotOverwritten(t *testing.T) {
	h, r, p := fixture(t, "claude", `{}`)
	if e := Install(h, "claude", r, "/safe/noria", "127.0.0.1:43119"); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(p)
	b = bytes.Replace(b, []byte("http://127.0.0.1:43119/claude"), []byte("https://other-gateway.invalid"), 1)
	_ = os.WriteFile(p, b, 0600)
	if e := SetActive(h, "/safe/noria", "127.0.0.1:43119", false); e == nil {
		t.Fatal("overwrote an external routing change")
	}
	after, _ := os.ReadFile(p)
	if !bytes.Equal(b, after) {
		t.Fatal("file changed despite conflict")
	}
}
func TestRefusesUpstreamAuthProfileIntegration(t *testing.T) {
	h := t.TempDir()
	_ = config.Init(h)
	root := filepath.Join(h, "accounts", "fixture")
	_ = os.MkdirAll(root, 0700)
	if e := Install(h, "claude", root, "/safe/noria", "127.0.0.1:43119"); e == nil {
		t.Fatal("routed upstream credential profile back into Noria")
	}
}
func TestHelperConfigurationContainsNoGatewaySecret(t *testing.T) {
	h, r, p := fixture(t, "codex", "")
	secret, _ := config.Secret(h)
	if e := Install(h, "codex", r, "/safe/noria", "127.0.0.1:43119"); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(p)
	if strings.Contains(string(b), secret) || !strings.Contains(string(b), "auth-token") {
		t.Fatal("helper missing or secret persisted to client config")
	}
}
