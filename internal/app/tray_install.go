package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/amxv/noria/internal/buildinfo"
	"github.com/amxv/noria/internal/config"
	"github.com/amxv/noria/internal/distribution"
)

// Earlier Noria versions upgraded only the standalone CLI. If one of those
// installations reaches a tray-enabled release without its companion asset,
// repair it on the first `noria tray open`, using the same checksummed archive
// and safe extract/replace code as the regular updater.
func ensureTrayInstalled(home, exe string) error {
	target := trayBinary(exe)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	// Two simultaneous GUI launches should never both replace the same binary.
	unlock, err := config.LockContext(ctx, home, "tray-install")
	if err != nil {
		return err
	}
	defer unlock()
	info, err := os.Lstat(target)
	if err == nil {
		if !info.Mode().IsRegular() {
			return errors.New("Noria tray path is not a regular file")
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	version := buildinfo.CurrentVersion()
	if !distribution.Newer(version, "0.0.0") {
		return errors.New("tray executable is missing; build the companion noria-tray binary")
	}
	release, err := distribution.Tagged(ctx, version)
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(home, ".tray-install-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	archive := filepath.Join(tmp, "release.tar.gz")
	if err := distribution.Download(ctx, release, distribution.ArchiveName(), archive); err != nil {
		return fmt.Errorf("download tray companion: %w", err)
	}
	unpacked := filepath.Join(tmp, "unpacked")
	if err := distribution.Extract(archive, unpacked); err != nil {
		return err
	}
	source := filepath.Join(unpacked, distribution.TrayName())
	if err := os.Chmod(source, 0755); err != nil {
		return err
	}
	output, err := exec.CommandContext(ctx, source, "--check").Output()
	if err != nil || strings.TrimSpace(string(output)) != "noria-tray "+version+" ready" {
		return errors.New("downloaded tray failed version verification")
	}
	_, finish, err := distribution.Swap(source, target, 0755)
	if err != nil {
		return err
	}
	finish()
	return nil
}
