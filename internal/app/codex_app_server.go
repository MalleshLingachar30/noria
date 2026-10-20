package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
)

// appServerRequestOnNoria enforces the selected provider on app-server
// thread creation and restoration. Native Codex otherwise restores the
// model_provider recorded in an old rollout, ignoring the current config.toml
// selection. This means even a properly integrated Noria profile can silently
// bypass the router for long-lived sessions.
//
// Only the three methods accepting modelProvider are edited. All other JSON-RPC
// messages, notifications, tool outputs and server responses pass unchanged.
func appServerRequestOnNoria(line []byte) []byte {
	var message map[string]json.RawMessage
	if json.Unmarshal(line, &message) != nil {
		return line
	}
	var method string
	if json.Unmarshal(message["method"], &method) != nil {
		return line
	}
	switch method {
	case "thread/start", "thread/resume", "thread/fork":
	default:
		return line
	}
	var params map[string]json.RawMessage
	if json.Unmarshal(message["params"], &params) != nil || params == nil {
		return line // Let Codex report invalid JSON-RPC params.
	}
	params["modelProvider"] = json.RawMessage(`"noria_runner"`)
	encoded, err := json.Marshal(params)
	if err != nil {
		return line
	}
	message["params"] = encoded
	encoded, err = json.Marshal(message)
	if err != nil {
		return line
	}
	return append(encoded, '\n')
}

// appServerInput forwards newline-delimited JSON without consuming prompts or
// storing any request contents. A native app-server may have arbitrarily large
// tool messages, so avoid Scanner's default 64 KiB token limit.
func appServerInput(dst io.Writer, src io.Reader) error {
	reader := bufio.NewReader(src)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			data := appServerRequestOnNoria(line)
			if _, writeErr := io.Copy(dst, bytes.NewReader(data)); writeErr != nil {
				return writeErr
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

// runCodexAppServer is an opt-in app-server adapter for clients which launch
// through `noria run codex -- app-server`. Command-local config overrides are
// not sufficient for OLD app-server threads, because their saved modelProvider
// takes precedence on thread/resume. This adapter changes that one RPC field.
func runCodexAppServer(native []string, overrides, env []string, stdin io.Reader, out, errOut io.Writer) error {
	binary := "codex"
	// Allows a narrow `codex` launcher shim to invoke the real binary directly
	// instead of recursively resolving itself on PATH.
	if explicit := os.Getenv("NORIA_NATIVE_CODEX_BIN"); explicit != "" {
		if !filepath.IsAbs(explicit) {
			return errors.New("NORIA_NATIVE_CODEX_BIN must be an absolute executable path")
		}
		info, err := os.Stat(explicit)
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("NORIA_NATIVE_CODEX_BIN is not a regular executable")
		}
		binary = explicit
	}
	// An abruptly terminated third-party GUI must not leave a detached Codex
	// child holding the resumed thread's exclusive writer lock.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cmd := exec.CommandContext(ctx, binary, append(append([]string{}, overrides...), native...)...)
	cmd.Env = env
	cmd.Stdout = out
	cmd.Stderr = errOut
	pipe, err := cmd.StdinPipe()
	if err != nil {
		return errors.New("cannot open Codex app-server input")
	}
	if err := cmd.Start(); err != nil {
		return errors.New("cannot start installed Codex app-server")
	}
	done := make(chan error, 1)
	go func() {
		done <- appServerInput(pipe, stdin)
		_ = pipe.Close()
	}()
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("Codex app-server exited: %w", err)
	}
	// Do not wait on a blocked caller input after the native child exits.
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, io.ErrClosedPipe) {
			return errors.New("Codex app-server input forwarding failed")
		}
	default:
	}
	return nil
}
