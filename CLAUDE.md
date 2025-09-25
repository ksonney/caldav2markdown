# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Development Commands

### Using Make (Recommended)

```bash
# Build the application (creates bin/caldav2markdown)
make build

# Build and run the application
make run

# Test connection to CalDAV server
make test-connection

# Run with custom configuration
make run-config CONFIG_FILE=myconfig.env

# Run tests
make test

# Clean build artifacts
make clean

# Cross-platform builds
make build-all          # All platforms
make build-linux        # Linux AMD64
make build-darwin       # macOS AMD64 and ARM64
make build-windows      # Windows AMD64

# Development build with debug info
make build-dev

# Show all available targets
make help
```

### Manual Build Commands

```bash
# Build the application
go build -o bin/caldav2markdown ./cmd/caldav2markdown

# Run the application
bin/caldav2markdown

# Test connection to CalDAV server
bin/caldav2markdown -test

# Run with custom configuration
bin/caldav2markdown -config myconfig.env

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
   - Daily aggregated markdown files: `YYYY-MM-DD.md` containing all events and tasks for that day
   - Smart file merging: updates existing files instead of overwriting, preserving manual edits
   - Tasks are integrated into daily files by due date, no separate tasks.md file

### Key Implementation Details

- **Smart File Merging**: Files are updated instead of overwritten, preserving manual edits and custom content while adding new calendar data
- **Progress Indicators**: Real-time progress reporting during CalDAV fetching and file generation operations
- **Date Filtering**: Configurable date range filtering with default start date of 2000-01-01 and end date 2 years from now. Events with invalid/unparseable dates are preserved rather than filtered out
- **Zero Date Handling**: Events and todos with 0001-01-01 dates (placeholder/unspecified dates) are now processed and included
- **Recurring Events**: Full RRULE support for expanding recurring events including FREQ, INTERVAL, COUNT, UNTIL, BYDAY, BYMONTH
- **Deduplication**: Uses UID properties to prevent duplicate events/todos across all calendar files and recurring event instances, includes duplicate detection during file merging
- **Enhanced Time Parsing**: Supports multiple iCalendar date/time formats including UTC times with Z suffix (`20060102T150405Z`), local times without Z suffix (`20060102T150405`), and all-day events (`20060102`) with automatic end-time calculation
- **Duration Handling**: Automatic calculation of event durations, with 1-hour default for events without end times
- **Directory Structure**: Daily markdown files organized in YYYY/MM directory tree, with zero-date events in `0001/01/`
- **Daily Aggregation**: Events and tasks are grouped by date and saved as daily markdown files with list format, including separate sections for all-day events, scheduled events, and tasks
- **Flexible Formatting Options**:
  - Optional 📅 emoji for due dates (controlled by `USE_DUE_DATE_EMOJI` config)
  - Optional #event and #task hashtags (controlled by `USE_HASHTAGS` config)
- **Todo Integration**: Tasks are organized by due date and included in daily files rather than separate files
- **iCalendar DURATION Support**: Parses RFC 5545 DURATION properties (e.g., `PT2H30M`)

### Recent Enhancements

#### Core Functionality
- **Smart File Merging**: Implemented intelligent file merging system that preserves existing content while adding new calendar data (`pkg/markdown/formatter.go:420-523`)
- **Task Integration**: Tasks are now integrated into daily files by due date instead of separate tasks.md file
- **Progress Indicators**: Added comprehensive progress reporting system with callbacks for real-time feedback during long operations (`pkg/caldav/client.go:78-79`, `pkg/markdown/formatter.go:620-621`)

#### Formatting and Display Options
- **Emoji Support**: Optional 📅 emoji for due dates via `USE_DUE_DATE_EMOJI` configuration (`pkg/config/config.go:70-71`)
- **Hashtag Support**: Optional #event and #task hashtags via `USE_HASHTAGS` configuration (`pkg/config/config.go:72-73`)
- **Enhanced CLI Options**: Added `-emoji` and `-hashtags` command-line flags for formatting control

#### File Processing Improvements
- **Deduplication During Merge**: Added duplicate detection and removal during file merging process
- **Content Preservation**: Manual edits and custom sections in markdown files are preserved during updates
- **Invalid Date Handling**: Modified date filtering logic to preserve events with unparseable/invalid date formats instead of filtering them out (`pkg/caldav/client.go:158`, `pkg/caldav/client.go:183`)

#### Infrastructure and Quality
- **Build System**: Added comprehensive Makefile with targets for building, testing, cross-compilation, and development workflows
- **Test Coverage**: Added comprehensive unit tests for formatter functionality, file merging, and progress callbacks (`pkg/markdown/formatter_test.go`)
- **Configuration Management**: Enhanced environment configuration with new formatting options

#### Legacy Enhancements
- **Zero Date Support**: Removed filters that excluded events with 0001-01-01 dates - these events are now processed and saved in `0001/01/` directory
- **Directory Organization**: Implemented YYYY/MM directory tree structure for better file organization
- **Recurring Event Support**: Added comprehensive RRULE parsing and event expansion (`pkg/rrule/rrule.go`)
- **Enhanced Event Processing**: Events without end times now automatically get appropriate durations
- **Date Range Configuration**: Configurable start and end dates via environment variables (`START_DATE`, `END_DATE`)
- **Improved Date/Time Parsing**: Enhanced parsing logic to handle iCalendar date/time formats without Z suffix, supporting both UTC (`20060102T150405Z`) and local time (`20060102T150405`) formats across all components (VEVENT, VTODO, RRULE)
- **Daily Event Aggregation**: Changed from individual event files to daily aggregated files (`YYYY-MM-DD.md`) containing all events for a specific date in markdown list format, with automatic sorting and separate sections for all-day and scheduled events

### Dependencies

- `github.com/arran4/golang-ical`: iCalendar parsing
- `github.com/studio-b12/gowebdav`: WebDAV/CalDAV client

The application processes only VEVENT and VTODO components, ignoring other iCalendar component types like VJOURNAL, VFREEBUSY, etc.