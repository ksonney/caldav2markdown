# caldav2markdown

A powerful CalDAV to Markdown converter that transforms your calendar events and tasks into organized, searchable markdown files. Perfect for integrating calendar data with static site generators, note-taking apps, or personal knowledge management systems.

## 🆕 Recent Updates

### Latest Enhancements (2025)
- **📓 Emacs Diary Output**: Traditional Emacs diary format support with automatic single-file mode, perfect for GNU Emacs calendar integration
- **📝 Org Mode Output**: Full Emacs Org mode format support with native scheduling, TODO states, properties drawers, and tags
- **💾 SQLite Database Storage**: Optional database for event/todo tracking, deduplication, and change detection with smart updates
- **🚫 Ignore Declined Events**: Automatically skip events you've declined (STATUS=CANCELLED or PARTSTAT=DECLINED) to keep your calendar clean
- **🎯 Obsidian Tasks Emoji Format**: Added support for Obsidian Tasks emoji format with 🛫 for start times and ✅ for end times
- **🔧 Fixed Autodiscovery**: Resolved configuration validation issue that prevented autodiscovery from working without explicit SOURCE_MODE setting
- **🏷️ Calendar Name Display & Aliases**: Events and tasks now display their source calendar with support for custom aliases
- **🕐 Time Zone Support**: Full TZID parameter support for accurate time zone handling in events and tasks
- **📝 Smart Todo Management**: Tasks without due dates are now saved to a separate `todo.md`/`todo.org` file with intelligent merging
- **🔀 Intelligent Frontmatter Merging**: Enhanced file merging preserves custom frontmatter fields while updating calendar statistics
- **✅ Past Event Marking**: When EVENT_CHECKBOXES is enabled, past events are automatically marked as completed [x] or DONE

### Enhanced Recurring Event Processing (2025)
- **Fixed Date Range Filtering**: Recurring events and tasks are now properly processed when their original start date falls outside the configured date filter range
- **Smart Instance Detection**: The application expands recurring items first, then filters individual instances by date, ensuring no valid occurrences are missed
- **Recurring Task Support**: Added full RRULE expansion support for VTODO components, matching the capabilities of VEVENT processing
- **Improved Accuracy**: Eliminates the previous issue where recurring events with start dates before the filter range were completely ignored

### ICS File Support (2025)
- **Local and Remote ICS**: Added comprehensive support for both local `.ics` files and remote HTTP/HTTPS ICS URLs
- **Multiple Authentication**: Support for no auth, basic auth, bearer tokens, and custom headers for remote ICS sources
- **Task-Only Calendar Support**: Fixed processing of calendars containing only tasks (VTODO components) with no events
- **Enhanced File Merging**: Improved frontmatter preservation and merging when updating existing markdown files

## ✨ Key Features

### 🔐 Authentication & Security
- **Google OAuth 2.0**: Full support for Google Calendar with automatic token management
- **Traditional CalDAV**: Username/password authentication for standard CalDAV servers
- **Proxy Support**: Full HTTP proxy support with optional authentication for corporate environments
- **Secure Token Storage**: OAuth tokens stored securely with automatic refresh

### 🚀 Performance & Scalability
- **SQLite Database Storage**: Optional database for event/todo tracking with change detection
- **Smart Markdown Updates**: Only process new or changed items, skip unchanged entries
- **Server-side Filtering**: CalDAV REPORT queries for efficient data retrieval
- **Multi-calendar Support**: Discover and process multiple calendars automatically
- **Smart Deduplication**: Global UID-based deduplication across all calendars
- **Parallel Processing**: Concurrent calendar processing with error handling

### 📁 Output & Organization
- **Multiple Output Formats**: Choose between Markdown (`.md`), Emacs Org mode (`.org`), or Emacs diary format
- **Daily Aggregation**: Events and tasks organized in `YYYY-MM-DD.md` or `YYYY-MM-DD.org` files (Markdown/Org mode)
- **Single File Mode**: Optional consolidated output in one file instead of daily files
- **Smart Todo Management**: Tasks without due dates saved to separate `todo.md` or `todo.org` file
- **Structured Metadata**: YAML frontmatter (Markdown) or Org properties drawer (Org mode) for static site generators
- **Intelligent File Merging**: Preserves custom content and manual edits while adding new calendar data
- **Flexible Formatting**: Customizable emoji, hashtags/tags, display options, optional description exclusion, and event checkboxes/TODO states
- **Past Event Marking**: Automatically marks past events as completed [x] (Markdown) or DONE (Org mode) when checkboxes are enabled
- **Directory Structure**: Organized in `YYYY/MM/` hierarchy for easy navigation

### 📅 Calendar Features
- **Event Processing**: Full support for recurring events with RRULE expansion
- **Smart Recurring Processing**: Recurring events/tasks are expanded first, then filtered by date range, ensuring items with start dates outside the filter range are included if they have instances within the range
- **Task Management**: Todo extraction with due dates, priorities, and status, including recurring task support
- **Calendar Name Display**: Events and tasks show their source calendar with customizable aliases
- **Time Zone Support**: Full TZID parameter support with IANA time zone database and common mappings
- **Date/Time Handling**: Support for UTC, local time, time zones, and all-day events
- **Category Integration**: Calendar categories preserved in frontmatter
- **Date Filtering**: Configurable date range filtering for focused output

## Installation

### Using Make (Recommended)

```bash
make build
```

The binary will be built in the `bin/` directory.

### Manual Build

```bash
go build -o bin/caldav2markdown ./cmd/caldav2markdown
```

## Configuration

### Method 1: Environment File (Recommended)

Copy the example configuration file and customize it:

```bash
cp .env.example .env
```

#### Basic CalDAV Configuration

```env
# CalDAV Server URL (required)
CALDAV_URL=https://your-caldav-server.com/calendars/username/calendar-name/

# Authentication Method 1: Basic Auth (traditional)
CALDAV_USERNAME=your-username
CALDAV_PASSWORD=your-password

# Output Settings
OUTPUT_DIR=./events
START_DATE=2024-01-01
END_DATE=2024-12-31
```

#### Google Calendar OAuth Configuration

For Google Calendar, use OAuth 2.0 authentication (recommended for 2025+):

```env
# CalDAV Server URL for Google Calendar
CALDAV_URL=https://apidata.googleusercontent.com/caldav/v2/primary/events

# OAuth 2.0 Authentication
USE_OAUTH=true
GOOGLE_CLIENT_ID=your_client_id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your_client_secret

# Output Settings
OUTPUT_DIR=./events
USE_FRONTMATTER=true
USE_HASHTAGS=true
USE_CALENDAR_TAGS=true
```

#### Multi-Calendar Configuration

```env
# Base URL for calendar discovery
CALDAV_URL=https://your-server.com/

# Authentication
CALDAV_USERNAME=your-username
CALDAV_PASSWORD=your-password

# Multi-Calendar Options
DISCOVER_CALENDARS=true
INCLUDE_CALENDARS=Work,Personal
EXCLUDE_CALENDARS=Archive,Test
CALENDAR_ALIASES=Personal Calendar:Personal,Work Calendar:Work

# Performance Options
USE_SERVER_SIDE_FILTERING=true
```

#### ICS File Mode Configuration

For processing local or remote ICS files instead of CalDAV:

```env
# Source mode
SOURCE_MODE=ics

# Local ICS file
ICS_PATH=./my-calendar.ics

# OR Remote ICS URL
ICS_URL=https://calendar.example.com/my-calendar.ics
ICS_AUTH=basic
ICS_USERNAME=your-username
ICS_PASSWORD=your-password

# Calendar aliases for ICS files
CALENDAR_ALIASES=My Calendar:Personal,Team Calendar:Work

# Output settings
OUTPUT_DIR=./events
USE_FRONTMATTER=true
USE_HASHTAGS=true
USE_CALENDAR_TAGS=true
```

#### Complete Configuration Options

```env
# Server and Authentication
CALDAV_URL=https://your-server.com/
CALDAV_USERNAME=your-username
CALDAV_PASSWORD=your-password

# OAuth Alternative (for Google Calendar)
USE_OAUTH=false
GOOGLE_CLIENT_ID=your_client_id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your_client_secret

# Output and Formatting
OUTPUT_DIR=./events
USE_DUE_DATE_EMOJI=true
USE_HASHTAGS=true
USE_CALENDAR_TAGS=true
USE_FRONTMATTER=true
IGNORE_DESCRIPTIONS=false
IGNORE_DECLINED=false
EVENT_CHECKBOXES=false
USE_OBSIDIAN_EMOJIS=false

# Date Filtering
START_DATE=2024-01-01
END_DATE=2025-12-31

# Database Storage (optional)
USE_DATABASE=true
DATABASE_PATH=~/.config/caldav2markdown/calendar.db

# Performance and Multi-Calendar
USE_SERVER_SIDE_FILTERING=true
DISCOVER_CALENDARS=true
INCLUDE_CALENDARS=Work,Personal
EXCLUDE_CALENDARS=Archive,Spam
CALENDAR_ALIASES=Personal Calendar:Personal,Work Calendar:Work,Google Calendar:GCal

# Proxy Configuration (optional, for corporate environments)
PROXY_URL=http://proxy.company.com:8080
PROXY_USERNAME=proxy_user
PROXY_PASSWORD=proxy_pass
```

### Method 2: Command Line Flags

#### Basic CalDAV Usage

```bash
bin/caldav2markdown -url "https://your-server.com/calendars/user/cal/" -username "user" -password "pass"
```

#### Google Calendar with OAuth

```bash
bin/caldav2markdown -url "https://apidata.googleusercontent.com/caldav/v2/primary/events" -oauth -client-id "your_id.apps.googleusercontent.com" -client-secret "your_secret" -frontmatter -hashtags
```

#### Multi-Calendar Processing

```bash
bin/caldav2markdown -url "https://your-server.com/" -username "user" -password "pass" -discover-calendars -include-calendars "Work,Personal" -server-side-filtering
```

## Usage

### Basic Usage

```bash
# Using config file (default: ~/.config/caldav2markdown/config.yaml)
bin/caldav2markdown

# Using custom config file
bin/caldav2markdown -config myconfig.yaml

# Using command line flags with emoji and hashtags
bin/caldav2markdown -url "https://example.com/cal/" -username "user" -password "pass" -emoji -hashtags

# With date filtering
bin/caldav2markdown -start 2024-01-01 -end 2024-12-31
```

### Google Calendar with OAuth

```bash
# First-time OAuth setup (opens browser for authentication)
bin/caldav2markdown -url "https://apidata.googleusercontent.com/caldav/v2/primary/events" -oauth -client-id "your_id.apps.googleusercontent.com" -client-secret "your_secret"

# Subsequent runs (uses stored token)
bin/caldav2markdown -config google.env

# With frontmatter for static site generators
bin/caldav2markdown -oauth -client-id "your_id.apps.googleusercontent.com" -client-secret "your_secret" -frontmatter -hashtags
```

### Multi-Calendar Operations

```bash
# List all available calendars
bin/caldav2markdown -url "https://your-server.com/" -username "user" -password "pass" -list-calendars

# Process all calendars with filtering
bin/caldav2markdown -url "https://your-server.com/" -username "user" -password "pass" -discover-calendars -include-calendars "Work,Personal" -exclude-calendars "Archive,Test"

# High-performance processing with server-side filtering
bin/caldav2markdown -url "https://your-server.com/" -username "user" -password "pass" -discover-calendars -server-side-filtering
```

### Advanced Usage Examples

```bash
# Full-featured processing with all options
bin/caldav2markdown -url "https://your-server.com/" -username "user" -password "pass" \
  -discover-calendars -include-calendars "Work,Personal" \
  -server-side-filtering -frontmatter -emoji -hashtags \
  -start 2024-01-01 -end 2024-12-31 -output ./my-calendar

# Org mode output for Emacs integration
bin/caldav2markdown -url "https://your-server.com/" -username "user" -password "pass" \
  -output-format org -hashtags -calendar-tags -event-checkboxes \
  -output ~/org/calendars

# Single Org file output
bin/caldav2markdown -url "https://your-server.com/" -username "user" -password "pass" \
  -output-format org -single-file -single-file-name calendar.org \
  -hashtags -event-checkboxes -output ./org-calendar

# Emacs diary output (automatic single file mode)
bin/caldav2markdown -url "https://your-server.com/" -username "user" -password "pass" \
  -output-format diary -hashtags -calendar-tags -event-checkboxes \
  -output ~/emacs

# Test connection to server
bin/caldav2markdown -url "https://your-server.com/" -username "user" -password "pass" -test

# Connect through corporate proxy with authentication
bin/caldav2markdown -url "https://your-server.com/" -username "user" -password "pass" \
  -proxy-url "http://proxy.company.com:8080" -proxy-username "proxy_user" -proxy-password "proxy_pass"

# Process specific date range with custom output
bin/caldav2markdown -config myconfig.env -start 2024-06-01 -end 2024-06-30 -output ./june-events
```

### Progress Indicators

The application provides real-time progress feedback during operation:

```
Fetching events from CalDAV server...
[100%] Processing calendar3.ics (3/3)

Found 15 unique events, converting to daily markdown files...
Converted 15 events to markdown format
Converting 8 tasks to markdown format...
Generating daily files...
[ 20%] Processing 2024-06-01 (1/5)
[ 20%] Merging with 2 existing items (1/5)
[ 40%] Processing 2024-06-02 (2/5)
[100%] Processing 2024-06-05 (5/5)

Successfully created 5 daily files for 15 events and 8 tasks
```

### Using Make Commands

```bash
# Build and run with default config
make run

# Test CalDAV connection
make test-connection

# Run with custom config file
make run-config CONFIG_FILE=myconfig.env

# Clean build artifacts
make clean

# Show all available targets
make help
```

### Test Connection

```bash
bin/caldav2markdown -test
```

### Command Line Options

#### Required Options

**Basic Authentication:**
- `-url`: CalDAV server URL

**Choose one authentication method:**
- Basic Auth: `-username` and `-password`
- OAuth: `-oauth`, `-client-id`, and `-client-secret`

#### Optional Options

**Output and Formatting:**
- `-output`: Output directory for files (default: ./events)
- `-output-format`: Output format: "markdown", "org", or "diary" (default: markdown)
- `-single-file`: Generate single file instead of daily files (automatic for diary format)
- `-single-file-name`: Name for single file output (default: calendar.md, calendar.org, or diary)
- `-config`: Configuration file path (default: ~/.config/caldav2markdown/config.yaml)
- `-emoji`: Use 📅 emoji for due dates in tasks (Markdown only)
- `-hashtags`: Add #event and #task hashtags (Markdown) or :tags: (Org)
- `-calendar-tags`: Add calendar name hashtags/tags (e.g., #work or :work:)
- `-frontmatter`: Add YAML frontmatter (Markdown) or properties drawer (Org)
- `-ignore-descriptions`: Ignore event and task descriptions in output
- `-event-checkboxes`: Add checkboxes (Markdown) or TODO states (Org) to events

**Date Filtering:**
- `-start`: Start date for event filtering (YYYY-MM-DD format)
- `-end`: End date for event filtering (YYYY-MM-DD format)

**Performance Options:**
- `-server-side-filtering`: Use CalDAV server-side filtering (faster for large calendars)
- `-test`: Test connection only, don't fetch events

**Multi-Calendar Options:**
- `-discover-calendars`: Discover and process all calendars on the server
- `-include-calendars`: Comma-separated list of calendar names to include
- `-exclude-calendars`: Comma-separated list of calendar names to exclude
- `-list-calendars`: List available calendars and exit

**OAuth Options (for Google Calendar):**
- `-oauth`: Enable OAuth 2.0 authentication
- `-client-id`: Google OAuth Client ID
- `-client-secret`: Google OAuth Client Secret

**Database Options:**
- `-use-database`: Enable SQLite database for storage and change tracking
- `-database-path`: Path to SQLite database file (default: same directory as config)
- `-from-database`: Generate markdown from database instead of fetching from calendar
- `-db-stats`: Show database statistics and exit
- `-db-clear`: Clear all data from database and exit

#### Complete Flag Reference

```bash
bin/caldav2markdown \
  -url "https://server.com/" \
  -username "user" -password "pass" \      # OR use OAuth flags below
  -oauth -client-id "id" -client-secret "secret" \
  -output "./events" \
  -config ".env" \
  -start "2024-01-01" -end "2024-12-31" \
  -emoji -hashtags -frontmatter -ignore-descriptions -event-checkboxes \
  -use-database -database-path "./calendar.db" \
  -server-side-filtering \
  -discover-calendars \
  -include-calendars "Work,Personal" \
  -exclude-calendars "Archive,Test" \
  -test
```

## Output Format

### File Organization

- **Daily Files**: Events and tasks with due dates are organized in `YYYY/MM/YYYY-MM-DD.md` (Markdown) or `YYYY/MM/YYYY-MM-DD.org` (Org mode) files
- **Single File Mode**: All content in one `calendar.md` or `calendar.org` file (when `-single-file` is enabled)
- **Todo File**: Tasks without due dates are saved to `todo.md` (Markdown) or `todo.org` (Org mode) in the output directory root
- **Zero Dates**: Events with zero or invalid dates are saved in `0001/01/` directory
- **Output Format**: Determined by `-output-format` flag or `OUTPUT_FORMAT` config (default: markdown)

### Daily File Format

Each day's events and tasks with due dates are combined in a single `YYYY-MM-DD.md` file:

#### Standard Format

```markdown
# Tuesday, January 15, 2024

## All Day Events

- **Company Holiday** (All Day) [Work] #event

## Scheduled Events

- **Morning Standup** (09:00 - 09:30) @ Conference Room A [Work] #event
  Daily team sync meeting
- **Project Review** (14:00 - 15:30) @ Room B [Work] #event
  Quarterly project progress review
- **One-on-One** at 16:00 @ Manager's Office [Personal] #event
  Weekly check-in meeting

## Tasks

- [ ] **High Priority Task** - 📅 2024-01-15 [Work] #task
- [x] **Completed Task** - 📅 2024-01-15 [Personal] #task
```

#### With YAML Frontmatter (Static Site Generators)

When using `-frontmatter` flag, files include structured metadata:

```markdown
---
date: 2024-01-15
title: Tuesday, January 15, 2024
event_count: 3
task_count: 3
allday_count: 1
tags:
  - work
  - meetings
  - personal
type: daily-calendar
---

# Tuesday, January 15, 2024

## All Day Events

- **Company Holiday** (All Day) #event

## Scheduled Events

- **Morning Standup** (09:00 - 09:30) @ Conference Room A #event
  Daily team sync meeting
- **Project Review** (14:00 - 15:30) @ Room B #event
  Quarterly project progress review

## Tasks

- [ ] **High Priority Task** - 📅 2024-01-15 #task
- [x] **Completed Task** - 📅 2024-01-15 #task
```

### Todo File Format

Tasks without due dates are saved to a separate `todo.md` file in the output directory root:

#### Standard Todo Format

```markdown
# Todo

## Tasks

- [ ] **Research new framework** (Priority: 2) - Status: NEEDS-ACTION #task
  Investigate modern web frameworks for project
- [x] **Update documentation** - Status: COMPLETED #task
  Review and update project documentation
- [ ] **Call client** (Priority: 1) - Status: NEEDS-ACTION #task
```

#### With YAML Frontmatter

```markdown
---
title: Todo List
task_count: 3
type: todo
tags:
  - research
  - documentation
  - client
---

# Todo

## Tasks

- [ ] **Research new framework** (Priority: 2) - Status: NEEDS-ACTION #task
  Investigate modern web frameworks for project
- [x] **Update documentation** - Status: COMPLETED #task
  Review and update project documentation
- [ ] **Call client** (Priority: 1) - Status: NEEDS-ACTION #task
```

### Smart File Merging

The application intelligently merges content when files already exist:
- **Preserves existing content**: Manual edits and custom content are retained
- **Intelligent frontmatter merging**: Custom frontmatter fields are preserved while calendar statistics are updated
- **Adds new items**: Fresh calendar data is merged with existing content
- **Deduplicates automatically**: Duplicate events and tasks are removed
- **Maintains organization**: Content stays properly organized in sections
- **Custom sections preserved**: User-added sections like notes or custom headers are maintained

### Formatting Options

#### With Emoji, Hashtags, and Calendar Tags (enabled)
```markdown
- [ ] **Review PR** - 📅 2024-01-20 [Work] #task #work
- **Team Meeting** (09:00 - 10:00) @ Room A [Personal] #event #personal
```

#### With Hashtags Only (calendar tags disabled)
```markdown
- [ ] **Review PR** - 📅 2024-01-20 [Work] #task
- **Team Meeting** (09:00 - 10:00) @ Room A [Personal] #event
```

#### Standard Format (disabled)
```markdown
- [ ] **Review PR** - Due: 2024-01-20
- **Team Meeting** (09:00 - 10:00) @ Room A
```

#### Description Handling

**With Descriptions (default)**:
```markdown
- **Project Review** (14:00 - 15:30) @ Room B #event
  Quarterly project progress review with stakeholders
- [ ] **Complete Report** - 📅 2024-01-20 #task
  Finish the quarterly report analysis and submit for review
```

**Without Descriptions (`-ignore-descriptions` flag or `IGNORE_DESCRIPTIONS=true`)**:
```markdown
- **Project Review** (14:00 - 15:30) @ Room B #event
- [ ] **Complete Report** - 📅 2024-01-20 #task
```

Use the ignore descriptions option for cleaner, more concise output when detailed descriptions aren't needed.

#### Event Checkboxes

**Regular Events (default)**:
```markdown
- **Team Meeting** (09:00 - 10:00) @ Conference Room A #event
- **Project Review** (14:00 - 15:30) @ Room B #event
```

**Events with Checkboxes (`-event-checkboxes` flag or `EVENT_CHECKBOXES=true`)**:
```markdown
- [x] **Past Meeting** (09:00 - 10:00) @ Conference Room A #event
- [ ] **Future Meeting** (14:00 - 15:30) @ Room B #event
```

Event checkboxes provide:
- **Automatic Past Event Marking**: Past events are automatically marked as completed [x]
- **Task-like Event Tracking**: Treat events as actionable items that can be checked off
- **Meeting Attendance**: Track which meetings you've attended
- **Time-aware Status**: Events are marked based on their actual end time
- **Unified Format**: Maintain consistency with task checkboxes

**Events with Obsidian Tasks Emoji Format (`-obsidian-emojis` flag or `USE_OBSIDIAN_EMOJIS=true`)**:
```markdown
- [ ] **Team Meeting** 🛫 2024-10-17 09:00 ✅ 10:00 @ Conference Room A #event
- [ ] **Project Review** 🛫 2024-10-17 14:00 ✅ 15:30 @ Room B #event
- [ ] **Multi-day Conference** 🛫 2024-10-17 09:00 ✅ 2024-10-18 17:00 @ Convention Center #event
```

Obsidian Tasks emoji format provides:
- **🛫 Start Time Marker**: Uses the Obsidian Tasks "start date" emoji for event start times
- **✅ End Time Marker**: Uses the Obsidian Tasks "done date" emoji for event end times
- **Full Date/Time**: Includes complete date and time information in Obsidian-compatible format
- **Multi-day Support**: Properly handles events spanning multiple days with separate date/time stamps
- **Obsidian Integration**: Seamlessly integrates with Obsidian Tasks plugin for query and filtering

### Ignoring Declined Events

You can automatically skip events you've declined to keep your calendar clean and focused on relevant meetings.

**Configuration Example**:
```env
IGNORE_DECLINED=true
```

**YAML Configuration**:
```yaml
ignore_declined: true
```

**CLI Flag**:
```bash
bin/caldav2markdown --ignore-declined
```

The feature detects declined events by checking:
- **STATUS Property**: Events with `STATUS=CANCELLED` are filtered out
- **PARTSTAT Parameter**: Events where any attendee has `PARTSTAT=DECLINED` are filtered out

**Benefits**:
- **Clean Calendar View**: Only see events you've accepted or are considering
- **Reduced Clutter**: Automatically removes declined meeting invitations
- **Better Focus**: Concentrate on meetings you're actually attending
- **Multi-Calendar Support**: Works across all calendar sources (CalDAV, ICS files)

When enabled, the output will show how many declined events were skipped:
```
Converted 45 events to markdown format (skipped 5 declined)
```

### Calendar Names and Aliases

Calendar names are automatically extracted from events and tasks and displayed in square brackets. You can configure custom aliases for cleaner display names.

**Configuration Example**:
```env
# Map long calendar names to shorter aliases
CALENDAR_ALIASES=Personal Calendar:Personal,Work Calendar:Work,Google Calendar:GCal,john.doe@company.com:John
```

**Output Examples**:

**Without Aliases**:
```markdown
- **Team Meeting** (09:00 - 10:00) @ Conference Room A [Personal Calendar] #event
- [ ] **Submit Report** - 📅 2024-01-15 [Work Calendar] #task
```

**With Aliases Applied**:
```markdown
- **Team Meeting** (09:00 - 10:00) @ Conference Room A [Personal] #event
- [ ] **Submit Report** - 📅 2024-01-15 [Work] #task
```

**Calendar Name Sources**:
- **X-WR-CALNAME Property**: Extracted from individual events/tasks when available
- **PRODID Detection**: Automatically detects Google Calendar and other common sources
- **ICS File Sources**: Calendar names from file metadata
- **Multi-calendar Discovery**: Names from CalDAV calendar discovery

**Benefits**:
- **Source Identification**: Easily see which calendar each item came from
- **Multi-calendar Organization**: Distinguish between work, personal, and shared calendars
- **Clean Display**: Use short aliases instead of long technical calendar names
- **Consistent Formatting**: Uniform display across all events and tasks

## Org Mode Output

The application provides full support for Emacs Org mode format as an alternative to Markdown, perfect for Emacs users and Org-agenda integration.

### Enabling Org Mode

**Environment File**:
```env
OUTPUT_FORMAT=org
```

**YAML Configuration**:
```yaml
output_format: org
```

**Command Line**:
```bash
bin/caldav2markdown -output-format org
```

### Org Mode Features

#### Native Org Syntax
- **TODO States**: Events and tasks use `** TODO` for future items, `** DONE` for completed/past items
- **Scheduling**: `SCHEDULED: <2024-11-05 Tue 09:00-10:00>` for timed events
- **Deadlines**: `DEADLINE: <2024-11-05 Tue>` for tasks with due dates
- **Properties Drawer**: Metadata stored in `:PROPERTIES:` drawer (ID, LOCATION, CALENDAR, STATUS, CATEGORIES)
- **Tags**: Org-style tags `:event:work:personal:` instead of hashtags
- **Priority Levels**: iCal priorities automatically mapped to Org priorities [#A], [#B], [#C]

#### Example Org Mode Output

**Daily Org File** (`2024-11-05.org`):
```org
#+TITLE: Tuesday, November 5, 2024

* All Day Events

# uid:company-holiday-123
** TODO Company Holiday :event:work:
   SCHEDULED: <2024-11-05 Tue>
   :PROPERTIES:
   :ID: company-holiday-123
   :CALENDAR: Work
   :STATUS: CONFIRMED
   :END:

* Scheduled Events

# uid:team-meeting-456
** TODO Team Meeting :event:work:
   SCHEDULED: <2024-11-05 Tue 09:00-10:00>
   :PROPERTIES:
   :ID: team-meeting-456
   :LOCATION: Conference Room A
   :CALENDAR: Work
   :STATUS: CONFIRMED
   :END:
   Weekly team sync to discuss project progress

# uid:project-review-789
** DONE Project Review :event:work:
   SCHEDULED: <2024-11-04 Mon 14:00-15:30>
   :PROPERTIES:
   :ID: project-review-789
   :LOCATION: Room B
   :CALENDAR: Work
   :STATUS: CONFIRMED
   :CATEGORIES: meetings, planning
   :END:
   Quarterly project progress review

* Tasks

# uid:complete-report-abc
** TODO Complete Report [#A] :task:work:
   DEADLINE: <2024-11-05 Tue>
   :PROPERTIES:
   :ID: complete-report-abc
   :CALENDAR: Work
   :STATUS: NEEDS-ACTION
   :PRIORITY: 1
   :CATEGORIES: reports, deadlines
   :END:
   Finish quarterly report and submit for review
```

**Todo Org File** (`todo.org`):
```org
#+TITLE: Tasks Without Due Date

* Tasks

# uid:research-framework-123
** TODO Research new framework [#B] :task:personal:
   :PROPERTIES:
   :ID: research-framework-123
   :CALENDAR: Personal
   :STATUS: NEEDS-ACTION
   :PRIORITY: 5
   :END:
   Investigate modern web frameworks for next project

# uid:call-client-456
** TODO Call client [#A] :task:work:
   :PROPERTIES:
   :ID: call-client-456
   :CALENDAR: Work
   :STATUS: NEEDS-ACTION
   :PRIORITY: 1
   :END:
```

### Single Org File Mode

Create a single consolidated Org file instead of daily files:

```bash
bin/caldav2markdown -output-format org -single-file -single-file-name calendar.org
```

**Single File Output** (`calendar.org`):
```org
#+TITLE: Calendar

* Monday, November 4, 2024

** All Day Events

# uid:event-123
*** TODO Holiday :event:work:
    SCHEDULED: <2024-11-04 Mon>
    :PROPERTIES:
    :ID: event-123
    :CALENDAR: Work
    :END:

** Scheduled Events

# uid:meeting-456
*** DONE Morning Meeting :event:work:
    SCHEDULED: <2024-11-04 Mon 09:00-10:00>
    :PROPERTIES:
    :ID: meeting-456
    :LOCATION: Room A
    :CALENDAR: Work
    :END:

* Tuesday, November 5, 2024

** Scheduled Events

# uid:meeting-789
*** TODO Afternoon Meeting :event:personal:
    SCHEDULED: <2024-11-05 Tue 14:00-15:00>
    :PROPERTIES:
    :ID: meeting-789
    :LOCATION: Coffee Shop
    :CALENDAR: Personal
    :END:

* Tasks Without Due Date

# uid:task-abc
** TODO General Task :task:work:
   :PROPERTIES:
   :ID: task-abc
   :CALENDAR: Work
   :END:
```

### Org Mode Configuration Examples

#### Basic Org Mode Setup
```env
OUTPUT_FORMAT=org
CALDAV_URL=https://your-server.com/caldav/calendars/user/calendar/
CALDAV_USERNAME=user
CALDAV_PASSWORD=password
OUTPUT_DIR=./org-calendar
USE_HASHTAGS=true
USE_CALENDAR_TAGS=true
EVENT_CHECKBOXES=true
```

#### Emacs Org-Agenda Integration
```env
OUTPUT_FORMAT=org
CALDAV_URL=https://your-server.com/caldav/calendars/user/calendar/
CALDAV_USERNAME=user
CALDAV_PASSWORD=password
OUTPUT_DIR=~/org/calendars
USE_HASHTAGS=true
USE_CALENDAR_TAGS=true
EVENT_CHECKBOXES=true
DISCOVER_CALENDARS=true
CALENDAR_ALIASES=Personal Calendar:Personal,Work Calendar:Work
```

#### Single Org File for All Calendars
```yaml
---
output_format: org
output: ./org-calendar
single_file: true
single_file_name: calendar.org
use_hashtags: true
event_checkboxes: true

sources:
  - type: caldav
    name: My Calendar
    caldav:
      url: https://your-server.com/caldav/
      username: user
      password: password
      discover_calendars: true
```

### Org Mode Formatting Options

All standard formatting options work with Org mode:

- **USE_HASHTAGS**: Generates `:event:` and `:task:` tags
- **USE_CALENDAR_TAGS**: Adds calendar name tags like `:work:`, `:personal:`
- **EVENT_CHECKBOXES**: Uses `TODO` for future items, `DONE` for past items
- **IGNORE_DESCRIPTIONS**: Omits event/task descriptions
- **CALENDAR_ALIASES**: Works with calendar name tags

**Note**: Markdown-specific options (USE_DUE_DATE_EMOJI, USE_OBSIDIAN_EMOJIS, OBSIDIAN_TASKS) are ignored when using Org mode format.

### Org-Agenda Integration

Add the generated Org files to your Org-agenda:

```elisp
;; In your Emacs configuration
(setq org-agenda-files '("~/org/calendars/"))

;; Or add individual files
(add-to-list 'org-agenda-files "~/org/calendars/2024/11/2024-11-05.org")
```

### Advantages of Org Mode

- **Native Emacs Integration**: Seamless integration with Emacs and Org-agenda
- **Powerful Scheduling**: Leverage Org mode's native scheduling and deadline system
- **Tag System**: Use Org's flexible tag system for organization
- **Properties Drawer**: Store structured metadata in a standard Org format
- **TODO States**: Native TODO/DONE states with optional custom keywords
- **Extensible**: Full access to Org mode's extensive feature set

## Emacs Diary Output

The application provides full support for traditional Emacs diary format, perfect for users who prefer the classic GNU Emacs calendar system.

### Enabling Diary Format

**Environment File**:
```env
OUTPUT_FORMAT=diary
```

**YAML Configuration**:
```yaml
output_format: diary
```

**Command Line**:
```bash
bin/caldav2markdown -output-format diary
```

### Diary Format Features

#### Automatic Single File Mode
When diary format is selected, single file mode is automatically enabled following the traditional Emacs convention of using a single `diary` file:

```bash
# Automatically creates ~/emacs/diary
bin/caldav2markdown -output-format diary -output ~/emacs

# Custom filename if desired
bin/caldav2markdown -output-format diary -single-file-name my-calendar -output ~/emacs
```

#### Diary Syntax
The diary format uses traditional Emacs diary notation:

```
; Emacs Diary File
; Generated by caldav2markdown
;
; Format: MM/DD/YYYY [TIME] TITLE [LOCATION] [CALENDAR] #tags {uid:...}
;

1/15/2025 10:00am-11:00am Team Meeting @ Conference Room A [Work] #event #work
1/20/2025 Company Holiday [Personal] #event #personal
1/25/2025 5:00pm Complete project report [Work] #task #work
%%(diary-entry "TODO") Research new framework [Personal] #task #personal
```

#### Format Elements

- **Date Format**: `MM/DD/YYYY` (standard American format used by Emacs)
- **Timed Events**: `1/15/2025 10:00am-11:00am Meeting title`
- **All-day Events**: `1/20/2025 Event title`
- **Multi-day Events**: `1/15/2025 9:00am-1/16/2025 5:00pm Conference`
- **Tasks with Due Dates**: `1/25/2025 Task title`
- **Tasks without Due Dates**: `%%(diary-entry "TODO") Task title`
- **Completion Markers**: `[DONE]` prefix for completed items (when `EVENT_CHECKBOXES` enabled)
- **Location**: `@ Location Name` after the title
- **Calendar Source**: `[Calendar Name]` showing source calendar
- **Tags**: `#event`, `#task`, `#calendar-name` (when hashtags enabled)
- **UID Tracking**: `{uid:...}` for deduplication (hidden metadata)

### Example Diary Output

**Single Diary File** (`diary`):
```
; Emacs Diary File
; Generated by caldav2markdown
;
; Format: MM/DD/YYYY [TIME] TITLE [LOCATION] [CALENDAR] #tags {uid:...}
;

1/15/2025 [DONE] 9:00am-10:00am Morning Standup @ Room A [Work] #event #work {uid:meeting-123}
1/15/2025 2:00pm-3:30pm Project Review @ Room B [Work] #event #work {uid:review-456}
1/15/2025 Company Holiday [Personal] #event #personal {uid:holiday-789}
1/20/2025 5:00pm Submit quarterly report [Work] #task #work {uid:task-abc}
1/25/2025 9:00am-1/26/2025 5:00pm Annual Conference @ Convention Center [Work] #event #work {uid:conf-def}
%%(diary-entry "TODO") Research new tools [Personal] #task #personal {uid:task-xyz}
%%(diary-entry "TODO") [DONE] Update documentation [Work] #task #work {uid:task-done}
```

**Todo Diary File** (`todo-diary`):
Tasks without due dates are automatically saved to a separate `todo-diary` file:
```
; Emacs Diary File
; Generated by caldav2markdown
;
; Format: MM/DD/YYYY [TIME] TITLE [LOCATION] [CALENDAR] #tags {uid:...}
;

%%(diary-entry "TODO") Research new framework [Personal] #task #personal {uid:research-123}
  Investigate modern web frameworks for next project
%%(diary-entry "TODO") Call client [Work] #task #work {uid:call-456}
%%(diary-entry "TODO") [DONE] Setup environment [Work] #task #work {uid:setup-789}
```

### Diary Configuration Examples

#### Basic Diary Setup
```env
OUTPUT_FORMAT=diary
CALDAV_URL=https://your-server.com/caldav/calendars/user/calendar/
CALDAV_USERNAME=user
CALDAV_PASSWORD=password
OUTPUT_DIR=~/emacs
USE_HASHTAGS=true
USE_CALENDAR_TAGS=true
EVENT_CHECKBOXES=true
```

#### Emacs Calendar Integration
```env
OUTPUT_FORMAT=diary
CALDAV_URL=https://your-server.com/caldav/calendars/user/calendar/
CALDAV_USERNAME=user
CALDAV_PASSWORD=password
OUTPUT_DIR=~/.emacs.d
USE_HASHTAGS=true
USE_CALENDAR_TAGS=true
EVENT_CHECKBOXES=true
DISCOVER_CALENDARS=true
CALENDAR_ALIASES=Personal Calendar:Personal,Work Calendar:Work
```

#### Google Calendar to Diary
```yaml
---
output_format: diary
output: ~/emacs
use_hashtags: true
use_calendar_tags: true
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
```

### Diary Formatting Options

All standard formatting options work with diary format:

- **USE_HASHTAGS**: Adds `#event` and `#task` tags to entries
- **USE_CALENDAR_TAGS**: Adds calendar name tags like `#work`, `#personal`
- **EVENT_CHECKBOXES**: Uses `[DONE]` prefix for past events and completed tasks
- **IGNORE_DESCRIPTIONS**: Omits event/task descriptions
- **CALENDAR_ALIASES**: Works with calendar name display and tags

**Note**: Markdown-specific options (USE_DUE_DATE_EMOJI, USE_OBSIDIAN_EMOJIS, OBSIDIAN_TASKS) and Org-specific options are ignored when using diary format.

### Emacs Calendar Integration

Use the generated diary file with Emacs calendar commands:

```elisp
;; In your Emacs configuration
;; Set diary file location
(setq diary-file "~/emacs/diary")

;; Enable diary display in calendar
(setq calendar-mark-diary-entries-flag t)

;; Optional: fancy diary display
(add-hook 'diary-list-entries-hook 'diary-include-other-diary-files)
(add-hook 'diary-mark-entries-hook 'diary-mark-included-diary-files)

;; View calendar and diary
;; M-x calendar
;; Press 'd' in calendar to view diary entries for selected day
;; Press 'm' to mark days with diary entries
```

### Smart Merging and Sorting

The diary formatter provides intelligent file management:

- **Smart Merging**: Preserves custom diary entries and comments when updating
- **Automatic Sorting**: Entries are chronologically sorted by date for easy reading
- **Deduplication**: UID-based deduplication prevents duplicate entries
- **Comment Preservation**: User-added comments and sections are maintained
- **Header Generation**: Auto-generates helpful header comments in new files

### Advantages of Diary Format

- **Traditional Emacs Integration**: Works seamlessly with GNU Emacs calendar-mode
- **Simple Syntax**: Easy-to-read date-based format
- **Single File**: Follows Emacs convention of consolidated diary file
- **Direct Compatibility**: No conversion needed for `M-x calendar` and `M-x diary`
- **Lightweight**: Minimal overhead, fast loading in Emacs
- **Time-Tested**: Based on decades-old Emacs diary system

## Date/Time Format Support

The application supports multiple iCalendar date/time formats with full time zone support:

### Date/Time Formats
- **UTC Time with Z suffix**: `20240924T143000Z` (September 24, 2024 at 14:30:00 UTC)
- **Local Time without Z suffix**: `20240924T143000` (September 24, 2024 at 14:30:00 local time)
- **Date Only**: `20240924` (September 24, 2024 - all-day event)

### Time Zone Support
- **TZID Parameters**: Full support for TZID parameters in events and tasks
- **IANA Time Zones**: Supports standard IANA time zone database (e.g., America/New_York, Europe/London)
- **Common Mappings**: Built-in mappings for Microsoft Exchange and other system-specific time zone names
- **Automatic Conversion**: Time zones are properly converted and displayed in their local time

### Example Time Zone Usage
```ics
DTSTART;TZID=America/New_York:20240924T143000
DTSTART;TZID=Europe/London:20240924T143000
DTSTART;TZID=Eastern Standard Time:20240924T143000
```

This enhancement ensures compatibility with various CalDAV servers and accurate time representation across different time zones.

## Recurring Events and Tasks

### Smart Processing Logic

The application uses an intelligent approach to handle recurring events and tasks:

1. **Expansion First**: All recurring items (VEVENT and VTODO with RRULE) are expanded into individual instances
2. **Date Filtering Second**: Each expanded instance is then filtered by its specific date/time
3. **Global Deduplication**: Duplicate instances across calendars are removed using UID-based tracking

### Benefits

- **No Missing Events**: Recurring events with start dates outside your filter range are still processed if they have instances within the range
- **Accurate Task Processing**: Recurring tasks work identically to events, expanding based on due dates or start dates
- **Performance Optimized**: Server-side filtering is attempted first, with client-side expansion as fallback

### Example Scenario

```
Filter Range: 2024-06-01 to 2024-06-30
Recurring Event: Weekly meeting starting 2024-01-01, every Monday

Result: All Monday meetings in June 2024 are included, even though the original event started before the filter range
```

This fix ensures comprehensive calendar processing without missing important recurring items.

## Database Storage and Change Tracking

### Overview

The application includes optional SQLite database storage for event and todo tracking, providing smart deduplication and change detection capabilities.

### Database Features

- **Event and Todo Storage**: All calendar items are stored with complete metadata
- **Change Detection**: Tracks which fields changed between calendar fetches
- **Smart Markdown Updates**: Only generates markdown for new or changed items
- **Historical Tracking**: Records first_seen and last_seen timestamps
- **UID-based Deduplication**: Prevents duplicate entries across multiple sources
- **Statistics**: View comprehensive database statistics

### Configuration

**Environment File**:
```env
USE_DATABASE=true
DATABASE_PATH=~/.config/caldav2markdown/calendar.db
```

**YAML Configuration**:
```yaml
use_database: true
database_path: ~/.config/caldav2markdown/calendar.db
```

**Command Line**:
```bash
bin/caldav2markdown -use-database -database-path ~/.config/caldav2markdown/calendar.db
```

### Usage Examples

#### Basic Database Usage

```bash
# First run - all items are new
bin/caldav2markdown -use-database -config config.yaml
Fetching events from CalDAV server...
Storing events and todos in database...
Database: 150 new events, 0 updated events, 25 new todos, 0 updated todos
Generating markdown for 150 new/changed events and 25 new/changed todos
Successfully created 45 daily files for 150 events and 25 tasks

# Second run - no changes
bin/caldav2markdown -use-database -config config.yaml
Fetching events from CalDAV server...
Storing events and todos in database...
Database: 0 new events, 150 updated events, 0 new todos, 25 updated todos
No new or changed items - skipping markdown generation
Successfully processed 0 events and 0 tasks (all unchanged)

# Third run - 3 events changed
bin/caldav2markdown -use-database -config config.yaml
Fetching events from CalDAV server...
Storing events and todos in database...
Changes detected: 3 events modified, 0 todos modified
Database: 0 new events, 150 updated events, 0 new todos, 25 updated todos
Generating markdown for 3 new/changed events and 0 new/changed todos
Successfully created 3 daily files for 3 events and 0 tasks
```

#### Generate Markdown from Database

Instead of fetching from the calendar server, generate markdown from stored database records:

```bash
# Generate markdown from database (no calendar fetch)
bin/caldav2markdown -use-database -from-database -config config.yaml

# With date filtering
bin/caldav2markdown -use-database -from-database -start 2024-06-01 -end 2024-06-30
```

#### Database Statistics

View comprehensive statistics about stored events and todos:

```bash
bin/caldav2markdown -use-database -db-stats -config config.yaml

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

Remove all stored events and todos:

```bash
bin/caldav2markdown -use-database -db-clear -config config.yaml
Database cleared successfully
```

### Change Detection

The database tracks changes to individual fields, providing detailed change information:

**Tracked Event Fields**:
- Summary (title)
- Description
- Location
- Start time
- End time
- All-day flag
- Status
- Categories

**Tracked Todo Fields**:
- Summary (title)
- Description
- Due date
- Priority
- Status
- Categories

**Example Change Report**:
```
Changes detected:
  Event "Team Meeting" modified: summary, location, start_time
  Todo "Submit Report" modified: due_date, priority
```

### Benefits

- **No Duplicate Processing**: Unchanged events never regenerate markdown
- **Faster Runs**: Only process items that actually changed
- **Bandwidth Savings**: Database tracks everything, markdown only shows changes
- **Historical Tracking**: Know when events were first seen and last updated
- **Offline Generation**: Generate markdown from database without fetching from server
- **Statistics and Reporting**: Comprehensive database statistics

### Database Location

The database file is stored in the location specified by `DATABASE_PATH`:

- **Default**: Same directory as config file, named `calendar.db`
- **XDG Config**: `~/.config/caldav2markdown/calendar.db` when using XDG paths
- **Custom**: Any path specified in configuration or via `-database-path` flag

The database uses SQLite's WAL (Write-Ahead Logging) mode for better concurrent access performance.

## Development

### Running Tests

```bash
make test
```

### Cross-Platform Builds

```bash
# Build for all platforms
make build-all

# Build for specific platforms
make build-linux    # Linux AMD64
make build-darwin   # macOS AMD64 and ARM64
make build-windows  # Windows AMD64
```

### Development Build with Debug Info

```bash
make build-dev
```

## CalDAV Server Compatibility

This tool works with any RFC 4791-compliant CalDAV server, including:

### Fully Supported Servers

- **NextCloud**: Full support with multi-calendar discovery and server-side filtering
- **ownCloud**: Complete CalDAV compatibility with advanced features
- **Apple Calendar Server**: Full CalDAV support and calendar discovery
- **SabreDAV**: Complete server-side filtering and multi-calendar support
- **Radicale**: Basic CalDAV support (client-side filtering recommended)

### Cloud Services

- **Google Calendar**:
  - **OAuth 2.0** (Recommended): Full access with automatic token management
  - **CalDAV URL**: `https://apidata.googleusercontent.com/caldav/v2/primary/events`
  - **Multi-calendar**: Replace `primary` with calendar ID for specific calendars
  - **Note**: Google requires OAuth 2.0 for new applications (2025+ requirement)

- **Microsoft Exchange**:
  - CalDAV interface supported when enabled
  - May require specific URL format depending on Exchange version
  - Server-side filtering support varies by Exchange version

- **iCloud**:
  - App-specific passwords required
  - CalDAV URL format: `https://caldav.icloud.com/[DSID]/calendars/`
  - Multi-calendar discovery supported

### Performance Recommendations

- **Large Calendars**: Use `-server-side-filtering` flag for better performance
- **Multiple Calendars**: Use `-discover-calendars` with include/exclude filters
- **Google Calendar**: Always use OAuth 2.0 for best performance and reliability
- **NextCloud/ownCloud**: Enable server-side filtering for optimal speed

### Authentication Methods by Server

| Server | Basic Auth | OAuth 2.0 | App Passwords |
|--------|------------|-----------|---------------|
| NextCloud | ✅ | ❌ | ✅ |
| ownCloud | ✅ | ❌ | ✅ |
| Google Calendar | ❌* | ✅ | ❌ |
| iCloud | ❌ | ❌ | ✅ |
| Exchange | ✅ | Varies | ✅ |

*Google Calendar requires OAuth 2.0 for new applications as of 2025

## OAuth Setup (Google Calendar)

### Prerequisites

1. **Google Cloud Console Setup**:
   - Visit [Google Cloud Console](https://console.cloud.google.com/)
   - Create a new project or select existing one
   - Enable the "Calendar API"
   - Go to "Credentials" → "Create Credentials" → "OAuth 2.0 Client ID"
   - Choose "Desktop application" as application type
   - Download the credentials JSON or note the Client ID and Client Secret

2. **Configure Application**:
   ```env
   # In your .env file
   USE_OAUTH=true
   GOOGLE_CLIENT_ID=your_client_id.apps.googleusercontent.com
   GOOGLE_CLIENT_SECRET=your_client_secret
   CALDAV_URL=https://apidata.googleusercontent.com/caldav/v2/primary/events
   ```

### First-Time Authentication

```bash
# Run the application - it will open your default browser
bin/caldav2markdown -oauth -client-id "your_id.apps.googleusercontent.com" -client-secret "your_secret"

# Follow the browser prompts to:
# 1. Sign in to your Google account
# 2. Grant calendar access permissions
# 3. Copy the authorization code if needed

# Token is automatically saved to ~/.config/caldav2markdown/token.json
# Subsequent runs will use the stored token automatically
```

### Token Management

- **Token Location**: `~/.config/caldav2markdown/token.json`
- **Token Refresh**: Automatic - no manual intervention needed
- **Token Expiry**: Handled automatically by the OAuth client
- **Revoke Access**: Delete token file or revoke in Google Account settings

### Troubleshooting

#### Common OAuth Issues

1. **"Invalid Client ID" Error**:
   - Verify Client ID is copied correctly (ends with `.apps.googleusercontent.com`)
   - Ensure Calendar API is enabled in Google Cloud Console

2. **"Permission Denied" Error**:
   - Check that OAuth consent screen is configured
   - Verify your Google account has calendar access

3. **"Token Refresh Failed"**:
   - Delete `~/.config/caldav2markdown/token.json`
   - Re-run authentication flow

4. **Browser Doesn't Open**:
   - Copy the authorization URL from console output
   - Manually paste into browser

#### Calendar Discovery Issues

1. **"No Calendars Found"**:
   - Verify you have calendars in your Google account
   - Try using specific calendar URLs instead of discovery

2. **"Permission Denied for Calendar"**:
   - Ensure OAuth scope includes calendar read access
   - Re-authenticate if scope changed

#### Performance Optimization

1. **Slow Processing**:
   - Use `-server-side-filtering` for large calendars
   - Specify date ranges with `-start` and `-end`
   - Use specific calendar IDs instead of discovery

2. **Rate Limiting**:
   - Google Calendar API has usage quotas
   - Consider processing smaller date ranges
   - Use server-side filtering to reduce API calls

### Multi-Calendar Google Setup

```bash
# List all your Google calendars
bin/caldav2markdown -oauth -client-id "your_id" -client-secret "secret" -list-calendars

# Process specific calendars by name
bin/caldav2markdown -oauth -client-id "your_id" -client-secret "secret" -discover-calendars -include-calendars "Work,Personal"

# Use specific calendar URLs (more efficient)
bin/caldav2markdown -url "https://apidata.googleusercontent.com/caldav/v2/calendar_id@group.calendar.google.com/events" -oauth -client-id "your_id" -client-secret "secret"
```

## Static Site Generator Integration

The frontmatter feature makes caldav2markdown perfect for static site generators:

### Hugo Integration

```yaml
# config.yaml
markup:
  goldmark:
    renderer:
      unsafe: true

# Generate calendar pages
bin/caldav2markdown -frontmatter -hashtags -output content/calendar/
```

### Jekyll Integration

```yaml
# _config.yml
plugins:
  - jekyll-feed

# Generate posts
bin/caldav2markdown -frontmatter -hashtags -output _posts/
```

### Obsidian Integration

```bash
# Generate for Obsidian vault with full task support
bin/caldav2markdown -obsidian-tasks -output /path/to/obsidian/vault/Calendar/

# Manual setup with specific options
bin/caldav2markdown -emoji -hashtags -calendar-tags -event-checkboxes -frontmatter -output /path/to/obsidian/vault/Calendar/

# Link events: Use [[2024-01-15]] to reference daily files
```

## Advanced Workflows

### Automated Sync

```bash
#!/bin/bash
# sync-calendar.sh - Automated calendar sync script

# Pull latest events
bin/caldav2markdown -config ~/.config/caldav2markdown/config.env

# Commit to git (optional)
cd /path/to/output/directory
git add .
git commit -m "Update calendar: $(date)"
git push origin main
```

### Custom Processing

```bash
# Weekly reports
bin/caldav2markdown -start $(date -d "7 days ago" +%Y-%m-%d) -end $(date +%Y-%m-%d) -output weekly/

# Monthly archives
bin/caldav2markdown -start 2024-01-01 -end 2024-01-31 -output archives/2024-01/

# Project-specific calendars
bin/caldav2markdown -discover-calendars -include-calendars "ProjectA,ProjectB" -hashtags -output projects/
```
