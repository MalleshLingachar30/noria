package app

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amxv/noria/internal/config"
	"github.com/amxv/noria/internal/model"
)

func TestEmailDefaultAndAliasNeverChangeRoutingID(t *testing.T) {
	home := t.TempDir()
	t.Setenv("NORIA_HOME", home)
	var output bytes.Buffer
	if e := Run([]string{"init"}, &output, &output); e != nil {
		t.Fatal(e)
	}
	profile := filepath.Join(home, "accounts", "account-stable")
	if e := config.PrivateDir(profile); e != nil {
		t.Fatal(e)
	}
	if e := config.Change(home, func(c *config.Config) error {
		c.Accounts = []model.Account{{ID: "account-stable", Email: "person@example.com", Provider: "codex", Profile: profile, Enabled: true, Authenticated: true}}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	output.Reset()
	if e := Run([]string{"account", "list"}, &output, &output); e != nil {
		t.Fatal(e)
	}
	if !strings.HasPrefix(output.String(), "person@example.com\t") {
		t.Fatal(output.String())
	}
	if e := Run([]string{"account", "rename", "person@example.com", "Work"}, &output, &output); e != nil {
		t.Fatal(e)
	}
	c, _ := config.Load(home)
	if c.Accounts[0].ID != "account-stable" || c.Accounts[0].Email != "person@example.com" || c.Accounts[0].Alias != "Work" {
		t.Fatal("rename changed identity")
	}
	if e := Run([]string{"account", "rename", "Work"}, &output, &output); e != nil {
		t.Fatal(e)
	}
	c, _ = config.Load(home)
	if c.Accounts[0].DisplayName() != "person@example.com" {
		t.Fatal("alias was not cleared")
	}
}
func TestAccountLookupRequiresUnambiguousEmail(t *testing.T) {
	c := config.Config{Accounts: []model.Account{{ID: "a", Email: "same@example.com", Provider: "codex"}, {ID: "b", Email: "same@example.com", Provider: "claude"}}}
	if _, e := resolveAccount(c, "same@example.com"); e == nil {
		t.Fatal("ambiguous account selected")
	}
	a, e := resolveAccount(c, "b")
	if e != nil || a.ID != "b" {
		t.Fatal(a, e)
	}
}
func TestPublicServeRemoved(t *testing.T) {
	var output bytes.Buffer
	if e := Run([]string{"serve"}, &output, &output); e == nil {
		t.Fatal("blocking serve remains a public command")
	}
}
func TestLegacyNamedAccountsAndUpdateAliasAreGone(t *testing.T) {
	h := t.TempDir()
	t.Setenv("NORIA_HOME", h)
	for _, cmd := range [][]string{
		{"account", "add", "named", "--provider", "codex"},
		{"account", "login", "named"},
		{"update", "--check"},
		{"autostart", "on"},
	} {
		if e := Run(cmd, &bytes.Buffer{}, &bytes.Buffer{}); e == nil {
			t.Fatalf("unexpected deprecated command accepted: %v", cmd)
		}
	}
}
