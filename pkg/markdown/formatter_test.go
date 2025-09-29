package markdown

import (
	"io/ioutil"
	"os"
	"path/filepath"
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

func TestGenerateFilenameWithDirectoryStructure(t *testing.T) {
	// Test the new directory structure generation
	md := EventMarkdown{
		Title:     "Test Event",
		StartTime: time.Date(2024, 6, 15, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC),
		AllDay:    false,
	}

	outputDir := "/tmp/events"
	filename := GenerateFilename(outputDir, md)

	expected := "/tmp/events/2024/06/2024-06-15_Test Event.md"
	if filename != expected {
		t.Errorf("Expected filename %s, got %s", expected, filename)
	}
}

func TestGenerateFilenameMultipleYears(t *testing.T) {
	// Test events from different years/months
	testCases := []struct {
		date     time.Time
		title    string
		expected string
	}{
		{
			date:     time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			title:    "New Year",
			expected: "events/2023/01/2023-01-01_New Year.md",
		},
		{
			date:     time.Date(2024, 12, 31, 23, 59, 0, 0, time.UTC),
			title:    "Year End",
			expected: "events/2024/12/2024-12-31_Year End.md",
		},
	}

	for _, tc := range testCases {
		md := EventMarkdown{
			Title:     tc.title,
			StartTime: tc.date,
			EndTime:   tc.date.Add(time.Hour),
			AllDay:    false,
		}

		filename := GenerateFilename("events", md)
		if filename != tc.expected {
			t.Errorf("For date %v, expected %s, got %s", tc.date, tc.expected, filename)
		}
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

func TestGenerateDailyFilesWithTasks(t *testing.T) {
	// Create a temporary directory for test output
	tmpDir, err := ioutil.TempDir("", "caldav_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test events
	events := []EventMarkdown{
		{
			Title:     "Morning Meeting",
			StartTime: time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC),
		},
	}

	// Create test tasks
	tasks := []TodoMarkdown{
		{
			Title:   "Complete project report",
			DueDate: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			Title:   "Review documents",
			DueDate: time.Date(2024, 6, 2, 0, 0, 0, 0, time.UTC),
		},
	}

	// Generate daily files with both events and tasks (without emoji or hashtags)
	err = GenerateDailyFilesWithTasks(tmpDir, events, tasks, false, false)
	if err != nil {
		t.Fatalf("Failed to generate daily files: %v", err)
	}

	// Check that the file for 2024-06-01 exists and contains both event and task
	expectedFile := filepath.Join(tmpDir, "2024", "06", "2024-06-01.md")
	content, err := ioutil.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read generated file %s: %v", expectedFile, err)
	}

	contentStr := string(content)

	// Check that the event is included
	if !strings.Contains(contentStr, "Morning Meeting") {
		t.Error("Daily file should contain the event")
	}

	// Check that the task is included
	if !strings.Contains(contentStr, "Complete project report") {
		t.Error("Daily file should contain the task")
	}

	// Check that the Tasks section exists
	if !strings.Contains(contentStr, "## Tasks") {
		t.Error("Daily file should contain the Tasks section")
	}

	// Check that the second task is in a different file
	expectedFile2 := filepath.Join(tmpDir, "2024", "06", "2024-06-02.md")
	content2, err := ioutil.ReadFile(expectedFile2)
	if err != nil {
		t.Fatalf("Failed to read generated file %s: %v", expectedFile2, err)
	}

	contentStr2 := string(content2)
	if !strings.Contains(contentStr2, "Review documents") {
		t.Error("Second daily file should contain the second task")
	}
}

func TestGenerateDailyFilesWithTasksAndEmoji(t *testing.T) {
	// Create a temporary directory for test output
	tmpDir, err := ioutil.TempDir("", "caldav_emoji_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test tasks
	tasks := []TodoMarkdown{
		{
			Title:   "Complete project report",
			DueDate: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	// Generate daily files with emoji enabled (hashtags disabled)
	err = GenerateDailyFilesWithTasks(tmpDir, nil, tasks, true, false)
	if err != nil {
		t.Fatalf("Failed to generate daily files with emoji: %v", err)
	}

	// Check that the file contains the emoji
	expectedFile := filepath.Join(tmpDir, "2024", "06", "2024-06-01.md")
	content, err := ioutil.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read generated file %s: %v", expectedFile, err)
	}

	contentStr := string(content)

	// Check that the task uses emoji for due date
	if !strings.Contains(contentStr, "📅 2024-06-01") {
		t.Error("Daily file should contain the due date with calendar emoji")
	}

	// Check that it doesn't contain the old "Due:" format
	if strings.Contains(contentStr, "Due: 2024-06-01") {
		t.Error("Daily file should not contain the old 'Due:' format when emoji is enabled")
	}
}

func TestGenerateDailyFilesWithHashtags(t *testing.T) {
	// Create a temporary directory for test output
	tmpDir, err := ioutil.TempDir("", "caldav_hashtag_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test events
	events := []EventMarkdown{
		{
			Title:     "Team Meeting",
			StartTime: time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC),
		},
		{
			Title:  "All Day Conference",
			AllDay: true,
			StartTime: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	// Create test tasks
	tasks := []TodoMarkdown{
		{
			Title:   "Complete project report",
			DueDate: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	// Generate daily files with hashtags enabled
	err = GenerateDailyFilesWithTasks(tmpDir, events, tasks, false, true)
	if err != nil {
		t.Fatalf("Failed to generate daily files with hashtags: %v", err)
	}

	// Check that the file contains the hashtags
	expectedFile := filepath.Join(tmpDir, "2024", "06", "2024-06-01.md")
	content, err := ioutil.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read generated file %s: %v", expectedFile, err)
	}

	contentStr := string(content)

	// Check that events have #event hashtag
	if !strings.Contains(contentStr, "Team Meeting") {
		t.Error("Daily file should contain the scheduled event")
	}
	if !strings.Contains(contentStr, "#event") {
		t.Error("Daily file should contain #event hashtag")
	}

	// Check that tasks have #task hashtag
	if !strings.Contains(contentStr, "Complete project report") {
		t.Error("Daily file should contain the task")
	}
	if !strings.Contains(contentStr, "#task") {
		t.Error("Daily file should contain #task hashtag")
	}

	// Count hashtag occurrences to ensure they're properly placed
	eventHashtagCount := strings.Count(contentStr, "#event")
	taskHashtagCount := strings.Count(contentStr, "#task")

	if eventHashtagCount != 2 { // Two events
		t.Errorf("Expected 2 #event hashtags, found %d", eventHashtagCount)
	}
	if taskHashtagCount != 1 { // One task
		t.Errorf("Expected 1 #task hashtag, found %d", taskHashtagCount)
	}
}

func TestFileMerging(t *testing.T) {
	// Create a temporary directory for test output
	tmpDir, err := ioutil.TempDir("", "caldav_merge_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// First, create an initial file with some content
	initialEvents := []EventMarkdown{
		{
			Title:     "Initial Meeting",
			StartTime: time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC),
		},
	}

	initialTasks := []TodoMarkdown{
		{
			Title:   "Initial task",
			DueDate: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	// Generate initial file
	err = GenerateDailyFilesWithTasks(tmpDir, initialEvents, initialTasks, false, false)
	if err != nil {
		t.Fatalf("Failed to generate initial daily files: %v", err)
	}

	// Verify initial content
	expectedFile := filepath.Join(tmpDir, "2024", "06", "2024-06-01.md")
	initialContent, err := ioutil.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read initial file: %v", err)
	}

	if !strings.Contains(string(initialContent), "Initial Meeting") {
		t.Error("Initial file should contain Initial Meeting")
	}
	if !strings.Contains(string(initialContent), "Initial task") {
		t.Error("Initial file should contain Initial task")
	}

	// Now add new content to the same date
	newEvents := []EventMarkdown{
		{
			Title:     "New Meeting",
			StartTime: time.Date(2024, 6, 1, 14, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2024, 6, 1, 15, 0, 0, 0, time.UTC),
		},
	}

	newTasks := []TodoMarkdown{
		{
			Title:   "New task",
			DueDate: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	// Generate file again (should merge, not overwrite)
	err = GenerateDailyFilesWithTasks(tmpDir, newEvents, newTasks, false, false)
	if err != nil {
		t.Fatalf("Failed to generate merged daily files: %v", err)
	}

	// Verify merged content contains both old and new items
	mergedContent, err := ioutil.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read merged file: %v", err)
	}

	contentStr := string(mergedContent)

	// Check that both initial and new events are present
	if !strings.Contains(contentStr, "Initial Meeting") {
		t.Error("Merged file should still contain Initial Meeting")
	}
	if !strings.Contains(contentStr, "New Meeting") {
		t.Error("Merged file should contain New Meeting")
	}

	// Check that both initial and new tasks are present
	if !strings.Contains(contentStr, "Initial task") {
		t.Error("Merged file should still contain Initial task")
	}
	if !strings.Contains(contentStr, "New task") {
		t.Error("Merged file should contain New task")
	}

	// Test deduplication - add the same event again
	duplicateEvents := []EventMarkdown{
		{
			Title:     "New Meeting", // Same as before
			StartTime: time.Date(2024, 6, 1, 14, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2024, 6, 1, 15, 0, 0, 0, time.UTC),
		},
	}

	err = GenerateDailyFilesWithTasks(tmpDir, duplicateEvents, nil, false, false)
	if err != nil {
		t.Fatalf("Failed to generate file with duplicates: %v", err)
	}

	// Verify no duplicate entries
	finalContent, err := ioutil.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read final file: %v", err)
	}

	finalContentStr := string(finalContent)
	newMeetingCount := strings.Count(finalContentStr, "New Meeting")
	if newMeetingCount != 1 {
		t.Errorf("Expected 1 occurrence of 'New Meeting', found %d", newMeetingCount)
	}
}

func TestProgressCallback(t *testing.T) {
	// Create a temporary directory for test output
	tmpDir, err := ioutil.TempDir("", "caldav_progress_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test events for multiple dates
	events := []EventMarkdown{
		{
			Title:     "Event 1",
			StartTime: time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC),
		},
		{
			Title:     "Event 2",
			StartTime: time.Date(2024, 6, 2, 14, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2024, 6, 2, 15, 0, 0, 0, time.UTC),
		},
	}

	// Track progress callbacks
	var progressMessages []string
	var progressCounts []int
	var progressTotals []int

	progressCallback := func(message string, current, total int) {
		progressMessages = append(progressMessages, message)
		progressCounts = append(progressCounts, current)
		progressTotals = append(progressTotals, total)
	}

	// Generate files with progress callback
	err = GenerateDailyFilesWithTasksAndProgress(tmpDir, events, nil, false, false, progressCallback)
	if err != nil {
		t.Fatalf("Failed to generate daily files with progress: %v", err)
	}

	// Verify that progress callbacks were called
	if len(progressMessages) == 0 {
		t.Error("Expected progress callbacks to be called")
	}

	// Check that we got progress for generating files
	foundGeneratingMessage := false
	for _, msg := range progressMessages {
		if strings.Contains(msg, "Generating") && strings.Contains(msg, "daily files") {
			foundGeneratingMessage = true
			break
		}
	}
	if !foundGeneratingMessage {
		t.Error("Expected to find 'Generating daily files' progress message")
	}

	// Check that progress counts make sense (should go from 0 to total)
	if len(progressCounts) > 0 {
		firstCount := progressCounts[0]
		lastCount := progressCounts[len(progressCounts)-1]
		lastTotal := progressTotals[len(progressTotals)-1]

		if firstCount != 0 {
			t.Errorf("Expected first progress count to be 0, got %d", firstCount)
		}
		if lastCount != lastTotal {
			t.Errorf("Expected last progress count (%d) to equal total (%d)", lastCount, lastTotal)
		}
	}
}

// TestIsInvalidHashtag tests the hashtag validation function
func TestIsInvalidHashtag(t *testing.T) {
	tests := []struct {
		tag      string
		expected bool
		name     string
	}{
		{"", true, "empty string"},
		{"   ", true, "whitespace only"},
		{"-", true, "single dash"},
		{"x", true, "single x"},
		{"_", true, "single underscore"},
		{".", true, "single dot"},
		{",", true, "single comma"},
		{":", true, "single colon"},
		{";", true, "single semicolon"},
		{"!", true, "single exclamation"},
		{"?", true, "single question mark"},
		{"a", false, "valid single letter"},
		{"5", false, "valid single digit"},
		{"work", false, "valid word"},
		{"personal-calendar", false, "valid hyphenated tag"},
		{"team123", false, "valid alphanumeric tag"},
	}

	for _, test := range tests {
		result := isInvalidHashtag(test.tag)
		if result != test.expected {
			t.Errorf("isInvalidHashtag(%q) = %v, expected %v (test: %s)", test.tag, result, test.expected, test.name)
		}
	}
}

// TestSanitizeForHashtag tests the hashtag sanitization function
func TestSanitizeForHashtag(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		name     string
	}{
		{"Work Calendar", "work-calendar", "spaces to hyphens"},
		{"Personal_Calendar", "personal-calendar", "underscores to hyphens"},
		{"Team@Company", "teamcompany", "special chars removed"},
		{"Project 123", "project-123", "numbers preserved"},
		{"", "", "empty input"},
		{" - ", "", "invalid single dash filtered"},
		{"X", "", "invalid single X filtered"},
		{"Multiple   Spaces", "multiple-spaces", "multiple spaces collapsed"},
		{"--dash--prefix--", "dash-prefix", "multiple dashes cleaned"},
		{"Calendar-2024", "calendar-2024", "valid hyphenated name"},
		{"___", "", "only underscores filtered"},
		{"!@#$%", "", "only symbols filtered"},
	}

	for _, test := range tests {
		result := sanitizeForHashtag(test.input)
		if result != test.expected {
			t.Errorf("sanitizeForHashtag(%q) = %q, expected %q (test: %s)", test.input, result, test.expected, test.name)
		}
	}
}

// TestDeduplicateHashtags tests the hashtag deduplication function
func TestDeduplicateHashtags(t *testing.T) {
	tests := []struct {
		input    []string
		expected []string
		name     string
	}{
		{
			[]string{"#event", "#work", "#event"},
			[]string{"#event", "#work"},
			"basic deduplication",
		},
		{
			[]string{"#event", "#-", "#work", "#x"},
			[]string{"#event", "#work"},
			"invalid hashtags filtered",
		},
		{
			[]string{"event", "work", "event"},
			[]string{"#event", "#work"},
			"hashtags without # prefix",
		},
		{
			[]string{"", " ", "#event", "  #work  "},
			[]string{"#event", "#work"},
			"whitespace handling",
		},
		{
			[]string{"#_", "#.", "#valid", "#!"},
			[]string{"#valid"},
			"all invalid symbols filtered",
		},
		{
			[]string{},
			[]string{},
			"empty input",
		},
	}

	for _, test := range tests {
		result := deduplicateHashtags(test.input)
		if len(result) != len(test.expected) {
			t.Errorf("deduplicateHashtags(%v) length = %d, expected %d (test: %s)", test.input, len(result), len(test.expected), test.name)
			continue
		}
		for i, hashtag := range result {
			if hashtag != test.expected[i] {
				t.Errorf("deduplicateHashtags(%v)[%d] = %q, expected %q (test: %s)", test.input, i, hashtag, test.expected[i], test.name)
			}
		}
	}
}