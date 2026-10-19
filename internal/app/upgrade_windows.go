//go:build windows

package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/amxv/noria/internal/buildinfo"
	"github.com/amxv/noria/internal/config"
	"github.com/amxv/noria/internal/distribution"
	"github.com/amxv/noria/internal/host"
)

// Windows loads executables with a file lock: the process that downloads a
// verified upgrade cannot replace its own image. Start the verified new binary
// from a staged sibling; it swaps the old image after the CLI process exits.
func scheduleWindowsUpgrade(home, source, target string, wasRunning bool, version string) error {
	if !strings.EqualFold(filepath.Base(target), "noria.exe") {
		return errors.New("native Windows upgrades require an installer-managed noria.exe")
	}
	dir := filepath.Dir(target)
	staged, err := os.CreateTemp(dir, ".noria-upgrade-*.exe")
	if err != nil {
		return err
	}
	pending := staged.Name()
	defer staged.Close()
	src, err := os.Open(source)
	if err != nil {
		_ = os.Remove(pending)
		return err
	}
	_, err = io.Copy(staged, src)
	_ = src.Close()
	if err == nil {
		err = staged.Sync()
	}
	if closeErr := staged.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(pending)
		return err
	}
	if err = os.Chmod(pending, 0700); err != nil {
		_ = os.Remove(pending)
		return err
	}

	logDir := filepath.Join(home, "logs")
	if err = config.PrivateDir(logDir); err != nil {
		_ = os.Remove(pending)
		return err
	}
	logFile, err := os.OpenFile(filepath.Join(logDir, "upgrade.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		_ = os.Remove(pending)
		return err
	}
	defer logFile.Close()
	cmd := exec.Command(pending, "__apply-windows-upgrade", target, strconv.FormatBool(wasRunning), version, "--home", home)
	cmd.Dir = home
	cmd.Stdin = nil
	cmd.Stdout, cmd.Stderr = logFile, logFile
	// No provider secrets or temporary routing variables cross this boundary.
	cmd.Env = scrubNativeUpgradeEnvironment()
	host.Detach(cmd)
	if err = cmd.Start(); err != nil {
		_ = os.Remove(pending)
		return fmt.Errorf("could not start verified upgrade helper: %w", err)
	}
	return cmd.Process.Release()
}

func scrubNativeUpgradeEnvironment() []string {
	environment := make([]string, 0, len(os.Environ()))
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		if strings.HasPrefix(key, "NORIA_") || strings.HasPrefix(key, "OPENAI_") ||
			strings.HasPrefix(key, "ANTHROPIC_") || strings.HasPrefix(key, "CLAUDE_") ||
			strings.HasPrefix(key, "CODEX_") {
			continue
		}
		environment = append(environment, value)
	}
	return environment
}

func applyWindowsUpgrade(home string, args []string, out io.Writer) error {
	if len(args) != 3 {
		return errors.New("internal Windows upgrade arguments are invalid")
	}
	target, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if !strings.HasPrefix(strings.ToLower(filepath.Base(exe)), ".noria-upgrade-") ||
		!strings.EqualFold(filepath.Base(target), "noria.exe") ||
		!strings.EqualFold(filepath.Dir(target), filepath.Dir(exe)) {
		return errors.New("Windows upgrade helper must be beside the native Noria executable")
	}
	running, err := strconv.ParseBool(args[1])
	if err != nil || args[2] != buildinfo.CurrentVersion() {
		return errors.New("Windows upgrade helper version or runtime flags are invalid")
	}
	// The original updater (and possibly the previous service supervisor)
	// may still have an image handle open. Retry boundedly; do not remove an
	// installation or switch routing until an atomic replacement succeeds.
	deadline := time.Now().Add(30 * time.Second)
	for {
		err = distribution.Replace(exe, target, 0755)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("Windows upgrade could not replace the old executable: %w", err)
		}
		time.Sleep(200 * time.Millisecond)
	}
	if running {
		command := exec.Command(target, "start", "--home", home)
		command.Dir = home
		command.Env = scrubNativeUpgradeEnvironment()
		if _, err = command.CombinedOutput(); err != nil {
			_ = restorePreviousBinary(target)
			return errors.New("new service failed to restart; restored previous executable")
		}
	}
	status := updateStatus{Current: args[2], Latest: args[2], Updated: true, CheckedAt: time.Now(), Message: "Upgraded to " + args[2]}
	if b, err := json.Marshal(status); err == nil {
		_ = config.Write(filepath.Join(home, "upgrade.json"), b)
	}
	fmt.Fprintln(out, status.Message)
	return nil
}
