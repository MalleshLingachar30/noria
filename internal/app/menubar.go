package app

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/amxv/noria/internal/config"
	"github.com/amxv/noria/internal/distribution"
	"github.com/amxv/noria/internal/service"
)

func menubar(home string, args []string, out, errOut io.Writer) error {
	if len(args) == 0 || isHelp(args[0]) {
		fmt.Fprintln(out, "Usage: noria menubar install [--source APP_PATH]\n       noria menubar open\n       noria menubar reopen\nmacOS native wrapper. Preferences and accounts are owned by the CLI.")
		return nil
	}
	if runtime.GOOS != "darwin" {
		return errors.New("the native menu bar app is available on macOS; CLI start/stop works on every platform")
	}
	if e := config.Init(home); e != nil {
		return e
	}
	exe, e := service.Executable()
	if e != nil {
		return e
	}
	target := filepath.Join(filepath.Dir(exe), "Noria.app")
	switch args[0] {
	case "install":
		f := flags("menubar install", errOut)
		source := f.String("source", "", "local signed bundle for development")
		if e = f.Parse(args[1:]); e != nil {
			if errors.Is(e, flag.ErrHelp) {
				return nil
			}
			return e
		}
		tmp := ""
		if *source == "" {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			r, e := distribution.Latest(ctx)
			if e != nil {
				return e
			}
			tmp, e = os.MkdirTemp(home, ".menubar-*")
			if e != nil {
				return e
			}
			defer os.RemoveAll(tmp)
			archive := filepath.Join(tmp, "app.tar.gz")
			if e = distribution.Download(ctx, r, distribution.ArchiveName(), archive); e != nil {
				return e
			}
			if e = distribution.Extract(archive, filepath.Join(tmp, "app")); e != nil {
				return e
			}
			*source = filepath.Join(tmp, "app", "Noria.app")
		}
		if e = verifyApp(*source); e != nil {
			return e
		}
		if e = replaceApp(*source, target); e != nil {
			return e
		}
		fmt.Fprintln(out, "Installed", target)
		return nil
	case "open":
		if _, e = os.Stat(target); e != nil {
			return errors.New("menu bar app is not installed; run noria menubar install")
		}
		userHome, e := os.UserHomeDir()
		if e != nil {
			return e
		}
		locator, _ := json.Marshal(map[string]string{"home": home})
		if e = config.Write(filepath.Join(userHome, "Library", "Application Support", "Noria", "menu-home.json"), locator); e != nil {
			return e
		}
		cmd := exec.Command("/usr/bin/open", target, "--args", "--home", home)
		return cmd.Run()
	case "reopen":
		cmd := exec.Command(exe, "__menubar-reopen", "--home", home)
		cmd.Stdin = nil
		cmd.Stdout = nil
		cmd.Stderr = nil
		if e = cmd.Start(); e != nil {
			return e
		}
		return cmd.Process.Release()
	default:
		return errors.New("unknown menubar command")
	}
}
