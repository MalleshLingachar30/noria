package core

import (
	"github.com/amxv/noria/internal/model"
	"testing"
	"time"
)

func TestReloginClearsOnlyOlderAuthenticationFailures(t *testing.T) {
	s, now := testScheduler(t, "a")
	first := acquire(t, s, "session")
	second := acquire(t, s, "session")
	if e := s.Reject(first, model.Rejection{Kind: "auth"}); e != nil {
		t.Fatal(e)
	}
	if e := s.Reject(second, model.Rejection{Kind: "quota", Window: "week", Until: now.Add(time.Hour)}); e != nil {
		t.Fatal(e)
	}
	*now = now.Add(time.Second)
	accounts := pool("a")
	accounts[0].LastLogin = *now
	if e := s.SetAccounts(accounts); e != nil {
		t.Fatal(e)
	}
	for _, b := range s.state.Blockers["a"] {
		if b.Kind == "auth" {
			t.Fatal("successful relogin left an old auth failure")
		}
	}
	if len(s.state.Blockers["a"]) != 1 || s.state.Blockers["a"][0].Kind != "quota" {
		t.Fatal("relogin improperly restored quota")
	}
}
