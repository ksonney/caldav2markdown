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
- **`pkg/org/formatter.go`**: Converts iCalendar components to Emacs Org mode format, handles both VEVENT and VTODO items with Org properties drawer support
- **`pkg/diary/formatter.go`**: Converts iCalendar components to Emacs diary format, traditional calendar format for GNU Emacs with simple date-based entries
- **`pkg/config/config.go`**: Configuration management supporting both environment files and CLI flags, with multi-calendar and OAuth options
- **`pkg/rrule/rrule.go`**: Recurrence rule (RRULE) parsing and recurring event expansion engine
- **`pkg/oauth/client.go`**: Google OAuth 2.0 client implementation for CalDAV authentication
- **`pkg/database/database.go`**: SQLite database implementation for event/todo storage and deduplication tracking
- **`pkg/database/events.go`**: Event database operations including upsert, retrieval, and querying
- **`pkg/database/todos.go`**: Todo database operations including upsert, retrieval, and querying
- **`pkg/database/convert.go`**: Conversion helpers between iCalendar objects and database structs

### Data Flow

1. **Configuration**: Load from `.env` file or CLI flags (CalDAV URL, credentials, OAuth settings, multi-calendar options, output directory, date range, database settings)
2. **Database Initialization** (optional): Open SQLite database for event/todo tracking and deduplication
3. **Authentication**: OAuth 2.0 flow for Google Calendar or basic authentication for other CalDAV servers
4. **Calendar Discovery** (optional): Use PROPFIND requests to discover all calendar collections on the server
5. **CalDAV Fetching**:
   - **Server-side filtering**: Use CalDAV REPORT queries with time-range filters (preferred for performance)
   - **Client-side filtering**: Fallback to reading all `.ics` files and filtering locally
   - **Multi-calendar**: Process multiple calendars with individual filtering and global deduplication
6. **Processing**:
   - Extract VEVENT items (filtered by configurable date range)
   - Expand recurring events using RRULE parsing
   - Extract VTODO items with category and status information
   - **Database Storage** (optional): Store events/todos in database with UID-based deduplication tracking
   - Global UID-based deduplication across all calendars and expanded instances
7. **Output Generation**:
   - **Markdown Format** (default): Daily aggregated markdown files `YYYY-MM-DD.md` containing all events and tasks with due dates
   - **Org Mode Format** (optional): Daily aggregated org files `YYYY-MM-DD.org` with proper Org mode syntax
   - Separate `todo.md` or `todo.org` file for tasks without due dates with intelligent merging
   - Optional YAML frontmatter (Markdown) or Org properties drawer (Org mode) with metadata
   - Smart file merging: updates existing files instead of overwriting, preserving manual edits and custom content
   - Intelligent content merging preserves user customizations while updating calendar data

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
- **Output Format Selection**: Choose between Markdown (`.md`), Org mode (`.org`), or Emacs diary format (controlled by `OUTPUT_FORMAT` config)
- **File Organization Modes**:
  - **Daily Files** (default): Events organized by date in `YYYY/MM/YYYY-MM-DD.md` format
  - **Weekly Files**: Events organized by ISO week in `YYYY/YYYY-WW.md` format (controlled by `WEEKLY_FILE_OUTPUT` config)
  - **Single File**: All events in one consolidated file (controlled by `SINGLE_FILE_OUTPUT` config)
- **Directory Structure**: Daily files organized in YYYY/MM directory tree, weekly files in YYYY directory, with zero-date events in `0001/` directory
- **Daily Aggregation**: Events and tasks are grouped by date and saved as daily files with list format, including separate sections for all-day events, scheduled events, and tasks
- **Weekly Aggregation**: Events and tasks are grouped by ISO week (Monday-Sunday) and saved as weekly files, with daily sections within each week file
- **Metadata Support**:
  - **Markdown**: Optional YAML frontmatter with metadata
  - **Org Mode**: Org properties drawer with metadata, #+TITLE directive
- **Flexible Formatting Options**:
  - **Output Format**: Markdown (default), Org mode, or Emacs diary (controlled by `OUTPUT_FORMAT` config)
  - Optional 📅 emoji for due dates (controlled by `USE_DUE_DATE_EMOJI` config, Markdown only)
  - Optional #event and #task hashtags or :event: :task: tags (controlled by `USE_HASHTAGS` config)
  - **Calendar Alias Hashtags/Tags**: Automatic hashtags/tags from calendar names/aliases (controlled by `USE_CALENDAR_TAGS` config)
  - Structured metadata with comprehensive fields (controlled by `USE_FRONTMATTER` config)
  - Optional description exclusion for cleaner output (controlled by `IGNORE_DESCRIPTIONS` config)
  - **Ignore Declined Events**: Automatically skip declined events (STATUS=CANCELLED or PARTSTAT=DECLINED) (controlled by `IGNORE_DECLINED` config)
  - Optional event checkboxes/TODO states for task-like formatting (controlled by `EVENT_CHECKBOXES` config)
  - **Calendar Name Display**: Automatic calendar name extraction with customizable aliases (controlled by `CALENDAR_ALIASES` config)
  - **Obsidian Tasks Emoji Format**: Use 🛫 for start times and ✅ for end times in Obsidian Tasks format (controlled by `USE_OBSIDIAN_EMOJIS` config, Markdown only)
  - **Obsidian Tasks Preset**: One-click configuration for Obsidian compatibility - enables event checkboxes, ignores descriptions, frontmatter, emojis, hashtags, calendar tags, and Obsidian time emojis (controlled by `OBSIDIAN_TASKS` config, Markdown only)
  - **Past Event Completion**: Automatic [x] marking (Markdown) or DONE state (Org mode) for past events when EVENT_CHECKBOXES is enabled
  - **Single File Output**: Optional single-file mode with `-single-file` flag, creates one consolidated file instead of daily files
- **Smart Todo Organization**:
  - Tasks with due dates are organized by date and included in daily files
  - Tasks without due dates are saved to separate `todo.md` or `todo.org` file with intelligent merging
  - Custom metadata preservation in todo files maintains user customizations

#### Org Mode Support
- **Full Org Mode Output**: Complete Emacs Org mode format support as an alternative to Markdown
- **Org Syntax**:
  - Events and tasks use `** TODO/DONE` headlines with proper Org structure
  - **Scheduled Events**: `SCHEDULED: <2024-11-05 Tue 09:00-10:00>` for timed events
  - **All-day Events**: `SCHEDULED: <2024-11-05 Tue>` for all-day events
  - **Task Deadlines**: `DEADLINE: <2024-11-05 Tue>` for tasks with due dates
  - **Properties Drawer**: `:PROPERTIES:` drawer with ID, LOCATION, CALENDAR, STATUS, CATEGORIES
  - **Tags**: Org tags syntax `:event:work:personal:` instead of hashtags
  - **Priority Levels**: iCal priorities 1-3 → `[#A]`, 4-6 → `[#B]`, 7-9 → `[#C]`
  - **UID Comments**: `# uid:...` comments for deduplication tracking
  - **Clean Output**: No section headers (like "* All Day Events", "* Scheduled Events", "* Tasks") - only individual event/task items are included
- **Org File Modes**:
  - **Daily Org Files** (default): Separate files for each date `YYYY/MM/YYYY-MM-DD.org`
  - **Single Org File**: All entries in one file without date headers (all items at level 2)
  - **Todo Org File**: Tasks without due dates in `todo.org` with no section headers
- **Smart Merging**: Intelligent merging with existing Org files, preserves custom content and user-added sections
- **Time Format**: Uses Org mode native date/time format `<YYYY-MM-DD Day HH:MM>` for all timestamps
- **Multi-day Events**: Properly formatted with range syntax `<2024-11-05 Tue 09:00>--<2024-11-06 Wed 17:00>`
- **Configuration**: Enable with `OUTPUT_FORMAT=org` environment variable, `-output-format org` CLI flag, or `output_format: org` in YAML config

#### Org-Diary Support
- **Full Org Mode with Diary Sexp Output**: Combines Org mode structure with Emacs diary sexp expressions for dates
- **Org-Diary Format**:
  - Events and tasks use `** TODO/DONE` headlines like standard Org mode
  - **Date Representation**: Uses diary sexp expressions for flexible date handling
  - **Timed Events**: `SCHEDULED: <%diary-date 12 22 2024> 09:00-10:00` for specific dates with times
  - **All-day Events**: `SCHEDULED: <%diary-date 12 22 2024>` for all-day events
  - **Multi-day Events**: `SCHEDULED: <%diary-block 12 22 2024 12 25 2024> 09:00-17:00` for date ranges
  - **Task Deadlines**: `DEADLINE: <%diary-date 12 25 2024>` for tasks with due dates
  - **Undated Items**: `SCHEDULED: <%%(diary-remind '(diary-entry "TBD") 0)>` for events without dates
  - **Tasks without Due Dates**: `DEADLINE: <%%(diary-entry "TODO")>` for undated tasks
  - **Properties Drawer**: `:PROPERTIES:` drawer with ID, LOCATION, CALENDAR, STATUS, CATEGORIES (same as Org mode)
  - **Tags**: Org tags syntax `:event:work:personal:` instead of hashtags
  - **Priority Levels**: iCal priorities 1-3 → `[#A]`, 4-6 → `[#B]`, 7-9 → `[#C]`
  - **UID Comments**: `# uid:...` comments for deduplication tracking
  - **Clean Output**: No section headers (like "* All Day Events", "* Scheduled Events", "* Tasks") - only individual event/task items are included
- **Org-Diary File Modes**:
  - **Single Org-Diary File** (default): All entries in one file without date headers (all items at level 2), automatically enabled for org-diary format
  - **Daily Org-Diary Files**: Separate files for each date `YYYY/MM/YYYY-MM-DD.org` with org-diary format (only when `-single-file=false` is explicitly set)
  - **Todo Org-Diary File**: Tasks without due dates in `todo.org` with diary sexp format and no section headers
- **Benefits of Org-Diary Format**:
  - **Flexible Date Expressions**: Diary sexps allow for complex recurring patterns and conditional dates
  - **Emacs Integration**: Works with both Org agenda and Emacs diary systems
  - **Advanced Scheduling**: Supports diary-anniversary, diary-float, diary-cyclic, and other advanced diary functions
  - **Symbolic Dates**: Represents dates in a form that can be programmatically processed by Emacs Lisp
- **Smart Merging**: Intelligent merging with existing Org files, preserves custom content and user-added sections
- **Compatibility**: Files are valid Org mode documents and can be used with standard Org mode commands
- **Configuration**: Enable with `OUTPUT_FORMAT=org-diary` environment variable, `-output-format org-diary` CLI flag, or `output_format: org-diary` in YAML config

#### Emacs Diary Support
- **Full Emacs Diary Output**: Traditional Emacs calendar diary format support, compatible with GNU Emacs calendar-mode and diary features
- **Diary Format Syntax**:
  - Date format: `MM/DD/YYYY` (standard American format used by Emacs diary)
  - **Timed Events**: `12/22/2024 10:30am-11:30am Meeting with team @ Location [Calendar] #event`
  - **All-day Events**: `12/22/2024 Holiday party @ Venue [Personal] #event`
  - **Multi-day Events**: `12/22/2024 9:00am-12/23/2024 5:00pm Conference`
  - **Tasks with Due Dates**: `12/25/2024 Complete project report [Work] #task`
  - **Tasks without Due Dates**: `%%(diary-entry "TODO") [DONE] Task title [Calendar] #task`
  - **Undated Events**: `%%(diary-remind '(diary-entry "TBD") 0) Event title @ Location`
  - **Completion Markers**: `[DONE]` prefix for completed tasks and past events (when `EVENT_CHECKBOXES` enabled)
  - **UID Tracking**: `{uid:...}` suffix for deduplication (hidden metadata)
- **Diary File Modes**:
  - **Single Diary File** (default): All entries in one consolidated `diary` file, automatically enabled for diary format
  - **Monthly Diary Files**: Separate files for each month `YYYY/MM-diary` (only when `-single-file=false` is explicitly set)
  - **Todo Diary File**: Tasks without due dates in `todo-diary` file
- **Smart Merging**: Intelligent merging with existing diary files, preserves custom entries and comments
- **Chronological Sorting**: Entries automatically sorted by date within each file for easy reading
- **Description Support**: Optional indented descriptions on continuation lines (controlled by `IGNORE_DESCRIPTIONS`)
- **Calendar Tags**: Optional calendar name and category tags using `#tag` syntax (controlled by `USE_HASHTAGS` and `USE_CALENDAR_TAGS`)
- **Comment Headers**: Auto-generated header comments in new files explaining diary format
- **Emacs Integration**: Files can be directly used with Emacs `M-x calendar` and `M-x diary` commands
- **Configuration**: Enable with `OUTPUT_FORMAT=diary` environment variable, `-output-format diary` CLI flag, or `output_format: diary` in YAML config

#### Multi-Calendar Support
- **Calendar Discovery**: Automatic discovery of all calendar collections on a CalDAV server
- **Include/Exclude Filters**: Flexible filtering to process only specific calendars by name or URL pattern
- **Global Deduplication**: UID-based deduplication across multiple calendars prevents duplicate events
- **Individual Calendar Processing**: Each calendar processed independently with fallback error handling

### Recent Major Updates

#### Latest Technical Enhancements (2025)
- **Obsidian Life Manager Directory Structure**: New optional directory structure compatible with Obsidian Life Manager vault organization. Daily files saved to `Daily/YYYY/MM - Month Name/YYYY-MM-DD.md` and weekly files to `Weekly/YYYY/YYYY-Www.md`. Perfect for users following the Obsidian Life Manager system. Controlled by `OBSIDIAN_LIFE_MANAGER` config option or `-obsidian-life-manager` CLI flag.
- **Weekly File Output Mode**: New file organization option that groups events and tasks by ISO week (Monday-Sunday) instead of daily files. Files are organized as `YYYY/YYYY-WW.md` with daily sections within each weekly file. Includes week-specific frontmatter with week number, year, date range, and event/task counts. Controlled by `WEEKLY_FILE_OUTPUT` config option or `-weekly-file` CLI flag.
- **Org-Diary Output Format**: New hybrid format combining Org mode structure with Emacs diary sexp expressions for flexible date handling. Supports both daily and single-file output modes with full Org mode and diary system integration.
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
- `github.com/mattn/go-sqlite3`: SQLite database driver for event/todo storage and tracking
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

#### Weekly File Output Mode
```bash
CALDAV_URL=https://your-server.com/caldav/calendars/username/calendar/
CALDAV_USERNAME=username
CALDAV_PASSWORD=password
WEEKLY_FILE_OUTPUT=true
USE_FRONTMATTER=true
USE_HASHTAGS=true
EVENT_CHECKBOXES=true
OUTPUT_DIR=./weekly-calendar
```

#### Obsidian Life Manager Directory Structure
```bash
CALDAV_URL=https://your-server.com/caldav/calendars/username/calendar/
CALDAV_USERNAME=username
CALDAV_PASSWORD=password
OBSIDIAN_LIFE_MANAGER=true
USE_FRONTMATTER=true
USE_HASHTAGS=true
EVENT_CHECKBOXES=true
OUTPUT_DIR=./ObsidianVault
# Creates: Daily/2025/11 - November/2025-11-24.md
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

#### Org Mode Output Format
```bash
# Output in Emacs Org mode format instead of Markdown
OUTPUT_FORMAT=org
CALDAV_URL=https://your-server.com/caldav/calendars/username/calendar/
CALDAV_USERNAME=username
CALDAV_PASSWORD=password
OUTPUT_DIR=./org-calendar
USE_HASHTAGS=true
USE_CALENDAR_TAGS=true
EVENT_CHECKBOXES=true
IGNORE_DESCRIPTIONS=true
```

#### Org Mode with Single File Output
```bash
# Create a single consolidated Org file instead of daily files
OUTPUT_FORMAT=org
SINGLE_FILE=true
SINGLE_FILE_NAME=calendar.org
CALDAV_URL=https://your-server.com/caldav/calendars/username/calendar/
CALDAV_USERNAME=username
CALDAV_PASSWORD=password
OUTPUT_DIR=./org-calendar
USE_HASHTAGS=true
EVENT_CHECKBOXES=true
```

#### Org Mode for Emacs Org-Agenda
```bash
# Optimized configuration for Emacs Org-agenda integration
OUTPUT_FORMAT=org
CALDAV_URL=https://your-server.com/caldav/calendars/username/calendar/
CALDAV_USERNAME=username
CALDAV_PASSWORD=password
OUTPUT_DIR=~/org/calendars
USE_HASHTAGS=true
USE_CALENDAR_TAGS=true
EVENT_CHECKBOXES=true
IGNORE_DESCRIPTIONS=false
DISCOVER_CALENDARS=true
CALENDAR_ALIASES=Personal Calendar:Personal,Work Calendar:Work
```

#### Org-Diary Output Format
```bash
# Output in Org mode format with diary sexp expressions
# Single file mode is automatically enabled for org-diary format
OUTPUT_FORMAT=org-diary
CALDAV_URL=https://your-server.com/caldav/calendars/username/calendar/
CALDAV_USERNAME=username
CALDAV_PASSWORD=password
OUTPUT_DIR=./org-diary-calendar
USE_HASHTAGS=true
USE_CALENDAR_TAGS=true
EVENT_CHECKBOXES=true
IGNORE_DESCRIPTIONS=true
# Output will be: ./org-diary-calendar/calendar.org
```

#### Org-Diary with Custom Filename
```bash
# Single file mode is automatic, but you can customize the filename
OUTPUT_FORMAT=org-diary
SINGLE_FILE_NAME=my-calendar.org
CALDAV_URL=https://your-server.com/caldav/calendars/username/calendar/
CALDAV_USERNAME=username
CALDAV_PASSWORD=password
OUTPUT_DIR=./org-diary
USE_HASHTAGS=true
EVENT_CHECKBOXES=true
```

#### Org-Diary for Emacs Dual Integration
```bash
# Optimized for both Org-agenda and Emacs diary integration
# Single file mode is automatically enabled for org-diary format
OUTPUT_FORMAT=org-diary
CALDAV_URL=https://your-server.com/caldav/calendars/username/calendar/
CALDAV_USERNAME=username
CALDAV_PASSWORD=password
OUTPUT_DIR=~/org/diary-calendars
USE_HASHTAGS=true
USE_CALENDAR_TAGS=true
EVENT_CHECKBOXES=true
IGNORE_DESCRIPTIONS=false
DISCOVER_CALENDARS=true
CALENDAR_ALIASES=Personal Calendar:Personal,Work Calendar:Work
# Output will be: ~/org/diary-calendars/calendar.org
```

#### Emacs Diary Output Format
```bash
# Output in Emacs diary format for use with GNU Emacs calendar
# Single file mode is automatically enabled for diary format
OUTPUT_FORMAT=diary
CALDAV_URL=https://your-server.com/caldav/calendars/username/calendar/
CALDAV_USERNAME=username
CALDAV_PASSWORD=password
OUTPUT_DIR=~/emacs
USE_HASHTAGS=true
USE_CALENDAR_TAGS=true
EVENT_CHECKBOXES=true
IGNORE_DESCRIPTIONS=false
# Output will be: ~/emacs/diary
```

#### Emacs Diary with Custom Filename
```bash
# Single file mode is automatic, but you can customize the filename
OUTPUT_FORMAT=diary
SINGLE_FILE_NAME=my-calendar
CALDAV_URL=https://your-server.com/caldav/calendars/username/calendar/
CALDAV_USERNAME=username
CALDAV_PASSWORD=password
OUTPUT_DIR=~/.emacs.d
USE_HASHTAGS=true
EVENT_CHECKBOXES=true
# Output will be: ~/.emacs.d/my-calendar
```

#### Emacs Diary for Calendar Integration
```bash
# Optimized configuration for Emacs calendar/diary integration
OUTPUT_FORMAT=diary
CALDAV_URL=https://your-server.com/caldav/calendars/username/calendar/
CALDAV_USERNAME=username
CALDAV_PASSWORD=password
OUTPUT_DIR=~/.emacs.d/diary
USE_HASHTAGS=true
USE_CALENDAR_TAGS=true
EVENT_CHECKBOXES=true
IGNORE_DESCRIPTIONS=true
DISCOVER_CALENDARS=true
CALENDAR_ALIASES=Personal Calendar:Personal,Work Calendar:Work
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

#### Weekly File Output Mode
```yaml
---
output: ./weekly-calendar
start_date: 2024-01-01T00:00:00Z
end_date: 2025-12-31T23:59:59Z
weekly_file_output: true
use_frontmatter: true
use_hashtags: true
event_checkboxes: true

sources:
  - type: caldav
    name: My Calendar
    caldav:
      url: https://your-server.com/caldav/calendars/user/calendar/
      username: user
      password: password
```

#### Obsidian Life Manager Directory Structure
```yaml
---
output: ./ObsidianVault
start_date: 2024-01-01T00:00:00Z
end_date: 2025-12-31T23:59:59Z
obsidian_life_manager: true
use_frontmatter: true
use_hashtags: true
event_checkboxes: true

sources:
  - type: caldav
    name: My Calendar
    caldav:
      url: https://your-server.com/caldav/calendars/user/calendar/
      username: user
      password: password
# Creates: Daily/2025/11 - November/2025-11-24.md
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

#### Org Mode Output Format
```yaml
---
# Emacs Org mode format output
output_format: org
output: ./org-calendar
start_date: 2024-01-01T00:00:00Z
end_date: 2024-12-31T23:59:59Z
use_hashtags: true
use_calendar_tags: true
event_checkboxes: true
ignore_descriptions: true

sources:
  - type: caldav
    name: Personal Calendar
    caldav:
      url: https://your-server.com/caldav/calendars/user/calendar/
      username: user
      password: password
```

#### Org Mode with Single File
```yaml
---
# Single consolidated Org file for all calendar data
output_format: org
output: ./org-calendar
single_file: true
single_file_name: calendar.org
start_date: 2024-01-01T00:00:00Z
end_date: 2025-12-31T23:59:59Z
use_hashtags: true
event_checkboxes: true

sources:
  - type: caldav
    name: My Calendar
    caldav:
      url: https://your-server.com/caldav/calendars/user/calendar/
      username: user
      password: password
```

#### Org-Diary Output Format
```yaml
---
# Org mode with diary sexp expressions
# Single file mode is automatically enabled for org-diary format
output_format: org-diary
output: ./org-diary-calendar
start_date: 2024-01-01T00:00:00Z
end_date: 2024-12-31T23:59:59Z
use_hashtags: true
use_calendar_tags: true
event_checkboxes: true
ignore_descriptions: true
# Output will be: ./org-diary-calendar/calendar.org

sources:
  - type: caldav
    name: Personal Calendar
    caldav:
      url: https://your-server.com/caldav/calendars/user/calendar/
      username: user
      password: password
```

#### Org-Diary with Custom Filename
```yaml
---
# Single file mode is automatic, but you can customize the filename
output_format: org-diary
output: ./org-diary
single_file_name: my-calendar.org
start_date: 2024-01-01T00:00:00Z
end_date: 2025-12-31T23:59:59Z
use_hashtags: true
event_checkboxes: true

sources:
  - type: caldav
    name: My Calendar
    caldav:
      url: https://your-server.com/caldav/calendars/user/calendar/
      username: user
      password: password
```

#### Emacs Diary Output Format
```yaml
---
# Emacs diary format output
# Single file mode is automatically enabled for diary format
output_format: diary
output: ~/emacs
start_date: 2024-01-01T00:00:00Z
end_date: 2024-12-31T23:59:59Z
use_hashtags: true
use_calendar_tags: true
event_checkboxes: true
ignore_descriptions: false
# Output will be: ~/emacs/diary

sources:
  - type: caldav
    name: Personal Calendar
    caldav:
      url: https://your-server.com/caldav/calendars/user/calendar/
      username: user
      password: password
```

#### Emacs Diary with Custom Filename
```yaml
---
# Single file mode is automatic, but you can customize the filename
output_format: diary
output: ~/.emacs.d
single_file_name: my-calendar
start_date: 2024-01-01T00:00:00Z
end_date: 2025-12-31T23:59:59Z
use_hashtags: true
event_checkboxes: true
# Output will be: ~/.emacs.d/my-calendar

sources:
  - type: caldav
    name: My Calendar
    caldav:
      url: https://your-server.com/caldav/calendars/user/calendar/
      username: user
      password: password
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
output_format: markdown             # Output format: markdown or org
single_file: false                  # Generate single file instead of daily files
single_file_name: calendar.md       # Name for single file output (when single_file: true)
weekly_file_output: false           # Generate weekly files instead of daily files (ISO week Monday-Sunday)
obsidian_life_manager: false        # Use Obsidian Life Manager directory structure
output_timezone: ""                 # IANA time zone for rendering event/task times (e.g. America/New_York; default: system local time zone)
start_date: 2024-01-01T00:00:00Z   # Start date (ISO 8601 format)
end_date: 2024-12-31T23:59:59Z     # End date (ISO 8601 format)

# Display options
use_due_date_emoji: true           # Use 📅 emoji for due dates (Markdown only)
use_hashtags: true                 # Add #event and #task hashtags (Markdown) or :tags: (Org)
use_calendar_tags: true            # Add calendar name tags
use_frontmatter: true              # Add YAML frontmatter (Markdown) or properties (Org)
ignore_descriptions: false         # Ignore event/task descriptions
event_checkboxes: true             # Add checkboxes (Markdown) or TODO states (Org)
ignore_declined: false             # Skip declined events
obsidian_tasks: false              # Enable Obsidian tasks preset (Markdown only)
use_obsidian_emojis: false         # Use Obsidian task emojis (Markdown only)
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

## Database Storage and Deduplication

The application includes optional SQLite database support for persistent event/todo storage and advanced deduplication tracking.

### Features

#### Event and Todo Storage
- **Persistent Storage**: All events and todos are stored in a SQLite database with full metadata
- **UID-based Deduplication**: Events and todos are deduplicated by UID across all sources and calendar runs
- **Recurrence Tracking**: Each recurring event instance is stored separately with its recurrence ID
- **Source Attribution**: Track which calendar source each event/todo came from
- **First/Last Seen Timestamps**: Monitor when events were first discovered and last updated

#### Query and Analysis
- **Date Range Queries**: Retrieve events within specific date ranges
- **Calendar Filtering**: Filter events/todos by source calendar
- **Status Tracking**: Track completion status of todos
- **Priority Filtering**: Query todos by priority level
- **Full-Text Search**: Search events and todos by summary or description
- **Statistics**: View detailed statistics about stored events and todos

### Configuration

#### Environment File Configuration
```bash
# Enable database storage
USE_DATABASE=true

# Optional: Specify custom database path
# Default: placed alongside config file as caldav2markdown.db
DATABASE_PATH=/path/to/custom/database.db

# Rest of your configuration...
CALDAV_URL=https://your-calendar-server.com/caldav/
CALDAV_USERNAME=username
CALDAV_PASSWORD=password
OUTPUT_DIR=./calendar-notes
```

#### YAML Configuration
```yaml
---
# Enable database storage
use_database: true

# Optional: Custom database path
database_path: /path/to/custom/database.db

# Rest of your configuration...
output: ./calendar-notes
sources:
  - type: caldav
    name: My Calendar
    caldav:
      url: https://your-calendar-server.com/caldav/
      username: username
      password: password
```

#### CLI Flags
```bash
# Enable database with default path
bin/caldav2markdown -use-database -config .env

# Specify custom database path
bin/caldav2markdown -use-database -database-path /path/to/db.db -config .env

# View database statistics
bin/caldav2markdown -use-database -db-stats -config .env

# Clear database (with confirmation)
bin/caldav2markdown -use-database -db-clear -config .env
```

### Database Location

By default, the database file is created alongside your configuration file:
- Config at `~/.config/caldav2markdown/config.yaml` → Database at `~/.config/caldav2markdown/caldav2markdown.db`
- Config at `.env` → Database at `./caldav2markdown.db`
- Custom config at `/etc/caldav/config.env` → Database at `/etc/caldav/caldav2markdown.db`

### Database Schema

#### Events Table
- **Primary Key**: Auto-incrementing ID
- **Unique Constraint**: (UID, RecurrenceID, SourceName)
- **Indexed Fields**: UID, StartTime, SourceName
- **Stored Fields**: Summary, Description, Location, StartTime, EndTime, AllDay, Calendar, Status, Categories, Source info, Timestamps

#### Todos Table
- **Primary Key**: Auto-incrementing ID
- **Unique Constraint**: (UID, SourceName)
- **Indexed Fields**: UID, DueDate, SourceName
- **Stored Fields**: Summary, Description, DueDate, StartDate, Completed, Priority, Status, Categories, Calendar, Source info, Timestamps

#### Metadata Table
- **Purpose**: Store application metadata like last sync times
- **Schema**: Key-value pairs with timestamps

### Usage Examples

#### Basic Usage with Database
```bash
# First run - creates database and stores all events
bin/caldav2markdown -use-database -config config.yaml
# Output: Database: 150 new events, 0 updated events, 25 new todos, 0 updated todos

# Subsequent runs - events already in database are updated, duplicates prevented
bin/caldav2markdown -use-database -config config.yaml
# Output: Database: 3 new events, 147 updated events, 1 new todos, 24 updated todos
```

**How Deduplication and Change Tracking Works:**
1. **Fetch calendars**: Events/todos are fetched from all configured sources
2. **Store in database**: Each event/todo is stored with UID-based deduplication
   - **New items**: Inserted with `first_seen` timestamp
   - **Existing items**: Compared field-by-field to detect changes
   - **Changed items**: Updated with new values and `last_seen` timestamp
   - **Unchanged items**: Only `last_seen` timestamp updated
3. **Change detection**: Automatically detects modifications to:
   - **Events**: summary, description, location, start time, end time, all-day status, calendar, status, categories
   - **Todos**: summary, description, due date, start date, completed status, priority, status, categories, calendar
4. **Generate markdown**: All items (new and updated) are converted to markdown
5. **Smart merging**: Markdown files are intelligently merged, preventing duplicates

**Key Benefits:**
- Tracks when events first appeared (`first_seen`)
- Tracks when events were last seen (`last_seen`)
- **Detects and reports changes** automatically
- Prevents duplicate markdown entries
- Maintains full history in database
- Shows exactly what changed in each update

#### Change Detection Example
```bash
# First run - create initial entries
$ bin/caldav2markdown -use-database -config config.yaml
Storing events and todos in database...
Database: 150 new events, 0 updated events, 25 new todos, 0 updated todos

# User modifies event time and description in calendar

# Second run - detects changes
$ bin/caldav2markdown -use-database -config config.yaml
Storing events and todos in database...
Changes detected: 3 events modified, 1 todos modified
Database: 0 new events, 150 updated events, 0 new todos, 25 updated todos
```

**What Gets Detected:**
- Event/todo moved to different time
- Summary/title changes
- Description updates
- Location changes
- Status changes (confirmed, cancelled, tentative)
- Priority changes (for todos)
- Completion status changes (for todos)
- Category/tag modifications

**When to Use:**
- **Calendar sync**: Automatically track changes from calendar servers
- **Collaboration**: See what team members changed in shared calendars
- **Audit trail**: Maintain history of all event modifications
- **Data integrity**: Ensure your markdown reflects latest calendar state

#### Smart Markdown Updates

When database is enabled, only new or changed items generate markdown:

```bash
# First run - all items are new
$ bin/caldav2markdown -use-database -config config.yaml
Database: 150 new events, 0 updated events, 25 new todos, 0 updated todos
Generating markdown for 150 new/changed events and 25 new/changed todos
Successfully created 45 daily files for 150 events and 25 tasks

# Second run - no changes
$ bin/caldav2markdown -use-database -config config.yaml
Database: 0 new events, 150 updated events, 0 new todos, 25 updated todos
No new or changed items - skipping markdown generation
Successfully processed 0 events and 0 tasks (all unchanged)

# Third run - 3 events changed
$ bin/caldav2markdown -use-database -config config.yaml
Changes detected: 3 events modified, 0 todos modified
Database: 0 new events, 150 updated events, 0 new todos, 25 updated todos
Generating markdown for 3 new/changed events and 0 new/changed todos
Successfully created 3 daily files for 3 events and 0 tasks
```

**Benefits:**
- **No duplicate processing**: Unchanged events never regenerate markdown
- **Faster runs**: Only process items that actually changed
- **Bandwidth savings**: Database tracks everything, markdown only shows changes
- **Clean updates**: Only modified items trigger file updates

#### View Database Statistics
```bash
$ bin/caldav2markdown -use-database -db-stats -config config.yaml

Database Statistics:
  Total events: 1250
  Unique events: 475
  Total todos: 83
  Unique todos: 67

  Events by source:
    Work Calendar: 687
    Personal Calendar: 428
    Family Events: 135

  Todos by source:
    Work Tasks: 45
    Personal Tasks: 38
```

#### Clear Database
```bash
$ bin/caldav2markdown -use-database -db-clear -config config.yaml
Are you sure you want to clear all database data? (yes/no): yes
Database cleared successfully.
```

#### Generate Markdown from Database
```bash
# Fetch calendars and store in database (normal run)
bin/caldav2markdown -use-database -config config.yaml

# Later: regenerate markdown files from database without fetching calendars
# Useful for:
#  - Changing output format options (hashtags, frontmatter, etc.)
#  - Regenerating with different date ranges
#  - Offline markdown generation
#  - Testing different formatting options
bin/caldav2markdown -use-database -from-database -config config.yaml

# Generate markdown from database with different date range
bin/caldav2markdown -use-database -from-database \
  -start 2025-01-01 -end 2025-12-31 \
  -config config.yaml

# Generate with different formatting options
bin/caldav2markdown -use-database -from-database \
  -frontmatter -hashtags -event-checkboxes \
  -config config.yaml
```

**Benefits of Database Generation:**
- **No Network Calls**: Generate markdown instantly without hitting calendar servers
- **Experimenting with Formats**: Try different output formats without refetching data
- **Offline Operation**: Work offline with your calendar data
- **Performance**: Much faster than fetching from remote calendars
- **Consistency**: Same data, different presentations

**Workflow Example:**
```bash
# Daily: fetch and update database
0 */6 * * * /usr/local/bin/caldav2markdown -use-database -config ~/config.yaml

# As needed: regenerate markdown with new formatting
bin/caldav2markdown -use-database -from-database -obsidian-tasks -config ~/config.yaml
```

### How Deduplication Works

#### Event Deduplication
1. **UID Matching**: Events with the same UID from the same source are considered duplicates
2. **Recurrence Handling**: Each recurring event instance has a unique recurrence ID, preventing over-deduplication
3. **Cross-Source**: Events from different sources with the same UID are stored separately (different calendar sources may have different event details)
4. **Update Detection**: When an event is seen again, it's updated with latest details and last_seen timestamp is refreshed

#### Todo Deduplication
1. **UID Matching**: Todos with the same UID from the same source are considered duplicates
2. **Status Updates**: When a todo's completion status changes, the database record is updated
3. **Cross-Source**: Similar to events, todos from different sources are tracked separately

### Database Benefits

#### Performance
- **Faster Subsequent Runs**: Only new or modified events need processing
- **Reduced API Calls**: Can compare local cache against remote calendars
- **Incremental Updates**: No need to re-process entire calendar history each run

#### Data Management
- **Historical Tracking**: See when events first appeared and when they were last updated
- **Source Auditing**: Track which calendar source provided each event
- **Duplicate Analysis**: Identify and resolve duplicate events across calendars
- **Data Export**: Export full event/todo history as JSON for backup or analysis

#### Advanced Features (Future)
- **Sync Detection**: Detect deleted events by comparing database with calendar fetches
- **Change History**: Track how event details change over time
- **Conflict Resolution**: Identify conflicting events across calendars
- **Smart Filtering**: Use database queries for complex filtering scenarios

### Database Maintenance

#### Backup
```bash
# Simple file copy
cp ~/.config/caldav2markdown/caldav2markdown.db ~/backups/calendar-backup.db

# Or use sqlite3 backup command
sqlite3 ~/.config/caldav2markdown/caldav2markdown.db ".backup ~/backups/calendar-backup.db"
```

#### Vacuum/Optimize
```bash
# Optimize database (removes deleted records, rebuilds indexes)
sqlite3 ~/.config/caldav2markdown/caldav2markdown.db "VACUUM"
```

#### Export Data
```bash
# Export events to JSON
sqlite3 -json ~/.config/caldav2markdown/caldav2markdown.db "SELECT * FROM events" > events.json

# Export todos to JSON
sqlite3 -json ~/.config/caldav2markdown/caldav2markdown.db "SELECT * FROM todos" > todos.json
```

### Technical Details

#### Database Engine
- **SQLite 3**: Embedded, serverless, zero-configuration
- **WAL Mode**: Write-Ahead Logging for better concurrent access
- **Foreign Keys**: Enabled for referential integrity

#### Transaction Handling
- Bulk inserts use transactions for performance
- Each upsert is atomic
- Database is properly closed on application exit

#### Concurrent Access
- WAL mode allows multiple readers
- Single writer at a time (safe for cron jobs)
- Lock timeouts handled gracefully

### Troubleshooting

#### Database Locked Error
```bash
# If you see "database is locked" errors:
# 1. Ensure no other caldav2markdown processes are running
ps aux | grep caldav2markdown

# 2. Check for stale lock files
ls -la ~/.config/caldav2markdown/

# 3. If necessary, remove WAL files (when no processes running)
rm ~/.config/caldav2markdown/caldav2markdown.db-wal
rm ~/.config/caldav2markdown/caldav2markdown.db-shm
```

#### Corrupted Database
```bash
# Check database integrity
sqlite3 ~/.config/caldav2markdown/caldav2markdown.db "PRAGMA integrity_check"

# If corrupted, restore from backup or clear and rebuild
bin/caldav2markdown -use-database -db-clear -config config.yaml
```

#### Migration from Non-Database Setup
```bash
# Simply enable database on your next run
# All events will be stored going forward
bin/caldav2markdown -use-database -config config.yaml

# Database will be populated with all current events/todos
# Subsequent runs will use the database for deduplication
```