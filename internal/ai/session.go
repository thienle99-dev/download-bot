package ai

import (
	"context"
	"sync"
	"time"
)

const maxHistoryMessages = 20 // 10 user + 10 assistant

// Session holds the conversation history for a single user.
type Session struct {
	Messages  []Message
	UpdatedAt time.Time
}

// SessionManager manages per-user AI conversation sessions.
type SessionManager struct {
	sessions map[int64]*Session
	mu       sync.RWMutex
}

// NewSessionManager creates a new SessionManager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[int64]*Session),
	}
}

// Add appends a new message to the user's session history, trimming to keep the last maxHistoryMessages.
func (m *SessionManager) Add(userID int64, role, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess, ok := m.sessions[userID]
	if !ok {
		sess = &Session{}
		m.sessions[userID] = sess
	}

	sess.Messages = append(sess.Messages, Message{Role: role, Content: content})
	if len(sess.Messages) > maxHistoryMessages {
		// Keep the most recent maxHistoryMessages messages
		sess.Messages = sess.Messages[len(sess.Messages)-maxHistoryMessages:]
	}
	sess.UpdatedAt = time.Now()
}

// Get returns a copy of the user's conversation history.
func (m *SessionManager) Get(userID int64) []Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sess, ok := m.sessions[userID]
	if !ok {
		return nil
	}

	// Return a copy to avoid mutation from callers
	out := make([]Message, len(sess.Messages))
	copy(out, sess.Messages)
	return out
}

// Clear removes the user's session entirely.
func (m *SessionManager) Clear(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, userID)
}

// cleanExpired removes sessions that have not been updated within timeout.
func (m *SessionManager) cleanExpired(timeout time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-timeout)
	for id, sess := range m.sessions {
		if sess.UpdatedAt.Before(cutoff) {
			delete(m.sessions, id)
		}
	}
}

// CleanExpiredLoop runs a background goroutine to remove stale sessions on a regular interval.
func (m *SessionManager) CleanExpiredLoop(ctx context.Context, timeout time.Duration) {
	ticker := time.NewTicker(timeout / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.cleanExpired(timeout)
		}
	}
}
