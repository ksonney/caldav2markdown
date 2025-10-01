package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// UpsertEvent inserts or updates an event in the database
// Returns true if this is a new event, false if it already existed
func (db *DB) UpsertEvent(event *Event) (bool, error) {
	now := time.Now()
	event.LastSeen = now

	// Check if event exists
	var exists bool
	var firstSeen time.Time
	err := db.conn.QueryRow(
		"SELECT 1, first_seen FROM events WHERE uid = ? AND recurrence_id = ? AND source_name = ?",
		event.UID, event.RecurrenceID, event.SourceName,
	).Scan(&exists, &firstSeen)

	isNew := err == sql.ErrNoRows

	if isNew {
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
	} else {
		event.FirstSeen = firstSeen
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
	}

	return isNew, err
}

// GetEventByUID retrieves an event by UID and optional recurrence ID
func (db *DB) GetEventByUID(uid, recurrenceID, sourceName string) (*Event, error) {
	event := &Event{}
	err := db.conn.QueryRow(`
		SELECT id, uid, recurrence_id, summary, description, location,
			start_time, end_time, all_day, calendar, status, categories,
			source_name, source_type, first_seen, last_seen
		FROM events
		WHERE uid = ? AND recurrence_id = ? AND source_name = ?`,
		uid, recurrenceID, sourceName,
	).Scan(
		&event.ID, &event.UID, &event.RecurrenceID, &event.Summary, &event.Description, &event.Location,
		&event.StartTime, &event.EndTime, &event.AllDay, &event.Calendar, &event.Status, &event.Categories,
		&event.SourceName, &event.SourceType, &event.FirstSeen, &event.LastSeen,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return event, err
}

// GetEventsByDateRange retrieves events within a date range
func (db *DB) GetEventsByDateRange(start, end time.Time) ([]*Event, error) {
	rows, err := db.conn.Query(`
		SELECT id, uid,
			COALESCE(recurrence_id, ''),
			COALESCE(summary, ''),
			COALESCE(description, ''),
			COALESCE(location, ''),
			start_time, end_time, all_day,
			COALESCE(calendar, ''),
			COALESCE(status, ''),
			COALESCE(categories, ''),
			COALESCE(source_name, ''),
			COALESCE(source_type, ''),
			first_seen, last_seen
		FROM events
		WHERE start_time >= ? AND start_time <= ?
		ORDER BY start_time ASC`,
		start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		event := &Event{}
		err := rows.Scan(
			&event.ID, &event.UID, &event.RecurrenceID, &event.Summary, &event.Description, &event.Location,
			&event.StartTime, &event.EndTime, &event.AllDay, &event.Calendar, &event.Status, &event.Categories,
			&event.SourceName, &event.SourceType, &event.FirstSeen, &event.LastSeen,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

// GetAllEventUIDs returns all unique event UIDs across all sources
func (db *DB) GetAllEventUIDs() (map[string]bool, error) {
	rows, err := db.conn.Query("SELECT DISTINCT uid FROM events")
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

// GetEventInstancesByUID returns all instances of a recurring event
func (db *DB) GetEventInstancesByUID(uid string) ([]*Event, error) {
	rows, err := db.conn.Query(`
		SELECT id, uid, recurrence_id, summary, description, location,
			start_time, end_time, all_day, calendar, status, categories,
			source_name, source_type, first_seen, last_seen
		FROM events
		WHERE uid = ?
		ORDER BY start_time ASC`,
		uid,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		event := &Event{}
		err := rows.Scan(
			&event.ID, &event.UID, &event.RecurrenceID, &event.Summary, &event.Description, &event.Location,
			&event.StartTime, &event.EndTime, &event.AllDay, &event.Calendar, &event.Status, &event.Categories,
			&event.SourceName, &event.SourceType, &event.FirstSeen, &event.LastSeen,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

// DeleteOldEvents removes events not seen since the specified time
func (db *DB) DeleteOldEvents(since time.Time) (int64, error) {
	result, err := db.conn.Exec("DELETE FROM events WHERE last_seen < ?", since)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetEventsByCalendar retrieves events for a specific calendar
func (db *DB) GetEventsByCalendar(calendar string) ([]*Event, error) {
	rows, err := db.conn.Query(`
		SELECT id, uid, recurrence_id, summary, description, location,
			start_time, end_time, all_day, calendar, status, categories,
			source_name, source_type, first_seen, last_seen
		FROM events
		WHERE calendar = ?
		ORDER BY start_time ASC`,
		calendar,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		event := &Event{}
		err := rows.Scan(
			&event.ID, &event.UID, &event.RecurrenceID, &event.Summary, &event.Description, &event.Location,
			&event.StartTime, &event.EndTime, &event.AllDay, &event.Calendar, &event.Status, &event.Categories,
			&event.SourceName, &event.SourceType, &event.FirstSeen, &event.LastSeen,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

// GetEventSourceStats returns statistics grouped by source
func (db *DB) GetEventSourceStats() (map[string]int64, error) {
	rows, err := db.conn.Query(`
		SELECT source_name, COUNT(*) as count
		FROM events
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

// ExportEventsAsJSON exports events as JSON for backup or analysis
func (db *DB) ExportEventsAsJSON(start, end time.Time) ([]byte, error) {
	events, err := db.GetEventsByDateRange(start, end)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(events, "", "  ")
}

// IsDuplicateEvent checks if an event with this UID and recurrence ID exists
func (db *DB) IsDuplicateEvent(uid, recurrenceID, sourceName string) (bool, error) {
	var exists bool
	err := db.conn.QueryRow(
		"SELECT 1 FROM events WHERE uid = ? AND recurrence_id = ? AND source_name = ?",
		uid, recurrenceID, sourceName,
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return exists, err
}

// CountEventsBySource returns the number of events per source
func (db *DB) CountEventsBySource() (map[string]int, error) {
	rows, err := db.conn.Query(`
		SELECT source_name, COUNT(*) as count
		FROM events
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

// VacuumEvents optimizes the events table
func (db *DB) VacuumEvents() error {
	_, err := db.conn.Exec("VACUUM")
	return err
}

// GetRecentEvents returns the most recently added events
func (db *DB) GetRecentEvents(limit int) ([]*Event, error) {
	rows, err := db.conn.Query(`
		SELECT id, uid, recurrence_id, summary, description, location,
			start_time, end_time, all_day, calendar, status, categories,
			source_name, source_type, first_seen, last_seen
		FROM events
		ORDER BY first_seen DESC
		LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		event := &Event{}
		err := rows.Scan(
			&event.ID, &event.UID, &event.RecurrenceID, &event.Summary, &event.Description, &event.Location,
			&event.StartTime, &event.EndTime, &event.AllDay, &event.Calendar, &event.Status, &event.Categories,
			&event.SourceName, &event.SourceType, &event.FirstSeen, &event.LastSeen,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

// SearchEvents searches events by summary or description
func (db *DB) SearchEvents(query string) ([]*Event, error) {
	searchPattern := fmt.Sprintf("%%%s%%", query)
	rows, err := db.conn.Query(`
		SELECT id, uid, recurrence_id, summary, description, location,
			start_time, end_time, all_day, calendar, status, categories,
			source_name, source_type, first_seen, last_seen
		FROM events
		WHERE summary LIKE ? OR description LIKE ?
		ORDER BY start_time DESC`,
		searchPattern, searchPattern,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		event := &Event{}
		err := rows.Scan(
			&event.ID, &event.UID, &event.RecurrenceID, &event.Summary, &event.Description, &event.Location,
			&event.StartTime, &event.EndTime, &event.AllDay, &event.Calendar, &event.Status, &event.Categories,
			&event.SourceName, &event.SourceType, &event.FirstSeen, &event.LastSeen,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}
