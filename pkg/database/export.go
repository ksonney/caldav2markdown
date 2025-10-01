package database

import (
	"encoding/json"
	"time"

	ics "github.com/arran4/golang-ical"
)

// ToICSEvent converts a database event back to an iCalendar event
func (e *Event) ToICSEvent() *ics.VEvent {
	uid := e.UID
	if uid == "" {
		uid = "unknown"
	}
	event := ics.NewEvent(uid)

	if e.RecurrenceID != "" {
		event.SetProperty(ics.ComponentPropertyRecurrenceId, e.RecurrenceID)
	}

	if e.Summary != "" {
		event.SetProperty(ics.ComponentPropertySummary, e.Summary)
	}

	if e.Description != "" {
		event.SetProperty(ics.ComponentPropertyDescription, e.Description)
	}

	if e.Location != "" {
		event.SetProperty(ics.ComponentPropertyLocation, e.Location)
	}

	if e.Status != "" {
		event.SetProperty(ics.ComponentPropertyStatus, e.Status)
	}

	// Set start time
	if !e.StartTime.IsZero() {
		if e.AllDay {
			event.SetProperty(ics.ComponentPropertyDtStart, e.StartTime.Format("20060102"))
		} else {
			event.SetProperty(ics.ComponentPropertyDtStart, e.StartTime.Format("20060102T150405Z"))
		}
	}

	// Set end time
	if !e.EndTime.IsZero() {
		if e.AllDay {
			event.SetProperty(ics.ComponentPropertyDtEnd, e.EndTime.Format("20060102"))
		} else {
			event.SetProperty(ics.ComponentPropertyDtEnd, e.EndTime.Format("20060102T150405Z"))
		}
	}

	// Add categories
	if e.Categories != "" {
		var categories []string
		if err := json.Unmarshal([]byte(e.Categories), &categories); err == nil {
			for _, cat := range categories {
				event.SetProperty(ics.ComponentPropertyCategories, cat)
			}
		}
	}

	// Add calendar name as a custom property for reference
	if e.Calendar != "" {
		event.SetProperty(ics.ComponentProperty("X-CALENDAR-NAME"), e.Calendar)
	}

	return event
}

// ToICSTodo converts a database todo back to an iCalendar todo
func (t *Todo) ToICSTodo() *ics.VTodo {
	uid := t.UID
	if uid == "" {
		uid = "unknown"
	}
	todo := ics.NewTodo(uid)

	if t.Summary != "" {
		todo.SetProperty(ics.ComponentPropertySummary, t.Summary)
	}

	if t.Description != "" {
		todo.SetProperty(ics.ComponentPropertyDescription, t.Description)
	}

	if t.Status != "" {
		todo.SetProperty(ics.ComponentPropertyStatus, t.Status)
	}

	// Set due date
	if t.DueDate != nil && !t.DueDate.IsZero() {
		todo.SetProperty(ics.ComponentPropertyDue, t.DueDate.Format("20060102T150405Z"))
	}

	// Set start date
	if t.StartDate != nil && !t.StartDate.IsZero() {
		todo.SetProperty(ics.ComponentPropertyDtStart, t.StartDate.Format("20060102T150405Z"))
	}

	// Set priority
	if t.Priority > 0 {
		todo.SetProperty(ics.ComponentPropertyPriority, string(rune(t.Priority+'0')))
	}

	// Set completed status
	if t.Completed {
		todo.SetProperty(ics.ComponentPropertyCompleted, time.Now().Format("20060102T150405Z"))
	}

	// Add categories
	if t.Categories != "" {
		var categories []string
		if err := json.Unmarshal([]byte(t.Categories), &categories); err == nil {
			for _, cat := range categories {
				todo.SetProperty(ics.ComponentPropertyCategories, cat)
			}
		}
	}

	// Add calendar name as a custom property for reference
	if t.Calendar != "" {
		todo.SetProperty(ics.ComponentProperty("X-CALENDAR-NAME"), t.Calendar)
	}

	return todo
}

// GetEventsForMarkdown retrieves events from database within date range for markdown generation
func (db *DB) GetEventsForMarkdown(startDate, endDate time.Time) ([]*Event, error) {
	return db.GetEventsByDateRange(startDate, endDate)
}

// GetTodosForMarkdown retrieves todos from database for markdown generation
func (db *DB) GetTodosForMarkdown(startDate, endDate time.Time) ([]*Todo, error) {
	// Get todos with due dates in range
	todosInRange, err := db.GetTodosByDateRange(startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Get todos without due dates
	todosNoDue, err := db.GetTodosWithoutDueDate()
	if err != nil {
		return nil, err
	}

	// Combine both lists
	allTodos := append(todosInRange, todosNoDue...)
	return allTodos, nil
}
