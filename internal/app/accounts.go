package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/amxv/noria/internal/config"
	"github.com/amxv/noria/internal/model"
	"github.com/amxv/noria/internal/provider"
)

func account(home string, args []string, out, errOut io.Writer) error {
	if len(args) == 0 || isHelp(args[0]) {
		fmt.Fprintln(out, "Usage: noria account add codex|claude [--alias NAME] [--json]\n       noria account list [--json]\n       noria account relogin|enable|disable|remove ACCOUNT [--json]\n       noria account rename ACCOUNT [ALIAS] [--json]\nAccounts display their email by default. Omit ALIAS to remove an alias. ACCOUNT can be an ID, unique email, or unique alias.")
		return nil
	}
	for _, arg := range args[1:] {
		if isHelp(arg) {
			return account(home, nil, out, errOut)
		}
	}
	asJSON := false
	clean := []string{}
	for _, arg := range args {
		if arg == "--json" {
			asJSON = true
		} else {
			clean = append(clean, arg)
		}
	}
	args = clean
	if e := config.Init(home); e != nil {
		return e
	}
	if args[0] == "list" {
		if len(args) != 1 {
			return errors.New("usage: noria account list [--json]")
		}
		c, e := config.Load(home)
		if e != nil {
			return e
		}
		if asJSON {
			return printJSON(out, safeAccounts(c.Accounts))
		}
		for _, a := range c.Accounts {
			fmt.Fprintf(out, "%s\t%s\tenabled=%t\temail=%s\tid=%s\n", a.DisplayName(), a.Provider, a.Enabled, a.Email, a.ID)
		}
		return nil
	}
	if len(args) < 2 {
		return errors.New("account command requires a provider or account")
	}
	if args[0] == "add" && (args[1] == "codex" || args[1] == "claude") {
		f := flags("account add", errOut)
		alias := f.String("alias", "", "optional display name; default is the account email")
		if e := f.Parse(args[2:]); e != nil {
			if errors.Is(e, flag.ErrHelp) {
				return nil
			}
			return e
		}
		if f.NArg() != 0 {
			return errors.New("unexpected account arguments")
		}
		if e := validateAlias(*alias); e != nil {
			return e
		}
		var random [8]byte
		if _, e := rand.Read(random[:]); e != nil {
			return e
		}
		id := "acc_" + hex.EncodeToString(random[:])
		a := model.Account{ID: id, Provider: args[1], Profile: filepath.Join(home, "accounts", id), Alias: *alias, Enabled: false}
		if e := config.PrivateDir(a.Profile); e != nil {
			return e
		}
		if e := config.Mutate(home, func(c *config.Config) error { c.Accounts = append(c.Accounts, a); return nil }); e != nil {
			return e
		}
		if !asJSON {
			fmt.Fprintln(out, "Opening native", a.Provider, "sign-in in your browser…")
		}
		if e := completeLogin(home, a); e != nil {
			return e
		}
		c, _ := config.Load(home)
		a, _ = findAccount(c, id)
		if asJSON {
			return printJSON(out, map[string]any{"ok": true, "account": safeAccount(a)})
		}
		fmt.Fprintln(out, "Added", a.DisplayName())
		return nil
	}
	if args[0] == "add" {
		return errors.New("usage: noria account add codex|claude [--alias NAME] [--json]")
	}
	c, e := config.Load(home)
	if e != nil {
		return e
	}
	a, e := resolveAccount(c, args[1])
	if e != nil {
		return e
	}
	if args[0] == "relogin" {
		if len(args) != 2 {
			return errors.New("usage: noria account relogin ACCOUNT [--json]")
		}
		if !asJSON {
			fmt.Fprintln(out, "Opening sign-in for", a.DisplayName())
		}
		if e = completeLogin(home, a); e != nil {
			return e
		}
		c, _ = config.Load(home)
		a, _ = findAccount(c, a.ID)
		if asJSON {
			return printJSON(out, map[string]any{"ok": true, "account": safeAccount(a)})
		}
		fmt.Fprintln(out, "Signed in:", a.DisplayName())
		return nil
	}
	if args[0] == "rename" {
		alias := ""
		if len(args) > 2 {
			alias = args[2]
		}
		if len(args) > 3 {
			return errors.New("usage: noria account rename ACCOUNT [ALIAS]")
		}
		if e = validateAlias(alias); e != nil {
			return e
		}
		if e = config.Mutate(home, func(c *config.Config) error {
			for i := range c.Accounts {
				if c.Accounts[i].ID == a.ID {
					c.Accounts[i].Alias = alias
					return nil
				}
			}
			return errors.New("account was removed")
		}); e != nil {
			return e
		}
		reloadAccounts(home)
		if asJSON {
			return printJSON(out, map[string]any{"ok": true})
		}
		fmt.Fprintln(out, "Account display name updated")
		return nil
	}
	if args[0] == "enable" || args[0] == "disable" || args[0] == "remove" {
		if len(args) != 2 {
			return fmt.Errorf("usage: noria account %s ACCOUNT [--json]", args[0])
		}
		op := args[0]
		if e = config.Mutate(home, func(c *config.Config) error {
			for i := range c.Accounts {
				if c.Accounts[i].ID != a.ID {
					continue
				}
				switch op {
				case "remove":
					c.Accounts = append(c.Accounts[:i], c.Accounts[i+1:]...)
				case "enable":
					if !c.Accounts[i].Authenticated {
						return errors.New("sign in to the account before enabling it")
					}
					c.Accounts[i].Enabled = true
				case "disable":
					c.Accounts[i].Enabled = false
				}
				return nil
			}
			return errors.New("account removed before modification")
		}); e != nil {
			return e
		}
		reloadAccounts(home)
		if asJSON {
			return printJSON(out, map[string]any{"ok": true})
		}
		fmt.Fprintln(out, "Account", a.DisplayName(), op)
		return nil
	}
	return errors.New("unknown account command")
}
func findAccount(c config.Config, id string) (model.Account, error) {
	for _, a := range c.Accounts {
		if a.ID == id {
			return a, nil
		}
	}
	return model.Account{}, fmt.Errorf("account %s does not exist", id)
}
func resolveAccount(c config.Config, query string) (model.Account, error) {
	for _, a := range c.Accounts {
		if a.ID == query {
			return a, nil
		}
	}
	found := []model.Account{}
	for _, a := range c.Accounts {
		if a.Email != "" && strings.EqualFold(a.Email, query) || a.Alias != "" && a.Alias == query {
			found = append(found, a)
		}
	}
	if len(found) == 1 {
		return found[0], nil
	}
	if len(found) > 1 {
		return model.Account{}, errors.New("account reference is ambiguous; use its stable ID from account list --json")
	}
	return model.Account{}, fmt.Errorf("account %q not found", query)
}
func validateAlias(alias string) error {
	if len(alias) > 120 || strings.ContainsAny(alias, "\r\n\x00\t") {
		return errors.New("alias must be at most 120 characters on one line")
	}
	return nil
}
func safeAccount(a model.Account) map[string]any {
	return map[string]any{"id": a.ID, "provider": a.Provider, "email": a.Email, "alias": a.Alias, "display_name": a.DisplayName(), "enabled": a.Enabled, "authenticated": a.Authenticated}
}
func safeAccounts(accounts []model.Account) []map[string]any {
	out := []map[string]any{}
	for _, a := range accounts {
		out = append(out, safeAccount(a))
	}
	return out
}
func reloadAccounts(home string) {
	if daemonRunning(home) {
		_, _ = controlRequest(home, "POST", "/_noria/reload", nil)
	}
}
func completeLogin(home string, a model.Account) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	manager := provider.New(home)
	// Native credential replacement is unavailable to routing until identity is
	// confirmed; a failed or duplicate relogin cannot silently swap pool identity.
	if e := config.Mutate(home, func(c *config.Config) error {
		for i := range c.Accounts {
			if c.Accounts[i].ID == a.ID {
				c.Accounts[i].Enabled = false
				c.Accounts[i].Authenticated = false
				return nil
			}
		}
		return errors.New("account removed before login")
	}); e != nil {
		return e
	}
	reloadAccounts(home)
	if e := manager.BrowserLogin(ctx, a); e != nil {
		return e
	}
	identity, e := manager.Identity(ctx, a)
	if e != nil {
		return e
	}
	if !identity.Authenticated {
		return errors.New("provider did not confirm sign-in")
	}
	if identity.Email == "" {
		return errors.New("signed in, but this client version did not return an account email; update the native client and retry")
	}
	if a.Email != "" && !strings.EqualFold(a.Email, identity.Email) {
		return errors.New("a different account was selected; this entry remains disabled. Add that account separately, or sign in again with the original email")
	}

	e = config.Mutate(home, func(c *config.Config) error {
		for _, v := range c.Accounts {
			if v.ID != a.ID && v.Provider == a.Provider && v.Authenticated && strings.EqualFold(v.Email, identity.Email) {
				return fmt.Errorf("%s is already in the pool as %s; relogin that account instead of counting its quota twice", identity.Email, v.ID)
			}
		}
		for i := range c.Accounts {
			if c.Accounts[i].ID == a.ID {
				c.Accounts[i].Email = identity.Email
				c.Accounts[i].Authenticated = true
				c.Accounts[i].Enabled = true
				c.Accounts[i].LastLogin = time.Now()
				return nil
			}
		}
		return errors.New("account removed during sign-in")
	})
	if e != nil {
		return e
	}
	reloadAccounts(home)
	// Failure to read telemetry does not undo a successful native login.
	_ = quota(home, []string{"refresh", a.ID}, io.Discard, io.Discard)
	return nil
}
