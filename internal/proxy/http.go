package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/amxv/noria/internal/core"
	"github.com/amxv/noria/internal/model"
)

func (s *Server) serveNative(w http.ResponseWriter, r *http.Request, provider, path string, body []byte, object map[string]json.RawMessage) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()
	session, requestedModel := sessionKey(r, object), modelName(object)
	excluded := map[string]bool{}
	for attempt := 0; attempt < maxAttempts; attempt++ {
		lease, err := s.scheduler.Acquire(provider, session, requestedModel, excluded)
		if err != nil {
			failUnavailable(w, err)
			return
		}
		credential, err := s.backend.Credentials(ctx, lease.Account)
		if err != nil || credential.Token == "" {
			// A busy native credential lock or temporary keychain failure is
			// not an authoritative authentication rejection.
			_ = s.scheduler.Complete(lease)
			excluded[lease.Account.ID] = true
			continue
		}
		address, err := endpoint(lease.Account, path, r.URL.RawQuery)
		if err != nil {
			_ = s.scheduler.Complete(lease)
			fail(w, http.StatusBadGateway, "invalid upstream endpoint")
			return
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, address, bytes.NewReader(body))
		if err != nil {
			_ = s.scheduler.Complete(lease)
			fail(w, http.StatusBadGateway, "cannot construct upstream request")
			return
		}
		req.Header = upstreamHeaders(r.Header, provider, credential)
		resp, err := s.client.Do(req)
		if err != nil {
			_ = s.scheduler.Complete(lease)
			if ctx.Err() == nil {
				fail(w, http.StatusBadGateway, "upstream connection failed")
			}
			return
		}
		if resp.StatusCode >= 400 {
			prefix, readErr := io.ReadAll(io.LimitReader(resp.Body, maxPrelude))
			rejection, retry := classify(resp.StatusCode, resp.Header, prefix, requestedModel)
			if retry && attempt+1 < maxAttempts {
				_ = resp.Body.Close()
				if s.reject(ctx, lease, rejection) != nil {
					fail(w, http.StatusServiceUnavailable, "cannot persist account rejection")
					return
				}
				excluded[lease.Account.ID] = true
				continue
			}
			if rejection.Kind != "" {
				_ = s.scheduler.Reject(lease, rejection)
				_ = s.scheduler.Complete(lease)
			} else {
				_ = s.scheduler.Complete(lease)
			}
			copyHeaders(w.Header(), resp.Header)
			w.WriteHeader(resp.StatusCode)
			_, _ = w.Write(prefix)
			if readErr == nil {
				_, _ = io.Copy(w, resp.Body)
			}
			_ = resp.Body.Close()
			return
		}
		var prelude []byte
		reader := bufio.NewReaderSize(resp.Body, 8192)
		if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
			var retry bool
			var rejection model.Rejection
			prelude, rejection, retry, err = inspectSSE(reader, requestedModel)
			if retry && attempt+1 < maxAttempts {
				_ = resp.Body.Close()
				if s.reject(ctx, lease, rejection) != nil {
					fail(w, http.StatusServiceUnavailable, "cannot persist account rejection")
					return
				}
				excluded[lease.Account.ID] = true
				continue
			}
			if retry {
				_ = s.scheduler.Reject(lease, rejection)
			} else if err != nil && len(prelude) == 0 {
				_ = resp.Body.Close()
				_ = s.scheduler.Complete(lease)
				fail(w, http.StatusBadGateway, "upstream stream closed before response")
				return
			} else {
				if err := s.scheduler.Accept(lease); err != nil {
					_ = resp.Body.Close()
					_ = s.scheduler.Complete(lease)
					fail(w, http.StatusServiceUnavailable, "cannot persist session affinity")
					return
				}
			}
		} else {
			if err := s.scheduler.Accept(lease); err != nil {
				_ = resp.Body.Close()
				_ = s.scheduler.Complete(lease)
				fail(w, http.StatusServiceUnavailable, "cannot persist session affinity")
				return
			}
		}
		copyHeaders(w.Header(), resp.Header)
		w.WriteHeader(resp.StatusCode)
		if len(prelude) > 0 {
			_, err = w.Write(prelude)
			flush(w)
		}
		if err == nil {
			_, _ = io.Copy(flushingWriter{w}, reader)
		}
		_ = resp.Body.Close()
		_ = s.scheduler.Complete(lease)
		return
	}
	fail(w, http.StatusServiceUnavailable, "account retry limit reached")
}

func (s *Server) reject(_ context.Context, lease *core.Lease, rejection model.Rejection) error {
	if err := s.scheduler.Reject(lease, rejection); err != nil {
		_ = s.scheduler.Complete(lease)
		return err
	}
	// Fetch the exhausted account's precise reset without delaying failover.
	// A single bounded control read per account is independent of this request.
	s.refreshQuota(lease.Account)
	return nil
}

func (s *Server) refreshQuota(account model.Account) {
	s.mu.Lock()
	if s.closed || s.refreshing[account.ID] || len(s.refreshing) >= 4 {
		s.mu.Unlock()
		return
	}
	s.refreshing[account.ID] = true
	s.refreshWorkers.Add(1)
	s.mu.Unlock()
	go func() {
		defer s.refreshWorkers.Done()
		defer func() { s.mu.Lock(); delete(s.refreshing, account.ID); s.mu.Unlock() }()
		ctx, cancel := context.WithTimeout(s.refreshCtx, 25*time.Second)
		defer cancel()
		if snapshot, err := s.backend.Refresh(ctx, account); err == nil {
			_ = s.scheduler.Update(snapshot)
		}
	}()
}

type flushingWriter struct{ http.ResponseWriter }

func (w flushingWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	flush(w.ResponseWriter)
	return n, err
}
func flush(w http.ResponseWriter) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// inspectSSE holds only harmless lifecycle events. Once any output is committed,
// subsequent failures pass through verbatim and can never splice generations.
func inspectSSE(reader *bufio.Reader, requestedModel string) ([]byte, model.Rejection, bool, error) {
	var held, event []byte
	for len(held) < maxPrelude {
		line, err := reader.ReadSlice('\n')
		held = append(held, line...)
		event = append(event, line...)
		if len(held) >= maxPrelude {
			return held, model.Rejection{}, false, nil
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if len(bytes.TrimSpace(line)) == 0 || err != nil {
			data := sseData(event)
			if rejection, retry := classify(0, nil, data, requestedModel); retry {
				return held, rejection, true, nil
			}
			if meaningfulEvent(data) {
				return held, model.Rejection{}, false, nil
			}
			event = event[:0]
		}
		if err != nil {
			return held, model.Rejection{}, false, err
		}
	}
	return held, model.Rejection{}, false, nil
}

func sseData(event []byte) []byte {
	var data []byte
	for _, line := range bytes.Split(event, []byte{'\n'}) {
		line = bytes.TrimSuffix(line, []byte{'\r'})
		if bytes.HasPrefix(line, []byte("data:")) {
			if len(data) > 0 {
				data = append(data, '\n')
			}
			value := bytes.TrimPrefix(line, []byte("data:"))
			value = bytes.TrimPrefix(value, []byte{' '})
			data = append(data, value...)
		}
	}
	return data
}

func meaningfulEvent(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	var event struct {
		Type    string `json:"type"`
		Message struct {
			Content []json.RawMessage `json:"content"`
		} `json:"message"`
	}
	if json.Unmarshal(data, &event) != nil {
		return true
	}
	if event.Type == "message_start" && len(event.Message.Content) == 0 {
		return false
	}
	return event.Type != "response.created" && event.Type != "response.in_progress" && event.Type != "ping"
}

func classify(status int, header http.Header, body []byte, requestedModel string) (model.Rejection, bool) {
	var event map[string]any
	_ = json.Unmarshal(body, &event)
	if n, ok := event["status"].(float64); ok && status == 0 {
		status = int(n)
	}
	// Examine provider error objects only, never arbitrary generated text.
	errorObject := event
	if response, ok := event["response"].(map[string]any); ok {
		if e, ok := response["error"].(map[string]any); ok {
			errorObject = e
		}
	}
	if inner, ok := errorObject["body"].(map[string]any); ok {
		errorObject = inner
	}
	if inner, ok := errorObject["error"].(map[string]any); ok {
		errorObject = inner
	}
	code, _ := errorObject["code"].(string)
	kind, _ := errorObject["type"].(string)
	message, _ := errorObject["message"].(string)
	marker := strings.ToLower(code + " " + kind)
	lowerMessage := strings.ToLower(message)
	shortWindow := containsAny(lowerMessage, "5-hour", "five-hour", "5 hour", "five hour", "five_hour")
	weeklyWindow := containsAny(lowerMessage, "7-day", "seven-day", "7 day", "seven day", "weekly", "seven_day")
	// Native subscription limits and auth apply across models unless the
	// provider explicitly identifies a model-specific allowance.
	rejection := model.Rejection{}
	if shortWindow && !weeklyWindow {
		rejection.Window = "five_hour"
	} else if weeklyWindow && !shortWindow {
		rejection.Window = "seven_day"
	}
	if strings.Contains(marker, "model") {
		rejection.Model = requestedModel
	}
	switch {
	case status == 401 || strings.Contains(marker, "authentication_error") || strings.Contains(marker, "invalid_api_key") || strings.Contains(marker, "token_expired"):
		rejection.Kind = "auth"
		rejection.Model = ""
	case strings.Contains(marker, "usage_limit_reached") || strings.Contains(marker, "insufficient_quota") || strings.Contains(marker, "quota_exceeded") || strings.Contains(marker, "model_limit_reached"):
		rejection.Kind = "quota"
	case status == 429 || strings.Contains(marker, "rate_limit"):
		// Claude uses rate_limit_error for both short throughput and subscription
		// limits. Explicit reset/usage language is needed to evict an affinity.
		lower := lowerMessage
		if shortWindow || weeklyWindow || strings.Contains(lower, "usage limit") || strings.Contains(lower, "extra usage") || strings.Contains(lower, "rate limit resets") || strings.Contains(lower, "hit your limit") || (strings.Contains(lower, "reached your") && strings.Contains(lower, "reset")) {
			rejection.Kind = "quota"
		} else {
			rejection.Kind = "throughput"
		}
	case status >= 500:
		rejection.Kind = "transient"
	default:
		return rejection, false
	}
	// Claude explicitly distinguishes subscription windows from short request
	// throughput in native response headers. Gate only rejected allowances;
	// an approaching or warning window must remain usable.
	if status == 429 && header != nil {
		shared := strings.EqualFold(header.Get("Anthropic-Ratelimit-Unified-5h-Status"), "rejected") || strings.EqualFold(header.Get("Anthropic-Ratelimit-Unified-7d-Status"), "rejected")
		rejectedWindows := 0
		for _, window := range []string{"5h", "7d", "7d_oi"} {
			if window == "7d_oi" && shared {
				continue
			}
			prefix := "Anthropic-Ratelimit-Unified-" + window
			if !strings.EqualFold(header.Get(prefix+"-Status"), "rejected") {
				continue
			}
			rejection.Kind = "quota"
			rejectedWindows++
			if window == "5h" {
				rejection.Window = "five_hour"
			} else {
				rejection.Window = "seven_day"
			}
			if window == "7d_oi" {
				rejection.Model = requestedModel
			}
			if seconds, err := strconv.ParseInt(header.Get(prefix+"-Reset"), 10, 64); err == nil {
				reset := time.Unix(seconds, 0)
				if reset.After(rejection.Until) {
					rejection.Until = reset
				}
			}
		}
		// When both shared limits reject, their latest known deadline wins.
		// A single-window name would otherwise be misleading on fresh reads.
		if rejectedWindows > 1 {
			rejection.Window = ""
		}
	}
	if seconds, ok := errorObject["resets_at"].(float64); ok && seconds > 0 {
		rejection.Until = time.Unix(int64(seconds), 0)
	}
	if rejection.Until.IsZero() {
		if seconds, ok := errorObject["resets_in_seconds"].(float64); ok && seconds > 0 {
			rejection.Until = time.Now().Add(time.Duration(seconds) * time.Second)
		}
	}
	if rejection.Until.IsZero() && header != nil {
		rejection.Until = retryAfter(header.Get("Retry-After"))
	}
	if rejection.Kind == "throughput" && rejection.Until.IsZero() {
		rejection.Until = time.Now().Add(2 * time.Second)
	}
	return rejection, rejection.Kind == "quota" || rejection.Kind == "auth"
}

func retryAfter(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	if n, err := time.ParseDuration(value + "s"); err == nil && n > 0 {
		return time.Now().Add(n)
	}
	t, _ := http.ParseTime(value)
	return t
}

func containsAny(value string, parts ...string) bool {
	for _, part := range parts {
		if strings.Contains(value, part) {
			return true
		}
	}
	return false
}
