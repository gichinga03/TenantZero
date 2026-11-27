// Package store handles all PostgreSQL persistence for TenantZero.
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "github.com/lib/pq"
)

// UserRow is a row from the users table.
type UserRow struct {
	ID           string
	Username     string
	PasswordHash string
}

// SessionRow is a row from the sessions table.
// State holds the JSON-encoded schema.SessionState.
type SessionRow struct {
	ID         string
	UserID     string
	LabID      string
	State      []byte
	StartedAt  time.Time
	LastActive time.Time
	Completed  bool
}

// JournalRow is a row from the journal table.
type JournalRow struct {
	SessionID  string
	Sequence   int
	Input      string
	Output     string
	Machine    string
	Username   string
	RecordedAt time.Time
}

// Store wraps a *sql.DB and exposes all persistence operations.
type Store struct {
	db *sql.DB
}

// New opens a PostgreSQL connection and verifies it with a ping.
func New(dsn string) (*Store, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("store.New: open: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("store.New: ping: %w", err)
	}
	return &Store{db: db}, nil
}

// Migrate runs all *.sql files in migrationsDir in lexicographic order.
// Files already applied are not tracked — use idempotent IF NOT EXISTS DDL.
func (s *Store) Migrate(migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("store.Migrate: readdir: %w", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".sql" {
			files = append(files, filepath.Join(migrationsDir, e.Name()))
		}
	}
	sort.Strings(files)
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("store.Migrate: read %s: %w", f, err)
		}
		if _, err := s.db.Exec(string(data)); err != nil {
			return fmt.Errorf("store.Migrate: exec %s: %w", f, err)
		}
	}
	return nil
}

// CreateUser inserts a new user row and returns the generated UUID.
func (s *Store) CreateUser(username, passwordHash string) (string, error) {
	var id string
	err := s.db.QueryRow(
		`INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id`,
		username, passwordHash,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("store.CreateUser: %w", err)
	}
	return id, nil
}

// GetUserByUsername fetches a user row by username.
// Returns sql.ErrNoRows if not found.
func (s *Store) GetUserByUsername(username string) (UserRow, error) {
	var u UserRow
	err := s.db.QueryRow(
		`SELECT id, username, password_hash FROM users WHERE username = $1`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash)
	if err != nil {
		return UserRow{}, fmt.Errorf("store.GetUserByUsername: %w", err)
	}
	return u, nil
}

// SaveSession upserts a session row keyed on id.
func (s *Store) SaveSession(sess SessionRow) error {
	// sess.State is already marshalled JSON bytes — pass directly to avoid double-encoding.
	_, err := s.db.Exec(`
		INSERT INTO sessions (id, user_id, lab_id, state, started_at, last_active, completed)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			state       = EXCLUDED.state,
			last_active = EXCLUDED.last_active,
			completed   = EXCLUDED.completed
	`, sess.ID, sess.UserID, sess.LabID, sess.State, sess.StartedAt, sess.LastActive, sess.Completed)
	if err != nil {
		return fmt.Errorf("store.SaveSession: %w", err)
	}
	return nil
}

// LoadSession fetches a session row by id.
// Returns sql.ErrNoRows if not found.
func (s *Store) LoadSession(id string) (SessionRow, error) {
	var sess SessionRow
	var stateJSON []byte
	err := s.db.QueryRow(`
		SELECT id, user_id, lab_id, state, started_at, last_active, completed
		FROM sessions WHERE id = $1
	`, id).Scan(
		&sess.ID, &sess.UserID, &sess.LabID, &stateJSON,
		&sess.StartedAt, &sess.LastActive, &sess.Completed,
	)
	if err != nil {
		return SessionRow{}, fmt.Errorf("store.LoadSession: %w", err)
	}
	sess.State = stateJSON
	return sess, nil
}

// MarkComplete sets completed = true for the given session id.
func (s *Store) MarkComplete(id string) error {
	_, err := s.db.Exec(`UPDATE sessions SET completed = TRUE WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("store.MarkComplete: %w", err)
	}
	return nil
}

// AppendJournal inserts a journal row.
func (s *Store) AppendJournal(entry JournalRow) error {
	_, err := s.db.Exec(`
		INSERT INTO journal (session_id, sequence, input, output, machine, username, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, entry.SessionID, entry.Sequence, entry.Input, entry.Output,
		entry.Machine, entry.Username, entry.RecordedAt)
	if err != nil {
		return fmt.Errorf("store.AppendJournal: %w", err)
	}
	return nil
}

// GetJournal returns all journal entries for a session in sequence order.
func (s *Store) GetJournal(sessionID string) ([]JournalRow, error) {
	rows, err := s.db.Query(`
		SELECT session_id, sequence, input, output, machine, username, recorded_at
		FROM journal WHERE session_id = $1 ORDER BY sequence ASC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("store.GetJournal: %w", err)
	}
	defer rows.Close()
	var entries []JournalRow
	for rows.Next() {
		var e JournalRow
		if err := rows.Scan(&e.SessionID, &e.Sequence, &e.Input, &e.Output,
			&e.Machine, &e.Username, &e.RecordedAt); err != nil {
			return nil, fmt.Errorf("store.GetJournal: scan: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}