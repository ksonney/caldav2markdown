package caldav

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/arran4/golang-ical"
	"caldav2markdown/pkg/rrule"
)

// CalDAV REPORT XML structures for calendar-query
type CalendarQuery struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:caldav calendar-query"`
	Prop    Prop     `xml:"DAV: prop"`
	Filter  Filter   `xml:"urn:ietf:params:xml:ns:caldav filter"`
}

type Prop struct {
	GetETag        *GetETag        `xml:"DAV: getetag,omitempty"`
	CalendarData   *CalendarData   `xml:"urn:ietf:params:xml:ns:caldav calendar-data,omitempty"`
}

type GetETag struct{}

type CalendarData struct {
	CompFilter *CompFilter `xml:"urn:ietf:params:xml:ns:caldav comp-filter,omitempty"`
}

type Filter struct {
	CompFilter CompFilter `xml:"urn:ietf:params:xml:ns:caldav comp-filter"`
}

type CompFilter struct {
	Name         string        `xml:"name,attr"`
	TimeRange    *TimeRange    `xml:"urn:ietf:params:xml:ns:caldav time-range,omitempty"`
	CompFilters  []CompFilter  `xml:"urn:ietf:params:xml:ns:caldav comp-filter,omitempty"`
}

type TimeRange struct {
	Start string `xml:"start,attr,omitempty"`
	End   string `xml:"end,attr,omitempty"`
}

// CalDAV REPORT response structures
type MultiStatus struct {
	XMLName   xml.Name   `xml:"DAV: multistatus"`
	Responses []Response `xml:"DAV: response"`
}

type Response struct {
	Href     string   `xml:"DAV: href"`
	PropStat PropStat `xml:"DAV: propstat"`
}

type PropStat struct {
	Prop   ResponseProp `xml:"DAV: prop"`
	Status string       `xml:"DAV: status"`
}

type ResponseProp struct {
	GetETag      string `xml:"DAV: getetag,omitempty"`
	CalendarData string `xml:"urn:ietf:params:xml:ns:caldav calendar-data,omitempty"`
}

// generateCalendarQuery creates a CalDAV calendar-query REPORT request XML
func generateCalendarQuery(startDate, endDate time.Time, componentTypes []string) (string, error) {
	query := CalendarQuery{
		Prop: Prop{
			GetETag:      &GetETag{},
			CalendarData: &CalendarData{},
		},
	}

	// Create time range filter
	var timeRange *TimeRange
	if !startDate.IsZero() || !endDate.IsZero() {
		timeRange = &TimeRange{}
		if !startDate.IsZero() {
			timeRange.Start = startDate.UTC().Format("20060102T150405Z")
		}
		if !endDate.IsZero() {
			timeRange.End = endDate.UTC().Format("20060102T150405Z")
		}
	}

	// Create main VCALENDAR filter
	query.Filter = Filter{
		CompFilter: CompFilter{
			Name: "VCALENDAR",
		},
	}

	// Add component type filters (VEVENT, VTODO, etc.)
	for _, compType := range componentTypes {
		compFilter := CompFilter{
			Name:      compType,
			TimeRange: timeRange,
		}
		query.Filter.CompFilter.CompFilters = append(query.Filter.CompFilter.CompFilters, compFilter)
	}

	// Marshal to XML
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="utf-8"?>`)
	buf.WriteString("\n")

	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", "  ")
	if err := encoder.Encode(query); err != nil {
		return "", fmt.Errorf("failed to encode calendar query XML: %w", err)
	}

	return buf.String(), nil
}

// performCalendarQuery executes a CalDAV calendar-query REPORT request
func (c *Client) performCalendarQuery(startDate, endDate time.Time, componentTypes []string) ([]string, error) {
	queryXML, err := generateCalendarQuery(startDate, endDate, componentTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate calendar query: %w", err)
	}

	// Create REPORT request
	req, err := http.NewRequest("REPORT", c.baseURL, strings.NewReader(queryXML))
	if err != nil {
		return nil, fmt.Errorf("failed to create REPORT request: %w", err)
	}

	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	req.Header.Set("Depth", "1")

	// Execute request using the configured transport (OAuth or basic auth)
	var resp *http.Response
	if c.httpClient != nil {
		// Use OAuth HTTP client
		resp, err = c.httpClient.Do(req)
	} else if c.username != "" && c.password != "" {
		// For basic auth, create HTTP client and set auth header
		httpClient := &http.Client{}
		req.SetBasicAuth(c.username, c.password)
		resp, err = httpClient.Do(req)
	} else {
		return nil, fmt.Errorf("no HTTP client configured")
	}

	if err != nil {
		return nil, fmt.Errorf("REPORT request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("REPORT request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read REPORT response: %w", err)
	}

	// Parse multistatus response
	var multiStatus MultiStatus
	if err := xml.Unmarshal(respBody, &multiStatus); err != nil {
		return nil, fmt.Errorf("failed to parse REPORT response XML: %w", err)
	}

	// Extract calendar data from responses
	var calendarData []string
	for _, response := range multiStatus.Responses {
		if response.PropStat.Prop.CalendarData != "" {
			calendarData = append(calendarData, response.PropStat.Prop.CalendarData)
		}
	}

	return calendarData, nil
}

// GetEventsWithServerSideFiltering uses CalDAV REPORT queries to fetch events with server-side filtering
func (c *Client) GetEventsWithServerSideFiltering() (*DeduplicationResult, error) {
	return c.GetEventsWithServerSideFilteringAndProgress(nil)
}

// GetEventsWithServerSideFilteringAndProgress uses CalDAV REPORT queries with progress reporting
func (c *Client) GetEventsWithServerSideFilteringAndProgress(progressCallback ProgressCallback) (*DeduplicationResult, error) {
	if progressCallback != nil {
		progressCallback("Performing server-side calendar query...", 0, 1)
	}

	// Query for both VEVENT and VTODO components
	componentTypes := []string{"VEVENT", "VTODO"}
	calendarDataList, err := c.performCalendarQuery(c.startDate, c.endDate, componentTypes)
	if err != nil {
		return nil, fmt.Errorf("calendar query failed: %w", err)
	}

	if progressCallback != nil {
		progressCallback(fmt.Sprintf("Processing %d calendar objects from server", len(calendarDataList)), 1, 1)
	}

	// Parse calendar data and extract events/todos
	var events []*ics.VEvent
	var todos []*ics.VTodo
	seenUIDs := make(map[string]bool)
	duplicatesFound := 0

	for i, calData := range calendarDataList {
		if progressCallback != nil && len(calendarDataList) > 10 && (i+1)%10 == 0 {
			progressCallback(fmt.Sprintf("Parsing calendar data %d/%d", i+1, len(calendarDataList)), i+1, len(calendarDataList))
		}

		calendar, err := ics.ParseCalendar(strings.NewReader(calData))
		if err != nil {
			fmt.Printf("Warning: failed to parse calendar data: %v\n", err)
			continue
		}

		// Process events
		for _, event := range calendar.Events() {
			// Server-side filtering should have already applied time range,
			// but we still need to check for edge cases and expand recurring events if needed
			if !c.isEventInDateRange(event) {
				continue
			}

			// Note: Server should handle recurring event expansion, but some servers don't
			// For now, we still do client-side expansion as a fallback
			expandedEvents, err := rrule.ExpandEvent(event, 1000, c.startDate, c.endDate)
			if err != nil {
				fmt.Printf("Warning: failed to expand recurring event: %v\n", err)
				expandedEvents = []*ics.VEvent{event}
			}

			for _, expandedEvent := range expandedEvents {
				if !c.isEventInDateRange(expandedEvent) {
					continue
				}

				uid := c.getEventUID(expandedEvent)
				if uid != "" && seenUIDs[uid] {
					duplicatesFound++
					continue
				}
				if uid != "" {
					seenUIDs[uid] = true
				}
				events = append(events, expandedEvent)
			}
		}

		// Process todos
		for _, todo := range calendar.Todos() {
			uid := c.getTodoUID(todo)
			if uid != "" && seenUIDs[uid] {
				duplicatesFound++
				continue
			}
			if uid != "" {
				seenUIDs[uid] = true
			}
			todos = append(todos, todo)
		}
	}

	return &DeduplicationResult{
		Events:          events,
		Todos:           todos,
		DuplicatesFound: duplicatesFound,
	}, nil
}