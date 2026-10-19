package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// This subprocess acts as the installed Claude binary and implements only
// its control transport. It rejects any user prompt and any credential
// inherited from the caller. No Node runtime or SDK is involved.
func claudeFixture() {
	p := os.Getenv("CLAUDE_CONFIG_DIR")
	if p == "" || os.Getenv("OPENAI_API_KEY") != "" || os.Getenv("ANTHROPIC_BASE_URL") != "" || os.Getenv("CLAUDE_SECURESTORAGE_CONFIG_DIR") != "" || os.Getenv("CLAUDE_CODE_ENTRYPOINT") != "sdk-go" {
		os.Exit(3)
	}
	args := strings.Join(os.Args[1:], "\x00")
	for _, v := range []string{"--input-format\x00stream-json", "--output-format\x00stream-json", "--setting-sources=", "--tools\x00\x00", "--no-session-persistence"} {
		if !strings.Contains(args, v) {
			os.Exit(4)
		}
	}
	trace, e := os.OpenFile(filepath.Join(p, "claude-calls"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if e != nil {
		os.Exit(5)
	}
	defer trace.Close()
	mode := os.Getenv("GO_CLAUDE_FIXTURE_MODE")
	reader := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	for reader.Scan() {
		var req struct {
			Type      string `json:"type"`
			RequestID string `json:"request_id"`
			Request   struct {
				Subtype       string `json:"subtype"`
				SkipBehaviors bool   `json:"skip_behaviors"`
			} `json:"request"`
		}
		if json.Unmarshal(reader.Bytes(), &req) != nil || req.Type != "control_request" || (req.Request.Subtype != "initialize" && req.Request.Subtype != "get_usage") {
			os.Exit(6)
		}
		if req.Request.Subtype == "get_usage" && !req.Request.SkipBehaviors {
			os.Exit(7)
		}
		fmt.Fprintln(trace, req.Request.Subtype)
		if mode == "hang" {
			time.Sleep(time.Hour)
		}
		if mode == "activity" {
			_ = encoder.Encode(map[string]any{"type": "assistant", "message": map[string]any{"role": "assistant"}})
			continue
		}
		if mode == "rejected" {
			_ = encoder.Encode(map[string]any{"type": "control_response", "response": map[string]any{"request_id": req.RequestID, "subtype": "error", "error": "fixture-sensitive"}})
			continue
		}
		var payload any = map[string]any{}
		if req.Request.Subtype == "get_usage" {
			payload = map[string]any{"rate_limits_available": true,
				"rate_limits": map[string]any{"five_hour": map[string]any{"utilization": 12.5, "resets_at": nil}, "seven_day": map[string]any{"utilization": 45.0, "resets_at": nil}},
				"secret":      "must-not-be-returned"}
		}
		_ = encoder.Encode(map[string]any{"type": "control_response", "response": map[string]any{"request_id": req.RequestID, "subtype": "success", "response": payload}})
	}
}

func TestClaudeQuotaUsesInstalledBinaryNoNodeAndNoUserTurn(t *testing.T) {
	m, a := fixtureProfile(t, "claude")
	m.claudeCommand = os.Args[0]
	t.Setenv("GO_PROVIDER_FIXTURE", "claude")
	t.Setenv("ANTHROPIC_API_KEY", "unsafe-inherited-token")
	t.Setenv("ANTHROPIC_BASE_URL", "http://127.0.0.1:43119")
	t.Setenv("CLAUDE_SECURESTORAGE_CONFIG_DIR", "/must/not/use")
	s, err := m.Refresh(context.Background(), a)
	if err != nil {
		t.Fatal(err)
	}
	if !s.Available || len(s.Windows) != 2 || *s.Windows[0].Used != 12.5 {
		t.Fatalf("direct Claude control returned wrong usage: %+v", s)
	}
	data, err := os.ReadFile(filepath.Join(a.Profile, "claude-calls"))
	if err != nil || strings.TrimSpace(string(data)) != "initialize\nget_usage" {
		t.Fatalf("unexpected Claude control sequence: %q %v", data, err)
	}
}

func TestClaudeControlRejectsUnexpectedModelActivityAndErrors(t *testing.T) {
	for _, mode := range []string{"activity", "rejected"} {
		t.Run(mode, func(t *testing.T) {
			m, a := fixtureProfile(t, "claude")
			m.claudeCommand = os.Args[0]
			t.Setenv("GO_PROVIDER_FIXTURE", "claude")
			t.Setenv("GO_CLAUDE_FIXTURE_MODE", mode)
			_, err := m.Refresh(context.Background(), a)
			if err == nil || strings.Contains(err.Error(), "fixture-sensitive") {
				t.Fatalf("unexpected response must fail without leaking data: %v", err)
			}
		})
	}
}

func TestClaudeControlDeadlineKillsUnresponsiveNativeProcess(t *testing.T) {
	m, a := fixtureProfile(t, "claude")
	m.claudeCommand = os.Args[0]
	t.Setenv("GO_PROVIDER_FIXTURE", "claude")
	t.Setenv("GO_CLAUDE_FIXTURE_MODE", "hang")
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := m.Refresh(ctx, a)
	if err == nil || time.Since(started) > 3*time.Second {
		t.Fatalf("Claude control deadline was ignored: %v", err)
	}
}

// Proves real native CLI compatibility without contacting any user's account.
// The isolated, unsigned-in profile returns unavailable limits; no prompts,
// SDK installation, user auth files, or actual model usage are involved.
func TestInstalledClaudeControlWithoutNode(t *testing.T) {
	if os.Getenv("NORIA_NATIVE_COMPAT") != "1" {
		t.Skip("opt-in installed-client compatibility test")
	}
	claude, err := exec.LookPath("claude")
	if err != nil {
		t.Skip("native Claude unavailable")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("XDG_CACHE_HOME", home)
	m, a := fixtureProfile(t, "claude")
	m.claudeCommand = claude
	s, err := m.Refresh(context.Background(), a)
	if err != nil {
		t.Fatal(err)
	}
	if s.Available || len(s.Windows) != 0 {
		t.Fatalf("unsigned-in profile unexpectedly yielded active subscription quotas: %+v", s)
	}
	if _, err := os.Stat(filepath.Join(m.home, "bridge")); !os.IsNotExist(err) {
		t.Fatal("native reader unexpectedly needs the old SDK bridge")
	}
}
