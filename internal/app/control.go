package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/amxv/noria/internal/config"
	"github.com/amxv/noria/internal/core"
	"github.com/amxv/noria/internal/provider"
)

func controlRequest(home, method, path string, payload any) ([]byte, error) {
	c, e := config.Load(home)
	if e != nil {
		return nil, e
	}
	secret, e := config.Secret(home)
	if e != nil {
		return nil, e
	}
	var body []byte
	if payload != nil {
		body, e = json.Marshal(payload)
		if e != nil {
			return nil, e
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	req, e := http.NewRequestWithContext(ctx, method, "http://"+daemonAddress(home, c.Listen)+path, bytes.NewReader(body))
	if e != nil {
		return nil, e
	}
	req.Header.Set("Authorization", "Bearer "+secret)
	req.Header.Set("Content-Type", "application/json")
	// Avoid proxy environment variables for the private loopback control plane.
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.CloseIdleConnections()
	resp, e := client.Do(req)
	if e != nil {
		return nil, fmt.Errorf("daemon unavailable (run noria serve): %w", e)
	}
	defer resp.Body.Close()
	b, e := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if e != nil {
		return nil, e
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("daemon control returned HTTP %d", resp.StatusCode)
	}
	return b, nil
}

func status(home string, args []string, out, errOut io.Writer) error {
	f := flags("status", errOut)
	asJSON := f.Bool("json", false, "print redacted JSON status")
	if e := f.Parse(args); e != nil {
		if errors.Is(e, flag.ErrHelp) {
			return nil
		}
		return e
	}
	if f.NArg() != 0 {
		return fmt.Errorf("usage: noria status [--json]")
	}
	c, e := config.Load(home)
	if e != nil {
		return e
	}
	b, liveErr := controlRequest(home, http.MethodGet, "/_noria/status", nil)
	if liveErr == nil {
		if *asJSON {
			_, e = out.Write(b)
			return e
		}
		fmt.Fprintln(out, "Noria running at http://"+daemonAddress(home, c.Listen))
		return renderStatus(out, b, time.Now())
	}
	// Loading persisted state is safe only when no daemon holds the state lock.
	// Status avoids opening a second scheduler that could rewrite live state.
	// Construction and Status are read-only. Never expose raw persisted
	// bindings, credentials, or endpoint/profile details in diagnostics.
	saved, e := core.New(statePath(home), c.Accounts)
	if e != nil {
		return e
	}
	data, e := json.Marshal(saved.Status())
	if e != nil {
		return e
	}
	if *asJSON {
		var result map[string]any
		if e = json.Unmarshal(data, &result); e != nil {
			return e
		}
		result["running"] = false
		return printJSON(out, result)
	}
	fmt.Fprintln(out, "Noria is not reachable; showing the last saved quota snapshot.")
	return renderStatus(out, data, time.Now())
}

func quota(home string, args []string, out, errOut io.Writer) error {
	if len(args) == 0 || isHelp(args[0]) {
		fmt.Fprintln(out, "Usage: noria quota refresh [account]\nRead native provider usage with isolated auth; sends no user prompt and spends no reset credits.\nStart the daemon to refresh its live scheduler; otherwise update saved state offline.")
		return nil
	}
	if args[0] != "refresh" {
		return fmt.Errorf("usage: noria quota refresh [account]")
	}
	if len(args) > 1 && isHelp(args[1]) {
		return quota(home, nil, out, errOut)
	}
	if len(args) > 2 {
		return fmt.Errorf("usage: noria quota refresh [account]")
	}
	id := ""
	if len(args) == 2 {
		id = args[1]
	}
	if b, e := controlRequest(home, http.MethodPost, "/_noria/refresh", map[string]string{"account": id}); e == nil {
		if _, e = out.Write(b); e != nil {
			return e
		}
		var results []refreshResult
		if e = json.Unmarshal(b, &results); e != nil {
			return fmt.Errorf("invalid quota refresh response: %w", e)
		}
		for _, result := range results {
			if !result.OK {
				return fmt.Errorf("one or more native quota reads failed; inspect per-account results")
			}
		}
		return nil
	}
	unlock, e := config.Lock(home, "daemon")
	if e != nil {
		return fmt.Errorf("live refresh failed and scheduler is locked: %w", e)
	}
	defer unlock()
	c, e := config.Load(home)
	if e != nil {
		return e
	}
	s, e := core.New(statePath(home), c.Accounts)
	if e != nil {
		return e
	}
	ref := &refresher{home: home, s: s, p: provider.New(home)}
	result, e := ref.refresh(context.Background(), id)
	if e != nil {
		return e
	}
	if e = printJSON(out, result); e != nil {
		return e
	}
	for _, r := range result {
		if !r.OK {
			return fmt.Errorf("one or more native quota reads failed; inspect per-account results")
		}
	}
	return nil
}

func doctor(home string, args []string, out io.Writer) error {
	if len(args) > 0 && isHelp(args[0]) {
		fmt.Fprintln(out, "Usage: noria doctor\nCheck configuration, permissions and native executables without reading live quotas or printing credentials.")
		return nil
	}
	if e := requireNoArgs(args, "noria doctor"); e != nil {
		return e
	}
	c, e := config.Load(home)
	if e != nil {
		return e
	}
	problems := 0
	needed := map[string]bool{}
	for _, a := range c.Accounts {
		if !a.Enabled {
			continue
		}
		needed[a.Provider] = true
		if a.Provider == "claude" {
			needed["node"], needed["npm"] = true, true
		}
	}
	for _, name := range []string{"codex", "claude", "node", "npm"} {
		if !needed[name] {
			continue
		}
		if _, e = exec.LookPath(name); e != nil {
			fmt.Fprintln(out, name+": unavailable")
			problems++
		} else {
			fmt.Fprintln(out, name+": available")
		}
	}
	for _, path := range []string{home, filepath.Join(home, "config.json"), filepath.Join(home, "secret")} {
		st, err := os.Stat(path)
		if err != nil || runtime.GOOS != "windows" && st.Mode().Perm()&0077 != 0 {
			fmt.Fprintln(out, "private permissions: repair needed for", path)
			problems++
		}
	}
	if _, e = config.Secret(home); e != nil {
		fmt.Fprintln(out, "gateway secret: invalid or missing")
		problems++
	}
	claudeNeeded := false
	for _, a := range c.Accounts {
		if a.Provider == "claude" && a.Enabled {
			claudeNeeded = true
		}
		st, err := os.Stat(a.Profile)
		if err != nil || !st.IsDir() || runtime.GOOS != "windows" && st.Mode().Perm()&0077 != 0 {
			fmt.Fprintln(out, "profile", a.ID+": missing or unsafe permissions")
			problems++
		} else {
			fmt.Fprintln(out, "profile", a.ID+": isolated")
		}
	}
	if claudeNeeded {
		if _, err := os.Stat(filepath.Join(home, "bridge", "node_modules", "@anthropic-ai", "claude-agent-sdk", "package.json")); err != nil {
			fmt.Fprintln(out, "Claude quota bridge: run noria bridge setup")
			problems++
		} else {
			fmt.Fprintln(out, "Claude quota bridge: installed (runtime schema checked on quota read)")
		}
	}
	if problems > 0 {
		return fmt.Errorf("doctor found %d prerequisite or permission issue(s)", problems)
	}
	fmt.Fprintln(out, "Local prerequisites ready. Account login and real usage have not been tested.")
	return nil
}
