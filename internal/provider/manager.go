// Package provider reads native, isolated account profiles and quota controls.
package provider

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/amxv/noria/internal/config"
	"github.com/amxv/noria/internal/model"
)

type Manager struct {
	home          string
	mu            sync.Mutex
	gates         map[string]chan struct{}
	snapshots     map[string]model.Snapshot
	completed     map[string]time.Time
	codexCommand  string
	claudeCommand string
}

func New(home string) *Manager {
	resolve := func(name string) string {
		if c, e := config.Load(home); e == nil && c.ToolPath != "" {
			for _, dir := range filepath.SplitList(c.ToolPath) {
				p := filepath.Join(dir, name)
				if q, e := exec.LookPath(p); e == nil {
					return q
				}
			}
		}
		return name
	}
	return &Manager{home: home, gates: make(map[string]chan struct{}), snapshots: make(map[string]model.Snapshot), completed: make(map[string]time.Time), codexCommand: resolve("codex"), claudeCommand: resolve("claude")}
}

func (m *Manager) profile(a model.Account) (string, error) {
	if a.Provider != "codex" && a.Provider != "claude" {
		return "", errors.New("unsupported provider")
	}
	if !filepath.IsAbs(a.Profile) {
		return "", errors.New("account profile must be absolute")
	}
	root, err := filepath.EvalSymlinks(m.home)
	if err != nil {
		return "", errors.New("Noria home is unavailable")
	}
	p, err := filepath.EvalSymlinks(a.Profile)
	if err != nil {
		return "", errors.New("account profile is unavailable; run account login")
	}
	rel, err := filepath.Rel(root, p)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("account profile must be isolated below Noria home")
	}
	return p, nil
}

func (m *Manager) lock(ctx context.Context, profile string) (func(), error) {
	return config.LockContext(ctx, profile, "native")
}

func (m *Manager) Credentials(ctx context.Context, a model.Account) (model.Credential, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	p, err := m.profile(a)
	if err != nil {
		return model.Credential{}, err
	}
	c, expired, err := m.readCredential(ctx, a, p)
	if err != nil {
		return c, err
	}
	if expired {
		// Valid tokens are read without waiting for a passive quota probe.
		// Only rotation needs the exclusive profile lock. Recheck after
		// acquiring it because another process may already have refreshed.
		unlock, lockErr := m.lock(ctx, p)
		if lockErr != nil {
			return model.Credential{}, lockErr
		}
		defer unlock()
		c, expired, err = m.readCredential(ctx, a, p)
		if err != nil || !expired {
			return c, err
		}
		if a.Provider == "codex" {
			_, err = m.codexControl(ctx, a, p, true)
		} else {
			_, err = m.refreshNative(ctx, a, p)
		}
		if err != nil {
			return model.Credential{}, fmt.Errorf("native credential refresh failed; run account login: %w", err)
		}
		c, expired, err = m.readCredential(ctx, a, p)
		if err != nil {
			return model.Credential{}, err
		}
		if expired {
			return model.Credential{}, errors.New("native account credentials expired; run account login")
		}
	}
	return c, nil
}

func (m *Manager) Refresh(ctx context.Context, a model.Account) (model.Snapshot, error) {
	started := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	p, err := m.profile(a)
	if err != nil {
		return model.Snapshot{}, err
	}
	unlock, err := m.lock(ctx, p)
	if err != nil {
		return model.Snapshot{}, err
	}
	defer unlock()
	m.mu.Lock()
	cached, ok := m.snapshots[p]
	completed := m.completed[p]
	m.mu.Unlock()
	if ok && completed.After(started) {
		cached.AccountID = a.ID
		return cached, nil
	}
	s, err := m.refreshNative(ctx, a, p)
	if err == nil {
		m.mu.Lock()
		m.snapshots[p] = s
		m.completed[p] = time.Now()
		m.mu.Unlock()
	}
	return s, err
}

func (m *Manager) refreshNative(ctx context.Context, a model.Account, p string) (model.Snapshot, error) {
	if a.Provider == "codex" {
		return m.codexQuota(ctx, a, p)
	}
	return m.claudeQuota(ctx, a, p)
}

func (m *Manager) Doctor() []string {
	var issues []string
	for _, tool := range []string{m.codexCommand, m.claudeCommand} {
		if _, err := lookPath(tool); err != nil {
			issues = append(issues, tool+" executable unavailable")
		}
	}
	return issues
}
