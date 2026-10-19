package integration

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/amxv/noria/internal/config"
)

// A real Codex config can contain nested arrays of tables, including the
// [[skills.config]] table used by installed skills. The TOML decoder panics
// when asked to reuse a map containing that slice on the validation pass.
// Menu-bar Preview must never panic or modify the user's original settings.
func TestCodexNestedArrayTablesPreviewInstallRestore(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	if err := config.Init(home); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "config.toml")
	original := []byte(`# regular Codex config with nested tables and arrays
model = "gpt-6"
model_provider = "openai" # keep my direct login
notify = ["echo", "ready"]

[features]
multi_agent = true

[skills]
enabled = true

[[skills.config]]
path = "/tmp/skill-alpha"
enabled = true

[[skills.config]]
path = "/tmp/skill-beta"
enabled = false

[hooks]
enabled = true
`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Preview(home, "codex", root, "/tmp/noria-fixture", "127.0.0.1:43119"); err != nil {
		t.Fatalf("preview: %v", err)
	}
	read, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(read, original) {
		t.Fatal("preview modified Codex settings")
	}
	if err := Install(home, "codex", root, "/tmp/noria-fixture", "127.0.0.1:43119"); err != nil {
		t.Fatalf("install: %v", err)
	}
	if err := SetActive(home, "/tmp/noria-fixture", "127.0.0.1:43119", false); err != nil {
		t.Fatalf("restore direct settings: %v", err)
	}
	read, err = os.ReadFile(path)
	if err != nil || !bytes.Equal(read, original) {
		t.Fatalf("Codex settings changed after restore: %s", read)
	}
	if err := SetActive(home, "/tmp/noria-fixture", "127.0.0.1:43119", true); err != nil {
		t.Fatalf("reactivate: %v", err)
	}
	if err := Remove(home, "codex", root); err != nil {
		t.Fatalf("remove: %v", err)
	}
	read, err = os.ReadFile(path)
	if err != nil || !bytes.Equal(read, original) {
		t.Fatalf("Codex settings changed after remove: %s", read)
	}
}
