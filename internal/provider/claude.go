package provider

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/amxv/noria/internal/model"
)

const SDKVersion = "0.3.287"

//go:embed usage-bridge.mjs
var usageBridge string

func (m *Manager) BridgeSetup(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	dir := filepath.Join(m.home, "bridge")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return errors.New("cannot create Claude bridge directory")
	}
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return errors.New("cannot resolve Claude bridge directory")
	}
	root, err := filepath.EvalSymlinks(m.home)
	if err != nil || real != filepath.Join(root, "bridge") {
		return errors.New("Claude bridge directory must not be a symlink")
	}
	_ = os.Chmod(dir, 0700)
	// Setup is the only path permitted to install dependencies. Never invoked
	// automatically by Refresh, Credentials, or an HTTP request.
	cmd := nativeCommand(ctx, cleanEnv(dir, "claude"), "npm", "install", "--prefix", dir, "--ignore-scripts", "--no-audit", "--no-fund", "--save-exact", "@anthropic-ai/claude-agent-sdk@"+SDKVersion)
	cmd.Dir = dir
	if cmd.Run() != nil {
		return errors.New("Claude bridge installation failed; verify npm/network and retry bridge setup")
	}
	return nil
}

func (m *Manager) claudeQuota(ctx context.Context, a model.Account, p string) (model.Snapshot, error) {
	started := time.Now()
	sdkDir := filepath.Join(m.home, "bridge")
	if _, err := os.Stat(filepath.Join(sdkDir, "node_modules", "@anthropic-ai", "claude-agent-sdk", "package.json")); err != nil {
		return model.Snapshot{}, errors.New("Claude usage bridge missing; run noria bridge setup")
	}
	claude, err := lookPath(m.claudeCommand)
	if err != nil {
		return model.Snapshot{}, errors.New("Claude CLI unavailable; install it before quota refresh")
	}
	claude, err = filepath.Abs(claude)
	if err != nil {
		return model.Snapshot{}, errors.New("cannot resolve Claude CLI")
	}
	cmd := nativeCommand(ctx, cleanEnv(p, "claude"), m.nodeCommand, "--input-type=module", "-", sdkDir, claude)
	defer killProcess(cmd)
	cmd.Dir = p
	cmd.Stdin = strings.NewReader(usageBridge)
	out := &boundedBuffer{max: 1 << 20}
	cmd.Stdout = out
	if cmd.Run() != nil {
		return model.Snapshot{}, errors.New("Claude usage bridge failed or timed out; verify native login and pinned SDK with bridge setup")
	}
	return parseClaude(a.ID, out.data, started)
}

func parseClaude(id string, data []byte, now time.Time) (model.Snapshot, error) {
	s := model.Snapshot{AccountID: id, ObservedAt: now}
	var root struct {
		Available *bool           `json:"rate_limits_available"`
		Limits    json.RawMessage `json:"rate_limits"`
	}
	if json.Unmarshal(data, &root) != nil || root.Available == nil || len(root.Limits) == 0 {
		return s, errors.New("Claude usage schema changed")
	}
	if !*root.Available {
		if !absent(root.Limits) {
			return s, errors.New("invalid unavailable Claude usage schema")
		}
		return s, nil
	}
	if absent(root.Limits) {
		return s, nil
	}
	var limits map[string]json.RawMessage
	if json.Unmarshal(root.Limits, &limits) != nil {
		return s, errors.New("Claude rate limits must be an object or null")
	}
	keys := make([]string, 0, len(limits))
	for key := range limits {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		// Provider billing/feature metadata is not a subscription window.
		// Ignore unknown buckets before applying the window schema.
		if key != "five_hour" && key != "seven_day" && !strings.HasPrefix(key, "five_hour_") && !strings.HasPrefix(key, "seven_day_") {
			continue
		}
		if absent(limits[key]) {
			continue
		}
		var fields map[string]json.RawMessage
		if json.Unmarshal(limits[key], &fields) != nil {
			return s, errors.New("Claude usage window schema changed")
		}
		if len(fields["utilization"]) == 0 || len(fields["resets_at"]) == 0 {
			return s, errors.New("Claude usage window fields missing")
		}
		var used *float64
		var reset *string
		if json.Unmarshal(fields["utilization"], &used) != nil || json.Unmarshal(fields["resets_at"], &reset) != nil {
			return s, errors.New("invalid Claude usage window")
		}
		if used != nil && (*used < 0 || *used > 100) {
			return s, errors.New("Claude utilization outside 0..100")
		}
		w := model.Window{Name: key, LimitID: key, Used: used}
		switch {
		case key == "five_hour":
			w.Duration = 5 * time.Hour
		case key == "seven_day":
			w.Duration = 7 * 24 * time.Hour
		case strings.HasPrefix(key, "seven_day_"):
			w.Duration = 7 * 24 * time.Hour
			w.Model = strings.TrimPrefix(key, "seven_day_")
		case strings.HasPrefix(key, "five_hour_"):
			w.Duration = 5 * time.Hour
			w.Model = strings.TrimPrefix(key, "five_hour_")
		default:
			continue
		}
		if reset != nil {
			v, err := time.Parse(time.RFC3339Nano, *reset)
			if err != nil {
				return s, errors.New("Claude usage reset must be timezone-aware ISO timestamp")
			}
			w.ResetsAt = &v
		}
		s.Windows = append(s.Windows, w)
	}
	s.Available = true
	return s, nil
}
