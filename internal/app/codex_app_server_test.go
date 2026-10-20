package app

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestAppServerForcesNoriaForNewResumedAndForkedThreads(t *testing.T) {
	for _, method := range []string{"thread/start", "thread/resume", "thread/fork"} {
		t.Run(method, func(t *testing.T) {
			input := `{"id":22,"method":"` + method + `","params":{"threadId":"saved-old-session","modelProvider":"openai","config":{"model":"gpt-6-luna"},"excludeTurns":true}}` + "\n"
			var output bytes.Buffer
			if err := appServerInput(&output, strings.NewReader(input)); err != nil {
				t.Fatal(err)
			}
			var result struct {
				ID     int    `json:"id"`
				Method string `json:"method"`
				Params struct {
					ModelProvider string                     `json:"modelProvider"`
					ThreadID      string                     `json:"threadId"`
					Config        map[string]json.RawMessage `json:"config"`
					ExcludeTurns  bool                       `json:"excludeTurns"`
				} `json:"params"`
			}
			if err := json.Unmarshal(output.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.ID != 22 || result.Method != method || result.Params.ModelProvider != "noria_runner" || result.Params.ThreadID != "saved-old-session" || !result.Params.ExcludeTurns || string(result.Params.Config["model"]) != `"gpt-6-luna"` {
				t.Fatalf("lost native app-server fields or provider override: %+v", result)
			}
		})
	}
}

func TestAppServerForwardingPreservesUnrelatedRPCAndLargeInputs(t *testing.T) {
	large := strings.Repeat("some tool output ", 22000)
	for _, input := range []string{
		`{"id":4,"method":"turn/start","params":{"threadId":"existing","input":[]}}` + "\n",
		`{"id":5,"method":"thread/list","params":{"limit":10}}` + "\n",
		`{"id":9,"method":"thread/resume","params":null}` + "\n",
		`{"id":10,"method":"thread/resume","params":{"threadId":"x"`,
		`{"jsonrpc":"2.0","method":"initialized","params":{}}` + "\n",
		`{"id":30,"method":"turn/steer","params":{"input":"` + large + `"}}` + "\n",
	} {
		var out bytes.Buffer
		if err := appServerInput(&out, strings.NewReader(input)); err != nil {
			t.Fatal(err)
		}
		if out.String() != input {
			t.Fatal("changed unrelated native app-server message")
		}
	}
}

func TestAppServerJSONLinesNeverConcatenateIndependentRequests(t *testing.T) {
	input := `{"id":1,"method":"thread/resume","params":{"threadId":"old"}}` + "\n" + `{"id":2,"method":"turn/start","params":{"input":[]}}` + "\n"
	var out bytes.Buffer
	if err := appServerInput(&out, strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("lost message boundaries: %d", len(lines))
	}
	if !strings.Contains(lines[0], `"modelProvider":"noria_runner"`) || lines[1] != `{"id":2,"method":"turn/start","params":{"input":[]}}` {
		t.Fatal("incorrect app-server message routing")
	}
}
