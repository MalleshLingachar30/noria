package app

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/amxv/noria/internal/buildinfo"
	"github.com/amxv/noria/internal/config"
	"github.com/amxv/noria/internal/model"
)

func Run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || isHelp(args[0]) {
		help(stdout)
		return nil
	}
	if args[0] == "--version" {
		fmt.Fprintln(stdout, "noria", buildinfo.CurrentVersion())
		return nil
	}
	home, e := config.Home()
	if e != nil {
		return e
	}
	switch args[0] {
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
	case "serve":
		return serve(home, args[1:], stdout, stderr)
	case "status":
		return status(home, args[1:], stdout, stderr)
	case "quota":
		return quota(home, args[1:], stdout, stderr)
	case "doctor":
		return doctor(home, args[1:], stdout)
	case "env", "run":
		return client(home, args[0], args[1:], stdout, stderr)
	case "bridge":
		return bridge(home, args[1:], stdout)
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
  serve [--listen 127.0.0.1:43119]
  status [--json]               inspect accounts and routing
  quota refresh [account]       read native quota without a prompt
  doctor                       check local prerequisites
  env codex|claude              print child-only routing configuration
  run codex|claude -- [args]    run a client through Noria
  bridge setup                 install the pinned Claude quota SDK

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

func account(home string, args []string, out, errOut io.Writer) error {
	if len(args) == 0 || isHelp(args[0]) {
		fmt.Fprint(out, `Usage: noria account add <id> --provider codex|claude [--model <model> ...]
       noria account login|remove|enable|disable <id>
       noria account list

Profiles are isolated under NORIA_HOME/accounts/<id>. Login runs the native
interactive client with that profile. Remove unregisters an account but retains
its credentials and native history so accidental removal is reversible.
`)
		return nil
	}
	op := args[0]
	for _, a := range args[1:] {
		if isHelp(a) {
			return account(home, nil, out, errOut)
		}
	}
	if op == "list" {
		if len(args) != 1 {
			return fmt.Errorf("usage: noria account list")
		}
		c, e := config.Load(home)
		if e != nil {
			return e
		}
		for _, a := range c.Accounts {
			fmt.Fprintf(out, "%s\t%s\tenabled=%t\tmodels=%v\n", a.ID, a.Provider, a.Enabled, a.Models)
		}
		return nil
	}
	if len(args) < 2 {
		return fmt.Errorf("account %s requires an ID", op)
	}
	id := args[1]
	if e := config.ValidateID(id); e != nil {
		return e
	}
	if op == "login" {
		if len(args) != 2 {
			return fmt.Errorf("usage: noria account login <id>")
		}
		c, e := config.Load(home)
		if e != nil {
			return e
		}
		a, e := findAccount(c, id)
		if e != nil {
			return e
		}
		return login(home, a, out, errOut)
	}
	if op != "add" && op != "remove" && op != "enable" && op != "disable" {
		return fmt.Errorf("unknown account command %q", op)
	}
	f := flags("account "+op, errOut)
	p := f.String("provider", "", "codex or claude")
	var models stringsFlag
	f.Var(&models, "model", "allowed model (repeatable; omit for any model)")
	if e := f.Parse(args[2:]); e != nil {
		if errors.Is(e, flag.ErrHelp) {
			return nil
		}
		return e
	}
	if f.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	unlock, e := config.Lock(home, "config")
	if e != nil {
		return e
	}
	defer unlock()
	c, e := config.Load(home)
	if e != nil {
		return e
	}
	idx := -1
	for i, a := range c.Accounts {
		if a.ID == id {
			idx = i
		}
	}
	if op == "add" {
		if idx >= 0 {
			return fmt.Errorf("account %s already exists", id)
		}
		if *p != "codex" && *p != "claude" {
			return fmt.Errorf("--provider must be codex or claude")
		}
		profile := filepath.Join(home, "accounts", id)
		if e = config.PrivateDir(profile); e != nil {
			return e
		}
		c.Accounts = append(c.Accounts, model.Account{ID: id, Provider: *p, Profile: profile, Enabled: true, Models: models})
	} else {
		if idx < 0 {
			return fmt.Errorf("account %s does not exist", id)
		}
		switch op {
		case "remove":
			c.Accounts = append(c.Accounts[:idx], c.Accounts[idx+1:]...)
		case "enable":
			c.Accounts[idx].Enabled = true
		case "disable":
			c.Accounts[idx].Enabled = false
		}
	}
	if e = config.Save(home, c); e != nil {
		return e
	}
	// Atomic config reads do not need the writer lock; a live daemon applies
	// authorization changes before the command reports success.
	if _, err := controlRequest(home, http.MethodPost, "/_noria/reload", nil); err != nil {
		if _, live := os.Stat(filepath.Join(home, "daemon.json")); live == nil {
			return fmt.Errorf("account configuration saved but live reload failed: %w", err)
		}
	}
	fmt.Fprintln(out, "Account", id, op)
	return nil
}

func findAccount(c config.Config, id string) (model.Account, error) {
	for _, a := range c.Accounts {
		if a.ID == id {
			return a, nil
		}
	}
	return model.Account{}, fmt.Errorf("account %s does not exist", id)
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
