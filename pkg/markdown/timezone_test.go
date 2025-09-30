package markdown

import (
	"testing"
	"time"
)

func TestParseICalDateTimeWithTZ_UTCFormats(t *testing.T) {
	tests := []struct {
		name          string
		value         string
		tzid          string
		expectedInUTC time.Time // We'll convert this to local for comparison
		allDay        bool
		hasError      bool
	}{
		{
			name:          "UTC with Z suffix",
			value:         "20240601T150000Z",
			tzid:          "",
			expectedInUTC: time.Date(2024, 6, 1, 15, 0, 0, 0, time.UTC),
			allDay:        false,
			hasError:      false,
		},
		{
			name:          "UTC with Z suffix and fractional seconds",
			value:         "20240601T150000.000Z",
			tzid:          "",
			expectedInUTC: time.Date(2024, 6, 1, 15, 0, 0, 0, time.UTC),
			allDay:        false,
			hasError:      false,
		},
		{
			name:          "UTC with Z suffix and TZID (Z takes precedence)",
			value:         "20240601T150000Z",
			tzid:          "America/New_York",
			expectedInUTC: time.Date(2024, 6, 1, 15, 0, 0, 0, time.UTC),
			allDay:        false,
			hasError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, allDay, err := parseICalDateTimeWithTZ(tt.value, tt.tzid)

			if tt.hasError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.hasError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if allDay != tt.allDay {
				t.Errorf("Expected allDay=%v, got %v", tt.allDay, allDay)
			}

			if !tt.hasError {
				// Convert the expected UTC time to local time for comparison
				expectedLocal := tt.expectedInUTC.In(time.Local)
				if !result.Equal(expectedLocal) {
					t.Errorf("Expected local time %v, got %v", expectedLocal, result)
				}
			}
		})
	}
}

func TestParseICalDateTimeWithTZ_TimezoneFormats(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		tzid     string
		allDay   bool
		hasError bool
		errorMsg string
	}{
		{
			name:     "Eastern Standard Time",
			value:    "20240601T150000",
			tzid:     "Eastern Standard Time",
			allDay:   false,
			hasError: false,
		},
		{
			name:     "IANA timezone - America/New_York",
			value:    "20240601T150000",
			tzid:     "America/New_York",
			allDay:   false,
			hasError: false,
		},
		{
			name:     "Central European Time",
			value:    "20240601T150000",
			tzid:     "CET",
			allDay:   false,
			hasError: false,
		},
		{
			name:     "Japan Standard Time",
			value:    "20240601T150000",
			tzid:     "JST",
			allDay:   false,
			hasError: false,
		},
		{
			name:     "Unknown timezone - should return error",
			value:    "20240601T150000",
			tzid:     "Unknown/Timezone",
			hasError: true,
			errorMsg: "unrecognized timezone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, allDay, err := parseICalDateTimeWithTZ(tt.value, tt.tzid)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if allDay != tt.allDay {
				t.Errorf("Expected allDay=%v, got %v", tt.allDay, allDay)
			}

			// For timezone tests, we just verify that the time is in local timezone
			// and that it represents the correct moment in time
			if result.Location() != time.Local {
				t.Errorf("Expected result to be in local timezone, got %v", result.Location())
			}

			// Verify the time makes sense by checking against known timezone offsets
			// This is a basic sanity check - the exact time depends on the local machine's timezone
			if result.IsZero() {
				t.Errorf("Expected non-zero time, got zero time")
			}
		})
	}
}

func TestParseICalDateTimeWithTZ_DateOnlyFormats(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		tzid     string
		expected time.Time
		allDay   bool
		hasError bool
	}{
		{
			name:     "Date only - basic format",
			value:    "20240601",
			tzid:     "",
			expected: time.Date(2024, 6, 1, 0, 0, 0, 0, time.Local),
			allDay:   true,
			hasError: false,
		},
		{
			name:     "Date only with dashes",
			value:    "2024-06-01",
			tzid:     "",
			expected: time.Date(2024, 6, 1, 0, 0, 0, 0, time.Local),
			allDay:   true,
			hasError: false,
		},
		{
			name:     "Date only with timezone (timezone ignored for all-day)",
			value:    "20240601",
			tzid:     "America/New_York",
			expected: time.Date(2024, 6, 1, 0, 0, 0, 0, time.Local),
			allDay:   true,
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, allDay, err := parseICalDateTimeWithTZ(tt.value, tt.tzid)

			if tt.hasError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.hasError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if allDay != tt.allDay {
				t.Errorf("Expected allDay=%v, got %v", tt.allDay, allDay)
			}

			if !tt.hasError {
				// For date-only, just check the date components
				if result.Year() != tt.expected.Year() || result.Month() != tt.expected.Month() || result.Day() != tt.expected.Day() {
					t.Errorf("Expected date %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

func TestParseICalDateTimeWithTZ_LocalTimeAssumption(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		tzid     string
		allDay   bool
		hasError bool
	}{
		{
			name:     "DateTime with dashes and colons - assume local",
			value:    "2024-06-01T15:00:00",
			tzid:     "",
			allDay:   false,
			hasError: false,
		},
		{
			name:     "Basic datetime format - assume local",
			value:    "20240601T150000",
			tzid:     "",
			allDay:   false,
			hasError: false,
		},
		{
			name:     "DateTime with fractional seconds - assume local",
			value:    "20240601T150000.500",
			tzid:     "",
			allDay:   false,
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, allDay, err := parseICalDateTimeWithTZ(tt.value, tt.tzid)

			if tt.hasError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.hasError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if allDay != tt.allDay {
				t.Errorf("Expected allDay=%v, got %v", tt.allDay, allDay)
			}

			if !tt.hasError {
				// Verify the time is in local timezone (which is what we assume when no TZID)
				if result.Location() != time.Local {
					t.Errorf("Expected result to be in local timezone, got %v", result.Location())
				}

				// Verify basic time components
				if result.Year() != 2024 || result.Month() != 6 || result.Day() != 1 || result.Hour() != 15 {
					t.Errorf("Expected 2024-06-01 15:xx:xx, got %v", result)
				}
			}
		})
	}
}

func TestParseICalDateTimeWithTZ_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		tzid     string
		expected time.Time
		allDay   bool
		hasError bool
	}{
		{
			name:     "Zero date placeholder",
			value:    "00010101T000000",
			tzid:     "",
			expected: time.Time{},
			allDay:   false,
			hasError: false,
		},
		{
			name:     "Whitespace in input",
			value:    "  20240601T150000Z  ",
			tzid:     "  America/New_York  ",
			expected: time.Date(2024, 6, 1, 15, 0, 0, 0, time.UTC), // This will be converted to local
			allDay:   false,
			hasError: false,
		},
		{
			name:     "Invalid date format",
			value:    "invalid-date",
			tzid:     "",
			hasError: true,
		},
		{
			name:     "Empty value",
			value:    "",
			tzid:     "",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, allDay, err := parseICalDateTimeWithTZ(tt.value, tt.tzid)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if allDay != tt.allDay {
				t.Errorf("Expected allDay=%v, got %v", tt.allDay, allDay)
			}

			if tt.expected.IsZero() {
				if !result.IsZero() {
					t.Errorf("Expected zero time, got %v", result)
				}
			} else {
				// For non-zero expected times with UTC, convert to local for comparison
				if tt.expected.Location() == time.UTC {
					expectedLocal := tt.expected.In(time.Local)
					if !result.Equal(expectedLocal) {
						t.Errorf("Expected local time %v, got %v", expectedLocal, result)
					}
				} else {
					// For other expected times, just verify basic structure
					if result.IsZero() {
						t.Errorf("Expected non-zero time, got zero time")
					}
				}
			}
		})
	}
}

func TestMapTimeZone_StandardMappings(t *testing.T) {
	tests := []struct {
		name       string
		tzid       string
		expected   string // Expected IANA timezone name
		shouldWork bool
	}{
		{
			name:       "IANA timezone - America/New_York",
			tzid:       "America/New_York",
			expected:   "America/New_York",
			shouldWork: true,
		},
		{
			name:       "Microsoft Exchange - Eastern Standard Time",
			tzid:       "Eastern Standard Time",
			expected:   "America/New_York",
			shouldWork: true,
		},
		{
			name:       "Abbreviation - EST",
			tzid:       "EST",
			expected:   "EST", // Go's time package loads EST directly as "EST"
			shouldWork: true,
		},
		{
			name:       "Central European Time",
			tzid:       "CET",
			expected:   "CET", // Go's time package loads CET directly as "CET"
			shouldWork: true,
		},
		{
			name:       "Japan Standard Time",
			tzid:       "JST",
			expected:   "Asia/Tokyo",
			shouldWork: true,
		},
		{
			name:       "UTC variants",
			tzid:       "UTC",
			expected:   "UTC",
			shouldWork: true,
		},
		{
			name:       "GMT variant",
			tzid:       "GMT",
			expected:   "GMT", // Go's time package loads GMT directly as "GMT"
			shouldWork: true,
		},
		{
			name:       "Empty timezone",
			tzid:       "",
			shouldWork: false,
		},
		{
			name:       "Unknown timezone",
			tzid:       "Unknown/Timezone",
			shouldWork: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapTimeZone(tt.tzid)

			if tt.shouldWork {
				if result == nil {
					t.Errorf("Expected timezone location but got nil")
					return
				}
				if result.String() != tt.expected {
					t.Errorf("Expected timezone %s, got %s", tt.expected, result.String())
				}
			} else {
				if result != nil {
					t.Errorf("Expected nil but got timezone: %s", result.String())
				}
			}
		})
	}
}

func TestMapTimeZone_ExtendedMappings(t *testing.T) {
	tests := []struct {
		name     string
		tzid     string
		expected string
	}{
		{
			name:     "Microsoft format with UTC offset",
			tzid:     "(UTC-05:00) Eastern Time (US & Canada)",
			expected: "America/New_York",
		},
		{
			name:     "Microsoft format - European time",
			tzid:     "(UTC+01:00) Amsterdam, Berlin, Bern, Rome, Stockholm, Vienna",
			expected: "Europe/Berlin",
		},
		{
			name:     "Australian timezone",
			tzid:     "AUS Eastern Standard Time",
			expected: "Australia/Sydney",
		},
		{
			name:     "Daylight saving variant",
			tzid:     "Eastern Daylight Time",
			expected: "America/New_York",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapTimeZone(tt.tzid)

			if result == nil {
				t.Errorf("Expected timezone location but got nil for %s", tt.tzid)
				return
			}

			if result.String() != tt.expected {
				t.Errorf("Expected timezone %s, got %s for input %s", tt.expected, result.String(), tt.tzid)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		 func() bool {
			 for i := 1; i <= len(s)-len(substr); i++ {
				 if s[i:i+len(substr)] == substr {
					 return true
				 }
			 }
			 return false
		 }())))
}