package caldav

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/arran4/golang-ical"
	"github.com/studio-b12/gowebdav"
	"caldav2markdown/pkg/rrule"
)

type Client struct {
	webdavClient *gowebdav.Client
	baseURL      string
	startDate    time.Time
	endDate      time.Time
}

type Config struct {
	URL       string
	Username  string
	Password  string
	StartDate time.Time
	EndDate   time.Time
}

func NewClient(config Config) *Client {
	client := gowebdav.NewClient(config.URL, config.Username, config.Password)

	return &Client{
		webdavClient: client,
		baseURL:      config.URL,
		startDate:    config.StartDate,
		endDate:      config.EndDate,
	}
}

type DeduplicationResult struct {
	Events          []*ics.VEvent
	Todos           []*ics.VTodo
	DuplicatesFound int
}

func (c *Client) GetEvents() ([]*ics.VEvent, error) {
	result, err := c.GetEventsWithDeduplication()
	if err != nil {
		return nil, err
	}
	return result.Events, nil
}

func (c *Client) GetEventsWithDeduplication() (*DeduplicationResult, error) {
	files, err := c.webdavClient.ReadDir("/")
	if err != nil {
		return nil, fmt.Errorf("failed to list calendar files: %w", err)
	}

	var events []*ics.VEvent
	var todos []*ics.VTodo
	seenUIDs := make(map[string]bool)
	duplicatesFound := 0

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".ics") {
			continue
		}

		content, err := c.webdavClient.Read(file.Name())
		if err != nil {
			fmt.Printf("Warning: failed to read file %s: %v\n", file.Name(), err)
			continue
		}

		calendar, err := ics.ParseCalendar(strings.NewReader(string(content)))
		if err != nil {
			fmt.Printf("Warning: failed to parse calendar %s: %v\n", file.Name(), err)
			continue
		}

		for _, event := range calendar.Events() {
			// Skip events outside the configured date range
			if !c.isEventInDateRange(event) {
				continue
			}

			// Expand recurring events
			expandedEvents, err := rrule.ExpandEvent(event, 1000, c.startDate, c.endDate)
			if err != nil {
				fmt.Printf("Warning: failed to expand recurring event: %v\n", err)
				expandedEvents = []*ics.VEvent{event} // Fall back to original event
			}

			for _, expandedEvent := range expandedEvents {
				// Skip events outside the configured date range (check again for expanded events)
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

func (c *Client) getEventUID(event *ics.VEvent) string {
	if uid := event.GetProperty(ics.ComponentPropertyUniqueId); uid != nil {
		return uid.Value
	}
	return ""
}

func (c *Client) getTodoUID(todo *ics.VTodo) string {
	if uid := todo.GetProperty(ics.ComponentPropertyUniqueId); uid != nil {
		return uid.Value
	}
	return ""
}

func (c *Client) isEventInDateRange(event *ics.VEvent) bool {
	if dtstart := event.GetProperty(ics.ComponentPropertyDtStart); dtstart != nil {
		var startTime time.Time
		var err error

		// Handle special case of 0001-01-01 dates - treat as valid but use zero time for comparison
		if strings.HasPrefix(dtstart.Value, "0001") {
			startTime = time.Time{} // Zero time
		} else {
			if startTime, err = time.Parse("20060102T150405Z", dtstart.Value); err != nil {
				if startTime, err = time.Parse("20060102", dtstart.Value); err != nil {
					// If we can't parse the date, include the event anyway (don't filter out invalid dates)
					return true
				}
			}
		}

		// For zero-time events, include them unconditionally
		if startTime.IsZero() {
			return true
		}

		// Check if event starts before our end date and doesn't end before our start date
		if startTime.After(c.endDate) {
			return false
		}

		// For events with end times, check if they end after our start date
		if dtend := event.GetProperty(ics.ComponentPropertyDtEnd); dtend != nil {
			var endTime time.Time
			// Handle special case of 0001-01-01 end dates
			if strings.HasPrefix(dtend.Value, "0001") {
				endTime = time.Time{} // Zero time
			} else {
				if endTime, err = time.Parse("20060102T150405Z", dtend.Value); err != nil {
					if endTime, err = time.Parse("20060102", dtend.Value); err != nil {
						// If we can't parse end time, include the event anyway (don't filter out invalid dates)
						return true
					}
				}
			}

			// If end time is zero, include the event
			if endTime.IsZero() {
				return true
			}

			return endTime.After(c.startDate) || endTime.Equal(c.startDate)
		}

		// For events without end times, just check if they start within our range
		return (startTime.After(c.startDate) || startTime.Equal(c.startDate))
	}

	return false
}

func (c *Client) TestConnection() error {
	resp, err := http.Get(c.baseURL)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned error status: %d", resp.StatusCode)
	}

	return nil
}