package provider

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

var lookPath = exec.LookPath

// Native controls must never inherit gateway credentials, provider overrides,
// injected SDK settings, or a second user's profile location.
func cleanEnv(profile, provider string) []string {
	var env []string
	for _, v := range os.Environ() {
		key, _, _ := strings.Cut(v, "=")
		u := strings.ToUpper(key)
		if strings.HasPrefix(u, "ANTHROPIC_") || strings.HasPrefix(u, "OPENAI_") || strings.HasPrefix(u, "CODEX_") || strings.HasPrefix(u, "CLAUDE_") || strings.HasPrefix(u, "NORIA_") || strings.HasPrefix(u, "NODE_") || u == "CLAUDECODE" || u == "HTTP_PROXY" || u == "HTTPS_PROXY" || u == "ALL_PROXY" {
			continue
		}
		env = append(env, v)
	}
	if provider == "codex" {
		env = append(env, "CODEX_HOME="+profile)
	} else {
		env = append(env, "CLAUDE_CONFIG_DIR="+profile)
	}
	return env
}

func nativeCommand(ctx context.Context, env []string, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = env
	cmd.Stderr = io.Discard
	cmd.WaitDelay = time.Second
	configureProcess(cmd)
	return cmd
}

type boundedBuffer struct {
	data []byte
	max  int
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if len(b.data)+len(p) > b.max {
		return 0, io.ErrShortBuffer
	}
	b.data = append(b.data, p...)
	return len(p), nil
}
