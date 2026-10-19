package core

import (
	"testing"
	"time"
)

func TestUnknownResetIsNotInventedFromTraffic(t *testing.T) {
	s, now := testScheduler(t, "a")
	w := window("five_hour", 0, time.Time{}, 5*time.Hour)
	update(t, s, "a", *now, w)
	acceptComplete(t, s, acquire(t, s, "known-session"))
	if len(s.state.Clocks["a"]) != 0 {
		t.Fatal("null reset was treated as a dormant window")
	}
}

func TestLearningSkipsNewEpochEvenWhenUtilizationIncreased(t *testing.T) {
	s, now := testScheduler(t, "a")
	w := window("five_hour", 2, now.Add(time.Minute), 5*time.Hour)
	update(t, s, "a", *now, w)
	acceptComplete(t, s, acquire(t, s, ""))
	*now = now.Add(2 * time.Minute)
	w = window("five_hour", 8, now.Add(5*time.Hour), 5*time.Hour)
	update(t, s, "a", *now, w)
	if s.state.Learning["a"][windowKey(w)].Samples != 0 {
		t.Fatal("cross-reset delta trained the cost estimate")
	}
	acceptComplete(t, s, acquire(t, s, ""))
	*now = now.Add(time.Second)
	w = window("five_hour", 10, *w.ResetsAt, 5*time.Hour)
	update(t, s, "a", *now, w)
	if s.state.Learning["a"][windowKey(w)].Mean != 2 {
		t.Fatal("same-epoch learning did not resume")
	}
}

func TestRecentBoundDemandReservesExpiringAllowanceBetweenRequests(t *testing.T) {
	s, now := testScheduler(t, "a", "b")
	// Five estimated requests remain on A; its existing session is expected
	// to use all five before expiry. No request is currently in flight.
	a := window("five_hour", 95, now.Add(5*time.Minute), 5*time.Hour)
	b := window("five_hour", 20, now.Add(4*time.Hour), 5*time.Hour)
	update(t, s, "a", *now, a)
	update(t, s, "b", *now, b)
	l := s.state.Learning["a"][windowKey(a)]
	l.Mean, l.Samples = 1, 1
	s.state.Learning["a"][windowKey(a)] = l
	key := bindingKey("codex", "busy-session")
	s.state.Bindings[key] = "a"
	s.state.Activity[key] = sessionActivity{AccountID: "a", Model: "gpt-5.3-codex", LastRequest: *now, CadenceSeconds: 60}
	lease := acquire(t, s, "new-session")
	if lease.Account.ID != "b" {
		t.Fatal("new session competed with sufficient bound demand on A")
	}
	_ = s.Complete(lease)
	lease = acquire(t, s, "busy-session")
	if lease.Account.ID != "a" {
		t.Fatal("activity forecast displaced a sticky session")
	}
	_ = s.Complete(lease)
	*now = now.Add(31 * time.Minute)
	count, rate := s.recentDemand("a", "", *now)
	if count != 0 || rate != 0 {
		t.Fatal("idle session reserves future work forever")
	}
	if s.state.Bindings[key] != "a" {
		t.Fatal("forecast expiry removed hard affinity")
	}
}

func TestSeparateLimitIDsNeverShareLearningIdentity(t *testing.T) {
	w := window("seven_day", 50, time.Now().Add(time.Hour), 7*24*time.Hour)
	w.LimitID = "codex_one"
	v := w
	v.LimitID = "codex_two"
	if windowKey(w) == windowKey(v) {
		t.Fatal("distinct upstream limits collided")
	}
}
