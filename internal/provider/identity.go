package provider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/amxv/noria/internal/model"
)

type Identity struct {
	Email         string `json:"email"`
	AccountID     string `json:"account_id,omitempty"`
	Authenticated bool   `json:"authenticated"`
}

func (m *Manager) Identity(ctx context.Context, a model.Account) (Identity, error) {
	p, e := m.profile(a)
	if e != nil {
		return Identity{}, e
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if a.Provider == "codex" {
		b, e := os.ReadFile(filepath.Join(p, "auth.json"))
		if e != nil {
			return Identity{}, errors.New("Codex profile is not signed in")
		}
		var root struct {
			Tokens struct {
				IDToken string `json:"id_token"`
				Access  string `json:"access_token"`
				Account string `json:"account_id"`
			} `json:"tokens"`
		}
		if json.Unmarshal(b, &root) != nil || root.Tokens.Access == "" {
			return Identity{}, errors.New("Codex profile is not signed in")
		}
		email := displayEmailFromJWT(root.Tokens.IDToken)
		if email == "" {
			email = displayEmailFromJWT(root.Tokens.Access)
		}
		return Identity{Email: email, AccountID: root.Tokens.Account, Authenticated: true}, nil
	}
	cmd := nativeCommand(ctx, cleanEnv(p, "claude"), m.claudeCommand, "auth", "status", "--json")
	cmd.Dir = p
	buffer := &boundedBuffer{max: 1 << 20}
	cmd.Stdout = buffer
	if e = cmd.Run(); e != nil {
		return Identity{}, errors.New("Claude profile is not signed in")
	}
	var identity struct {
		LoggedIn bool   `json:"loggedIn"`
		Email    string `json:"email"`
		UUID     string `json:"accountUuid"`
	}
	if json.Unmarshal(buffer.data, &identity) != nil {
		return Identity{}, errors.New("Claude identity response changed")
	}
	return Identity{Email: displayEmail(identity.Email), AccountID: identity.UUID, Authenticated: identity.LoggedIn}, nil
}

// Native auth storage is trusted for display only; JWT parsing grants no access.
func displayEmailFromJWT(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}
	b, e := base64.RawURLEncoding.DecodeString(parts[1])
	if e != nil {
		return ""
	}
	var claims map[string]any
	if json.Unmarshal(b, &claims) != nil {
		return ""
	}
	if value, ok := claims["email"].(string); ok {
		return displayEmail(value)
	}
	for _, key := range []string{"https://api.openai.com/profile", "profile"} {
		if profile, ok := claims[key].(map[string]any); ok {
			if email, ok := profile["email"].(string); ok {
				return displayEmail(email)
			}
		}
	}
	return ""
}

// BrowserLogin retains the user-facing action while the native control reader
// owns token issuance and browser callbacks.
func (m *Manager) BrowserLogin(ctx context.Context, a model.Account) error { return m.Login(ctx, a) }

func displayEmail(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 254 || strings.ContainsAny(s, "\r\n\t\x00") {
		return ""
	}
	m, e := mail.ParseAddress(s)
	if e != nil || m.Address != s {
		return ""
	}
	return s
}
