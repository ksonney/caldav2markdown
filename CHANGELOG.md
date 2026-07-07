# Changelog

All notable changes to caldav2markdown are documented here.

## [Unreleased]

### Added
- **Rescan / regenerate markdown from database**: New `-rescan` CLI flag and `RESCAN=true` config option (YAML: `rescan: true`) to load all events and todos from the database and regenerate all markdown files, bypassing the "only new/changed items" filter used in normal database-backed runs. Useful when changing formatting options (hashtags, frontmatter, etc.) without re-fetching from the calendar server. Acts as a named alias for the existing `-from-database` flag.

### Fixed
- **ICS parsing of non-conformant calendars**: Some servers (e.g. SOGo) place calendar-level properties like `X-WR-CALNAME` after all component blocks (`VEVENT`, `VTIMEZONE`, etc.), violating RFC 5545. The `golang-ical` parser rejects such files with "malformed calendar; expected begin or end". Added `normalizeICSContent()` to pre-process ICS content before parsing, moving any misplaced calendar-level properties back to their correct position. Also normalizes CRLF line endings. Affects both local and remote ICS sources.

## [2026-03-22]

### Added
- **Customizable Daily Note path**: New configuration option to customize the output path for daily note files.

## [2025-11-24]

### Added
- **Obsidian Life Manager (OML) directory structure support**: Daily files can now be organized in the `Daily/YYYY/MM - Month Name/YYYY-MM-DD.md` and weekly files in `Weekly/YYYY/YYYY-Www.md` layout, compatible with the Obsidian Life Manager vault system. Controlled by `OBSIDIAN_LIFE_MANAGER` config option or `-obsidian-life-manager` CLI flag.
- All-day events and timed events/tasks now included in markdown output.

### Fixed
- Spelling corrections.

## [2025-11-06]

### Improved
- Org mode handler improvements and refinements.

## [2025-11-04]

### Added
- **Emacs diary output format**: New `OUTPUT_FORMAT=diary` option generates traditional Emacs calendar diary files, compatible with GNU Emacs `M-x calendar` and `M-x diary`. Single file mode is automatically enabled. Supports timed events, all-day events, multi-day events, tasks, completion markers, and UID tracking.
- Comprehensive Org mode documentation and configuration examples added to README/CLAUDE.md.

## [2025-10-20]

### Added
- **Single file output flags**: New `-single-file` and `-single-file-name` CLI flags for consolidated single-file output mode.

## [2025-10-01]

### Added
- **SQLite database storage**: Optional persistent event/todo storage with smart change tracking. Tracks new, updated, and unchanged items across runs. Enables offline markdown regeneration from the database without re-fetching calendars. Controlled by `USE_DATABASE` config option or `-use-database` CLI flag.

### Fixed
- Markdown deduplication and UID comment handling.
- Task and event updates in markdown when database detects changes.

## [2025-09-29]

### Added
- **Ignore declined events**: New `IGNORE_DECLINED` option to automatically skip events with `STATUS=CANCELLED` or `PARTSTAT=DECLINED`.
- **Obsidian Tasks emoji support**: `USE_OBSIDIAN_EMOJIS` option adds 🛫 for start times and ✅ for end times.

### Fixed
- Empty hashtags on events and tasks.
- Timezone handling improvements including TZID parameter support and common Microsoft Exchange timezone mappings.

### Improved
- Hashtag and frontmatter tag generation.
- YAML config file format support with auto-detection.
- XDG base directory support for config paths.

## [2025-09-26]

### Added
- **YAML configuration format**: Structured YAML config files as an alternative to `.env` files, with auto-detection based on file extension or content.
- Multi-calendar detection fixes and improved calendar collection discovery.

## [2025-09-25]

### Added
- **Obsidian Tasks preset**: `OBSIDIAN_TASKS=true` enables a bundle of settings for Obsidian compatibility (checkboxes, emojis, hashtags, frontmatter, calendar tags).
- **Events as tasks**: Option to render calendar events with checkbox/task formatting.
- **ICS file and URL support**: Local `.ics` files and remote HTTP/HTTPS ICS URLs as calendar sources, with support for no auth, basic auth, bearer tokens, and custom headers.
- **Recurring task (VTODO) expansion**: Full RRULE support for recurring tasks matching VEVENT capabilities.

### Fixed
- Recurring events and tasks with start dates outside the filter range are now correctly included if they have instances within the range (expand-then-filter approach).
- Various bugfixes for ICS file handling.

## [2025-09-24]

### Added
- **Google OAuth 2.0**: Full OAuth integration for Google Calendar CalDAV access with automatic token management, refresh, and browser-based authorization flow.
- **Multi-calendar support**: Automatic discovery of all calendar collections via PROPFIND requests with include/exclude filtering.
- **Server-side filtering**: CalDAV REPORT queries with time-range filters (RFC 4791) to reduce network traffic.
- **Org mode output format**: Full Emacs Org mode format support (`OUTPUT_FORMAT=org`) with SCHEDULED/DEADLINE timestamps, properties drawers, TODO/DONE states, priorities, and tags.
- **Org-diary output format**: Hybrid format combining Org mode structure with Emacs diary sexp expressions (`OUTPUT_FORMAT=org-diary`).
- **Weekly file output mode**: Events grouped by ISO week (`WEEKLY_FILE_OUTPUT=true`) instead of daily files.
- **Calendar name display and aliases**: Automatic calendar name extraction with configurable alias mapping (`CALENDAR_ALIASES`).
- **Calendar alias hashtags/tags**: Auto-generated hashtags or Org tags from calendar names (`USE_CALENDAR_TAGS`).
- **Past event auto-completion**: Automatic `[x]` (Markdown) or `DONE` (Org) marking for past events when `EVENT_CHECKBOXES` is enabled.
- **Smart file merging**: Existing daily/weekly files are updated rather than overwritten, preserving manual edits and custom content.
- **YAML frontmatter**: Optional structured metadata in markdown files (`USE_FRONTMATTER`).
- **Makefile**: Build, test, and cross-compilation targets.
- **Progress indicators**: Real-time progress reporting during CalDAV fetching and file generation.

### Fixed
- Events with poorly formatted or unparseable dates are now preserved rather than filtered out.
- UTC and non-UTC timezone handling for event start/end times.
- Events outside UTC now handled correctly; all events combined into single file per day.

### Initial release (2025-09-24)
- CalDAV to Markdown conversion with VEVENT and VTODO support.
- Configurable date range filtering.
- RRULE recurring event expansion.
- UID-based deduplication across calendars.
- Daily aggregated output files in `YYYY/MM/YYYY-MM-DD.md` format.
- Optional YAML frontmatter, hashtags, due date emoji, and description inclusion.
