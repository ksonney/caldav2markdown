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
- **`pkg/caldav/client.go`**: CalDAV client implementation using WebDAV protocol to fetch .ics files, handles deduplication and date filtering (only processes events after 2000-01-01)
- **`pkg/markdown/formatter.go`**: Converts iCalendar components to Markdown format, handles both VEVENT and VTODO items
- **`pkg/config/config.go`**: Configuration management supporting both environment files and CLI flags

### Data Flow

1. **Configuration**: Load from `.env` file or CLI flags (CalDAV URL, credentials, output directory)
2. **CalDAV Fetching**: Connect to server, read all `.ics` files, parse with `github.com/arran4/golang-ical`
3. **Processing**:
   - Extract VEVENT items (filtered to dates after 2000-01-01)
   - Extract VTODO items
   - Deduplicate by UID across all calendar files
4. **Output Generation**:
   - Individual markdown files for each event: `YYYY-MM-DD_Event-Title.md`
   - Single `tasks.md` file containing all todos as checkboxes

### Key Implementation Details

- **Date Filtering**: Only processes events with DTSTART on or after 2000-01-01 (`pkg/caldav/client.go:isEventAfter2000`)
- **Deduplication**: Uses UID properties to prevent duplicate events/todos across multiple calendar files
- **Time Parsing**: Supports both timed events (`20060102T150405Z`) and all-day events (`20060102`)
- **Todo Format**: Uses markdown checkboxes with priority, due dates, and completion status

### Dependencies

- `github.com/arran4/golang-ical`: iCalendar parsing
- `github.com/studio-b12/gowebdav`: WebDAV/CalDAV client

The application processes only VEVENT and VTODO components, ignoring other iCalendar component types like VJOURNAL, VFREEBUSY, etc.