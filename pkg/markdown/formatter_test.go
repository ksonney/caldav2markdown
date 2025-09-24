package markdown

import (
	"strings"
	"testing"
	"time"

	"github.com/arran4/golang-ical"
)

func TestConvertEventWithEndTime(t *testing.T) {
	// Create a test event with both start and end times
	cal := ics.NewCalendar()
	event := cal.AddEvent("test-uid-1")
	event.SetProperty(ics.ComponentPropertySummary, "Meeting with End Time")
	event.SetProperty(ics.ComponentPropertyDtStart, "20240601T090000Z")
	event.SetProperty(ics.ComponentPropertyDtEnd, "20240601T100000Z")

	md := ConvertEvent(event)

	if md.StartTime.IsZero() {
		t.Error("Start time should not be zero")
	}
	if md.EndTime.IsZero() {
		t.Error("End time should not be zero")
	}

	expectedStart := time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC)

	if !md.StartTime.Equal(expectedStart) {
		t.Errorf("Expected start time %v, got %v", expectedStart, md.StartTime)
	}
	if !md.EndTime.Equal(expectedEnd) {
		t.Errorf("Expected end time %v, got %v", expectedEnd, md.EndTime)
	}
}

func TestConvertEventWithoutEndTime(t *testing.T) {
	// Create a test event without end time
	cal := ics.NewCalendar()
	event := cal.AddEvent("test-uid-2")
	event.SetProperty(ics.ComponentPropertySummary, "Meeting without End Time")
	event.SetProperty(ics.ComponentPropertyDtStart, "20240601T090000Z")
	// No DTEND property

	md := ConvertEvent(event)

	if md.StartTime.IsZero() {
		t.Error("Start time should not be zero")
	}
	if md.EndTime.IsZero() {
		t.Error("End time should be automatically set when missing")
	}

	expectedStart := time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC)
	expectedEnd := expectedStart.Add(time.Hour) // Default 1-hour duration

	if !md.StartTime.Equal(expectedStart) {
		t.Errorf("Expected start time %v, got %v", expectedStart, md.StartTime)
	}
	if !md.EndTime.Equal(expectedEnd) {
		t.Errorf("Expected end time %v, got %v", expectedEnd, md.EndTime)
	}
}

func TestConvertEventWithDuration(t *testing.T) {
	// Create a test event with duration instead of end time
	cal := ics.NewCalendar()
	event := cal.AddEvent("test-uid-3")
	event.SetProperty(ics.ComponentPropertySummary, "Meeting with Duration")
	event.SetProperty(ics.ComponentPropertyDtStart, "20240601T090000Z")
	event.SetProperty(ics.ComponentPropertyDuration, "PT2H30M") // 2 hours 30 minutes
	// No DTEND property

	md := ConvertEvent(event)

	if md.StartTime.IsZero() {
		t.Error("Start time should not be zero")
	}
	if md.EndTime.IsZero() {
		t.Error("End time should be automatically set based on duration")
	}

	expectedStart := time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC)
	expectedEnd := expectedStart.Add(2*time.Hour + 30*time.Minute)

	if !md.StartTime.Equal(expectedStart) {
		t.Errorf("Expected start time %v, got %v", expectedStart, md.StartTime)
	}
	if !md.EndTime.Equal(expectedEnd) {
		t.Errorf("Expected end time %v, got %v", expectedEnd, md.EndTime)
	}
}

func TestConvertAllDayEvent(t *testing.T) {
	// Create an all-day event
	cal := ics.NewCalendar()
	event := cal.AddEvent("test-uid-4")
	event.SetProperty(ics.ComponentPropertySummary, "All Day Event")
	event.SetProperty(ics.ComponentPropertyDtStart, "20240601")
	// No DTEND property

	md := ConvertEvent(event)

	if !md.AllDay {
		t.Error("Event should be marked as all-day")
	}
	if md.StartTime.IsZero() {
		t.Error("Start time should not be zero")
	}
	if md.EndTime.IsZero() {
		t.Error("End time should be automatically set for all-day events")
	}

	expectedStart := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	// All-day events should end at 23:59:59 of the same day
	expectedEnd := expectedStart.Add(24*time.Hour - time.Second)

	if !md.StartTime.Equal(expectedStart) {
		t.Errorf("Expected start time %v, got %v", expectedStart, md.StartTime)
	}
	if !md.EndTime.Equal(expectedEnd) {
		t.Errorf("Expected end time %v, got %v", expectedEnd, md.EndTime)
	}
}

func TestToMarkdownShowsBothTimes(t *testing.T) {
	// Create event markdown with both times
	md := EventMarkdown{
		Title:     "Test Event",
		StartTime: time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 6, 1, 10, 30, 0, 0, time.UTC),
		AllDay:    false,
	}

	markdown := md.ToMarkdown()

	// Check that both start and end times are shown
	if !strings.Contains(markdown, "**Start:** 2024-06-01 09:00") {
		t.Error("Markdown should contain start time")
	}
	if !strings.Contains(markdown, "**End:** 2024-06-01 10:30") {
		t.Error("Markdown should contain end time")
	}
	if !strings.Contains(markdown, "**Duration:** 1h 30m") {
		t.Error("Markdown should contain duration")
	}
}

func TestToMarkdownAllDayEvent(t *testing.T) {
	// Create all-day event markdown
	md := EventMarkdown{
		Title:     "All Day Test",
		StartTime: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 6, 1, 23, 59, 59, 0, time.UTC),
		AllDay:    true,
	}

	markdown := md.ToMarkdown()

	// Check that all-day format is used
	if !strings.Contains(markdown, "**Date:** 2024-06-01 (All Day)") {
		t.Error("Markdown should contain all-day date format")
	}
}

func TestParseICalDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"PT1H", time.Hour},
		{"PT30M", 30 * time.Minute},
		{"PT1H30M", time.Hour + 30*time.Minute},
		{"P1D", 24 * time.Hour},
		{"P1W", 7 * 24 * time.Hour},
		{"PT2H30M15S", 2*time.Hour + 30*time.Minute + 15*time.Second},
	}

	for _, test := range tests {
		result, err := parseICalDuration(test.input)
		if err != nil {
			t.Errorf("Failed to parse duration %s: %v", test.input, err)
			continue
		}
		if result != test.expected {
			t.Errorf("For input %s, expected %v, got %v", test.input, test.expected, result)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{30 * time.Minute, "30m"},
		{time.Hour, "1h"},
		{time.Hour + 30*time.Minute, "1h 30m"},
		{2*time.Hour + 15*time.Minute, "2h 15m"},
		{0, "0m"},
	}

	for _, test := range tests {
		result := formatDuration(test.input)
		if result != test.expected {
			t.Errorf("For duration %v, expected %s, got %s", test.input, test.expected, result)
		}
	}
}