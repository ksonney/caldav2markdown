package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"caldav2markdown/pkg/caldav"
	"caldav2markdown/pkg/config"
	"caldav2markdown/pkg/markdown"
)

func main() {
	var (
		url        = flag.String("url", "", "CalDAV server URL")
		username   = flag.String("username", "", "CalDAV username")
		password   = flag.String("password", "", "CalDAV password")
		outputDir  = flag.String("output", "./events", "Output directory for markdown files")
		configFile = flag.String("config", ".env", "Configuration file path")
		testConn   = flag.Bool("test", false, "Test connection only")
		startDate  = flag.String("start", "", "Start date for events (YYYY-MM-DD)")
		endDate    = flag.String("end", "", "End date for events (YYYY-MM-DD)")
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
		fmt.Println("Usage: caldav2markdown -url <caldav-url> -username <user> -password <pass> [-output <dir>] [-config <file>] [-test]")
		fmt.Println("")
		fmt.Println("Required (via flags or config file):")
		fmt.Println("  -url      CalDAV server URL")
		fmt.Println("  -username CalDAV username")
		fmt.Println("  -password CalDAV password")
		fmt.Println("")
		fmt.Println("Optional flags:")
		fmt.Println("  -output   Output directory for markdown files (default: ./events)")
		fmt.Println("  -config   Configuration file path (default: .env)")
		fmt.Println("  -test     Test connection only, don't fetch events")
		fmt.Println("  -start    Start date for events (YYYY-MM-DD, default: 2000-01-01)")
		fmt.Println("  -end      End date for events (YYYY-MM-DD, default: 2 years from now)")
		fmt.Println("")
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	caldavConfig := caldav.Config{
		URL:       cfg.URL,
		Username:  cfg.Username,
		Password:  cfg.Password,
		StartDate: cfg.StartDate,
		EndDate:   cfg.EndDate,
	}

	client := caldav.NewClient(caldavConfig)

	if *testConn {
		fmt.Println("Testing connection...")
		if err := client.TestConnection(); err != nil {
			fmt.Printf("Connection test failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Connection successful!")
		return
	}

	fmt.Println("Fetching events from CalDAV server...")
	result, err := client.GetEventsWithDeduplication()
	if err != nil {
		fmt.Printf("Failed to fetch events: %v\n", err)
		os.Exit(1)
	}

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
	var eventMarkdowns []markdown.EventMarkdown
	for _, event := range result.Events {
		eventMarkdowns = append(eventMarkdowns, markdown.ConvertEvent(event))
	}

	// Generate daily aggregated files
	if err := markdown.GenerateDailyFiles(cfg.Output, eventMarkdowns); err != nil {
		fmt.Printf("Failed to generate daily files: %v\n", err)
		os.Exit(1)
	}

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

	fmt.Printf("Successfully created %d daily files for %d events\n", len(dateCount), len(result.Events))

	if len(result.Todos) > 0 {
		fmt.Printf("Found %d todos, converting to tasks.md...\n", len(result.Todos))

		var todoMarkdowns []markdown.TodoMarkdown
		for _, todo := range result.Todos {
			todoMarkdowns = append(todoMarkdowns, markdown.ConvertTodo(todo))
		}

		if err := markdown.GenerateTasksFile(cfg.Output, todoMarkdowns); err != nil {
			fmt.Printf("Warning: failed to write tasks.md: %v\n", err)
		} else {
			fmt.Println("Written: tasks.md")
		}
	}

	fmt.Printf("Successfully converted %d events to daily markdown files in %s\n", len(result.Events), cfg.Output)
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