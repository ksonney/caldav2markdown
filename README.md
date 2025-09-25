# caldav2markdown

Proof-of-concept tool to read from CalDAV servers and write calendar events as markdown files.

## Features

- **CalDAV Integration**: Connect to CalDAV servers with authentication
- **Event Processing**: Fetch calendar events from .ics files with deduplication
- **Markdown Conversion**: Convert events and tasks to well-formatted markdown files
- **Smart File Management**: Intelligent file merging - updates existing files instead of overwriting
- **Progress Indicators**: Real-time progress reporting for long operations
- **Flexible Formatting**:
  - Support for both all-day and timed events
  - Optional 📅 emoji for due dates in tasks
  - Optional #event and #task hashtags for better organization
- **Date/Time Handling**: Enhanced date/time format support (UTC, local time, date-only)
- **Recurring Events**: Full RRULE expansion support for recurring events
- **Task Management**: Todo/task extraction and markdown conversion with due dates
- **Date Filtering**: Configurable date range filtering for events and tasks
- **Organization**: Events organized in daily files with YYYY/MM directory structure
- **Configuration**: Multiple configuration methods (environment file or command line flags)
- **Deduplication**: Automatic removal of duplicate events across calendar files

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

Edit `.env` with your CalDAV server details:

```env
# Required Settings
CALDAV_URL=https://your-caldav-server.com/calendars/username/calendar-name/
CALDAV_USERNAME=your-username
CALDAV_PASSWORD=your-password

# Optional Settings
OUTPUT_DIR=./events
START_DATE=2024-01-01
END_DATE=2024-12-31
USE_DUE_DATE_EMOJI=true
USE_HASHTAGS=true
```

### Method 2: Command Line Flags

```bash
bin/caldav2markdown -url "https://your-server.com/calendars/user/cal/" -username "user" -password "pass"
```

## Usage

### Basic Usage

```bash
# Using config file (default: .env)
bin/caldav2markdown

# Using custom config file
bin/caldav2markdown -config myconfig.env

# Using command line flags with emoji and hashtags
bin/caldav2markdown -url "https://example.com/cal/" -username "user" -password "pass" -emoji -hashtags

# With date filtering
bin/caldav2markdown -start 2024-01-01 -end 2024-12-31
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

**Required Options:**
- `-url`: CalDAV server URL
- `-username`: CalDAV username
- `-password`: CalDAV password

**Optional Options:**
- `-output`: Output directory for markdown files (default: ./events)
- `-config`: Configuration file path (default: .env)
- `-start`: Start date for event filtering (YYYY-MM-DD format)
- `-end`: End date for event filtering (YYYY-MM-DD format)
- `-emoji`: Use 📅 emoji for due dates in tasks
- `-hashtags`: Add #event and #task hashtags
- `-test`: Test connection only, don't fetch events

## Output Format

### Events

Events are organized in a `YYYY/MM/` directory structure. All events for a single day are saved in a single file named `YYYY-MM-DD.md` as a markdown list.

For events with zero or invalid dates, files are saved in `0001/01/` directory.

### Daily File Format

Each day's events and tasks are combined in a single `YYYY-MM-DD.md` file:

```markdown
# Tuesday, January 15, 2024

## All Day Events

- **Company Holiday** (All Day) #event

## Scheduled Events

- **Morning Standup** (09:00 - 09:30) @ Conference Room A #event
  Daily team sync meeting
- **Project Review** (14:00 - 15:30) @ Room B #event
  Quarterly project progress review
- **One-on-One** at 16:00 @ Manager's Office #event
  Weekly check-in meeting

## Tasks

- [ ] **High Priority Task** - 📅 2024-01-15 #task
- [x] **Completed Task** - 📅 2024-01-15 #task
- [ ] **No Due Date Task** #task
```

### Smart File Merging

The application intelligently merges content when files already exist:
- **Preserves existing content**: Manual edits and custom content are retained
- **Adds new items**: Fresh calendar data is merged with existing content
- **Deduplicates automatically**: Duplicate events and tasks are removed
- **Maintains organization**: Content stays properly organized in sections

### Formatting Options

#### With Emoji and Hashtags (enabled)
```markdown
- [ ] **Review PR** - 📅 2024-01-20 #task
- **Team Meeting** (09:00 - 10:00) @ Room A #event
```

#### Standard Format (disabled)
```markdown
- [ ] **Review PR** - Due: 2024-01-20
- **Team Meeting** (09:00 - 10:00) @ Room A
```

## Date/Time Format Support

The application supports multiple iCalendar date/time formats:

- **UTC Time with Z suffix**: `20240924T143000Z` (September 24, 2024 at 14:30:00 UTC)
- **Local Time without Z suffix**: `20240924T143000` (September 24, 2024 at 14:30:00 local time)
- **Date Only**: `20240924` (September 24, 2024 - all-day event)

This enhancement ensures compatibility with various CalDAV servers that may format date/time values differently.

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

This tool should work with any CalDAV-compliant server, including:

- NextCloud
- ownCloud
- Apple Calendar Server
- Google Calendar (CalDAV interface)
- Microsoft Exchange (with CalDAV enabled)
