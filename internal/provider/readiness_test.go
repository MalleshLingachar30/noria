package provider

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/amxv/noria/internal/core"
	"github.com/amxv/noria/internal/model"
)

func TestCodexBucketAdmissionDoesNotExhaustHealthyWeek(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	shortReset, weekReset := now.Add(time.Hour), now.Add(5*24*time.Hour)
	raw := fmt.Sprintf(`{"rateLimits":{"primary":{"usedPercent":100,"windowDurationMins":300,"resetsAt":%d},"secondary":{"usedPercent":20,"windowDurationMins":10080,"resetsAt":%d},"rateLimitReachedType":"rate_limit_reached"}}`, shortReset.Unix(), weekReset.Unix())
	snapshot, err := parseCodex("a", []byte(raw), now)
	if err != nil {
		t.Fatal(err)
	}
	scheduler, err := core.New("", []model.Account{{ID: "a", Provider: "codex", Enabled: true}})
	if err != nil {
		t.Fatal(err)
	}
	if err := scheduler.Update(snapshot); err != nil {
		t.Fatal(err)
	}
	_, err = scheduler.Acquire("codex", "test", "model", nil)
	var unavailable *core.UnavailableError
	if !errors.As(err, &unavailable) || !unavailable.RetryAt.Equal(shortReset) {
		t.Fatalf("healthy weekly window delayed recovery: %v", err)
	}
	// An explicit null reached state is positive recovery, not an omitted field.
	recovered := fmt.Sprintf(`{"rateLimits":{"primary":{"usedPercent":0,"windowDurationMins":300,"resetsAt":%d},"secondary":{"usedPercent":20,"windowDurationMins":10080,"resetsAt":%d},"rateLimitReachedType":null}}`, now.Add(5*time.Hour).Unix(), weekReset.Unix())
	snapshot, err = parseCodex("a", []byte(recovered), now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := scheduler.Update(snapshot); err != nil {
		t.Fatal(err)
	}
	lease, err := scheduler.Acquire("codex", "test", "model", nil)
	if err != nil {
		t.Fatalf("positive provider admission did not recover: %v", err)
	}
	_ = scheduler.Complete(lease)
}

func TestCodexCreditGateHasNoInventedTimeBasedRecovery(t *testing.T) {
	now := time.Now()
	raw := fmt.Sprintf(`{"rateLimits":{"primary":{"usedPercent":100,"windowDurationMins":300,"resetsAt":%d},"rateLimitReachedType":"workspace_owner_credits_depleted"}}`, now.Add(time.Hour).Unix())
	snapshot, err := parseCodex("a", []byte(raw), now)
	if err != nil {
		t.Fatal(err)
	}
	for _, window := range snapshot.Windows {
		if window.Name == "provider" && (!window.Exhausted || window.ResetsAt != nil) {
			t.Fatal("credit gate borrowed a five-hour reset")
		}
	}
}

func TestCodexModelSlugOverridesDisplayLabel(t *testing.T) {
	raw := `{"rateLimitsByLimitId":{"codex":{"primary":{"usedPercent":1}},"codex_alias":{"limitName":"Human friendly display label","normalModelSlug":"gpt-test-model","primary":{"usedPercent":100,"windowDurationMins":10080}}}}`
	snapshot, err := parseCodex("a", []byte(raw), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Windows) != 2 || snapshot.Windows[1].Model != "gpt-test-model" || snapshot.Windows[1].LimitID != "codex_alias" {
		t.Fatal("scoped quota lost its machine model identity")
	}
}
