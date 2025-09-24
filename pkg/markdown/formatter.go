package markdown

import (
	"fmt"
	"path/filepath"
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
		if startTime, err := time.Parse("20060102T150405Z", dtstart.Value); err == nil {
			md.StartTime = startTime
		} else if startTime, err := time.Parse("20060102", dtstart.Value); err == nil {
			md.StartTime = startTime
			md.AllDay = true
		}
	}

	if dtend := event.GetProperty(ics.ComponentPropertyDtEnd); dtend != nil {
		if endTime, err := time.Parse("20060102T150405Z", dtend.Value); err == nil {
			md.EndTime = endTime
		} else if endTime, err := time.Parse("20060102", dtend.Value); err == nil {
			md.EndTime = endTime
			md.AllDay = true
		}
	}

	return md
}

func (em EventMarkdown) ToMarkdown() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", em.Title))

	if em.AllDay {
		sb.WriteString(fmt.Sprintf("**Date:** %s\n\n", em.StartTime.Format("2006-01-02")))
	} else {
		sb.WriteString(fmt.Sprintf("**Start:** %s\n", em.StartTime.Format("2006-01-02 15:04")))
		if !em.EndTime.IsZero() {
			sb.WriteString(fmt.Sprintf("**End:** %s\n", em.EndTime.Format("2006-01-02 15:04")))
		}
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

	dateStr := em.StartTime.Format("2006-01-02")
	return fmt.Sprintf("%s_%s.md", dateStr, safeTitle)
}

func GenerateFilename(outputDir string, event EventMarkdown) string {
	return filepath.Join(outputDir, event.Filename())
}