package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	"caldav2markdown/pkg/caldav"
	"caldav2markdown/pkg/config"
	"caldav2markdown/pkg/database"
	"caldav2markdown/pkg/diary"
	icsource "caldav2markdown/pkg/ics"
	"caldav2markdown/pkg/markdown"
	"caldav2markdown/pkg/org"
)

// SourcedEvent tracks an event with its source information
type SourcedEvent struct {
	Event           *ics.VEvent
	SourceName      string
	CalendarAliases map[string]string
}

// SourcedTodo tracks a todo with its source information
type SourcedTodo struct {
	Todo            *ics.VTodo
	SourceName      string
	CalendarAliases map[string]string
}

func main() {
	// Get default config path with smart detection
	defaultConfigPath := ".env" // Fallback for backward compatibility
	if xdgConfigPath, err := config.GetDefaultConfigPath(); err == nil {
		// Priority order for existing configs:
		// 1. XDG YAML config (preferred)
		// 2. XDG .env config (for users who already moved to XDG)
		// 3. Local .env file (backward compatibility)
		// 4. Use XDG YAML for new setups
		xdgEnvPath := filepath.Join(filepath.Dir(xdgConfigPath), "config.env")

		if _, err := os.Stat(xdgConfigPath); err == nil {
			// XDG YAML config exists - use it
			defaultConfigPath = xdgConfigPath
		} else if _, err := os.Stat(xdgEnvPath); err == nil {
			// XDG .env config exists - use it
			defaultConfigPath = xdgEnvPath
		} else if _, err := os.Stat(".env"); err == nil {
			// Local .env exists - use it for backward compatibility
			defaultConfigPath = ".env"
		} else {
			// No existing config - use XDG YAML for new setups
			defaultConfigPath = xdgConfigPath
		}
	}

	var (
		// Legacy CalDAV flags
		url                    = flag.String("url", "", "CalDAV server URL")
		username               = flag.String("username", "", "CalDAV username")
		password               = flag.String("password", "", "CalDAV password")
		outputDir              = flag.String("output", "./events", "Output directory for markdown files")
		configFile             = flag.String("config", defaultConfigPath, "Configuration file path")
		testConn               = flag.Bool("test", false, "Test connection only (CalDAV mode)")
		startDate              = flag.String("start", "", "Start date for events (YYYY-MM-DD)")
		endDate                = flag.String("end", "", "End date for events (YYYY-MM-DD)")
		useDueDateEmoji        = flag.Bool("emoji", false, "Use 📅 emoji for due dates in tasks")
		useHashtags            = flag.Bool("hashtags", false, "Add #event and #task hashtags")
		useFrontmatter         = flag.Bool("frontmatter", false, "Add YAML frontmatter to markdown files")
		ignoreDescriptions     = flag.Bool("ignore-descriptions", false, "Ignore event and task descriptions in output")
		ignoreDeclined         = flag.Bool("ignore-declined", false, "Ignore declined events (STATUS=CANCELLED or PARTSTAT=DECLINED)")
		eventCheckboxes        = flag.Bool("event-checkboxes", false, "Add checkboxes to events for task-like formatting")
		obsidianTasks          = flag.Bool("obsidian-tasks", false, "Enable Obsidian tasks preset (event checkboxes, ignore descriptions, frontmatter, emojis, hashtags)")
		useObsidianEmojis      = flag.Bool("obsidian-emojis", false, "Use Obsidian Tasks emoji format for start/end times (🛫 for start, ✅ for end)")
		useCalendarTags        = flag.Bool("calendar-tags", false, "Add calendar name tags to events and tasks")
		singleFileOutput       = flag.Bool("single-file", false, "Generate a single file instead of separate daily files")
		singleFileName         = flag.String("single-file-name", "calendar.md", "Name of the single output file when using -single-file")
		weeklyFileOutput       = flag.Bool("weekly-file", false, "Generate weekly files instead of daily files (Monday-Sunday, ISO week numbers)")
		obsidianLifeManager    = flag.Bool("obsidian-life-manager", false, "Use Obsidian Life Manager directory structure (Daily/YYYY/MM - Month/)")
		outputFormat           = flag.String("output-format", "markdown", "Output format: markdown, org, org-diary, or diary")
		useServerSideFiltering = flag.Bool("server-side-filtering", false, "Use CalDAV server-side filtering (faster for large calendars)")
		discoverCalendars      = flag.Bool("discover-calendars", false, "Discover and process all calendars on the server")
		includeCalendars       = flag.String("include-calendars", "", "Comma-separated list of calendar names/URLs to include")
		excludeCalendars       = flag.String("exclude-calendars", "", "Comma-separated list of calendar names/URLs to exclude")
		listCalendars          = flag.Bool("list-calendars", false, "List available calendars and exit (CalDAV mode)")
		useOAuth               = flag.Bool("oauth", false, "Use OAuth 2.0 authentication for Google Calendar")
		clientID               = flag.String("client-id", "", "Google OAuth Client ID")
		clientSecret           = flag.String("client-secret", "", "Google OAuth Client Secret")
		// Proxy flags
		proxyURL               = flag.String("proxy-url", "", "Proxy server URL (e.g., http://proxy.example.com:8080)")
		proxyUsername          = flag.String("proxy-username", "", "Proxy username for authentication")
		proxyPassword          = flag.String("proxy-password", "", "Proxy password for authentication")
		// New ICS flags
		sourceMode             = flag.String("source-mode", "", "Source mode: caldav or ics (default: caldav)")
		icsPath                = flag.String("ics-path", "", "Path to local ICS file")
		icsURL                 = flag.String("ics-url", "", "URL to remote ICS file")
		icsAuth                = flag.String("ics-auth", "none", "Auth method for remote ICS: none, basic, bearer, header")
		icsUsername            = flag.String("ics-username", "", "Username for ICS basic auth")
		icsPassword            = flag.String("ics-password", "", "Password for ICS basic auth")
		icsToken               = flag.String("ics-token", "", "Token for ICS bearer auth")
		// YAML conversion flags
		convertToYAML          = flag.String("convert-to-yaml", "", "Convert env config file to YAML format and save to specified path")
		exportYAML             = flag.String("export-yaml", "", "Export current configuration to YAML file")
		// Database flags
		useDatabase            = flag.Bool("use-database", false, "Enable SQLite database for deduplication and tracking")
		databasePath           = flag.String("database-path", "", "Path to SQLite database file (default: alongside config file)")
		dbStats                = flag.Bool("db-stats", false, "Show database statistics and exit")
		dbClear                = flag.Bool("db-clear", false, "Clear all data from database and exit")
		fromDatabase           = flag.Bool("from-database", false, "Generate markdown from database instead of fetching from calendars")
	)
	flag.Parse()

	var cfg *config.Config

	if _, err := os.Stat(*configFile); err == nil {
		fmt.Printf("Loading configuration from %s\n", *configFile)
		cfg, err = config.LoadConfig(*configFile)
		if err != nil {
			fmt.Printf("Warning: failed to load config file: %v\n", err)
			cfg = &config.Config{}
		}
	} else {
		cfg = &config.Config{}
	}

	// Handle YAML conversion flags
	if *convertToYAML != "" {
		fmt.Printf("Converting env config to YAML format...\n")
		err := config.ConvertEnvToYAML(*configFile, *convertToYAML)
		if err != nil {
			fmt.Printf("Error converting to YAML: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully converted %s to %s\n", *configFile, *convertToYAML)
		return
	}

	if *exportYAML != "" {
		fmt.Printf("Exporting current configuration to YAML...\n")
		err := cfg.SaveToYAMLFile(*exportYAML)
		if err != nil {
			fmt.Printf("Error exporting to YAML: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully exported configuration to %s\n", *exportYAML)
		return
	}

	if *url != "" {
		cfg.URL = *url
	}
	if *username != "" {
		cfg.Username = *username
	}
	if *password != "" {
		cfg.Password = *password
	}
	if *outputDir != "./events" {
		cfg.Output = *outputDir
	}
	if cfg.Output == "" {
		cfg.Output = "./events"
	}
	if *useDueDateEmoji {
		cfg.UseDueDateEmoji = true
	}
	if *useHashtags {
		cfg.UseHashtags = true
	}
	if *useFrontmatter {
		cfg.UseFrontmatter = true
	}
	if *ignoreDescriptions {
		cfg.IgnoreDescriptions = true
	}
	if *ignoreDeclined {
		cfg.IgnoreDeclined = true
	}
	if *eventCheckboxes {
		cfg.EventCheckboxes = true
	}
	if *obsidianTasks {
		cfg.ObsidianTasks = true
	}
	if *useObsidianEmojis {
		cfg.UseObsidianEmojis = true
	}
	if *useCalendarTags {
		cfg.UseCalendarTags = true
	}
	if *singleFileOutput {
		cfg.SingleFileOutput = true
	}
	if *singleFileName != "calendar.md" {
		cfg.SingleFileName = *singleFileName
	}
	if cfg.SingleFileName == "" {
		cfg.SingleFileName = "calendar.md"
	}
	if *weeklyFileOutput {
		cfg.WeeklyFileOutput = true
	}
	if *obsidianLifeManager {
		cfg.ObsidianLifeManager = true
	}
	if *outputFormat != "markdown" {
		switch strings.ToLower(*outputFormat) {
		case "org":
			cfg.OutputFormat = "org"
		case "markdown", "md":
			cfg.OutputFormat = "markdown"
		case "diary":
			cfg.OutputFormat = "diary"
		case "org-diary", "orgdiary":
			cfg.OutputFormat = "org-diary"
		default:
			fmt.Printf("Invalid output format: %s (expected: markdown, org, org-diary, or diary)\n", *outputFormat)
			os.Exit(1)
		}
		// Re-normalize to apply format-specific defaults (like single-file for diary)
		cfg.NormalizeOutputFormat()
	}
	if *useServerSideFiltering {
		cfg.UseServerSideFiltering = true
	}
	if *discoverCalendars {
		cfg.DiscoverCalendars = true
	}
	if *includeCalendars != "" {
		cfg.IncludeCalendars = strings.Split(*includeCalendars, ",")
		for i, cal := range cfg.IncludeCalendars {
			cfg.IncludeCalendars[i] = strings.TrimSpace(cal)
		}
	}
	if *excludeCalendars != "" {
		cfg.ExcludeCalendars = strings.Split(*excludeCalendars, ",")
		for i, cal := range cfg.ExcludeCalendars {
			cfg.ExcludeCalendars[i] = strings.TrimSpace(cal)
		}
	}
	if *useOAuth {
		cfg.UseOAuth = true
	}
	if *clientID != "" {
		cfg.ClientID = *clientID
	}
	if *clientSecret != "" {
		cfg.ClientSecret = *clientSecret
	}
	if *proxyURL != "" {
		cfg.ProxyURL = *proxyURL
	}
	if *proxyUsername != "" {
		cfg.ProxyUsername = *proxyUsername
	}
	if *proxyPassword != "" {
		cfg.ProxyPassword = *proxyPassword
	}

	// Handle database flags
	if *useDatabase {
		cfg.UseDatabase = true
	}
	if *databasePath != "" {
		cfg.DatabasePath = *databasePath
	}
	// Set default database path if not specified but database is enabled
	if cfg.UseDatabase && cfg.DatabasePath == "" {
		// Place database alongside config file
		configDir := filepath.Dir(*configFile)
		cfg.DatabasePath = filepath.Join(configDir, "caldav2markdown.db")
	}

	// Handle ICS flags
	if *sourceMode != "" {
		switch strings.ToLower(*sourceMode) {
		case "caldav":
			cfg.SourceMode = config.SourceModeCalDAV
		case "ics":
			cfg.SourceMode = config.SourceModeICS
		default:
			fmt.Printf("Invalid source mode: %s (expected: caldav or ics)\n", *sourceMode)
			os.Exit(1)
		}
	}
	if *icsPath != "" {
		cfg.ICSPath = *icsPath
		cfg.SourceMode = config.SourceModeICS // Auto-switch to ICS mode
	}
	if *icsURL != "" {
		cfg.ICSURL = *icsURL
		cfg.SourceMode = config.SourceModeICS // Auto-switch to ICS mode
	}
	if *icsAuth != "" {
		switch strings.ToLower(*icsAuth) {
		case "none":
			cfg.ICSAuth = icsource.AuthNone
		case "basic":
			cfg.ICSAuth = icsource.AuthBasic
		case "bearer":
			cfg.ICSAuth = icsource.AuthBearer
		case "header":
			cfg.ICSAuth = icsource.AuthHeader
		default:
			fmt.Printf("Invalid ICS auth method: %s (expected: none, basic, bearer, header)\n", *icsAuth)
			os.Exit(1)
		}
	}
	if *icsUsername != "" {
		cfg.ICSUsername = *icsUsername
	}
	if *icsPassword != "" {
		cfg.ICSPassword = *icsPassword
	}
	if *icsToken != "" {
		cfg.ICSToken = *icsToken
	}

	// Handle date flags
	if *startDate != "" {
		if start, err := time.Parse("2006-01-02", *startDate); err == nil {
			cfg.StartDate = start
		} else {
			fmt.Printf("Invalid start date format: %v (expected YYYY-MM-DD)\n", err)
			os.Exit(1)
		}
	}
	if *endDate != "" {
		if end, err := time.Parse("2006-01-02", *endDate); err == nil {
			cfg.EndDate = end
		} else {
			fmt.Printf("Invalid end date format: %v (expected YYYY-MM-DD)\n", err)
			os.Exit(1)
		}
	}

	// Set default dates if not configured
	if cfg.StartDate.IsZero() {
		cfg.StartDate = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if cfg.EndDate.IsZero() {
		cfg.EndDate = time.Now().AddDate(2, 0, 0)
	}

	// Apply Obsidian tasks preset if enabled (applies to both config file and CLI flags)
	cfg.ApplyObsidianTasksPreset()

	if err := cfg.Validate(); err != nil {
		fmt.Println("Usage: caldav2markdown [options]")
		fmt.Println("")
		fmt.Println("Source Modes:")
		fmt.Println("  -source-mode      Source mode: caldav or ics (default: caldav)")
		fmt.Println("")
		fmt.Println("CalDAV Mode Options:")
		fmt.Println("  -url              CalDAV server URL (required for CalDAV mode)")
		fmt.Println("  Authentication (choose one):")
		fmt.Println("    Basic Auth:")
		fmt.Println("      -username     CalDAV username")
		fmt.Println("      -password     CalDAV password")
		fmt.Println("    OAuth 2.0 (for Google Calendar):")
		fmt.Println("      -oauth        Enable OAuth 2.0 authentication")
		fmt.Println("      -client-id    Google OAuth Client ID")
		fmt.Println("      -client-secret Google OAuth Client Secret")
		fmt.Println("  CalDAV-specific options:")
		fmt.Println("    -test                   Test connection only, don't fetch events")
		fmt.Println("    -server-side-filtering  Use CalDAV server-side filtering (faster)")
		fmt.Println("    -discover-calendars     Discover and process all calendars")
		fmt.Println("    -include-calendars      Comma-separated list of calendar names to include")
		fmt.Println("    -exclude-calendars      Comma-separated list of calendar names to exclude")
		fmt.Println("    -list-calendars         List available calendars and exit")
		fmt.Println("")
		fmt.Println("Proxy Options (for all modes):")
		fmt.Println("  -proxy-url        Proxy server URL (e.g., http://proxy.example.com:8080)")
		fmt.Println("  -proxy-username   Proxy username for authentication")
		fmt.Println("  -proxy-password   Proxy password for authentication")
		fmt.Println("")
		fmt.Println("ICS Mode Options:")
		fmt.Println("  Source (choose one):")
		fmt.Println("    -ics-path         Path to local ICS file")
		fmt.Println("    -ics-url          URL to remote ICS file")
		fmt.Println("  Authentication for remote ICS:")
		fmt.Println("    -ics-auth         Auth method: none, basic, bearer, header (default: none)")
		fmt.Println("    -ics-username     Username for basic auth")
		fmt.Println("    -ics-password     Password for basic auth")
		fmt.Println("    -ics-token        Bearer token for bearer auth")
		fmt.Println("")
		fmt.Println("Database Options:")
		fmt.Println("  -use-database     Enable SQLite database for deduplication")
		fmt.Println("  -database-path    Path to SQLite database (default: alongside config file)")
		fmt.Println("  -from-database    Generate markdown from database instead of fetching calendars")
		fmt.Println("  -db-stats         Show database statistics and exit")
		fmt.Println("  -db-clear         Clear all database data and exit")
		fmt.Println("")
		fmt.Println("Common Options:")
		fmt.Println("  -output           Output directory for files (default: ./events)")
		fmt.Printf("  -config           Configuration file path (default: %s)\n", defaultConfigPath)
		fmt.Println("  -start            Start date for events (YYYY-MM-DD, default: 2000-01-01)")
		fmt.Println("  -end              End date for events (YYYY-MM-DD, default: 2 years from now)")
		fmt.Println("  -output-format    Output format: markdown, org, org-diary, or diary (default: markdown)")
		fmt.Println("  -emoji            Use 📅 emoji for due dates in tasks")
		fmt.Println("  -hashtags         Add #event and #task hashtags")
		fmt.Println("  -frontmatter      Add YAML frontmatter to markdown files")
		fmt.Println("  -ignore-descriptions  Ignore event and task descriptions in output")
		fmt.Println("  -event-checkboxes     Add checkboxes to events for task-like formatting")
		fmt.Println("  -obsidian-tasks   Enable Obsidian preset (event checkboxes + ignore descriptions + frontmatter + emojis + hashtags)")
		fmt.Println("  -single-file      Generate a single file instead of separate daily files")
		fmt.Println("  -single-file-name Name of the single output file (default: calendar.md or calendar.org)")
		fmt.Println("  -weekly-file      Generate weekly files instead of daily files (Monday-Sunday, ISO week numbers)")
		fmt.Println("")
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Initialize database if enabled
	var db *database.DB
	if cfg.UseDatabase {
		var err error
		db, err = database.Open(cfg.DatabasePath)
		if err != nil {
			fmt.Printf("Error opening database: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()
		fmt.Printf("Using database: %s\n", cfg.DatabasePath)
	}

	// Handle database management commands
	if *dbStats {
		if db == nil {
			fmt.Println("Database is not enabled. Use -use-database flag.")
			os.Exit(1)
		}
		stats, err := db.GetStats()
		if err != nil {
			fmt.Printf("Error getting database stats: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Database Statistics:")
		fmt.Printf("  Total events: %d\n", stats["events"])
		fmt.Printf("  Unique events: %d\n", stats["unique_events"])
		fmt.Printf("  Total todos: %d\n", stats["todos"])
		fmt.Printf("  Unique todos: %d\n", stats["unique_todos"])

		eventSources, err := db.CountEventsBySource()
		if err == nil && len(eventSources) > 0 {
			fmt.Println("\n  Events by source:")
			for source, count := range eventSources {
				fmt.Printf("    %s: %d\n", source, count)
			}
		}

		todoSources, err := db.CountTodosBySource()
		if err == nil && len(todoSources) > 0 {
			fmt.Println("\n  Todos by source:")
			for source, count := range todoSources {
				fmt.Printf("    %s: %d\n", source, count)
			}
		}
		return
	}

	if *dbClear {
		if db == nil {
			fmt.Println("Database is not enabled. Use -use-database flag.")
			os.Exit(1)
		}
		fmt.Print("Are you sure you want to clear all database data? (yes/no): ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) == "yes" {
			if err := db.Clear(); err != nil {
				fmt.Printf("Error clearing database: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Database cleared successfully.")
		} else {
			fmt.Println("Operation cancelled.")
		}
		return
	}

	// Create progress callback function
	progressCallback := func(message string, current, total int) {
		if total > 0 {
			percent := (current * 100) / total
			fmt.Printf("\r[%3d%%] %s (%d/%d)", percent, message, current, total)
		} else {
			fmt.Printf("\r%s", message)
		}
	}

	// Process sources (either multi-source or legacy single-source)
	var sourcedEvents []SourcedEvent
	var sourcedTodos []SourcedTodo
	var duplicatesFound int

	// Check if we should generate from database instead of fetching calendars
	if *fromDatabase {
		if db == nil {
			fmt.Println("Error: -from-database requires -use-database to be enabled")
			os.Exit(1)
		}
		fmt.Println("Generating markdown from database...")
		if err := generateFromDatabase(db, cfg, &sourcedEvents, &sourcedTodos); err != nil {
			fmt.Printf("Error generating from database: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Loaded %d events and %d todos from database\n", len(sourcedEvents), len(sourcedTodos))
	} else if cfg.HasMultipleSources() {
		// Multi-source processing
		if err := processMultipleSources(cfg, progressCallback, *listCalendars, *testConn, &sourcedEvents, &sourcedTodos, &duplicatesFound); err != nil {
			fmt.Printf("\nError: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Legacy single-source processing
		var events []*ics.VEvent
		var todos []*ics.VTodo
		switch cfg.SourceMode {
		case config.SourceModeCalDAV:
			if err := processCalDAVMode(cfg, progressCallback, *listCalendars, *testConn, &events, &todos, &duplicatesFound); err != nil {
				fmt.Printf("\nError: %v\n", err)
				os.Exit(1)
			}
		case config.SourceModeICS:
			if err := processICSMode(cfg, progressCallback, &events, &todos, &duplicatesFound); err != nil {
				fmt.Printf("\nError: %v\n", err)
				os.Exit(1)
			}
		default:
			fmt.Printf("Unsupported source mode: %s\n", cfg.SourceMode)
			os.Exit(1)
		}

		// Convert legacy events and todos to sourced format
		for _, event := range events {
			sourcedEvents = append(sourcedEvents, SourcedEvent{
				Event:           event,
				SourceName:      cfg.URL, // Use URL as source name for legacy mode
				CalendarAliases: cfg.CalendarAliases,
			})
		}
		for _, todo := range todos {
			sourcedTodos = append(sourcedTodos, SourcedTodo{
				Todo:            todo,
				SourceName:      cfg.URL, // Use URL as source name for legacy mode
				CalendarAliases: cfg.CalendarAliases,
			})
		}
	}

	if len(sourcedEvents) == 0 && len(sourcedTodos) == 0 {
		fmt.Println("No events or tasks found.")
		return
	}

	if duplicatesFound > 0 {
		fmt.Printf("Found %d duplicate events (skipped)\n", duplicatesFound)
	}

	if err := os.MkdirAll(cfg.Output, 0755); err != nil {
		fmt.Printf("Failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	// Store in database and filter to only new/changed items if database is enabled
	var dbNewEvents, dbNewTodos, dbUpdatedEvents, dbUpdatedTodos int
	if db != nil && !*fromDatabase {
		fmt.Println("Storing events and todos in database...")
		var newOrChangedEventUIDs, newOrChangedTodoUIDs map[string]bool
		dbNewEvents, dbUpdatedEvents, dbNewTodos, dbUpdatedTodos, newOrChangedEventUIDs, newOrChangedTodoUIDs = storeInDatabaseAndTrackChanges(db, sourcedEvents, sourcedTodos)

		// Filter to only new or changed items to prevent duplicate markdown
		sourcedEvents, sourcedTodos = filterNewOrChanged(sourcedEvents, sourcedTodos, newOrChangedEventUIDs, newOrChangedTodoUIDs)

		fmt.Printf("Database: %d new events, %d updated events, %d new todos, %d updated todos\n",
			dbNewEvents, dbUpdatedEvents, dbNewTodos, dbUpdatedTodos)
		if len(newOrChangedEventUIDs) > 0 || len(newOrChangedTodoUIDs) > 0 {
			fmt.Printf("Generating markdown for %d new/changed events and %d new/changed todos\n",
				len(newOrChangedEventUIDs), len(newOrChangedTodoUIDs))
		} else {
			fmt.Println("No new or changed items - skipping markdown generation")
			fmt.Println("Successfully processed 0 events and 0 tasks (all unchanged)")
			return
		}
	}

	// Report what was found
	if len(sourcedEvents) > 0 && len(sourcedTodos) > 0 {
		fmt.Printf("Found %d events and %d tasks, converting to daily markdown files...\n", len(sourcedEvents), len(sourcedTodos))
	} else if len(sourcedEvents) > 0 {
		fmt.Printf("Found %d events, converting to daily markdown files...\n", len(sourcedEvents))
	} else {
		fmt.Printf("Found %d tasks, converting to daily markdown files...\n", len(sourcedTodos))
	}

	// Convert events to EventMarkdown format
	var eventMarkdowns []markdown.EventMarkdown
	if len(sourcedEvents) > 0 {
		fmt.Print("Converting events to markdown format...")
		declinedCount := 0
		for i, sourcedEvent := range sourcedEvents {
			// Check if we should skip declined events
			if cfg.IgnoreDeclined && isEventDeclined(sourcedEvent.Event) {
				declinedCount++
				continue
			}
			eventMarkdowns = append(eventMarkdowns, markdown.ConvertEventWithCalendar(sourcedEvent.Event, sourcedEvent.SourceName, cfg.CalendarAliases))
			if len(sourcedEvents) > 10 && (i+1)%10 == 0 {
				fmt.Printf("\rConverting events to markdown format... (%d/%d)", i+1, len(sourcedEvents))
			}
		}
		if declinedCount > 0 {
			fmt.Printf("\rConverted %d events to markdown format (skipped %d declined)\n", len(eventMarkdowns), declinedCount)
		} else {
			fmt.Printf("\rConverted %d events to markdown format\n", len(eventMarkdowns))
		}
	}

	// Convert todos to TodoMarkdown format
	var todoMarkdowns []markdown.TodoMarkdown
	if len(sourcedTodos) > 0 {
		fmt.Printf("Converting %d tasks to markdown format...\n", len(sourcedTodos))
		for _, sourcedTodo := range sourcedTodos {
			todoMarkdowns = append(todoMarkdowns, markdown.ConvertTodoWithCalendar(sourcedTodo.Todo, sourcedTodo.SourceName, cfg.CalendarAliases))
		}
	}

	// Generate output files based on format - either single file or daily files
	if cfg.OutputFormat == "org" {
		// Org mode output
		if cfg.SingleFileOutput {
			fmt.Println("Generating single Org file output...")
			outputPath := filepath.Join(cfg.Output, cfg.SingleFileName)
			if !strings.HasSuffix(outputPath, ".org") {
				outputPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".org"
			}
			if err := org.GenerateSingleOrgFile(outputPath, eventMarkdowns, todoMarkdowns, cfg.UseDueDateEmoji, cfg.UseHashtags, cfg.IgnoreDescriptions, cfg.EventCheckboxes, cfg.UseCalendarTags, cfg.UseObsidianEmojis, progressCallback); err != nil {
				fmt.Printf("\nFailed to generate single Org file: %v\n", err)
				os.Exit(1)
			}
			fmt.Println() // New line after progress
			fmt.Printf("Successfully created %s with %d events and %d tasks\n", outputPath, len(sourcedEvents), len(sourcedTodos))
			return
		}

		// Generate daily Org files
		fmt.Println("Generating daily Org files...")
		if err := org.GenerateDailyOrgFiles(cfg.Output, eventMarkdowns, todoMarkdowns, cfg.UseDueDateEmoji, cfg.UseHashtags, cfg.IgnoreDescriptions, cfg.EventCheckboxes, cfg.UseCalendarTags, cfg.UseObsidianEmojis, progressCallback); err != nil {
			fmt.Printf("\nFailed to generate daily Org files: %v\n", err)
			os.Exit(1)
		}
		fmt.Println() // New line after progress
	} else if cfg.OutputFormat == "org-diary" {
		// Org mode with diary sexp output
		if cfg.SingleFileOutput {
			fmt.Println("Generating single Org file output (org-diary format)...")
			outputPath := filepath.Join(cfg.Output, cfg.SingleFileName)
			if !strings.HasSuffix(outputPath, ".org") {
				outputPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".org"
			}
			if err := org.GenerateSingleOrgDiaryFile(outputPath, eventMarkdowns, todoMarkdowns, cfg.UseDueDateEmoji, cfg.UseHashtags, cfg.IgnoreDescriptions, cfg.EventCheckboxes, cfg.UseCalendarTags, cfg.UseObsidianEmojis, progressCallback); err != nil {
				fmt.Printf("\nFailed to generate single Org file (org-diary format): %v\n", err)
				os.Exit(1)
			}
			fmt.Println() // New line after progress
			fmt.Printf("Successfully created %s with %d events and %d tasks (org-diary format)\n", outputPath, len(sourcedEvents), len(sourcedTodos))
			return
		}

		// Generate daily Org diary files
		fmt.Println("Generating daily Org files (org-diary format)...")
		if err := org.GenerateDailyOrgDiaryFiles(cfg.Output, eventMarkdowns, todoMarkdowns, cfg.UseDueDateEmoji, cfg.UseHashtags, cfg.IgnoreDescriptions, cfg.EventCheckboxes, cfg.UseCalendarTags, cfg.UseObsidianEmojis, progressCallback); err != nil {
			fmt.Printf("\nFailed to generate daily Org files (org-diary format): %v\n", err)
			os.Exit(1)
		}
		fmt.Println() // New line after progress
	} else if cfg.OutputFormat == "diary" {
		// Emacs diary output
		if cfg.SingleFileOutput {
			fmt.Println("Generating single diary file output...")
			outputPath := filepath.Join(cfg.Output, cfg.SingleFileName)
			// Don't add suffix if filename already ends with "diary" or has no extension
			// Emacs diary files traditionally have no extension
			if err := diary.GenerateSingleDiaryFile(outputPath, eventMarkdowns, todoMarkdowns, cfg.UseHashtags, cfg.IgnoreDescriptions, cfg.UseCalendarTags, cfg.EventCheckboxes, progressCallback); err != nil {
				fmt.Printf("\nFailed to generate single diary file: %v\n", err)
				os.Exit(1)
			}
			fmt.Println() // New line after progress
			fmt.Printf("Successfully created %s with %d events and %d tasks\n", outputPath, len(sourcedEvents), len(sourcedTodos))
			return
		}

		// Generate monthly diary files (Emacs diary convention)
		fmt.Println("Generating monthly diary files...")
		if err := diary.GenerateDailyDiaryFiles(cfg.Output, eventMarkdowns, todoMarkdowns, cfg.UseHashtags, cfg.IgnoreDescriptions, cfg.UseCalendarTags, cfg.EventCheckboxes, progressCallback); err != nil {
			fmt.Printf("\nFailed to generate monthly diary files: %v\n", err)
			os.Exit(1)
		}
		fmt.Println() // New line after progress
	} else {
		// Markdown output (default)
		if cfg.SingleFileOutput {
			fmt.Println("Generating single file output...")
			outputPath := filepath.Join(cfg.Output, cfg.SingleFileName)
			if err := markdown.GenerateSingleFile(outputPath, eventMarkdowns, todoMarkdowns, cfg.UseDueDateEmoji, cfg.UseHashtags, cfg.UseFrontmatter, cfg.IgnoreDescriptions, cfg.EventCheckboxes, cfg.UseCalendarTags, cfg.UseObsidianEmojis, progressCallback); err != nil {
				fmt.Printf("\nFailed to generate single file: %v\n", err)
				os.Exit(1)
			}
			fmt.Println() // New line after progress
			fmt.Printf("Successfully created %s with %d events and %d tasks\n", outputPath, len(sourcedEvents), len(sourcedTodos))
			return
		}

		// Generate aggregated markdown files (daily or weekly) with both events and tasks
		if cfg.WeeklyFileOutput {
			fmt.Println("Generating weekly files...")
			if err := markdown.GenerateWeeklyFilesWithAllOptions(cfg.Output, eventMarkdowns, todoMarkdowns, cfg.UseDueDateEmoji, cfg.UseHashtags, cfg.UseFrontmatter, cfg.IgnoreDescriptions, cfg.EventCheckboxes, cfg.UseCalendarTags, cfg.UseObsidianEmojis, cfg.ObsidianLifeManager, progressCallback); err != nil {
				fmt.Printf("\nFailed to generate weekly files: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Println("Generating daily files...")
			if err := markdown.GenerateDailyFilesWithAllOptions(cfg.Output, eventMarkdowns, todoMarkdowns, cfg.UseDueDateEmoji, cfg.UseHashtags, cfg.UseFrontmatter, cfg.IgnoreDescriptions, cfg.EventCheckboxes, cfg.UseCalendarTags, cfg.UseObsidianEmojis, cfg.ObsidianLifeManager, progressCallback); err != nil {
				fmt.Printf("\nFailed to generate daily files: %v\n", err)
				os.Exit(1)
			}
		}
		fmt.Println() // New line after progress
	}

	// Count unique dates/weeks for reporting
	if cfg.WeeklyFileOutput {
		weekCount := make(map[string]bool)
		for _, eventMd := range eventMarkdowns {
			var weekKey string
			if eventMd.StartTime.IsZero() {
				weekKey = "0001-01"
			} else {
				year, week := eventMd.StartTime.ISOWeek()
				weekKey = fmt.Sprintf("%04d-%02d", year, week)
			}
			weekCount[weekKey] = true
		}

		// Count task weeks for reporting as well
		for _, todoMd := range todoMarkdowns {
			var weekKey string
			if todoMd.DueDate.IsZero() {
				weekKey = "0001-01"
			} else {
				year, week := todoMd.DueDate.ISOWeek()
				weekKey = fmt.Sprintf("%04d-%02d", year, week)
			}
			weekCount[weekKey] = true
		}

		fmt.Printf("Successfully created %d weekly files for %d events and %d tasks\n", len(weekCount), len(sourcedEvents), len(sourcedTodos))
	} else {
		dateCount := make(map[string]bool)
		for _, eventMd := range eventMarkdowns {
			var dateKey string
			if eventMd.StartTime.IsZero() {
				dateKey = "0001-01-01"
			} else {
				dateKey = eventMd.StartTime.Format("2006-01-02")
			}
			dateCount[dateKey] = true
		}

		// Count task dates for reporting as well
		for _, todoMd := range todoMarkdowns {
			var dateKey string
			if todoMd.DueDate.IsZero() {
				dateKey = "0001-01-01"
			} else {
				dateKey = todoMd.DueDate.Format("2006-01-02")
			}
			dateCount[dateKey] = true
		}

		fmt.Printf("Successfully created %d daily files for %d events and %d tasks\n", len(dateCount), len(sourcedEvents), len(sourcedTodos))
	}
}

// processCalDAVMode handles CalDAV-specific processing
func processCalDAVMode(cfg *config.Config, progressCallback func(string, int, int), listCalendars, testConn bool, events *[]*ics.VEvent, todos *[]*ics.VTodo, duplicatesFound *int) error {
	caldavConfig := caldav.Config{
		URL:                    cfg.URL,
		Username:               cfg.Username,
		Password:               cfg.Password,
		StartDate:              cfg.StartDate,
		EndDate:                cfg.EndDate,
		UseOAuth:               cfg.UseOAuth,
		ClientID:               cfg.ClientID,
		ClientSecret:           cfg.ClientSecret,
		UseServerSideFiltering: cfg.UseServerSideFiltering,
		DiscoverCalendars:      cfg.DiscoverCalendars,
		IncludeCalendars:       cfg.IncludeCalendars,
		ExcludeCalendars:       cfg.ExcludeCalendars,
		TraceWebCalls:          cfg.TraceWebCalls,
		ProxyURL:               cfg.ProxyURL,
		ProxyUsername:          cfg.ProxyUsername,
		ProxyPassword:          cfg.ProxyPassword,
	}

	client := caldav.NewClient(caldavConfig)

	if listCalendars {
		fmt.Println("Discovering calendars...")
		calendars, err := client.DiscoverCalendars()
		if err != nil {
			return fmt.Errorf("failed to discover calendars: %w", err)
		}

		if len(calendars) == 0 {
			fmt.Println("No calendars found.")
			os.Exit(0)
		}

		fmt.Printf("Found %d calendar(s):\n", len(calendars))
		for i, calendar := range calendars {
			fmt.Printf("%d. %s\n", i+1, calendar.DisplayName)
			fmt.Printf("   URL: %s\n", calendar.URL)
			fmt.Printf("   Supported components: %v\n", calendar.Components)
			fmt.Println()
		}
		os.Exit(0)
	}

	if testConn {
		fmt.Println("Testing connection...")
		if err := client.TestConnection(); err != nil {
			return fmt.Errorf("connection test failed: %w", err)
		}
		fmt.Println("Connection successful!")
		os.Exit(0)
	}

	fmt.Println("Fetching events from CalDAV server...")
	result, err := client.GetEventsWithDeduplicationAndProgress(progressCallback)
	if err != nil {
		return fmt.Errorf("failed to fetch events: %w", err)
	}
	fmt.Println() // New line after progress

	*events = result.Events
	*todos = result.Todos
	*duplicatesFound = result.DuplicatesFound

	return nil
}

// processICSMode handles ICS file processing
func processICSMode(cfg *config.Config, progressCallback func(string, int, int), events *[]*ics.VEvent, todos *[]*ics.VTodo, duplicatesFound *int) error {
	// Convert config to ICS source
	source, err := cfg.ToICSSource()
	if err != nil {
		return fmt.Errorf("failed to configure ICS source: %w", err)
	}

	// Create processor with single source
	processor := icsource.NewProcessor(source, cfg.StartDate, cfg.EndDate)

	fmt.Printf("Processing ICS source: %s\n", source.Name)

	// Process the source
	ctx := context.Background()
	result, err := processor.ProcessSource(ctx, progressCallback)
	if err != nil {
		return fmt.Errorf("failed to process ICS sources: %w", err)
	}
	fmt.Println() // New line after progress

	// Print processing report
	if result.ProcessingReport != "" {
		fmt.Println(result.ProcessingReport)
	}

	*events = result.Events
	*todos = result.Todos
	*duplicatesFound = result.DuplicatesFound

	return nil
}

// processMultipleSources handles multi-source processing
func processMultipleSources(cfg *config.Config, progressCallback func(string, int, int), listCalendars, testConn bool, sourcedEvents *[]SourcedEvent, sourcedTodos *[]SourcedTodo, duplicatesFound *int) error {
	sources := cfg.GetSources()
	if len(sources) == 0 {
		return fmt.Errorf("no sources configured")
	}

	fmt.Printf("Processing %d calendar source(s)...\n", len(sources))

	var allSourcedEvents []SourcedEvent
	var allSourcedTodos []SourcedTodo
	var totalDuplicates int
	seenUIDs := make(map[string]bool)

	var totalSourceDuplicates int
	for i, source := range sources {
		fmt.Printf("\n--- Processing source %d/%d: %s (%s) ---\n", i+1, len(sources), source.Name, source.Type)

		var sourceEvents []*ics.VEvent
		var sourceTodos []*ics.VTodo
		var sourceDuplicates int

		switch source.Type {
		case "caldav":
			if err := processCalDAVSource(source.CalDAVSource, cfg, progressCallback, listCalendars, testConn, &sourceEvents, &sourceTodos, &sourceDuplicates); err != nil {
				fmt.Printf("Warning: Failed to process CalDAV source '%s': %v\n", source.Name, err)
				continue
			}
		case "ics":
			if err := processICSSource(source.ICSSource, cfg, progressCallback, &sourceEvents, &sourceTodos, &sourceDuplicates); err != nil {
				fmt.Printf("Warning: Failed to process ICS source '%s': %v\n", source.Name, err)
				continue
			}
		default:
			fmt.Printf("Warning: Unsupported source type '%s' for source '%s'\n", source.Type, source.Name)
			continue
		}

		// Apply global deduplication across all sources
		eventsAdded := 0
		todosAdded := 0

		// Deduplicate events globally and preserve source information
		for _, event := range sourceEvents {
			uid := getEventUID(event)
			if uid != "" && seenUIDs[uid] {
				totalDuplicates++
				continue
			}
			if uid != "" {
				seenUIDs[uid] = true
			}
			allSourcedEvents = append(allSourcedEvents, SourcedEvent{
				Event:           event,
				SourceName:      source.Name,
				CalendarAliases: cfg.CalendarAliases,
			})
			eventsAdded++
		}

		// Deduplicate todos globally and preserve source information
		for _, todo := range sourceTodos {
			uid := getTodoUID(todo)
			if uid != "" && seenUIDs[uid] {
				totalDuplicates++
				continue
			}
			if uid != "" {
				seenUIDs[uid] = true
			}
			allSourcedTodos = append(allSourcedTodos, SourcedTodo{
				Todo:            todo,
				SourceName:      source.Name,
				CalendarAliases: cfg.CalendarAliases,
			})
			todosAdded++
		}

		fmt.Printf("Source '%s' contributed %d events and %d tasks", source.Name, eventsAdded, todosAdded)
		if len(sourceEvents)-eventsAdded > 0 || len(sourceTodos)-todosAdded > 0 {
			fmt.Printf(" (%d duplicates skipped)", (len(sourceEvents)-eventsAdded)+(len(sourceTodos)-todosAdded))
		}
		fmt.Println()

		totalSourceDuplicates += sourceDuplicates
	}

	*sourcedEvents = allSourcedEvents
	*sourcedTodos = allSourcedTodos
	*duplicatesFound = totalDuplicates + totalSourceDuplicates

	if totalDuplicates > 0 {
		fmt.Printf("\nGlobal deduplication removed %d duplicate items across all sources\n", totalDuplicates)
	}

	fmt.Printf("\nTotal: %d events and %d tasks from %d sources\n", len(allSourcedEvents), len(allSourcedTodos), len(sources))

	return nil
}

// processCalDAVSource processes a single CalDAV source
func processCalDAVSource(source *config.CalDAVSource, cfg *config.Config, progressCallback func(string, int, int), listCalendars, testConn bool, events *[]*ics.VEvent, todos *[]*ics.VTodo, duplicatesFound *int) error {
	// Create CalDAV client configuration from source
	clientConfig := caldav.Config{
		URL:                    source.URL,
		Username:               source.Username,
		Password:               source.Password,
		StartDate:              cfg.StartDate,
		EndDate:                cfg.EndDate,
		UseOAuth:               source.UseOAuth,
		ClientID:               source.ClientID,
		ClientSecret:           source.ClientSecret,
		UseServerSideFiltering: source.UseServerSideFiltering,
		DiscoverCalendars:      source.DiscoverCalendars,
		IncludeCalendars:       source.IncludeCalendars,
		ExcludeCalendars:       source.ExcludeCalendars,
		TraceWebCalls:          cfg.TraceWebCalls,
		ProxyURL:               source.ProxyURL,
		ProxyUsername:          source.ProxyUsername,
		ProxyPassword:          source.ProxyPassword,
	}

	client := caldav.NewClient(clientConfig)

	// Handle test connection
	if testConn {
		fmt.Printf("Testing connection to CalDAV source '%s'...\n", source.Name)
		if err := client.TestConnection(); err != nil {
			return fmt.Errorf("connection test failed: %w", err)
		}
		fmt.Println("Connection test successful!")
		return nil
	}

	// Handle list calendars
	if listCalendars {
		fmt.Printf("Listing calendars for source '%s'...\n", source.Name)
		calendars, err := client.DiscoverCalendars()
		if err != nil {
			return fmt.Errorf("failed to discover calendars: %w", err)
		}

		if len(calendars) == 0 {
			fmt.Println("No calendars found")
		} else {
			fmt.Printf("Found %d calendar(s):\n", len(calendars))
			for i, cal := range calendars {
				fmt.Printf("  %d. %s (%s) - Components: %v\n", i+1, cal.DisplayName, cal.URL, cal.Components)
			}
		}
		return nil
	}

	// Fetch events and todos
	result, err := client.GetEventsWithDeduplicationAndProgress(progressCallback)
	if err != nil {
		return fmt.Errorf("failed to fetch calendar data: %w", err)
	}

	*events = result.Events
	*todos = result.Todos
	*duplicatesFound = result.DuplicatesFound

	return nil
}

// processICSSource processes a single ICS source
func processICSSource(source *icsource.Source, cfg *config.Config, progressCallback func(string, int, int), events *[]*ics.VEvent, todos *[]*ics.VTodo, duplicatesFound *int) error {
	// Create processor with single source
	processor := icsource.NewProcessor(*source, cfg.StartDate, cfg.EndDate)

	// Process the source
	ctx := context.Background()
	result, err := processor.ProcessSource(ctx, progressCallback)
	if err != nil {
		return fmt.Errorf("failed to process ICS source: %w", err)
	}

	*events = result.Events
	*todos = result.Todos
	*duplicatesFound = result.DuplicatesFound

	return nil
}

// Helper functions for UID extraction
func getEventUID(event *ics.VEvent) string {
	if uid := event.GetProperty(ics.ComponentPropertyUniqueId); uid != nil {
		return uid.Value
	}
	return ""
}

func getTodoUID(todo *ics.VTodo) string {
	if uid := todo.GetProperty(ics.ComponentPropertyUniqueId); uid != nil {
		return uid.Value
	}
	return ""
}

// isEventDeclined checks if an event has been declined by checking STATUS and PARTSTAT properties
func isEventDeclined(event *ics.VEvent) bool {
	// Check the overall event STATUS property
	if status := event.GetProperty(ics.ComponentPropertyStatus); status != nil {
		if strings.ToUpper(status.Value) == "CANCELLED" {
			return true
		}
	}

	// Check PARTSTAT (participation status) for ATTENDEE properties
	// This is the most reliable way to check if the user declined an event
	for _, attendee := range event.GetProperties("ATTENDEE") {
		if partstat, exists := attendee.ICalParameters["PARTSTAT"]; exists && len(partstat) > 0 {
			// Check if any attendee has DECLINED status
			if strings.ToUpper(partstat[0]) == "DECLINED" {
				return true
			}
		}
	}

	return false
}

func writeMarkdownFile(filename, content string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

// generateFromDatabase loads events and todos from the database and converts them to sourced format
func generateFromDatabase(db *database.DB, cfg *config.Config, sourcedEvents *[]SourcedEvent, sourcedTodos *[]SourcedTodo) error {
	// Get events from database within date range
	dbEvents, err := db.GetEventsForMarkdown(cfg.StartDate, cfg.EndDate)
	if err != nil {
		return fmt.Errorf("failed to get events from database: %w", err)
	}

	// Get todos from database
	dbTodos, err := db.GetTodosForMarkdown(cfg.StartDate, cfg.EndDate)
	if err != nil {
		return fmt.Errorf("failed to get todos from database: %w", err)
	}

	// Convert database events to iCalendar events and add to sourced events
	for _, dbEvent := range dbEvents {
		icsEvent := dbEvent.ToICSEvent()
		*sourcedEvents = append(*sourcedEvents, SourcedEvent{
			Event:           icsEvent,
			SourceName:      dbEvent.SourceName,
			CalendarAliases: cfg.CalendarAliases,
		})
	}

	// Convert database todos to iCalendar todos and add to sourced todos
	for _, dbTodo := range dbTodos {
		icsTodo := dbTodo.ToICSTodo()
		*sourcedTodos = append(*sourcedTodos, SourcedTodo{
			Todo:            icsTodo,
			SourceName:      dbTodo.SourceName,
			CalendarAliases: cfg.CalendarAliases,
		})
	}

	return nil
}

// storeInDatabaseAndTrackChanges stores events/todos and tracks which ones are new or changed
// Returns: newEvents, updatedEvents, newTodos, updatedTodos, newOrChangedEventUIDs, newOrChangedTodoUIDs
func storeInDatabaseAndTrackChanges(db *database.DB, sourcedEvents []SourcedEvent, sourcedTodos []SourcedTodo) (int, int, int, int, map[string]bool, map[string]bool) {
	var newEvents, updatedEvents, changedEvents, newTodos, updatedTodos, changedTodos int
	newOrChangedEventUIDs := make(map[string]bool)
	newOrChangedTodoUIDs := make(map[string]bool)

	// Store events with change detection
	for _, sourcedEvent := range sourcedEvents {
		// Get calendar name from event or use source name
		calendarName := sourcedEvent.SourceName
		if calProp := sourcedEvent.Event.GetProperty(ics.ComponentProperty("X-CALENDAR-NAME")); calProp != nil {
			calendarName = calProp.Value
		}

		isNew, changes, err := db.StoreEventWithChanges(sourcedEvent.Event, sourcedEvent.SourceName, "caldav", calendarName)
		if err != nil {
			fmt.Printf("Warning: failed to store event in database: %v\n", err)
			continue
		}

		// Get event UID
		uid := ""
		if uidProp := sourcedEvent.Event.GetProperty(ics.ComponentPropertyUniqueId); uidProp != nil {
			uid = uidProp.Value
		}

		if isNew {
			newEvents++
			newOrChangedEventUIDs[uid] = true
		} else {
			updatedEvents++
			if changes != nil && changes.Changed {
				changedEvents++
				newOrChangedEventUIDs[uid] = true
			}
		}
	}

	// Store todos with change detection
	for _, sourcedTodo := range sourcedTodos {
		// Get calendar name from todo or use source name
		calendarName := sourcedTodo.SourceName
		if calProp := sourcedTodo.Todo.GetProperty(ics.ComponentProperty("X-CALENDAR-NAME")); calProp != nil {
			calendarName = calProp.Value
		}

		isNew, changes, err := db.StoreTodoWithChanges(sourcedTodo.Todo, sourcedTodo.SourceName, "caldav", calendarName)
		if err != nil {
			fmt.Printf("Warning: failed to store todo in database: %v\n", err)
			continue
		}

		// Get todo UID
		uid := ""
		if uidProp := sourcedTodo.Todo.GetProperty(ics.ComponentPropertyUniqueId); uidProp != nil {
			uid = uidProp.Value
		}

		if isNew {
			newTodos++
			newOrChangedTodoUIDs[uid] = true
		} else {
			updatedTodos++
			if changes != nil && changes.Changed {
				changedTodos++
				newOrChangedTodoUIDs[uid] = true
			}
		}
	}

	// Report changes if any
	if changedEvents > 0 || changedTodos > 0 {
		fmt.Printf("Changes detected: %d events modified, %d todos modified\n", changedEvents, changedTodos)
	}

	return newEvents, updatedEvents, newTodos, updatedTodos, newOrChangedEventUIDs, newOrChangedTodoUIDs
}

// filterNewOrChanged filters events/todos to only include new or changed items
func filterNewOrChanged(sourcedEvents []SourcedEvent, sourcedTodos []SourcedTodo, eventUIDs, todoUIDs map[string]bool) ([]SourcedEvent, []SourcedTodo) {
	var filteredEvents []SourcedEvent
	var filteredTodos []SourcedTodo

	// Filter events
	for _, sourcedEvent := range sourcedEvents {
		uid := ""
		if uidProp := sourcedEvent.Event.GetProperty(ics.ComponentPropertyUniqueId); uidProp != nil {
			uid = uidProp.Value
		}
		if eventUIDs[uid] {
			filteredEvents = append(filteredEvents, sourcedEvent)
		}
	}

	// Filter todos
	for _, sourcedTodo := range sourcedTodos {
		uid := ""
		if uidProp := sourcedTodo.Todo.GetProperty(ics.ComponentPropertyUniqueId); uidProp != nil {
			uid = uidProp.Value
		}
		if todoUIDs[uid] {
			filteredTodos = append(filteredTodos, sourcedTodo)
		}
	}

	return filteredEvents, filteredTodos
}
