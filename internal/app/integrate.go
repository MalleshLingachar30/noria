package app

import (
	"errors"
	"flag"
	"fmt"
	"github.com/amxv/noria/internal/config"
	"github.com/amxv/noria/internal/integration"
	"io"
	"os"
	"path/filepath"
)

func authCommand(home string, args []string, out, errOut io.Writer) error {
	if len(args) != 1 || args[0] != "token" {
		return errors.New("usage: noria auth token [--home <NORIA_HOME>]")
	}
	secret, e := config.Secret(home)
	if e != nil {
		return e
	}
	_, e = fmt.Fprintln(out, secret)
	return e
}
func integrateCommand(home string, args []string, out, errOut io.Writer) error {
	if len(args) == 0 || isHelp(args[0]) {
		fmt.Fprintln(out, "Usage: noria integrate codex|claude [--client-home path] [--yes] [--remove] [--json]\nPreview changes by default. --yes authorizes a reversible settings edit. Restart existing clients after switching routing.")
		return nil
	}
	f := flags("integrate", errOut)
	root := f.String("client-home", "", "client configuration directory (supports CODEX_HOME / CLAUDE_CONFIG_DIR)")
	yes := f.Bool("yes", false, "approve settings change")
	remove := f.Bool("remove", false, "restore direct settings and unregister integration")
	asJSON := f.Bool("json", false, "JSON preview/result")
	if e := f.Parse(args[1:]); e != nil {
		if errors.Is(e, flag.ErrHelp) {
			return nil
		}
		return e
	}
	if f.NArg() != 0 {
		return errors.New("unexpected integration arguments")
	}
	if e := config.Init(home); e != nil {
		return e
	}
	c, e := config.Load(home)
	if e != nil {
		return e
	}
	exe, e := os.Executable()
	if e != nil {
		return e
	}
	exe, e = filepath.Abs(exe)
	if e != nil {
		return e
	}
	plan, e := integration.Preview(home, args[0], *root, exe, c.Listen)
	if e != nil {
		return e
	}
	if !*yes {
		if *asJSON {
			return printJSON(out, plan)
		}
		fmt.Fprintln(out, "Settings file:", plan["path"])
		fmt.Fprintln(out, plan["note"])
		_ = printJSON(out, plan["changes"])
		fmt.Fprintln(out, "Run again with --yes to apply.")
		return nil
	}
	if *remove {
		e = integration.Remove(home, args[0], *root)
	} else {
		if !daemonRunning(home) {
			if e = serviceCommand(home, "start", nil, io.Discard, errOut); e != nil {
				return e
			}
		}
		e = integration.Install(home, args[0], *root, exe, c.Listen)
	}
	if e != nil {
		return e
	}
	if *asJSON {
		return printJSON(out, map[string]any{"path": plan["path"], "configured": !*remove, "active": serviceWanted(home) && !*remove, "client_restart_required": true})
	}
	fmt.Fprintln(out, "Updated", plan["path"])
	if !serviceWanted(home) && !*remove {
		fmt.Fprintln(out, "Integration saved. Run noria start to enable routing.")
	} else {
		fmt.Fprintln(out, "Reopen existing clients to pick up routing.")
	}
	return nil
}
func integrationsCommand(home string, args []string, out io.Writer) error {
	if len(args) > 1 || len(args) == 1 && args[0] != "--json" {
		return errors.New("usage: noria integrations [--json]")
	}
	views, e := integration.Views(home)
	if e != nil {
		return e
	}
	if len(args) == 1 {
		return printJSON(out, views)
	}
	for _, v := range views {
		state := "direct"
		if v.Active {
			state = "Noria"
		}
		if !v.Healthy {
			state = "changed outside Noria"
		}
		fmt.Fprintf(out, "%s\t%s\t%s\n", v.Provider, v.Path, state)
	}
	return nil
}
func nativeRoots() []map[string]string {
	out := []map[string]string{}
	for _, p := range []string{"codex", "claude"} {
		root, e := integration.DefaultRoot(p)
		if e == nil {
			out = append(out, map[string]string{"provider": p, "root": root})
		}
		home, _ := os.UserHomeDir()
		custom := filepath.Join(home, ".gg", p)
		if custom != root {
			if st, e := os.Stat(custom); e == nil && st.IsDir() {
				out = append(out, map[string]string{"provider": p, "root": custom})
			}
		}
	}
	return out
}
