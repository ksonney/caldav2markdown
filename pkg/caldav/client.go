package caldav

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/arran4/golang-ical"
	"github.com/studio-b12/gowebdav"
	"caldav2markdown/pkg/rrule"
	"caldav2markdown/pkg/oauth"
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

type Client struct {
	webdavClient           *gowebdav.Client
	httpClient             *http.Client
	baseURL                string
	username               string
	password               string
	startDate              time.Time
	endDate                time.Time
	useOAuth               bool
	useServerSideFiltering bool
	enableCalendarDiscovery bool
	includeCalendars       []string
	excludeCalendars       []string
}

type Config struct {
	URL               string
	Username          string
	Password          string
	StartDate         time.Time
	EndDate           time.Time
	// OAuth fields
	UseOAuth          bool
	ClientID          string
	ClientSecret      string
	// Performance options
	UseServerSideFiltering bool
	// Multi-calendar options
	DiscoverCalendars     bool
	IncludeCalendars      []string // Specific calendar names/URLs to include
	ExcludeCalendars      []string // Specific calendar names/URLs to exclude
}

func NewClient(config Config) *Client {
	c := &Client{
		baseURL:                config.URL,
		username:               config.Username,
		password:               config.Password,
		startDate:              config.StartDate,
		endDate:                config.EndDate,
		useOAuth:               config.UseOAuth,
		useServerSideFiltering: config.UseServerSideFiltering,
		enableCalendarDiscovery: config.DiscoverCalendars,
		includeCalendars:       config.IncludeCalendars,
		excludeCalendars:       config.ExcludeCalendars,
	}

	if config.UseOAuth {
		oauthClient := oauth.NewClient(oauth.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
		})

		httpClient, err := oauthClient.GetHTTPClient(context.Background())
		if err != nil {
			// For now, fall back to basic auth if OAuth fails
			// In production, you might want to return the error
			fmt.Printf("Warning: OAuth failed, falling back to basic auth: %v\n", err)
			c.webdavClient = gowebdav.NewClient(config.URL, config.Username, config.Password)
		} else {
			c.httpClient = httpClient
			c.webdavClient = gowebdav.NewClient(config.URL, "", "")
			c.webdavClient.SetTransport(httpClient.Transport)
		}
	} else {
		c.webdavClient = gowebdav.NewClient(config.URL, config.Username, config.Password)
	}

	return c
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

// ProgressCallback is a function type for reporting progress
type ProgressCallback func(message string, current, total int)

func (c *Client) GetEventsWithDeduplication() (*DeduplicationResult, error) {
	return c.GetEventsWithDeduplicationAndProgress(nil)
}

func (c *Client) GetEventsWithDeduplicationAndProgress(progressCallback ProgressCallback) (*DeduplicationResult, error) {
	if c.enableCalendarDiscovery {
		// Multi-calendar processing
		return c.getEventsFromMultipleCalendarsAndProgress(progressCallback)
	} else if c.useServerSideFiltering {
		// Single calendar with server-side filtering
		result, err := c.GetEventsWithServerSideFilteringAndProgress(progressCallback)
		if err != nil {
			if progressCallback != nil {
				progressCallback("Server-side filtering failed, falling back to client-side...", 0, 1)
			}
			fmt.Printf("Warning: Server-side filtering failed (%v), falling back to client-side processing\n", err)
			return c.getEventsWithClientSideFilteringAndProgress(progressCallback)
		}
		return result, nil
	} else {
		// Single calendar with client-side filtering (original behavior)
		return c.getEventsWithClientSideFilteringAndProgress(progressCallback)
	}
}

// getEventsWithClientSideFilteringAndProgress is the original implementation
func (c *Client) getEventsWithClientSideFilteringAndProgress(progressCallback ProgressCallback) (*DeduplicationResult, error) {
	if progressCallback != nil {
		progressCallback("Connecting to CalDAV server...", 0, 1)
	}

	files, err := c.webdavClient.ReadDir("/")
	if err != nil {
		return nil, fmt.Errorf("failed to list calendar files: %w", err)
	}

	var events []*ics.VEvent
	var todos []*ics.VTodo
	seenUIDs := make(map[string]bool)
	duplicatesFound := 0

	// Count .ics files for progress reporting
	icsFiles := []os.FileInfo{}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".ics") {
			icsFiles = append(icsFiles, file)
		}
	}

	if progressCallback != nil {
		progressCallback(fmt.Sprintf("Found %d calendar files to process", len(icsFiles)), 1, 1)
	}

	for i, file := range icsFiles {
		if progressCallback != nil {
			progressCallback(fmt.Sprintf("Processing %s", file.Name()), i+1, len(icsFiles))
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
			// Always expand recurring events first, then filter the instances
			expandedEvents, err := rrule.ExpandEvent(event, 1000, c.startDate, c.endDate)
			if err != nil {
				fmt.Printf("Warning: failed to expand recurring event: %v\n", err)
				expandedEvents = []*ics.VEvent{event} // Fall back to original event
			}

			for _, expandedEvent := range expandedEvents {
				// Filter expanded instances based on their individual dates
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
			// Always expand recurring todos first, then filter the instances
			expandedTodos, err := rrule.ExpandTodo(todo, 1000, c.startDate, c.endDate)
			if err != nil {
				fmt.Printf("Warning: failed to expand recurring todo: %v\n", err)
				expandedTodos = []*ics.VTodo{todo} // Fall back to original todo
			}

			for _, expandedTodo := range expandedTodos {
				// Filter expanded instances based on their individual dates
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

		// Handle special case of 0001-01-01 dates - treat as valid but use zero time for comparison
		if strings.HasPrefix(dtstart.Value, "0001") {
			startTime = time.Time{} // Zero time
		} else {
			if t, _, parseErr := parseICalDateTime(dtstart.Value); parseErr == nil {
				startTime = t
			} else {
				// If we can't parse the date, include the event anyway (don't filter out invalid dates)
				return true
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
				if t, _, parseErr := parseICalDateTime(dtend.Value); parseErr == nil {
					endTime = t
				} else {
					// If we can't parse end time, include the event anyway (don't filter out invalid dates)
					return true
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

func (c *Client) isTodoInDateRange(todo *ics.VTodo) bool {
	// Check due date first
	if due := todo.GetProperty(ics.ComponentPropertyDue); due != nil {
		var dueTime time.Time

		// Handle special case of 0001-01-01 dates - treat as valid but use zero time for comparison
		if strings.HasPrefix(due.Value, "0001") {
			dueTime = time.Time{} // Zero time
		} else {
			if t, _, parseErr := parseICalDateTime(due.Value); parseErr == nil {
				dueTime = t
			} else {
				// If we can't parse the date, include the todo anyway (don't filter out invalid dates)
				return true
			}
		}

		// For zero-time todos, include them unconditionally
		if dueTime.IsZero() {
			return true
		}

		// Check if todo due date is within our range
		return (dueTime.After(c.startDate) || dueTime.Equal(c.startDate)) && !dueTime.After(c.endDate)
	}

	// If no due date, check start date
	if dtstart := todo.GetProperty(ics.ComponentPropertyDtStart); dtstart != nil {
		var startTime time.Time

		if strings.HasPrefix(dtstart.Value, "0001") {
			startTime = time.Time{} // Zero time
		} else {
			if t, _, parseErr := parseICalDateTime(dtstart.Value); parseErr == nil {
				startTime = t
			} else {
				return true
			}
		}

		if startTime.IsZero() {
			return true
		}

		return (startTime.After(c.startDate) || startTime.Equal(c.startDate)) && !startTime.After(c.endDate)
	}

	// If no date information, include the todo
	return true
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

// filterCalendars applies include/exclude filters to discovered calendars
func (c *Client) filterCalendars(calendars []CalendarInfo) []CalendarInfo {
	filtered := make([]CalendarInfo, 0, len(calendars))

	for _, calendar := range calendars {
		// Check exclude list first
		excluded := false
		for _, exclude := range c.excludeCalendars {
			if strings.EqualFold(calendar.DisplayName, exclude) ||
			   strings.EqualFold(calendar.URL, exclude) ||
			   strings.Contains(strings.ToLower(calendar.URL), strings.ToLower(exclude)) {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		// Check include list (if specified)
		if len(c.includeCalendars) > 0 {
			included := false
			for _, include := range c.includeCalendars {
				if strings.EqualFold(calendar.DisplayName, include) ||
				   strings.EqualFold(calendar.URL, include) ||
				   strings.Contains(strings.ToLower(calendar.URL), strings.ToLower(include)) {
					included = true
					break
				}
			}
			if !included {
				continue
			}
		}

		filtered = append(filtered, calendar)
	}

	return filtered
}

// getEventsFromMultipleCalendarsAndProgress discovers and processes multiple calendars
func (c *Client) getEventsFromMultipleCalendarsAndProgress(progressCallback ProgressCallback) (*DeduplicationResult, error) {
	if progressCallback != nil {
		progressCallback("Discovering calendars...", 0, 1)
	}

	// Discover all calendars
	calendars, err := c.DiscoverCalendars()
	if err != nil {
		return nil, fmt.Errorf("failed to discover calendars: %w", err)
	}

	if len(calendars) == 0 {
		return nil, fmt.Errorf("no calendars found")
	}

	// Apply filters
	filteredCalendars := c.filterCalendars(calendars)
	if len(filteredCalendars) == 0 {
		return nil, fmt.Errorf("no calendars match the specified filters")
	}

	if progressCallback != nil {
		progressCallback(fmt.Sprintf("Found %d calendar(s) to process", len(filteredCalendars)), 1, 1)
	}

	// Process each calendar
	var allEvents []*ics.VEvent
	var allTodos []*ics.VTodo
	seenUIDs := make(map[string]bool)
	duplicatesFound := 0

	for i, calendar := range filteredCalendars {
		if progressCallback != nil {
			progressCallback(fmt.Sprintf("Processing calendar: %s", calendar.DisplayName), i+1, len(filteredCalendars))
		}

		// Create a temporary client for this specific calendar
		tempClient := &Client{
			webdavClient:           c.webdavClient,
			httpClient:             c.httpClient,
			baseURL:                calendar.URL,
			username:               c.username,
			password:               c.password,
			startDate:              c.startDate,
			endDate:                c.endDate,
			useOAuth:               c.useOAuth,
			useServerSideFiltering: c.useServerSideFiltering,
		}

		var calendarResult *DeduplicationResult
		var calendarErr error

		if c.useServerSideFiltering {
			// Try server-side filtering for this calendar
			calendarResult, calendarErr = tempClient.GetEventsWithServerSideFilteringAndProgress(nil)
			if calendarErr != nil {
				fmt.Printf("Warning: Server-side filtering failed for calendar %s (%v), trying client-side\n",
					calendar.DisplayName, calendarErr)
				calendarResult, calendarErr = tempClient.getEventsWithClientSideFilteringAndProgress(nil)
			}
		} else {
			// Use client-side filtering for this calendar
			calendarResult, calendarErr = tempClient.getEventsWithClientSideFilteringAndProgress(nil)
		}

		if calendarErr != nil {
			fmt.Printf("Warning: Failed to process calendar %s: %v\n", calendar.DisplayName, calendarErr)
			continue
		}

		// Merge results with global deduplication
		for _, event := range calendarResult.Events {
			uid := c.getEventUID(event)
			if uid != "" && seenUIDs[uid] {
				duplicatesFound++
				continue
			}
			if uid != "" {
				seenUIDs[uid] = true
			}
			allEvents = append(allEvents, event)
		}

		for _, todo := range calendarResult.Todos {
			uid := c.getTodoUID(todo)
			if uid != "" && seenUIDs[uid] {
				duplicatesFound++
				continue
			}
			if uid != "" {
				seenUIDs[uid] = true
			}
			allTodos = append(allTodos, todo)
		}

		// Add local duplicates to the count
		duplicatesFound += calendarResult.DuplicatesFound
	}

	return &DeduplicationResult{
		Events:          allEvents,
		Todos:           allTodos,
		DuplicatesFound: duplicatesFound,
	}, nil
}