package database

import (
	"fmt"
	"strings"
	"time"
)

// EventChanges tracks what changed in an event update
type EventChanges struct {
	Changed      bool
	ChangedFields []string
}

// TodoChanges tracks what changed in a todo update
type TodoChanges struct {
	Changed      bool
	ChangedFields []string
}

// UpsertEventWithChanges inserts or updates an event and returns what changed
// Returns: isNew, changes, error
func (db *DB) UpsertEventWithChanges(event *Event) (bool, *EventChanges, error) {
	now := time.Now()
	event.LastSeen = now

	// Get existing event if it exists
	existing, err := db.GetEventByUID(event.UID, event.RecurrenceID, event.SourceName)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return false, nil, err
	}

	isNew := existing == nil
	changes := &EventChanges{Changed: false, ChangedFields: []string{}}

	if isNew {
		// New event
		event.FirstSeen = now
		_, err = db.conn.Exec(`
			INSERT INTO events (
				uid, recurrence_id, summary, description, location,
				start_time, end_time, all_day, calendar, status, categories,
				source_name, source_type, first_seen, last_seen
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			event.UID, event.RecurrenceID, event.Summary, event.Description, event.Location,
			event.StartTime, event.EndTime, event.AllDay, event.Calendar, event.Status, event.Categories,
			event.SourceName, event.SourceType, event.FirstSeen, event.LastSeen,
		)
		return isNew, changes, err
	}

	// Compare existing vs new
	event.FirstSeen = existing.FirstSeen

	if existing.Summary != event.Summary {
		changes.Changed = true
		changes.ChangedFields = append(changes.ChangedFields, "summary")
	}
	if existing.Description != event.Description {
		changes.Changed = true
		changes.ChangedFields = append(changes.ChangedFields, "description")
	}
	if existing.Location != event.Location {
		changes.Changed = true
		changes.ChangedFields = append(changes.ChangedFields, "location")
	}
	if !existing.StartTime.Equal(event.StartTime) {
		changes.Changed = true
		changes.ChangedFields = append(changes.ChangedFields, "start_time")
	}
	if !existing.EndTime.Equal(event.EndTime) {
		changes.Changed = true
		changes.ChangedFields = append(changes.ChangedFields, "end_time")
	}
	if existing.AllDay != event.AllDay {
		changes.Changed = true
		changes.ChangedFields = append(changes.ChangedFields, "all_day")
	}
	if existing.Calendar != event.Calendar {
		changes.Changed = true
		changes.ChangedFields = append(changes.ChangedFields, "calendar")
	}
	if existing.Status != event.Status {
		changes.Changed = true
		changes.ChangedFields = append(changes.ChangedFields, "status")
	}
	if existing.Categories != event.Categories {
		changes.Changed = true
		changes.ChangedFields = append(changes.ChangedFields, "categories")
	}

	// Update the event
	_, err = db.conn.Exec(`
		UPDATE events SET
			summary = ?, description = ?, location = ?,
			start_time = ?, end_time = ?, all_day = ?,
			calendar = ?, status = ?, categories = ?,
			source_type = ?, last_seen = ?
		WHERE uid = ? AND recurrence_id = ? AND source_name = ?`,
		event.Summary, event.Description, event.Location,
		event.StartTime, event.EndTime, event.AllDay,
		event.Calendar, event.Status, event.Categories,
		event.SourceType, event.LastSeen,
		event.UID, event.RecurrenceID, event.SourceName,
	)

	return isNew, changes, err
}

// UpsertTodoWithChanges inserts or updates a todo and returns what changed
// Returns: isNew, changes, error
func (db *DB) UpsertTodoWithChanges(todo *Todo) (bool, *TodoChanges, error) {
	now := time.Now()
	todo.LastSeen = now

	// Get existing todo if it exists
	existing, err := db.GetTodoByUID(todo.UID, todo.SourceName)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return false, nil, err
	}

	isNew := existing == nil
	changes := &TodoChanges{Changed: false, ChangedFields: []string{}}

	if isNew {
		// New todo
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
		return isNew, changes, err
	}

	// Compare existing vs new
	todo.FirstSeen = existing.FirstSeen

	if existing.Summary != todo.Summary {
		changes.Changed = true
		changes.ChangedFields = append(changes.ChangedFields, "summary")
	}
	if existing.Description != todo.Description {
		changes.Changed = true
		changes.ChangedFields = append(changes.ChangedFields, "description")
	}

	// Handle nullable time fields
	dueDateChanged := false
	if existing.DueDate == nil && todo.DueDate != nil {
		dueDateChanged = true
	} else if existing.DueDate != nil && todo.DueDate == nil {
		dueDateChanged = true
	} else if existing.DueDate != nil && todo.DueDate != nil && !existing.DueDate.Equal(*todo.DueDate) {
		dueDateChanged = true
	}
	if dueDateChanged {
		changes.Changed = true
		changes.ChangedFields = append(changes.ChangedFields, "due_date")
	}

	startDateChanged := false
	if existing.StartDate == nil && todo.StartDate != nil {
		startDateChanged = true
	} else if existing.StartDate != nil && todo.StartDate == nil {
		startDateChanged = true
	} else if existing.StartDate != nil && todo.StartDate != nil && !existing.StartDate.Equal(*todo.StartDate) {
		startDateChanged = true
	}
	if startDateChanged {
		changes.Changed = true
		changes.ChangedFields = append(changes.ChangedFields, "start_date")
	}

	if existing.Completed != todo.Completed {
		changes.Changed = true
		changes.ChangedFields = append(changes.ChangedFields, "completed")
	}
	if existing.Priority != todo.Priority {
		changes.Changed = true
		changes.ChangedFields = append(changes.ChangedFields, "priority")
	}
	if existing.Status != todo.Status {
		changes.Changed = true
		changes.ChangedFields = append(changes.ChangedFields, "status")
	}
	if existing.Categories != todo.Categories {
		changes.Changed = true
		changes.ChangedFields = append(changes.ChangedFields, "categories")
	}
	if existing.Calendar != todo.Calendar {
		changes.Changed = true
		changes.ChangedFields = append(changes.ChangedFields, "calendar")
	}

	// Update the todo
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

	return isNew, changes, err
}

// FormatChanges returns a human-readable description of changes
func (ec *EventChanges) FormatChanges() string {
	if !ec.Changed || len(ec.ChangedFields) == 0 {
		return "no changes"
	}
	return fmt.Sprintf("changed: %s", strings.Join(ec.ChangedFields, ", "))
}

// FormatChanges returns a human-readable description of changes
func (tc *TodoChanges) FormatChanges() string {
	if !tc.Changed || len(tc.ChangedFields) == 0 {
		return "no changes"
	}
	return fmt.Sprintf("changed: %s", strings.Join(tc.ChangedFields, ", "))
}
