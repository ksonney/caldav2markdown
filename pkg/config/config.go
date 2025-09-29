package config

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"caldav2markdown/pkg/ics"
	"gopkg.in/yaml.v3"
)

// GetXDGConfigDir returns the XDG config directory for caldav2markdown
// Following XDG Base Directory Specification
func GetXDGConfigDir() (string, error) {
	// Check for XDG_CONFIG_HOME first
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "caldav2markdown"), nil
	}

	// Fall back to ~/.config/caldav2markdown
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(homeDir, ".config", "caldav2markdown"), nil
}

// GetDefaultConfigPath returns the default configuration file path using XDG directories
func GetDefaultConfigPath() (string, error) {
	configDir, err := GetXDGConfigDir()
	if err != nil {
		return "", err
	}

	// Ensure the directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(configDir, "config.yaml"), nil
}

// SourceMode represents the type of calendar source
type SourceMode string

const (
	SourceModeCalDAV SourceMode = "caldav"
	SourceModeICS    SourceMode = "ics"
)

// CalDAVSource represents a CalDAV server configuration
type CalDAVSource struct {
	Name                   string            `yaml:"name"`
	URL                    string            `yaml:"url"`
	Username               string            `yaml:"username"`
	Password               string            `yaml:"password"`
	UseOAuth               bool              `yaml:"use_oauth"`
	ClientID               string            `yaml:"client_id"`
	ClientSecret           string            `yaml:"client_secret"`
	UseServerSideFiltering bool              `yaml:"use_server_side_filtering"`
	DiscoverCalendars      bool              `yaml:"discover_calendars"`
	IncludeCalendars       []string          `yaml:"include_calendars"`
	ExcludeCalendars       []string          `yaml:"exclude_calendars"`
	CalendarAliases        map[string]string `yaml:"calendar_aliases"`
	ProxyURL               string            `yaml:"proxy_url"`
	ProxyUsername          string            `yaml:"proxy_username"`
	ProxyPassword          string            `yaml:"proxy_password"`
}

type Config struct {
	// Global options
	Output            string    `yaml:"output"`
	StartDate         time.Time `yaml:"start_date"`
	EndDate           time.Time `yaml:"end_date"`
	UseDueDateEmoji   bool      `yaml:"use_due_date_emoji"`
	UseHashtags       bool      `yaml:"use_hashtags"`
	UseFrontmatter    bool      `yaml:"use_frontmatter"`
	IgnoreDescriptions bool     `yaml:"ignore_descriptions"`
	EventCheckboxes   bool      `yaml:"event_checkboxes"`
	ObsidianTasks     bool      `yaml:"obsidian_tasks"`
	TraceWebCalls     bool      `yaml:"trace_web_calls"`
	UseCalendarTags   bool      `yaml:"use_calendar_tags"`

	// Multi-source configuration
	Sources []SourceConfig `yaml:"sources"`

	// Legacy single-source fields (still supported for backward compatibility)
	URL             string
	Username        string
	Password        string
	UseServerSideFiltering bool
	DiscoverCalendars     bool
	IncludeCalendars      []string
	ExcludeCalendars      []string
	CalendarAliases       map[string]string
	UseOAuth        bool
	ClientID        string
	ClientSecret    string
	ProxyURL        string
	ProxyUsername   string
	ProxyPassword   string
	SourceMode   SourceMode   `yaml:"source_mode"`
	ICSSources   []ics.Source `yaml:"ics_sources"`
	ICSPath      string        `yaml:"ics_path"`
	ICSURL       string        `yaml:"ics_url"`
	ICSAuth      ics.AuthMethod `yaml:"ics_auth"`
	ICSUsername  string        `yaml:"ics_username"`
	ICSPassword  string        `yaml:"ics_password"`
	ICSToken     string        `yaml:"ics_token"`
	ICSHeaders   map[string]string `yaml:"ics_headers"`
	ICSTimeout   time.Duration `yaml:"ics_timeout"`
}

// SourceConfig represents a single calendar source (CalDAV or ICS)
type SourceConfig struct {
	Type         string        `yaml:"type"` // "caldav" or "ics"
	Name         string        `yaml:"name"` // Optional display name for the source
	CalDAVSource *CalDAVSource `yaml:"caldav,omitempty"`
	ICSSource    *ics.Source   `yaml:"ics,omitempty"`
}

// parseMultiSourceConfig parses multi-source configuration from env file
func (c *Config) parseMultiSourceConfig(key, value string) bool {
	// Pattern: SOURCE_<index>_<field> or SOURCE_<index>_<type>_<field>
	if !strings.HasPrefix(key, "SOURCE_") {
		return false
	}

	parts := strings.Split(key, "_")
	if len(parts) < 3 {
		fmt.Printf("Warning: Invalid source configuration key format: %s\n", key)
		return false
	}

	// Parse source index
	sourceIndex, err := strconv.Atoi(parts[1])
	if err != nil {
		fmt.Printf("Warning: Invalid source index in key '%s': %v\n", key, err)
		return false
	}

	// Validate source index range
	if sourceIndex < 0 || sourceIndex > 99 {
		fmt.Printf("Warning: Source index %d is out of valid range (0-99) in key '%s'\n", sourceIndex, key)
		return false
	}

	// Ensure we have enough sources
	for len(c.Sources) <= sourceIndex {
		c.Sources = append(c.Sources, SourceConfig{})
	}

	source := &c.Sources[sourceIndex]

	if len(parts) == 3 {
		// Pattern: SOURCE_<index>_<field>
		switch parts[2] {
		case "TYPE":
			sourceType := strings.ToLower(value)
			if sourceType != "caldav" && sourceType != "ics" {
				fmt.Printf("Warning: Invalid source type '%s' for source %d, must be 'caldav' or 'ics'\n", value, sourceIndex)
				return false
			}
			source.Type = sourceType
		case "NAME":
			if len(value) > 100 {
				fmt.Printf("Warning: Source name too long for source %d, truncating to 100 characters\n", sourceIndex)
				value = value[:100]
			}
			source.Name = value
		default:
			fmt.Printf("Warning: Unknown source field '%s' in key '%s'\n", parts[2], key)
			return false
		}
		return true
	}

	if len(parts) >= 4 {
		sourceType := strings.ToLower(parts[2])
		field := strings.Join(parts[3:], "_")

		switch sourceType {
		case "caldav":
			if source.CalDAVSource == nil {
				source.CalDAVSource = &CalDAVSource{CalendarAliases: make(map[string]string)}
			}
			success := c.parseCalDAVSourceField(source.CalDAVSource, field, value)
			if !success {
				fmt.Printf("Warning: Failed to parse CalDAV field '%s' in key '%s'\n", field, key)
			}
			return success
		case "ics":
			if source.ICSSource == nil {
				source.ICSSource = &ics.Source{
					Headers:         make(map[string]string),
					CalendarAliases: make(map[string]string),
				}
			}
			success := c.parseICSSourceField(source.ICSSource, field, value)
			if !success {
				fmt.Printf("Warning: Failed to parse ICS field '%s' in key '%s'\n", field, key)
			}
			return success
		default:
			fmt.Printf("Warning: Unknown source type '%s' in key '%s'\n", sourceType, key)
			return false
		}
	}

	return false
}

// parseCalDAVSourceField parses CalDAV-specific source fields
func (c *Config) parseCalDAVSourceField(source *CalDAVSource, field, value string) bool {
	switch field {
	case "NAME":
		source.Name = value
	case "URL":
		source.URL = value
	case "USERNAME":
		source.Username = value
	case "PASSWORD":
		source.Password = value
	case "USE_OAUTH":
		source.UseOAuth = strings.ToLower(value) == "true" || value == "1"
	case "CLIENT_ID":
		source.ClientID = value
	case "CLIENT_SECRET":
		source.ClientSecret = value
	case "USE_SERVER_SIDE_FILTERING":
		source.UseServerSideFiltering = strings.ToLower(value) == "true" || value == "1"
	case "DISCOVER_CALENDARS":
		source.DiscoverCalendars = strings.ToLower(value) == "true" || value == "1"
	case "INCLUDE_CALENDARS":
		if value != "" {
			source.IncludeCalendars = strings.Split(value, ",")
			for i, cal := range source.IncludeCalendars {
				source.IncludeCalendars[i] = strings.TrimSpace(cal)
			}
		}
	case "EXCLUDE_CALENDARS":
		if value != "" {
			source.ExcludeCalendars = strings.Split(value, ",")
			for i, cal := range source.ExcludeCalendars {
				source.ExcludeCalendars[i] = strings.TrimSpace(cal)
			}
		}
	case "CALENDAR_ALIASES":
		if value != "" {
			pairs := strings.Split(value, ",")
			for _, pair := range pairs {
				parts := strings.Split(pair, ":")
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					alias := strings.TrimSpace(parts[1])
					if key != "" && alias != "" {
						source.CalendarAliases[key] = alias
					}
				}
			}
		}
	case "PROXY_URL":
		source.ProxyURL = value
	case "PROXY_USERNAME":
		source.ProxyUsername = value
	case "PROXY_PASSWORD":
		source.ProxyPassword = value
	default:
		return false
	}
	return true
}

// parseICSSourceField parses ICS-specific source fields
func (c *Config) parseICSSourceField(source *ics.Source, field, value string) bool {
	switch field {
	case "NAME":
		source.Name = value
	case "TYPE":
		switch strings.ToLower(value) {
		case "local":
			source.Type = ics.SourceTypeLocal
		case "remote":
			source.Type = ics.SourceTypeRemote
		default:
			return false
		}
	case "PATH":
		source.Path = value
	case "AUTH":
		switch strings.ToLower(value) {
		case "none":
			source.Auth = ics.AuthNone
		case "basic":
			source.Auth = ics.AuthBasic
		case "bearer":
			source.Auth = ics.AuthBearer
		case "header":
			source.Auth = ics.AuthHeader
		default:
			return false
		}
	case "USERNAME":
		source.Username = value
	case "PASSWORD":
		source.Password = value
	case "TOKEN":
		source.Token = value
	case "TIMEOUT":
		if timeout, err := time.ParseDuration(value); err == nil {
			source.Timeout = timeout
		} else {
			fmt.Printf("Warning: invalid timeout format '%s', using default\n", value)
		}
	case "CALENDAR_ALIASES":
		if value != "" {
			if source.CalendarAliases == nil {
				source.CalendarAliases = make(map[string]string)
			}
			pairs := strings.Split(value, ",")
			for _, pair := range pairs {
				parts := strings.Split(pair, ":")
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					alias := strings.TrimSpace(parts[1])
					if key != "" && alias != "" {
						source.CalendarAliases[key] = alias
					}
				}
			}
		}
	default:
		// Handle headers (format: HEADER_<name>)
		if strings.HasPrefix(field, "HEADER_") {
			headerName := strings.TrimPrefix(field, "HEADER_")
			source.Headers[headerName] = value
			return true
		}
		return false
	}
	return true
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
		case "CALENDAR_ALIASES":
			if value != "" {
				config.CalendarAliases = make(map[string]string)
				pairs := strings.Split(value, ",")
				for _, pair := range pairs {
					parts := strings.Split(pair, ":")
					if len(parts) == 2 {
						key := strings.TrimSpace(parts[0])
						alias := strings.TrimSpace(parts[1])
						if key != "" && alias != "" {
							config.CalendarAliases[key] = alias
						}
					}
				}
			}
		case "USE_OAUTH":
			config.UseOAuth = strings.ToLower(value) == "true" || value == "1"
		case "GOOGLE_CLIENT_ID":
			config.ClientID = value
		case "GOOGLE_CLIENT_SECRET":
			config.ClientSecret = value
		case "TRACE_WEB_CALLS":
			config.TraceWebCalls = strings.ToLower(value) == "true" || value == "1"
		case "USE_CALENDAR_TAGS":
			config.UseCalendarTags = strings.ToLower(value) == "true" || value == "1"
		case "PROXY_URL":
			config.ProxyURL = value
		case "PROXY_USERNAME":
			config.ProxyUsername = value
		case "PROXY_PASSWORD":
			config.ProxyPassword = value
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
			} else if strings.HasPrefix(key, "SOURCE_") {
				// Try to parse as multi-source configuration
				config.parseMultiSourceConfig(key, value)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Apply Obsidian tasks preset if enabled
	config.ApplyObsidianTasksPreset()

	// Convert legacy single-source config to multi-source format if needed
	config.migrateLegacyConfig()

	return config, nil
}

// LoadFromYAMLFile loads configuration from a YAML file
func LoadFromYAMLFile(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open YAML config file: %w", err)
	}
	defer file.Close()

	// Read file content
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML config file: %w", err)
	}

	// Create config with defaults
	config := &Config{
		Output:     "./events",
		StartDate:  time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), // Default start date
		EndDate:    time.Now().AddDate(2, 0, 0),                  // Default end date (2 years from now)
		SourceMode: SourceModeCalDAV,                             // Default to CalDAV for backward compatibility
		ICSTimeout: 30 * time.Second,                             // Default HTTP timeout for ICS sources
		ICSHeaders: make(map[string]string),
	}

	// Parse YAML
	err = yaml.Unmarshal(content, config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Initialize calendar aliases for sources if not present
	for i := range config.Sources {
		if config.Sources[i].CalDAVSource != nil && config.Sources[i].CalDAVSource.CalendarAliases == nil {
			config.Sources[i].CalDAVSource.CalendarAliases = make(map[string]string)
		}
		if config.Sources[i].ICSSource != nil && config.Sources[i].ICSSource.Headers == nil {
			config.Sources[i].ICSSource.Headers = make(map[string]string)
		}
		if config.Sources[i].ICSSource != nil && config.Sources[i].ICSSource.CalendarAliases == nil {
			config.Sources[i].ICSSource.CalendarAliases = make(map[string]string)
		}
	}

	// Apply Obsidian tasks preset if enabled
	config.ApplyObsidianTasksPreset()

	// Convert legacy single-source config to multi-source format if needed
	config.migrateLegacyConfig()

	return config, nil
}

// LoadConfig loads configuration from a file, auto-detecting format by extension
func LoadConfig(filename string) (*Config, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".yaml", ".yml":
		return LoadFromYAMLFile(filename)
	case ".env":
		return LoadFromEnvFile(filename)
	default:
		// Try to detect format by reading the file
		content, err := os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		// If it looks like YAML (starts with YAML indicators), parse as YAML
		contentStr := strings.TrimSpace(string(content))
		if strings.HasPrefix(contentStr, "---") ||
		   strings.Contains(contentStr, "sources:") ||
		   strings.Contains(contentStr, "  ") { // Indentation suggests YAML
			return LoadFromYAMLFile(filename)
		}

		// Otherwise, try env format
		return LoadFromEnvFile(filename)
	}
}

// SaveToYAMLFile saves the current configuration to a YAML file
func (c *Config) SaveToYAMLFile(filename string) error {
	// Create a clean copy for export (remove legacy fields)
	exportConfig := *c
	exportConfig.URL = ""
	exportConfig.Username = ""
	exportConfig.Password = ""
	exportConfig.UseServerSideFiltering = false
	exportConfig.DiscoverCalendars = false
	exportConfig.IncludeCalendars = nil
	exportConfig.ExcludeCalendars = nil
	exportConfig.CalendarAliases = nil
	exportConfig.UseOAuth = false
	exportConfig.ClientID = ""
	exportConfig.ClientSecret = ""
	exportConfig.ProxyURL = ""
	exportConfig.ProxyUsername = ""
	exportConfig.ProxyPassword = ""
	exportConfig.ICSSources = nil
	exportConfig.ICSPath = ""
	exportConfig.ICSURL = ""
	exportConfig.ICSAuth = ""
	exportConfig.ICSUsername = ""
	exportConfig.ICSPassword = ""
	exportConfig.ICSToken = ""
	exportConfig.ICSHeaders = nil
	exportConfig.ICSTimeout = 0

	// Marshal to YAML with custom formatting
	data, err := yaml.Marshal(&exportConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	// Write to file
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write YAML config file: %w", err)
	}

	return nil
}

// ConvertEnvToYAML converts an env file to YAML format
func ConvertEnvToYAML(envFile, yamlFile string) error {
	config, err := LoadFromEnvFile(envFile)
	if err != nil {
		return fmt.Errorf("failed to load env config: %w", err)
	}

	err = config.SaveToYAMLFile(yamlFile)
	if err != nil {
		return fmt.Errorf("failed to save YAML config: %w", err)
	}

	return nil
}

// ValidateYAMLConfig performs YAML-specific validation
func (c *Config) ValidateYAMLConfig() error {
	// First run standard validation
	if err := c.Validate(); err != nil {
		return err
	}

	// Additional YAML-specific validations
	if err := c.validateYAMLStructure(); err != nil {
		return fmt.Errorf("YAML structure validation failed: %w", err)
	}

	return nil
}

// validateYAMLStructure validates YAML-specific structure requirements
func (c *Config) validateYAMLStructure() error {
	// Validate date formats in YAML
	if c.StartDate.IsZero() {
		return fmt.Errorf("start_date must be specified in YAML config")
	}
	if c.EndDate.IsZero() {
		return fmt.Errorf("end_date must be specified in YAML config")
	}

	// Validate sources structure
	for i, source := range c.Sources {
		if err := c.validateYAMLSource(source, i); err != nil {
			return fmt.Errorf("source %d validation failed: %w", i, err)
		}
	}

	return nil
}

// validateYAMLSource validates a single source in YAML format
func (c *Config) validateYAMLSource(source SourceConfig, index int) error {
	// Validate that exactly one source type is configured
	hasCalDAV := source.CalDAVSource != nil
	hasICS := source.ICSSource != nil

	if !hasCalDAV && !hasICS {
		return fmt.Errorf("source must have either caldav or ics configuration")
	}
	if hasCalDAV && hasICS {
		return fmt.Errorf("source cannot have both caldav and ics configuration")
	}

	// Validate type field matches the configured source
	if hasCalDAV && source.Type != "caldav" {
		return fmt.Errorf("source type should be 'caldav' when caldav configuration is present")
	}
	if hasICS && source.Type != "ics" {
		return fmt.Errorf("source type should be 'ics' when ics configuration is present")
	}

	return nil
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
		c.UseCalendarTags = true
	}
}

// migrateLegacyConfig converts legacy single-source config to multi-source format
func (c *Config) migrateLegacyConfig() {
	// If we already have sources configured, don't migrate
	if len(c.Sources) > 0 {
		return
	}

	// Check if we have legacy CalDAV config
	if c.URL != "" {
		source := SourceConfig{
			Type: "caldav",
			Name: "Legacy CalDAV",
			CalDAVSource: &CalDAVSource{
				URL:                    c.URL,
				Username:               c.Username,
				Password:               c.Password,
				UseOAuth:               c.UseOAuth,
				ClientID:               c.ClientID,
				ClientSecret:           c.ClientSecret,
				UseServerSideFiltering: c.UseServerSideFiltering,
				DiscoverCalendars:      c.DiscoverCalendars,
				IncludeCalendars:       c.IncludeCalendars,
				ExcludeCalendars:       c.ExcludeCalendars,
				CalendarAliases:        c.CalendarAliases,
				ProxyURL:               c.ProxyURL,
				ProxyUsername:          c.ProxyUsername,
				ProxyPassword:          c.ProxyPassword,
			},
		}
		c.Sources = append(c.Sources, source)
		return
	}

	// Check if we have legacy ICS config
	if c.ICSPath != "" || c.ICSURL != "" {
		icsSource, err := c.ToICSSource()
		if err == nil {
			source := SourceConfig{
				Type:      "ics",
				Name:      "Legacy ICS",
				ICSSource: &icsSource,
			}
			c.Sources = append(c.Sources, source)
		}
		return
	}

	// Check if we have multiple ICS sources configured
	if len(c.ICSSources) > 0 {
		for i, icsSource := range c.ICSSources {
			name := icsSource.Name
			if name == "" {
				name = fmt.Sprintf("ICS Source %d", i+1)
			}
			source := SourceConfig{
				Type:      "ics",
				Name:      name,
				ICSSource: &icsSource,
			}
			c.Sources = append(c.Sources, source)
		}
	}
}

// HasMultipleSources returns true if multiple sources are configured
func (c *Config) HasMultipleSources() bool {
	return len(c.Sources) > 1
}

// GetSources returns all configured sources
func (c *Config) GetSources() []SourceConfig {
	return c.Sources
}

func (c *Config) Validate() error {
	// Validate global configuration first
	if err := c.ValidateGlobalConfig(); err != nil {
		return fmt.Errorf("global configuration error: %w", err)
	}

	// If we have multi-source configuration, validate each source
	if len(c.Sources) > 0 {
		return c.validateMultiSourceConfig()
	}

	// Fall back to legacy validation
	if c.SourceMode == "" {
		c.SourceMode = SourceModeCalDAV
	}

	switch c.SourceMode {
	case SourceModeCalDAV:
		return c.validateCalDAVConfig()
	case SourceModeICS:
		return c.validateICSConfig()
	default:
		return fmt.Errorf("unsupported source mode: %s", c.SourceMode)
	}
}

// validateMultiSourceConfig validates multi-source configuration
func (c *Config) validateMultiSourceConfig() error {
	if len(c.Sources) == 0 {
		return fmt.Errorf("no sources configured")
	}

	// Validate each source
	sourceNames := make(map[string]int)
	for i, source := range c.Sources {
		if err := c.validateSource(source, i); err != nil {
			return fmt.Errorf("source %d (%s): %w", i, source.Name, err)
		}

		// Check for duplicate source names
		if source.Name != "" {
			if existingIndex, exists := sourceNames[source.Name]; exists {
				return fmt.Errorf("duplicate source name '%s' found at indices %d and %d", source.Name, existingIndex, i)
			}
			sourceNames[source.Name] = i
		}
	}

	// Validate that we have at least one valid source
	if err := c.validateAtLeastOneValidSource(); err != nil {
		return err
	}

	return nil
}

// validateAtLeastOneValidSource ensures at least one source is properly configured
func (c *Config) validateAtLeastOneValidSource() error {
	validSources := 0

	for i, source := range c.Sources {
		if err := c.validateSourceConfiguration(source); err == nil {
			validSources++
		} else {
			fmt.Printf("Warning: Source %d (%s) has configuration issues: %v\n", i, source.Name, err)
		}
	}

	if validSources == 0 {
		return fmt.Errorf("no valid sources found - all sources have configuration errors")
	}

	return nil
}

// validateSource validates a single source configuration
func (c *Config) validateSource(source SourceConfig, index int) error {
	if source.Type == "" {
		return fmt.Errorf("source type is required")
	}

	// Validate source name if provided
	if source.Name != "" && len(source.Name) > 100 {
		return fmt.Errorf("source name too long (max 100 characters)")
	}

	switch source.Type {
	case "caldav":
		if source.CalDAVSource == nil {
			return fmt.Errorf("CalDAV source configuration is required for CalDAV type")
		}
		return c.validateCalDAVSource(*source.CalDAVSource)
	case "ics":
		if source.ICSSource == nil {
			return fmt.Errorf("ICS source configuration is required for ICS type")
		}
		return c.validateICSSource(*source.ICSSource)
	default:
		return fmt.Errorf("unsupported source type: %s (supported: caldav, ics)", source.Type)
	}
}

// validateSourceConfiguration validates the basic configuration of a source without detailed validation
func (c *Config) validateSourceConfiguration(source SourceConfig) error {
	if source.Type == "" {
		return fmt.Errorf("source type is required")
	}

	switch source.Type {
	case "caldav":
		if source.CalDAVSource == nil {
			return fmt.Errorf("CalDAV source configuration is required")
		}
		if source.CalDAVSource.URL == "" {
			return fmt.Errorf("CalDAV URL is required")
		}
	case "ics":
		if source.ICSSource == nil {
			return fmt.Errorf("ICS source configuration is required")
		}
		if source.ICSSource.Path == "" {
			return fmt.Errorf("ICS path is required")
		}
	default:
		return fmt.Errorf("unsupported source type: %s", source.Type)
	}

	return nil
}

// validateCalDAVSource validates a CalDAV source
func (c *Config) validateCalDAVSource(source CalDAVSource) error {
	if source.URL == "" {
		return fmt.Errorf("URL is required for CalDAV source")
	}

	// Validate URL format
	if err := c.validateURL(source.URL); err != nil {
		return fmt.Errorf("invalid CalDAV URL: %w", err)
	}

	if source.UseOAuth {
		if source.ClientID == "" {
			return fmt.Errorf("client ID is required when using OAuth")
		}
		if source.ClientSecret == "" {
			return fmt.Errorf("client secret is required when using OAuth")
		}
		// OAuth doesn't require username/password
		if source.Username != "" || source.Password != "" {
			fmt.Printf("Warning: Username/password will be ignored when using OAuth for CalDAV source\n")
		}
	} else {
		if source.Username == "" {
			return fmt.Errorf("username is required when not using OAuth")
		}
		if source.Password == "" {
			return fmt.Errorf("password is required when not using OAuth")
		}
		// OAuth credentials shouldn't be set for basic auth
		if source.ClientID != "" || source.ClientSecret != "" {
			fmt.Printf("Warning: OAuth credentials will be ignored when not using OAuth for CalDAV source\n")
		}
	}

	// Validate proxy configuration if provided
	if source.ProxyURL != "" {
		if err := c.validateURL(source.ProxyURL); err != nil {
			return fmt.Errorf("invalid proxy URL: %w", err)
		}
		// If proxy username is provided, password should also be provided
		if source.ProxyUsername != "" && source.ProxyPassword == "" {
			return fmt.Errorf("proxy password is required when proxy username is provided")
		}
	}

	// Validate calendar filters
	if err := c.validateCalendarFilters(source.IncludeCalendars, source.ExcludeCalendars); err != nil {
		return fmt.Errorf("invalid calendar filters: %w", err)
	}

	// Validate calendar aliases
	if err := c.validateCalendarAliases(source.CalendarAliases); err != nil {
		return fmt.Errorf("invalid calendar aliases: %w", err)
	}

	return nil
}

// validateICSSource validates an ICS source
func (c *Config) validateICSSource(source ics.Source) error {
	if source.Path == "" {
		return fmt.Errorf("path is required for ICS source")
	}

	// Validate source type
	if source.Type != ics.SourceTypeLocal && source.Type != ics.SourceTypeRemote {
		// Attempt to auto-detect type if not explicitly set
		if strings.HasPrefix(source.Path, "http://") || strings.HasPrefix(source.Path, "https://") {
			fmt.Printf("Info: Auto-detected remote ICS source for %s\n", source.Path)
		} else {
			fmt.Printf("Info: Auto-detected local ICS source for %s\n", source.Path)
		}
	}

	if source.Type == ics.SourceTypeRemote {
		// Validate URL format for remote sources
		if err := c.validateURL(source.Path); err != nil {
			return fmt.Errorf("invalid ICS URL: %w", err)
		}

		// Validate auth requirements for remote sources
		switch source.Auth {
		case ics.AuthBasic:
			if source.Username == "" || source.Password == "" {
				return fmt.Errorf("username and password required for basic auth")
			}
			// Warn if token is set for basic auth
			if source.Token != "" {
				fmt.Printf("Warning: Token will be ignored when using basic auth for ICS source\n")
			}
		case ics.AuthBearer:
			if source.Token == "" {
				return fmt.Errorf("token required for bearer auth")
			}
			// Warn if username/password are set for bearer auth
			if source.Username != "" || source.Password != "" {
				fmt.Printf("Warning: Username/password will be ignored when using bearer auth for ICS source\n")
			}
		case ics.AuthHeader:
			if len(source.Headers) == 0 {
				return fmt.Errorf("custom headers required for header auth")
			}
		case ics.AuthNone:
			// No auth required, but warn if credentials are provided
			if source.Username != "" || source.Password != "" || source.Token != "" {
				fmt.Printf("Warning: Authentication credentials will be ignored when using no auth for ICS source\n")
			}
		default:
			return fmt.Errorf("invalid auth method for ICS source: %v", source.Auth)
		}

		// Validate timeout
		if source.Timeout < 0 {
			return fmt.Errorf("timeout cannot be negative")
		}
		if source.Timeout > 10*time.Minute {
			fmt.Printf("Warning: ICS timeout of %v is very high, consider reducing it\n", source.Timeout)
		}
	} else if source.Type == ics.SourceTypeLocal {
		// For local files, auth credentials should not be set
		if source.Auth != ics.AuthNone {
			fmt.Printf("Warning: Authentication method will be ignored for local ICS file\n")
		}
		if source.Username != "" || source.Password != "" || source.Token != "" {
			fmt.Printf("Warning: Authentication credentials will be ignored for local ICS file\n")
		}
	}

	// Validate headers format
	if err := c.validateHeaders(source.Headers); err != nil {
		return fmt.Errorf("invalid headers: %w", err)
	}

	return nil
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
		Name:            "default",
		Auth:            c.ICSAuth,
		Username:        c.ICSUsername,
		Password:        c.ICSPassword,
		Token:           c.ICSToken,
		Headers:         c.ICSHeaders,
		Timeout:         c.ICSTimeout,
		CalendarAliases: c.CalendarAliases,
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

// validateURL validates that a URL is properly formatted
func (c *Config) validateURL(urlStr string) error {
	if urlStr == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if parsedURL.Scheme == "" {
		return fmt.Errorf("URL must include a scheme (http:// or https://)")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https, got: %s", parsedURL.Scheme)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("URL must include a host")
	}

	return nil
}

// validateCalendarFilters validates include and exclude calendar filters
func (c *Config) validateCalendarFilters(includeCalendars, excludeCalendars []string) error {
	// Check for overlap between include and exclude lists
	includeMap := make(map[string]bool)
	for _, cal := range includeCalendars {
		if cal == "" {
			return fmt.Errorf("calendar name cannot be empty in include list")
		}
		includeMap[cal] = true
	}

	for _, cal := range excludeCalendars {
		if cal == "" {
			return fmt.Errorf("calendar name cannot be empty in exclude list")
		}
		if includeMap[cal] {
			return fmt.Errorf("calendar '%s' appears in both include and exclude lists", cal)
		}
	}

	return nil
}

// validateCalendarAliases validates calendar alias mappings
func (c *Config) validateCalendarAliases(aliases map[string]string) error {
	aliasValues := make(map[string]string)

	for calendarName, alias := range aliases {
		if calendarName == "" {
			return fmt.Errorf("calendar name cannot be empty in aliases")
		}
		if alias == "" {
			return fmt.Errorf("alias cannot be empty for calendar '%s'", calendarName)
		}

		// Check for duplicate alias values
		if existingCalendar, exists := aliasValues[alias]; exists {
			return fmt.Errorf("duplicate alias '%s' used for calendars '%s' and '%s'", alias, existingCalendar, calendarName)
		}
		aliasValues[alias] = calendarName

		// Validate length limits
		if len(calendarName) > 200 {
			return fmt.Errorf("calendar name '%s' is too long (max 200 characters)", calendarName)
		}
		if len(alias) > 100 {
			return fmt.Errorf("alias '%s' is too long (max 100 characters)", alias)
		}
	}

	return nil
}

// validateHeaders validates custom HTTP headers
func (c *Config) validateHeaders(headers map[string]string) error {
	for name, value := range headers {
		if name == "" {
			return fmt.Errorf("header name cannot be empty")
		}
		if len(name) > 100 {
			return fmt.Errorf("header name '%s' is too long (max 100 characters)", name)
		}
		if len(value) > 1000 {
			return fmt.Errorf("header value for '%s' is too long (max 1000 characters)", name)
		}

		// Check for problematic header names
		normalizedName := strings.ToLower(name)
		if normalizedName == "content-length" || normalizedName == "host" {
			return fmt.Errorf("header '%s' cannot be set manually", name)
		}
	}

	return nil
}

// ValidateGlobalConfig validates global configuration options
func (c *Config) ValidateGlobalConfig() error {
	// Validate output directory
	if c.Output == "" {
		return fmt.Errorf("output directory cannot be empty")
	}

	// Validate date range
	if c.EndDate.Before(c.StartDate) {
		return fmt.Errorf("end date (%s) cannot be before start date (%s)",
			c.EndDate.Format("2006-01-02"), c.StartDate.Format("2006-01-02"))
	}

	// Warn about very large date ranges
	duration := c.EndDate.Sub(c.StartDate)
	if duration > 10*365*24*time.Hour { // 10 years
		fmt.Printf("Warning: Date range is very large (%v), this may impact performance\n", duration)
	}

	return nil
}

// GetSourceNames returns a list of all configured source names
func (c *Config) GetSourceNames() []string {
	var names []string
	for i, source := range c.Sources {
		name := source.Name
		if name == "" {
			name = fmt.Sprintf("Source %d", i+1)
		}
		names = append(names, name)
	}
	return names
}

// GetSourceByName returns a source by its name
func (c *Config) GetSourceByName(name string) (SourceConfig, bool) {
	for _, source := range c.Sources {
		if source.Name == name {
			return source, true
		}
	}
	return SourceConfig{}, false
}