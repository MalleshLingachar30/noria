package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"github.com/amxv/noria/internal/host"
	"github.com/amxv/noria/internal/model"
	"io"
	"net/url"
	"time"
)

var openLoginURL = host.OpenURL

// Login delegates credential issuance/storage to the provider's native client.
// GUI callers receive no tokens, auth file contents or raw process output.
func (m *Manager) Login(ctx context.Context, a model.Account) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	p, e := m.profile(a)
	if e != nil {
		return e
	}
	unlock, e := m.lock(ctx, p)
	if e != nil {
		return e
	}
	defer unlock()
	if a.Provider == "codex" {
		return m.codexLogin(ctx, p)
	}
	cmd := nativeCommand(ctx, cleanEnv(p, "claude"), m.claudeCommand, "auth", "login", "--claudeai")
	cmd.Dir = p
	cmd.Stdout = io.Discard
	cmd.Stdin = nil
	// Native Claude opens the system browser and handles its local OAuth callback.
	if e = cmd.Run(); e != nil {
		if ctx.Err() != nil {
			return errors.New("Claude sign-in cancelled or timed out")
		}
		return errors.New("Claude sign-in failed; retry or use the native CLI in a terminal")
	}
	return nil
}
func (m *Manager) codexLogin(ctx context.Context, p string) error {
	cmd := nativeCommand(ctx, cleanEnv(p, "codex"), m.codexCommand, "app-server", "-c", `cli_auth_credentials_store="file"`, "-c", `model_provider="openai"`)
	cmd.Dir = p
	in, e := cmd.StdinPipe()
	if e != nil {
		return e
	}
	out, e := cmd.StdoutPipe()
	if e != nil {
		return e
	}
	if e = cmd.Start(); e != nil {
		return errors.New("install Codex before signing in")
	}
	defer func() { _ = in.Close(); killProcess(cmd); _ = cmd.Wait() }()
	enc := json.NewEncoder(in)
	if e = enc.Encode(map[string]any{"id": 1, "method": "initialize", "params": map[string]any{"clientInfo": map[string]string{"name": "noria", "version": "0.2.0"}}}); e != nil {
		return e
	}
	scan := bufio.NewScanner(out)
	scan.Buffer(make([]byte, 4096), 1<<20)
	loginID := ""
	for scan.Scan() {
		var m struct {
			ID     int             `json:"id"`
			Method string          `json:"method"`
			Result json.RawMessage `json:"result"`
			Error  json.RawMessage `json:"error"`
			Params json.RawMessage `json:"params"`
		}
		if json.Unmarshal(scan.Bytes(), &m) != nil {
			return errors.New("invalid Codex login response")
		}
		if len(m.Error) > 0 && string(m.Error) != "null" {
			return errors.New("Codex login was rejected")
		}
		if m.ID == 1 {
			_ = enc.Encode(map[string]any{"method": "initialized"})
			if e = enc.Encode(map[string]any{"id": 2, "method": "account/login/start", "params": map[string]string{"type": "chatgpt"}}); e != nil {
				return e
			}
		}
		if m.ID == 2 {
			var r struct {
				URL string `json:"authUrl"`
				ID  string `json:"loginId"`
			}
			if json.Unmarshal(m.Result, &r) != nil {
				return errors.New("invalid Codex sign-in URL")
			}
			u, e := url.Parse(r.URL)
			if e != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || r.ID == "" || (u.Hostname() != "auth.openai.com" && u.Hostname() != "chatgpt.com") {
				return errors.New("unexpected provider login URL")
			}
			loginID = r.ID
			if e = openLoginURL(r.URL); e != nil {
				return errors.New("could not open the provider sign-in page in your default browser")
			}
		}
		if m.Method == "account/login/completed" {
			var r struct {
				ID      string `json:"loginId"`
				Success bool   `json:"success"`
			}
			_ = json.Unmarshal(m.Params, &r)
			if loginID == "" || r.ID != loginID {
				continue
			}
			if !r.Success {
				return errors.New("Codex sign-in did not complete")
			}
			return nil
		}
	}
	return errors.New("Codex sign-in closed or timed out")
}
