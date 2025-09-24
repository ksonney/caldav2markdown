# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Development Commands

```bash
# Build the application
go build -o caldav2markdown ./cmd/caldav2markdown

# Run the application
./caldav2markdown

# Test connection to CalDAV server
./caldav2markdown -test

# Run with custom configuration
./caldav2markdown -config myconfig.env

# Build and run in one command
go run ./cmd/caldav2markdown
```

## Architecture Overview

This is a Go application that converts CalDAV calendar data to Markdown files. The architecture follows a clean separation of concerns:

### Package Structure

- **`cmd/caldav2markdown/main.go`**: Main application entry point, handles CLI flags, configuration loading, and orchestrates the conversion process
- **`pkg/caldav/client.go`**: CalDAV client implementation using WebDAV protocol to fetch .ics files, handles deduplication, date filtering, and recurring event expansion
- **`pkg/markdown/formatter.go`**: Converts iCalendar components to Markdown format, handles both VEVENT and VTODO items with enhanced duration handling
- **`pkg/config/config.go`**: Configuration management supporting both environment files and CLI flags, with configurable date ranges
- **`pkg/rrule/rrule.go`**: Recurrence rule (RRULE) parsing and recurring event expansion engine

### Data Flow

1. **Configuration**: Load from `.env` file or CLI flags (CalDAV URL, credentials, output directory, date range)
2. **CalDAV Fetching**: Connect to server, read all `.ics` files, parse with `github.com/arran4/golang-ical`
3. **Processing**:
   - Extract VEVENT items (filtered by configurable date range)
   - Expand recurring events using RRULE parsing
   - Extract VTODO items
   - Deduplicate by UID across all calendar files and expanded instances
4. **Output Generation**:
   - Individual markdown files for each event: `YYYY-MM-DD_Event-Title.md`
   - Single `tasks.md` file containing all todos as checkboxes

### Key Implementation Details

- **Date Filtering**: Configurable date range filtering with default start date of 2000-01-01 and end date 2 years from now
- **Zero Date Handling**: Events and todos with 0001-01-01 dates (placeholder/unspecified dates) are now processed and included
- **Recurring Events**: Full RRULE support for expanding recurring events including FREQ, INTERVAL, COUNT, UNTIL, BYDAY, BYMONTH
- **Deduplication**: Uses UID properties to prevent duplicate events/todos across all calendar files and recurring event instances
- **Enhanced Time Parsing**: Supports both timed events (`20060102T150405Z`) and all-day events (`20060102`) with automatic end-time calculation
- **Duration Handling**: Automatic calculation of event durations, with 1-hour default for events without end times
- **Directory Structure**: Markdown files organized in YYYY/MM directory tree, with zero-date events in `0001/01/`
- **Todo Format**: Uses markdown checkboxes with priority, due dates, completion status, and descriptions
- **iCalendar DURATION Support**: Parses RFC 5545 DURATION properties (e.g., `PT2H30M`)

### Recent Enhancements

- **Zero Date Support**: Removed filters that excluded events with 0001-01-01 dates - these events are now processed and saved in `0001/01/` directory
- **Directory Organization**: Implemented YYYY/MM directory tree structure for better file organization
- **Recurring Event Support**: Added comprehensive RRULE parsing and event expansion (`pkg/rrule/rrule.go`)
- **Enhanced Event Processing**: Events without end times now automatically get appropriate durations
- **Improved Output**: Markdown now shows duration calculations and better formatting for all-day events and zero-date events
- **Test Coverage**: Added comprehensive unit tests for formatter functionality (`pkg/markdown/formatter_test.go`)
- **Date Range Configuration**: Configurable start and end dates via environment variables (`START_DATE`, `END_DATE`)

### Dependencies

- `github.com/arran4/golang-ical`: iCalendar parsing
- `github.com/studio-b12/gowebdav`: WebDAV/CalDAV client

The application processes only VEVENT and VTODO components, ignoring other iCalendar component types like VJOURNAL, VFREEBUSY, etc.