package provider

import (
	"context"
	"github.com/amxv/noria/internal/config"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestValidCredentialDoesNotWaitForQuotaLock(t *testing.T) {
	m, account := fixtureProfile(t, "codex")
	if err := os.WriteFile(filepath.Join(account.Profile, "auth.json"), []byte(`{"tokens":{"access_token":"valid-fixture-token","account_id":"fixture"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	unlock, err := config.Lock(account.Profile, "native")
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	credential, err := m.Credentials(ctx, account)
	if err != nil || credential.Token != "valid-fixture-token" {
		t.Fatalf("valid credential blocked behind telemetry: %v", err)
	}
}
