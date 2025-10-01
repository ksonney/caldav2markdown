package markdown

import (
	"strings"
	"testing"
)

func TestExtractUID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "UID present",
			input:    "<!-- uid:abc123 -->\n- **Meeting**",
			expected: "abc123",
		},
		{
			name:     "UID with spaces",
			input:    "<!--   uid:xyz789   -->\n- **Call**",
			expected: "xyz789",
		},
		{
			name:     "No UID",
			input:    "- **Random event**",
			expected: "",
		},
		{
			name:     "Complex UID",
			input:    "<!-- uid:event-2024-01-15-abc@example.com -->\n- **Event**",
			expected: "event-2024-01-15-abc@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractUID(tt.input)
			if result != tt.expected {
				t.Errorf("extractUID() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSmartDeduplicateWithUIDs(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name: "Replace old version with new version (same UID)",
			input: []string{
				"<!-- uid:event123 -->\n- **Meeting** - 14:00-15:00 #event",
				"<!-- uid:event123 -->\n- **Meeting** - 15:00-16:00 #event",
			},
			expected: []string{
				"<!-- uid:event123 -->\n- **Meeting** - 15:00-16:00 #event",
			},
		},
		{
			name: "Keep different UIDs",
			input: []string{
				"<!-- uid:event123 -->\n- **Meeting A** - 14:00-15:00 #event",
				"<!-- uid:event456 -->\n- **Meeting B** - 14:00-15:00 #event",
			},
			expected: []string{
				"<!-- uid:event123 -->\n- **Meeting A** - 14:00-15:00 #event",
				"<!-- uid:event456 -->\n- **Meeting B** - 14:00-15:00 #event",
			},
		},
		{
			name: "Mixed UID and non-UID items",
			input: []string{
				"<!-- uid:event123 -->\n- **Meeting** - 14:00-15:00 #event",
				"- **Manual entry** - Some note",
				"<!-- uid:event123 -->\n- **Meeting** - 15:00-16:00 #event",
			},
			expected: []string{
				"<!-- uid:event123 -->\n- **Meeting** - 15:00-16:00 #event",
				"- **Manual entry** - Some note",
			},
		},
		{
			name: "Todo items with UIDs",
			input: []string{
				"<!-- uid:todo123 -->\n- [ ] **Task A** - Due: 2024-01-15 #task",
				"<!-- uid:todo123 -->\n- [x] **Task A** - Due: 2024-01-15 #task",
			},
			expected: []string{
				"<!-- uid:todo123 -->\n- [x] **Task A** - Due: 2024-01-15 #task",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := smartDeduplicate(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("smartDeduplicate() returned %d items, want %d", len(result), len(tt.expected))
				t.Logf("Got: %v", result)
				t.Logf("Want: %v", tt.expected)
				return
			}

			for i := range result {
				if strings.TrimSpace(result[i]) != strings.TrimSpace(tt.expected[i]) {
					t.Errorf("smartDeduplicate()[%d] = %q, want %q", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestSmartDeduplicateBackwardCompatibility(t *testing.T) {
	// Test that old behavior (content-based dedup) still works for non-UID items
	input := []string{
		"- **Meeting** - 14:00-15:00",
		"- **Meeting** - 14:00-15:00 #event",
		"- **Meeting** - 14:00-15:00 #event #work",
	}

	result := smartDeduplicate(input)

	// Should keep the version with the most hashtags
	if len(result) != 1 {
		t.Errorf("smartDeduplicate() returned %d items, want 1", len(result))
	}

	expected := "- **Meeting** - 14:00-15:00 #event #work"
	if len(result) > 0 && strings.TrimSpace(result[0]) != expected {
		t.Errorf("smartDeduplicate() = %q, want %q", result[0], expected)
	}
}
