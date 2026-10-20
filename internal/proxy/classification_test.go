package proxy

import (
	"net/http"
	"testing"
	"time"
)

func TestWeeklyRejectionsAreQuotaRatherThanThroughput(t *testing.T) {
	for _, message := range []string{
		`{"error":{"type":"rate_limit_error","message":"Number of requests has exceeded your 7-day rate limit."}}`,
		`{"error":{"type":"rate_limit_error","message":"Your weekly limit has been reached."}}`,
	} {
		rejection, retry := classify(429, nil, []byte(message), "claude-test")
		if !retry || rejection.Kind != "quota" || rejection.Window != "seven_day" {
			t.Fatalf("weekly limit classified incorrectly: %+v", rejection)
		}
	}
	rejection, retry := classify(429, http.Header{"Retry-After": []string{"2"}}, []byte(`{"error":{"type":"rate_limit_error","message":"Too many requests per minute"}}`), "claude-test")
	if retry || rejection.Kind != "throughput" || rejection.Until.Before(time.Now()) {
		t.Fatal("throughput evicted a healthy session")
	}
}

// Subscription errors with an otherwise generic 429 must switch a resumed
// conversation to a different account before exposing any generated output.
// Request-per-minute throttling must not be mistaken for exhausted quota.
func TestCodexSubscriptionRejectionsTriggerSafeFailover(t *testing.T) {
	for _, tc := range []struct {
		name  string
		body  string
		quota bool
	}{
		{"usage limit exceeded code", `{"error":{"code":"usage_limit_exceeded"}}`, true},
		{"generic rate limit usage", `{"error":{"type":"rate_limit_error","message":"You've hit your usage limit. Try again later."}}`, true},
		{"generic rate limit reset", `{"error":{"type":"rate_limit_error","message":"Your limit has been reached. Resets tomorrow."}}`, true},
		{"resumed stream", `{"type":"response.failed","response":{"error":{"type":"rate_limit_error","message":"You've reached your limit; reset at 8:52 AM."}}}`, true},
		{"request throughput", `{"error":{"type":"rate_limit_error","message":"Too many requests per minute"}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status := 429
			if tc.name == "resumed stream" {
				status = 0
			}
			rejection, retry := classify(status, nil, []byte(tc.body), "gpt-test")
			if retry != tc.quota {
				t.Fatalf("retry=%t, rejection=%+v", retry, rejection)
			}
			if tc.quota && rejection.Kind != "quota" {
				t.Fatalf("subscription error not classified as quota: %+v", rejection)
			}
			if !tc.quota && rejection.Kind != "throughput" {
				t.Fatalf("throughput error misclassified: %+v", rejection)
			}
		})
	}
}
