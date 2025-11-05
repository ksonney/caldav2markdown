package org

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"caldav2markdown/pkg/markdown"
)

// EventOrg represents an event in Org format (wraps markdown.EventMarkdown)
type EventOrg struct {
	markdown.EventMarkdown
}

// TodoOrg represents a todo in Org format (wraps markdown.TodoMarkdown)
type TodoOrg struct {
	markdown.TodoMarkdown
}

// ToOrgEntry converts an event to Org mode format
func (eo EventOrg) ToOrgEntry() string {
	return eo.ToOrgEntryWithOptions(false, false, false, false, false)
}

// ToOrgEntryWithOptions converts an event to Org mode format with formatting options
func (eo EventOrg) ToOrgEntryWithOptions(useHashtags, ignoreDescriptions, useCheckboxes, useCalendarTags, useObsidianEmojis bool) string {
	var sb strings.Builder

	// Add UID as comment for deduplication
	if eo.UID != "" {
		sb.WriteString(fmt.Sprintf("# uid:%s\n", eo.UID))
	}

	// Determine TODO state based on completion
	todoState := "TODO"
	if useCheckboxes && eo.IsPastEvent() {
		todoState = "DONE"
	}

	// Start with headline (event title)
	if useCheckboxes {
		sb.WriteString(fmt.Sprintf("** %s %s", todoState, eo.Title))
	} else {
		sb.WriteString(fmt.Sprintf("** %s", eo.Title))
	}

	// Add tags
	var tags []string
	if useHashtags {
		tags = append(tags, "event")
	}
	if useCalendarTags && eo.CalendarOriginal != "" {
		calendarTag := sanitizeForOrgTag(eo.CalendarOriginal)
		if calendarTag != "" {
			tags = append(tags, calendarTag)
		}
	}
	if len(tags) > 0 {
		sb.WriteString(fmt.Sprintf(" :%s:", strings.Join(tags, ":")))
	}
	sb.WriteString("\n")

	// Add scheduling information
	if !eo.StartTime.IsZero() {
		if eo.AllDay {
			// All-day event
			sb.WriteString(fmt.Sprintf("   SCHEDULED: <%s>\n", eo.StartTime.Format("2006-01-02 Mon")))
		} else {
			// Timed event with time range
			if !eo.EndTime.IsZero() && !eo.EndTime.Equal(eo.StartTime) {
				startDate := eo.StartTime.Format("2006-01-02 Mon 15:04")
				endTime := eo.EndTime.Format("15:04")
				// Check if event spans multiple days
				if eo.StartTime.Format("2006-01-02") != eo.EndTime.Format("2006-01-02") {
					endDate := eo.EndTime.Format("2006-01-02 Mon 15:04")
					sb.WriteString(fmt.Sprintf("   SCHEDULED: <%s>--<%s>\n", startDate, endDate))
				} else {
					sb.WriteString(fmt.Sprintf("   SCHEDULED: <%s-%s>\n", startDate, endTime))
				}
			} else {
				sb.WriteString(fmt.Sprintf("   SCHEDULED: <%s>\n", eo.StartTime.Format("2006-01-02 Mon 15:04")))
			}
		}
	}

	// Add properties drawer
	sb.WriteString("   :PROPERTIES:\n")
	if eo.UID != "" {
		sb.WriteString(fmt.Sprintf("   :ID: %s\n", eo.UID))
	}
	if eo.Location != "" {
		sb.WriteString(fmt.Sprintf("   :LOCATION: %s\n", eo.Location))
	}
	if eo.Calendar != "" {
		sb.WriteString(fmt.Sprintf("   :CALENDAR: %s\n", eo.Calendar))
	}
	if eo.Status != "" {
		sb.WriteString(fmt.Sprintf("   :STATUS: %s\n", eo.Status))
	}
	if len(eo.Categories) > 0 {
		sb.WriteString(fmt.Sprintf("   :CATEGORIES: %s\n", strings.Join(eo.Categories, ", ")))
	}
	sb.WriteString("   :END:\n")

	// Add description if not ignored
	if !ignoreDescriptions && eo.Description != "" {
		sb.WriteString(fmt.Sprintf("   %s\n", strings.ReplaceAll(eo.Description, "\n", "\n   ")))
	}

	sb.WriteString("\n")
	return sb.String()
}

// ToOrgEntry converts a todo to Org mode format
func (to TodoOrg) ToOrgEntry() string {
	return to.ToOrgEntryWithOptions(false, false, false, false)
}

// ToOrgEntryWithOptions converts a todo to Org mode format with formatting options
func (to TodoOrg) ToOrgEntryWithOptions(useDueDateEmoji, useHashtags, ignoreDescriptions, useCalendarTags bool) string {
	var sb strings.Builder

	// Add UID as comment for deduplication
	if to.UID != "" {
		sb.WriteString(fmt.Sprintf("# uid:%s\n", to.UID))
	}

	// Determine TODO state
	todoState := "TODO"
	if to.Completed {
		todoState = "DONE"
	}

	// Start with headline (task title)
	sb.WriteString(fmt.Sprintf("** %s %s", todoState, to.Title))

	// Add priority if set
	if to.Priority != "" && to.Priority != "0" {
		// Org mode priorities: [#A], [#B], [#C]
		// Map iCal priorities (1-9) to Org priorities
		var orgPriority string
		switch to.Priority {
		case "1", "2", "3":
			orgPriority = "[#A]"
		case "4", "5", "6":
			orgPriority = "[#B]"
		case "7", "8", "9":
			orgPriority = "[#C]"
		}
		if orgPriority != "" {
			sb.WriteString(" " + orgPriority)
		}
	}

	// Add tags
	var tags []string
	if useHashtags {
		tags = append(tags, "task")
	}
	if useCalendarTags && to.CalendarOriginal != "" {
		calendarTag := sanitizeForOrgTag(to.CalendarOriginal)
		if calendarTag != "" {
			tags = append(tags, calendarTag)
		}
	}
	if len(tags) > 0 {
		sb.WriteString(fmt.Sprintf(" :%s:", strings.Join(tags, ":")))
	}
	sb.WriteString("\n")

	// Add deadline if set
	if !to.DueDate.IsZero() {
		sb.WriteString(fmt.Sprintf("   DEADLINE: <%s>\n", to.DueDate.Format("2006-01-02 Mon")))
	}

	// Add properties drawer
	sb.WriteString("   :PROPERTIES:\n")
	if to.UID != "" {
		sb.WriteString(fmt.Sprintf("   :ID: %s\n", to.UID))
	}
	if to.Calendar != "" {
		sb.WriteString(fmt.Sprintf("   :CALENDAR: %s\n", to.Calendar))
	}
	if to.Status != "" {
		sb.WriteString(fmt.Sprintf("   :STATUS: %s\n", to.Status))
	}
	if to.Priority != "" && to.Priority != "0" {
		sb.WriteString(fmt.Sprintf("   :PRIORITY: %s\n", to.Priority))
	}
	if len(to.Categories) > 0 {
		sb.WriteString(fmt.Sprintf("   :CATEGORIES: %s\n", strings.Join(to.Categories, ", ")))
	}
	sb.WriteString("   :END:\n")

	// Add description if not ignored
	if !ignoreDescriptions && to.Description != "" {
		sb.WriteString(fmt.Sprintf("   %s\n", strings.ReplaceAll(to.Description, "\n", "\n   ")))
	}

	sb.WriteString("\n")
	return sb.String()
}

// sanitizeForOrgTag converts a string to a valid Org mode tag
func sanitizeForOrgTag(input string) string {
	// Org tags can contain letters, numbers, underscores, and @ symbols
	tag := strings.ToLower(input)
	tag = strings.ReplaceAll(tag, " ", "_")
	tag = strings.ReplaceAll(tag, "-", "_")

	// Remove invalid characters
	var cleanTag strings.Builder
	for _, r := range tag {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '@' {
			cleanTag.WriteRune(r)
		}
	}

	result := cleanTag.String()
	// Trim leading/trailing underscores
	result = strings.Trim(result, "_")

	if result == "" || len(result) < 2 {
		return ""
	}

	return result
}

// ExistingOrgContent represents parsed content from an existing Org file
type ExistingOrgContent struct {
	Title           string
	AllDayEvents    []string
	ScheduledEvents []string
	Tasks           []string
	OtherContent    []string
	Properties      map[string]string
	CustomSections  map[string][]string
}

// parseExistingOrgFile reads and parses an existing Org file
func parseExistingOrgFile(filename string) (*ExistingOrgContent, error) {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open existing file %s: %v", filename, err)
	}
	defer file.Close()

	content := &ExistingOrgContent{
		Properties:     make(map[string]string),
		CustomSections: make(map[string][]string),
	}

	scanner := bufio.NewScanner(file)
	currentSection := ""
	var currentItems []string
	var currentEntry strings.Builder
	inEntry := false

	for scanner.Scan() {
		line := scanner.Text()

		// Check for top-level title
		if strings.HasPrefix(line, "#+TITLE:") {
			content.Title = strings.TrimSpace(strings.TrimPrefix(line, "#+TITLE:"))
			continue
		}

		// Check for section headers (level 1 headings)
		if strings.HasPrefix(line, "* ") && !strings.HasPrefix(line, "** ") {
			// Save previous section
			if inEntry && currentEntry.Len() > 0 {
				item := currentEntry.String()
				currentItems = append(currentItems, item)
				currentEntry.Reset()
				inEntry = false
			}
			saveOrgSection(content, currentSection, currentItems)

			// Start new section
			sectionTitle := strings.TrimSpace(strings.TrimPrefix(line, "* "))
			switch sectionTitle {
			case "All Day Events":
				currentSection = "allday"
			case "Scheduled Events":
				currentSection = "scheduled"
			case "Tasks":
				currentSection = "tasks"
			default:
				currentSection = line
			}
			currentItems = []string{}
			continue
		}

		// Check for entry headers (level 2 headings)
		if strings.HasPrefix(line, "** ") {
			// Save previous entry
			if inEntry && currentEntry.Len() > 0 {
				item := currentEntry.String()
				currentItems = append(currentItems, item)
				currentEntry.Reset()
			}

			// Start new entry
			inEntry = true
			currentEntry.WriteString(line + "\n")
			continue
		}

		// If we're in an entry, add the line
		if inEntry {
			currentEntry.WriteString(line + "\n")
		} else if currentSection == "" && strings.TrimSpace(line) != "" {
			// Content before any section headers
			content.OtherContent = append(content.OtherContent, line)
		}
	}

	// Save the last entry and section
	if inEntry && currentEntry.Len() > 0 {
		item := currentEntry.String()
		currentItems = append(currentItems, item)
	}
	saveOrgSection(content, currentSection, currentItems)

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file %s: %v", filename, err)
	}

	return content, nil
}

// saveOrgSection saves items to the appropriate section
func saveOrgSection(content *ExistingOrgContent, section string, items []string) {
	switch section {
	case "allday":
		content.AllDayEvents = items
	case "scheduled":
		content.ScheduledEvents = items
	case "tasks":
		content.Tasks = items
	default:
		if section != "" {
			if strings.HasPrefix(section, "* ") {
				content.CustomSections[section] = items
			} else {
				content.OtherContent = append(content.OtherContent, items...)
			}
		}
	}
}

// extractOrgUID extracts the UID from an Org entry comment
func extractOrgUID(entry string) string {
	uidPattern := regexp.MustCompile(`^#\s*uid:(\S+)`)
	lines := strings.Split(entry, "\n")
	for _, line := range lines {
		matches := uidPattern.FindStringSubmatch(line)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

// smartDeduplicateOrg removes duplicate entries based on UID
func smartDeduplicateOrg(items []string) []string {
	uidVersions := make(map[string]string)
	var result []string
	seenUIDs := make(map[string]bool)

	for _, item := range items {
		normalized := strings.TrimSpace(item)
		if normalized == "" {
			continue
		}

		uid := extractOrgUID(normalized)
		if uid != "" {
			uidVersions[uid] = normalized
		}
	}

	for _, item := range items {
		normalized := strings.TrimSpace(item)
		if normalized == "" {
			continue
		}

		uid := extractOrgUID(normalized)
		if uid != "" {
			if !seenUIDs[uid] {
				seenUIDs[uid] = true
				result = append(result, uidVersions[uid])
			}
		} else {
			result = append(result, normalized)
		}
	}

	return result
}

// generateMergedOrgContent creates merged Org file content
func generateMergedOrgContent(date string, existingContent *ExistingOrgContent, newAllDay, newScheduled, newTasks []string) string {
	var sb strings.Builder

	// Generate title
	if existingContent != nil && existingContent.Title != "" {
		sb.WriteString(fmt.Sprintf("#+TITLE: %s\n\n", existingContent.Title))
	} else {
		if date == "0001-01-01" {
			sb.WriteString("#+TITLE: Events (Date TBD)\n\n")
		} else {
			parsedDate, _ := time.Parse("2006-01-02", date)
			sb.WriteString(fmt.Sprintf("#+TITLE: %s\n\n", parsedDate.Format("Monday, January 2, 2006")))
		}
	}

	// Add other content from existing file
	if existingContent != nil && len(existingContent.OtherContent) > 0 {
		for _, line := range existingContent.OtherContent {
			sb.WriteString(line + "\n")
		}
		sb.WriteString("\n")
	}

	// Merge and deduplicate all-day events
	var allDayItems []string
	if existingContent != nil {
		allDayItems = append(allDayItems, existingContent.AllDayEvents...)
	}
	allDayItems = append(allDayItems, newAllDay...)
	allDayItems = smartDeduplicateOrg(allDayItems)

	// Merge and deduplicate scheduled events
	var scheduledItems []string
	if existingContent != nil {
		scheduledItems = append(scheduledItems, existingContent.ScheduledEvents...)
	}
	scheduledItems = append(scheduledItems, newScheduled...)
	scheduledItems = smartDeduplicateOrg(scheduledItems)

	// Merge and deduplicate tasks
	var taskItems []string
	if existingContent != nil {
		taskItems = append(taskItems, existingContent.Tasks...)
	}
	taskItems = append(taskItems, newTasks...)
	taskItems = smartDeduplicateOrg(taskItems)

	// Add custom sections
	if existingContent != nil {
		for sectionHeader, sectionItems := range existingContent.CustomSections {
			if sectionHeader == "* All Day Events" || sectionHeader == "* Scheduled Events" || sectionHeader == "* Tasks" {
				continue
			}
			if len(sectionItems) > 0 {
				sb.WriteString(sectionHeader + "\n")
				for _, item := range sectionItems {
					sb.WriteString(item)
				}
				sb.WriteString("\n")
			}
		}
	}

	// Add all-day events section
	if len(allDayItems) > 0 {
		sb.WriteString("* All Day Events\n\n")
		for _, item := range allDayItems {
			sb.WriteString(item)
		}
	}

	// Add scheduled events section
	if len(scheduledItems) > 0 {
		sb.WriteString("* Scheduled Events\n\n")
		for _, item := range scheduledItems {
			sb.WriteString(item)
		}
	}

	// Add tasks section
	if len(taskItems) > 0 {
		sb.WriteString("* Tasks\n\n")
		for _, item := range taskItems {
			sb.WriteString(item)
		}
	}

	return sb.String()
}

// GenerateTodoOrgFile creates a todo.org file for tasks without due dates
func GenerateTodoOrgFile(outputDir string, todos []markdown.TodoMarkdown, useDueDateEmoji, useHashtags, ignoreDescriptions, useCalendarTags bool, progressCallback markdown.ProgressCallback) error {
	if progressCallback != nil {
		progressCallback(fmt.Sprintf("Generating todo.org with %d tasks", len(todos)), 0, 1)
	}

	todoPath := filepath.Join(outputDir, "todo.org")

	// Parse existing todo.org file if it exists
	existingContent, err := parseExistingOrgFile(todoPath)
	if err != nil {
		return fmt.Errorf("failed to parse existing todo.org file: %v", err)
	}

	// Convert tasks to Org strings
	var newTasks []string
	for _, task := range todos {
		todoOrg := TodoOrg{TodoMarkdown: task}
		newTasks = append(newTasks, strings.TrimSpace(todoOrg.ToOrgEntryWithOptions(useDueDateEmoji, useHashtags, ignoreDescriptions, useCalendarTags)))
	}

	// Generate merged content
	var sb strings.Builder
	if existingContent != nil && existingContent.Title != "" {
		sb.WriteString(fmt.Sprintf("#+TITLE: %s\n\n", existingContent.Title))
	} else {
		sb.WriteString("#+TITLE: Todo\n\n")
	}

	// Merge tasks
	var taskItems []string
	if existingContent != nil {
		taskItems = append(taskItems, existingContent.Tasks...)
	}
	taskItems = append(taskItems, newTasks...)
	taskItems = smartDeduplicateOrg(taskItems)

	if len(taskItems) > 0 {
		sb.WriteString("* Tasks\n\n")
		for _, item := range taskItems {
			sb.WriteString(item)
		}
	}

	// Write to file
	if err := writeOrgToFile(todoPath, sb.String()); err != nil {
		return fmt.Errorf("failed to write todo.org file: %v", err)
	}

	if progressCallback != nil {
		progressCallback("Generated todo.org", 1, 1)
	}

	return nil
}

// writeOrgToFile writes content to an Org file
func writeOrgToFile(filename, content string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

// GenerateDailyOrgFiles creates daily Org files for events and tasks
func GenerateDailyOrgFiles(outputDir string, events []markdown.EventMarkdown, tasks []markdown.TodoMarkdown, useDueDateEmoji, useHashtags, ignoreDescriptions, eventCheckboxes, useCalendarTags, useObsidianEmojis bool, progressCallback markdown.ProgressCallback) error {
	// Group events by date
	eventsByDate := make(map[string][]markdown.EventMarkdown)
	tasksByDate := make(map[string][]markdown.TodoMarkdown)
	var noDueDateTasks []markdown.TodoMarkdown

	for _, event := range events {
		var dateKey string
		if event.StartTime.IsZero() {
			dateKey = "0001-01-01"
		} else {
			dateKey = event.StartTime.Format("2006-01-02")
		}
		eventsByDate[dateKey] = append(eventsByDate[dateKey], event)
	}

	for _, task := range tasks {
		if task.DueDate.IsZero() {
			noDueDateTasks = append(noDueDateTasks, task)
		} else {
			dateKey := task.DueDate.Format("2006-01-02")
			tasksByDate[dateKey] = append(tasksByDate[dateKey], task)
		}
	}

	// Get all unique dates
	allDates := make(map[string]bool)
	for date := range eventsByDate {
		allDates[date] = true
	}
	for date := range tasksByDate {
		allDates[date] = true
	}

	datesList := make([]string, 0, len(allDates))
	for date := range allDates {
		datesList = append(datesList, date)
	}

	if progressCallback != nil {
		progressCallback(fmt.Sprintf("Generating %d daily Org files", len(datesList)), 0, len(datesList))
	}

	// Create a file for each date
	for i, date := range datesList {
		if progressCallback != nil {
			if date == "0001-01-01" {
				progressCallback("Processing items with no date", i+1, len(datesList))
			} else {
				progressCallback(fmt.Sprintf("Processing %s", date), i+1, len(datesList))
			}
		}

		dayEvents := eventsByDate[date]
		dayTasks := tasksByDate[date]

		// Sort events by start time
		var allDayEvents, timedEvents []markdown.EventMarkdown
		for _, event := range dayEvents {
			if event.AllDay || event.StartTime.IsZero() {
				allDayEvents = append(allDayEvents, event)
			} else {
				timedEvents = append(timedEvents, event)
			}
		}

		// Simple sort by start time
		for i := 0; i < len(timedEvents)-1; i++ {
			for j := i + 1; j < len(timedEvents); j++ {
				if timedEvents[i].StartTime.After(timedEvents[j].StartTime) {
					timedEvents[i], timedEvents[j] = timedEvents[j], timedEvents[i]
				}
			}
		}

		// Convert to Org format
		var newAllDayEvents, newScheduledEvents, newTasks []string

		for _, event := range allDayEvents {
			eventOrg := EventOrg{EventMarkdown: event}
			newAllDayEvents = append(newAllDayEvents, strings.TrimSpace(eventOrg.ToOrgEntryWithOptions(useHashtags, ignoreDescriptions, eventCheckboxes, useCalendarTags, useObsidianEmojis)))
		}

		for _, event := range timedEvents {
			eventOrg := EventOrg{EventMarkdown: event}
			newScheduledEvents = append(newScheduledEvents, strings.TrimSpace(eventOrg.ToOrgEntryWithOptions(useHashtags, ignoreDescriptions, eventCheckboxes, useCalendarTags, useObsidianEmojis)))
		}

		for _, task := range dayTasks {
			todoOrg := TodoOrg{TodoMarkdown: task}
			newTasks = append(newTasks, strings.TrimSpace(todoOrg.ToOrgEntryWithOptions(useDueDateEmoji, useHashtags, ignoreDescriptions, useCalendarTags)))
		}

		// Create directory structure
		var dirPath, filename string
		if date == "0001-01-01" {
			dirPath = filepath.Join(outputDir, "0001", "01")
		} else {
			parsedDate, _ := time.Parse("2006-01-02", date)
			dirPath = filepath.Join(outputDir, parsedDate.Format("2006"), parsedDate.Format("01"))
		}

		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dirPath, err)
		}

		filename = filepath.Join(dirPath, fmt.Sprintf("%s.org", date))

		// Parse existing file if it exists
		existingContent, err := parseExistingOrgFile(filename)
		if err != nil {
			return fmt.Errorf("failed to parse existing file %s: %v", filename, err)
		}

		// Generate merged content
		mergedContent := generateMergedOrgContent(date, existingContent, newAllDayEvents, newScheduledEvents, newTasks)

		// Write to file
		if err := writeOrgToFile(filename, mergedContent); err != nil {
			return fmt.Errorf("failed to write file %s: %v", filename, err)
		}
	}

	// Generate todo.org file for tasks without due dates
	if len(noDueDateTasks) > 0 {
		if err := GenerateTodoOrgFile(outputDir, noDueDateTasks, useDueDateEmoji, useHashtags, ignoreDescriptions, useCalendarTags, progressCallback); err != nil {
			return fmt.Errorf("failed to generate todo.org file: %v", err)
		}
	}

	return nil
}

// GenerateSingleOrgFile creates a single Org file with all events and tasks
func GenerateSingleOrgFile(outputPath string, events []markdown.EventMarkdown, tasks []markdown.TodoMarkdown, useDueDateEmoji, useHashtags, ignoreDescriptions, eventCheckboxes, useCalendarTags, useObsidianEmojis bool, progressCallback markdown.ProgressCallback) error {
	if progressCallback != nil {
		progressCallback("Generating single Org file output", 0, 1)
	}

	// Group events and tasks by date
	eventsByDate := make(map[string][]markdown.EventMarkdown)
	tasksByDate := make(map[string][]markdown.TodoMarkdown)
	var noDueDateTasks []markdown.TodoMarkdown

	for _, event := range events {
		var dateKey string
		if event.StartTime.IsZero() {
			dateKey = "0001-01-01"
		} else {
			dateKey = event.StartTime.Format("2006-01-02")
		}
		eventsByDate[dateKey] = append(eventsByDate[dateKey], event)
	}

	for _, task := range tasks {
		if task.DueDate.IsZero() {
			noDueDateTasks = append(noDueDateTasks, task)
		} else {
			dateKey := task.DueDate.Format("2006-01-02")
			tasksByDate[dateKey] = append(tasksByDate[dateKey], task)
		}
	}

	// Get all unique dates and sort them
	allDates := make(map[string]bool)
	for date := range eventsByDate {
		allDates[date] = true
	}
	for date := range tasksByDate {
		allDates[date] = true
	}

	datesList := make([]string, 0, len(allDates))
	for date := range allDates {
		datesList = append(datesList, date)
	}

	// Sort dates chronologically
	for i := 0; i < len(datesList)-1; i++ {
		for j := i + 1; j < len(datesList); j++ {
			if datesList[i] > datesList[j] {
				datesList[i], datesList[j] = datesList[j], datesList[i]
			}
		}
	}

	// Build the single file content
	var sb strings.Builder

	sb.WriteString("#+TITLE: Calendar Events and Tasks\n")
	sb.WriteString(fmt.Sprintf("#+DATE: %s\n\n", time.Now().Format("2006-01-02")))

	// Process each date
	for i, date := range datesList {
		if progressCallback != nil {
			progressCallback(fmt.Sprintf("Processing %s", date), i+1, len(datesList))
		}

		dayEvents := eventsByDate[date]
		dayTasks := tasksByDate[date]

		if len(dayEvents) == 0 && len(dayTasks) == 0 {
			continue
		}

		// Add date header
		if date == "0001-01-01" {
			sb.WriteString("* Events (Date TBD)\n\n")
		} else {
			parsedDate, _ := time.Parse("2006-01-02", date)
			sb.WriteString(fmt.Sprintf("* %s\n\n", parsedDate.Format("Monday, January 2, 2006")))
		}

		// Sort events
		var allDayEvents, timedEvents []markdown.EventMarkdown
		for _, event := range dayEvents {
			if event.AllDay || event.StartTime.IsZero() {
				allDayEvents = append(allDayEvents, event)
			} else {
				timedEvents = append(timedEvents, event)
			}
		}

		for i := 0; i < len(timedEvents)-1; i++ {
			for j := i + 1; j < len(timedEvents); j++ {
				if timedEvents[i].StartTime.After(timedEvents[j].StartTime) {
					timedEvents[i], timedEvents[j] = timedEvents[j], timedEvents[i]
				}
			}
		}

		// Add all-day events
		if len(allDayEvents) > 0 {
			sb.WriteString("** All Day Events\n\n")
			for _, event := range allDayEvents {
				eventOrg := EventOrg{EventMarkdown: event}
				// Increase heading level for sub-entries
				entry := eventOrg.ToOrgEntryWithOptions(useHashtags, ignoreDescriptions, eventCheckboxes, useCalendarTags, useObsidianEmojis)
				entry = strings.ReplaceAll(entry, "** ", "*** ")
				sb.WriteString(entry)
			}
		}

		// Add scheduled events
		if len(timedEvents) > 0 {
			sb.WriteString("** Scheduled Events\n\n")
			for _, event := range timedEvents {
				eventOrg := EventOrg{EventMarkdown: event}
				entry := eventOrg.ToOrgEntryWithOptions(useHashtags, ignoreDescriptions, eventCheckboxes, useCalendarTags, useObsidianEmojis)
				entry = strings.ReplaceAll(entry, "** ", "*** ")
				sb.WriteString(entry)
			}
		}

		// Add tasks
		if len(dayTasks) > 0 {
			sb.WriteString("** Tasks\n\n")
			for _, task := range dayTasks {
				todoOrg := TodoOrg{TodoMarkdown: task}
				entry := todoOrg.ToOrgEntryWithOptions(useDueDateEmoji, useHashtags, ignoreDescriptions, useCalendarTags)
				entry = strings.ReplaceAll(entry, "** ", "*** ")
				sb.WriteString(entry)
			}
		}
	}

	// Add tasks without due dates
	if len(noDueDateTasks) > 0 {
		sb.WriteString("* Tasks Without Due Date\n\n")
		for _, task := range noDueDateTasks {
			todoOrg := TodoOrg{TodoMarkdown: task}
			entry := todoOrg.ToOrgEntryWithOptions(useDueDateEmoji, useHashtags, ignoreDescriptions, useCalendarTags)
			entry = strings.ReplaceAll(entry, "** ", "*** ")
			sb.WriteString(entry)
		}
	}

	// Write to file
	if err := writeOrgToFile(outputPath, sb.String()); err != nil {
		return fmt.Errorf("failed to write single Org file: %v", err)
	}

	if progressCallback != nil {
		progressCallback("Generated single Org file", 1, 1)
	}

	return nil
}
