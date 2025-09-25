package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"caldav2markdown/pkg/caldav"
	"caldav2markdown/pkg/config"
	"caldav2markdown/pkg/markdown"
)

func main() {
	var (
		url                    = flag.String("url", "", "CalDAV server URL")
		username               = flag.String("username", "", "CalDAV username")
		password               = flag.String("password", "", "CalDAV password")
		outputDir              = flag.String("output", "./events", "Output directory for markdown files")
		configFile             = flag.String("config", ".env", "Configuration file path")
		testConn               = flag.Bool("test", false, "Test connection only")
		startDate              = flag.String("start", "", "Start date for events (YYYY-MM-DD)")
		endDate                = flag.String("end", "", "End date for events (YYYY-MM-DD)")
		useDueDateEmoji        = flag.Bool("emoji", false, "Use 📅 emoji for due dates in tasks")
		useHashtags            = flag.Bool("hashtags", false, "Add #event and #task hashtags")
		useFrontmatter         = flag.Bool("frontmatter", false, "Add YAML frontmatter to markdown files")
		useServerSideFiltering = flag.Bool("server-side-filtering", false, "Use CalDAV server-side filtering (faster for large calendars)")
		discoverCalendars      = flag.Bool("discover-calendars", false, "Discover and process all calendars on the server")
		includeCalendars       = flag.String("include-calendars", "", "Comma-separated list of calendar names/URLs to include")
		excludeCalendars       = flag.String("exclude-calendars", "", "Comma-separated list of calendar names/URLs to exclude")
		listCalendars          = flag.Bool("list-calendars", false, "List available calendars and exit")
		useOAuth               = flag.Bool("oauth", false, "Use OAuth 2.0 authentication for Google Calendar")
		clientID               = flag.String("client-id", "", "Google OAuth Client ID")
		clientSecret           = flag.String("client-secret", "", "Google OAuth Client Secret")
	)
	flag.Parse()

	var cfg *config.Config
	var err error

	if _, err := os.Stat(*configFile); err == nil {
		fmt.Printf("Loading configuration from %s\n", *configFile)
		cfg, err = config.LoadFromEnvFile(*configFile)
		if err != nil {
			fmt.Printf("Warning: failed to load config file: %v\n", err)
			cfg = &config.Config{}
		}
	} else {
		cfg = &config.Config{}
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

	if err := cfg.Validate(); err != nil {
		fmt.Println("Usage: caldav2markdown -url <caldav-url> [-username <user> -password <pass>] [-oauth -client-id <id> -client-secret <secret>] [-output <dir>] [-config <file>] [-test]")
		fmt.Println("")
		fmt.Println("Required (via flags or config file):")
		fmt.Println("  -url      CalDAV server URL")
		fmt.Println("")
		fmt.Println("Authentication (choose one):")
		fmt.Println("  Basic Auth:")
		fmt.Println("    -username CalDAV username")
		fmt.Println("    -password CalDAV password")
		fmt.Println("  OAuth 2.0 (for Google Calendar):")
		fmt.Println("    -oauth         Enable OAuth 2.0 authentication")
		fmt.Println("    -client-id     Google OAuth Client ID")
		fmt.Println("    -client-secret Google OAuth Client Secret")
		fmt.Println("")
		fmt.Println("Optional flags:")
		fmt.Println("  -output   Output directory for markdown files (default: ./events)")
		fmt.Println("  -config   Configuration file path (default: .env)")
		fmt.Println("  -test     Test connection only, don't fetch events")
		fmt.Println("  -start    Start date for events (YYYY-MM-DD, default: 2000-01-01)")
		fmt.Println("  -end         End date for events (YYYY-MM-DD, default: 2 years from now)")
		fmt.Println("  -emoji              Use 📅 emoji for due dates in tasks")
		fmt.Println("  -hashtags           Add #event and #task hashtags")
		fmt.Println("  -frontmatter        Add YAML frontmatter to markdown files")
		fmt.Println("  -server-side-filtering Use CalDAV server-side filtering (faster for large calendars)")
		fmt.Println("  -discover-calendars Discover and process all calendars on the server")
		fmt.Println("  -include-calendars  Comma-separated list of calendar names to include")
		fmt.Println("  -exclude-calendars  Comma-separated list of calendar names to exclude")
		fmt.Println("  -list-calendars     List available calendars and exit")
		fmt.Println("")
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

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
	}

	client := caldav.NewClient(caldavConfig)

	if *listCalendars {
		fmt.Println("Discovering calendars...")
		calendars, err := client.DiscoverCalendars()
		if err != nil {
			fmt.Printf("Failed to discover calendars: %v\n", err)
			os.Exit(1)
		}

		if len(calendars) == 0 {
			fmt.Println("No calendars found.")
			return
		}

		fmt.Printf("Found %d calendar(s):\n", len(calendars))
		for i, calendar := range calendars {
			fmt.Printf("%d. %s\n", i+1, calendar.DisplayName)
			fmt.Printf("   URL: %s\n", calendar.URL)
			fmt.Printf("   Supported components: %v\n", calendar.Components)
			fmt.Println()
		}
		return
	}

	if *testConn {
		fmt.Println("Testing connection...")
		if err := client.TestConnection(); err != nil {
			fmt.Printf("Connection test failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Connection successful!")
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

	fmt.Println("Fetching events from CalDAV server...")
	result, err := client.GetEventsWithDeduplicationAndProgress(progressCallback)
	if err != nil {
		fmt.Printf("\nFailed to fetch events: %v\n", err)
		os.Exit(1)
	}
	fmt.Println() // New line after progress

	if len(result.Events) == 0 {
		fmt.Println("No events found.")
		return
	}

	if result.DuplicatesFound > 0 {
		fmt.Printf("Found %d duplicate events (skipped)\n", result.DuplicatesFound)
	}

	if err := os.MkdirAll(cfg.Output, 0755); err != nil {
		fmt.Printf("Failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d unique events, converting to daily markdown files...\n", len(result.Events))

	// Convert events to EventMarkdown format
	fmt.Print("Converting events to markdown format...")
	var eventMarkdowns []markdown.EventMarkdown
	for i, event := range result.Events {
		eventMarkdowns = append(eventMarkdowns, markdown.ConvertEvent(event))
		if len(result.Events) > 10 && (i+1)%10 == 0 {
			fmt.Printf("\rConverting events to markdown format... (%d/%d)", i+1, len(result.Events))
		}
	}
	fmt.Printf("\rConverted %d events to markdown format\n", len(result.Events))

	// Convert todos to TodoMarkdown format
	var todoMarkdowns []markdown.TodoMarkdown
	if len(result.Todos) > 0 {
		fmt.Printf("Converting %d tasks to markdown format...\n", len(result.Todos))
		for _, todo := range result.Todos {
			todoMarkdowns = append(todoMarkdowns, markdown.ConvertTodo(todo))
		}
	}

	// Generate daily aggregated files with both events and tasks
	fmt.Println("Generating daily files...")
	if err := markdown.GenerateDailyFilesWithTasksProgressAndFrontmatter(cfg.Output, eventMarkdowns, todoMarkdowns, cfg.UseDueDateEmoji, cfg.UseHashtags, cfg.UseFrontmatter, progressCallback); err != nil {
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

	fmt.Printf("Successfully created %d daily files for %d events and %d tasks\n", len(dateCount), len(result.Events), len(result.Todos))
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