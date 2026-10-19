package provider

import (
	"context"
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
	if err := os.WriteFile(filepath.Join(account.Profile, "native.lock"), []byte("active quota read"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	credential, err := m.Credentials(ctx, account)
	if err != nil || credential.Token != "valid-fixture-token" {
		t.Fatalf("valid credential blocked behind telemetry: %v", err)
	}
}
