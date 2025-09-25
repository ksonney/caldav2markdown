# caldav2markdown

A powerful CalDAV to Markdown converter that transforms your calendar events and tasks into organized, searchable markdown files. Perfect for integrating calendar data with static site generators, note-taking apps, or personal knowledge management systems.

## 🆕 Recent Updates

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
- **Secure Token Storage**: OAuth tokens stored securely with automatic refresh

### 🚀 Performance & Scalability
- **Server-side Filtering**: CalDAV REPORT queries for efficient data retrieval
- **Multi-calendar Support**: Discover and process multiple calendars automatically
- **Smart Deduplication**: Global UID-based deduplication across all calendars
- **Parallel Processing**: Concurrent calendar processing with error handling

### 📁 Output & Organization
- **Daily Aggregation**: Events and tasks organized in `YYYY-MM-DD.md` files
- **YAML Frontmatter**: Optional structured metadata for static site generators
- **Smart File Merging**: Preserves manual edits while adding new calendar data
- **Flexible Formatting**: Customizable emoji, hashtags, display options, optional description exclusion, and event checkboxes
- **Directory Structure**: Organized in `YYYY/MM/` hierarchy for easy navigation

### 📅 Calendar Features
- **Event Processing**: Full support for recurring events with RRULE expansion
- **Smart Recurring Processing**: Recurring events/tasks are expanded first, then filtered by date range, ensuring items with start dates outside the filter range are included if they have instances within the range
- **Task Management**: Todo extraction with due dates, priorities, and status, including recurring task support
- **Date/Time Handling**: Support for UTC, local time, and all-day events
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

# Performance Options
USE_SERVER_SIDE_FILTERING=true
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
USE_FRONTMATTER=true
IGNORE_DESCRIPTIONS=false
EVENT_CHECKBOXES=false

# Date Filtering
START_DATE=2024-01-01
END_DATE=2025-12-31

# Performance and Multi-Calendar
USE_SERVER_SIDE_FILTERING=true
DISCOVER_CALENDARS=true
INCLUDE_CALENDARS=Work,Personal
EXCLUDE_CALENDARS=Archive,Spam
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
# Using config file (default: .env)
bin/caldav2markdown

# Using custom config file
bin/caldav2markdown -config myconfig.env

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

# Test connection to server
bin/caldav2markdown -url "https://your-server.com/" -username "user" -password "pass" -test

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
- `-output`: Output directory for markdown files (default: ./events)
- `-config`: Configuration file path (default: .env)
- `-emoji`: Use 📅 emoji for due dates in tasks
- `-hashtags`: Add #event and #task hashtags
- `-frontmatter`: Add YAML frontmatter to markdown files
- `-ignore-descriptions`: Ignore event and task descriptions in output
- `-event-checkboxes`: Add checkboxes to events for task-like formatting

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
  -server-side-filtering \
  -discover-calendars \
  -include-calendars "Work,Personal" \
  -exclude-calendars "Archive,Test" \
  -test
```

## Output Format

### Events

Events are organized in a `YYYY/MM/` directory structure. All events for a single day are saved in a single file named `YYYY-MM-DD.md` as a markdown list.

For events with zero or invalid dates, files are saved in `0001/01/` directory.

### Daily File Format

Each day's events and tasks are combined in a single `YYYY-MM-DD.md` file:

#### Standard Format

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
- [ ] **Team Meeting** (09:00 - 10:00) @ Conference Room A #event
- [ ] **Project Review** (14:00 - 15:30) @ Room B #event
```

Event checkboxes are useful for:
- **Task-like Event Tracking**: Treat events as actionable items that can be checked off
- **Meeting Attendance**: Track which meetings you've attended
- **Event Completion**: Mark events as done in your workflow
- **Unified Format**: Maintain consistency with task checkboxes

## Date/Time Format Support

The application supports multiple iCalendar date/time formats:

- **UTC Time with Z suffix**: `20240924T143000Z` (September 24, 2024 at 14:30:00 UTC)
- **Local Time without Z suffix**: `20240924T143000` (September 24, 2024 at 14:30:00 local time)
- **Date Only**: `20240924` (September 24, 2024 - all-day event)

This enhancement ensures compatibility with various CalDAV servers that may format date/time values differently.

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
# Generate for Obsidian vault
bin/caldav2markdown -emoji -hashtags -output /path/to/obsidian/vault/Calendar/

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
