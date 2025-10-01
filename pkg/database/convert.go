package database

import (
	"encoding/json"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
)

// EventFromICS converts an iCalendar event to a database event
func EventFromICS(event *ics.VEvent, sourceName, sourceType, calendarName string) *Event {
	if event == nil {
		return nil
	}

	uid := ""
	if uidProp := event.GetProperty(ics.ComponentPropertyUniqueId); uidProp != nil {
		uid = uidProp.Value
	}

	recurrenceID := ""
	if recIDProp := event.GetProperty(ics.ComponentPropertyRecurrenceId); recIDProp != nil {
		recurrenceID = recIDProp.Value
	}

	summary := ""
	if summaryProp := event.GetProperty(ics.ComponentPropertySummary); summaryProp != nil {
		summary = summaryProp.Value
	}

	description := ""
	if descProp := event.GetProperty(ics.ComponentPropertyDescription); descProp != nil {
		description = descProp.Value
	}

	location := ""
	if locProp := event.GetProperty(ics.ComponentPropertyLocation); locProp != nil {
		location = locProp.Value
	}

	status := ""
	if statusProp := event.GetProperty(ics.ComponentPropertyStatus); statusProp != nil {
		status = statusProp.Value
	}

	// Parse start time
	startTime := time.Time{}
	allDay := false
	if dtStart, err := event.GetStartAt(); err == nil {
		startTime = dtStart
		// Check if it's an all-day event (no time component)
		if dtStartProp := event.GetProperty(ics.ComponentPropertyDtStart); dtStartProp != nil {
			// All-day events typically have VALUE=DATE
			if valueParam := dtStartProp.ICalParameters["VALUE"]; len(valueParam) > 0 && valueParam[0] == "DATE" {
				allDay = true
			}
		}
	}

	// Parse end time
	endTime := time.Time{}
	if dtEnd, err := event.GetEndAt(); err == nil {
		endTime = dtEnd
	} else if !startTime.IsZero() {
		// Default to 1 hour duration if no end time
		endTime = startTime.Add(time.Hour)
	}

	// Get categories
	categories := []string{}
	for _, catProp := range event.Properties {
		if catProp.IANAToken == string(ics.ComponentPropertyCategories) {
			categories = append(categories, catProp.Value)
		}
	}
	categoriesJSON, _ := json.Marshal(categories)

	return &Event{
		UID:          uid,
		RecurrenceID: recurrenceID,
		Summary:      summary,
		Description:  description,
		Location:     location,
		StartTime:    startTime,
		EndTime:      endTime,
		AllDay:       allDay,
		Calendar:     calendarName,
		Status:       status,
		Categories:   string(categoriesJSON),
		SourceName:   sourceName,
		SourceType:   sourceType,
	}
}

// TodoFromICS converts an iCalendar todo to a database todo
func TodoFromICS(todo *ics.VTodo, sourceName, sourceType, calendarName string) *Todo {
	if todo == nil {
		return nil
	}

	uid := ""
	if uidProp := todo.GetProperty(ics.ComponentPropertyUniqueId); uidProp != nil {
		uid = uidProp.Value
	}

	summary := ""
	if summaryProp := todo.GetProperty(ics.ComponentPropertySummary); summaryProp != nil {
		summary = summaryProp.Value
	}

	description := ""
	if descProp := todo.GetProperty(ics.ComponentPropertyDescription); descProp != nil {
		description = descProp.Value
	}

	status := ""
	if statusProp := todo.GetProperty(ics.ComponentPropertyStatus); statusProp != nil {
		status = statusProp.Value
	}

	// Parse due date
	var dueDate *time.Time
	if dueProp := todo.GetProperty(ics.ComponentPropertyDue); dueProp != nil {
		if t, err := time.Parse("20060102T150405Z", dueProp.Value); err == nil {
			dueDate = &t
		} else if t, err := time.Parse("20060102", dueProp.Value); err == nil {
			dueDate = &t
		}
	}

	// Parse start date
	var startDate *time.Time
	if dtStartProp := todo.GetProperty(ics.ComponentPropertyDtStart); dtStartProp != nil {
		if t, err := time.Parse("20060102T150405Z", dtStartProp.Value); err == nil {
			startDate = &t
		} else if t, err := time.Parse("20060102", dtStartProp.Value); err == nil {
			startDate = &t
		}
	}

	// Check if completed
	completed := false
	if status != "" && (strings.ToUpper(status) == "COMPLETED" || strings.ToUpper(status) == "DONE") {
		completed = true
	}
	if completedProp := todo.GetProperty(ics.ComponentPropertyCompleted); completedProp != nil {
		completed = true
	}

	// Parse priority
	priority := 0
	if prioProp := todo.GetProperty(ics.ComponentPropertyPriority); prioProp != nil {
		if p, err := json.Number(prioProp.Value).Int64(); err == nil {
			priority = int(p)
		}
	}

	// Get categories
	categories := []string{}
	for _, catProp := range todo.Properties {
		if catProp.IANAToken == string(ics.ComponentPropertyCategories) {
			categories = append(categories, catProp.Value)
		}
	}
	categoriesJSON, _ := json.Marshal(categories)

	return &Todo{
		UID:         uid,
		Summary:     summary,
		Description: description,
		DueDate:     dueDate,
		StartDate:   startDate,
		Completed:   completed,
		Priority:    priority,
		Status:      status,
		Categories:  string(categoriesJSON),
		Calendar:    calendarName,
		SourceName:  sourceName,
		SourceType:  sourceType,
	}
}

// StoreEventIfNew stores an event in the database if it's new
// Returns true if the event was new, false if it already existed
func (db *DB) StoreEventIfNew(event *ics.VEvent, sourceName, sourceType, calendarName string) (bool, error) {
	if event == nil {
		return false, nil
	}

	dbEvent := EventFromICS(event, sourceName, sourceType, calendarName)
	if dbEvent == nil || dbEvent.UID == "" {
		return false, nil
	}

	return db.UpsertEvent(dbEvent)
}

// StoreEventWithChanges stores an event and returns what changed
// Returns: isNew, changes, error
func (db *DB) StoreEventWithChanges(event *ics.VEvent, sourceName, sourceType, calendarName string) (bool, *EventChanges, error) {
	if event == nil {
		return false, nil, nil
	}

	dbEvent := EventFromICS(event, sourceName, sourceType, calendarName)
	if dbEvent == nil || dbEvent.UID == "" {
		return false, nil, nil
	}

	return db.UpsertEventWithChanges(dbEvent)
}

// StoreTodoIfNew stores a todo in the database if it's new
// Returns true if the todo was new, false if it already existed
func (db *DB) StoreTodoIfNew(todo *ics.VTodo, sourceName, sourceType, calendarName string) (bool, error) {
	if todo == nil {
		return false, nil
	}

	dbTodo := TodoFromICS(todo, sourceName, sourceType, calendarName)
	if dbTodo == nil || dbTodo.UID == "" {
		return false, nil
	}

	return db.UpsertTodo(dbTodo)
}

// StoreTodoWithChanges stores a todo and returns what changed
// Returns: isNew, changes, error
func (db *DB) StoreTodoWithChanges(todo *ics.VTodo, sourceName, sourceType, calendarName string) (bool, *TodoChanges, error) {
	if todo == nil {
		return false, nil, nil
	}

	dbTodo := TodoFromICS(todo, sourceName, sourceType, calendarName)
	if dbTodo == nil || dbTodo.UID == "" {
		return false, nil, nil
	}

	return db.UpsertTodoWithChanges(dbTodo)
}
