package app

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/amxv/noria/internal/buildinfo"
	"github.com/amxv/noria/internal/config"
	"github.com/amxv/noria/internal/distribution"
	"github.com/amxv/noria/internal/service"
)

type updateStatus struct {
	Current   string    `json:"current"`
	Latest    string    `json:"latest"`
	Available bool      `json:"available"`
	Updated   bool      `json:"updated"`
	Deferred  bool      `json:"deferred"`
	CheckedAt time.Time `json:"checked_at"`
	Message   string    `json:"message"`
}

func updateCommand(home string, args []string, out, errOut io.Writer) error {
	f := flags("update", errOut)
	check := f.Bool("check", false, "check without installing")
	automatic := f.Bool("automatic", false, "respect automatic-update preference and defer while busy")
	_ = f.Bool("json", false, "print structured status")
	if e := f.Parse(args); e != nil {
		if errors.Is(e, flag.ErrHelp) {
			return nil
		}
		return e
	}
	if f.NArg() != 0 {
		return errors.New("usage: noria update [--check] [--json]")
	}
	if e := config.Init(home); e != nil {
		return e
	}
	unlock, e := service.Guard(home, "update")
	if e != nil {
		return e
	}
	defer unlock()
	if *automatic {
		p, e := config.ReadPreferences(home)
		if e != nil {
			return e
		}
		if !p.AutoUpdate {
			return nil
		}
	}
	if *automatic {
		b, _ := os.ReadFile(filepath.Join(home, "update.json"))
		var old updateStatus
		if json.Unmarshal(b, &old) == nil && time.Since(old.CheckedAt) < 6*time.Hour && !old.Available {
			return nil
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	release, e := distribution.Latest(ctx)
	if e != nil {
		return e
	}
	status := updateStatus{Current: buildinfo.CurrentVersion(), Latest: strings.TrimPrefix(release.Tag, "v"), CheckedAt: time.Now()}
	status.Available = distribution.Newer(release.Tag, status.Current)
	record := func() { b, _ := json.Marshal(status); _ = config.Write(filepath.Join(home, "update.json"), b) }
	if !status.Available || *check {
		status.Message = "Up to date"
		if status.Available {
			status.Message = "Update available"
		}
		record()
		return printJSON(out, status)
	}
	if *automatic {
		if b, e := os.ReadFile(filepath.Join(home, "update.json")); e == nil {
			var old updateStatus
			if json.Unmarshal(b, &old) == nil && old.Updated && time.Since(old.CheckedAt) < time.Hour {
				return nil
			}
		}
	}
	temp, e := os.MkdirTemp(home, ".release-*")
	if e != nil {
		return e
	}
	defer os.RemoveAll(temp)
	exe, e := service.Executable()
	if e != nil {
		return e
	}
	source := filepath.Join(temp, "noria")
	appSource := ""
	appTarget := filepath.Join(filepath.Dir(exe), "Noria.app")
	if runtime.GOOS == "darwin" {
		archive := filepath.Join(temp, "release.tar.gz")
		if e = distribution.Download(ctx, release, distribution.ArchiveName(), archive); e != nil {
			return e
		}
		unpacked := filepath.Join(temp, "unpacked")
		if e = distribution.Extract(archive, unpacked); e != nil {
			return e
		}
		source = filepath.Join(unpacked, "noria")
		appSource = filepath.Join(unpacked, "Noria.app")
		if e = verifyApp(appSource); e != nil {
			return e
		}
	} else {
		if e = distribution.Download(ctx, release, distribution.BinaryName(), source); e != nil {
			return e
		}
		if e = os.Chmod(source, 0755); e != nil {
			return e
		}
	}
	// Downloading is outside the lifecycle lock so Stop remains responsive.
	lifecycle, e := service.Guard(home, "lifecycle")
	if e != nil {
		return e
	}
	defer lifecycle()
	// Close the admission race before replacing anything; existing work wins.
	running := daemonRunning(home)
	if running {
		b, e := controlRequestTimeout(home, "POST", "/_noria/drain", nil, 2*time.Second)
		if e != nil {
			return errors.New("running service cannot prepare a safe update; stop Noria first")
		}
		var drain struct {
			Busy bool `json:"busy"`
		}
		if json.Unmarshal(b, &drain) != nil {
			return errors.New("invalid service drain response")
		}
		if drain.Busy {
			status.Deferred = true
			status.Message = "Update ready; waiting for active requests to finish"
			record()
			return printJSON(out, status)
		}
		defer func() { _, _ = controlRequestTimeout(home, "POST", "/_noria/resume", nil, time.Second) }()
	}
	// Execute only the already checksum-verified staged binary, before touching
	// the running install. A wrong architecture/version cannot replace it.
	if e = os.Chmod(source, 0755); e != nil {
		return e
	}
	probe := exec.CommandContext(ctx, source, "--version")
	versionOutput, err := probe.Output()
	if err != nil || strings.TrimSpace(string(versionOutput)) != "noria "+status.Latest {
		return errors.New("downloaded executable did not pass version verification")
	}
	if running {
		if e = stopRuntimeForUpdate(home, exe); e != nil {
			return e
		}
	}
	if e = distribution.Replace(source, exe, 0755); e != nil {
		if running {
			_ = startService(home)
		}
		return e
	}
	changedApp := false
	if appSource != "" {
		if _, err := os.Stat(appTarget); err == nil {
			if e = replaceApp(appSource, appTarget); e != nil {
				_ = restorePreviousBinary(exe)
				if running {
					_ = startService(home)
				}
				return e
			}
			changedApp = true
		}
	}
	if running {
		if e = startService(home); e != nil {
			_ = restorePreviousBinary(exe)
			if changedApp {
				_ = os.RemoveAll(appTarget)
				_ = os.Rename(appTarget+".previous", appTarget)
			}
			_ = startService(home)
			return errors.New("updated service did not become healthy; restored the previous release")
		}
	}
	status.Updated = true
	status.Message = "Updated to " + status.Latest
	record()

	return printJSON(out, status)
}
func verifyApp(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if e := exec.CommandContext(ctx, "/usr/bin/codesign", "--verify", "--deep", "--strict", path).Run(); e != nil {
		return errors.New("menu bar app signature verification failed")
	}
	return nil
}
func replaceApp(source, target string) error {
	stage := target + ".staged"
	_ = os.RemoveAll(stage)
	if e := copyTree(source, stage); e != nil {
		return e
	}
	if e := verifyApp(stage); e != nil {
		_ = os.RemoveAll(stage)
		return e
	}
	old := target + ".previous"
	_ = os.RemoveAll(old)
	had := false
	if _, e := os.Stat(target); e == nil {
		if e = os.Rename(target, old); e != nil {
			return e
		}
		had = true
	}
	if e := os.Rename(stage, target); e != nil {
		if had {
			_ = os.Rename(old, target)
		}
		return e
	}
	return nil
}
func copyTree(source, target string) error {
	return filepath.WalkDir(source, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		rel, e := filepath.Rel(source, path)
		if e != nil {
			return e
		}
		dest := filepath.Join(target, rel)
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("application bundle cannot contain symlinks")
		}
		if d.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		info, e := d.Info()
		if e != nil {
			return e
		}
		in, e := os.Open(path)
		if e != nil {
			return e
		}
		defer in.Close()
		out, e := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
		if e != nil {
			return e
		}
		_, e = io.Copy(out, in)
		ce := out.Close()
		if e != nil {
			return e
		}
		return ce
	})
}

func stopRuntimeForUpdate(home, exe string) error {
	if e := service.SetIntent(home, exe, false); e != nil {
		return e
	}
	_, _ = controlRequestTimeout(home, "POST", "/_noria/shutdown", nil, 2*time.Second)
	if e := service.Stop(home, exe); e != nil {
		return e
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if !daemonRunning(home) {
			unlock, e := service.Guard(home, "daemon")
			if e == nil {
				unlock()
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return errors.New("previous service did not stop cleanly; install was not changed")
}
func restorePreviousBinary(exe string) error {
	previous := exe + ".previous"
	if _, e := os.Stat(previous); e != nil {
		return e
	}
	failed := exe + ".failed"
	_ = os.Remove(failed)
	if e := os.Rename(exe, failed); e != nil {
		return e
	}
	if e := os.Rename(previous, exe); e != nil {
		_ = os.Rename(failed, exe)
		return e
	}
	_ = os.Remove(failed)
	return nil
}
