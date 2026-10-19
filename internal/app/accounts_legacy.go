package app

import (
	"errors"
	"flag"
	"fmt"
	"github.com/amxv/noria/internal/config"
	"github.com/amxv/noria/internal/model"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func accountLegacy(home string, args []string, out, errOut io.Writer) error {
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
		return completeLogin(home, a)
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
