package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"caldav2markdown/pkg/ics"
)

// SourceMode represents the type of calendar source
type SourceMode string

const (
	SourceModeCalDAV SourceMode = "caldav"
	SourceModeICS    SourceMode = "ics"
)

type Config struct {
	// Legacy CalDAV fields (still supported)
	URL             string
	Username        string
	Password        string
	Output          string
	StartDate       time.Time
	EndDate         time.Time
	UseDueDateEmoji   bool
	UseHashtags       bool
	UseFrontmatter    bool
	IgnoreDescriptions bool
	EventCheckboxes   bool
	ObsidianTasks     bool
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
	// New ICS source options
	SourceMode   SourceMode   `yaml:"source_mode"`
	ICSSources   []ics.Source `yaml:"ics_sources"`
	// Single ICS source (simplified config)
	ICSPath      string        `yaml:"ics_path"`
	ICSURL       string        `yaml:"ics_url"`
	ICSAuth      ics.AuthMethod `yaml:"ics_auth"`
	ICSUsername  string        `yaml:"ics_username"`
	ICSPassword  string        `yaml:"ics_password"`
	ICSToken     string        `yaml:"ics_token"`
	ICSHeaders   map[string]string `yaml:"ics_headers"`
	ICSTimeout   time.Duration `yaml:"ics_timeout"`
}

func LoadFromEnvFile(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	config := &Config{
		Output:     "./events",
		StartDate:  time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), // Default start date
		EndDate:    time.Now().AddDate(2, 0, 0),                  // Default end date (2 years from now)
		SourceMode: SourceModeCalDAV,                             // Default to CalDAV for backward compatibility
		ICSTimeout: 30 * time.Second,                             // Default HTTP timeout for ICS sources
		ICSHeaders: make(map[string]string),
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
		case "IGNORE_DESCRIPTIONS":
			config.IgnoreDescriptions = strings.ToLower(value) == "true" || value == "1"
		case "EVENT_CHECKBOXES":
			config.EventCheckboxes = strings.ToLower(value) == "true" || value == "1"
		case "OBSIDIAN_TASKS":
			config.ObsidianTasks = strings.ToLower(value) == "true" || value == "1"
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
		// New ICS-related configuration options
		case "SOURCE_MODE":
			switch strings.ToLower(value) {
			case "caldav":
				config.SourceMode = SourceModeCalDAV
			case "ics":
				config.SourceMode = SourceModeICS
			default:
				fmt.Printf("Warning: unknown source mode '%s', using CalDAV\n", value)
				config.SourceMode = SourceModeCalDAV
			}
		case "ICS_PATH":
			config.ICSPath = value
		case "ICS_URL":
			config.ICSURL = value
		case "ICS_AUTH":
			switch strings.ToLower(value) {
			case "none":
				config.ICSAuth = ics.AuthNone
			case "basic":
				config.ICSAuth = ics.AuthBasic
			case "bearer":
				config.ICSAuth = ics.AuthBearer
			case "header":
				config.ICSAuth = ics.AuthHeader
			default:
				fmt.Printf("Warning: unknown auth method '%s', using none\n", value)
				config.ICSAuth = ics.AuthNone
			}
		case "ICS_USERNAME":
			config.ICSUsername = value
		case "ICS_PASSWORD":
			config.ICSPassword = value
		case "ICS_TOKEN":
			config.ICSToken = value
		case "ICS_TIMEOUT":
			if timeout, err := time.ParseDuration(value); err == nil {
				config.ICSTimeout = timeout
			} else {
				fmt.Printf("Warning: invalid timeout format '%s', using default\n", value)
			}
		// Handle ICS headers (format: ICS_HEADER_HeaderName=value)
		default:
			if strings.HasPrefix(key, "ICS_HEADER_") {
				headerName := strings.TrimPrefix(key, "ICS_HEADER_")
				config.ICSHeaders[headerName] = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Apply Obsidian tasks preset if enabled
	config.ApplyObsidianTasksPreset()

	return config, nil
}

// ApplyObsidianTasksPreset enables all formatting options that work well with Obsidian tasks
// Only enables options that weren't explicitly set to false
func (c *Config) ApplyObsidianTasksPreset() {
	if c.ObsidianTasks {
		c.EventCheckboxes = true
		c.IgnoreDescriptions = true
		c.UseFrontmatter = true
		c.UseDueDateEmoji = true
		c.UseHashtags = true
	}
}

func (c *Config) Validate() error {
	// Validate based on source mode
	switch c.SourceMode {
	case SourceModeCalDAV:
		return c.validateCalDAVConfig()
	case SourceModeICS:
		return c.validateICSConfig()
	default:
		return fmt.Errorf("unsupported source mode: %s", c.SourceMode)
	}
}

func (c *Config) validateCalDAVConfig() error {
	if c.URL == "" {
		return fmt.Errorf("CALDAV_URL is required for CalDAV mode")
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

func (c *Config) validateICSConfig() error {
	// Check if we have either a simple ICS source config or multiple sources
	hasSimpleConfig := c.ICSPath != "" || c.ICSURL != ""
	hasMultipleConfig := len(c.ICSSources) > 0

	if !hasSimpleConfig && !hasMultipleConfig {
		return fmt.Errorf("ICS mode requires either ICS_PATH/ICS_URL or ICS_SOURCES configuration")
	}

	// Validate simple config if provided
	if hasSimpleConfig {
		if c.ICSPath != "" && c.ICSURL != "" {
			return fmt.Errorf("cannot specify both ICS_PATH and ICS_URL")
		}

		// Validate auth requirements
		switch c.ICSAuth {
		case ics.AuthBasic:
			if c.ICSUsername == "" || c.ICSPassword == "" {
				return fmt.Errorf("ICS_USERNAME and ICS_PASSWORD required for basic auth")
			}
		case ics.AuthBearer:
			if c.ICSToken == "" {
				return fmt.Errorf("ICS_TOKEN required for bearer auth")
			}
		}
	}

	return nil
}

// ToICSSource converts the simple ICS config to an ICS Source
func (c *Config) ToICSSource() (ics.Source, error) {
	if c.ICSPath == "" && c.ICSURL == "" {
		return ics.Source{}, fmt.Errorf("no ICS source configured")
	}

	source := ics.Source{
		Name:     "default",
		Auth:     c.ICSAuth,
		Username: c.ICSUsername,
		Password: c.ICSPassword,
		Token:    c.ICSToken,
		Headers:  c.ICSHeaders,
		Timeout:  c.ICSTimeout,
	}

	if c.ICSPath != "" {
		source.Type = ics.SourceTypeLocal
		source.Path = c.ICSPath
	} else {
		source.Type = ics.SourceTypeRemote
		source.Path = c.ICSURL
	}

	return source, nil
}