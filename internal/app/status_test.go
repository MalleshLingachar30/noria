package app

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/amxv/noria/internal/model"
)

func TestReadableQuotaStatus(t *testing.T) {
	now := time.Now()
	used, reset := 95.0, now.Add(45*time.Minute)
	value := map[string]any{"bindings": 2, "accounts": []any{map[string]any{
		"id": "codex-a", "provider": "codex", "enabled": true,
		"snapshot": model.Snapshot{Available: true, ObservedAt: now, Windows: []model.Window{{Duration: 7 * 24 * time.Hour, Used: &used, ResetsAt: &reset}}},
	}}}
	data, _ := json.Marshal(value)
	var output bytes.Buffer
	if err := renderStatus(&output, data, now); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"codex-a", "5.0% / 45m", "2 saved session", "unknown"} {
		if !strings.Contains(output.String(), text) {
			t.Fatalf("missing %q in status: %s", text, output.String())
		}
	}
}
