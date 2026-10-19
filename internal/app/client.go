package app

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/amxv/noria/internal/config"
)

// Native control processes must never inherit gateway routing or another
// profile's credentials. Other ordinary OS environment values are retained.
func isolatedEnv(profile, kind string) []string {
	env := []string{}
	for _, v := range os.Environ() {
		k, _, _ := strings.Cut(v, "=")
		if strings.HasPrefix(k, "NORIA_") || strings.HasPrefix(k, "ANTHROPIC_") || strings.HasPrefix(k, "OPENAI_") || strings.HasPrefix(k, "CODEX_") || strings.HasPrefix(k, "CLAUDE_") || k == "ENABLE_TOOL_SEARCH" {
			continue
		}
		env = append(env, v)
	}
	if kind == "codex" {
		env = append(env, "CODEX_HOME="+profile)
	} else {
		env = append(env, "CLAUDE_CONFIG_DIR="+profile)
	}
	return env
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }

func client(home, op string, args []string, out, errOut io.Writer) error {
	if len(args) == 0 || isHelp(args[0]) {
		fmt.Fprintf(out, "Usage: noria %s codex|claude", op)
		if op == "run" {
			fmt.Fprint(out, " -- [native client arguments]")
		}
		fmt.Fprintln(out, "\nRetain your client home, history, skills and settings. Routing changes apply to the child only.\nThe env command intentionally prints the local gateway credential.\nFor Codex, env also prints a shell function carrying command-local config overrides.\nExample: eval \"$(noria env claude)\"")
		return nil
	}
	kind := args[0]
	if kind != "codex" && kind != "claude" {
		return fmt.Errorf("client must be codex or claude")
	}
	if op == "env" && len(args) != 1 {
		return fmt.Errorf("usage: noria env codex|claude")
	}
	c, e := config.Load(home)
	if e != nil {
		return e
	}
	secret, e := config.Secret(home)
	if e != nil {
		return e
	}
	address := daemonAddress(home, c.Listen)
	env := clientEnv()
	var overrides []string
	exported := map[string]string{}
	if kind == "claude" {
		exported["ANTHROPIC_BASE_URL"] = "http://" + address + "/claude"
		exported["ANTHROPIC_AUTH_TOKEN"] = secret
		exported["ENABLE_TOOL_SEARCH"] = "true"
	} else {
		// Command-local native provider configuration keeps the existing client
		// home and credentials untouched, while choosing only the gateway token.
		overrides = codexOverrides(address)
		exported["NORIA_GATEWAY_TOKEN"] = secret
	}
	keys := []string{"CODEX_HOME", "NORIA_GATEWAY_TOKEN", "CLAUDE_CONFIG_DIR", "ANTHROPIC_BASE_URL", "ANTHROPIC_AUTH_TOKEN", "ENABLE_TOOL_SEARCH"}
	for _, k := range keys {
		if v, ok := exported[k]; ok {
			env = append(env, k+"="+v)
		}
	}
	if op == "env" {
		fmt.Fprintln(out, "unset OPENAI_API_KEY OPENAI_BASE_URL ANTHROPIC_API_KEY CLAUDE_CODE_OAUTH_TOKEN ANTHROPIC_CUSTOM_HEADERS")
		for _, k := range keys {
			if v, ok := exported[k]; ok {
				fmt.Fprintf(out, "export %s=%s\n", k, shellQuote(v))
			}
		}
		if kind == "codex" {
			fmt.Fprint(out, "codex() { command codex")
			for _, v := range overrides {
				fmt.Fprint(out, " "+shellQuote(v))
			}
			fmt.Fprintln(out, ` "$@"; }`)
		}
		return nil
	}
	native := args[1:]
	if len(native) > 0 && native[0] == "--" {
		native = native[1:]
	}
	cmd := exec.Command(kind, append(overrides, native...)...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = out
	cmd.Stderr = errOut
	if e = cmd.Run(); e != nil {
		return fmt.Errorf("%s exited: %w", kind, e)
	}
	return nil
}

func clientEnv() []string {
	env := []string{}
	for _, v := range os.Environ() {
		k, _, _ := strings.Cut(v, "=")
		switch k {
		case "OPENAI_API_KEY", "OPENAI_BASE_URL", "ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_BASE_URL", "ANTHROPIC_CUSTOM_HEADERS", "CLAUDE_CODE_OAUTH_TOKEN", "CLAUDE_CODE_USE_BEDROCK", "CLAUDE_CODE_USE_VERTEX", "CLAUDE_CODE_USE_FOUNDRY", "NORIA_GATEWAY_TOKEN", "ENABLE_TOOL_SEARCH":
			continue
		}
		env = append(env, v)
	}
	return env
}

func codexOverrides(address string) []string {
	return []string{"-c", `model_provider="noria"`, "-c", `model_providers.noria={name="Noria",base_url=` + strconv.Quote("http://"+address+"/codex/v1") + `,env_key="NORIA_GATEWAY_TOKEN",wire_api="responses",supports_websockets=true}`}
}
