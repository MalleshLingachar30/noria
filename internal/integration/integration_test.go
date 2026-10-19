package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amxv/noria/internal/config"
	"github.com/pelletier/go-toml/v2"
)

func TestPersistentRoutingRoundTripPreservesNativeState(t *testing.T) {
	for _, provider := range []string{"codex", "claude"} {
		t.Run(provider, func(t *testing.T) {
			home := t.TempDir()
			if e := config.Init(home); e != nil {
				t.Fatal(e)
			}
			root := t.TempDir()
			path, _ := TargetPath(provider, root)
			original := []byte("# Personal config\nmodel_provider = \"openai\" # keep this comment\nmodel = \"fixture\"\n[projects.\"/my/project\"]\ntrust_level = \"trusted\"\n")
			if provider == "claude" {
				original = []byte(`{"apiKeyHelper":"original-helper","env":{"KEEP_ME":"yes"},"permissions":{"allow":["Read"]}}`)
			}
			if e := os.WriteFile(path, original, 0600); e != nil {
				t.Fatal(e)
			}
			sentinel := filepath.Join(root, "auth-sentinel")
			_ = os.WriteFile(sentinel, []byte("original-login"), 0600)
			executable := filepath.Join(t.TempDir(), "path with spaces", "noria")
			if _, e := Preview(home, provider, root, executable, "127.0.0.1:43119"); e != nil {
				t.Fatal(e)
			}
			untouched, _ := os.ReadFile(path)
			if string(untouched) != string(original) {
				t.Fatal("preview changed user configuration")
			}
			if e := Install(home, provider, root, executable, "127.0.0.1:43119"); e != nil {
				t.Fatal(e)
			}
			if e := Install(home, provider, root, executable, "127.0.0.1:43119"); e != nil {
				t.Fatal("idempotent install", e)
			}
			if _, e := Preview(home, provider, root, executable, "127.0.0.1:43119"); e != nil {
				t.Fatal("registered preview", e)
			}
			installed, _ := os.ReadFile(path)
			secret, _ := config.Secret(home)
			if strings.Contains(string(installed), secret) {
				t.Fatal("integration put local secret into config")
			}
			if !strings.Contains(string(installed), "auth-token") {
				t.Fatal("auth helper missing")
			}
			if provider == "codex" {
				installed = append(installed, []byte("\n[unrelated]\nnew_setting = 3\n")...)
			} else {
				var data map[string]any
				_ = json.Unmarshal(installed, &data)
				data["theme"] = "dark"
				installed, _ = json.Marshal(data)
			}
			_ = os.WriteFile(path, installed, 0600)
			for round := 0; round < 2; round++ {
				if e := SetActive(home, executable, "127.0.0.1:43119", false); e != nil {
					t.Fatal("restore", e)
				}
				restored, _ := os.ReadFile(path)
				var decoded map[string]any
				if provider == "codex" {
					if e := toml.Unmarshal(restored, &decoded); e != nil {
						t.Fatal(e)
					}
					if decoded["model_provider"] != "openai" || decoded["unrelated"] == nil || !strings.Contains(string(restored), "# keep this comment") {
						t.Fatal("lost original or unrelated TOML", string(restored))
					}
				}
				if provider == "claude" {
					_ = json.Unmarshal(restored, &decoded)
					if decoded["apiKeyHelper"] != "original-helper" || decoded["theme"] != "dark" {
						t.Fatal("lost original or unrelated JSON", string(restored))
					}
					env := decoded["env"].(map[string]any)
					if env["KEEP_ME"] != "yes" || env["ANTHROPIC_BASE_URL"] != nil {
						t.Fatal("environment not restored")
					}
				}
				if round == 0 {
					if e := SetActive(home, executable, "127.0.0.1:43119", true); e != nil {
						t.Fatal(e)
					}
				}
			}
			native, _ := os.ReadFile(sentinel)
			if string(native) != "original-login" {
				t.Fatal("native login modified")
			}
		})
	}
}
func TestExternalOwnedEditsAreNeverOverwritten(t *testing.T) {
	for _, provider := range []string{"codex", "claude"} {
		t.Run(provider, func(t *testing.T) {
			home := t.TempDir()
			_ = config.Init(home)
			root := t.TempDir()
			path, _ := TargetPath(provider, root)
			if e := Install(home, provider, root, "/fixture/noria", "127.0.0.1:43119"); e != nil {
				t.Fatal(e)
			}
			data, _ := os.ReadFile(path)
			data = []byte(strings.ReplaceAll(string(data), "43119", "43210"))
			_ = os.WriteFile(path, data, 0600)
			if e := SetActive(home, "/fixture/noria", "127.0.0.1:43119", false); e == nil {
				t.Fatal("external routing modification was overwritten")
			}
			after, _ := os.ReadFile(path)
			if string(after) != string(data) {
				t.Fatal("failed restore mutated settings")
			}
		})
	}
}
func TestCustomRootsAndAliasesDoNotTouchOtherProfiles(t *testing.T) {
	home := t.TempDir()
	_ = config.Init(home)
	user := t.TempDir()
	t.Setenv("HOME", user)
	t.Setenv("USERPROFILE", user)
	custom := filepath.Join(user, "custom")
	t.Setenv("CODEX_HOME", custom)
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(user, "claude-custom"))
	path, e := TargetPath("codex", "")
	if e != nil || path != filepath.Join(custom, "config.toml") {
		t.Fatal(path, e)
	}
	if _, e = Preview(home, "codex", "", filepath.Join(user, "noria"), "127.0.0.1:43119"); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(custom); !os.IsNotExist(e) {
		t.Fatal("preview created native profile")
	}
}
func TestRemoveActiveCodexRegistration(t *testing.T) {
	home := t.TempDir()
	_ = config.Init(home)
	root := t.TempDir()
	if e := Install(home, "codex", root, "/fixture/noria", "127.0.0.1:43119"); e != nil {
		t.Fatal(e)
	}
	if e := Remove(home, "codex", root); e != nil {
		t.Fatal(e)
	}
	s, _ := Load(home)
	if len(s.Entries) != 0 {
		t.Fatal("still registered")
	}
}
