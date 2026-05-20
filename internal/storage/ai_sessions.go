package storage

import "time"

// AIMessage represents a single persisted AI chat message.
type AIMessage struct {
	Role      string
	Content   string
	CreatedAt time.Time
}

// AppendAIMessage saves a new message to the ai_sessions table.
func (db *DB) AppendAIMessage(userID int64, role, content string) error {
	_, err := db.Exec(
		`INSERT INTO ai_sessions (user_id, role, content) VALUES (?, ?, ?)`,
		userID, role, content,
	)
	return err
}

// GetAIMessages returns the last `limit` messages for a user, ordered oldest first.
func (db *DB) GetAIMessages(userID int64, limit int) ([]AIMessage, error) {
	rows, err := db.Query(
		`SELECT role, content, created_at FROM (
			SELECT role, content, created_at FROM ai_sessions
			WHERE user_id = ?
			ORDER BY id DESC
			LIMIT ?
		) ORDER BY created_at ASC`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []AIMessage
	for rows.Next() {
		var m AIMessage
		if err := rows.Scan(&m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// DeleteAIMessages removes all messages for a user.
func (db *DB) DeleteAIMessages(userID int64) error {
	_, err := db.Exec(`DELETE FROM ai_sessions WHERE user_id = ?`, userID)
	return err
}

// DeleteExpiredAIMessages removes messages older than the given cutoff time.
func (db *DB) DeleteExpiredAIMessages(before time.Time) error {
	_, err := db.Exec(`DELETE FROM ai_sessions WHERE created_at < ?`, before)
	return err
}
