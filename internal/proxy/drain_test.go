package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpdateDrainDoesNotAdmitNewWorkAndResumes(t *testing.T) {
	server, _ := fixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }), "codex", 1)
	control := func(path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "http://127.0.0.1"+path, strings.NewReader(`{}`))
		r.Header.Set("Authorization", "Bearer local-secret")
		w := httptest.NewRecorder()
		server.ServeHTTP(w, r)
		return w
	}
	server.slots <- struct{}{}
	w := control("/_noria/drain")
	var busy struct {
		Busy bool `json:"busy"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &busy) != nil || !busy.Busy {
		t.Fatal("active request wasn't protected")
	}
	<-server.slots
	w = control("/_noria/drain")
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &busy) != nil || busy.Busy {
		t.Fatal("idle drain did not engage")
	}
	w = control("/responses")
	if w.Code != 503 {
		t.Fatal("request admitted while draining")
	}
	w = control("/_noria/resume")
	if w.Code != 200 {
		t.Fatal("resume failed")
	}
	w = control("/responses")
	if w.Code == 503 {
		t.Fatal("drain stayed active")
	}
}
