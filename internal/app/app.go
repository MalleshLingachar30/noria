package app

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"

	"path/filepath"

	"github.com/amxv/noria/internal/buildinfo"
	"github.com/amxv/noria/internal/config"
	"github.com/amxv/noria/internal/service"
)

func Run(args []string, stdout, stderr io.Writer) error {
	// Global private-home selection also works for credential helpers launched
	// by GUI clients, which do not inherit a terminal's environment.
	selectedHome := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			break
		}
		if args[i] == "--home" {
			if i+1 >= len(args) {
				return errors.New("--home requires a directory")
			}
			selectedHome = args[i+1]
			args = append(args[:i], args[i+2:]...)
			i--
		}
	}

	if len(args) == 0 || isHelp(args[0]) {
		help(stdout)
		return nil
	}
	if args[0] == "--version" {
		fmt.Fprintln(stdout, "noria", buildinfo.CurrentVersion())
		return nil
	}
	home, e := config.Home()
	if selectedHome != "" {
		home, e = filepath.Abs(selectedHome)
	}
	if e != nil {
		return e
	}
	switch args[0] {
	case "__menubar-reopen":
		return delayedMenuHome(home)
	case "__open-menu":
		if len(args) != 2 {
			return errors.New("missing app path")
		}
		return delayedMenu(args[1])
	case "__supervise", "__supervisor":
		return service.Supervise(home)
	case "__daemon":
		return serve(home, nil, stdout, stderr)
	case "__login":
		c, e := config.Load(home)
		if e != nil {
			return e
		}
		if !c.StartAtLogin {
			return nil
		}
		return serviceCommand(home, "start", []string{"--json"}, stdout, stderr)
	case "auth-token":
		return authCommand(home, []string{"token"}, stdout, stderr)
	case "auth":
		return authCommand(home, args[1:], stdout, stderr)
	case "integrate":
		return integrateCommand(home, args[1:], stdout, stderr)
	case "integrations":
		return integrationsCommand(home, args[1:], stdout)
	case "preferences", "autostart":
		return preferencesCommand(home, args[0], args[1:], stdout)
	case "health":
		return healthCommand(home, args[1:], stdout)
	case "menubar":
		return menubar(home, args[1:], stdout, stderr)
	case "update":
		return updateCommand(home, args[1:], stdout, stderr)
	case "init":
		if len(args) > 1 && isHelp(args[1]) {
			fmt.Fprintln(stdout, "Usage: noria init\nCreate a private home and local gateway secret. Existing accounts are preserved.")
			return nil
		}
		if len(args) != 1 {
			return fmt.Errorf("usage: noria init")
		}
		if e = config.Init(home); e != nil {
			return e
		}
		fmt.Fprintln(stdout, "Initialized", home)
		return nil
	case "account":
		return account(home, args[1:], stdout, stderr)
	case "start", "stop", "restart":
		return serviceCommand(home, args[0], args[1:], stdout, stderr)
	case "status":
		return status(home, args[1:], stdout, stderr)
	case "quota":
		return quota(home, args[1:], stdout, stderr)
	case "doctor":
		return doctor(home, args[1:], stdout)
	case "env", "run":
		return client(home, args[0], args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown command %q (run noria --help)", args[0])
	}
}

func isHelp(s string) bool { return s == "help" || s == "--help" || s == "-h" }
func flags(name string, w io.Writer) *flag.FlagSet {
	f := flag.NewFlagSet("noria "+name, flag.ContinueOnError)
	f.SetOutput(w)
	return f
}
func help(w io.Writer) {
	fmt.Fprint(w, `Noria: local native Codex and Claude account gateway

Usage: noria <command> [arguments]
       noria --version

Commands:
  init                         create private configuration
  account add|login|list|remove|enable|disable
  start | stop | restart       background routing; stop restores direct settings
  integrate codex|claude        preview and install persistent client routing
  integrations [--json]         inspect configured client profiles
  preferences [key on|off]      login and automatic update settings
  menubar open|install          native macOS menu bar
  update [--check]              verified CLI and menu bar updates
  health [--json]               check gateway and client configuration
  status [--json]               inspect accounts and routing
  quota refresh [account]       read native quota without a prompt
  doctor                       check local prerequisites
  env codex|claude              print child-only routing configuration
  run codex|claude -- [args]    run a client through Noria

Use <command> --help for command details. NORIA_HOME defaults to ~/.noria.
Account login is explicit. Noria never imports your existing credentials.
`)
}

type stringsFlag []string

func (s *stringsFlag) String() string { return fmt.Sprint([]string(*s)) }
func (s *stringsFlag) Set(v string) error {
	if v == "" {
		return errors.New("model cannot be empty")
	}
	*s = append(*s, v)
	return nil
}

func printJSON(w io.Writer, v any) error {
	en := json.NewEncoder(w)
	en.SetIndent("", "  ")
	return en.Encode(v)
}
func statePath(home string) string { return filepath.Join(home, "scheduler.json") }
func requireNoArgs(args []string, usage string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: %s", usage)
	}
	return nil
}
