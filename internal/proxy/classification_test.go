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
