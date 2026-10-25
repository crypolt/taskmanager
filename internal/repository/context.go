package repository

import (
	"context"
	"time"

	"taskmanager/models"
)

// AddContext appends one shared-context entry.
func (s *Store) AddContext(ctx context.Context, entry *models.ContextEntry) error {
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO context_entries
		(id, root_id, task_id, kind, title, body, tokens, created_at) VALUES (?,?,?,?,?,?,?,?)`,
		entry.ID, entry.RootID, entry.TaskID, entry.Kind, entry.Title, entry.Body, entry.Tokens,
		fmtTime(entry.CreatedAt))
	return err
}

// ListContext returns the shared context of a root task, oldest first.
func (s *Store) ListContext(ctx context.Context, rootID string, limit int) ([]models.ContextEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, root_id, task_id, kind, title, body, tokens, created_at
		FROM context_entries WHERE root_id = ? ORDER BY created_at ASC LIMIT ?`, rootID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.ContextEntry, 0, limit)
	for rows.Next() {
		var entry models.ContextEntry
		var created string
		if err := rows.Scan(&entry.ID, &entry.RootID, &entry.TaskID, &entry.Kind, &entry.Title,
			&entry.Body, &entry.Tokens, &created); err != nil {
			return nil, err
		}
		entry.CreatedAt = parseTime(created)
		out = append(out, entry)
	}
	return out, rows.Err()
}
