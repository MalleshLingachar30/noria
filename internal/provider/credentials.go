package provider

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/amxv/noria/internal/model"
)

func (m *Manager) readCredential(ctx context.Context, a model.Account, p string) (model.Credential, bool, error) {
	data, _, _, err := nativeCredentialData(ctx, a, p)
	if err != nil {
		return model.Credential{}, false, err
	}
	return parseCredential(a.Provider, data)
}

// nativeCredentialData returns the provider-owned credential payload and, when
// stored in macOS Keychain, the exact item identity to update on rotation.
// Keychain takes precedence over the profile file, matching Claude Code.
func nativeCredentialData(ctx context.Context, a model.Account, p string) ([]byte, string, string, error) {
	name := "auth.json"
	if a.Provider == "claude" {
		name = ".credentials.json"
	}
	// Native macOS Claude prioritizes its config-dir scoped keychain entry.
	var data []byte
	if a.Provider == "claude" && runtime.GOOS == "darwin" {
		who, e := user.Current()
		if e == nil {
			sum := sha256.Sum256([]byte(p))
			service := "Claude Code-credentials-" + hex.EncodeToString(sum[:])[:8]
			keyctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			cmd := exec.CommandContext(keyctx, "security", "find-generic-password", "-a", who.Username, "-w", "-s", service)
			cmd.Stderr = nil
			data, _ = cmd.Output()
			cancel()
			if len(data) != 0 {
				return data, service, who.Username, nil
			}
		}
	}
	if len(data) == 0 {
		path := filepath.Join(p, name)
		info, e := os.Lstat(path)
		if e != nil || !info.Mode().IsRegular() {
			return nil, "", "", errors.New("native credentials unavailable; run account login")
		}
		if info.Size() > 1<<20 {
			return nil, "", "", errors.New("native credentials file is too large")
		}
		data, e = os.ReadFile(path)
		if e != nil {
			return nil, "", "", errors.New("cannot read native credential file")
		}
	}
	return data, "", "", nil
}

func parseCredential(provider string, data []byte) (model.Credential, bool, error) {
	var root map[string]json.RawMessage
	if json.Unmarshal(data, &root) != nil {
		return model.Credential{}, false, errors.New("invalid native credentials")
	}
	var c model.Credential
	var expires time.Time
	if provider == "codex" {
		var tokens struct {
			Access  string `json:"access_token"`
			Account string `json:"account_id"`
		}
		_ = json.Unmarshal(root["tokens"], &tokens)
		c.Token = tokens.Access
		c.AccountID = tokens.Account
		// JWT claims are used solely for expiry; they never establish identity/trust.
		parts := strings.Split(c.Token, ".")
		if len(parts) == 3 {
			if b, e := base64.RawURLEncoding.DecodeString(parts[1]); e == nil {
				var claims struct {
					Exp int64 `json:"exp"`
				}
				if json.Unmarshal(b, &claims) == nil && claims.Exp > 0 {
					expires = time.Unix(claims.Exp, 0)
				}
			}
		}
	} else {
		var oauth struct {
			Access  string `json:"accessToken"`
			Expires int64  `json:"expiresAt"`
		}
		if json.Unmarshal(root["claudeAiOauth"], &oauth) != nil {
			return c, false, errors.New("native Claude OAuth credentials unavailable; run account login")
		}
		c.Token = oauth.Access
		if oauth.Expires > 0 {
			expires = time.UnixMilli(oauth.Expires)
		}
	}
	if c.Token == "" {
		return c, false, errors.New("native account has no access token; run account login")
	}
	return c, !expires.IsZero() && !expires.After(time.Now().Add(time.Minute)), nil
}
