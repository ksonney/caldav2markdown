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
	icsource "caldav2markdown/pkg/ics"
	"caldav2markdown/pkg/markdown"
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
		eventCheckboxes        = flag.Bool("event-checkboxes", false, "Add checkboxes to events for task-like formatting")
		obsidianTasks          = flag.Bool("obsidian-tasks", false, "Enable Obsidian tasks preset (event checkboxes, ignore descriptions, frontmatter, emojis, hashtags)")
		useCalendarTags        = flag.Bool("calendar-tags", false, "Add calendar name tags to events and tasks")
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
	if *eventCheckboxes {
		cfg.EventCheckboxes = true
	}
	if *obsidianTasks {
		cfg.ObsidianTasks = true
	}
	if *useCalendarTags {
		cfg.UseCalendarTags = true
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
		fmt.Println("Common Options:")
		fmt.Println("  -output           Output directory for markdown files (default: ./events)")
		fmt.Printf("  -config           Configuration file path (default: %s)\n", defaultConfigPath)
		fmt.Println("  -start            Start date for events (YYYY-MM-DD, default: 2000-01-01)")
		fmt.Println("  -end              End date for events (YYYY-MM-DD, default: 2 years from now)")
		fmt.Println("  -emoji            Use 📅 emoji for due dates in tasks")
		fmt.Println("  -hashtags         Add #event and #task hashtags")
		fmt.Println("  -frontmatter      Add YAML frontmatter to markdown files")
		fmt.Println("  -ignore-descriptions  Ignore event and task descriptions in output")
		fmt.Println("  -event-checkboxes     Add checkboxes to events for task-like formatting")
		fmt.Println("  -obsidian-tasks   Enable Obsidian preset (event checkboxes + ignore descriptions + frontmatter + emojis + hashtags)")
		fmt.Println("")
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
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

	if cfg.HasMultipleSources() {
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
		for i, sourcedEvent := range sourcedEvents {
			eventMarkdowns = append(eventMarkdowns, markdown.ConvertEventWithCalendar(sourcedEvent.Event, sourcedEvent.SourceName, cfg.CalendarAliases))
			if len(sourcedEvents) > 10 && (i+1)%10 == 0 {
				fmt.Printf("\rConverting events to markdown format... (%d/%d)", i+1, len(sourcedEvents))
			}
		}
		fmt.Printf("\rConverted %d events to markdown format\n", len(sourcedEvents))
	}

	// Convert todos to TodoMarkdown format
	var todoMarkdowns []markdown.TodoMarkdown
	if len(sourcedTodos) > 0 {
		fmt.Printf("Converting %d tasks to markdown format...\n", len(sourcedTodos))
		for _, sourcedTodo := range sourcedTodos {
			todoMarkdowns = append(todoMarkdowns, markdown.ConvertTodoWithCalendar(sourcedTodo.Todo, sourcedTodo.SourceName, cfg.CalendarAliases))
		}
	}

	// Generate daily aggregated files with both events and tasks
	fmt.Println("Generating daily files...")
	if err := markdown.GenerateDailyFilesWithAllOptions(cfg.Output, eventMarkdowns, todoMarkdowns, cfg.UseDueDateEmoji, cfg.UseHashtags, cfg.UseFrontmatter, cfg.IgnoreDescriptions, cfg.EventCheckboxes, cfg.UseCalendarTags, progressCallback); err != nil {
		fmt.Printf("\nFailed to generate daily files: %v\n", err)
		os.Exit(1)
	}
	fmt.Println() // New line after progress

	// Count unique dates for reporting
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

func writeMarkdownFile(filename, content string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}