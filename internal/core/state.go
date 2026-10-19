// Package core implements atomic, quota-aware account scheduling.
package core

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/amxv/noria/internal/model"
)

const telemetryAge = 2 * time.Minute
const unknownConcurrency = 2

type blocker struct {
	Kind      string    `json:"kind"`
	Source    string    `json:"source,omitempty"`
	Model     string    `json:"model,omitempty"`
	Window    string    `json:"window,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Until     time.Time `json:"until,omitempty"`
}

type learning struct {
	Mean       float64    `json:"mean"`
	Samples    uint64     `json:"samples"`
	Used       float64    `json:"used"`
	Completed  uint64     `json:"completed"`
	ObservedAt time.Time  `json:"observed_at"`
	ResetAt    *time.Time `json:"reset_at,omitempty"`
}

type sessionActivity struct {
	AccountID      string    `json:"account_id"`
	Model          string    `json:"model"`
	LastRequest    time.Time `json:"last_request"`
	CadenceSeconds float64   `json:"cadence_seconds"`
}

type diskState struct {
	Version         int                             `json:"version"`
	Bindings        map[string]string               `json:"bindings"`
	Activity        map[string]sessionActivity      `json:"activity"`
	Blockers        map[string][]blocker            `json:"blockers"`
	Snapshots       map[string]model.Snapshot       `json:"snapshots"`
	Learning        map[string]map[string]learning  `json:"learning"`
	Completed       map[string]uint64               `json:"completed"`
	CompletedModels map[string]map[string]uint64    `json:"completed_models"`
	Clocks          map[string]map[string]time.Time `json:"predicted_resets"`
	WindowObserved  map[string]map[string]time.Time `json:"window_observed"`
}

// Scheduler serializes selection and reservation under the same lock.
// The state file contains metadata only, never credentials or request content.
type Scheduler struct {
	mu       sync.Mutex
	path     string
	accounts []model.Account
	state    diskState
	leases   map[string]*reservation
	now      func() time.Time
}

// Lease is a single in-flight account reservation. Its ID is opaque.
type Lease struct {
	ID      string
	Account model.Account
}

type reservation struct {
	lease           Lease
	binding         string
	model           string
	accepted        bool
	previousBinding string
	rebind          bool
}

// UnavailableError reports the earliest time an eligible account could have
// all of its applicable blockers cleared. A zero RetryAt needs fresh telemetry
// or operator intervention; callers must not spin on it.
type UnavailableError struct {
	Provider string
	Model    string
	RetryAt  time.Time
}

func (e *UnavailableError) Error() string {
	if !e.RetryAt.IsZero() {
		return fmt.Sprintf("no %s account available; retry after %s", e.Provider, e.RetryAt.UTC().Format(time.RFC3339))
	}
	return fmt.Sprintf("no %s account available; refresh quota or check account authentication", e.Provider)
}

func newState() diskState {
	return diskState{Version: 1, Bindings: map[string]string{}, Activity: map[string]sessionActivity{}, Blockers: map[string][]blocker{},
		Snapshots: map[string]model.Snapshot{}, Learning: map[string]map[string]learning{},
		Completed: map[string]uint64{}, CompletedModels: map[string]map[string]uint64{}, Clocks: map[string]map[string]time.Time{}, WindowObserved: map[string]map[string]time.Time{}}
}

// New restores metadata from path. An empty path creates an in-memory scheduler.
func New(path string, accounts []model.Account) (*Scheduler, error) {
	s := &Scheduler{path: path, state: newState(), leases: map[string]*reservation{}, now: time.Now}
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("read scheduler state: %w", err)
		}
		if err == nil {
			if err := json.Unmarshal(b, &s.state); err != nil {
				return nil, fmt.Errorf("decode scheduler state: %w", err)
			}
			if s.state.Version != 1 {
				return nil, fmt.Errorf("unsupported scheduler state version: %d", s.state.Version)
			}
			// Unmarshal into an initialized state preserves absent maps, but explicit
			// JSON null still needs repair before the first mutation.
			if s.state.Bindings == nil {
				s.state.Bindings = map[string]string{}
			}
			if s.state.Activity == nil {
				s.state.Activity = map[string]sessionActivity{}
			}
			if s.state.Blockers == nil {
				s.state.Blockers = map[string][]blocker{}
			}
			if s.state.Snapshots == nil {
				s.state.Snapshots = map[string]model.Snapshot{}
			}
			if s.state.Learning == nil {
				s.state.Learning = map[string]map[string]learning{}
			}
			if s.state.Completed == nil {
				s.state.Completed = map[string]uint64{}
			}
			if s.state.CompletedModels == nil {
				s.state.CompletedModels = map[string]map[string]uint64{}
			}
			if s.state.Clocks == nil {
				s.state.Clocks = map[string]map[string]time.Time{}
			}
			if s.state.WindowObserved == nil {
				s.state.WindowObserved = map[string]map[string]time.Time{}
			}
		}
	}
	if err := s.SetAccounts(accounts); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Scheduler) save() error {
	if s.path == "" {
		return nil
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return err
	}
	b, err := json.Marshal(s.state)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".scheduler-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	defer f.Close()
	if err := f.Chmod(0600); err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, s.path); err != nil {
		return err
	}
	// Directory fsync is best effort because Windows does not support it.
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

func randomID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func cloneAccount(a model.Account) model.Account {
	a.Models = append([]string(nil), a.Models...)
	return a
}

func cloneSnapshot(s model.Snapshot) model.Snapshot {
	s.Windows = append([]model.Window(nil), s.Windows...)
	for i := range s.Windows {
		if s.Windows[i].Used != nil {
			v := *s.Windows[i].Used
			s.Windows[i].Used = &v
		}
		if s.Windows[i].ResetsAt != nil {
			v := *s.Windows[i].ResetsAt
			s.Windows[i].ResetsAt = &v
		}
		if s.Windows[i].Allowed != nil {
			v := *s.Windows[i].Allowed
			s.Windows[i].Allowed = &v
		}
	}
	return s
}
