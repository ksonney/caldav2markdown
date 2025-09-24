package rrule

import (
	"strings"
	"testing"
	"time"

	"github.com/arran4/golang-ical"
)

func TestParseRRule(t *testing.T) {
	tests := []struct {
		name     string
		rrule    string
		expected *RecurrenceRule
	}{
		{
			name:  "Daily with interval",
			rrule: "FREQ=DAILY;INTERVAL=2",
			expected: &RecurrenceRule{
				Freq:     "DAILY",
				Interval: 2,
			},
		},
		{
			name:  "Weekly with count",
			rrule: "FREQ=WEEKLY;COUNT=10",
			expected: &RecurrenceRule{
				Freq:     "WEEKLY",
				Interval: 1,
				Count:    10,
			},
		},
		{
			name:  "Monthly with BYMONTH",
			rrule: "FREQ=MONTHLY;BYMONTH=1,6,12",
			expected: &RecurrenceRule{
				Freq:     "MONTHLY",
				Interval: 1,
				ByMonth:  []int{1, 6, 12},
			},
		},
		{
			name:  "Weekly with BYDAY",
			rrule: "FREQ=WEEKLY;BYDAY=MO,WE,FR",
			expected: &RecurrenceRule{
				Freq:     "WEEKLY",
				Interval: 1,
				ByDay:    []string{"MO", "WE", "FR"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseRRule(tt.rrule)
			if err != nil {
				t.Errorf("ParseRRule() error = %v", err)
				return
			}

			if result.Freq != tt.expected.Freq {
				t.Errorf("Freq = %v, want %v", result.Freq, tt.expected.Freq)
			}
			if result.Interval != tt.expected.Interval {
				t.Errorf("Interval = %v, want %v", result.Interval, tt.expected.Interval)
			}
			if result.Count != tt.expected.Count {
				t.Errorf("Count = %v, want %v", result.Count, tt.expected.Count)
			}
		})
	}
}

func TestExpandEvent(t *testing.T) {
	// Create a test calendar with a recurring event
	cal := ics.NewCalendar()
	event := cal.AddEvent("test-uid-123")
	event.SetProperty(ics.ComponentPropertySummary, "Daily Meeting")
	event.SetProperty(ics.ComponentPropertyDtStart, "20240101T090000Z")
	event.SetProperty(ics.ComponentPropertyDtEnd, "20240101T100000Z")
	event.SetProperty(ics.ComponentPropertyRrule, "FREQ=DAILY;COUNT=5")

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	expanded, err := ExpandEvent(event, 10, startDate, endDate)

	if err != nil {
		t.Errorf("ExpandEvent() error = %v", err)
		return
	}

	if len(expanded) != 5 {
		t.Errorf("Expected 5 occurrences, got %d", len(expanded))
	}

	// Check that each occurrence has a unique UID
	uids := make(map[string]bool)
	for _, occurrence := range expanded {
		uid := occurrence.GetProperty(ics.ComponentPropertyUniqueId).Value
		if uids[uid] {
			t.Errorf("Duplicate UID found: %s", uid)
		}
		uids[uid] = true
	}

	// Check that the first occurrence starts on Jan 1st
	firstStart := expanded[0].GetProperty(ics.ComponentPropertyDtStart).Value
	if !strings.HasPrefix(firstStart, "20240101") {
		t.Errorf("First occurrence should start on 20240101, got %s", firstStart)
	}

	// Check that the second occurrence starts on Jan 2nd
	secondStart := expanded[1].GetProperty(ics.ComponentPropertyDtStart).Value
	if !strings.HasPrefix(secondStart, "20240102") {
		t.Errorf("Second occurrence should start on 20240102, got %s", secondStart)
	}
}

func TestExpandEventNonRecurring(t *testing.T) {
	// Create a test calendar with a non-recurring event
	cal := ics.NewCalendar()
	event := cal.AddEvent("test-uid-456")
	event.SetProperty(ics.ComponentPropertySummary, "One-time Meeting")
	event.SetProperty(ics.ComponentPropertyDtStart, "20240101T090000Z")
	event.SetProperty(ics.ComponentPropertyDtEnd, "20240101T100000Z")

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	expanded, err := ExpandEvent(event, 10, startDate, endDate)

	if err != nil {
		t.Errorf("ExpandEvent() error = %v", err)
		return
	}

	if len(expanded) != 1 {
		t.Errorf("Expected 1 occurrence for non-recurring event, got %d", len(expanded))
	}

	if expanded[0] != event {
		t.Error("Non-recurring event should return the original event")
	}
}