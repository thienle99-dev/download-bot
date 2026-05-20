package bot

import (
	"context"
	"log"
	"sync"
	"time"

	"download-bot/internal/ai"
	"download-bot/internal/storage"
)

const defaultMaxHistory = 20 // 10 user + 10 assistant turns

// PersistentSessionManager is a drop-in replacement for ai.SessionManager that
// persists conversation history to SQLite so sessions survive bot restarts.
// It keeps an in-memory cache to avoid DB reads on every streaming token.
type PersistentSessionManager struct {
	db      *storage.DB
	cache   map[int64][]ai.Message
	mu      sync.RWMutex
	maxMsgs int
}

// NewPersistentSessionManager creates a manager backed by the given DB.
func NewPersistentSessionManager(db *storage.DB, maxMsgs int) *PersistentSessionManager {
	if maxMsgs <= 0 {
		maxMsgs = defaultMaxHistory
	}
	return &PersistentSessionManager{
		db:      db,
		cache:   make(map[int64][]ai.Message),
		maxMsgs: maxMsgs,
	}
}

// Add appends a message to the user's session, persisting it to the DB.
func (m *PersistentSessionManager) Add(userID int64, role, content string) {
	// Persist first so we never lose a message even if the process crashes.
	if err := m.db.AppendAIMessage(userID, role, content); err != nil {
		log.Printf("[AI session] failed to persist message for user %d: %v", userID, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	msgs := m.cache[userID]
	msgs = append(msgs, ai.Message{Role: role, Content: content})
	if len(msgs) > m.maxMsgs {
		msgs = msgs[len(msgs)-m.maxMsgs:]
	}
	m.cache[userID] = msgs
}

// Get returns the user's conversation history.
// On a cache miss (e.g. after restart) it loads from the DB.
func (m *PersistentSessionManager) Get(userID int64) []ai.Message {
	m.mu.RLock()
	cached, ok := m.cache[userID]
	m.mu.RUnlock()

	if ok {
		out := make([]ai.Message, len(cached))
		copy(out, cached)
		return out
	}

	// Cache miss — load from DB.
	rows, err := m.db.GetAIMessages(userID, m.maxMsgs)
	if err != nil {
		log.Printf("[AI session] failed to load messages for user %d: %v", userID, err)
		return nil
	}

	msgs := make([]ai.Message, len(rows))
	for i, r := range rows {
		msgs[i] = ai.Message{Role: r.Role, Content: r.Content}
	}

	m.mu.Lock()
	m.cache[userID] = msgs
	m.mu.Unlock()

	out := make([]ai.Message, len(msgs))
	copy(out, msgs)
	return out
}

// Clear removes the user's session from both the cache and the DB.
func (m *PersistentSessionManager) Clear(userID int64) {
	if err := m.db.DeleteAIMessages(userID); err != nil {
		log.Printf("[AI session] failed to delete messages for user %d: %v", userID, err)
	}

	m.mu.Lock()
	delete(m.cache, userID)
	m.mu.Unlock()
}

// CleanExpiredLoop runs a background goroutine that removes sessions older than
// timeout from the DB and evicts them from the in-memory cache.
func (m *PersistentSessionManager) CleanExpiredLoop(ctx context.Context, timeout time.Duration) {
	ticker := time.NewTicker(timeout / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-timeout)
			if err := m.db.DeleteExpiredAIMessages(cutoff); err != nil {
				log.Printf("[AI session] cleanup error: %v", err)
			}
			// Evict in-memory cache entries that are now stale.
			// We do a simple full-evict; they'll be reloaded from DB on next Get.
			m.mu.Lock()
			m.cache = make(map[int64][]ai.Message)
			m.mu.Unlock()
		}
	}
}
