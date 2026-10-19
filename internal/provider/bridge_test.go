package provider

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClaudeBridgeNoUserTurnAndCleanup(t *testing.T) {
	if _, err := lookPath("node"); err != nil {
		t.Skip("Node required for bridge fixture")
	}
	m, a := fixtureProfile(t, "claude")
	m.claudeCommand = os.Args[0]
	dir := filepath.Join(m.home, "bridge", "node_modules", "@anthropic-ai", "claude-agent-sdk")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"@anthropic-ai/claude-agent-sdk","version":"0.3.287","type":"module","main":"index.mjs"}`), 0600); err != nil {
		t.Fatal(err)
	}
	fake := `
import {appendFileSync} from 'node:fs';
import {join,isAbsolute} from 'node:path';
export function query({prompt,options}) {
 if (!isAbsolute(options.pathToClaudeCodeExecutable) || options.persistSession !== false ||
     options.settingSources.length || options.tools.length || options.resume ||
     options.sessionId || options.systemPrompt || options.mcpServers ||
     process.env.ANTHROPIC_API_KEY || process.env.ANTHROPIC_BASE_URL || process.env.CLAUDE_SECURESTORAGE_CONFIG_DIR)
   throw new Error('unsafe bridge options/environment');
 const mark = value => appendFileSync(join(process.env.CLAUDE_CONFIG_DIR,'bridge-calls'),value+'\n');
 let yielded = false;
 prompt.next().then(({done}) => { if(!done) yielded = true; });
 return {
  async usage_EXPERIMENTAL_MAY_CHANGE_DO_NOT_RELY_ON_THIS_API_YET() {
   await Promise.resolve(); if(yielded) throw new Error('user turn');
   if((await options.canUseTool('x',{})).behavior !== 'deny') throw new Error('tools enabled');
   mark('usage');
   return {rate_limits_available:true,rate_limits:{five_hour:{utilization:12.5,resets_at:null}},secret:'must-not-output'};
  },
  async interrupt() {mark('interrupt');},
  close() {mark('close');}
 };
}
`
	if err := os.WriteFile(filepath.Join(dir, "index.mjs"), []byte(fake), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ANTHROPIC_API_KEY", "unsafe-inherited-token")
	t.Setenv("ANTHROPIC_BASE_URL", "http://127.0.0.1:43119")
	t.Setenv("CLAUDE_SECURESTORAGE_CONFIG_DIR", "/must/not/use")
	s, err := m.Refresh(context.Background(), a)
	if err != nil {
		t.Fatal(err)
	}
	if !s.Available || len(s.Windows) != 1 || *s.Windows[0].Used != 12.5 {
		t.Fatalf("bad bridge result: %+v", s)
	}
	calls, err := os.ReadFile(filepath.Join(a.Profile, "bridge-calls"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(calls)) != "usage\ninterrupt\nclose" {
		t.Fatalf("bridge cleanup incorrect: %s", calls)
	}
	// A drifted install must fail closed without running npm on request paths.
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"version":"0.0.1","type":"module","main":"index.mjs"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Refresh(context.Background(), a); err == nil {
		t.Fatal("unpinned SDK accepted")
	}
}

func TestCredentialParsingPreservesUnknownExpiryAndNoSecretErrors(t *testing.T) {
	c, expired, err := parseCredential("claude", []byte(`{"claudeAiOauth":{"accessToken":"fixture-only","expiresAt":1}}`))
	if err != nil || !expired || c.Token != "fixture-only" {
		t.Fatal("Claude millisecond expiry not parsed")
	}
	c, expired, err = parseCredential("codex", []byte(`{"tokens":{"access_token":"opaque-fixture-token","account_id":"abc"}}`))
	if err != nil || expired || c.AccountID != "abc" {
		t.Fatal("unknown token expiry synthesized")
	}
	_, _, err = parseCredential("codex", []byte(`fixture-secret-invalid-json`))
	if err == nil || strings.Contains(err.Error(), "fixture-secret") {
		t.Fatal("credential parse error leaks token")
	}
	if _, _, err := parseCredential("codex", []byte(`{"OPENAI_API_KEY":"paid-key-fixture"}`)); err == nil {
		t.Fatal("subscription account silently used a paid API key")
	}
}
