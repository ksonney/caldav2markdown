package markdown

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/arran4/golang-ical"
	"gopkg.in/yaml.v3"
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

type EventMarkdown struct {
	Title       string
	Description string
	Location    string
	StartTime   time.Time
	EndTime     time.Time
	AllDay      bool
	UID         string
	Status      string
	Categories  []string
}

type TodoMarkdown struct {
	Title       string
	Description string
	Status      string
	Priority    string
	DueDate     time.Time
	Completed   bool
	UID         string
	Categories  []string
}

func ConvertEvent(event *ics.VEvent) EventMarkdown {
	var md EventMarkdown

	if summary := event.GetProperty(ics.ComponentPropertySummary); summary != nil {
		md.Title = summary.Value
	}

	if description := event.GetProperty(ics.ComponentPropertyDescription); description != nil {
		md.Description = description.Value
	}

	if location := event.GetProperty(ics.ComponentPropertyLocation); location != nil {
		md.Location = location.Value
	}

	if uid := event.GetProperty(ics.ComponentPropertyUniqueId); uid != nil {
		md.UID = uid.Value
	}

	if status := event.GetProperty(ics.ComponentPropertyStatus); status != nil {
		md.Status = status.Value
	}

	// Parse categories
	if categories := event.GetProperty(ics.ComponentPropertyCategories); categories != nil {
		md.Categories = strings.Split(categories.Value, ",")
		for i, cat := range md.Categories {
			md.Categories[i] = strings.TrimSpace(cat)
		}
	}

	if dtstart := event.GetProperty(ics.ComponentPropertyDtStart); dtstart != nil {
		if startTime, allDay, err := parseICalDateTime(dtstart.Value); err == nil {
			md.StartTime = startTime
			if allDay {
				md.AllDay = true
			}
		}
	}

	if dtend := event.GetProperty(ics.ComponentPropertyDtEnd); dtend != nil {
		if endTime, allDay, err := parseICalDateTime(dtend.Value); err == nil {
			md.EndTime = endTime
			if allDay {
				md.AllDay = true
			}
		}
	}

	// Ensure all events have both start and end times
	if !md.StartTime.IsZero() && md.EndTime.IsZero() {
		// If no end time is specified, set default duration based on event type
		if md.AllDay {
			// For all-day events, end at the end of the same day
			md.EndTime = md.StartTime.Add(24 * time.Hour).Add(-1 * time.Second)
		} else {
			// For timed events, default to 1 hour duration if no DURATION property
			defaultDuration := time.Hour

			// Check for DURATION property
			if duration := event.GetProperty(ics.ComponentPropertyDuration); duration != nil {
				if parsedDuration, err := parseICalDuration(duration.Value); err == nil {
					defaultDuration = parsedDuration
				}
			}

			md.EndTime = md.StartTime.Add(defaultDuration)
		}
	}

	return md
}

func (em EventMarkdown) ToMarkdown() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", em.Title))

	// Handle events with zero time (0001-01-01)
	if em.StartTime.IsZero() && em.EndTime.IsZero() {
		sb.WriteString("**Date:** No date specified\n\n")
	} else if em.StartTime.IsZero() {
		sb.WriteString(fmt.Sprintf("**End:** %s\n\n", em.EndTime.Format("2006-01-02 15:04")))
	} else if em.EndTime.IsZero() {
		if em.AllDay {
			sb.WriteString(fmt.Sprintf("**Date:** %s (All Day)\n\n", em.StartTime.Format("2006-01-02")))
		} else {
			sb.WriteString(fmt.Sprintf("**Start:** %s\n\n", em.StartTime.Format("2006-01-02 15:04")))
		}
	} else if em.AllDay {
		// For all-day events, show the date range
		if em.StartTime.Format("2006-01-02") == em.EndTime.Format("2006-01-02") {
			sb.WriteString(fmt.Sprintf("**Date:** %s (All Day)\n\n", em.StartTime.Format("2006-01-02")))
		} else {
			sb.WriteString(fmt.Sprintf("**Date:** %s to %s (All Day)\n\n",
				em.StartTime.Format("2006-01-02"),
				em.EndTime.Format("2006-01-02")))
		}
	} else {
		// For timed events, always show both start and end times
		sb.WriteString(fmt.Sprintf("**Start:** %s\n", em.StartTime.Format("2006-01-02 15:04")))
		sb.WriteString(fmt.Sprintf("**End:** %s\n", em.EndTime.Format("2006-01-02 15:04")))

		// Calculate and show duration
		duration := em.EndTime.Sub(em.StartTime)
		sb.WriteString(fmt.Sprintf("**Duration:** %s\n", formatDuration(duration)))
		sb.WriteString("\n")
	}

	if em.Location != "" {
		sb.WriteString(fmt.Sprintf("**Location:** %s\n\n", em.Location))
	}

	if em.Description != "" {
		sb.WriteString("## Description\n\n")
		sb.WriteString(em.Description)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// ToListItem returns a compact markdown list format for daily aggregation
func (em EventMarkdown) ToListItem() string {
	return em.ToListItemWithOptions(false)
}

// ToListItemWithOptions returns a compact markdown list format with optional hashtags
func (em EventMarkdown) ToListItemWithOptions(useHashtags bool) string {
	var sb strings.Builder

	if em.AllDay {
		sb.WriteString(fmt.Sprintf("- **%s** (All Day)", em.Title))
	} else if em.StartTime.IsZero() {
		sb.WriteString(fmt.Sprintf("- **%s** (Time TBD)", em.Title))
	} else {
		startTime := em.StartTime.Format("15:04")
		if em.EndTime.IsZero() || em.EndTime.Equal(em.StartTime) {
			sb.WriteString(fmt.Sprintf("- **%s** at %s", em.Title, startTime))
		} else {
			endTime := em.EndTime.Format("15:04")
			sb.WriteString(fmt.Sprintf("- **%s** (%s - %s)", em.Title, startTime, endTime))
		}
	}

	if em.Location != "" {
		sb.WriteString(fmt.Sprintf(" @ %s", em.Location))
	}

	if useHashtags {
		sb.WriteString(" #event")
	}

	if em.Description != "" {
		// Add description on a new line with indentation
		sb.WriteString(fmt.Sprintf("\n  %s", strings.ReplaceAll(em.Description, "\n", "\n  ")))
	}

	sb.WriteString("\n")
	return sb.String()
}

func (em EventMarkdown) Filename() string {
	safeTitle := strings.ReplaceAll(em.Title, "/", "-")
	safeTitle = strings.ReplaceAll(safeTitle, "\\", "-")
	safeTitle = strings.ReplaceAll(safeTitle, ":", "-")
	safeTitle = strings.ReplaceAll(safeTitle, "*", "-")
	safeTitle = strings.ReplaceAll(safeTitle, "?", "-")
	safeTitle = strings.ReplaceAll(safeTitle, "\"", "-")
	safeTitle = strings.ReplaceAll(safeTitle, "<", "-")
	safeTitle = strings.ReplaceAll(safeTitle, ">", "-")
	safeTitle = strings.ReplaceAll(safeTitle, "|", "-")

	var dateStr string
	if em.StartTime.IsZero() {
		dateStr = "0001-01-01"
	} else {
		dateStr = em.StartTime.Format("2006-01-02")
	}
	return fmt.Sprintf("%s_%s.md", dateStr, safeTitle)
}

func GenerateFilename(outputDir string, event EventMarkdown) string {
	// Create YYYY/MM directory structure
	var year, month string
	if event.StartTime.IsZero() {
		year = "0001"
		month = "01"
	} else {
		year = event.StartTime.Format("2006")
		month = event.StartTime.Format("01")
	}
	yearMonthDir := filepath.Join(outputDir, year, month)

	return filepath.Join(yearMonthDir, event.Filename())
}

func EnsureDirectoryExists(filePath string) error {
	dir := filepath.Dir(filePath)
	return os.MkdirAll(dir, 0755)
}

func ConvertTodo(todo *ics.VTodo) TodoMarkdown {
	var md TodoMarkdown

	if summary := todo.GetProperty(ics.ComponentPropertySummary); summary != nil {
		md.Title = summary.Value
	}

	if description := todo.GetProperty(ics.ComponentPropertyDescription); description != nil {
		md.Description = description.Value
	}

	if status := todo.GetProperty(ics.ComponentPropertyStatus); status != nil {
		md.Status = status.Value
		md.Completed = status.Value == "COMPLETED"
	}

	if priority := todo.GetProperty(ics.ComponentPropertyPriority); priority != nil {
		md.Priority = priority.Value
	}

	if due := todo.GetProperty(ics.ComponentPropertyDue); due != nil {
		if dueTime, _, err := parseICalDateTime(due.Value); err == nil {
			md.DueDate = dueTime
		}
	}

	if uid := todo.GetProperty(ics.ComponentPropertyUniqueId); uid != nil {
		md.UID = uid.Value
	}

	// Parse categories
	if categories := todo.GetProperty(ics.ComponentPropertyCategories); categories != nil {
		md.Categories = strings.Split(categories.Value, ",")
		for i, cat := range md.Categories {
			md.Categories[i] = strings.TrimSpace(cat)
		}
	}

	return md
}

func (tm TodoMarkdown) ToMarkdown() string {
	return tm.ToMarkdownWithOptions(false, false)
}

func (tm TodoMarkdown) ToMarkdownWithOptions(useDueDateEmoji, useHashtags bool) string {
	var sb strings.Builder

	checkbox := "- [ ]"
	if tm.Completed {
		checkbox = "- [x]"
	}

	sb.WriteString(fmt.Sprintf("%s **%s**", checkbox, tm.Title))

	if tm.Priority != "" && tm.Priority != "0" {
		sb.WriteString(fmt.Sprintf(" (Priority: %s)", tm.Priority))
	}

	if !tm.DueDate.IsZero() {
		if useDueDateEmoji {
			sb.WriteString(fmt.Sprintf(" - 📅 %s", tm.DueDate.Format("2006-01-02")))
		} else {
			sb.WriteString(fmt.Sprintf(" - Due: %s", tm.DueDate.Format("2006-01-02")))
		}
	}

	if tm.Status != "" {
		sb.WriteString(fmt.Sprintf(" - Status: %s", tm.Status))
	}

	if useHashtags {
		sb.WriteString(" #task")
	}

	sb.WriteString("\n")

	if tm.Description != "" {
		sb.WriteString(fmt.Sprintf("  %s\n", tm.Description))
	}

	sb.WriteString("\n")
	return sb.String()
}

func GenerateTasksFile(outputDir string, todos []TodoMarkdown) error {
	var sb strings.Builder
	sb.WriteString("# Tasks\n\n")

	for _, todo := range todos {
		sb.WriteString(todo.ToMarkdown())
	}

	tasksPath := filepath.Join(outputDir, "tasks.md")
	return writeToFile(tasksPath, sb.String())
}

func writeToFile(filename, content string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

// parseICalDuration parses an iCalendar DURATION value (RFC 5545)
// Format: ["+"|"-"] "P" [n "W"] [n "D"] ["T" [n "H"] [n "M"] [n "S"]]
func parseICalDuration(durationStr string) (time.Duration, error) {
	// Remove optional leading + or -
	sign := 1
	if strings.HasPrefix(durationStr, "-") {
		sign = -1
		durationStr = durationStr[1:]
	} else if strings.HasPrefix(durationStr, "+") {
		durationStr = durationStr[1:]
	}

	// Must start with P
	if !strings.HasPrefix(durationStr, "P") {
		return 0, fmt.Errorf("invalid duration format: must start with P")
	}
	durationStr = durationStr[1:]

	var totalDuration time.Duration

	// Parse weeks (W)
	if idx := strings.Index(durationStr, "W"); idx >= 0 {
		weeksPart := durationStr[:idx]
		if weeks, err := strconv.Atoi(weeksPart); err == nil {
			totalDuration += time.Duration(weeks) * 7 * 24 * time.Hour
		}
		durationStr = durationStr[idx+1:]
	}

	// Parse days (D)
	if idx := strings.Index(durationStr, "D"); idx >= 0 {
		daysPart := durationStr[:idx]
		if days, err := strconv.Atoi(daysPart); err == nil {
			totalDuration += time.Duration(days) * 24 * time.Hour
		}
		durationStr = durationStr[idx+1:]
	}

	// Parse time components (after T)
	if strings.HasPrefix(durationStr, "T") {
		durationStr = durationStr[1:]

		// Parse hours (H)
		if idx := strings.Index(durationStr, "H"); idx >= 0 {
			hoursPart := durationStr[:idx]
			if hours, err := strconv.Atoi(hoursPart); err == nil {
				totalDuration += time.Duration(hours) * time.Hour
			}
			durationStr = durationStr[idx+1:]
		}

		// Parse minutes (M)
		if idx := strings.Index(durationStr, "M"); idx >= 0 {
			minutesPart := durationStr[:idx]
			if minutes, err := strconv.Atoi(minutesPart); err == nil {
				totalDuration += time.Duration(minutes) * time.Minute
			}
			durationStr = durationStr[idx+1:]
		}

		// Parse seconds (S)
		if idx := strings.Index(durationStr, "S"); idx >= 0 {
			secondsPart := durationStr[:idx]
			if seconds, err := strconv.Atoi(secondsPart); err == nil {
				totalDuration += time.Duration(seconds) * time.Second
			}
		}
	}

	return time.Duration(sign) * totalDuration, nil
}

// ExistingDayContent represents parsed content from an existing daily markdown file
type ExistingDayContent struct {
	Title           string
	AllDayEvents    []string
	ScheduledEvents []string
	Tasks           []string
	OtherContent    []string // Any content that doesn't fit the standard sections
	Frontmatter     map[string]interface{} // Raw existing YAML frontmatter
	HasFrontmatter  bool                   // Whether the file had frontmatter
	CustomSections  map[string][]string    // Custom sections with their content
}

// parseExistingFile reads and parses an existing daily markdown file
func parseExistingFile(filename string) (*ExistingDayContent, error) {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // File doesn't exist, return nil (not an error)
		}
		return nil, fmt.Errorf("failed to open existing file %s: %v", filename, err)
	}
	defer file.Close()

	content := &ExistingDayContent{
		CustomSections: make(map[string][]string),
	}
	scanner := bufio.NewScanner(file)

	currentSection := ""
	var currentItems []string
	inFrontmatter := false
	frontmatterComplete := false
	var frontmatterLines []string

	for scanner.Scan() {
		line := scanner.Text()

		// Handle YAML frontmatter
		if !frontmatterComplete {
			if line == "---" {
				if !inFrontmatter {
					// Starting frontmatter
					inFrontmatter = true
					content.HasFrontmatter = true
					continue
				} else {
					// Ending frontmatter
					frontmatterComplete = true
					// Parse the collected frontmatter
					if len(frontmatterLines) > 0 {
						frontmatterYAML := strings.Join(frontmatterLines, "\n")
						var fm map[string]interface{}
						if err := yaml.Unmarshal([]byte(frontmatterYAML), &fm); err == nil {
							content.Frontmatter = fm
						}
					}
					continue
				}
			}
			if inFrontmatter {
				frontmatterLines = append(frontmatterLines, line)
				continue
			}
			// If we reach here and haven't seen frontmatter delimiter,
			// then there's no frontmatter
			if line != "" && !strings.HasPrefix(line, "#") {
				frontmatterComplete = true
			}
		}

		// Check for title (date header)
		if strings.HasPrefix(line, "# ") {
			content.Title = line
			continue
		}

		// Check for section headers
		if strings.HasPrefix(line, "## ") {
			// Save previous section
			savePreviousSection(content, currentSection, currentItems)

			// Start new section
			switch line {
			case "## All Day Events":
				currentSection = "allday"
			case "## Scheduled Events":
				currentSection = "scheduled"
			case "## Tasks":
				currentSection = "tasks"
			default:
				// This is a custom section - store the header name
				currentSection = line
			}
			currentItems = []string{}
			continue
		}

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Add line to current section
		if currentSection != "" {
			currentItems = append(currentItems, line)
		} else {
			// Content before any section headers (but after frontmatter)
			content.OtherContent = append(content.OtherContent, line)
		}
	}

	// Save the last section
	savePreviousSection(content, currentSection, currentItems)

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file %s: %v", filename, err)
	}

	return content, nil
}

// savePreviousSection saves items to the appropriate section in ExistingDayContent
func savePreviousSection(content *ExistingDayContent, section string, items []string) {
	switch section {
	case "allday":
		content.AllDayEvents = items
	case "scheduled":
		content.ScheduledEvents = items
	case "tasks":
		content.Tasks = items
	default:
		if section != "" {
			if strings.HasPrefix(section, "##") {
				// This is a custom section header
				content.CustomSections[section] = items
			} else {
				// This is other content (before any headers)
				content.OtherContent = append(content.OtherContent, items...)
			}
		}
	}
}

// deduplicate removes duplicate items from a slice while preserving order
func deduplicate(items []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, item := range items {
		// Normalize the item for comparison (trim whitespace)
		normalized := strings.TrimSpace(item)
		if normalized == "" {
			continue
		}

		if !seen[normalized] {
			seen[normalized] = true
			result = append(result, item)
		}
	}

	return result
}

// generateFrontmatter creates YAML frontmatter for a daily file
func generateFrontmatter(date string, allDayCount, scheduledCount, taskCount int, events []EventMarkdown, tasks []TodoMarkdown, useFrontmatter bool) string {
	if !useFrontmatter {
		return ""
	}

	var title string
	if date == "0001-01-01" {
		title = "Events (Date TBD)"
	} else {
		if parsedDate, err := time.Parse("2006-01-02", date); err == nil {
			title = parsedDate.Format("Monday, January 2, 2006")
		} else {
			title = date
		}
	}

	// Collect unique tags from events and tasks
	tagSet := make(map[string]bool)
	for _, event := range events {
		for _, cat := range event.Categories {
			if cat != "" {
				tagSet[cat] = true
			}
		}
	}
	for _, task := range tasks {
		for _, cat := range task.Categories {
			if cat != "" {
				tagSet[cat] = true
			}
		}
	}

	var tags []string
	for tag := range tagSet {
		tags = append(tags, tag)
	}

	frontmatter := DayFrontmatter{
		Date:        date,
		Title:       title,
		EventCount:  allDayCount + scheduledCount,
		TaskCount:   taskCount,
		AllDayCount: allDayCount,
		Tags:        tags,
		Type:        "daily",
	}

	yamlData, err := yaml.Marshal(frontmatter)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("---\n%s---\n\n", string(yamlData))
}

// generateMergedContent creates the final markdown content by merging existing and new items
func generateMergedContent(date string, existingContent *ExistingDayContent, newAllDay, newScheduled, newTasks []string) string {
	return generateMergedContentWithFrontmatter(date, existingContent, newAllDay, newScheduled, newTasks, nil, nil, false)
}

// generateMergedContentWithFrontmatter creates the final markdown content with optional frontmatter
func generateMergedContentWithFrontmatter(date string, existingContent *ExistingDayContent, newAllDay, newScheduled, newTasks []string, events []EventMarkdown, tasks []TodoMarkdown, useFrontmatter bool) string {
	var sb strings.Builder

	// Handle frontmatter - merge existing with new if requested
	if useFrontmatter || (existingContent != nil && existingContent.HasFrontmatter) {
		var mergedFrontmatter map[string]interface{}

		// Start with existing frontmatter if available
		if existingContent != nil && existingContent.Frontmatter != nil {
			mergedFrontmatter = make(map[string]interface{})
			// Copy existing frontmatter
			for k, v := range existingContent.Frontmatter {
				mergedFrontmatter[k] = v
			}
		} else {
			mergedFrontmatter = make(map[string]interface{})
		}

		// Update with current data
		mergedFrontmatter["date"] = date
		if date == "0001-01-01" {
			mergedFrontmatter["title"] = "Events (Date TBD)"
		} else {
			if parsedDate, err := time.Parse("2006-01-02", date); err == nil {
				mergedFrontmatter["title"] = parsedDate.Format("Monday, January 2, 2006")
			} else {
				mergedFrontmatter["title"] = date
			}
		}
		mergedFrontmatter["event_count"] = len(newAllDay) + len(newScheduled)
		mergedFrontmatter["task_count"] = len(newTasks)
		mergedFrontmatter["allday_count"] = len(newAllDay)
		mergedFrontmatter["type"] = "daily"

		// Collect unique tags from existing and new events/tasks
		tagSet := make(map[string]bool)

		// Preserve existing tags
		if existingTags, exists := mergedFrontmatter["tags"]; exists {
			if tagSlice, ok := existingTags.([]interface{}); ok {
				for _, tag := range tagSlice {
					if tagStr, ok := tag.(string); ok && tagStr != "" {
						tagSet[tagStr] = true
					}
				}
			}
		}

		// Add new tags from events and tasks
		for _, event := range events {
			for _, cat := range event.Categories {
				if cat != "" {
					tagSet[cat] = true
				}
			}
		}
		for _, task := range tasks {
			for _, cat := range task.Categories {
				if cat != "" {
					tagSet[cat] = true
				}
			}
		}

		var tags []string
		for tag := range tagSet {
			tags = append(tags, tag)
		}
		mergedFrontmatter["tags"] = tags

		// Generate YAML frontmatter
		yamlData, err := yaml.Marshal(mergedFrontmatter)
		if err == nil {
			sb.WriteString(fmt.Sprintf("---\n%s---\n\n", string(yamlData)))
		}
	}

	// Generate title
	if date == "0001-01-01" {
		sb.WriteString("# Events (Date TBD)\n\n")
	} else {
		parsedDate, _ := time.Parse("2006-01-02", date)
		sb.WriteString(fmt.Sprintf("# %s\n\n", parsedDate.Format("Monday, January 2, 2006")))
	}

	// Merge and deduplicate all-day events
	var allDayItems []string
	if existingContent != nil {
		allDayItems = append(allDayItems, existingContent.AllDayEvents...)
	}
	allDayItems = append(allDayItems, newAllDay...)
	allDayItems = deduplicate(allDayItems)

	// Merge and deduplicate scheduled events
	var scheduledItems []string
	if existingContent != nil {
		scheduledItems = append(scheduledItems, existingContent.ScheduledEvents...)
	}
	scheduledItems = append(scheduledItems, newScheduled...)
	scheduledItems = deduplicate(scheduledItems)

	// Merge and deduplicate tasks
	var taskItems []string
	if existingContent != nil {
		taskItems = append(taskItems, existingContent.Tasks...)
	}
	taskItems = append(taskItems, newTasks...)
	taskItems = deduplicate(taskItems)

	// Add other content from existing file (if any)
	if existingContent != nil && len(existingContent.OtherContent) > 0 {
		for _, line := range existingContent.OtherContent {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Add custom sections from existing file (if any)
	if existingContent != nil {
		for sectionHeader, sectionItems := range existingContent.CustomSections {
			// Skip standard sections as they'll be handled below
			if sectionHeader == "## All Day Events" || sectionHeader == "## Scheduled Events" || sectionHeader == "## Tasks" {
				continue
			}

			if len(sectionItems) > 0 {
				sb.WriteString(sectionHeader)
				sb.WriteString("\n\n")
				for _, item := range sectionItems {
					sb.WriteString(item)
					sb.WriteString("\n")
				}
				sb.WriteString("\n")
			}
		}
	}

	// Add all-day events section
	if len(allDayItems) > 0 {
		sb.WriteString("## All Day Events\n\n")
		for _, item := range allDayItems {
			sb.WriteString(item)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Add scheduled events section
	if len(scheduledItems) > 0 {
		sb.WriteString("## Scheduled Events\n\n")
		for _, item := range scheduledItems {
			sb.WriteString(item)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Add tasks section
	if len(taskItems) > 0 {
		sb.WriteString("## Tasks\n\n")
		for _, item := range taskItems {
			sb.WriteString(item)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "0m"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours == 0 {
		return fmt.Sprintf("%dm", minutes)
	} else if minutes == 0 {
		return fmt.Sprintf("%dh", hours)
	} else {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
}

// ProgressCallback is a function type for reporting progress during file generation
type ProgressCallback func(message string, current, total int)

// DayFrontmatter represents the YAML frontmatter for daily files
type DayFrontmatter struct {
	Date        string   `yaml:"date"`
	Title       string   `yaml:"title"`
	EventCount  int      `yaml:"event_count"`
	TaskCount   int      `yaml:"task_count"`
	AllDayCount int      `yaml:"allday_count"`
	Tags        []string `yaml:"tags,omitempty"`
	Type        string   `yaml:"type"`
}

// EventFrontmatter represents event-specific frontmatter data
type EventFrontmatter struct {
	Title      string    `yaml:"title"`
	StartTime  time.Time `yaml:"start_time,omitempty"`
	EndTime    time.Time `yaml:"end_time,omitempty"`
	AllDay     bool      `yaml:"all_day"`
	Location   string    `yaml:"location,omitempty"`
	UID        string    `yaml:"uid,omitempty"`
	Status     string    `yaml:"status,omitempty"`
	Categories []string  `yaml:"categories,omitempty"`
	Type       string    `yaml:"type"`
}

// TaskFrontmatter represents task-specific frontmatter data
type TaskFrontmatter struct {
	Title      string    `yaml:"title"`
	Status     string    `yaml:"status,omitempty"`
	Priority   string    `yaml:"priority,omitempty"`
	DueDate    time.Time `yaml:"due_date,omitempty"`
	Completed  bool      `yaml:"completed"`
	UID        string    `yaml:"uid,omitempty"`
	Categories []string  `yaml:"categories,omitempty"`
	Type       string    `yaml:"type"`
}

// GenerateDailyFilesWithTasks groups events and tasks by date and creates daily markdown files
func GenerateDailyFilesWithTasks(outputDir string, events []EventMarkdown, tasks []TodoMarkdown, useDueDateEmoji, useHashtags bool) error {
	return GenerateDailyFilesWithTasksAndProgress(outputDir, events, tasks, useDueDateEmoji, useHashtags, nil)
}

// GenerateDailyFilesWithTasksAndFrontmatter groups events and tasks by date and creates daily markdown files with frontmatter
func GenerateDailyFilesWithTasksAndFrontmatter(outputDir string, events []EventMarkdown, tasks []TodoMarkdown, useDueDateEmoji, useHashtags, useFrontmatter bool) error {
	return GenerateDailyFilesWithTasksProgressAndFrontmatter(outputDir, events, tasks, useDueDateEmoji, useHashtags, useFrontmatter, nil)
}

// GenerateDailyFilesWithTasksAndProgress groups events and tasks by date and creates daily markdown files with progress reporting
func GenerateDailyFilesWithTasksAndProgress(outputDir string, events []EventMarkdown, tasks []TodoMarkdown, useDueDateEmoji, useHashtags bool, progressCallback ProgressCallback) error {
	return GenerateDailyFilesWithTasksProgressAndFrontmatter(outputDir, events, tasks, useDueDateEmoji, useHashtags, false, progressCallback)
}

// GenerateDailyFilesWithTasksProgressAndFrontmatter groups events and tasks by date and creates daily markdown files with progress reporting and frontmatter support
func GenerateDailyFilesWithTasksProgressAndFrontmatter(outputDir string, events []EventMarkdown, tasks []TodoMarkdown, useDueDateEmoji, useHashtags, useFrontmatter bool, progressCallback ProgressCallback) error {
	// Group events by date
	eventsByDate := make(map[string][]EventMarkdown)
	tasksByDate := make(map[string][]TodoMarkdown)

	for _, event := range events {
		var dateKey string
		if event.StartTime.IsZero() {
			dateKey = "0001-01-01"
		} else {
			dateKey = event.StartTime.Format("2006-01-02")
		}
		eventsByDate[dateKey] = append(eventsByDate[dateKey], event)
	}

	// Group tasks by due date
	for _, task := range tasks {
		var dateKey string
		if task.DueDate.IsZero() {
			dateKey = "0001-01-01"
		} else {
			dateKey = task.DueDate.Format("2006-01-02")
		}
		tasksByDate[dateKey] = append(tasksByDate[dateKey], task)
	}

	// Get all unique dates from both events and tasks
	allDates := make(map[string]bool)
	for date := range eventsByDate {
		allDates[date] = true
	}
	for date := range tasksByDate {
		allDates[date] = true
	}

	// Convert to slice for indexed iteration
	datesList := make([]string, 0, len(allDates))
	for date := range allDates {
		datesList = append(datesList, date)
	}

	if progressCallback != nil {
		progressCallback(fmt.Sprintf("Generating %d daily files", len(datesList)), 0, len(datesList))
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
		var allDayEvents, timedEvents []EventMarkdown
		for _, event := range dayEvents {
			if event.AllDay || event.StartTime.IsZero() {
				allDayEvents = append(allDayEvents, event)
			} else {
				timedEvents = append(timedEvents, event)
			}
		}

		// Simple sort by start time for timed events
		for i := 0; i < len(timedEvents)-1; i++ {
			for j := i + 1; j < len(timedEvents); j++ {
				if timedEvents[i].StartTime.After(timedEvents[j].StartTime) {
					timedEvents[i], timedEvents[j] = timedEvents[j], timedEvents[i]
				}
			}
		}

		// Collect new content as strings for merging
		var newAllDayEvents, newScheduledEvents, newTasks []string

		for _, event := range allDayEvents {
			newAllDayEvents = append(newAllDayEvents, strings.TrimSpace(event.ToListItemWithOptions(useHashtags)))
		}

		for _, event := range timedEvents {
			newScheduledEvents = append(newScheduledEvents, strings.TrimSpace(event.ToListItemWithOptions(useHashtags)))
		}

		for _, task := range dayTasks {
			newTasks = append(newTasks, strings.TrimSpace(task.ToMarkdownWithOptions(useDueDateEmoji, useHashtags)))
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

		filename = filepath.Join(dirPath, fmt.Sprintf("%s.md", date))

		// Parse existing file if it exists
		existingContent, err := parseExistingFile(filename)
		if err != nil {
			return fmt.Errorf("failed to parse existing file %s: %v", filename, err)
		}

		// Report if we found existing content to merge
		if progressCallback != nil && existingContent != nil {
			existingItems := len(existingContent.AllDayEvents) + len(existingContent.ScheduledEvents) + len(existingContent.Tasks)
			if existingItems > 0 {
				progressCallback(fmt.Sprintf("Merging with %d existing items", existingItems), i+1, len(datesList))
			}
		}

		// Generate merged content
		mergedContent := generateMergedContentWithFrontmatter(date, existingContent, newAllDayEvents, newScheduledEvents, newTasks, dayEvents, dayTasks, useFrontmatter)

		// Write merged content to file
		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %v", filename, err)
		}
		defer file.Close()

		if _, err := file.WriteString(mergedContent); err != nil {
			return fmt.Errorf("failed to write content to %s: %v", filename, err)
		}
	}

	return nil
}

// GenerateDailyFiles groups events by date and creates daily markdown files (backward compatibility)
func GenerateDailyFiles(outputDir string, events []EventMarkdown) error {
	return GenerateDailyFilesWithTasks(outputDir, events, nil, false, false)
}