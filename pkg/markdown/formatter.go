package markdown

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/arran4/golang-ical"
)

type EventMarkdown struct {
	Title       string
	Description string
	Location    string
	StartTime   time.Time
	EndTime     time.Time
	AllDay      bool
}

type TodoMarkdown struct {
	Title       string
	Description string
	Status      string
	Priority    string
	DueDate     time.Time
	Completed   bool
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

	if dtstart := event.GetProperty(ics.ComponentPropertyDtStart); dtstart != nil {
		if strings.HasPrefix(dtstart.Value, "0001") {
			// Keep 0001-01-01 dates as zero time but allow processing
			md.StartTime = time.Time{}
		} else if startTime, err := time.Parse("20060102T150405Z", dtstart.Value); err == nil {
			md.StartTime = startTime
		} else if startTime, err := time.Parse("20060102", dtstart.Value); err == nil {
			md.StartTime = startTime
			md.AllDay = true
		}
	}

	if dtend := event.GetProperty(ics.ComponentPropertyDtEnd); dtend != nil {
		if strings.HasPrefix(dtend.Value, "0001") {
			// Keep 0001-01-01 dates as zero time but allow processing
			md.EndTime = time.Time{}
		} else if endTime, err := time.Parse("20060102T150405Z", dtend.Value); err == nil {
			md.EndTime = endTime
		} else if endTime, err := time.Parse("20060102", dtend.Value); err == nil {
			md.EndTime = endTime
			md.AllDay = true
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
		if strings.HasPrefix(due.Value, "0001") {
			// Keep 0001-01-01 dates as zero time but allow processing
			md.DueDate = time.Time{}
		} else if dueTime, err := time.Parse("20060102T150405Z", due.Value); err == nil {
			md.DueDate = dueTime
		} else if dueTime, err := time.Parse("20060102", due.Value); err == nil {
			md.DueDate = dueTime
		}
	}

	return md
}

func (tm TodoMarkdown) ToMarkdown() string {
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
		sb.WriteString(fmt.Sprintf(" - Due: %s", tm.DueDate.Format("2006-01-02")))
	}

	if tm.Status != "" {
		sb.WriteString(fmt.Sprintf(" - Status: %s", tm.Status))
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