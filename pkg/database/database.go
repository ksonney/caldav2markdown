package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQLite database connection
type DB struct {
	conn *sql.DB
	path string
}

// Event represents a calendar event in the database
type Event struct {
	ID            int64
	UID           string
	RecurrenceID  string // For recurring event instances
	Summary       string
	Description   string
	Location      string
	StartTime     time.Time
	EndTime       time.Time
	AllDay        bool
	Calendar      string
	Status        string
	Categories    string // JSON array
	SourceName    string
	SourceType    string // caldav, ics
	FirstSeen     time.Time
	LastSeen      time.Time
}

// Todo represents a task in the database
type Todo struct {
	ID          int64
	UID         string
	Summary     string
	Description string
	DueDate     *time.Time
	StartDate   *time.Time
	Completed   bool
	Priority    int
	Status      string
	Categories  string // JSON array
	Calendar    string
	SourceName  string
	SourceType  string // caldav, ics
	FirstSeen   time.Time
	LastSeen    time.Time
}

// Metadata represents processing metadata
type Metadata struct {
	Key       string
	Value     string
	UpdatedAt time.Time
}

// Open opens or creates the database at the specified path
func Open(path string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrent access
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := conn.Exec("PRAGMA foreign_keys=ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	db := &DB{conn: conn, path: path}

	// Initialize schema
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// initSchema creates the database schema if it doesn't exist
func (db *DB) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		uid TEXT NOT NULL,
		recurrence_id TEXT DEFAULT '',
		summary TEXT NOT NULL,
		description TEXT,
		location TEXT,
		start_time DATETIME NOT NULL,
		end_time DATETIME NOT NULL,
		all_day BOOLEAN NOT NULL DEFAULT 0,
		calendar TEXT,
		status TEXT,
		categories TEXT,
		source_name TEXT,
		source_type TEXT,
		first_seen DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_seen DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(uid, recurrence_id, source_name)
	);

	CREATE INDEX IF NOT EXISTS idx_events_uid ON events(uid);
	CREATE INDEX IF NOT EXISTS idx_events_start_time ON events(start_time);
	CREATE INDEX IF NOT EXISTS idx_events_source ON events(source_name);

	CREATE TABLE IF NOT EXISTS todos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		uid TEXT NOT NULL,
		summary TEXT NOT NULL,
		description TEXT,
		due_date DATETIME,
		start_date DATETIME,
		completed BOOLEAN NOT NULL DEFAULT 0,
		priority INTEGER DEFAULT 0,
		status TEXT,
		categories TEXT,
		calendar TEXT,
		source_name TEXT,
		source_type TEXT,
		first_seen DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_seen DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(uid, source_name)
	);

	CREATE INDEX IF NOT EXISTS idx_todos_uid ON todos(uid);
	CREATE INDEX IF NOT EXISTS idx_todos_due_date ON todos(due_date);
	CREATE INDEX IF NOT EXISTS idx_todos_source ON todos(source_name);

	CREATE TABLE IF NOT EXISTS metadata (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	INSERT OR IGNORE INTO schema_version (version) VALUES (1);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// GetPath returns the database file path
func (db *DB) GetPath() string {
	return db.path
}

// Clear removes all data from the database
func (db *DB) Clear() error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM events"); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM todos"); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM metadata"); err != nil {
		return err
	}

	return tx.Commit()
}

// GetStats returns database statistics
func (db *DB) GetStats() (map[string]int64, error) {
	stats := make(map[string]int64)

	var count int64
	if err := db.conn.QueryRow("SELECT COUNT(*) FROM events").Scan(&count); err != nil {
		return nil, err
	}
	stats["events"] = count

	if err := db.conn.QueryRow("SELECT COUNT(*) FROM todos").Scan(&count); err != nil {
		return nil, err
	}
	stats["todos"] = count

	if err := db.conn.QueryRow("SELECT COUNT(DISTINCT uid) FROM events").Scan(&count); err != nil {
		return nil, err
	}
	stats["unique_events"] = count

	if err := db.conn.QueryRow("SELECT COUNT(DISTINCT uid) FROM todos").Scan(&count); err != nil {
		return nil, err
	}
	stats["unique_todos"] = count

	return stats, nil
}

// SetMetadata stores a metadata key-value pair
func (db *DB) SetMetadata(key, value string) error {
	_, err := db.conn.Exec(
		"INSERT OR REPLACE INTO metadata (key, value, updated_at) VALUES (?, ?, ?)",
		key, value, time.Now(),
	)
	return err
}

// GetMetadata retrieves a metadata value
func (db *DB) GetMetadata(key string) (string, error) {
	var value string
	err := db.conn.QueryRow("SELECT value FROM metadata WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}
