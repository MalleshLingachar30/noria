package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func drain(t *testing.T, s *Server, path string) bool {
	t.Helper()
	req := httptest.NewRequest("POST", "http://127.0.0.1"+path, nil)
	req.Header.Set("Authorization", "Bearer local-secret")
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("drain status %d: %s", rr.Code, rr.Body.String())
	}
	var state struct {
		Busy bool `json:"busy"`
	}
	if e := json.Unmarshal(rr.Body.Bytes(), &state); e != nil {
		t.Fatal(e)
	}
	return state.Busy
}
func TestDrainAdmissionIsAtomicAndBusyDoesNotDisableRouting(t *testing.T) {
	var requests atomic.Int32
	s, _ := fixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.WriteHeader(200) }), "codex", 1)
	if !s.beginCall() {
		t.Fatal("initial admission rejected")
	}
	if !drain(t, s, "/_noria/drain") {
		t.Fatal("active work not detected")
	}
	if !s.beginCall() {
		t.Fatal("busy drain disabled healthy service")
	}
	s.endCall()
	s.endCall()
	if drain(t, s, "/_noria/drain") {
		t.Fatal("idle drain is busy")
	}
	if s.beginCall() {
		t.Fatal("drain allowed work")
	}
	req := httptest.NewRequest("POST", "http://127.0.0.1/responses", strings.NewReader(`{"model":"fixture"}`))
	req.Header.Set("Authorization", "Bearer local-secret")
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	if rr.Code != 503 || requests.Load() != 0 {
		t.Fatal("request escaped drained admission")
	}
	drain(t, s, "/_noria/resume")
	if !s.beginCall() {
		t.Fatal("resume failed")
	}
	s.endCall()
}
