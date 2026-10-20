package provider

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/amxv/noria/internal/model"
)

// Claude Code's public OAuth client ID. Token rotation is handled against
// Anthropic's endpoint without sending a model prompt or invoking native login.
const claudeOAuthClientID = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"

// Call only while holding the account's native.lock. A rotated refresh token
// must be committed to its original native store before it is used by a proxy.
func (m *Manager) refreshClaudeOAuth(ctx context.Context, a model.Account, profile string) error {
	data, service, username, err := nativeCredentialData(ctx, a, profile)
	if err != nil {
		return err
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(data, &root) != nil {
		return errors.New("invalid native Claude credentials")
	}
	var oauth map[string]json.RawMessage
	if json.Unmarshal(root["claudeAiOauth"], &oauth) != nil {
		return errors.New("native Claude OAuth credentials unavailable")
	}
	var refreshToken string
	if json.Unmarshal(oauth["refreshToken"], &refreshToken) != nil || refreshToken == "" {
		return errors.New("Claude OAuth refresh token unavailable; sign in again")
	}
	requestFields := map[string]string{
		"grant_type": "refresh_token", "refresh_token": refreshToken, "client_id": claudeOAuthClientID,
	}
	var scopes []string
	if len(oauth["scopes"]) != 0 && json.Unmarshal(oauth["scopes"], &scopes) == nil && len(scopes) > 0 {
		// Preserve this account's granted scope set rather than requesting
		// additional permissions during a non-interactive rotation.
		requestFields["scope"] = strings.Join(scopes, " ")
	}
	requestBody, _ := json.Marshal(requestFields)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.claudeTokenURL, bytes.NewReader(requestBody))
	if err != nil {
		return errors.New("cannot construct Claude OAuth refresh request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := m.usageClient.Do(req)
	if err != nil {
		return errors.New("Claude OAuth refresh endpoint unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized {
			return errors.New("Claude OAuth refresh rejected; sign in again")
		}
		return fmt.Errorf("Claude OAuth refresh returned HTTP %d", resp.StatusCode)
	}
	responseBytes, err := io.ReadAll(io.LimitReader(resp.Body, (64<<10)+1))
	if err != nil || len(responseBytes) > 64<<10 {
		return errors.New("invalid Claude OAuth refresh response size")
	}
	var rotated struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if json.Unmarshal(responseBytes, &rotated) != nil || rotated.AccessToken == "" || rotated.ExpiresIn < 1 || rotated.ExpiresIn > 7*24*3600 {
		return errors.New("invalid Claude OAuth refresh response")
	}
	if rotated.RefreshToken == "" {
		rotated.RefreshToken = refreshToken
	}
	update := func(key string, value any) {
		encoded, _ := json.Marshal(value)
		oauth[key] = encoded
	}
	update("accessToken", rotated.AccessToken)
	update("refreshToken", rotated.RefreshToken)
	update("expiresAt", time.Now().Add(time.Duration(rotated.ExpiresIn)*time.Second).UnixMilli())
	root["claudeAiOauth"], _ = json.Marshal(oauth)
	updated, _ := json.Marshal(root)
	if service != "" {
		return updateClaudeKeychain(ctx, a, profile, service, username, updated)
	}
	return writeClaudeCredentialFile(profile, updated)
}

func updateClaudeKeychain(ctx context.Context, a model.Account, profile, service, username string, updated []byte) error {
	// Use security's stdin command interpreter: never put secrets on argv, in
	// environment variables, or into CLI output/logs. -U updates in place,
	// preserving the existing item's Keychain access-control policy.
	command := "add-generic-password -U -a " + strconv.Quote(username) + " -s " + strconv.Quote(service) + " -X " + hex.EncodeToString(updated) + "\n"
	cmd := exec.CommandContext(ctx, "security", "-i")
	cmd.Stdin = strings.NewReader(command)
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	if err := cmd.Run(); err != nil {
		return errors.New("cannot update native Claude Keychain credential")
	}
	// security -i may exit successfully despite a failed command. Read back
	// the exact item and fail closed if the rotated secret was not persisted.
	actual, actualService, actualUser, err := nativeCredentialData(ctx, a, profile)
	if err != nil || actualService != service || actualUser != username || !bytes.Equal(bytes.TrimSpace(actual), updated) {
		return errors.New("native Claude Keychain credential update was not verified")
	}
	return nil
}

func writeClaudeCredentialFile(profile string, updated []byte) error {
	path := filepath.Join(profile, ".credentials.json")
	if info, err := os.Lstat(path); err != nil || !info.Mode().IsRegular() {
		return errors.New("Claude credential file is unavailable or unsafe")
	}
	file, err := os.CreateTemp(profile, ".noria-credential-*")
	if err != nil {
		return errors.New("cannot stage Claude credential rotation")
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(0600); err != nil {
		file.Close()
		return errors.New("cannot protect rotated Claude credentials")
	}
	_, err = file.Write(updated)
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil || closeErr != nil || os.Rename(file.Name(), path) != nil {
		return errors.New("cannot persist Claude credential rotation")
	}
	return nil
}
