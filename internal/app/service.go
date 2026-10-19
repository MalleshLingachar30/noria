package app

import (
	"errors"
	"flag"
	"fmt"
	"github.com/amxv/noria/internal/config"
	"github.com/amxv/noria/internal/integration"
	"github.com/amxv/noria/internal/service"
	"io"
	"net/http"
	"time"
)

func serviceWanted(home string) bool { return service.ReadIntent(home).Wanted }
func serviceCommand(home, op string, args []string, out, errOut io.Writer) error {
	f := flags(op, errOut)
	asJSON := f.Bool("json", false, "machine-readable result")
	if e := f.Parse(args); e != nil {
		if errors.Is(e, flag.ErrHelp) {
			return nil
		}
		return e
	}
	if f.NArg() != 0 {
		return fmt.Errorf("usage: noria %s [--json]", op)
	}
	if e := config.Init(home); e != nil {
		return e
	}
	unlock, e := config.Lock(home, "lifecycle")
	if e != nil {
		return e
	}
	defer unlock()
	if op == "stop" || op == "restart" {
		if e = stopService(home); e != nil {
			return e
		}
	}
	if op == "start" || op == "restart" {
		if e = startService(home); e != nil {
			return e
		}
	}
	message := "Noria is running. Configured clients use the account pool."
	if op == "stop" {
		message = "Noria stopped. Previous direct provider routing and login restored. Reopen existing clients to reload settings."
	}
	if *asJSON {
		return printJSON(out, map[string]any{"running": op != "stop", "message": message, "client_restart_required": true})
	}
	fmt.Fprintln(out, message)
	return nil
}
func daemonRunning(home string) bool {
	_, e := controlRequest(home, http.MethodGet, "/_noria/health", nil)
	return e == nil
}
func startService(home string) error {
	c, e := config.Load(home)
	if e != nil {
		return e
	}
	exe, e := service.Executable()
	if e != nil {
		return e
	}
	if !daemonRunning(home) {
		if e = service.Start(home, exe); e != nil {
			return e
		}
		deadline := time.Now().Add(12 * time.Second)
		for time.Now().Before(deadline) && !daemonRunning(home) {
			time.Sleep(100 * time.Millisecond)
		}
		if !daemonRunning(home) {
			_ = service.Stop(home, exe)
			return errors.New("Noria did not become healthy; inspect noria health and the private logs/service.log")
		}
	}
	if e = integration.SetActive(home, exe, c.Listen, true); e != nil {
		return e
	}
	return nil
}
func stopService(home string) error {
	c, e := config.Load(home)
	if e != nil {
		return e
	}
	exe, e := service.Executable()
	if e != nil {
		return e
	}
	if e = integration.SetActive(home, exe, c.Listen, false); e != nil {
		return e
	}
	if e = service.SetIntent(home, exe, false); e != nil {
		return e
	}
	_, _ = controlRequest(home, http.MethodPost, "/_noria/shutdown", nil)
	if e = service.Stop(home, exe); e != nil {
		return e
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if !daemonRunning(home) {
			// Detached supervisors must release ownership before a subsequent start;
			// otherwise a rapid stop/start could reuse an exiting supervisor.
			release, err := config.Lock(home, "supervisor")
			if err == nil {
				release()
			}
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
	return errors.New("stop requested but requests are still draining; check noria status")
}
