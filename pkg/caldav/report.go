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
	Href      string     `xml:"DAV: href"`
	PropStats []PropStat `xml:"DAV: propstat"`
}

type PropStat struct {
	Prop   ResponseProp `xml:"DAV: prop"`
	Status string       `xml:"DAV: status"`
}

type ResponseProp struct {
	GetETag      string `xml:"DAV: getetag,omitempty"`
	CalendarData string `xml:"urn:ietf:params:xml:ns:caldav calendar-data,omitempty"`
}

// parseMultiStatusResponse extracts calendar data from a multistatus response with enhanced error handling
func parseMultiStatusResponse(multiStatus *MultiStatus, traceWebCalls bool) ([]string, error) {
	var calendarData []string
	var errors []string
	successCount := 0

	if traceWebCalls {
		fmt.Printf("=== Parsing MultiStatus Response ===\n")
		fmt.Printf("Total responses: %d\n", len(multiStatus.Responses))
	}

	for i, response := range multiStatus.Responses {
		if traceWebCalls {
			fmt.Printf("Response %d: %s\n", i+1, response.Href)
			fmt.Printf("  PropStats count: %d\n", len(response.PropStats))
		}

		// A response can have multiple propstat elements, we need to check each one
		foundData := false
		for j, propStat := range response.PropStats {
			if traceWebCalls {
				fmt.Printf("  PropStat %d: Status = %s\n", j+1, propStat.Status)
			}

			// Check propstat status - should be "HTTP/1.1 200 OK" for successful responses
			if !strings.Contains(propStat.Status, "200") {
				errors = append(errors, fmt.Sprintf("Resource %s returned status: %s", response.Href, propStat.Status))
				continue
			}

			if propStat.Prop.CalendarData != "" {
				calendarData = append(calendarData, propStat.Prop.CalendarData)
				successCount++
				foundData = true
				if traceWebCalls {
					fmt.Printf("    Found calendar data (%d bytes)\n", len(propStat.Prop.CalendarData))
				}
				break // Only need one successful calendar-data per response
			}
		}

		// If no successful propstat was found for this response, log it
		if !foundData && len(response.PropStats) > 0 {
			// Check if there were any propstats at all
			hasCalendarDataProp := false
			for _, propStat := range response.PropStats {
				if propStat.Prop.CalendarData != "" {
					hasCalendarDataProp = true
					break
				}
			}
			if !hasCalendarDataProp {
				errors = append(errors, fmt.Sprintf("Resource %s has no calendar data", response.Href))
			}
		}
	}

	if traceWebCalls {
		fmt.Printf("=== MultiStatus Parsing Complete ===\n")
		fmt.Printf("Success count: %d\n", successCount)
		fmt.Printf("Error count: %d\n", len(errors))
	}

	// Log any errors but don't fail the entire operation
	if len(errors) > 0 {
		fmt.Printf("Warning: Some calendar resources had errors:\n")
		for _, errMsg := range errors {
			fmt.Printf("  %s\n", errMsg)
		}
	}

	if len(calendarData) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("no calendar data retrieved: all %d responses had errors", len(errors))
	}

	if successCount > 0 {
		fmt.Printf("Successfully retrieved %d calendar objects\n", successCount)
	}

	return calendarData, nil
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

	// Extract calendar data using the enhanced multistatus parser
	return parseMultiStatusResponse(&multiStatus, c.traceWebCalls)
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
			// but some servers don't handle recurring events properly, so we always
			// expand them and then filter the instances
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
			// Always expand recurring todos first, then filter the instances
			expandedTodos, err := rrule.ExpandTodo(todo, 1000, c.startDate, c.endDate)
			if err != nil {
				fmt.Printf("Warning: failed to expand recurring todo: %v\n", err)
				expandedTodos = []*ics.VTodo{todo} // Fall back to original todo
			}

			for _, expandedTodo := range expandedTodos {
				if !c.isTodoInDateRange(expandedTodo) {
					continue
				}

				uid := c.getTodoUID(expandedTodo)
				if uid != "" && seenUIDs[uid] {
					duplicatesFound++
					continue
				}
				if uid != "" {
					seenUIDs[uid] = true
				}
				todos = append(todos, expandedTodo)
			}
		}
	}

	return &DeduplicationResult{
		Events:          events,
		Todos:           todos,
		DuplicatesFound: duplicatesFound,
	}, nil
}