package core

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/amxv/noria/internal/model"
)

// SetAccounts changes the authorized pool without moving existing sessions.
func (s *Scheduler) SetAccounts(accounts []model.Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := map[string]bool{}
	copyAccounts := make([]model.Account, len(accounts))
	for i, account := range accounts {
		if account.ID == "" || account.Provider == "" {
			return errors.New("account ID and provider are required")
		}
		if seen[account.ID] {
			return fmt.Errorf("duplicate account ID %q", account.ID)
		}
		seen[account.ID] = true
		copyAccounts[i] = cloneAccount(account)
	}
	// A completed native sign-in is authoritative for authentication only.
	// It must not clear five-hour/weekly quota blockers or move sessions.
	changed := false
	oldBlocks := map[string][]blocker{}
	oldSnapshots := map[string]model.Snapshot{}
	for _, account := range copyAccounts {
		if account.LastLogin.IsZero() {
			continue
		}
		blocks := s.state.Blockers[account.ID]
		kept := make([]blocker, 0, len(blocks))
		for _, b := range blocks {
			if b.Kind == "auth" && !b.CreatedAt.After(account.LastLogin) {
				changed = true
				continue
			}
			kept = append(kept, b)
		}
		if len(kept) != len(blocks) {
			oldBlocks[account.ID] = blocks
			s.state.Blockers[account.ID] = kept
		}
		if snapshot, ok := s.state.Snapshots[account.ID]; ok && snapshot.AuthInvalid && !snapshot.ObservedAt.After(account.LastLogin) {
			oldSnapshots[account.ID] = snapshot
			snapshot.AuthInvalid = false
			s.state.Snapshots[account.ID] = snapshot
			changed = true
		}
	}
	if changed {
		if err := s.save(); err != nil {
			for id, v := range oldBlocks {
				s.state.Blockers[id] = v
			}
			for id, v := range oldSnapshots {
				s.state.Snapshots[id] = v
			}
			return err
		}
	}
	s.accounts = copyAccounts
	return nil
}

func bindingKey(provider, session string) string {
	if session == "" {
		return ""
	}
	h := sha256.Sum256([]byte(provider + "\x00" + session))
	return hex.EncodeToString(h[:])
}

// Acquire atomically selects and reserves estimated work. Unknown telemetry is
// used only when fresh known candidates cannot serve, with bounded concurrency.
func (s *Scheduler) Acquire(provider, session, modelName string, exclude map[string]bool) (*Lease, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	key := bindingKey(provider, session)
	bound := s.state.Bindings[key]
	previousBinding := bound
	if key != "" && bound == "" {
		// Provisional affinity keeps concurrent first requests together without
		// persisting a binding before any real upstream work is accepted.
		for _, r := range s.leases {
			if r.binding == key {
				bound = r.lease.Account.ID
				break
			}
		}
	}
	var chosen *model.Account
	var fallback *model.Account
	best, fallbackBest := math.Inf(-1), math.Inf(-1)
	var earliest time.Time
	for i := range s.accounts {
		a := &s.accounts[i]
		if !matches(*a, provider, modelName) || exclude[a.ID] {
			continue
		}
		usable, known, retry := s.eligibility(a.ID, modelName, now, false)
		// The cautious unknown cap limits new placements. Existing and
		// provisional affinity cannot be displaced by a local load estimate.
		if usable && !known && a.ID != bound && s.inflight(a.ID, "") >= unknownConcurrency {
			usable, retry = false, now.Add(time.Second)
		}
		if !usable {
			if !retry.IsZero() && (earliest.IsZero() || retry.Before(earliest)) {
				earliest = retry
			}
			continue
		}
		// A healthy binding always wins, including one with unknown telemetry.
		if bound != "" && a.ID == bound {
			chosen = a
			break
		}
		score := s.score(a.ID, modelName, now)
		if known && score > best {
			chosen, best = a, score
		}
		if !known && score > fallbackBest {
			fallback, fallbackBest = a, score
		}
	}
	if chosen == nil {
		chosen = fallback
	}
	if chosen == nil {
		return nil, &UnavailableError{Provider: provider, Model: modelName, RetryAt: earliest}
	}
	id, err := randomID()
	if err != nil {
		return nil, err
	}
	lease := Lease{ID: id, Account: cloneAccount(*chosen)}
	rebind := true
	if bound != "" && bound != chosen.ID {
		for _, a := range s.accounts {
			if a.ID == bound && matches(a, provider, modelName) {
				if healthy, _, _ := s.eligibility(bound, modelName, now, true); healthy {
					rebind = false
				}
			}
		}
	}
	s.leases[id] = &reservation{lease: lease, binding: key, model: modelName, previousBinding: previousBinding, rebind: rebind}
	result := lease
	result.Account = cloneAccount(lease.Account)
	return &result, nil
}

// Accept commits affinity only when an upstream accepted actual work. A retry
// caused by a transient transport failure leaves a healthy original binding.
func (s *Scheduler) Accept(lease *Lease) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.reservation(lease)
	if err != nil {
		return err
	}
	if r.accepted {
		return nil
	}
	now := s.now()
	oldID := s.state.Bindings[r.binding]
	// Compare-and-set prevents a delayed acceptance from an older concurrent
	// request from undoing a failover another request has already committed.
	changeBinding := r.binding != "" && r.rebind && oldID == r.previousBinding
	if changeBinding {
		s.state.Bindings[r.binding] = r.lease.Account.ID
	}
	oldClocks := s.state.Clocks[r.lease.Account.ID]
	s.startClocks(r.lease.Account.ID, r.model, now)
	oldActivity, hadActivity := s.state.Activity[r.binding]
	if r.binding != "" && s.state.Bindings[r.binding] == r.lease.Account.ID {
		activity := sessionActivity{AccountID: r.lease.Account.ID, Model: r.model, LastRequest: now, CadenceSeconds: 60}
		if hadActivity {
			elapsed := now.Sub(oldActivity.LastRequest).Seconds()
			if elapsed > 0 && elapsed < 30*60 {
				previous := oldActivity.CadenceSeconds
				if previous <= 0 {
					previous = 60
				}
				activity.CadenceSeconds = .7*previous + .3*math.Max(elapsed, 1)
			}
		}
		s.state.Activity[r.binding] = activity
	}
	if err := s.save(); err != nil {
		if hadActivity {
			s.state.Activity[r.binding] = oldActivity
		} else {
			delete(s.state.Activity, r.binding)
		}
		if changeBinding {
			if oldID == "" {
				delete(s.state.Bindings, r.binding)
			} else {
				s.state.Bindings[r.binding] = oldID
			}
		}
		if oldClocks == nil {
			delete(s.state.Clocks, r.lease.Account.ID)
		} else {
			s.state.Clocks[r.lease.Account.ID] = oldClocks
		}
		return err
	}
	r.accepted = true
	return nil
}

// Reject releases an unaccepted reservation and records the applicable blocker.
// Independent model/window blockers coexist and must all clear before reuse.
func (s *Scheduler) Reject(lease *Lease, rejection model.Rejection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.reservation(lease)
	if err != nil {
		return err
	}
	if r.accepted {
		return errors.New("cannot retry an already accepted lease")
	}
	now := s.now()
	kind := rejection.Kind
	if kind != "quota" && kind != "auth" && kind != "throughput" && kind != "transient" {
		return fmt.Errorf("unknown rejection kind %q", kind)
	}
	until := rejection.Until
	if until.IsZero() {
		switch kind {
		case "transient":
			until = now.Add(time.Second)
		case "throughput":
			until = now.Add(2 * time.Second)
		case "quota":
			until = s.quotaReset(r.lease.Account.ID, rejection.Model, rejection.Window, now)
			if until.IsZero() {
				until = now.Add(time.Minute)
			}
		}
	}
	// Past Retry-After values still need a fresh read rather than an immediate
	// retry loop against a backend that has just authoritatively refused work.
	if !until.IsZero() && !until.After(now) {
		until = now.Add(time.Second)
	}
	b := blocker{Kind: kind, Source: "backend", Model: rejection.Model, Window: rejection.Window, Until: until, CreatedAt: now}
	id := r.lease.Account.ID
	previous := append([]blocker(nil), s.state.Blockers[id]...)
	blocks := append([]blocker(nil), previous...)
	merged := false
	for i, old := range blocks {
		if old.Kind == b.Kind && old.Source == b.Source && old.Model == b.Model && old.Window == b.Window {
			// Keep the longer cooldown; a second rejection cannot shorten it.
			if old.Until.IsZero() || (!b.Until.IsZero() && old.Until.After(b.Until)) {
				b.Until = old.Until
			}
			blocks[i], merged = b, true
			break
		}
	}
	if !merged {
		blocks = append(blocks, b)
	}
	s.state.Blockers[id] = blocks
	if err := s.save(); err != nil {
		s.state.Blockers[id] = previous
		return err
	}
	delete(s.leases, lease.ID)
	return nil
}

// Complete releases the reservation exactly once. Only accepted work enters
// the completion counter used to learn percent consumed per completed request.
func (s *Scheduler) Complete(lease *Lease) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if lease == nil {
		return nil
	}
	r := s.leases[lease.ID]
	if r == nil {
		return nil
	}
	// Completion must release memory reservations even when storage is full or
	// unavailable. Real completed work remains accounted in this process.
	defer delete(s.leases, lease.ID)
	if r.accepted {
		s.state.Completed[r.lease.Account.ID]++
		byModel := s.state.CompletedModels[r.lease.Account.ID]
		if byModel == nil {
			byModel = map[string]uint64{}
			s.state.CompletedModels[r.lease.Account.ID] = byModel
		}
		byModel[r.model]++
		if err := s.save(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Scheduler) reservation(l *Lease) (*reservation, error) {
	if l == nil {
		return nil, errors.New("nil lease")
	}
	r := s.leases[l.ID]
	if r == nil {
		return nil, errors.New("unknown or completed lease")
	}
	return r, nil
}

func matches(a model.Account, provider, name string) bool {
	if !a.Enabled || a.Provider != provider {
		return false
	}
	if len(a.Models) == 0 {
		return true
	}
	for _, m := range a.Models {
		if m == name {
			return true
		}
	}
	return false
}

func applicable(scope, name string) bool {
	if scope == "" {
		return true
	}
	normalize := func(v string) string {
		return strings.Trim(strings.NewReplacer("_", "-", " ", "-").Replace(strings.ToLower(v)), "-")
	}
	scope, name = normalize(scope), normalize(name)
	if scope == name {
		return true
	}
	// Claude reports family buckets (sonnet/opus), while requests include
	// versioned native model IDs. Match complete family tokens only.
	if scope == "sonnet" || scope == "opus" || scope == "haiku" || scope == "fable" {
		return strings.Contains("-"+name+"-", "-"+scope+"-")
	}
	return false
}

func windowKey(w model.Window) string {
	return w.LimitID + "\x00" + w.Model + "\x00" + w.Name + "\x00" + w.Duration.String()
}

func sameWindow(name string, w model.Window) bool {
	return name == "" || strings.EqualFold(name, w.Name)
}

func (s *Scheduler) inflight(id, modelName string) int {
	n := 0
	for _, l := range s.leases {
		if l.lease.Account.ID == id && applicable(modelName, l.model) {
			n++
		}
	}
	return n
}

func (s *Scheduler) completedFor(id, scope string) uint64 {
	if scope == "" {
		return s.state.Completed[id]
	}
	var completed uint64
	for name, count := range s.state.CompletedModels[id] {
		if applicable(scope, name) {
			completed += count
		}
	}
	return completed
}

func (s *Scheduler) startClocks(id, modelName string, now time.Time) {
	clocks := map[string]time.Time{}
	for k, v := range s.state.Clocks[id] {
		clocks[k] = v
	}
	for _, w := range s.state.Snapshots[id].Windows {
		if !applicable(w.Model, modelName) {
			continue
		}
		// Null reset data alone is not evidence that the provider's window
		// has never started. Only explicit adapter evidence permits prediction.
		if !w.Dormant || w.ResetsAt != nil || w.Duration <= 0 {
			continue
		}
		k := windowKey(w)
		if reset := clocks[k]; reset.IsZero() || !reset.After(now) {
			clocks[k] = now.Add(w.Duration)
		}
	}
	if len(clocks) > 0 {
		s.state.Clocks[id] = clocks
	}
}

func (s *Scheduler) quotaReset(id, modelName, name string, now time.Time) time.Time {
	var latest time.Time
	for _, w := range s.state.Snapshots[id].Windows {
		if !applicable(w.Model, modelName) || !sameWindow(name, w) {
			continue
		}
		// A named backend rejection identifies the exhausted window itself.
		// Without a name, healthy windows are not evidence of when the actual
		// rejection will recover, even when they have a later reset.
		if name == "" && !windowExhausted(w) {
			continue
		}
		if w.ResetsAt != nil && w.ResetsAt.After(now) && w.ResetsAt.After(latest) {
			latest = *w.ResetsAt
		}
	}
	return latest
}
