package mcpapi

import (
	"encoding/json"
	"errors"
	"net/url"
	"sync"

	"github.com/google/uuid"
)

var (
	ErrSSESessionNotFound     = errors.New("sse session not found")
	ErrSSESessionBackpressure = errors.New("sse session event queue is full")
)

type SSEEvent struct {
	ID   string
	Type string
	Data []byte
}

type SSESession struct {
	ID       string
	Endpoint string
	Events   <-chan SSEEvent

	events chan SSEEvent
}

type SSEBroker struct {
	service  Service
	mu       sync.RWMutex
	sessions map[string]*SSESession
}

func NewSSEBroker(service Service) *SSEBroker {
	return &SSEBroker{
		service:  service,
		sessions: make(map[string]*SSESession),
	}
}

func (b *SSEBroker) OpenSession() *SSESession {
	id := uuid.NewString()
	events := make(chan SSEEvent, 32)
	session := &SSESession{
		ID:       id,
		Endpoint: "/mcp/message?session_id=" + url.QueryEscape(id),
		Events:   events,
		events:   events,
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.sessions[id] = session
	return session
}

func (b *SSEBroker) CloseSession(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	session, ok := b.sessions[id]
	if !ok {
		return
	}
	delete(b.sessions, id)
	close(session.events)
}

func (b *SSEBroker) Dispatch(sessionID string, req JSONRPCRequest) error {
	if req.ID == nil {
		return nil
	}

	response := b.service.HandleJSONRPC(req)
	data, err := json.Marshal(response)
	if err != nil {
		return err
	}

	b.mu.RLock()
	defer b.mu.RUnlock()
	session, ok := b.sessions[sessionID]
	if !ok {
		return ErrSSESessionNotFound
	}

	select {
	case session.events <- SSEEvent{Type: "message", Data: data}:
		return nil
	default:
		return ErrSSESessionBackpressure
	}
}
