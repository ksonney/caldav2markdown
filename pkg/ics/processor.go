package ics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/arran4/golang-ical"
	"caldav2markdown/pkg/rrule"
)

// EventData represents processed event data similar to CalDAV's DeduplicationResult
type EventData struct {
	Events            []*ics.VEvent
	Todos             []*ics.VTodo
	DuplicatesFound   int
	ProcessingReport  string
	SourceName        string // Name of the ICS source
}

// ProgressCallback is used for progress reporting during processing
type ProgressCallback func(message string, current, total int)

// Processor handles a single ICS source and extracts calendar data
type Processor struct {
	source    Source
	startDate time.Time
	endDate   time.Time
}

// NewProcessor creates a new ICS processor for a single source
func NewProcessor(source Source, startDate, endDate time.Time) *Processor {
	return &Processor{
		source:    source,
		startDate: startDate,
		endDate:   endDate,
	}
}

// ProcessSource reads from the ICS source and returns event data
func (p *Processor) ProcessSource(ctx context.Context, progressCallback ProgressCallback) (*EventData, error) {
	reader, err := NewICSReader(p.source)
	if err != nil {
		return nil, fmt.Errorf("failed to create ICS reader: %w", err)
	}

	if progressCallback != nil {
		progressCallback(fmt.Sprintf("Reading from %s", p.source.Name), 0, 1)
	}

	// Read the calendar
	calendar, err := reader.ReadICS(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read ICS source: %w", err)
	}

	if progressCallback != nil {
		progressCallback("Processing calendar data...", 1, 1)
	}

	// Process calendar and extract events/todos
	return p.processCalendar(calendar, progressCallback)
}

// processCalendar extracts events and todos from a single calendar
func (p *Processor) processCalendar(calendar *ics.Calendar, progressCallback ProgressCallback) (*EventData, error) {
	var allEvents []*ics.VEvent
	var allTodos []*ics.VTodo
	seenUIDs := make(map[string]bool)
	duplicatesFound := 0

	totalItems := len(calendar.Events()) + len(calendar.Todos())
	processedItems := 0

	if progressCallback != nil {
		progressCallback("Processing calendar events and tasks", 0, totalItems)
	}

	// Process events
	for _, event := range calendar.Events() {
		processedItems++
		if processedItems%10 == 0 && progressCallback != nil {
			progressCallback(fmt.Sprintf("Processed %d/%d items", processedItems, totalItems), processedItems, totalItems)
		}

		uid := p.getEventUID(event)
		if uid == "" {
			continue // Skip events without UID
		}

		// Check for duplicates
		if seenUIDs[uid] {
			duplicatesFound++
			continue
		}

		// Always expand recurring events first, then filter the instances
		expandedEvents := p.expandRecurringEvent(event)
		for _, expandedEvent := range expandedEvents {
			// Filter expanded instances based on their individual dates
			if !p.isEventInDateRange(expandedEvent) {
				continue
			}

			expandedUID := p.getEventUID(expandedEvent)
			if !seenUIDs[expandedUID] {
				seenUIDs[expandedUID] = true
				allEvents = append(allEvents, expandedEvent)
			} else {
				duplicatesFound++
			}
		}

		seenUIDs[uid] = true
	}

	// Process todos
	for _, todo := range calendar.Todos() {
		processedItems++
		if processedItems%10 == 0 && progressCallback != nil {
			progressCallback(fmt.Sprintf("Processed %d/%d items", processedItems, totalItems), processedItems, totalItems)
		}

		uid := p.getTodoUID(todo)
		if uid == "" {
			continue // Skip todos without UID
		}

		// Check for duplicates
		if seenUIDs[uid] {
			duplicatesFound++
			continue
		}

		// Always expand recurring todos first, then filter the instances
		expandedTodos := p.expandRecurringTodo(todo)
		for _, expandedTodo := range expandedTodos {
			// Filter expanded instances based on their individual dates
			if !p.isTodoInDateRange(expandedTodo) {
				continue
			}

			expandedUID := p.getTodoUID(expandedTodo)
			if !seenUIDs[expandedUID] {
				seenUIDs[expandedUID] = true
				allTodos = append(allTodos, expandedTodo)
			} else {
				duplicatesFound++
			}
		}

		seenUIDs[uid] = true
	}

	if progressCallback != nil {
		progressCallback(fmt.Sprintf("Completed processing %d events and %d tasks", len(allEvents), len(allTodos)), totalItems, totalItems)
	}

	// Generate processing report
	reader, _ := NewICSReader(p.source)
	sourceInfo := reader.GetSourceInfo()

	report := fmt.Sprintf("Processed ICS source:\n%s\nFound %d events, %d tasks (%d duplicates removed)",
		sourceInfo, len(allEvents), len(allTodos), duplicatesFound)

	return &EventData{
		Events:           allEvents,
		Todos:            allTodos,
		DuplicatesFound:  duplicatesFound,
		ProcessingReport: report,
		SourceName:       p.source.Name,
	}, nil
}

// getEventUID extracts UID from an event (similar to CalDAV client)
func (p *Processor) getEventUID(event *ics.VEvent) string {
	if uid := event.GetProperty(ics.ComponentPropertyUniqueId); uid != nil {
		return uid.Value
	}
	return ""
}

// getTodoUID extracts UID from a todo (similar to CalDAV client)
func (p *Processor) getTodoUID(todo *ics.VTodo) string {
	if uid := todo.GetProperty(ics.ComponentPropertyUniqueId); uid != nil {
		return uid.Value
	}
	return ""
}

// isEventInDateRange checks if an event falls within the configured date range
func (p *Processor) isEventInDateRange(event *ics.VEvent) bool {
	dtstart := event.GetProperty(ics.ComponentPropertyDtStart)
	if dtstart == nil {
		return true // Include events without start date
	}

	// Extract TZID parameter if present
	var tzid string
	if tzids, exists := dtstart.ICalParameters["TZID"]; exists && len(tzids) > 0 {
		tzid = tzids[0]
	}

	startTime, _, err := parseICalDateTimeWithTZ(dtstart.Value, tzid)
	if err != nil {
		return true // Include events with unparseable dates
	}

	// If zero time (placeholder date), always include
	if startTime.IsZero() {
		return true
	}

	return !startTime.Before(p.startDate) && !startTime.After(p.endDate)
}

// isTodoInDateRange checks if a todo falls within the configured date range (based on due date)
func (p *Processor) isTodoInDateRange(todo *ics.VTodo) bool {
	due := todo.GetProperty(ics.ComponentPropertyDue)
	if due == nil {
		return true // Include todos without due date
	}

	// Extract TZID parameter if present
	var tzid string
	if tzids, exists := due.ICalParameters["TZID"]; exists && len(tzids) > 0 {
		tzid = tzids[0]
	}

	dueTime, _, err := parseICalDateTimeWithTZ(due.Value, tzid)
	if err != nil {
		return true // Include todos with unparseable dates
	}

	// If zero time (placeholder date), always include
	if dueTime.IsZero() {
		return true
	}

	return !dueTime.Before(p.startDate) && !dueTime.After(p.endDate)
}

// expandRecurringEvent handles recurring events using RRULE (similar to CalDAV client)
func (p *Processor) expandRecurringEvent(event *ics.VEvent) []*ics.VEvent {
	rruleProp := event.GetProperty(ics.ComponentPropertyRrule)
	if rruleProp == nil {
		return []*ics.VEvent{event} // Not recurring, return as-is
	}

	dtstart := event.GetProperty(ics.ComponentPropertyDtStart)
	if dtstart == nil {
		return []*ics.VEvent{event} // No start date, return as-is
	}

	// Extract TZID parameter if present
	var tzid string
	if tzids, exists := dtstart.ICalParameters["TZID"]; exists && len(tzids) > 0 {
		tzid = tzids[0]
	}

	_, _, err := parseICalDateTimeWithTZ(dtstart.Value, tzid)
	if err != nil {
		return []*ics.VEvent{event} // Unparseable start date, return as-is
	}

	// Use the rrule package to expand the recurring event
	maxOccurrences := 1000 // Reasonable limit to prevent infinite loops
	expandedEvents, err := rrule.ExpandEvent(event, maxOccurrences, p.startDate, p.endDate)
	if err != nil {
		// If we can't parse the RRULE, return the original event
		return []*ics.VEvent{event}
	}

	return expandedEvents
}

// expandRecurringTodo handles recurring todos using RRULE (similar to CalDAV client)
func (p *Processor) expandRecurringTodo(todo *ics.VTodo) []*ics.VTodo {
	rruleProp := todo.GetProperty(ics.ComponentPropertyRrule)
	if rruleProp == nil {
		return []*ics.VTodo{todo} // Not recurring, return as-is
	}

	due := todo.GetProperty(ics.ComponentPropertyDue)
	dtstart := todo.GetProperty(ics.ComponentPropertyDtStart)
	if due == nil && dtstart == nil {
		return []*ics.VTodo{todo} // No date information, return as-is
	}

	// Check for parseable dates
	if due != nil {
		// Extract TZID parameter if present
		var tzid string
		if tzids, exists := due.ICalParameters["TZID"]; exists && len(tzids) > 0 {
			tzid = tzids[0]
		}
		if _, _, err := parseICalDateTimeWithTZ(due.Value, tzid); err != nil {
			return []*ics.VTodo{todo} // Unparseable due date, return as-is
		}
	} else if dtstart != nil {
		// Extract TZID parameter if present
		var tzid string
		if tzids, exists := dtstart.ICalParameters["TZID"]; exists && len(tzids) > 0 {
			tzid = tzids[0]
		}
		if _, _, err := parseICalDateTimeWithTZ(dtstart.Value, tzid); err != nil {
			return []*ics.VTodo{todo} // Unparseable start date, return as-is
		}
	}

	// Use the rrule package to expand the recurring todo
	maxOccurrences := 1000 // Reasonable limit to prevent infinite loops
	expandedTodos, err := rrule.ExpandTodo(todo, maxOccurrences, p.startDate, p.endDate)
	if err != nil {
		// If we can't parse the RRULE, return the original todo
		return []*ics.VTodo{todo}
	}

	return expandedTodos
}

// parseICalDateTime parses iCalendar date/time formats (duplicated from CalDAV for consistency)
func parseICalDateTime(value string) (time.Time, bool, error) {
	return parseICalDateTimeWithTZ(value, "")
}

// parseICalDateTimeWithTZ parses iCalendar date/time formats with optional time zone
// If timezone is specified, converts to local time. If no timezone is specified, assumes local time.
func parseICalDateTimeWithTZ(value, tzid string) (time.Time, bool, error) {
	if strings.HasPrefix(value, "0001") {
		return time.Time{}, false, nil
	}

	// Remove any whitespace
	value = strings.TrimSpace(value)
	tzid = strings.TrimSpace(tzid)

	// Try UTC format with Z suffix first - convert to local time
	if strings.HasSuffix(value, "Z") {
		if t, err := time.Parse("20060102T150405Z", value); err == nil {
			return t.In(time.Local), false, nil // Convert UTC to local time
		}
		// Try with fractional seconds
		if t, err := time.Parse("20060102T150405.000Z", value); err == nil {
			return t.In(time.Local), false, nil // Convert UTC to local time
		}
	}

	// Handle local time with TZID - convert to local time
	if tzid != "" {
		// Try parsing with time component
		if t, err := time.Parse("20060102T150405", value); err == nil {
			// Try to load the time zone using our enhanced mapping
			if loc := mapTimeZone(tzid); loc != nil {
				// Parse the time in the specified time zone
				sourceTime := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
				// Convert to local time
				return sourceTime.In(time.Local), false, nil
			}
			// If timezone cannot be resolved, treat as local time and warn
			return t, false, fmt.Errorf("unrecognized timezone '%s', treating as local time", tzid)
		}

		// Try with fractional seconds
		if t, err := time.Parse("20060102T150405.000", value); err == nil {
			if loc := mapTimeZone(tzid); loc != nil {
				sourceTime := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
				return sourceTime.In(time.Local), false, nil
			}
			return t, false, fmt.Errorf("unrecognized timezone '%s', treating as local time", tzid)
		}
	}

	// Try local time format without Z suffix and without TZID - assume local time
	if t, err := time.Parse("20060102T150405", value); err == nil {
		// No TZID provided, interpret as local time by reconstructing in local timezone
		localTime := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.Local)
		return localTime, false, nil
	}

	// Try with fractional seconds
	if t, err := time.Parse("20060102T150405.000", value); err == nil {
		// No TZID provided, interpret as local time by reconstructing in local timezone
		localTime := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.Local)
		return localTime, false, nil
	}

	// Try date-only format - always local
	if t, err := time.Parse("20060102", value); err == nil {
		// Ensure date is in local time
		localTime := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
		return localTime, true, nil // true indicates all-day
	}

	// Try date with dashes (less common but valid)
	if t, err := time.Parse("2006-01-02", value); err == nil {
		// Ensure date is in local time
		localTime := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
		return localTime, true, nil
	}

	// Try datetime with dashes and colons - assume local
	if t, err := time.Parse("2006-01-02T15:04:05", value); err == nil {
		// No timezone specified, interpret as local time
		localTime := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.Local)
		return localTime, false, nil
	}

	// Try datetime with dashes, colons, and timezone - convert to local
	if t, err := time.Parse("2006-01-02T15:04:05Z", value); err == nil {
		return t.In(time.Local), false, nil // Convert UTC to local time
	}

	return time.Time{}, false, fmt.Errorf("could not parse date/time: %s (tzid: %s)", value, tzid)
}

// mapTimeZone provides common time zone ID mappings for CalDAV/iCalendar compatibility
func mapTimeZone(tzid string) *time.Location {
	// Handle empty timezone
	if tzid == "" {
		return nil
	}

	// Try loading the timezone directly first (for IANA standard names)
	if loc, err := time.LoadLocation(tzid); err == nil {
		return loc
	}

	// Common mappings for Microsoft Exchange and other systems
	commonMappings := map[string]string{
		// Microsoft Exchange Standard Time Names
		"Eastern Standard Time":                "America/New_York",
		"Eastern Daylight Time":               "America/New_York",
		"Central Standard Time":               "America/Chicago",
		"Central Daylight Time":               "America/Chicago",
		"Mountain Standard Time":              "America/Denver",
		"Mountain Daylight Time":              "America/Denver",
		"Pacific Standard Time":               "America/Los_Angeles",
		"Pacific Daylight Time":               "America/Los_Angeles",
		"GMT Standard Time":                   "Europe/London",
		"Greenwich Standard Time":             "Europe/London",
		"W. Europe Standard Time":             "Europe/Berlin",
		"Central European Standard Time":      "Europe/Berlin",
		"Central European Summer Time":        "Europe/Berlin",
		"E. Europe Standard Time":             "Europe/Bucharest",
		"Tokyo Standard Time":                 "Asia/Tokyo",
		"China Standard Time":                 "Asia/Shanghai",
		"India Standard Time":                 "Asia/Kolkata",
		"AUS Eastern Standard Time":           "Australia/Sydney",
		"Australian Eastern Standard Time":    "Australia/Sydney",
		"AUS Central Standard Time":           "Australia/Adelaide",
		"AUS Western Standard Time":           "Australia/Perth",
		// Common abbreviations
		"EST":                                 "America/New_York",
		"EDT":                                 "America/New_York",
		"CST":                                 "America/Chicago",
		"CDT":                                 "America/Chicago",
		"MST":                                 "America/Denver",
		"MDT":                                 "America/Denver",
		"PST":                                 "America/Los_Angeles",
		"PDT":                                 "America/Los_Angeles",
		"UTC":                                 "UTC",
		"GMT":                                 "UTC",
		"CET":                                 "Europe/Berlin",
		"CEST":                                "Europe/Berlin",
		"JST":                                 "Asia/Tokyo",
		"AEST":                                "Australia/Sydney",
		"AEDT":                                "Australia/Sydney",
		// Additional common formats
		"(UTC-05:00) Eastern Time (US & Canada)": "America/New_York",
		"(UTC-06:00) Central Time (US & Canada)": "America/Chicago",
		"(UTC-07:00) Mountain Time (US & Canada)": "America/Denver",
		"(UTC-08:00) Pacific Time (US & Canada)": "America/Los_Angeles",
		"(UTC+00:00) Dublin, Edinburgh, Lisbon, London": "Europe/London",
		"(UTC+01:00) Amsterdam, Berlin, Bern, Rome, Stockholm, Vienna": "Europe/Berlin",
		"(UTC+09:00) Osaka, Sapporo, Tokyo": "Asia/Tokyo",
	}

	if iana, exists := commonMappings[tzid]; exists {
		if loc, err := time.LoadLocation(iana); err == nil {
			return loc
		}
	}

	// Try common timezone name variations
	variations := []string{
		// Remove common suffixes
		strings.ReplaceAll(tzid, " Standard Time", ""),
		strings.ReplaceAll(tzid, " Daylight Time", ""),
		strings.ReplaceAll(tzid, " Summer Time", ""),
		// Replace spaces with underscores (common variation)
		strings.ReplaceAll(tzid, " ", "_"),
	}

	for _, variation := range variations {
		if variation != tzid && variation != "" {
			if loc, err := time.LoadLocation(variation); err == nil {
				return loc
			}
		}
	}

	return nil
}