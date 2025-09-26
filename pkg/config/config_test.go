package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"caldav2markdown/pkg/ics"
)

func TestParseMultiSourceConfig(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		value       string
		expectParse bool
		description string
	}{
		// Valid cases
		{
			name:        "valid source type",
			key:         "SOURCE_0_TYPE",
			value:       "caldav",
			expectParse: true,
			description: "should parse source type correctly",
		},
		{
			name:        "valid source name",
			key:         "SOURCE_0_NAME",
			value:       "My Calendar",
			expectParse: true,
			description: "should parse source name correctly",
		},
		{
			name:        "valid caldav url",
			key:         "SOURCE_0_CALDAV_URL",
			value:       "https://example.com/caldav",
			expectParse: true,
			description: "should parse CalDAV URL correctly",
		},
		{
			name:        "valid ics path",
			key:         "SOURCE_1_ICS_PATH",
			value:       "/path/to/calendar.ics",
			expectParse: true,
			description: "should parse ICS path correctly",
		},
		// Invalid cases
		{
			name:        "non-source key",
			key:         "OTHER_CONFIG",
			value:       "value",
			expectParse: false,
			description: "should not parse non-source keys",
		},
		{
			name:        "invalid source format",
			key:         "SOURCE_INVALID",
			value:       "value",
			expectParse: false,
			description: "should reject invalid source key format",
		},
		{
			name:        "invalid source index",
			key:         "SOURCE_abc_TYPE",
			value:       "caldav",
			expectParse: false,
			description: "should reject non-numeric source index",
		},
		{
			name:        "invalid source type",
			key:         "SOURCE_0_TYPE",
			value:       "invalid",
			expectParse: false,
			description: "should reject invalid source type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			result := config.parseMultiSourceConfig(tt.key, tt.value)

			if result != tt.expectParse {
				t.Errorf("parseMultiSourceConfig() = %v, expected %v: %s", result, tt.expectParse, tt.description)
			}
		})
	}
}

func TestLoadMultiSourceFromEnvFile(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.env")

	configContent := `# Multi-source configuration test
OUTPUT_DIR=./test-output
START_DATE=2024-01-01
END_DATE=2024-12-31

# Source 0 - CalDAV
SOURCE_0_TYPE=caldav
SOURCE_0_NAME=Google Calendar
SOURCE_0_CALDAV_URL=https://apidata.googleusercontent.com/caldav/v2/primary/events
SOURCE_0_CALDAV_USE_OAUTH=true
SOURCE_0_CALDAV_CLIENT_ID=test_client_id
SOURCE_0_CALDAV_CLIENT_SECRET=test_client_secret
SOURCE_0_CALDAV_DISCOVER_CALENDARS=true
SOURCE_0_CALDAV_CALENDAR_ALIASES=Work:W,Personal:P

# Source 1 - ICS file
SOURCE_1_TYPE=ics
SOURCE_1_NAME=Local Calendar
SOURCE_1_ICS_TYPE=local
SOURCE_1_ICS_PATH=/path/to/calendar.ics

# Source 2 - Remote ICS
SOURCE_2_TYPE=ics
SOURCE_2_NAME=Remote Calendar
SOURCE_2_ICS_TYPE=remote
SOURCE_2_ICS_PATH=https://example.com/calendar.ics
SOURCE_2_ICS_AUTH=basic
SOURCE_2_ICS_USERNAME=user
SOURCE_2_ICS_PASSWORD=pass
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	config, err := LoadFromEnvFile(configFile)
	if err != nil {
		t.Fatalf("LoadFromEnvFile() failed: %v", err)
	}

	// Verify multi-source configuration
	if len(config.Sources) != 3 {
		t.Errorf("Expected 3 sources, got %d", len(config.Sources))
	}

	// Test Source 0 (CalDAV)
	if config.Sources[0].Type != "caldav" {
		t.Errorf("Source 0 type = %s, expected caldav", config.Sources[0].Type)
	}
	if config.Sources[0].Name != "Google Calendar" {
		t.Errorf("Source 0 name = %s, expected 'Google Calendar'", config.Sources[0].Name)
	}
	if config.Sources[0].CalDAVSource == nil {
		t.Fatal("Source 0 CalDAV configuration is nil")
	}
	if config.Sources[0].CalDAVSource.URL != "https://apidata.googleusercontent.com/caldav/v2/primary/events" {
		t.Errorf("Source 0 URL = %s, expected Google CalDAV URL", config.Sources[0].CalDAVSource.URL)
	}
	if !config.Sources[0].CalDAVSource.UseOAuth {
		t.Error("Source 0 should have OAuth enabled")
	}

	// Test Source 1 (Local ICS)
	if config.Sources[1].Type != "ics" {
		t.Errorf("Source 1 type = %s, expected ics", config.Sources[1].Type)
	}
	if config.Sources[1].ICSSource == nil {
		t.Fatal("Source 1 ICS configuration is nil")
	}
	if config.Sources[1].ICSSource.Type != ics.SourceTypeLocal {
		t.Errorf("Source 1 ICS type = %v, expected local", config.Sources[1].ICSSource.Type)
	}

	// Test Source 2 (Remote ICS)
	if config.Sources[2].ICSSource.Type != ics.SourceTypeRemote {
		t.Errorf("Source 2 ICS type = %v, expected remote", config.Sources[2].ICSSource.Type)
	}
	if config.Sources[2].ICSSource.Auth != ics.AuthBasic {
		t.Errorf("Source 2 auth = %v, expected basic", config.Sources[2].ICSSource.Auth)
	}
}

func TestValidateMultiSourceConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid multi-source config",
			config: &Config{
				Output:    "./test",
				StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
				Sources: []SourceConfig{
					{
						Type: "caldav",
						Name: "Test CalDAV",
						CalDAVSource: &CalDAVSource{
							URL:      "https://example.com/caldav",
							Username: "user",
							Password: "pass",
						},
					},
					{
						Type: "ics",
						Name: "Test ICS",
						ICSSource: &ics.Source{
							Type: ics.SourceTypeLocal,
							Path: "/path/to/file.ics",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "fallback to legacy validation with empty sources",
			config: &Config{
				Output:    "./test",
				StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
				Sources:   []SourceConfig{}, // Empty sources should fall back to legacy validation
			},
			expectError: true,
			errorMsg:    "CALDAV_URL is required", // Legacy CalDAV validation error
		},
		{
			name: "duplicate source names",
			config: &Config{
				Output:    "./test",
				StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
				Sources: []SourceConfig{
					{
						Type: "caldav",
						Name: "Duplicate Name",
						CalDAVSource: &CalDAVSource{
							URL:      "https://example.com/caldav",
							Username: "user",
							Password: "pass",
						},
					},
					{
						Type: "ics",
						Name: "Duplicate Name",
						ICSSource: &ics.Source{
							Type: ics.SourceTypeLocal,
							Path: "/path/to/file.ics",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "duplicate source name",
		},
		{
			name: "invalid date range",
			config: &Config{
				Output:    "./test",
				StartDate: time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), // End before start
				Sources: []SourceConfig{
					{
						Type: "caldav",
						Name: "Test CalDAV",
						CalDAVSource: &CalDAVSource{
							URL:      "https://example.com/caldav",
							Username: "user",
							Password: "pass",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "end date",
		},
		{
			name: "truly empty multi-source config",
			config: &Config{
				Output:    "./test",
				StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
				Sources: []SourceConfig{
					{}, // Empty source config to trigger multi-source validation
				},
			},
			expectError: true,
			errorMsg:    "source type is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError && err == nil {
				t.Error("Expected validation error, but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
			if tt.expectError && err != nil && !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("Error message should contain '%s', got: %v", tt.errorMsg, err)
			}
		})
	}
}

func TestValidateCalDAVSource(t *testing.T) {
	config := &Config{}

	tests := []struct {
		name        string
		source      CalDAVSource
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid basic auth",
			source: CalDAVSource{
				URL:      "https://example.com/caldav",
				Username: "user",
				Password: "pass",
			},
			expectError: false,
		},
		{
			name: "valid oauth",
			source: CalDAVSource{
				URL:          "https://example.com/caldav",
				UseOAuth:     true,
				ClientID:     "client_id",
				ClientSecret: "client_secret",
			},
			expectError: false,
		},
		{
			name: "missing url",
			source: CalDAVSource{
				Username: "user",
				Password: "pass",
			},
			expectError: true,
			errorMsg:    "URL is required",
		},
		{
			name: "invalid url",
			source: CalDAVSource{
				URL:      "not-a-url",
				Username: "user",
				Password: "pass",
			},
			expectError: true,
			errorMsg:    "invalid CalDAV URL",
		},
		{
			name: "oauth missing client id",
			source: CalDAVSource{
				URL:          "https://example.com/caldav",
				UseOAuth:     true,
				ClientSecret: "client_secret",
			},
			expectError: true,
			errorMsg:    "client ID is required",
		},
		{
			name: "basic auth missing username",
			source: CalDAVSource{
				URL:      "https://example.com/caldav",
				Password: "pass",
			},
			expectError: true,
			errorMsg:    "username is required",
		},
		{
			name: "conflicting calendar filters",
			source: CalDAVSource{
				URL:              "https://example.com/caldav",
				Username:         "user",
				Password:         "pass",
				IncludeCalendars: []string{"Work"},
				ExcludeCalendars: []string{"Work"},
			},
			expectError: true,
			errorMsg:    "appears in both include and exclude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.validateCalDAVSource(tt.source)

			if tt.expectError && err == nil {
				t.Error("Expected validation error, but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
			if tt.expectError && err != nil && !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("Error message should contain '%s', got: %v", tt.errorMsg, err)
			}
		})
	}
}

func TestValidateICSSource(t *testing.T) {
	config := &Config{}

	tests := []struct {
		name        string
		source      ics.Source
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid local source",
			source: ics.Source{
				Type: ics.SourceTypeLocal,
				Path: "/path/to/file.ics",
			},
			expectError: false,
		},
		{
			name: "valid remote source with basic auth",
			source: ics.Source{
				Type:     ics.SourceTypeRemote,
				Path:     "https://example.com/calendar.ics",
				Auth:     ics.AuthBasic,
				Username: "user",
				Password: "pass",
			},
			expectError: false,
		},
		{
			name: "valid remote source with bearer auth",
			source: ics.Source{
				Type:  ics.SourceTypeRemote,
				Path:  "https://example.com/calendar.ics",
				Auth:  ics.AuthBearer,
				Token: "bearer_token",
			},
			expectError: false,
		},
		{
			name: "missing path",
			source: ics.Source{
				Type: ics.SourceTypeLocal,
			},
			expectError: true,
			errorMsg:    "path is required",
		},
		{
			name: "invalid remote url",
			source: ics.Source{
				Type: ics.SourceTypeRemote,
				Path: "not-a-url",
			},
			expectError: true,
			errorMsg:    "invalid ICS URL",
		},
		{
			name: "basic auth missing credentials",
			source: ics.Source{
				Type: ics.SourceTypeRemote,
				Path: "https://example.com/calendar.ics",
				Auth: ics.AuthBasic,
			},
			expectError: true,
			errorMsg:    "username and password required",
		},
		{
			name: "bearer auth missing token",
			source: ics.Source{
				Type: ics.SourceTypeRemote,
				Path: "https://example.com/calendar.ics",
				Auth: ics.AuthBearer,
			},
			expectError: true,
			errorMsg:    "token required",
		},
		{
			name: "negative timeout",
			source: ics.Source{
				Type:    ics.SourceTypeRemote,
				Path:    "https://example.com/calendar.ics",
				Auth:    ics.AuthNone,
				Timeout: -5 * time.Second,
			},
			expectError: true,
			errorMsg:    "timeout cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.validateICSSource(tt.source)

			if tt.expectError && err == nil {
				t.Error("Expected validation error, but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
			if tt.expectError && err != nil && !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("Error message should contain '%s', got: %v", tt.errorMsg, err)
			}
		})
	}
}

func TestConfigHelperFunctions(t *testing.T) {
	config := &Config{
		Sources: []SourceConfig{
			{Type: "caldav", Name: "Google Calendar"},
			{Type: "ics", Name: "Local Calendar"},
			{Type: "caldav", Name: ""}, // No name
		},
	}

	// Test HasMultipleSources
	if !config.HasMultipleSources() {
		t.Error("Should return true for multiple sources")
	}

	// Test GetSourceNames
	names := config.GetSourceNames()
	expectedNames := []string{"Google Calendar", "Local Calendar", "Source 3"}
	if len(names) != len(expectedNames) {
		t.Errorf("GetSourceNames() returned %d names, expected %d", len(names), len(expectedNames))
	}
	for i, name := range names {
		if name != expectedNames[i] {
			t.Errorf("GetSourceNames()[%d] = %s, expected %s", i, name, expectedNames[i])
		}
	}

	// Test GetSourceByName
	source, found := config.GetSourceByName("Google Calendar")
	if !found {
		t.Error("Should find 'Google Calendar' source")
	}
	if source.Type != "caldav" {
		t.Errorf("Found source type = %s, expected caldav", source.Type)
	}

	// Test GetSourceByName with non-existent name
	_, found = config.GetSourceByName("Non-existent")
	if found {
		t.Error("Should not find non-existent source")
	}
}

func TestMigrateLegacyConfig(t *testing.T) {
	// Test CalDAV legacy migration
	config := &Config{
		URL:      "https://example.com/caldav",
		Username: "user",
		Password: "pass",
	}

	config.migrateLegacyConfig()

	if len(config.Sources) != 1 {
		t.Errorf("Expected 1 migrated source, got %d", len(config.Sources))
	}
	if config.Sources[0].Type != "caldav" {
		t.Errorf("Migrated source type = %s, expected caldav", config.Sources[0].Type)
	}
	if config.Sources[0].Name != "Legacy CalDAV" {
		t.Errorf("Migrated source name = %s, expected 'Legacy CalDAV'", config.Sources[0].Name)
	}

	// Test ICS legacy migration
	config2 := &Config{
		ICSPath: "/path/to/calendar.ics",
	}

	config2.migrateLegacyConfig()

	if len(config2.Sources) != 1 {
		t.Errorf("Expected 1 migrated ICS source, got %d", len(config2.Sources))
	}
	if config2.Sources[0].Type != "ics" {
		t.Errorf("Migrated ICS source type = %s, expected ics", config2.Sources[0].Type)
	}
}

func TestValidationHelpers(t *testing.T) {
	config := &Config{}

	// Test validateURL
	tests := []struct {
		url   string
		valid bool
	}{
		{"https://example.com", true},
		{"http://example.com", true},
		{"https://example.com/path", true},
		{"", false},
		{"not-a-url", false},
		{"ftp://example.com", false},
		{"//example.com", false},
	}

	for _, tt := range tests {
		err := config.validateURL(tt.url)
		if tt.valid && err != nil {
			t.Errorf("validateURL(%s) should be valid, got error: %v", tt.url, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("validateURL(%s) should be invalid, got no error", tt.url)
		}
	}

	// Test validateCalendarFilters
	err := config.validateCalendarFilters([]string{"Work", "Personal"}, []string{"Spam", "Test"})
	if err != nil {
		t.Errorf("validateCalendarFilters should pass with non-overlapping lists: %v", err)
	}

	err = config.validateCalendarFilters([]string{"Work", "Personal"}, []string{"Personal", "Test"})
	if err == nil {
		t.Error("validateCalendarFilters should fail with overlapping lists")
	}

	// Test validateCalendarAliases
	aliases := map[string]string{
		"Work Calendar": "Work",
		"Personal Calendar": "Personal",
	}
	err = config.validateCalendarAliases(aliases)
	if err != nil {
		t.Errorf("validateCalendarAliases should pass with valid aliases: %v", err)
	}

	duplicateAliases := map[string]string{
		"Work Calendar": "Work",
		"Business Calendar": "Work", // Duplicate alias
	}
	err = config.validateCalendarAliases(duplicateAliases)
	if err == nil {
		t.Error("validateCalendarAliases should fail with duplicate aliases")
	}
}

// Benchmark tests for performance
func BenchmarkParseMultiSourceConfig(b *testing.B) {
	config := &Config{}

	for i := 0; i < b.N; i++ {
		config.parseMultiSourceConfig("SOURCE_0_CALDAV_URL", "https://example.com/caldav")
	}
}

func BenchmarkValidateMultiSourceConfig(b *testing.B) {
	config := &Config{
		Output:    "./test",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		Sources: []SourceConfig{
			{
				Type: "caldav",
				Name: "Test CalDAV",
				CalDAVSource: &CalDAVSource{
					URL:      "https://example.com/caldav",
					Username: "user",
					Password: "pass",
				},
			},
		},
	}

	for i := 0; i < b.N; i++ {
		config.Validate()
	}
}

// YAML Configuration Tests

func TestLoadFromYAMLFile(t *testing.T) {
	// Create a temporary YAML config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.yaml")

	yamlContent := `---
output: ./yaml-output
start_date: 2024-01-01T00:00:00Z
end_date: 2024-12-31T23:59:59Z
use_frontmatter: true
use_hashtags: true
event_checkboxes: true
obsidian_tasks: false

sources:
  - type: caldav
    name: Google Calendar
    caldav:
      url: https://apidata.googleusercontent.com/caldav/v2/primary/events
      use_oauth: true
      client_id: test_client_id
      client_secret: test_client_secret
      discover_calendars: true
      calendar_aliases:
        Work: W
        Personal: P

  - type: ics
    name: Local Calendar
    ics:
      type: local
      path: /path/to/calendar.ics

  - type: ics
    name: Remote Calendar
    ics:
      type: remote
      path: https://example.com/calendar.ics
      auth: basic
      username: user
      password: pass
      timeout: 60s
      headers:
        Authorization: Bearer token
        X-Custom: value
`

	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test YAML config file: %v", err)
	}

	config, err := LoadFromYAMLFile(configFile)
	if err != nil {
		t.Fatalf("LoadFromYAMLFile() failed: %v", err)
	}

	// Verify global configuration
	if config.Output != "./yaml-output" {
		t.Errorf("Output = %s, expected ./yaml-output", config.Output)
	}
	if !config.UseFrontmatter {
		t.Error("UseFrontmatter should be true")
	}
	if !config.UseHashtags {
		t.Error("UseHashtags should be true")
	}
	if !config.EventCheckboxes {
		t.Error("EventCheckboxes should be true")
	}

	// Verify multi-source configuration
	if len(config.Sources) != 3 {
		t.Fatalf("Expected 3 sources, got %d", len(config.Sources))
	}

	// Test Source 0 (CalDAV)
	source0 := config.Sources[0]
	if source0.Type != "caldav" {
		t.Errorf("Source 0 type = %s, expected caldav", source0.Type)
	}
	if source0.Name != "Google Calendar" {
		t.Errorf("Source 0 name = %s, expected 'Google Calendar'", source0.Name)
	}
	if source0.CalDAVSource == nil {
		t.Fatal("Source 0 CalDAV configuration is nil")
	}
	if !source0.CalDAVSource.UseOAuth {
		t.Error("Source 0 should have OAuth enabled")
	}
	if len(source0.CalDAVSource.CalendarAliases) != 2 {
		t.Errorf("Expected 2 calendar aliases, got %d", len(source0.CalDAVSource.CalendarAliases))
	}

	// Test Source 1 (Local ICS)
	source1 := config.Sources[1]
	if source1.Type != "ics" {
		t.Errorf("Source 1 type = %s, expected ics", source1.Type)
	}
	if source1.ICSSource == nil {
		t.Fatal("Source 1 ICS configuration is nil")
	}
	if source1.ICSSource.Type != ics.SourceTypeLocal {
		t.Errorf("Source 1 ICS type = %v, expected local", source1.ICSSource.Type)
	}

	// Test Source 2 (Remote ICS)
	source2 := config.Sources[2]
	if source2.ICSSource.Type != ics.SourceTypeRemote {
		t.Errorf("Source 2 ICS type = %v, expected remote", source2.ICSSource.Type)
	}
	if source2.ICSSource.Auth != ics.AuthBasic {
		t.Errorf("Source 2 auth = %v, expected basic", source2.ICSSource.Auth)
	}
	if source2.ICSSource.Timeout != 60*time.Second {
		t.Errorf("Source 2 timeout = %v, expected 60s", source2.ICSSource.Timeout)
	}
	if len(source2.ICSSource.Headers) != 2 {
		t.Errorf("Expected 2 headers, got %d", len(source2.ICSSource.Headers))
	}
}

func TestLoadConfigAutoDetection(t *testing.T) {
	tempDir := t.TempDir()

	// Test YAML file detection
	yamlFile := filepath.Join(tempDir, "config.yaml")
	yamlContent := `---
output: ./yaml-test
sources:
  - type: caldav
    name: Test
    caldav:
      url: https://example.com
      username: user
      password: pass
`
	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write YAML file: %v", err)
	}

	config, err := LoadConfig(yamlFile)
	if err != nil {
		t.Fatalf("LoadConfig() failed for YAML: %v", err)
	}
	if config.Output != "./yaml-test" {
		t.Errorf("YAML config not loaded correctly: output = %s", config.Output)
	}

	// Test env file detection
	envFile := filepath.Join(tempDir, "config.env")
	envContent := `OUTPUT_DIR=./env-test
CALDAV_URL=https://example.com
CALDAV_USERNAME=user
CALDAV_PASSWORD=pass
`
	err = os.WriteFile(envFile, []byte(envContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write env file: %v", err)
	}

	config, err = LoadConfig(envFile)
	if err != nil {
		t.Fatalf("LoadConfig() failed for env: %v", err)
	}
	if config.Output != "./env-test" {
		t.Errorf("Env config not loaded correctly: output = %s", config.Output)
	}

	// Test auto-detection without extension
	noExtFile := filepath.Join(tempDir, "config")
	err = os.WriteFile(noExtFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write no-ext file: %v", err)
	}

	config, err = LoadConfig(noExtFile)
	if err != nil {
		t.Fatalf("LoadConfig() failed for auto-detection: %v", err)
	}
	if config.Output != "./yaml-test" {
		t.Errorf("Auto-detection failed: output = %s", config.Output)
	}
}

func TestSaveToYAMLFile(t *testing.T) {
	tempDir := t.TempDir()
	yamlFile := filepath.Join(tempDir, "export.yaml")

	// Create a config with multi-source setup
	config := &Config{
		Output:         "./test-output",
		StartDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		UseFrontmatter: true,
		UseHashtags:    true,
		Sources: []SourceConfig{
			{
				Type: "caldav",
				Name: "Test CalDAV",
				CalDAVSource: &CalDAVSource{
					URL:      "https://example.com/caldav",
					Username: "user",
					Password: "pass",
					CalendarAliases: map[string]string{
						"Work": "W",
					},
				},
			},
			{
				Type: "ics",
				Name: "Test ICS",
				ICSSource: &ics.Source{
					Type:     ics.SourceTypeRemote,
					Path:     "https://example.com/calendar.ics",
					Auth:     ics.AuthBearer,
					Token:    "token123",
					Timeout:  30 * time.Second,
					Headers:  map[string]string{"X-Test": "value"},
				},
			},
		},
	}

	// Save to YAML
	err := config.SaveToYAMLFile(yamlFile)
	if err != nil {
		t.Fatalf("SaveToYAMLFile() failed: %v", err)
	}

	// Load it back
	loadedConfig, err := LoadFromYAMLFile(yamlFile)
	if err != nil {
		t.Fatalf("Failed to load saved YAML: %v", err)
	}

	// Verify the data
	if loadedConfig.Output != config.Output {
		t.Errorf("Output mismatch: got %s, expected %s", loadedConfig.Output, config.Output)
	}
	if len(loadedConfig.Sources) != len(config.Sources) {
		t.Errorf("Sources count mismatch: got %d, expected %d", len(loadedConfig.Sources), len(config.Sources))
	}

	// Verify CalDAV source
	if loadedConfig.Sources[0].CalDAVSource.URL != config.Sources[0].CalDAVSource.URL {
		t.Errorf("CalDAV URL mismatch")
	}

	// Verify ICS source
	if loadedConfig.Sources[1].ICSSource.Token != config.Sources[1].ICSSource.Token {
		t.Errorf("ICS token mismatch")
	}
}

func TestConvertEnvToYAML(t *testing.T) {
	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, "config.env")
	yamlFile := filepath.Join(tempDir, "config.yaml")

	// Create an env config file
	envContent := `OUTPUT_DIR=./converted-output
USE_FRONTMATTER=true
USE_HASHTAGS=true

# Multi-source configuration
SOURCE_0_TYPE=caldav
SOURCE_0_NAME=Google Calendar
SOURCE_0_CALDAV_URL=https://example.com/caldav
SOURCE_0_CALDAV_USERNAME=user
SOURCE_0_CALDAV_PASSWORD=pass

SOURCE_1_TYPE=ics
SOURCE_1_NAME=Local Calendar
SOURCE_1_ICS_TYPE=local
SOURCE_1_ICS_PATH=/path/to/calendar.ics
`

	err := os.WriteFile(envFile, []byte(envContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write env file: %v", err)
	}

	// Convert to YAML
	err = ConvertEnvToYAML(envFile, yamlFile)
	if err != nil {
		t.Fatalf("ConvertEnvToYAML() failed: %v", err)
	}

	// Load the YAML file
	config, err := LoadFromYAMLFile(yamlFile)
	if err != nil {
		t.Fatalf("Failed to load converted YAML: %v", err)
	}

	// Verify conversion
	if config.Output != "./converted-output" {
		t.Errorf("Output = %s, expected ./converted-output", config.Output)
	}
	if len(config.Sources) != 2 {
		t.Errorf("Expected 2 sources after conversion, got %d", len(config.Sources))
	}
}

func TestValidateYAMLConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid YAML config",
			config: &Config{
				Output:    "./test",
				StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
				Sources: []SourceConfig{
					{
						Type: "caldav",
						Name: "Test",
						CalDAVSource: &CalDAVSource{
							URL:      "https://example.com",
							Username: "user",
							Password: "pass",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "zero start date",
			config: &Config{
				Output:    "./test",
				StartDate: time.Time{}, // Zero value
				EndDate:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
				Sources: []SourceConfig{
					{
						Type: "caldav",
						CalDAVSource: &CalDAVSource{
							URL:      "https://example.com",
							Username: "user",
							Password: "pass",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "start_date must be specified",
		},
		{
			name: "both caldav and ics configured",
			config: &Config{
				Output:    "./test",
				StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
				Sources: []SourceConfig{
					{
						Type: "caldav",
						CalDAVSource: &CalDAVSource{
							URL:      "https://example.com",
							Username: "user",
							Password: "pass",
						},
						ICSSource: &ics.Source{
							Path: "/test.ics",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "cannot have both caldav and ics",
		},
		{
			name: "type mismatch",
			config: &Config{
				Output:    "./test",
				StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
				Sources: []SourceConfig{
					{
						Type: "ics", // Type says ICS
						CalDAVSource: &CalDAVSource{ // But CalDAV config is present
							URL:      "https://example.com",
							Username: "user",
							Password: "pass",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "ICS source configuration is required", // This is the actual error that gets hit first
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateYAMLConfig()

			if tt.expectError && err == nil {
				t.Error("Expected validation error, but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
			if tt.expectError && err != nil && !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("Error message should contain '%s', got: %v", tt.errorMsg, err)
			}
		})
	}
}

func TestYAMLComplexConfiguration(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "complex.yaml")

	// Create a complex YAML configuration
	yamlContent := `---
output: ./complex-output
start_date: 2024-01-01T00:00:00Z
end_date: 2025-12-31T23:59:59Z
use_due_date_emoji: true
use_hashtags: true
use_frontmatter: true
ignore_descriptions: false
event_checkboxes: true
obsidian_tasks: false
trace_web_calls: false

sources:
  - type: caldav
    name: Google Work Calendar
    caldav:
      url: https://apidata.googleusercontent.com/caldav/v2/primary/events
      use_oauth: true
      client_id: work_client_id
      client_secret: work_client_secret
      discover_calendars: true
      use_server_side_filtering: true
      include_calendars:
        - Work
        - Meetings
        - Projects
      exclude_calendars:
        - Archive
        - Spam
      calendar_aliases:
        "Work Calendar": "Work"
        "Meeting Room": "Meetings"
      proxy_url: http://proxy.company.com:8080
      proxy_username: proxy_user
      proxy_password: proxy_pass

  - type: caldav
    name: Personal Nextcloud
    caldav:
      url: https://nextcloud.example.com/remote.php/dav/calendars/user/personal/
      username: personal_user
      password: personal_pass
      use_server_side_filtering: false
      discover_calendars: false

  - type: ics
    name: Local Tasks
    ics:
      type: local
      path: /home/user/tasks.ics

  - type: ics
    name: Public Holidays
    ics:
      type: remote
      path: https://calendar.google.com/calendar/ical/en.usa%23holiday%40group.v.calendar.google.com/public/basic.ics
      auth: none
      timeout: 30s

  - type: ics
    name: Company Events
    ics:
      type: remote
      path: https://company.com/events.ics
      auth: header
      timeout: 60s
      headers:
        Authorization: Bearer company_token
        X-Client-ID: caldav2markdown
        User-Agent: CalDAV2Markdown/1.0
`

	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write complex YAML config: %v", err)
	}

	config, err := LoadFromYAMLFile(configFile)
	if err != nil {
		t.Fatalf("LoadFromYAMLFile() failed: %v", err)
	}

	// Validate the complex configuration
	err = config.ValidateYAMLConfig()
	if err != nil {
		t.Fatalf("ValidateYAMLConfig() failed: %v", err)
	}

	// Verify specific complex configurations
	if len(config.Sources) != 5 {
		t.Errorf("Expected 5 sources, got %d", len(config.Sources))
	}

	// Check CalDAV with proxy
	source0 := config.Sources[0]
	if source0.CalDAVSource.ProxyURL != "http://proxy.company.com:8080" {
		t.Errorf("Proxy URL not loaded correctly")
	}
	if len(source0.CalDAVSource.IncludeCalendars) != 3 {
		t.Errorf("Expected 3 include calendars, got %d", len(source0.CalDAVSource.IncludeCalendars))
	}

	// Check ICS with custom headers
	source4 := config.Sources[4]
	if len(source4.ICSSource.Headers) != 3 {
		t.Errorf("Expected 3 custom headers, got %d", len(source4.ICSSource.Headers))
	}
	if source4.ICSSource.Headers["Authorization"] != "Bearer company_token" {
		t.Errorf("Authorization header not loaded correctly")
	}
}

// Benchmark tests for YAML performance
func BenchmarkLoadFromYAMLFile(b *testing.B) {
	tempDir := b.TempDir()
	configFile := filepath.Join(tempDir, "bench.yaml")

	yamlContent := `---
output: ./bench-output
start_date: 2024-01-01T00:00:00Z
end_date: 2024-12-31T23:59:59Z
use_frontmatter: true

sources:
  - type: caldav
    name: Test CalDAV
    caldav:
      url: https://example.com/caldav
      username: user
      password: pass
`

	os.WriteFile(configFile, []byte(yamlContent), 0644)

	for i := 0; i < b.N; i++ {
		_, err := LoadFromYAMLFile(configFile)
		if err != nil {
			b.Fatalf("LoadFromYAMLFile failed: %v", err)
		}
	}
}

func BenchmarkSaveToYAMLFile(b *testing.B) {
	config := &Config{
		Output:         "./bench-output",
		StartDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		UseFrontmatter: true,
		Sources: []SourceConfig{
			{
				Type: "caldav",
				Name: "Bench CalDAV",
				CalDAVSource: &CalDAVSource{
					URL:      "https://example.com/caldav",
					Username: "user",
					Password: "pass",
				},
			},
		},
	}

	tempDir := b.TempDir()

	for i := 0; i < b.N; i++ {
		configFile := filepath.Join(tempDir, fmt.Sprintf("bench-%d.yaml", i))
		err := config.SaveToYAMLFile(configFile)
		if err != nil {
			b.Fatalf("SaveToYAMLFile failed: %v", err)
		}
	}
}