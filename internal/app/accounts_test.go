package app

import (
	"bytes"
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
	if e := Run([]string{"account", "add", "a", "--provider", "codex"}, &output, &output); e != nil {
		t.Fatal(e)
	}
	if e := config.Change(home, func(c *config.Config) error {
		c.Accounts[0].Email = "person@example.com"
		c.Accounts[0].Authenticated = true
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
	if c.Accounts[0].ID != "a" || c.Accounts[0].Email != "person@example.com" || c.Accounts[0].Alias != "Work" {
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
