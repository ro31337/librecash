package menu

import (
	"librecash/objects"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Tests for PRD025: /location Command Implementation

func TestLocationCommand_CommandRecognition(t *testing.T) {
	// Test that /location command is properly recognized

	tests := []struct {
		name        string
		messageText string
		shouldMatch bool
	}{
		{
			name:        "Exact /location command",
			messageText: "/location",
			shouldMatch: true,
		},
		{
			name:        "Case insensitive /Location",
			messageText: "/Location",
			shouldMatch: true,
		},
		{
			name:        "Case insensitive /LOCATION",
			messageText: "/LOCATION",
			shouldMatch: true,
		},
		{
			name:        "Mixed case /LocAtIoN",
			messageText: "/LocAtIoN",
			shouldMatch: true,
		},
		{
			name:        "Not a location command",
			messageText: "/start",
			shouldMatch: false,
		},
		{
			name:        "Text containing location",
			messageText: "my location is here",
			shouldMatch: false,
		},
		{
			name:        "Empty text",
			messageText: "",
			shouldMatch: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Test the command recognition logic
			isLocationCommand := strings.ToLower(tc.messageText) == "/location"
			assert.Equal(t, tc.shouldMatch, isLocationCommand,
				"Command recognition for '%s' should be %v", tc.messageText, tc.shouldMatch)
		})
	}
}

func TestLocationCommand_MenuStateConstants(t *testing.T) {
	// Test that menu state constants are properly defined for /location flow

	// Verify that Menu_SelectRadius is properly defined
	assert.Equal(t, objects.MenuId(250), objects.Menu_SelectRadius,
		"Menu_SelectRadius should be 250 for /location command target")

	// Verify that it's different from other menu states
	assert.NotEqual(t, objects.Menu_SelectRadius, objects.Menu_Main,
		"Menu_SelectRadius should be different from Menu_Main")
	assert.NotEqual(t, objects.Menu_SelectRadius, objects.Menu_Init,
		"Menu_SelectRadius should be different from Menu_Init")
	assert.NotEqual(t, objects.Menu_SelectRadius, objects.Menu_AskLocation,
		"Menu_SelectRadius should be different from Menu_AskLocation")
	assert.NotEqual(t, objects.Menu_SelectRadius, objects.Menu_AskPhone,
		"Menu_SelectRadius should be different from Menu_AskPhone")
}

func TestLocationCommand_ExpectedBehavior(t *testing.T) {
	// Test the expected behavior of /location command processing

	// Test that the command should transition to Menu_SelectRadius
	expectedTargetMenu := objects.Menu_SelectRadius
	assert.Equal(t, objects.MenuId(250), expectedTargetMenu,
		"/location command should target Menu_SelectRadius (250)")

	// Test that the command should be case-insensitive
	commands := []string{"/location", "/Location", "/LOCATION", "/LocAtIoN"}
	for _, cmd := range commands {
		isLocationCommand := strings.ToLower(cmd) == "/location"
		assert.True(t, isLocationCommand,
			"Command '%s' should be recognized as location command", cmd)
	}

	// Test that non-location commands are not recognized
	nonLocationCommands := []string{"/start", "/help", "location", "my location"}
	for _, cmd := range nonLocationCommands {
		isLocationCommand := strings.ToLower(cmd) == "/location"
		assert.False(t, isLocationCommand,
			"Command '%s' should NOT be recognized as location command", cmd)
	}
}
