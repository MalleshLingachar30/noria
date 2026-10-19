package proxy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/amxv/noria/internal/core"
	"github.com/gorilla/websocket"
)

type wsFrame struct {
	connection *websocket.Conn
	kind       int
	body       []byte
	err        error
}
type pendingResponse struct {
	body           []byte
	model, session string
	lease          *core.Lease
	excluded       map[string]bool
	attempts       int
	committed      bool
	noRetry        bool
	warmup         bool
	held           []wsFrame
	heldBytes      int
}

func (s *Server) serveWebSocket(w http.ResponseWriter, r *http.Request, path string) {
	// A connection without native conversation metadata still represents its
	// own session. Never collapse unrelated connections into a shared empty key.
	r = r.Clone(r.Context())
	r.Header = r.Header.Clone()
	if r.Header.Get("X-Noria-Session") == "" {
		r.Header.Set("X-Noria-Session", randomSession())
	}
	upgrader := websocket.Upgrader{CheckOrigin: validOrigin, ReadBufferSize: 8192, WriteBufferSize: 8192}
	client, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		_ = client.Close()
		return
	}
	s.sockets[client] = struct{}{}
	s.mu.Unlock()
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer func() { s.mu.Lock(); delete(s.sockets, client); s.mu.Unlock(); _ = client.Close() }()
	client.SetReadLimit(maxBody)
	clientEvents := make(chan wsFrame, 1)
	upstreamEvents := make(chan wsFrame, 1)
	go readSocket(ctx, client, clientEvents)
	var upstream *websocket.Conn
	var accountID string
	var pending *pendingResponse
	var warmed *pendingResponse
	defer func() {
		if upstream != nil {
			_ = upstream.Close()
		}
		if pending != nil && pending.lease != nil {
			_ = s.scheduler.Complete(pending.lease)
		}
		if warmed != nil && warmed.lease != nil {
			_ = s.scheduler.Complete(warmed.lease)
		}
	}()
	writeError := func(message string) {
		payload, _ := json.Marshal(map[string]any{"type": "error", "error": map[string]string{"type": "noria_error", "message": message}})
		_ = writeSocket(client, websocket.TextMessage, payload)
	}
	start := func() bool {
		for pending.attempts < maxAttempts {
			pending.attempts++
			var lease *core.Lease
			var err error
			if warmed != nil {
				if warmed.session == pending.session && warmed.model == pending.model && pending.attempts == 1 {
					lease = warmed.lease
				} else {
					_ = s.scheduler.Complete(warmed.lease)
				}
				warmed = nil
			}
			if lease == nil {
				lease, err = s.scheduler.Acquire("codex", pending.session, pending.model, pending.excluded)
			}
			if err != nil {
				details, _ := unavailableDetails(err)
				payload, _ := json.Marshal(map[string]any{"type": "error", "error": details})
				_ = writeSocket(client, websocket.TextMessage, payload)
				return false
			}
			pending.lease = lease
			if upstream == nil || accountID != lease.Account.ID {
				if upstream != nil {
					_ = upstream.Close()
					upstream = nil
				}
				credential, err := s.backend.Credentials(ctx, lease.Account)
				if err != nil || credential.Token == "" {
					_ = s.scheduler.Complete(lease)
					pending.lease = nil
					pending.excluded[lease.Account.ID] = true
					continue
				}
				address, err := endpoint(lease.Account, path, r.URL.RawQuery)
				if err != nil {
					writeError("invalid upstream endpoint")
					return false
				}
				address = strings.Replace(address, "https://", "wss://", 1)
				address = strings.Replace(address, "http://", "ws://", 1)
				dialer := websocket.Dialer{HandshakeTimeout: time.Minute, ReadBufferSize: 8192, WriteBufferSize: 8192}
				conn, resp, err := dialer.DialContext(ctx, address, upstreamHeaders(r.Header, "codex", credential))
				if err != nil {
					if resp != nil {
						body, _ := io.ReadAll(io.LimitReader(resp.Body, maxPrelude))
						_ = resp.Body.Close()
						rejection, retry := classify(resp.StatusCode, resp.Header, body, pending.model)
						if retry {
							if s.reject(ctx, lease, rejection) != nil {
								writeError("cannot persist account rejection")
								return false
							}
							pending.lease = nil
							pending.excluded[lease.Account.ID] = true
							continue
						}
					}
					writeError("upstream WebSocket handshake failed; HTTP/SSE mode remains available")
					return false
				}
				upstream = conn
				accountID = lease.Account.ID
				conn.SetReadLimit(maxBody)
				go readSocket(ctx, conn, upstreamEvents)
			}
			if err := writeSocket(upstream, websocket.TextMessage, pending.body); err != nil {
				writeError("upstream WebSocket write failed")
				return false
			}
			return true
		}
		writeError("account retry limit reached")
		return false
	}
	for {
		select {
		case <-ctx.Done():
			return
		case frame := <-clientEvents:
			if frame.err != nil {
				return
			}
			if frame.kind != websocket.TextMessage {
				writeError("native JSON text messages required")
				return
			}
			var object map[string]json.RawMessage
			if json.Unmarshal(frame.body, &object) != nil || object == nil {
				writeError("JSON object required")
				return
			}
			var kind string
			_ = json.Unmarshal(object["type"], &kind)
			if kind == "response.create" {
				if pending != nil {
					writeError("one active response per connection is supported; use another connection for concurrent responses")
					return
				}
				pending = &pendingResponse{body: frame.body, model: modelName(object), session: sessionKey(r, object), excluded: map[string]bool{}}
				var generate *bool
				_ = json.Unmarshal(object["generate"], &generate)
				pending.warmup = generate != nil && !*generate
				if !start() {
					return
				}
			} else {
				if pending != nil {
					pending.noRetry = true
				}
				if upstream == nil {
					writeError("response.create is required before control messages")
					return
				}
				if err := writeSocket(upstream, frame.kind, frame.body); err != nil {
					return
				}
			}
		case frame := <-upstreamEvents:
			if frame.connection != upstream {
				continue
			}
			if frame.err != nil {
				if pending != nil {
					writeError("upstream connection closed; generation was not retried")
				}
				return
			}
			if pending == nil {
				if writeSocket(client, frame.kind, frame.body) != nil {
					return
				}
				continue
			}
			rejection, retry := classify(0, nil, frame.body, pending.model)
			if retry && !pending.committed && !pending.noRetry && pending.attempts < maxAttempts {
				lease := pending.lease
				if s.reject(ctx, lease, rejection) != nil {
					writeError("cannot persist account rejection")
					return
				}
				pending.lease = nil
				pending.excluded[lease.Account.ID] = true
				_ = upstream.Close()
				upstream = nil
				pending.held = nil
				pending.heldBytes = 0
				if !start() {
					return
				}
				continue
			}
			if !pending.committed {
				pending.held = append(pending.held, frame)
				pending.heldBytes += len(frame.body)
				if !meaningfulEvent(frame.body) && pending.heldBytes < maxPrelude {
					continue
				}
				if retry {
					_ = s.scheduler.Reject(pending.lease, rejection)
				} else if !pending.warmup {
					if err := s.scheduler.Accept(pending.lease); err != nil {
						writeError("cannot persist session affinity")
						return
					}
				}
				pending.committed = true
				for _, held := range pending.held {
					if writeSocket(client, held.kind, held.body) != nil {
						return
					}
				}
				pending.held = nil
			} else {
				if writeSocket(client, frame.kind, frame.body) != nil {
					return
				}
			}
			var event struct {
				Type string `json:"type"`
			}
			_ = json.Unmarshal(frame.body, &event)
			if event.Type == "response.completed" || event.Type == "response.failed" || event.Type == "response.incomplete" || event.Type == "response.cancelled" || event.Type == "error" {
				if pending.warmup && event.Type == "response.completed" {
					// Native prewarm is connection setup. Keep its reservation and
					// account for the next real create without recording usage or
					// predicting a subscription window from non-generating work.
					warmed = pending
				} else {
					_ = s.scheduler.Complete(pending.lease)
				}
				pending = nil
			}
		}
	}
}

func readSocket(ctx context.Context, connection *websocket.Conn, events chan<- wsFrame) {
	for {
		_ = connection.SetReadDeadline(time.Now().Add(30 * time.Minute))
		kind, body, err := connection.ReadMessage()
		select {
		case events <- wsFrame{connection: connection, kind: kind, body: body, err: err}:
		case <-ctx.Done():
			return
		}
		if err != nil {
			return
		}
	}
}

func writeSocket(connection *websocket.Conn, kind int, body []byte) error {
	_ = connection.SetWriteDeadline(time.Now().Add(time.Minute))
	return connection.WriteMessage(kind, body)
}
