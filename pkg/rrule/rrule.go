package rrule

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/arran4/golang-ical"
)

// parseICalDateTime attempts to parse an iCalendar date/time string in various formats
func parseICalDateTime(value string) (time.Time, bool, error) {
	if strings.HasPrefix(value, "0001") {
		return time.Time{}, false, nil
	}

	// Try UTC format with Z suffix first
	if t, err := time.Parse("20060102T150405Z", value); err == nil {
		return t, false, nil
	}

	// Try local time format without Z suffix
	if t, err := time.Parse("20060102T150405", value); err == nil {
		return t, false, nil
	}

	// Try date-only format
	if t, err := time.Parse("20060102", value); err == nil {
		return t, true, nil // true indicates all-day
	}

	return time.Time{}, false, fmt.Errorf("could not parse date/time: %s", value)
}

// RecurrenceRule represents a parsed RRULE
type RecurrenceRule struct {
	Freq     string
	Interval int
	Count    int
	Until    time.Time
	ByMonth  []int
	ByDay    []string
	ByHour   []int
	ByMinute []int
}

// ParseRRule parses an RRULE string into a RecurrenceRule struct
func ParseRRule(rruleStr string) (*RecurrenceRule, error) {
	rule := &RecurrenceRule{
		Interval: 1, // default interval is 1
	}

	// Split the RRULE string by semicolons
	parts := strings.Split(rruleStr, ";")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key, value := kv[0], kv[1]
		switch key {
		case "FREQ":
			rule.Freq = value
		case "INTERVAL":
			if interval, err := strconv.Atoi(value); err == nil {
				rule.Interval = interval
			}
		case "COUNT":
			if count, err := strconv.Atoi(value); err == nil {
				rule.Count = count
			}
		case "UNTIL":
			if until, _, err := parseICalDateTime(value); err == nil {
				rule.Until = until
			}
		case "BYMONTH":
			months := strings.Split(value, ",")
			for _, monthStr := range months {
				if month, err := strconv.Atoi(monthStr); err == nil {
					rule.ByMonth = append(rule.ByMonth, month)
				}
			}
		case "BYDAY":
			rule.ByDay = strings.Split(value, ",")
		case "BYHOUR":
			hours := strings.Split(value, ",")
			for _, hourStr := range hours {
				if hour, err := strconv.Atoi(hourStr); err == nil {
					rule.ByHour = append(rule.ByHour, hour)
				}
			}
		case "BYMINUTE":
			minutes := strings.Split(value, ",")
			for _, minuteStr := range minutes {
				if minute, err := strconv.Atoi(minuteStr); err == nil {
					rule.ByMinute = append(rule.ByMinute, minute)
				}
			}
		}
	}

	return rule, nil
}

// ExpandEvent expands a recurring event into individual occurrences within a given time range
func ExpandEvent(event *ics.VEvent, maxOccurrences int, startDate, endDate time.Time) ([]*ics.VEvent, error) {
	rruleProp := event.GetProperty(ics.ComponentPropertyRrule)
	if rruleProp == nil {
		// Not a recurring event, return the original
		return []*ics.VEvent{event}, nil
	}

	rule, err := ParseRRule(rruleProp.Value)
	if err != nil {
		return []*ics.VEvent{event}, nil // Return original on parse error
	}

	// Get the start date of the original event
	dtstart := event.GetProperty(ics.ComponentPropertyDtStart)
	if dtstart == nil {
		return []*ics.VEvent{event}, nil
	}

	var startTime time.Time
	var allDay bool

	if strings.HasPrefix(dtstart.Value, "0001") {
		// Allow processing of 0001-01-01 events but don't expand them
		return []*ics.VEvent{event}, nil
	} else if t, isAllDay, err := parseICalDateTime(dtstart.Value); err == nil {
		startTime = t
		allDay = isAllDay
	} else {
		return []*ics.VEvent{event}, nil
	}

	// Get the duration of the event
	var duration time.Duration
	if dtend := event.GetProperty(ics.ComponentPropertyDtEnd); dtend != nil && !allDay {
		if endTime, _, err := parseICalDateTime(dtend.Value); err == nil {
			duration = endTime.Sub(startTime)
		}
	}

	var events []*ics.VEvent
	current := startTime
	count := 0

	// Limit to prevent infinite loops
	if maxOccurrences == 0 {
		maxOccurrences = 1000
	}

	for count < maxOccurrences {
		// Check if we've reached the UNTIL date or COUNT limit
		if !rule.Until.IsZero() && current.After(rule.Until) {
			break
		}
		if rule.Count > 0 && count >= rule.Count {
			break
		}
		if current.After(endDate) {
			break
		}

		// Skip occurrences before the start date
		if current.Before(startDate) {
			current = getNextOccurrence(current, rule)
			continue
		}

		// Check constraints (BYMONTH, BYDAY, etc.)
		if !matchesConstraints(current, rule) {
			current = getNextOccurrence(current, rule)
			continue
		}

		// Create a new event for this occurrence
		newEvent := createEventOccurrence(event, current, duration, allDay)
		events = append(events, newEvent)
		count++

		current = getNextOccurrence(current, rule)
	}

	return events, nil
}

// matchesConstraints checks if a date matches the RRULE constraints
func matchesConstraints(date time.Time, rule *RecurrenceRule) bool {
	// Check BYMONTH
	if len(rule.ByMonth) > 0 {
		month := int(date.Month())
		found := false
		for _, m := range rule.ByMonth {
			if m == month {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check BYDAY (simplified - only handles basic weekday names)
	if len(rule.ByDay) > 0 {
		weekday := date.Weekday().String()
		found := false
		for _, day := range rule.ByDay {
			// Convert RRULE day format (MO, TU, WE, TH, FR, SA, SU) to Go format
			var goDay string
			switch day {
			case "MO":
				goDay = "Monday"
			case "TU":
				goDay = "Tuesday"
			case "WE":
				goDay = "Wednesday"
			case "TH":
				goDay = "Thursday"
			case "FR":
				goDay = "Friday"
			case "SA":
				goDay = "Saturday"
			case "SU":
				goDay = "Sunday"
			}
			if goDay == weekday {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check BYHOUR
	if len(rule.ByHour) > 0 {
		hour := date.Hour()
		found := false
		for _, h := range rule.ByHour {
			if h == hour {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check BYMINUTE
	if len(rule.ByMinute) > 0 {
		minute := date.Minute()
		found := false
		for _, m := range rule.ByMinute {
			if m == minute {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// getNextOccurrence calculates the next occurrence based on frequency and interval
func getNextOccurrence(current time.Time, rule *RecurrenceRule) time.Time {
	switch rule.Freq {
	case "DAILY":
		return current.AddDate(0, 0, rule.Interval)
	case "WEEKLY":
		return current.AddDate(0, 0, 7*rule.Interval)
	case "MONTHLY":
		return current.AddDate(0, rule.Interval, 0)
	case "YEARLY":
		return current.AddDate(rule.Interval, 0, 0)
	default:
		return current.AddDate(0, 0, rule.Interval) // Default to daily
	}
}

// createEventOccurrence creates a new event occurrence for a specific date
func createEventOccurrence(originalEvent *ics.VEvent, newStart time.Time, duration time.Duration, allDay bool) *ics.VEvent {
	newEvent := &ics.VEvent{}

	// Copy all properties from the original event
	for _, prop := range originalEvent.Properties {
		newProp := &ics.IANAProperty{
			BaseProperty: ics.BaseProperty{
				IANAToken:       prop.IANAToken,
				Value:           prop.Value,
				ICalParameters:  make(map[string][]string),
			},
		}

		// Copy parameters
		if prop.ICalParameters != nil {
			for key, values := range prop.ICalParameters {
				newProp.ICalParameters[key] = make([]string, len(values))
				copy(newProp.ICalParameters[key], values)
			}
		}

		newEvent.Properties = append(newEvent.Properties, *newProp)
	}

	// Update DTSTART
	var dtStartValue string
	if allDay {
		dtStartValue = newStart.Format("20060102")
	} else {
		dtStartValue = newStart.UTC().Format("20060102T150405Z")
	}
	newEvent.SetProperty(ics.ComponentPropertyDtStart, dtStartValue)

	// Update DTEND if duration is set
	if duration > 0 {
		newEnd := newStart.Add(duration)
		var dtEndValue string
		if allDay {
			dtEndValue = newEnd.Format("20060102")
		} else {
			dtEndValue = newEnd.UTC().Format("20060102T150405Z")
		}
		newEvent.SetProperty(ics.ComponentPropertyDtEnd, dtEndValue)
	}

	// Remove the RRULE property from the occurrence
	newEvent.RemoveProperty(ics.ComponentPropertyRrule)

	// Update UID to make each occurrence unique
	if uid := newEvent.GetProperty(ics.ComponentPropertyUniqueId); uid != nil {
		newUID := uid.Value + "_" + newStart.Format("20060102T150405")
		newEvent.SetProperty(ics.ComponentPropertyUniqueId, newUID)
	}

	return newEvent
}

// ExpandTodo expands a recurring todo into individual occurrences within a given time range
func ExpandTodo(todo *ics.VTodo, maxOccurrences int, startDate, endDate time.Time) ([]*ics.VTodo, error) {
	rruleProp := todo.GetProperty(ics.ComponentPropertyRrule)
	if rruleProp == nil {
		// Not a recurring todo, return the original
		return []*ics.VTodo{todo}, nil
	}

	rule, err := ParseRRule(rruleProp.Value)
	if err != nil {
		return []*ics.VTodo{todo}, nil // Return original on parse error
	}

	// Get the due date or start date of the original todo
	var baseTime time.Time
	var allDay bool

	// Try DUE date first, then DTSTART
	if due := todo.GetProperty(ics.ComponentPropertyDue); due != nil {
		if strings.HasPrefix(due.Value, "0001") {
			// Allow processing of 0001-01-01 todos but don't expand them
			return []*ics.VTodo{todo}, nil
		} else if t, isAllDay, err := parseICalDateTime(due.Value); err == nil {
			baseTime = t
			allDay = isAllDay
		}
	} else if dtstart := todo.GetProperty(ics.ComponentPropertyDtStart); dtstart != nil {
		if strings.HasPrefix(dtstart.Value, "0001") {
			return []*ics.VTodo{todo}, nil
		} else if t, isAllDay, err := parseICalDateTime(dtstart.Value); err == nil {
			baseTime = t
			allDay = isAllDay
		}
	}

	if baseTime.IsZero() {
		return []*ics.VTodo{todo}, nil
	}

	var todos []*ics.VTodo
	current := baseTime
	count := 0

	// Limit to prevent infinite loops
	if maxOccurrences == 0 {
		maxOccurrences = 1000
	}

	for count < maxOccurrences {
		// Check if we've reached the UNTIL date or COUNT limit
		if !rule.Until.IsZero() && current.After(rule.Until) {
			break
		}
		if rule.Count > 0 && count >= rule.Count {
			break
		}
		if current.After(endDate) {
			break
		}

		// Skip occurrences before the start date
		if current.Before(startDate) {
			current = getNextOccurrence(current, rule)
			continue
		}

		// Check constraints (BYMONTH, BYDAY, etc.)
		if !matchesConstraints(current, rule) {
			current = getNextOccurrence(current, rule)
			continue
		}

		// Create a new todo for this occurrence
		newTodo := createTodoOccurrence(todo, current, allDay)
		todos = append(todos, newTodo)
		count++

		current = getNextOccurrence(current, rule)
	}

	return todos, nil
}

// createTodoOccurrence creates a new todo occurrence for a specific date
func createTodoOccurrence(originalTodo *ics.VTodo, newDate time.Time, allDay bool) *ics.VTodo {
	newTodo := &ics.VTodo{}

	// Copy all properties from the original todo
	for _, prop := range originalTodo.Properties {
		newProp := &ics.IANAProperty{
			BaseProperty: ics.BaseProperty{
				IANAToken:       prop.IANAToken,
				Value:           prop.Value,
				ICalParameters:  make(map[string][]string),
			},
		}

		// Copy parameters
		if prop.ICalParameters != nil {
			for key, values := range prop.ICalParameters {
				newProp.ICalParameters[key] = make([]string, len(values))
				copy(newProp.ICalParameters[key], values)
			}
		}

		newTodo.Properties = append(newTodo.Properties, *newProp)
	}

	// Update DUE date if it exists, otherwise update DTSTART
	var dateValue string
	if allDay {
		dateValue = newDate.Format("20060102")
	} else {
		dateValue = newDate.UTC().Format("20060102T150405Z")
	}

	if originalTodo.GetProperty(ics.ComponentPropertyDue) != nil {
		newTodo.SetProperty(ics.ComponentPropertyDue, dateValue)
	} else if originalTodo.GetProperty(ics.ComponentPropertyDtStart) != nil {
		newTodo.SetProperty(ics.ComponentPropertyDtStart, dateValue)
	}

	// Remove the RRULE property from the occurrence
	newTodo.RemoveProperty(ics.ComponentPropertyRrule)

	// Update UID to make each occurrence unique
	if uid := newTodo.GetProperty(ics.ComponentPropertyUniqueId); uid != nil {
		newUID := uid.Value + "_" + newDate.Format("20060102T150405")
		newTodo.SetProperty(ics.ComponentPropertyUniqueId, newUID)
	}

	return newTodo
}