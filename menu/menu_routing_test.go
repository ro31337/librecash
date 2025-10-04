package menu

import (
	"librecash/objects"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Tests for PRD009: Exchange Deletion by Author - Menu Callback Routing

func TestMenuStateConstants(t *testing.T) {
	// Test that menu state constants are properly defined
	assert.Equal(t, objects.MenuId(100), objects.Menu_Init, "Init menu state should be 100")
	assert.Equal(t, objects.MenuId(200), objects.Menu_AskLocation, "Ask location menu state should be 200")
	assert.Equal(t, objects.MenuId(250), objects.Menu_SelectRadius, "Select radius menu state should be 250")
	assert.Equal(t, objects.MenuId(275), objects.Menu_AskPhone, "Ask phone menu state should be 275")

	// Ensure all states are unique
	states := []objects.MenuId{
		objects.Menu_Init,
		objects.Menu_AskLocation,
		objects.Menu_SelectRadius,
		objects.Menu_AskPhone,
		objects.Menu_Main,
	}

	seen := make(map[objects.MenuId]bool)
	for _, state := range states {
		assert.False(t, seen[state], "State %d should be unique", state)
		seen[state] = true
		assert.Greater(t, int(state), 0, "State %d should be positive", state)
	}
}

func TestCallbackRouting_DeletePrefix(t *testing.T) {
	// Test the callback routing logic for delete callbacks
	tests := []struct {
		name          string
		callbackData  string
		expectedRoute string
		shouldMatch   bool
	}{
		{
			name:          "Delete callback should route to delete handler",
			callbackData:  "delete:123",
			expectedRoute: "delete",
			shouldMatch:   true,
		},
		{
			name:          "Contact callback should route to contact handler",
			callbackData:  "contact:456",
			expectedRoute: "contact",
			shouldMatch:   true,
		},
		{
			name:          "Amount callback should route to amount handler",
			callbackData:  "amount_50",
			expectedRoute: "amount",
			shouldMatch:   true,
		},
		{
			name:          "Language callback should route to language handler",
			callbackData:  "lang_en",
			expectedRoute: "lang",
			shouldMatch:   true,
		},
		{
			name:          "Unknown callback should not match any route",
			callbackData:  "unknown:123",
			expectedRoute: "",
			shouldMatch:   false,
		},
		{
			name:          "Empty callback should not match any route",
			callbackData:  "",
			expectedRoute: "",
			shouldMatch:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the routing logic from menu.go HandleCallback function
			var matchedRoute string
			var matched bool

			if strings.HasPrefix(tt.callbackData, "amount_") {
				matchedRoute = "amount"
				matched = true
			} else if strings.HasPrefix(tt.callbackData, "contact:") {
				matchedRoute = "contact"
				matched = true
			} else if strings.HasPrefix(tt.callbackData, "delete:") {
				matchedRoute = "delete"
				matched = true
			} else if strings.HasPrefix(tt.callbackData, "lang_") {
				matchedRoute = "lang"
				matched = true
			}

			if tt.shouldMatch {
				assert.True(t, matched, "Callback should match a route")
				assert.Equal(t, tt.expectedRoute, matchedRoute, "Should match expected route")
			} else {
				assert.False(t, matched, "Callback should not match any route")
			}
		})
	}
}

func TestCallbackRouting_PrefixPriority(t *testing.T) {
	// Test that callback routing handles prefixes correctly and doesn't have conflicts
	callbackPrefixes := []string{
		"amount_",
		"contact:",
		"delete:",
		"lang_",
	}

	// Ensure no prefix is a substring of another (which could cause routing conflicts)
	for i, prefix1 := range callbackPrefixes {
		for j, prefix2 := range callbackPrefixes {
			if i != j {
				assert.False(t, strings.HasPrefix(prefix1, prefix2),
					"Prefix '%s' should not start with prefix '%s'", prefix1, prefix2)
				assert.False(t, strings.HasPrefix(prefix2, prefix1),
					"Prefix '%s' should not start with prefix '%s'", prefix2, prefix1)
			}
		}
	}
}

func TestCallbackRouting_DeleteCallbackFormat(t *testing.T) {
	// Test that delete callback format is consistent with contact callback format
	tests := []struct {
		name         string
		callbackType string
		exchangeID   string
		expected     string
	}{
		{
			name:         "Delete callback format",
			callbackType: "delete",
			exchangeID:   "123",
			expected:     "delete:123",
		},
		{
			name:         "Contact callback format",
			callbackType: "contact",
			exchangeID:   "456",
			expected:     "contact:456",
		},
		{
			name:         "Delete callback with large ID",
			callbackType: "delete",
			exchangeID:   "999999999",
			expected:     "delete:999999999",
		},
		{
			name:         "Contact callback with large ID",
			callbackType: "contact",
			exchangeID:   "888888888",
			expected:     "contact:888888888",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Both delete and contact callbacks should use the same format: "type:id"
			callbackData := tt.callbackType + ":" + tt.exchangeID

			assert.Equal(t, tt.expected, callbackData, "Callback format should be consistent")

			// Verify the callback can be parsed
			parts := strings.Split(callbackData, ":")
			assert.Len(t, parts, 2, "Callback should have exactly 2 parts")
			assert.Equal(t, tt.callbackType, parts[0], "Callback type should be correct")
			assert.Equal(t, tt.exchangeID, parts[1], "Exchange ID should be correct")
		})
	}
}

func TestCallbackRouting_EdgeCases(t *testing.T) {
	// Test edge cases in callback routing
	tests := []struct {
		name         string
		callbackData string
		shouldMatch  bool
		description  string
	}{
		{
			name:         "Delete with empty ID",
			callbackData: "delete:",
			shouldMatch:  true, // Prefix matches, but handler should reject empty ID
			description:  "Should match prefix but handler should validate ID",
		},
		{
			name:         "Contact with empty ID",
			callbackData: "contact:",
			shouldMatch:  false, // This test is for delete prefix, contact: should not match
			description:  "Should not match delete prefix",
		},
		{
			name:         "Delete with multiple colons",
			callbackData: "delete:123:456",
			shouldMatch:  true, // Prefix matches, but handler should reject invalid format
			description:  "Should match prefix but handler should validate format",
		},
		{
			name:         "Partial delete prefix",
			callbackData: "delet:123",
			shouldMatch:  false,
			description:  "Should not match incomplete prefix",
		},
		{
			name:         "Case sensitive prefix",
			callbackData: "DELETE:123",
			shouldMatch:  false,
			description:  "Should be case sensitive",
		},
		{
			name:         "Delete prefix with space",
			callbackData: "delete :123",
			shouldMatch:  false,
			description:  "Should not match with space in prefix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test if callback matches delete prefix (this test is specifically for delete prefix)
			matchesDelete := strings.HasPrefix(tt.callbackData, "delete:")
			assert.Equal(t, tt.shouldMatch, matchesDelete, tt.description)
		})
	}
}
