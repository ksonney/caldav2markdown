# caldav2markdown

Proof-of-concept tool to read from CalDAV servers and write calendar events as markdown files.

## Features

- Connect to CalDAV servers with authentication
- Fetch calendar events from .ics files
- Convert events to well-formatted markdown files
- Support for both all-day and timed events
- Configuration via environment file or command line flags
- Safe filename generation (handles special characters)

## Installation

```bash
go build -o caldav2markdown ./cmd/caldav2markdown
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
```

### Method 2: Command Line Flags

```bash
./caldav2markdown -url "https://your-server.com/calendars/user/cal/" -username "user" -password "pass"
```

## Usage

### Basic Usage

```bash
# Using config file (default: .env)
./caldav2markdown

# Using custom config file
./caldav2markdown -config myconfig.env

# Using command line flags
./caldav2markdown -url "https://example.com/cal/" -username "user" -password "pass" -output "./my-events"
```

### Test Connection

```bash
./caldav2markdown -test
```

### Command Line Options

- `-url`: CalDAV server URL
- `-username`: CalDAV username
- `-password`: CalDAV password
- `-output`: Output directory for markdown files (default: ./events)
- `-config`: Configuration file path (default: .env)
- `-test`: Test connection only, don't fetch events

## Output Format

Each event is saved as a markdown file with the format `YYYY-MM-DD_Event-Title.md`:

```markdown
# Meeting with Team

**Start:** 2024-01-15 09:00
**End:** 2024-01-15 10:00

**Location:** Conference Room A

## Description

Weekly team sync to discuss project progress and upcoming milestones.
```

## CalDAV Server Compatibility

This tool should work with any CalDAV-compliant server, including:

- NextCloud
- ownCloud
- Apple Calendar Server
- Google Calendar (CalDAV interface)
- Microsoft Exchange (with CalDAV enabled)
