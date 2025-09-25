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
- **`pkg/caldav/report.go`**: CalDAV REPORT query implementation for server-side filtering using RFC 4791-compliant calendar-query requests
- **`pkg/caldav/discovery.go`**: CalDAV calendar discovery implementation using PROPFIND requests for multi-calendar support
- **`pkg/markdown/formatter.go`**: Converts iCalendar components to Markdown format, handles both VEVENT and VTODO items with YAML frontmatter support
- **`pkg/config/config.go`**: Configuration management supporting both environment files and CLI flags, with multi-calendar and OAuth options
- **`pkg/rrule/rrule.go`**: Recurrence rule (RRULE) parsing and recurring event expansion engine
- **`pkg/oauth/client.go`**: Google OAuth 2.0 client implementation for CalDAV authentication

### Data Flow

1. **Configuration**: Load from `.env` file or CLI flags (CalDAV URL, credentials, OAuth settings, multi-calendar options, output directory, date range)
2. **Authentication**: OAuth 2.0 flow for Google Calendar or basic authentication for other CalDAV servers
3. **Calendar Discovery** (optional): Use PROPFIND requests to discover all calendar collections on the server
4. **CalDAV Fetching**:
   - **Server-side filtering**: Use CalDAV REPORT queries with time-range filters (preferred for performance)
   - **Client-side filtering**: Fallback to reading all `.ics` files and filtering locally
   - **Multi-calendar**: Process multiple calendars with individual filtering and global deduplication
5. **Processing**:
   - Extract VEVENT items (filtered by configurable date range)
   - Expand recurring events using RRULE parsing
   - Extract VTODO items with category and status information
   - Global UID-based deduplication across all calendars and expanded instances
6. **Output Generation**:
   - Daily aggregated markdown files: `YYYY-MM-DD.md` containing all events and tasks for that day
   - Optional YAML frontmatter with metadata (date, event counts, categories, etc.)
   - Smart file merging: updates existing files instead of overwriting, preserving manual edits
   - Tasks are integrated into daily files by due date, no separate tasks.md file

### Key Implementation Details

#### Core Features
- **Smart File Merging**: Files are updated instead of overwritten, preserving manual edits and custom content while adding new calendar data
- **Progress Indicators**: Real-time progress reporting during CalDAV fetching and file generation operations
- **Date Filtering**: Configurable date range filtering with default start date of 2000-01-01 and end date 2 years from now. Events with invalid/unparseable dates are preserved rather than filtered out
- **Zero Date Handling**: Events and todos with 0001-01-01 dates (placeholder/unspecified dates) are now processed and included
- **Recurring Events**: Full RRULE support for expanding recurring events including FREQ, INTERVAL, COUNT, UNTIL, BYDAY, BYMONTH
  - **Smart Date Range Processing**: Recurring events and tasks are expanded first, then filtered by date range, ensuring recurring items with start dates outside the filter range are still processed if they have instances within the range
  - **Recurring Task Support**: Both VEVENT and VTODO components support RRULE expansion with proper due date and start date handling
- **Deduplication**: Uses UID properties to prevent duplicate events/todos across all calendar files and recurring event instances, includes duplicate detection during file merging
- **Enhanced Time Parsing**: Supports multiple iCalendar date/time formats including UTC times with Z suffix (`20060102T150405Z`), local times without Z suffix (`20060102T150405`), and all-day events (`20060102`) with automatic end-time calculation
- **Duration Handling**: Automatic calculation of event durations, with 1-hour default for events without end times
- **iCalendar DURATION Support**: Parses RFC 5545 DURATION properties (e.g., `PT2H30M`)

#### Authentication & Security
- **OAuth 2.0 Support**: Full Google Calendar OAuth integration with automatic token management and refresh
- **Token Storage**: Secure token persistence in `~/.config/caldav2markdown/token.json` with proper file permissions
- **Automatic Refresh**: Handles expired tokens with automatic refresh using refresh tokens
- **Basic Auth Fallback**: Maintains compatibility with traditional username/password authentication

#### Performance Optimizations
- **Server-side Filtering**: CalDAV REPORT queries with time-range filters reduce network traffic and processing
- **Multi-calendar Discovery**: RFC 4791-compliant calendar collection discovery via PROPFIND requests
- **Parallel Processing**: Concurrent calendar processing with global deduplication
- **Smart Caching**: Intelligent file merging reduces redundant processing

#### Output & Formatting
- **Directory Structure**: Daily markdown files organized in YYYY/MM directory tree, with zero-date events in `0001/01/`
- **Daily Aggregation**: Events and tasks are grouped by date and saved as daily markdown files with list format, including separate sections for all-day events, scheduled events, and tasks
- **YAML Frontmatter**: Optional structured metadata including date, event counts, categories, and tags
- **Flexible Formatting Options**:
  - Optional 📅 emoji for due dates (controlled by `USE_DUE_DATE_EMOJI` config)
  - Optional #event and #task hashtags (controlled by `USE_HASHTAGS` config)
  - YAML frontmatter with comprehensive metadata (controlled by `USE_FRONTMATTER` config)
- **Todo Integration**: Tasks are organized by due date and included in daily files rather than separate files

#### Multi-Calendar Support
- **Calendar Discovery**: Automatic discovery of all calendar collections on a CalDAV server
- **Include/Exclude Filters**: Flexible filtering to process only specific calendars by name or URL pattern
- **Global Deduplication**: UID-based deduplication across multiple calendars prevents duplicate events
- **Individual Calendar Processing**: Each calendar processed independently with fallback error handling

### Recent Major Updates

#### 2025 Recurring Events & Tasks Enhancement
- **Fixed Recurring Event Filtering**: Recurring events and tasks are now properly processed when their original start date falls outside the configured date filter range
- **Smart Processing Order**: Changed processing logic to expand recurring items first, then filter individual instances by date, ensuring no valid occurrences are missed
- **Enhanced RRULE Support**: Added full recurring task (VTODO) support with `ExpandTodo()` function matching VEVENT capabilities
- **Improved Date Range Logic**: All processors (CalDAV client, CalDAV report, ICS processor) now use consistent expand-then-filter approach
- **Comprehensive Task Support**: Fixed task-only calendar processing and added recurring task expansion with due date and start date handling

#### 2025 ICS File Support & Multi-Source Processing
- **Local and Remote ICS Files**: Added comprehensive support for both local `.ics` files and remote HTTP/HTTPS ICS URLs with multiple authentication methods
- **Enhanced File Merging**: Improved frontmatter preservation and intelligent merging when updating existing markdown files
- **Task-Only Calendar Support**: Fixed early exit logic that was ignoring calendars containing only tasks (VTODO components) with no events
- **Multi-Authentication Support**: Support for no auth, basic auth, bearer tokens, and custom headers for remote ICS sources

#### 2025 Authentication & Security Updates
- **Google OAuth 2.0**: Full implementation for Google Calendar CalDAV access (mandatory as of March 2025)
- **Token Management**: Automatic token storage, refresh, and browser-based authorization flow
- **Security Compliance**: Meets Google's updated security requirements for calendar access

#### Performance & Scalability Features
- **Server-side Filtering**: CalDAV REPORT queries with time-range filters (RFC 4791 compliant)
- **Multi-calendar Support**: Discover and process multiple calendars from a single CalDAV server
- **Parallel Processing**: Concurrent calendar processing with intelligent error handling and fallbacks

#### Enhanced Output Features
- **YAML Frontmatter**: Structured metadata in markdown files for static site generators and note-taking apps
- **Smart Formatting**: Emoji support, hashtags, and configurable display options
- **Category Integration**: Calendar categories/tags are preserved and included in frontmatter

#### Discovery & Automation
- **Calendar Discovery**: Automatic discovery of all calendar collections using PROPFIND requests
- **Flexible Filtering**: Include/exclude specific calendars by name or URL pattern
- **Calendar Listing**: `-list-calendars` command to preview available calendars before processing

#### Backward Compatibility
- **Legacy Support**: All existing configurations and workflows continue to work unchanged
- **Graceful Fallbacks**: Server-side filtering falls back to client-side when not supported
- **Configuration Migration**: New features are opt-in with sensible defaults

### Dependencies

- `github.com/arran4/golang-ical`: iCalendar parsing and component handling
- `github.com/studio-b12/gowebdav`: WebDAV/CalDAV client for basic authentication and file operations
- `golang.org/x/oauth2`: OAuth 2.0 client implementation with Google provider support
- `gopkg.in/yaml.v3`: YAML marshaling for frontmatter generation

### Supported Calendar Components

The application processes the following iCalendar component types:
- **VEVENT**: Calendar events with full recurring event support
- **VTODO**: Tasks and todos with status, priority, and due date handling

#### Calendar Types Supported
- **Event-only calendars**: Calendars containing only VEVENT components
- **Task-only calendars**: Calendars containing only VTODO components (useful for task management systems)
- **Mixed calendars**: Calendars containing both events and tasks
- **Empty calendars**: Gracefully handled with appropriate messaging

#### Task-Only Calendar Features
- **Due date handling**: Tasks are organized by due date in daily files
- **No due date tasks**: Tasks without due dates are placed in "0001/01/0001-01-01.md" (Date TBD)
- **Task status**: Supports NEEDS-ACTION, IN-PROCESS, COMPLETED, and other standard statuses
- **Completed tasks**: Properly marked with `[x]` checkbox in markdown
- **Priority levels**: Task priorities (1-9) are displayed and can be used for filtering
- **Categories**: Task categories are preserved in YAML frontmatter tags

Other iCalendar component types (VJOURNAL, VFREEBUSY, VTIMEZONE, etc.) are ignored during processing.

### Configuration Examples

#### Google Calendar with OAuth
```bash
USE_OAUTH=true
CALDAV_URL=https://apidata.googleusercontent.com/caldav/v2/primary/events
GOOGLE_CLIENT_ID=your_client_id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your_client_secret
USE_FRONTMATTER=true
DISCOVER_CALENDARS=true
```

#### Multi-calendar with Server-side Filtering
```bash
DISCOVER_CALENDARS=true
USE_SERVER_SIDE_FILTERING=true
INCLUDE_CALENDARS=Work,Personal,Family
EXCLUDE_CALENDARS=Archive,Test,Spam
USE_HASHTAGS=true
USE_FRONTMATTER=true
```

#### Traditional CalDAV Server
```bash
CALDAV_URL=https://your-server.com/caldav/calendars/username/calendar/
CALDAV_USERNAME=username
CALDAV_PASSWORD=password
USE_DUE_DATE_EMOJI=true
OUTPUT_DIR=./calendar-notes
```

#### Task-Only Calendar (ICS Mode)
```bash
SOURCE_MODE=ics
ICS_PATH=./tasks.ics
# Or for remote task management systems:
# ICS_URL=https://tasks.example.com/export/my-tasks.ics
# ICS_AUTH=basic
# ICS_USERNAME=username
# ICS_PASSWORD=password
OUTPUT_DIR=./task-notes
USE_FRONTMATTER=true
USE_HASHTAGS=true
USE_DUE_DATE_EMOJI=true
```