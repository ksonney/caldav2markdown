package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	URL             string
	Username        string
	Password        string
	Output          string
	StartDate       time.Time
	EndDate         time.Time
	UseDueDateEmoji bool
	UseHashtags     bool
	UseFrontmatter  bool
	// Performance options
	UseServerSideFiltering bool
	// Multi-calendar options
	DiscoverCalendars     bool
	IncludeCalendars      []string
	ExcludeCalendars      []string
	// OAuth fields
	UseOAuth        bool
	ClientID        string
	ClientSecret    string
}

func LoadFromEnvFile(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	config := &Config{
		Output:    "./events",
		StartDate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), // Default start date
		EndDate:   time.Now().AddDate(2, 0, 0),                  // Default end date (2 years from now)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		value = strings.Trim(value, "\"'")

		switch key {
		case "CALDAV_URL":
			config.URL = value
		case "CALDAV_USERNAME":
			config.Username = value
		case "CALDAV_PASSWORD":
			config.Password = value
		case "OUTPUT_DIR":
			config.Output = value
		case "START_DATE":
			if startDate, err := time.Parse("2006-01-02", value); err == nil {
				config.StartDate = startDate
			}
		case "END_DATE":
			if endDate, err := time.Parse("2006-01-02", value); err == nil {
				config.EndDate = endDate
			}
		case "USE_DUE_DATE_EMOJI":
			config.UseDueDateEmoji = strings.ToLower(value) == "true" || value == "1"
		case "USE_HASHTAGS":
			config.UseHashtags = strings.ToLower(value) == "true" || value == "1"
		case "USE_FRONTMATTER":
			config.UseFrontmatter = strings.ToLower(value) == "true" || value == "1"
		case "USE_SERVER_SIDE_FILTERING":
			config.UseServerSideFiltering = strings.ToLower(value) == "true" || value == "1"
		case "DISCOVER_CALENDARS":
			config.DiscoverCalendars = strings.ToLower(value) == "true" || value == "1"
		case "INCLUDE_CALENDARS":
			if value != "" {
				config.IncludeCalendars = strings.Split(value, ",")
				for i, cal := range config.IncludeCalendars {
					config.IncludeCalendars[i] = strings.TrimSpace(cal)
				}
			}
		case "EXCLUDE_CALENDARS":
			if value != "" {
				config.ExcludeCalendars = strings.Split(value, ",")
				for i, cal := range config.ExcludeCalendars {
					config.ExcludeCalendars[i] = strings.TrimSpace(cal)
				}
			}
		case "USE_OAUTH":
			config.UseOAuth = strings.ToLower(value) == "true" || value == "1"
		case "GOOGLE_CLIENT_ID":
			config.ClientID = value
		case "GOOGLE_CLIENT_SECRET":
			config.ClientSecret = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	return config, nil
}

func (c *Config) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("CALDAV_URL is required")
	}

	if c.UseOAuth {
		if c.ClientID == "" {
			return fmt.Errorf("GOOGLE_CLIENT_ID is required when using OAuth")
		}
		if c.ClientSecret == "" {
			return fmt.Errorf("GOOGLE_CLIENT_SECRET is required when using OAuth")
		}
	} else {
		if c.Username == "" {
			return fmt.Errorf("CALDAV_USERNAME is required when not using OAuth")
		}
		if c.Password == "" {
			return fmt.Errorf("CALDAV_PASSWORD is required when not using OAuth")
		}
	}

	return nil
}