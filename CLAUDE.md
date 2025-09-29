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
   - Daily aggregated markdown files: `YYYY-MM-DD.md` containing all events and tasks with due dates
   - Separate `todo.md` file for tasks without due dates with intelligent merging
   - Optional YAML frontmatter with metadata (date, event counts, categories, etc.) supporting custom field preservation
   - Smart file merging: updates existing files instead of overwriting, preserving manual edits and custom frontmatter
   - Intelligent frontmatter merging preserves user customizations while updating calendar statistics

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
- **Time Zone Support**: Full TZID parameter extraction and processing with IANA time zone database integration and common Microsoft Exchange time zone mappings
- **Duration Handling**: Automatic calculation of event durations, with 1-hour default for events without end times
- **iCalendar DURATION Support**: Parses RFC 5545 DURATION properties (e.g., `PT2H30M`)
- **Smart Todo Management**: Tasks without due dates are separated and saved to `todo.md` with intelligent merging and frontmatter preservation
- **Past Event Detection**: Automatic detection and marking of past events when EVENT_CHECKBOXES is enabled, using end time for timed events and date comparison for all-day events

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
  - **Calendar Alias Hashtags**: Automatic hashtags from calendar names/aliases (controlled by `USE_CALENDAR_TAGS` config)
  - YAML frontmatter with comprehensive metadata (controlled by `USE_FRONTMATTER` config)
  - Optional description exclusion for cleaner output (controlled by `IGNORE_DESCRIPTIONS` config)
  - **Ignore Declined Events**: Automatically skip declined events (STATUS=CANCELLED or PARTSTAT=DECLINED) (controlled by `IGNORE_DECLINED` config)
  - Optional event checkboxes for task-like formatting (controlled by `EVENT_CHECKBOXES` config)
  - **Calendar Name Display**: Automatic calendar name extraction with customizable aliases (controlled by `CALENDAR_ALIASES` config)
  - **Obsidian Tasks Emoji Format**: Use 🛫 for start times and ✅ for end times in Obsidian Tasks format (controlled by `USE_OBSIDIAN_EMOJIS` config)
  - **Obsidian Tasks Preset**: One-click configuration for Obsidian compatibility - enables event checkboxes, ignores descriptions, frontmatter, emojis, hashtags, calendar tags, and Obsidian time emojis (controlled by `OBSIDIAN_TASKS` config)
  - **Past Event Completion**: Automatic [x] marking for past events when EVENT_CHECKBOXES is enabled
- **Smart Todo Organization**:
  - Tasks with due dates are organized by date and included in daily files
  - Tasks without due dates are saved to separate `todo.md` file with intelligent merging
  - Custom frontmatter preservation in todo.md maintains user customizations

#### Multi-Calendar Support
- **Calendar Discovery**: Automatic discovery of all calendar collections on a CalDAV server
- **Include/Exclude Filters**: Flexible filtering to process only specific calendars by name or URL pattern
- **Global Deduplication**: UID-based deduplication across multiple calendars prevents duplicate events
- **Individual Calendar Processing**: Each calendar processed independently with fallback error handling

### Recent Major Updates

#### Latest Technical Enhancements (2025)
- **Calendar Discovery De-duplication**: Automatic removal of duplicate calendar collections using normalized URL comparison (case-insensitive, trailing slash handling) to prevent processing the same calendar multiple times
- **Smart URL Filtering**: Intelligent filtering of non-calendar endpoints including ICS files, XML exports, and download URLs to focus only on proper CalDAV calendar collections
- **Enhanced CalDAV Discovery**: Improved calendar collection detection with fallback to supported calendar components when resource type is missing or blank, includes comprehensive multistatus response handling with proper error reporting
- **Smart Calendar Detection**: Calendars are now detected via supported components (VEVENT, VTODO, VJOURNAL, VFREEBUSY) when explicit calendar resource types are missing, preventing missed calendars from legacy or non-compliant servers
- **Robust Multistatus Handling**: Complete rewrite of CalDAV discovery and calendar-query response parsing with multiple propstat support, individual status code checking, and detailed error reporting
- **Fixed Autodiscovery Configuration**: Resolved validation issue where autodiscovery would fail when SOURCE_MODE wasn't explicitly set, now properly defaults to CalDAV mode for backward compatibility
- **Calendar Name Display & Aliases**: Added calendar name extraction from ICS properties with configurable alias mapping for cleaner display
- **Time Zone Support**: Added comprehensive TZID parameter support with IANA time zone database integration and common time zone mappings for Microsoft Exchange compatibility
- **Smart Todo Management**: Implemented separate `todo.md` file generation for tasks without due dates, with intelligent file merging and custom frontmatter preservation
- **Enhanced Frontmatter Merging**: Complete rewrite of frontmatter merging logic to preserve custom fields while updating calendar-specific statistics
- **Past Event Marking**: Automatic completion marking for past events when EVENT_CHECKBOXES is enabled, with time-aware status detection

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
- **Calendar Name Display**: Events and tasks show their source calendar with customizable aliases for cleaner display

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
- **Due date handling**: Tasks with due dates are organized by date in daily files
- **No due date tasks**: Tasks without due dates are saved to separate `todo.md` file in output directory root
- **Smart file merging**: Todo.md file supports intelligent merging with custom frontmatter preservation
- **Task status**: Supports NEEDS-ACTION, IN-PROCESS, COMPLETED, and other standard statuses
- **Completed tasks**: Properly marked with `[x]` checkbox in markdown
- **Priority levels**: Task priorities (1-9) are displayed and can be used for filtering
- **Categories**: Task categories are preserved in YAML frontmatter tags

Other iCalendar component types (VJOURNAL, VFREEBUSY, VTIMEZONE, etc.) are ignored during processing.

### Technical Implementation Details

#### Enhanced CalDAV Discovery Architecture
- **Function**: `isCalendarCollection(prop PropFindResponseProp, traceWebCalls bool)` - Enhanced calendar detection with component fallback
- **Function**: `parseDiscoveryMultiStatus(multiStatus *PropFindMultiStatus, traceWebCalls bool)` - Robust multistatus parsing with error handling
- **Function**: `getComponentNames(components []CalendarComponent)` - Component name extraction for logging
- **Function**: `deduplicateCalendars(calendars []CalendarInfo, traceWebCalls bool)` - De-duplication and URL filtering
- **Function**: `isValidCalendarURL(url string)` - URL validation to filter out non-calendar endpoints
- **Function**: `normalizeCalendarURL(url string)` - URL normalization for consistent comparison
- **Detection Strategy**:
  - **Primary**: Supported calendar components (VEVENT, VTODO, VJOURNAL, VFREEBUSY) - most reliable indicator
  - **Secondary**: Resource type when components unavailable - conservative approach to prevent false positives
- **MultiStatus Handling**:
  - Multiple propstat elements per response supported
  - Individual status code validation (200 OK vs. 403 Forbidden, 404 Not Found)
  - Comprehensive error reporting with detailed logging when tracing enabled
- **URL Filtering**:
  - Filters out ICS files (.ics), XML exports (.xml), and other file downloads
  - Removes export URLs with query parameters and download patterns
  - Case-insensitive filtering with comprehensive file extension checking
- **De-duplication**:
  - Case-insensitive URL comparison with trailing slash normalization
  - First occurrence wins for duplicate calendar collections
  - Detailed logging of duplicates found and removed when tracing enabled
- **Fallback Support**: Calendars missing explicit `<cal:calendar/>` resource type are detected via supported components
- **Integration**: All discovery functions (`discoverPrincipalURL`, `discoverCalendarHomeSet`, `discoverCalendarCollections`) use enhanced parsing

#### Calendar Name & Alias Architecture
- **Data Structure**: Added `Calendar string` field to both `EventMarkdown` and `TodoMarkdown` structs
- **Configuration**: `CalendarAliases map[string]string` field in Config struct for alias mapping
- **Environment Variable**: `CALENDAR_ALIASES` parsing with format "name1:alias1,name2:alias2"
- **Functions**:
  - `ConvertEventWithCalendar(event *ics.VEvent, calendarName string, calendarAliases map[string]string)` - Converts events with calendar context
  - `ConvertTodoWithCalendar(todo *ics.VTodo, calendarName string, calendarAliases map[string]string)` - Converts todos with calendar context
- **Property Extraction**: Automatic calendar name extraction from `X-WR-CALNAME` and `PRODID` ICS properties
- **Display Integration**: Calendar names shown in square brackets `[CalendarName]` in markdown output between location/status and hashtags

#### Time Zone Support Architecture
- **Function**: `parseICalDateTimeWithTZ(value, tzid string)` - Enhanced date/time parsing with TZID support
- **Function**: `mapTimeZone(tzid string)` - Maps common time zone names to IANA identifiers
- **Integration**: TZID parameters extracted from DTSTART, DTEND, and DUE properties in all processors
- **Fallback**: Graceful degradation when time zones cannot be resolved

#### Todo.md Management
- **Function**: `GenerateTodoFile()` - Creates and manages todo.md with smart merging
- **Function**: `parseExistingTodoFile()` - Parses existing todo.md preserving custom content
- **Function**: `generateMergedTodoContent()` - Intelligent content merging with frontmatter preservation
- **Architecture**: Tasks without due dates separated in `GenerateDailyFilesWithAllOptions()`

#### Enhanced Frontmatter Merging
- **Function**: `generateMergedContentWithFrontmatter()` - Updated to preserve custom fields
- **Function**: `isGeneratedTitle()` - Detects auto-generated vs. custom titles
- **Features**: Preserves custom frontmatter fields while updating calendar-specific statistics
- **Tag Merging**: Intelligent tag aggregation supporting multiple YAML formats

#### Past Event Detection
- **Method**: `EventMarkdown.IsPastEvent()` - Determines if event has occurred
- **Logic**: Uses end time for timed events, date comparison for all-day events
- **Integration**: Automatic [x] marking in `ToListItemWithOptions()` when EVENT_CHECKBOXES enabled
- **Time Awareness**: Considers actual event duration and time zones

#### Calendar Alias Hashtag Generation
- **Functions**:
  - `EventMarkdown.ToListItemWithOptionsAndCalendarTags()` - Generates calendar hashtags for events
  - `TodoMarkdown.ToMarkdownWithOptionsAndCalendarTags()` - Generates calendar hashtags for todos
- **Configuration**: Controlled by `USE_CALENDAR_TAGS` config option and `--calendar-tags` CLI flag
- **Sanitization Logic**:
  - Converts calendar names/aliases to lowercase
  - Replaces spaces and underscores with hyphens
  - Removes non-alphanumeric characters except hyphens
  - Collapses multiple hyphens and trims leading/trailing hyphens
- **Integration**: Works with calendar aliases to create clean, readable hashtags
- **Example**: Calendar "Work Calendar" with alias "Work" → `#work` hashtag

### Configuration Examples

#### Configuration Formats

The application supports two configuration formats:
- **Environment files** (`.env`): Traditional key-value format, backwards compatible
- **YAML files** (`.yaml`, `.yml`): Structured format with better organization and readability

You can convert between formats using the built-in conversion tools:
```bash
# Convert env to YAML
bin/caldav2markdown -convert-to-yaml config.yaml -config .env

# Export current configuration to YAML
bin/caldav2markdown -export-yaml config.yaml -config .env
```

The application auto-detects the format based on file extension or content structure.

### Environment File Examples

#### Google Calendar with OAuth, Calendar Aliases, and Hashtags
```bash
USE_OAUTH=true
CALDAV_URL=https://apidata.googleusercontent.com/caldav/v2/primary/events
GOOGLE_CLIENT_ID=your_client_id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your_client_secret
USE_FRONTMATTER=true
USE_HASHTAGS=true
USE_CALENDAR_TAGS=true
DISCOVER_CALENDARS=true
CALENDAR_ALIASES=Google Calendar:GCal,Personal Calendar:Personal,Work Calendar:Work
```

#### Google Calendar with OAuth and Proxy
```bash
USE_OAUTH=true
CALDAV_URL=https://apidata.googleusercontent.com/caldav/v2/primary/events
GOOGLE_CLIENT_ID=your_client_id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your_client_secret
PROXY_URL=http://proxy.company.com:8080
PROXY_USERNAME=proxy_user
PROXY_PASSWORD=proxy_pass
USE_FRONTMATTER=true
DISCOVER_CALENDARS=true
```

#### Google Calendar with OAuth
```bash
USE_OAUTH=true
CALDAV_URL=https://apidata.googleusercontent.com/caldav/v2/primary/events
GOOGLE_CLIENT_ID=your_client_id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your_client_secret
USE_FRONTMATTER=true
DISCOVER_CALENDARS=true
```

#### Multi-calendar with Server-side Filtering and Calendar Hashtags
```bash
DISCOVER_CALENDARS=true
USE_SERVER_SIDE_FILTERING=true
INCLUDE_CALENDARS=Work,Personal,Family
EXCLUDE_CALENDARS=Archive,Test,Spam
CALENDAR_ALIASES=Work Calendar:Work,Personal Calendar:Personal,Family Events:Family
USE_HASHTAGS=true
USE_CALENDAR_TAGS=true
USE_FRONTMATTER=true
EVENT_CHECKBOXES=true
```

#### Traditional CalDAV Server
```bash
CALDAV_URL=https://your-server.com/caldav/calendars/username/calendar/
CALDAV_USERNAME=username
CALDAV_PASSWORD=password
USE_DUE_DATE_EMOJI=true
IGNORE_DESCRIPTIONS=true
OUTPUT_DIR=./calendar-notes
```

#### CalDAV Server Behind Corporate Proxy
```bash
CALDAV_URL=https://calendar.company.com/caldav/calendars/username/calendar/
CALDAV_USERNAME=username
CALDAV_PASSWORD=password
PROXY_URL=http://proxy.company.com:8080
PROXY_USERNAME=proxy_user
PROXY_PASSWORD=proxy_pass
USE_FRONTMATTER=true
USE_HASHTAGS=true
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

### Multi-Source Configuration Examples

The application supports combining multiple calendar sources in a single configuration file. Each source is configured using indexed environment variables with the pattern `SOURCE_<index>_<field>` or `SOURCE_<index>_<type>_<field>`.

#### Multiple CalDAV Sources
```bash
# Global settings
OUTPUT_DIR=./all-calendars
USE_FRONTMATTER=true
USE_HASHTAGS=true
EVENT_CHECKBOXES=true

# Source 0 - Work Google Calendar
SOURCE_0_TYPE=caldav
SOURCE_0_NAME=Work Calendar
SOURCE_0_CALDAV_URL=https://apidata.googleusercontent.com/caldav/v2/primary/events
SOURCE_0_CALDAV_USE_OAUTH=true
SOURCE_0_CALDAV_CLIENT_ID=work_client_id.apps.googleusercontent.com
SOURCE_0_CALDAV_CLIENT_SECRET=work_client_secret
SOURCE_0_CALDAV_DISCOVER_CALENDARS=true
SOURCE_0_CALDAV_CALENDAR_ALIASES=Work Calendar:Work,Meetings:Meet

# Source 1 - Personal Nextcloud
SOURCE_1_TYPE=caldav
SOURCE_1_NAME=Personal Calendar
SOURCE_1_CALDAV_URL=https://nextcloud.example.com/remote.php/dav/calendars/user/personal/
SOURCE_1_CALDAV_USERNAME=user
SOURCE_1_CALDAV_PASSWORD=password
SOURCE_1_CALDAV_USE_SERVER_SIDE_FILTERING=true
SOURCE_1_CALDAV_CALENDAR_ALIASES=Personal:Personal,Family:Fam
```

#### Mixed CalDAV and ICS Sources
```bash
# Global settings
OUTPUT_DIR=./mixed-calendars
USE_FRONTMATTER=true
USE_HASHTAGS=true
USE_DUE_DATE_EMOJI=true

# Source 0 - Google Calendar (OAuth)
SOURCE_0_TYPE=caldav
SOURCE_0_NAME=Google Calendar
SOURCE_0_CALDAV_URL=https://apidata.googleusercontent.com/caldav/v2/primary/events
SOURCE_0_CALDAV_USE_OAUTH=true
SOURCE_0_CALDAV_CLIENT_ID=google_client_id
SOURCE_0_CALDAV_CLIENT_SECRET=google_client_secret

# Source 1 - Local ICS file
SOURCE_1_TYPE=ics
SOURCE_1_NAME=Local Events
SOURCE_1_ICS_TYPE=local
SOURCE_1_ICS_PATH=/home/user/calendars/events.ics

# Source 2 - Remote ICS with authentication
SOURCE_2_TYPE=ics
SOURCE_2_NAME=Company Calendar
SOURCE_2_ICS_TYPE=remote
SOURCE_2_ICS_PATH=https://company.com/calendar/export.ics
SOURCE_2_ICS_AUTH=basic
SOURCE_2_ICS_USERNAME=employee
SOURCE_2_ICS_PASSWORD=company_pass

# Source 3 - Public ICS feed
SOURCE_3_TYPE=ics
SOURCE_3_NAME=Holidays
SOURCE_3_ICS_TYPE=remote
SOURCE_3_ICS_PATH=https://calendar.google.com/calendar/ical/en.usa%23holiday%40group.v.calendar.google.com/public/basic.ics
SOURCE_3_ICS_AUTH=none
```

#### Multi-Source with Proxy and Custom Headers
```bash
# Global settings
OUTPUT_DIR=./enterprise-calendars
USE_FRONTMATTER=true
EVENT_CHECKBOXES=true

# Source 0 - Corporate CalDAV through proxy
SOURCE_0_TYPE=caldav
SOURCE_0_NAME=Corporate Calendar
SOURCE_0_CALDAV_URL=https://calendar.corp.com/caldav/
SOURCE_0_CALDAV_USERNAME=employee
SOURCE_0_CALDAV_PASSWORD=corp_password
SOURCE_0_CALDAV_PROXY_URL=http://proxy.corp.com:8080
SOURCE_0_CALDAV_PROXY_USERNAME=proxy_user
SOURCE_0_CALDAV_PROXY_PASSWORD=proxy_pass
SOURCE_0_CALDAV_DISCOVER_CALENDARS=true
SOURCE_0_CALDAV_INCLUDE_CALENDARS=Work,Projects
SOURCE_0_CALDAV_EXCLUDE_CALENDARS=Archive,Test

# Source 1 - External API with custom headers
SOURCE_1_TYPE=ics
SOURCE_1_NAME=CRM Calendar
SOURCE_1_ICS_TYPE=remote
SOURCE_1_ICS_PATH=https://crm.company.com/api/calendar/export
SOURCE_1_ICS_AUTH=header
SOURCE_1_ICS_HEADER_Authorization=Bearer your_api_token
SOURCE_1_ICS_HEADER_X-Client-ID=caldav2markdown
SOURCE_1_ICS_TIMEOUT=60s
```

#### Task Management Multi-Source Setup
```bash
# Global settings
OUTPUT_DIR=./task-management
USE_FRONTMATTER=true
USE_HASHTAGS=true
EVENT_CHECKBOXES=true
OBSIDIAN_TASKS=true

# Source 0 - Google Tasks via CalDAV
SOURCE_0_TYPE=caldav
SOURCE_0_NAME=Google Tasks
SOURCE_0_CALDAV_URL=https://apidata.googleusercontent.com/caldav/v2/primary/events
SOURCE_0_CALDAV_USE_OAUTH=true
SOURCE_0_CALDAV_CLIENT_ID=google_client_id
SOURCE_0_CALDAV_CLIENT_SECRET=google_client_secret

# Source 1 - Local todo file
SOURCE_1_TYPE=ics
SOURCE_1_NAME=Personal Tasks
SOURCE_1_ICS_TYPE=local
SOURCE_1_ICS_PATH=/home/user/todos/personal.ics

# Source 2 - Project management system export
SOURCE_2_TYPE=ics
SOURCE_2_NAME=Project Tasks
SOURCE_2_ICS_TYPE=remote
SOURCE_2_ICS_PATH=https://project.company.com/calendar/tasks.ics
SOURCE_2_ICS_AUTH=bearer
SOURCE_2_ICS_TOKEN=project_api_token

# Source 3 - Team calendar
SOURCE_3_TYPE=caldav
SOURCE_3_NAME=Team Calendar
SOURCE_3_CALDAV_URL=https://team.calendar.com/caldav/shared/
SOURCE_3_CALDAV_USERNAME=team_user
SOURCE_3_CALDAV_PASSWORD=team_pass
SOURCE_3_CALDAV_CALENDAR_ALIASES=Team Events:Team,Deadlines:Due
```

### Multi-Source Configuration Field Reference

#### Global Source Fields
- `SOURCE_<index>_TYPE`: Source type (`caldav` or `ics`)
- `SOURCE_<index>_NAME`: Display name for the source (optional)

#### CalDAV Source Fields
All CalDAV source fields are prefixed with `SOURCE_<index>_CALDAV_`:
- `URL`: CalDAV server URL (required)
- `USERNAME`: Username for basic authentication
- `PASSWORD`: Password for basic authentication
- `USE_OAUTH`: Enable OAuth 2.0 authentication (`true`/`false`)
- `CLIENT_ID`: OAuth client ID (required for OAuth)
- `CLIENT_SECRET`: OAuth client secret (required for OAuth)
- `USE_SERVER_SIDE_FILTERING`: Enable server-side date filtering (`true`/`false`)
- `DISCOVER_CALENDARS`: Auto-discover calendar collections (`true`/`false`)
- `INCLUDE_CALENDARS`: Comma-separated list of calendars to include
- `EXCLUDE_CALENDARS`: Comma-separated list of calendars to exclude
- `CALENDAR_ALIASES`: Calendar name aliases (`name1:alias1,name2:alias2`)
- `PROXY_URL`: HTTP proxy URL
- `PROXY_USERNAME`: Proxy authentication username
- `PROXY_PASSWORD`: Proxy authentication password

#### ICS Source Fields
All ICS source fields are prefixed with `SOURCE_<index>_ICS_`:
- `TYPE`: Source type (`local` or `remote`)
- `PATH`: File path for local files or URL for remote files (required)
- `AUTH`: Authentication method (`none`, `basic`, `bearer`, `header`)
- `USERNAME`: Username for basic authentication
- `PASSWORD`: Password for basic authentication
- `TOKEN`: Bearer token for bearer authentication
- `TIMEOUT`: HTTP timeout for remote sources (e.g., `30s`, `1m`)
- `HEADER_<name>`: Custom HTTP headers (e.g., `HEADER_Authorization`)

### Multi-Source Processing Features

#### Global Deduplication
Events and tasks with identical UIDs are automatically deduplicated across all sources, ensuring no duplicate entries in the output even when the same calendar is accessed through multiple sources.

#### Source Attribution
Each event and task in the generated markdown includes the source calendar name in square brackets `[SourceName]`, making it easy to identify which calendar contributed each item.

#### Parallel Processing
Multiple sources are processed concurrently for improved performance, with global aggregation and deduplication applied after all sources are processed.

#### Error Handling
Individual source failures don't prevent processing of other sources. The application continues processing remaining sources and reports any issues encountered.

#### Legacy Compatibility
Existing single-source configurations continue to work unchanged. The application automatically migrates legacy configurations to the new multi-source format internally.

## YAML Configuration Examples

YAML configuration provides a more structured and readable alternative to environment files. All functionality is equivalent between formats.

### Basic YAML Configuration

#### Single CalDAV Source with OAuth
```yaml
---
output: ./calendar-output
start_date: 2024-01-01T00:00:00Z
end_date: 2025-12-31T23:59:59Z
use_frontmatter: true
use_hashtags: true
event_checkboxes: true

sources:
  - type: caldav
    name: Google Calendar
    caldav:
      url: https://apidata.googleusercontent.com/caldav/v2/primary/events
      use_oauth: true
      client_id: your_client_id.apps.googleusercontent.com
      client_secret: your_client_secret
      discover_calendars: true
      calendar_aliases:
        "Work Calendar": "Work"
        "Personal Calendar": "Personal"
```

#### Single ICS Source
```yaml
---
output: ./ics-output
start_date: 2024-01-01T00:00:00Z
end_date: 2024-12-31T23:59:59Z
use_due_date_emoji: true
use_hashtags: true
ignore_descriptions: true

sources:
  - type: ics
    name: Local Calendar
    ics:
      type: local
      path: /path/to/calendar.ics
```

### Advanced YAML Configuration

#### Multi-Source Enterprise Setup
```yaml
---
# Global settings
output: ./enterprise-calendars
start_date: 2024-01-01T00:00:00Z
end_date: 2025-12-31T23:59:59Z
use_frontmatter: true
use_hashtags: true
event_checkboxes: true
obsidian_tasks: false
trace_web_calls: false

sources:
  # Corporate Google Calendar with OAuth
  - type: caldav
    name: Corporate Google
    caldav:
      url: https://apidata.googleusercontent.com/caldav/v2/primary/events
      use_oauth: true
      client_id: corporate_client_id
      client_secret: corporate_client_secret
      discover_calendars: true
      use_server_side_filtering: true
      include_calendars:
        - Work
        - Meetings
        - Projects
      exclude_calendars:
        - Archive
        - Spam
      calendar_aliases:
        "Work Calendar": "Work"
        "Meeting Rooms": "Meetings"
        "Project Deadlines": "Projects"

  # Nextcloud CalDAV with proxy
  - type: caldav
    name: Personal Nextcloud
    caldav:
      url: https://nextcloud.example.com/remote.php/dav/calendars/user/personal/
      username: nextcloud_user
      password: nextcloud_password
      use_server_side_filtering: false
      proxy_url: http://proxy.company.com:8080
      proxy_username: proxy_user
      proxy_password: proxy_pass

  # Local task file
  - type: ics
    name: Local Tasks
    ics:
      type: local
      path: /home/user/tasks/personal.ics

  # Remote calendar with authentication
  - type: ics
    name: Team Calendar
    ics:
      type: remote
      path: https://team.company.com/calendar/export.ics
      auth: basic
      username: team_user
      password: team_password
      timeout: 60s

  # API-based calendar with custom headers
  - type: ics
    name: CRM Calendar
    ics:
      type: remote
      path: https://crm.company.com/api/calendar/export
      auth: header
      timeout: 30s
      headers:
        Authorization: Bearer api_token_here
        X-Client-ID: caldav2markdown
        User-Agent: CalDAV2Markdown/1.0
```

#### Task Management Focus
```yaml
---
output: ./task-management
start_date: 2024-01-01T00:00:00Z
end_date: 2025-12-31T23:59:59Z
obsidian_tasks: true  # Enables event_checkboxes, ignore_descriptions, frontmatter, emojis, hashtags

sources:
  # Google Tasks
  - type: caldav
    name: Google Tasks
    caldav:
      url: https://apidata.googleusercontent.com/caldav/v2/primary/events
      use_oauth: true
      client_id: google_client_id
      client_secret: google_client_secret

  # Local todo files
  - type: ics
    name: Personal Todos
    ics:
      type: local
      path: /home/user/todos/personal.ics

  - type: ics
    name: Work Todos
    ics:
      type: local
      path: /home/user/todos/work.ics

  # Project management system
  - type: ics
    name: Project Tasks
    ics:
      type: remote
      path: https://project.company.com/api/tasks/export.ics
      auth: bearer
      token: project_management_token
      timeout: 45s
```

#### Development & Testing Setup
```yaml
---
output: ./dev-calendars
start_date: 2023-01-01T00:00:00Z
end_date: 2026-12-31T23:59:59Z
use_frontmatter: true
use_hashtags: true
trace_web_calls: true  # Enable detailed logging

sources:
  # Production calendar (read-only)
  - type: caldav
    name: Production Events
    caldav:
      url: https://prod-calendar.company.com/caldav/
      username: readonly_user
      password: readonly_password
      discover_calendars: false
      use_server_side_filtering: true

  # Test data from local files
  - type: ics
    name: Test Data
    ics:
      type: local
      path: /dev/test-data/sample-calendar.ics

  # Development API
  - type: ics
    name: Dev API
    ics:
      type: remote
      path: https://dev-api.company.com/calendar/export
      auth: header
      timeout: 10s
      headers:
        Authorization: Bearer dev_token
        X-Environment: development
        X-Debug: "true"
```

### YAML Configuration Field Reference

#### Global Configuration
```yaml
# Output and date settings
output: ./events                    # Output directory
start_date: 2024-01-01T00:00:00Z   # Start date (ISO 8601 format)
end_date: 2024-12-31T23:59:59Z     # End date (ISO 8601 format)

# Display options
use_due_date_emoji: true           # Use 📅 emoji for due dates
use_hashtags: true                 # Add #event and #task hashtags
use_frontmatter: true              # Add YAML frontmatter to markdown files
ignore_descriptions: false         # Ignore event/task descriptions
event_checkboxes: true             # Add checkboxes to events
obsidian_tasks: false              # Enable Obsidian tasks preset
trace_web_calls: false             # Enable detailed HTTP logging
```

#### CalDAV Source Configuration
```yaml
sources:
  - type: caldav
    name: Source Display Name        # Optional display name
    caldav:
      # Connection settings
      url: https://example.com/caldav # CalDAV server URL (required)
      username: user                  # Username for basic auth
      password: pass                  # Password for basic auth

      # OAuth settings (alternative to username/password)
      use_oauth: true                 # Enable OAuth 2.0
      client_id: oauth_client_id      # OAuth client ID
      client_secret: oauth_secret     # OAuth client secret

      # Server features
      use_server_side_filtering: true # Use server-side date filtering
      discover_calendars: true        # Auto-discover calendars

      # Calendar filtering
      include_calendars:              # Include only these calendars
        - Work
        - Personal
      exclude_calendars:              # Exclude these calendars
        - Archive
        - Spam

      # Display customization
      calendar_aliases:               # Calendar name aliases
        "Long Calendar Name": "Short"
        "Work Calendar": "Work"

      # Proxy settings
      proxy_url: http://proxy:8080    # HTTP proxy URL
      proxy_username: proxy_user      # Proxy username
      proxy_password: proxy_pass      # Proxy password
```

#### ICS Source Configuration
```yaml
sources:
  - type: ics
    name: Source Display Name        # Optional display name
    ics:
      # Source type and location
      type: remote                   # local or remote
      path: https://example.com/cal.ics # File path or URL (required)

      # Authentication for remote sources
      auth: basic                    # none, basic, bearer, header
      username: user                 # For basic auth
      password: pass                 # For basic auth
      token: bearer_token            # For bearer auth

      # HTTP settings for remote sources
      timeout: 30s                   # HTTP timeout (e.g., 30s, 1m)
      headers:                       # Custom HTTP headers
        Authorization: Bearer token
        X-Client-ID: caldav2markdown
        User-Agent: Custom-Agent/1.0
```

### YAML Best Practices

#### Date Format
Always use ISO 8601 format for dates in YAML:
```yaml
start_date: 2024-01-01T00:00:00Z
end_date: 2024-12-31T23:59:59Z
```

#### Multi-line Strings
For complex headers or long values, use YAML multi-line syntax:
```yaml
headers:
  Authorization: >
    Bearer very_long_token_that_might_span_
    multiple_lines_for_readability
```

#### Comments
Use comments to document your configuration:
```yaml
sources:
  # Production Google Calendar
  - type: caldav
    name: Production
    # ... configuration

  # Development test data
  - type: ics
    name: Test Data
    # ... configuration
```

#### Environment Variables
You can still use environment variable interpolation in some YAML parsers:
```yaml
sources:
  - type: caldav
    name: Google Calendar
    caldav:
      client_id: ${GOOGLE_CLIENT_ID}
      client_secret: ${GOOGLE_CLIENT_SECRET}
```

### Converting Between Formats

#### Command Line Conversion
```bash
# Convert existing .env file to YAML
bin/caldav2markdown -convert-to-yaml config.yaml -config .env

# Export current configuration (from any format) to YAML
bin/caldav2markdown -export-yaml exported-config.yaml -config current.env

# The application will auto-detect format:
bin/caldav2markdown -config config.yaml    # Uses YAML
bin/caldav2markdown -config config.env     # Uses env format
bin/caldav2markdown -config config         # Auto-detects based on content
```

#### Migration Strategy
1. **Start with env**: Keep your existing `.env` configuration
2. **Convert**: Use `-convert-to-yaml` to create equivalent YAML
3. **Validate**: Test the YAML configuration with your setup
4. **Switch**: Update your workflows to use the YAML file
5. **Maintain**: Use YAML for new configurations going forward