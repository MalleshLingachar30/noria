package app

import (
	"encoding/json"
	"fmt"
	"github.com/amxv/noria/internal/buildinfo"
	"github.com/amxv/noria/internal/config"
	"github.com/amxv/noria/internal/core"
	"github.com/amxv/noria/internal/integration"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// dashboard joins CLI-owned metadata for all frontends. No Swift quota logic.
func dashboard(home string) (map[string]any, error) {
	c, e := config.Load(home)
	if e != nil {
		return nil, e
	}
	b, err := controlRequest(home, http.MethodGet, "/_noria/status", nil)
	running := err == nil
	if !running {
		s, e := core.New(statePath(home), c.Accounts)
		if e != nil {
			return nil, e
		}
		b, e = json.Marshal(s.Status())
		if e != nil {
			return nil, e
		}
	}
	var v map[string]any
	if e = json.Unmarshal(b, &v); e != nil {
		return nil, e
	}
	views, e := integration.Views(home)
	if e != nil {
		return nil, e
	}
	v["schema_version"] = 1
	v["version"] = buildinfo.CurrentVersion()
	v["running"] = running
	v["address"] = c.Listen
	v["integrations"] = views
	v["client_roots"] = nativeRoots()
	v["preferences"] = map[string]bool{"start_at_login": c.StartAtLogin, "menubar_at_login": c.MenuBarAtLogin, "auto_update": c.AutoUpdate}
	if runtime.GOOS != "darwin" {
		v["preferences"].(map[string]bool)["tray_at_login"] = c.MenuBarAtLogin
	}
	v["observed_at"] = time.Now()
	return v, nil
}
func healthCommand(home string, args []string, out io.Writer) error {
	if len(args) > 0 && isHelp(args[0]) {
		fmt.Fprintln(out, "Usage: noria health [--json]\nChecks local service, configured routing and native executables. Sends no model prompt.")
		return nil
	}
	c, e := config.Load(home)
	if e != nil {
		return e
	}
	views, e := integration.Views(home)
	if e != nil {
		return e
	}
	checks := []map[string]any{}
	_, err := controlRequest(home, http.MethodGet, "/_noria/health", nil)
	checks = append(checks, map[string]any{"name": "Local router", "ok": err == nil, "detail": c.Listen})
	for _, view := range views {
		checks = append(checks, map[string]any{"name": view.Provider + " settings", "ok": view.Healthy, "detail": view.Path})
	}
	seen := map[string]bool{}
	for _, a := range c.Accounts {
		if seen[a.Provider] || !a.Enabled {
			continue
		}
		seen[a.Provider] = true
		_, e := exec.LookPath(a.Provider)
		checks = append(checks, map[string]any{"name": a.Provider + " executable", "ok": e == nil})
	}
	for _, key := range []string{"ANTHROPIC_AUTH_TOKEN", "CLAUDE_CODE_OAUTH_TOKEN", "CLAUDE_CODE_USE_BEDROCK", "CLAUDE_CODE_USE_VERTEX", "CLAUDE_CODE_USE_FOUNDRY"} {
		if os.Getenv(key) != "" {
			checks = append(checks, map[string]any{"name": "Environment override: " + key, "ok": false, "detail": "May bypass settings.json or override the gateway helper. Clear it for persistent routing."})
		}
	}
	ok := true
	for _, v := range checks {
		ok = ok && v["ok"] == true
	}
	report := map[string]any{"ok": ok, "checks": checks, "note": "Configuration and local reachability only; no live model request was sent. Reopen clients after routing changes."}
	if len(args) == 1 && args[0] == "--json" {
		return printJSON(out, report)
	}
	for _, v := range checks {
		fmt.Fprintf(out, "%s: %t\n", v["name"], v["ok"])
	}
	return nil
}
