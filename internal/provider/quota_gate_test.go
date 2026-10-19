package provider

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/amxv/noria/internal/core"
	"github.com/amxv/noria/internal/model"
)

func TestCodexAccountGateDoesNotExhaustHealthyWeeklyWindow(t *testing.T) {
	now := time.Now()
	shortReset := now.Add(10 * time.Minute).Truncate(time.Second)
	data, _ := json.Marshal(map[string]any{
		"ordinaryUsageAllowed": false,
		"rateLimits": map[string]any{
			"primary":              map[string]any{"usedPercent": 100, "windowDurationMins": 300, "resetsAt": shortReset.Unix()},
			"secondary":            map[string]any{"usedPercent": 25, "windowDurationMins": 10080, "resetsAt": now.Add(6 * 24 * time.Hour).Unix()},
			"rateLimitReachedType": "rate_limit_reached",
		},
	})
	snapshot, err := parseCodex("a", data, now)
	if err != nil {
		t.Fatal(err)
	}
	scheduler, err := core.New("", []model.Account{{ID: "a", Provider: "codex", Enabled: true}})
	if err != nil {
		t.Fatal(err)
	}
	if err = scheduler.Update(snapshot); err != nil {
		t.Fatal(err)
	}
	_, err = scheduler.Acquire("codex", "session", "test", nil)
	var unavailable *core.UnavailableError
	if !errors.As(err, &unavailable) || !unavailable.RetryAt.Equal(shortReset) {
		t.Fatalf("healthy weekly quota extended actual five-hour block: %v", err)
	}
}

func TestClaudeIgnoresNonWindowBillingMetadata(t *testing.T) {
	snapshot, err := parseClaude("a", []byte(`{"rate_limits_available":true,"rate_limits":{"five_hour":{"utilization":20,"resets_at":null},"extra_usage":{"is_enabled":false,"monthly_limit":null}}}`), time.Now())
	if err != nil || len(snapshot.Windows) != 1 {
		t.Fatalf("billing metadata invalidated real windows: %v", err)
	}
}

func TestCodexLimitIDsArePreservedSeparately(t *testing.T) {
	snapshot, err := parseCodex("a", []byte(`{"rateLimits":{},"rateLimitsByLimitId":{"codex_one":{"limitName":"Same model","primary":{"usedPercent":10,"windowDurationMins":300}},"codex_two":{"limitName":"Same model","primary":{"usedPercent":20,"windowDurationMins":300}}}}`), time.Now())
	if err != nil || len(snapshot.Windows) != 2 {
		t.Fatalf("invalid scoped snapshot: %v", err)
	}
	if snapshot.Windows[0].LimitID == snapshot.Windows[1].LimitID {
		t.Fatal("distinct quota buckets collapsed")
	}
}
