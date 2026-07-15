package main

import (
	"sync"
	"time"
)

type auditEvent struct {
	TS         string `json:"ts"`
	Source     string `json:"source"`
	AuthID     string `json:"auth_id,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
	Action     string `json:"action,omitempty"`
	Result     string `json:"result,omitempty"`
	Message    string `json:"message,omitempty"`
}

type auditLog struct {
	mu     sync.Mutex
	max    int
	events []auditEvent
}

func newAuditLog(max int) *auditLog {
	if max <= 0 {
		max = 200
	}
	return &auditLog{max: max, events: make([]auditEvent, 0, max)}
}

func (a *auditLog) setMax(max int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if max <= 0 {
		max = 200
	}
	a.max = max
	if len(a.events) > max {
		a.events = a.events[len(a.events)-max:]
	}
}

func (a *auditLog) add(source, authID, action, result, message string, statusCode int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ev := auditEvent{
		TS:         time.Now().Format(time.RFC3339),
		Source:     source,
		AuthID:     truncateID(authID),
		StatusCode: statusCode,
		Action:     action,
		Result:     result,
		Message:    message,
	}
	a.events = append(a.events, ev)
	if len(a.events) > a.max {
		a.events = a.events[len(a.events)-a.max:]
	}
}

func (a *auditLog) list() []auditEvent {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]auditEvent, len(a.events))
	copy(out, a.events)
	return out
}

func truncateID(id string) string {
	if len(id) <= 48 {
		return id
	}
	return id[:24] + "..." + id[len(id)-12:]
}
