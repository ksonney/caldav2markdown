package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// UpsertTodo inserts or updates a todo in the database
// Returns true if this is a new todo, false if it already existed
func (db *DB) UpsertTodo(todo *Todo) (bool, error) {
	now := time.Now()
	todo.LastSeen = now

	// Check if todo exists
	var exists bool
	var firstSeen time.Time
	err := db.conn.QueryRow(
		"SELECT 1, first_seen FROM todos WHERE uid = ? AND source_name = ?",
		todo.UID, todo.SourceName,
	).Scan(&exists, &firstSeen)

	isNew := err == sql.ErrNoRows

	if isNew {
		todo.FirstSeen = now
		_, err = db.conn.Exec(`
			INSERT INTO todos (
				uid, summary, description, due_date, start_date,
				completed, priority, status, categories, calendar,
				source_name, source_type, first_seen, last_seen
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			todo.UID, todo.Summary, todo.Description, todo.DueDate, todo.StartDate,
			todo.Completed, todo.Priority, todo.Status, todo.Categories, todo.Calendar,
			todo.SourceName, todo.SourceType, todo.FirstSeen, todo.LastSeen,
		)
	} else {
		todo.FirstSeen = firstSeen
		_, err = db.conn.Exec(`
			UPDATE todos SET
				summary = ?, description = ?, due_date = ?, start_date = ?,
				completed = ?, priority = ?, status = ?, categories = ?,
				calendar = ?, source_type = ?, last_seen = ?
			WHERE uid = ? AND source_name = ?`,
			todo.Summary, todo.Description, todo.DueDate, todo.StartDate,
			todo.Completed, todo.Priority, todo.Status, todo.Categories,
			todo.Calendar, todo.SourceType, todo.LastSeen,
			todo.UID, todo.SourceName,
		)
	}

	return isNew, err
}

// GetTodoByUID retrieves a todo by UID
func (db *DB) GetTodoByUID(uid, sourceName string) (*Todo, error) {
	todo := &Todo{}
	err := db.conn.QueryRow(`
		SELECT id, uid, summary, description, due_date, start_date,
			completed, priority, status, categories, calendar,
			source_name, source_type, first_seen, last_seen
		FROM todos
		WHERE uid = ? AND source_name = ?`,
		uid, sourceName,
	).Scan(
		&todo.ID, &todo.UID, &todo.Summary, &todo.Description, &todo.DueDate, &todo.StartDate,
		&todo.Completed, &todo.Priority, &todo.Status, &todo.Categories, &todo.Calendar,
		&todo.SourceName, &todo.SourceType, &todo.FirstSeen, &todo.LastSeen,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return todo, err
}

// GetAllTodos retrieves all todos
func (db *DB) GetAllTodos() ([]*Todo, error) {
	rows, err := db.conn.Query(`
		SELECT id, uid, summary, description, due_date, start_date,
			completed, priority, status, categories, calendar,
			source_name, source_type, first_seen, last_seen
		FROM todos
		ORDER BY due_date ASC NULLS LAST, priority DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []*Todo
	for rows.Next() {
		todo := &Todo{}
		err := rows.Scan(
			&todo.ID, &todo.UID, &todo.Summary, &todo.Description, &todo.DueDate, &todo.StartDate,
			&todo.Completed, &todo.Priority, &todo.Status, &todo.Categories, &todo.Calendar,
			&todo.SourceName, &todo.SourceType, &todo.FirstSeen, &todo.LastSeen,
		)
		if err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}
	return todos, rows.Err()
}

// GetTodosByDateRange retrieves todos with due dates in a date range
func (db *DB) GetTodosByDateRange(start, end time.Time) ([]*Todo, error) {
	rows, err := db.conn.Query(`
		SELECT id, uid,
			COALESCE(summary, ''),
			COALESCE(description, ''),
			due_date, start_date,
			completed, priority,
			COALESCE(status, ''),
			COALESCE(categories, ''),
			COALESCE(calendar, ''),
			COALESCE(source_name, ''),
			COALESCE(source_type, ''),
			first_seen, last_seen
		FROM todos
		WHERE due_date >= ? AND due_date <= ?
		ORDER BY due_date ASC, priority DESC`,
		start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []*Todo
	for rows.Next() {
		todo := &Todo{}
		err := rows.Scan(
			&todo.ID, &todo.UID, &todo.Summary, &todo.Description, &todo.DueDate, &todo.StartDate,
			&todo.Completed, &todo.Priority, &todo.Status, &todo.Categories, &todo.Calendar,
			&todo.SourceName, &todo.SourceType, &todo.FirstSeen, &todo.LastSeen,
		)
		if err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}
	return todos, rows.Err()
}

// GetTodosWithoutDueDate retrieves todos that don't have a due date
func (db *DB) GetTodosWithoutDueDate() ([]*Todo, error) {
	rows, err := db.conn.Query(`
		SELECT id, uid,
			COALESCE(summary, ''),
			COALESCE(description, ''),
			due_date, start_date,
			completed, priority,
			COALESCE(status, ''),
			COALESCE(categories, ''),
			COALESCE(calendar, ''),
			COALESCE(source_name, ''),
			COALESCE(source_type, ''),
			first_seen, last_seen
		FROM todos
		WHERE due_date IS NULL
		ORDER BY priority DESC, first_seen DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []*Todo
	for rows.Next() {
		todo := &Todo{}
		err := rows.Scan(
			&todo.ID, &todo.UID, &todo.Summary, &todo.Description, &todo.DueDate, &todo.StartDate,
			&todo.Completed, &todo.Priority, &todo.Status, &todo.Categories, &todo.Calendar,
			&todo.SourceName, &todo.SourceType, &todo.FirstSeen, &todo.LastSeen,
		)
		if err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}
	return todos, rows.Err()
}

// GetCompletedTodos retrieves completed todos
func (db *DB) GetCompletedTodos() ([]*Todo, error) {
	rows, err := db.conn.Query(`
		SELECT id, uid, summary, description, due_date, start_date,
			completed, priority, status, categories, calendar,
			source_name, source_type, first_seen, last_seen
		FROM todos
		WHERE completed = 1
		ORDER BY last_seen DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []*Todo
	for rows.Next() {
		todo := &Todo{}
		err := rows.Scan(
			&todo.ID, &todo.UID, &todo.Summary, &todo.Description, &todo.DueDate, &todo.StartDate,
			&todo.Completed, &todo.Priority, &todo.Status, &todo.Categories, &todo.Calendar,
			&todo.SourceName, &todo.SourceType, &todo.FirstSeen, &todo.LastSeen,
		)
		if err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}
	return todos, rows.Err()
}

// GetIncompleteTodos retrieves incomplete todos
func (db *DB) GetIncompleteTodos() ([]*Todo, error) {
	rows, err := db.conn.Query(`
		SELECT id, uid, summary, description, due_date, start_date,
			completed, priority, status, categories, calendar,
			source_name, source_type, first_seen, last_seen
		FROM todos
		WHERE completed = 0
		ORDER BY due_date ASC NULLS LAST, priority DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []*Todo
	for rows.Next() {
		todo := &Todo{}
		err := rows.Scan(
			&todo.ID, &todo.UID, &todo.Summary, &todo.Description, &todo.DueDate, &todo.StartDate,
			&todo.Completed, &todo.Priority, &todo.Status, &todo.Categories, &todo.Calendar,
			&todo.SourceName, &todo.SourceType, &todo.FirstSeen, &todo.LastSeen,
		)
		if err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}
	return todos, rows.Err()
}

// GetAllTodoUIDs returns all unique todo UIDs across all sources
func (db *DB) GetAllTodoUIDs() (map[string]bool, error) {
	rows, err := db.conn.Query("SELECT DISTINCT uid FROM todos")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	uids := make(map[string]bool)
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		uids[uid] = true
	}
	return uids, rows.Err()
}

// DeleteOldTodos removes todos not seen since the specified time
func (db *DB) DeleteOldTodos(since time.Time) (int64, error) {
	result, err := db.conn.Exec("DELETE FROM todos WHERE last_seen < ?", since)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetTodosByCalendar retrieves todos for a specific calendar
func (db *DB) GetTodosByCalendar(calendar string) ([]*Todo, error) {
	rows, err := db.conn.Query(`
		SELECT id, uid, summary, description, due_date, start_date,
			completed, priority, status, categories, calendar,
			source_name, source_type, first_seen, last_seen
		FROM todos
		WHERE calendar = ?
		ORDER BY due_date ASC NULLS LAST, priority DESC`,
		calendar,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []*Todo
	for rows.Next() {
		todo := &Todo{}
		err := rows.Scan(
			&todo.ID, &todo.UID, &todo.Summary, &todo.Description, &todo.DueDate, &todo.StartDate,
			&todo.Completed, &todo.Priority, &todo.Status, &todo.Categories, &todo.Calendar,
			&todo.SourceName, &todo.SourceType, &todo.FirstSeen, &todo.LastSeen,
		)
		if err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}
	return todos, rows.Err()
}

// GetTodoSourceStats returns statistics grouped by source
func (db *DB) GetTodoSourceStats() (map[string]int64, error) {
	rows, err := db.conn.Query(`
		SELECT source_name, COUNT(*) as count
		FROM todos
		GROUP BY source_name
		ORDER BY count DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int64)
	for rows.Next() {
		var source string
		var count int64
		if err := rows.Scan(&source, &count); err != nil {
			return nil, err
		}
		stats[source] = count
	}
	return stats, rows.Err()
}

// ExportTodosAsJSON exports todos as JSON for backup or analysis
func (db *DB) ExportTodosAsJSON() ([]byte, error) {
	todos, err := db.GetAllTodos()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(todos, "", "  ")
}

// IsDuplicateTodo checks if a todo with this UID exists
func (db *DB) IsDuplicateTodo(uid, sourceName string) (bool, error) {
	var exists bool
	err := db.conn.QueryRow(
		"SELECT 1 FROM todos WHERE uid = ? AND source_name = ?",
		uid, sourceName,
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return exists, err
}

// CountTodosBySource returns the number of todos per source
func (db *DB) CountTodosBySource() (map[string]int, error) {
	rows, err := db.conn.Query(`
		SELECT source_name, COUNT(*) as count
		FROM todos
		GROUP BY source_name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var source string
		var count int
		if err := rows.Scan(&source, &count); err != nil {
			return nil, err
		}
		counts[source] = count
	}
	return counts, rows.Err()
}

// SearchTodos searches todos by summary or description
func (db *DB) SearchTodos(query string) ([]*Todo, error) {
	searchPattern := fmt.Sprintf("%%%s%%", query)
	rows, err := db.conn.Query(`
		SELECT id, uid, summary, description, due_date, start_date,
			completed, priority, status, categories, calendar,
			source_name, source_type, first_seen, last_seen
		FROM todos
		WHERE summary LIKE ? OR description LIKE ?
		ORDER BY due_date ASC NULLS LAST`,
		searchPattern, searchPattern,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []*Todo
	for rows.Next() {
		todo := &Todo{}
		err := rows.Scan(
			&todo.ID, &todo.UID, &todo.Summary, &todo.Description, &todo.DueDate, &todo.StartDate,
			&todo.Completed, &todo.Priority, &todo.Status, &todo.Categories, &todo.Calendar,
			&todo.SourceName, &todo.SourceType, &todo.FirstSeen, &todo.LastSeen,
		)
		if err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}
	return todos, rows.Err()
}

// GetOverdueTodos returns todos that are past their due date and not completed
func (db *DB) GetOverdueTodos() ([]*Todo, error) {
	now := time.Now()
	rows, err := db.conn.Query(`
		SELECT id, uid, summary, description, due_date, start_date,
			completed, priority, status, categories, calendar,
			source_name, source_type, first_seen, last_seen
		FROM todos
		WHERE due_date < ? AND completed = 0
		ORDER BY due_date ASC`,
		now,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []*Todo
	for rows.Next() {
		todo := &Todo{}
		err := rows.Scan(
			&todo.ID, &todo.UID, &todo.Summary, &todo.Description, &todo.DueDate, &todo.StartDate,
			&todo.Completed, &todo.Priority, &todo.Status, &todo.Categories, &todo.Calendar,
			&todo.SourceName, &todo.SourceType, &todo.FirstSeen, &todo.LastSeen,
		)
		if err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}
	return todos, rows.Err()
}

// GetTodosByPriority retrieves todos filtered by priority level
func (db *DB) GetTodosByPriority(minPriority int) ([]*Todo, error) {
	rows, err := db.conn.Query(`
		SELECT id, uid, summary, description, due_date, start_date,
			completed, priority, status, categories, calendar,
			source_name, source_type, first_seen, last_seen
		FROM todos
		WHERE priority >= ?
		ORDER BY priority DESC, due_date ASC NULLS LAST`,
		minPriority,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []*Todo
	for rows.Next() {
		todo := &Todo{}
		err := rows.Scan(
			&todo.ID, &todo.UID, &todo.Summary, &todo.Description, &todo.DueDate, &todo.StartDate,
			&todo.Completed, &todo.Priority, &todo.Status, &todo.Categories, &todo.Calendar,
			&todo.SourceName, &todo.SourceType, &todo.FirstSeen, &todo.LastSeen,
		)
		if err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}
	return todos, rows.Err()
}
