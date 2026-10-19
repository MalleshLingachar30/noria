package app

import (
	"bytes"
	"github.com/amxv/noria/internal/config"
	"github.com/amxv/noria/internal/core"
	"github.com/amxv/noria/internal/integration"
	"github.com/amxv/noria/internal/model"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmailAliasChangesKeepInternalIDAndSessionAffinity(t *testing.T) {
	h := t.TempDir()
	t.Setenv("NORIA_HOME", h)
	if e := config.Init(h); e != nil {
		t.Fatal(e)
	}
	a := model.Account{ID: "account-stable", Provider: "codex", Email: "alex@example.com", Profile: filepath.Join(h, "accounts", "account-stable"), Enabled: true, Authenticated: true}
	_ = config.PrivateDir(a.Profile)
	if e := config.Change(h, func(c *config.Config) error { c.Accounts = []model.Account{a}; return nil }); e != nil {
		t.Fatal(e)
	}
	scheduler, e := core.New(filepath.Join(h, "scheduler.json"), []model.Account{a})
	if e != nil {
		t.Fatal(e)
	}
	lease, e := scheduler.Acquire("codex", "session", "fixture-model", nil)
	if e != nil {
		t.Fatal(e)
	}
	_ = scheduler.Accept(lease)
	_ = scheduler.Complete(lease)
	for _, alias := range []string{"Work", ""} {
		if e := Run([]string{"account", "rename", a.Email, alias, "--json"}, io.Discard, io.Discard); e != nil {
			t.Fatal(e)
		}
		c, _ := config.Load(h)
		if c.Accounts[0].ID != a.ID || c.Accounts[0].Email != a.Email || c.Accounts[0].Alias != alias {
			t.Fatal("rename changed identity")
		}
		_ = scheduler.SetAccounts(c.Accounts)
		lease, e = scheduler.Acquire("codex", "session", "fixture-model", nil)
		if e != nil || lease.Account.ID != a.ID {
			t.Fatal("rename moved session")
		}
		_ = scheduler.Complete(lease)
	}
	var out bytes.Buffer
	if e := Run([]string{"account", "list"}, &out, io.Discard); e != nil {
		t.Fatal(e)
	}
	if !strings.HasPrefix(out.String(), a.Email) {
		t.Fatal("email was not default account display")
	}
}
func TestPublicServeIsRemoved(t *testing.T) {
	t.Setenv("NORIA_HOME", t.TempDir())
	if e := Run([]string{"serve"}, io.Discard, io.Discard); e == nil {
		t.Fatal("public foreground serve remained")
	}
}
func TestCodexIntegrationCanBeRemovedThroughCLI(t *testing.T) {
	h := t.TempDir()
	r := t.TempDir()
	t.Setenv("NORIA_HOME", h)
	_ = config.Init(h)
	exe := "/fixture/noria"
	if e := integrationInstallForTest(h, r, exe); e != nil {
		t.Fatal(e)
	}
	if e := Run([]string{"integrate", "codex", "--client-home", r, "--remove", "--yes", "--json"}, io.Discard, io.Discard); e != nil {
		t.Fatal(e)
	}
}

func integrationInstallForTest(home, root, exe string) error {
	return integration.Install(home, "codex", root, exe, "127.0.0.1:43119")
}
