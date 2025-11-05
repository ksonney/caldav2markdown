package diary

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

// EventDiary represents an event in Emacs diary format (wraps markdown.EventMarkdown)
type EventDiary struct {
	markdown.EventMarkdown
}

// TodoDiary represents a todo in Emacs diary format (wraps markdown.TodoMarkdown)
type TodoDiary struct {
	markdown.TodoMarkdown
}

// ToDiaryEntry converts an event to Emacs diary format
// Emacs diary format documentation: https://www.gnu.org/software/emacs/manual/html_node/emacs/Diary.html
//
// Basic format examples:
//   12/22/2024 Dental appointment at 10:30am
//   Oct 31, 2024 Halloween party
//   %%(diary-float t 1 1) First Monday of every month
//
// With time ranges:
//   12/22/2024 10:30am-11:30am Meeting with team
//
// With categories/tags (custom extension):
//   12/22/2024 10:30am-11:30am Meeting [Work] #event
func (ed EventDiary) ToDiaryEntry() string {
	return ed.ToDiaryEntryWithOptions(false, false, false, false)
}

// ToDiaryEntryWithOptions converts an event to Emacs diary format with formatting options
func (ed EventDiary) ToDiaryEntryWithOptions(useHashtags, ignoreDescriptions, useCalendarTags, markCompleted bool) string {
	var sb strings.Builder

	// Handle events with zero time
	if ed.StartTime.IsZero() {
		// For events without dates, create a special entry
		sb.WriteString(fmt.Sprintf("%%(diary-remind '(diary-entry \"TBD\") 0) %s", ed.Title))
		if ed.Location != "" {
			sb.WriteString(fmt.Sprintf(" @ %s", ed.Location))
		}
		if ed.Calendar != "" {
			sb.WriteString(fmt.Sprintf(" [%s]", ed.Calendar))
		}
		if useHashtags {
			sb.WriteString(" #event")
			if useCalendarTags && ed.CalendarOriginal != "" {
				calendarTag := sanitizeForDiaryTag(ed.CalendarOriginal)
				if calendarTag != "" {
					sb.WriteString(fmt.Sprintf(" #%s", calendarTag))
				}
			}
		}
		if !ignoreDescriptions && ed.Description != "" {
			sb.WriteString(fmt.Sprintf("\n  %s", strings.ReplaceAll(ed.Description, "\n", "\n  ")))
		}
		sb.WriteString(fmt.Sprintf(" {uid:%s}\n", ed.UID))
		return sb.String()
	}

	// Format date in Emacs diary preferred format: MM/DD/YYYY
	dateStr := ed.StartTime.Format("1/2/2006")

	// Start entry with date
	sb.WriteString(dateStr)
	sb.WriteString(" ")

	// Add completion marker if requested and event is past
	if markCompleted && ed.IsPastEvent() {
		sb.WriteString("[DONE] ")
	}

	// Add time information if not all-day
	if !ed.AllDay {
		startTimeStr := ed.StartTime.Format("3:04pm")
		if !ed.EndTime.IsZero() && !ed.EndTime.Equal(ed.StartTime) {
			endTimeStr := ed.EndTime.Format("3:04pm")
			// Check if multi-day event
			if ed.StartTime.Format("2006-01-02") != ed.EndTime.Format("2006-01-02") {
				// Multi-day event
				endDateStr := ed.EndTime.Format("1/2/2006")
				sb.WriteString(fmt.Sprintf("%s-%s %s ", startTimeStr, endDateStr, endTimeStr))
			} else {
				// Same day event with time range
				sb.WriteString(fmt.Sprintf("%s-%s ", startTimeStr, endTimeStr))
			}
		} else {
			sb.WriteString(fmt.Sprintf("%s ", startTimeStr))
		}
	}

	// Add title
	sb.WriteString(ed.Title)

	// Add location
	if ed.Location != "" {
		sb.WriteString(fmt.Sprintf(" @ %s", ed.Location))
	}

	// Add status if present
	if ed.Status != "" && ed.Status != "CONFIRMED" {
		sb.WriteString(fmt.Sprintf(" (%s)", ed.Status))
	}

	// Add calendar name
	if ed.Calendar != "" {
		sb.WriteString(fmt.Sprintf(" [%s]", ed.Calendar))
	}

	// Add hashtags
	if useHashtags {
		sb.WriteString(" #event")
		if useCalendarTags && ed.CalendarOriginal != "" {
			calendarTag := sanitizeForDiaryTag(ed.CalendarOriginal)
			if calendarTag != "" {
				sb.WriteString(fmt.Sprintf(" #%s", calendarTag))
			}
		}
	}

	// Add categories as tags
	if len(ed.Categories) > 0 {
		for _, cat := range ed.Categories {
			if cat != "" {
				sb.WriteString(fmt.Sprintf(" #%s", sanitizeForDiaryTag(cat)))
			}
		}
	}

	// Add UID for deduplication (hidden in braces)
	sb.WriteString(fmt.Sprintf(" {uid:%s}", ed.UID))

	// Add description if not ignored
	if !ignoreDescriptions && ed.Description != "" {
		sb.WriteString(fmt.Sprintf("\n  %s", strings.ReplaceAll(ed.Description, "\n", "\n  ")))
	}

	sb.WriteString("\n")
	return sb.String()
}

// ToDiaryEntry converts a todo to Emacs diary format
func (td TodoDiary) ToDiaryEntry() string {
	return td.ToDiaryEntryWithOptions(false, false, false)
}

// ToDiaryEntryWithOptions converts a todo to Emacs diary format with formatting options
// Todos in diary format use the DEADLINE notation or appear on their due date
func (td TodoDiary) ToDiaryEntryWithOptions(useHashtags, ignoreDescriptions, useCalendarTags bool) string {
	var sb strings.Builder

	// Handle todos without due dates
	if td.DueDate.IsZero() {
		sb.WriteString("%%(diary-entry \"TODO\") ")
		if td.Completed {
			sb.WriteString("[DONE] ")
		}
		sb.WriteString(td.Title)

		// Add priority
		if td.Priority != "" && td.Priority != "0" {
			sb.WriteString(fmt.Sprintf(" [Priority:%s]", td.Priority))
		}

		// Add status
		if td.Status != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", td.Status))
		}

		// Add calendar name
		if td.Calendar != "" {
			sb.WriteString(fmt.Sprintf(" [%s]", td.Calendar))
		}

		// Add hashtags
		if useHashtags {
			sb.WriteString(" #task")
			if useCalendarTags && td.CalendarOriginal != "" {
				calendarTag := sanitizeForDiaryTag(td.CalendarOriginal)
				if calendarTag != "" {
					sb.WriteString(fmt.Sprintf(" #%s", calendarTag))
				}
			}
		}

		// Add categories as tags
		if len(td.Categories) > 0 {
			for _, cat := range td.Categories {
				if cat != "" {
					sb.WriteString(fmt.Sprintf(" #%s", sanitizeForDiaryTag(cat)))
				}
			}
		}

		sb.WriteString(fmt.Sprintf(" {uid:%s}", td.UID))

		if !ignoreDescriptions && td.Description != "" {
			sb.WriteString(fmt.Sprintf("\n  %s", strings.ReplaceAll(td.Description, "\n", "\n  ")))
		}

		sb.WriteString("\n")
		return sb.String()
	}

	// Format date
	dateStr := td.DueDate.Format("1/2/2006")

	sb.WriteString(dateStr)
	sb.WriteString(" ")

	// Add completion marker
	if td.Completed {
		sb.WriteString("[DONE] ")
	}

	sb.WriteString(td.Title)

	// Add priority
	if td.Priority != "" && td.Priority != "0" {
		sb.WriteString(fmt.Sprintf(" [Priority:%s]", td.Priority))
	}

	// Add status
	if td.Status != "" && td.Status != "NEEDS-ACTION" {
		sb.WriteString(fmt.Sprintf(" (%s)", td.Status))
	}

	// Add calendar name
	if td.Calendar != "" {
		sb.WriteString(fmt.Sprintf(" [%s]", td.Calendar))
	}

	// Add hashtags
	if useHashtags {
		sb.WriteString(" #task")
		if useCalendarTags && td.CalendarOriginal != "" {
			calendarTag := sanitizeForDiaryTag(td.CalendarOriginal)
			if calendarTag != "" {
				sb.WriteString(fmt.Sprintf(" #%s", calendarTag))
			}
		}
	}

	// Add categories as tags
	if len(td.Categories) > 0 {
		for _, cat := range td.Categories {
			if cat != "" {
				sb.WriteString(fmt.Sprintf(" #%s", sanitizeForDiaryTag(cat)))
			}
		}
	}

	sb.WriteString(fmt.Sprintf(" {uid:%s}", td.UID))

	if !ignoreDescriptions && td.Description != "" {
		sb.WriteString(fmt.Sprintf("\n  %s", strings.ReplaceAll(td.Description, "\n", "\n  ")))
	}

	sb.WriteString("\n")
	return sb.String()
}

// sanitizeForDiaryTag converts a string to a valid tag for diary entries
func sanitizeForDiaryTag(input string) string {
	// Convert to lowercase and replace spaces/special chars with hyphens
	tag := strings.ToLower(input)
	tag = strings.ReplaceAll(tag, " ", "-")
	tag = strings.ReplaceAll(tag, "_", "-")

	// Remove invalid characters (keep alphanumeric and hyphens)
	var cleanTag strings.Builder
	for _, r := range tag {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			cleanTag.WriteRune(r)
		}
	}

	result := cleanTag.String()
	// Collapse multiple hyphens
	re := regexp.MustCompile(`-+`)
	result = re.ReplaceAllString(result, "-")
	// Trim leading/trailing hyphens
	result = strings.Trim(result, "-")

	if result == "" || len(result) < 2 {
		return ""
	}

	return result
}

// ExistingDiaryContent represents parsed content from an existing diary file
type ExistingDiaryContent struct {
	Header       []string
	Entries      []string
	CustomContent []string
}

// parseExistingDiaryFile reads and parses an existing diary file
func parseExistingDiaryFile(filename string) (*ExistingDiaryContent, error) {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open existing file %s: %v", filename, err)
	}
	defer file.Close()

	content := &ExistingDiaryContent{
		Header:       []string{},
		Entries:      []string{},
		CustomContent: []string{},
	}

	scanner := bufio.NewScanner(file)
	inHeader := true
	var currentEntry strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// Check for diary entry patterns
		// Diary entries start with dates or special %% expressions
		isDiaryEntry := false
		if strings.HasPrefix(line, "%%") {
			isDiaryEntry = true
		} else if matched, _ := regexp.MatchString(`^\d{1,2}/\d{1,2}/\d{4}`, line); matched {
			isDiaryEntry = true
		}

		if isDiaryEntry {
			// Save previous entry if any
			if currentEntry.Len() > 0 {
				content.Entries = append(content.Entries, currentEntry.String())
				currentEntry.Reset()
			}
			inHeader = false
			currentEntry.WriteString(line + "\n")
		} else if strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t") {
			// Continuation line (indented description)
			if currentEntry.Len() > 0 {
				currentEntry.WriteString(line + "\n")
			} else if inHeader {
				content.Header = append(content.Header, line)
			} else {
				content.CustomContent = append(content.CustomContent, line)
			}
		} else {
			// Header or custom content
			if inHeader && (strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "") {
				content.Header = append(content.Header, line)
			} else if strings.TrimSpace(line) != "" {
				// Custom content between entries
				if currentEntry.Len() > 0 {
					content.Entries = append(content.Entries, currentEntry.String())
					currentEntry.Reset()
				}
				content.CustomContent = append(content.CustomContent, line)
				inHeader = false
			} else if strings.TrimSpace(line) == "" {
				// Empty line
				if inHeader {
					content.Header = append(content.Header, line)
				} else if currentEntry.Len() > 0 {
					// Part of current entry
					currentEntry.WriteString(line + "\n")
				} else {
					content.CustomContent = append(content.CustomContent, line)
				}
			}
		}
	}

	// Save the last entry
	if currentEntry.Len() > 0 {
		content.Entries = append(content.Entries, currentEntry.String())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file %s: %v", filename, err)
	}

	return content, nil
}

// extractDiaryUID extracts the UID from a diary entry
func extractDiaryUID(entry string) string {
	uidPattern := regexp.MustCompile(`\{uid:(\S+)\}`)
	matches := uidPattern.FindStringSubmatch(entry)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// smartDeduplicateDiary removes duplicate entries based on UID
func smartDeduplicateDiary(entries []string) []string {
	uidVersions := make(map[string]string)
	var result []string
	seenUIDs := make(map[string]bool)

	// First pass: collect all entries by UID
	for _, entry := range entries {
		normalized := strings.TrimSpace(entry)
		if normalized == "" {
			continue
		}

		uid := extractDiaryUID(normalized)
		if uid != "" {
			uidVersions[uid] = normalized
		}
	}

	// Second pass: deduplicate
	for _, entry := range entries {
		normalized := strings.TrimSpace(entry)
		if normalized == "" {
			continue
		}

		uid := extractDiaryUID(normalized)
		if uid != "" {
			if !seenUIDs[uid] {
				seenUIDs[uid] = true
				result = append(result, uidVersions[uid])
			}
		} else {
			// Entries without UIDs are preserved as-is
			result = append(result, normalized)
		}
	}

	return result
}

// generateMergedDiaryContent creates merged diary file content
func generateMergedDiaryContent(existingContent *ExistingDiaryContent, newEntries []string) string {
	var sb strings.Builder

	// Add header
	if existingContent != nil && len(existingContent.Header) > 0 {
		for _, line := range existingContent.Header {
			sb.WriteString(line + "\n")
		}
	} else {
		// Default header for new files
		sb.WriteString("; Emacs Diary File\n")
		sb.WriteString("; Generated by caldav2markdown\n")
		sb.WriteString(";\n")
		sb.WriteString("; Format: MM/DD/YYYY [TIME] TITLE [LOCATION] [CALENDAR] #tags {uid:...}\n")
		sb.WriteString(";\n\n")
	}

	// Merge and deduplicate entries
	var allEntries []string
	if existingContent != nil {
		allEntries = append(allEntries, existingContent.Entries...)
	}
	allEntries = append(allEntries, newEntries...)
	allEntries = smartDeduplicateDiary(allEntries)

	// Sort entries by date (simple chronological sort)
	// Diary entries should be in date order for easier reading
	sortedEntries := sortDiaryEntriesByDate(allEntries)

	// Add all entries
	for _, entry := range sortedEntries {
		if strings.TrimSpace(entry) != "" {
			sb.WriteString(entry)
			if !strings.HasSuffix(entry, "\n") {
				sb.WriteString("\n")
			}
		}
	}

	// Add custom content at the end
	if existingContent != nil && len(existingContent.CustomContent) > 0 {
		sb.WriteString("\n")
		for _, line := range existingContent.CustomContent {
			sb.WriteString(line + "\n")
		}
	}

	return sb.String()
}

// sortDiaryEntriesByDate sorts diary entries chronologically
func sortDiaryEntriesByDate(entries []string) []string {
	type entryWithDate struct {
		entry string
		date  time.Time
		order int // Original order for stable sort
	}

	var withDates []entryWithDate
	datePattern := regexp.MustCompile(`^(\d{1,2})/(\d{1,2})/(\d{4})`)

	for i, entry := range entries {
		matches := datePattern.FindStringSubmatch(entry)
		if len(matches) == 4 {
			// Parse date
			month := matches[1]
			day := matches[2]
			year := matches[3]
			dateStr := fmt.Sprintf("%s/%s/%s", month, day, year)
			if t, err := time.Parse("1/2/2006", dateStr); err == nil {
				withDates = append(withDates, entryWithDate{entry: entry, date: t, order: i})
				continue
			}
		}
		// Entries without dates or special entries go at the beginning
		withDates = append(withDates, entryWithDate{entry: entry, date: time.Time{}, order: i})
	}

	// Simple bubble sort by date
	for i := 0; i < len(withDates)-1; i++ {
		for j := i + 1; j < len(withDates); j++ {
			// Entries without dates come first
			if withDates[i].date.IsZero() && !withDates[j].date.IsZero() {
				continue
			}
			if !withDates[i].date.IsZero() && withDates[j].date.IsZero() {
				withDates[i], withDates[j] = withDates[j], withDates[i]
				continue
			}
			// Both have dates or both don't have dates
			if !withDates[i].date.IsZero() && !withDates[j].date.IsZero() {
				if withDates[i].date.After(withDates[j].date) {
					withDates[i], withDates[j] = withDates[j], withDates[i]
				} else if withDates[i].date.Equal(withDates[j].date) && withDates[i].order > withDates[j].order {
					// Stable sort: preserve original order for same dates
					withDates[i], withDates[j] = withDates[j], withDates[i]
				}
			} else {
				// Both have no dates, preserve original order
				if withDates[i].order > withDates[j].order {
					withDates[i], withDates[j] = withDates[j], withDates[i]
				}
			}
		}
	}

	result := make([]string, len(withDates))
	for i, wd := range withDates {
		result[i] = wd.entry
	}
	return result
}

// writeDiaryToFile writes content to a diary file
func writeDiaryToFile(filename, content string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

// GenerateSingleDiaryFile creates a single diary file with all events and tasks
func GenerateSingleDiaryFile(outputPath string, events []markdown.EventMarkdown, tasks []markdown.TodoMarkdown, useHashtags, ignoreDescriptions, useCalendarTags, markCompleted bool, progressCallback markdown.ProgressCallback) error {
	if progressCallback != nil {
		progressCallback("Generating single diary file output", 0, 1)
	}

	// Convert all events and tasks to diary format
	var newEntries []string

	for _, event := range events {
		eventDiary := EventDiary{EventMarkdown: event}
		entry := strings.TrimSpace(eventDiary.ToDiaryEntryWithOptions(useHashtags, ignoreDescriptions, useCalendarTags, markCompleted))
		if entry != "" {
			newEntries = append(newEntries, entry)
		}
	}

	for _, task := range tasks {
		todoDiary := TodoDiary{TodoMarkdown: task}
		entry := strings.TrimSpace(todoDiary.ToDiaryEntryWithOptions(useHashtags, ignoreDescriptions, useCalendarTags))
		if entry != "" {
			newEntries = append(newEntries, entry)
		}
	}

	// Parse existing file if it exists
	existingContent, err := parseExistingDiaryFile(outputPath)
	if err != nil {
		return fmt.Errorf("failed to parse existing diary file: %v", err)
	}

	// Generate merged content
	mergedContent := generateMergedDiaryContent(existingContent, newEntries)

	// Write to file
	if err := writeDiaryToFile(outputPath, mergedContent); err != nil {
		return fmt.Errorf("failed to write diary file: %v", err)
	}

	if progressCallback != nil {
		progressCallback("Generated single diary file", 1, 1)
	}

	return nil
}

// GenerateDailyDiaryFiles creates separate diary files for each month
// This follows the Emacs convention of having monthly diary files
func GenerateDailyDiaryFiles(outputDir string, events []markdown.EventMarkdown, tasks []markdown.TodoMarkdown, useHashtags, ignoreDescriptions, useCalendarTags, markCompleted bool, progressCallback markdown.ProgressCallback) error {
	// Group events and tasks by month
	entriesByMonth := make(map[string][]string)

	// Process events
	for _, event := range events {
		var monthKey string
		if event.StartTime.IsZero() {
			monthKey = "0001-01" // Special month for undated events
		} else {
			monthKey = event.StartTime.Format("2006-01")
		}

		eventDiary := EventDiary{EventMarkdown: event}
		entry := strings.TrimSpace(eventDiary.ToDiaryEntryWithOptions(useHashtags, ignoreDescriptions, useCalendarTags, markCompleted))
		if entry != "" {
			entriesByMonth[monthKey] = append(entriesByMonth[monthKey], entry)
		}
	}

	// Process tasks
	for _, task := range tasks {
		var monthKey string
		if task.DueDate.IsZero() {
			monthKey = "todo" // Special file for tasks without due dates
		} else {
			monthKey = task.DueDate.Format("2006-01")
		}

		todoDiary := TodoDiary{TodoMarkdown: task}
		entry := strings.TrimSpace(todoDiary.ToDiaryEntryWithOptions(useHashtags, ignoreDescriptions, useCalendarTags))
		if entry != "" {
			entriesByMonth[monthKey] = append(entriesByMonth[monthKey], entry)
		}
	}

	if progressCallback != nil {
		progressCallback(fmt.Sprintf("Generating %d monthly diary files", len(entriesByMonth)), 0, len(entriesByMonth))
	}

	// Create a file for each month
	i := 0
	for monthKey, entries := range entriesByMonth {
		if progressCallback != nil {
			if monthKey == "0001-01" {
				progressCallback("Processing undated entries", i+1, len(entriesByMonth))
			} else if monthKey == "todo" {
				progressCallback("Processing todo entries", i+1, len(entriesByMonth))
			} else {
				progressCallback(fmt.Sprintf("Processing %s", monthKey), i+1, len(entriesByMonth))
			}
		}
		i++

		// Determine file path
		var filename string
		if monthKey == "todo" {
			filename = filepath.Join(outputDir, "todo-diary")
		} else if monthKey == "0001-01" {
			dirPath := filepath.Join(outputDir, "0001")
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %v", dirPath, err)
			}
			filename = filepath.Join(dirPath, "01-diary")
		} else {
			// Parse month key
			parts := strings.Split(monthKey, "-")
			if len(parts) == 2 {
				year := parts[0]
				month := parts[1]
				dirPath := filepath.Join(outputDir, year)
				if err := os.MkdirAll(dirPath, 0755); err != nil {
					return fmt.Errorf("failed to create directory %s: %v", dirPath, err)
				}
				filename = filepath.Join(dirPath, fmt.Sprintf("%s-diary", month))
			}
		}

		// Parse existing file if it exists
		existingContent, err := parseExistingDiaryFile(filename)
		if err != nil {
			return fmt.Errorf("failed to parse existing file %s: %v", filename, err)
		}

		// Generate merged content
		mergedContent := generateMergedDiaryContent(existingContent, entries)

		// Write to file
		if err := writeDiaryToFile(filename, mergedContent); err != nil {
			return fmt.Errorf("failed to write file %s: %v", filename, err)
		}
	}

	return nil
}
