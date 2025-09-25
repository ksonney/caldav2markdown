# caldav2markdown

Proof-of-concept tool to read from CalDAV servers and write calendar events as markdown files.

## Features

- Connect to CalDAV servers with authentication
- Fetch calendar events from .ics files
- Convert events to well-formatted markdown files
- Support for both all-day and timed events with flexible date/time parsing
- Configuration via environment file or command line flags
- Safe filename generation (handles special characters)
- Recurring events support with RRULE expansion
- Todo/task extraction and markdown conversion
- Enhanced date/time format support (UTC with Z suffix, local time without Z suffix, date-only)
- Configurable date range filtering

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
CALDAV_URL=https://your-caldav-server.com/calendars/username/calendar-name/
CALDAV_USERNAME=your-username
CALDAV_PASSWORD=your-password
OUTPUT_DIR=./events
START_DATE=2000-01-01
END_DATE=2026-12-31
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

# Using command line flags
bin/caldav2markdown -url "https://example.com/cal/" -username "user" -password "pass" -output "./my-events"
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

- `-url`: CalDAV server URL
- `-username`: CalDAV username
- `-password`: CalDAV password
- `-output`: Output directory for markdown files (default: ./events)
- `-config`: Configuration file path (default: .env)
- `-start-date`: Start date for event filtering (YYYY-MM-DD format)
- `-end-date`: End date for event filtering (YYYY-MM-DD format)
- `-test`: Test connection only, don't fetch events

## Output Format

### Events

Events are organized in a `YYYY/MM/` directory structure. All events for a single day are saved in a single file named `YYYY-MM-DD.md` as a markdown list.

For events with zero or invalid dates, files are saved in `0001/01/` directory.

### Daily Event Format

```markdown
# Tuesday, January 15, 2024

## All Day Events

- **Company Holiday** (All Day)

## Scheduled Events

- **Morning Standup** (09:00 - 09:30) @ Conference Room A
  Daily team sync meeting
- **Project Review** (14:00 - 15:30) @ Room B
  Quarterly project progress review
- **One-on-One** at 16:00 @ Manager's Office
  Weekly check-in meeting
```

### Tasks/Todos

Todos are saved in a single `tasks.md` file in the output directory:

```markdown
# Tasks

- [ ] **High Priority Task** - Due: 2024-01-20 15:00
- [x] **Completed Task** - Due: 2024-01-15 12:00
- [ ] **No Due Date Task**
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
