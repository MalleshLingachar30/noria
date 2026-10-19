package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/amxv/noria/internal/model"
)

func pool(ids ...string) []model.Account {
	var accounts []model.Account
	for _, id := range ids {
		accounts = append(accounts, model.Account{ID: id, Provider: "codex", Enabled: true})
	}
	return accounts
}

func testScheduler(t *testing.T, ids ...string) (*Scheduler, *time.Time) {
	t.Helper()
	s, err := New("", pool(ids...))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	return s, &now
}

func window(name string, used float64, reset time.Time, duration time.Duration) model.Window {
	w := model.Window{Name: name, Used: &used, Duration: duration}
	if !reset.IsZero() {
		w.ResetsAt = &reset
	}
	return w
}

func update(t *testing.T, s *Scheduler, id string, at time.Time, windows ...model.Window) {
	t.Helper()
	if err := s.Update(model.Snapshot{AccountID: id, ObservedAt: at, Available: true, Windows: windows}); err != nil {
		t.Fatal(err)
	}
}

func acquire(t *testing.T, s *Scheduler, session string) *Lease {
	t.Helper()
	l, err := s.Acquire("codex", session, "gpt-5.3-codex", nil)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func acceptComplete(t *testing.T, s *Scheduler, l *Lease) {
	t.Helper()
	if err := s.Accept(l); err != nil {
		t.Fatal(err)
	}
	if err := s.Complete(l); err != nil {
		t.Fatal(err)
	}
}

func TestAffinityExpiryAndNoSnapback(t *testing.T) {
	s, now := testScheduler(t, "a", "b")
	update(t, s, "a", *now, window("week", 95, now.Add(10*time.Minute), 7*24*time.Hour))
	update(t, s, "b", *now, window("week", 20, now.Add(5*24*time.Hour), 7*24*time.Hour))
	l := acquire(t, s, "conversation")
	if l.Account.ID != "a" {
		t.Fatalf("expiring 5%% allowance lost placement: %s", l.Account.ID)
	}
	acceptComplete(t, s, l)
	*now = now.Add(time.Second)
	update(t, s, "b", *now, window("week", 0, now.Add(time.Minute), 7*24*time.Hour))
	l = acquire(t, s, "conversation")
	if l.Account.ID != "a" {
		t.Fatal("healthy session migrated due to changed scores")
	}
	if err := s.Reject(l, model.Rejection{Kind: "quota", Window: "week", Until: now.Add(10 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	l = acquire(t, s, "conversation")
	if l.Account.ID != "b" {
		t.Fatal("did not fail over")
	}
	if got := s.state.Bindings[bindingKey("codex", "conversation")]; got != "a" {
		t.Fatal("bound before acceptance")
	}
	acceptComplete(t, s, l)
	*now = now.Add(11 * time.Minute)
	update(t, s, "a", *now, window("week", 0, now.Add(time.Minute), 7*24*time.Hour))
	update(t, s, "b", *now, window("week", 80, now.Add(7*24*time.Hour), 7*24*time.Hour))
	l = acquire(t, s, "conversation")
	if l.Account.ID != "b" {
		t.Fatal("snapped back to recovered original account")
	}
	_ = s.Complete(l)
}

func TestTransientAndThroughputPreserveOriginalAffinity(t *testing.T) {
	for _, kind := range []string{"transient", "throughput"} {
		t.Run(kind, func(t *testing.T) {
			s, now := testScheduler(t, "a", "b")
			for _, id := range []string{"a", "b"} {
				update(t, s, id, *now, window("short", 20, now.Add(time.Hour), 5*time.Hour))
			}
			acceptComplete(t, s, acquire(t, s, "session"))
			l := acquire(t, s, "session")
			if err := s.Reject(l, model.Rejection{Kind: kind}); err != nil {
				t.Fatal(err)
			}
			l = acquire(t, s, "session")
			if l.Account.ID != "b" {
				t.Fatal("retry did not use alternate account")
			}
			acceptComplete(t, s, l)
			if got := s.state.Bindings[bindingKey("codex", "session")]; got != "a" {
				t.Fatalf("network blip migrated binding to %s", got)
			}
			*now = now.Add(3 * time.Second)
			l = acquire(t, s, "session")
			if l.Account.ID != "a" {
				t.Fatal("original binding did not survive")
			}
			_ = s.Complete(l)
		})
	}
}

func TestConcurrentAcceptDoesNotUndoFailover(t *testing.T) {
	s, now := testScheduler(t, "a", "b")
	for _, id := range []string{"a", "b"} {
		update(t, s, id, *now, window("short", 20, now.Add(time.Hour), 5*time.Hour))
	}
	acceptComplete(t, s, acquire(t, s, "session"))
	delayed := acquire(t, s, "session")
	rejected := acquire(t, s, "session")
	if err := s.Reject(rejected, model.Rejection{Kind: "quota", Until: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	acceptComplete(t, s, acquire(t, s, "session"))
	acceptComplete(t, s, delayed)
	if got := s.state.Bindings[bindingKey("codex", "session")]; got != "b" {
		t.Fatal("late acceptance undid failover")
	}
}

func TestConcurrentFirstRequestsShareProvisionalAffinity(t *testing.T) {
	s, now := testScheduler(t, "a", "b", "c")
	for _, id := range []string{"a", "b", "c"} {
		update(t, s, id, *now, window("short", 20, now.Add(time.Hour), 5*time.Hour))
	}
	var wg sync.WaitGroup
	leases := make(chan *Lease, 20)
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l, err := s.Acquire("codex", "same-new-session", "gpt-5.3-codex", nil)
			if err != nil {
				errs <- err
				return
			}
			leases <- l
		}()
	}
	wg.Wait()
	close(leases)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if len(s.state.Bindings) != 0 {
		t.Fatal("provisional binding persisted before acceptance")
	}
	var first string
	for l := range leases {
		if first == "" {
			first = l.Account.ID
		}
		if l.Account.ID != first {
			t.Fatalf("same new session split over %s and %s", first, l.Account.ID)
		}
		acceptComplete(t, s, l)
	}
	if len(s.state.Bindings) != 1 {
		t.Fatal("accepted session did not bind exactly once")
	}
}

func TestFailoverDoesNotSnapBackWhenOldResetPassesBeforeAccept(t *testing.T) {
	s, now := testScheduler(t, "a", "b")
	for _, id := range []string{"a", "b"} {
		update(t, s, id, *now, window("short", 20, now.Add(time.Hour), 5*time.Hour))
	}
	acceptComplete(t, s, acquire(t, s, "session"))
	l := acquire(t, s, "session")
	if err := s.Reject(l, model.Rejection{Kind: "quota", Until: now.Add(time.Second)}); err != nil {
		t.Fatal(err)
	}
	l = acquire(t, s, "session")
	*now = now.Add(2 * time.Second)
	acceptComplete(t, s, l)
	if got := s.state.Bindings[bindingKey("codex", "session")]; got != "b" {
		t.Fatal("failover reverted during acceptance")
	}
}

func TestTwentyAtomicReservationsSpreadWithoutDoubleCounting(t *testing.T) {
	s, now := testScheduler(t, "a", "b", "c")
	for _, id := range []string{"a", "b", "c"} {
		update(t, s, id, *now, window("week", 50, now.Add(7*24*time.Hour), 7*24*time.Hour))
	}
	var wg sync.WaitGroup
	leases := make(chan *Lease, 20)
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l, err := s.Acquire("codex", "", "gpt-5.3-codex", nil)
			if err != nil {
				errs <- err
				return
			}
			leases <- l
		}()
	}
	wg.Wait()
	close(leases)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	counts := map[string]int{}
	seen := map[string]bool{}
	for l := range leases {
		if seen[l.ID] {
			t.Fatal("duplicate lease")
		}
		seen[l.ID] = true
		counts[l.Account.ID]++
		acceptComplete(t, s, l)
		if err := s.Complete(l); err != nil {
			t.Fatal(err)
		}
	}
	if len(seen) != 20 || len(s.leases) != 0 {
		t.Fatalf("reservation leak: %d issued, %d remain", len(seen), len(s.leases))
	}
	for id, n := range counts {
		if n < 6 || n > 7 {
			t.Fatalf("load not reserved atomically: %s has %d", id, n)
		}
	}
	var completed uint64
	for _, n := range s.state.Completed {
		completed += n
	}
	if completed != 20 {
		t.Fatalf("completion counted %d times", completed)
	}
	if len(s.state.Bindings) != 0 {
		t.Fatal("empty sessions coalesced into a global binding")
	}
}

func TestUnknownTelemetryFallbackIsBoundedAndDoesNotDisplaceKnown(t *testing.T) {
	s, now := testScheduler(t, "unknown", "known")
	update(t, s, "known", *now, window("short", 99.99, now.Add(time.Hour), 5*time.Hour))
	l := acquire(t, s, "")
	if l.Account.ID != "known" {
		t.Fatal("unknown account displaced a fresh usable one")
	}
	if err := s.Reject(l, model.Rejection{Kind: "quota", Until: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	var unknown []*Lease
	for i := 0; i < unknownConcurrency; i++ {
		unknown = append(unknown, acquire(t, s, ""))
	}
	if _, err := s.Acquire("codex", "", "gpt-5.3-codex", nil); err == nil {
		t.Fatal("unbounded unknown telemetry assignments")
	}
	for _, l := range unknown {
		_ = s.Complete(l)
	}
	if l := acquire(t, s, ""); l.Account.ID != "unknown" {
		t.Fatal("unknown reservation did not release")
	}
}

func TestPersistedMetadataAndSafeStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", "scheduler.json")
	accounts := pool("a", "b")
	accounts[0].Profile = "secret-profile-path"
	accounts[0].Endpoint = "https://secret-token@example.invalid"
	s, err := New(path, accounts)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	unstarted := window("short", 0, time.Time{}, 5*time.Hour)
	unstarted.Dormant = true // explicit fixture evidence, not inferred from null reset
	update(t, s, "a", now, unstarted)
	acceptComplete(t, s, acquire(t, s, "private-session"))
	l := acquire(t, s, "private-session")
	if err := s.Reject(l, model.Rejection{Kind: "quota", Window: "short", Until: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	restored, err := New(path, accounts)
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.state.Bindings) != 1 || len(restored.state.Clocks["a"]) != 1 || len(restored.state.Blockers["a"]) != 1 {
		t.Fatal("metadata did not survive restart")
	}
	l = acquire(t, restored, "private-session")
	if l.Account.ID != "b" {
		t.Fatal("restart forgot quota rejection")
	}
	acceptComplete(t, restored, l)
	restored, err = New(path, accounts)
	if err != nil {
		t.Fatal(err)
	}
	l = acquire(t, restored, "private-session")
	if l.Account.ID != "b" {
		t.Fatal("restart forgot failover binding")
	}
	if info, err := os.Stat(path); err != nil || runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatalf("unsafe state permissions: %v %v", info, err)
	}
	if info, err := os.Stat(filepath.Dir(path)); err != nil || runtime.GOOS != "windows" && info.Mode().Perm() != 0700 {
		t.Fatalf("unsafe parent permissions: %v %v", info, err)
	}
	b, err := json.Marshal(restored.Status())
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"private-session", "secret-profile-path", "secret-token"} {
		if strings.Contains(string(b), secret) {
			t.Fatalf("status leaked %s", secret)
		}
	}
	b, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "private-session") || strings.Contains(string(b), "secret-token") {
		t.Fatal("state contains request identifiers or credentials")
	}
}

func TestBindingsNeverEvictedByPoolSizeOrAge(t *testing.T) {
	s, now := testScheduler(t, "a", "b")
	update(t, s, "a", *now, window("short", 20, now.Add(time.Hour), 5*time.Hour))
	acceptComplete(t, s, acquire(t, s, "historical-session"))
	// Seed past the former retention limit without thousands of disk writes.
	for i := 0; i < 65537; i++ {
		s.state.Bindings[fmt.Sprintf("idle-%06d", i)] = "a"
	}
	*now = now.Add(60 * 24 * time.Hour)
	update(t, s, "a", *now, window("short", 80, now.Add(5*time.Hour), 5*time.Hour))
	update(t, s, "b", *now, window("short", 0, now.Add(time.Minute), 5*time.Hour))
	acceptComplete(t, s, acquire(t, s, "new-session"))
	if len(s.state.Bindings) != 65539 {
		t.Fatalf("historical bindings were evicted: %d remain", len(s.state.Bindings))
	}
	l := acquire(t, s, "historical-session")
	if l.Account.ID != "a" {
		t.Fatal("historical healthy session migrated due to age or pool size")
	}
	_ = s.Complete(l)
}

func TestUnknownConcurrencyNeverDisplacesBoundOrProvisionalAffinity(t *testing.T) {
	for _, accepted := range []bool{false, true} {
		t.Run(fmt.Sprintf("accepted-%t", accepted), func(t *testing.T) {
			s, _ := testScheduler(t, "a", "b")
			first := acquire(t, s, "same-session")
			var leases []*Lease
			if accepted {
				acceptComplete(t, s, first)
			} else {
				leases = append(leases, first)
			}
			for i := 0; i < 5; i++ {
				l := acquire(t, s, "same-session")
				if l.Account.ID != first.Account.ID {
					t.Fatal("unknown concurrency estimate displaced affinity")
				}
				leases = append(leases, l)
			}
			// A separate new session still obeys the cautious placement cap.
			other := acquire(t, s, "other-session")
			if other.Account.ID == first.Account.ID {
				t.Fatal("new-session unknown placement ignored concurrency cap")
			}
			_ = s.Complete(other)
			for _, l := range leases {
				_ = s.Complete(l)
			}
		})
	}
}

func TestCompletionReleasesReservationEvenWhenPersistenceFails(t *testing.T) {
	s, _ := testScheduler(t, "a")
	l := acquire(t, s, "")
	if err := s.Accept(l); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Join(t.TempDir(), "regular-file")
	if err := os.WriteFile(parent, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	s.path = filepath.Join(parent, "scheduler.json")
	if err := s.Complete(l); err == nil {
		t.Fatal("persistence failure hidden")
	}
	if len(s.leases) != 0 {
		t.Fatal("storage failure leaked reservation")
	}
	if s.state.Completed["a"] != 1 {
		t.Fatal("storage failure forgot real completed work")
	}
	if err := s.Complete(l); err != nil {
		t.Fatal(err)
	}
	if s.state.Completed["a"] != 1 {
		t.Fatal("completion retry counted work twice")
	}
}

func unavailable(t *testing.T, s *Scheduler, modelName string) *UnavailableError {
	t.Helper()
	_, err := s.Acquire("codex", "", modelName, nil)
	var unavailable *UnavailableError
	if !errors.As(err, &unavailable) {
		t.Fatalf("expected unavailable, got %v", err)
	}
	return unavailable
}
