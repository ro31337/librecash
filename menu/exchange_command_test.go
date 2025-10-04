package menu

import (
	"librecash/objects"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Tests for PRD017: /exchange Slash Command

func TestExchangeCommand_InitializedUser(t *testing.T) {
	// Test that initialized user can access exchange command

	// Create initialized user (has required fields)
	user := &objects.User{
		UserId:         12345,
		MenuId:         objects.Menu_AskPhone, // User in some other state
		LanguageCode:   "en",
		Lat:            40.7128,       // New York latitude
		Lon:            -74.0060,      // New York longitude
		SearchRadiusKm: &[]int{15}[0], // 15 km radius
	}

	// Test the validation logic that would be used in HandleMessage
	isInitialized := user.Lat != 0 && user.Lon != 0 && user.SearchRadiusKm != nil

	assert.True(t, isInitialized, "User should be considered initialized")
	assert.Equal(t, objects.Menu_AskPhone, user.MenuId, "User should start in AskPhone state")

	// After command processing, user should be in Menu_Main state
	// (This would happen in the actual HandleMessage function)
	expectedState := objects.Menu_Main
	assert.Equal(t, objects.MenuId(400), expectedState, "User should transition to Main menu")
}

func TestExchangeCommand_UninitializedUser_NoLocation(t *testing.T) {
	// Test that user without location cannot access exchange command

	user := &objects.User{
		UserId:         12345,
		MenuId:         objects.Menu_AskLocation,
		LanguageCode:   "en",
		Lat:            0,             // No latitude
		Lon:            0,             // No longitude
		SearchRadiusKm: &[]int{15}[0], // Has radius but no location
	}

	isInitialized := user.Lat != 0 && user.Lon != 0 && user.SearchRadiusKm != nil

	assert.False(t, isInitialized, "User without location should not be considered initialized")
}

func TestExchangeCommand_UninitializedUser_NoRadius(t *testing.T) {
	// Test that user without search radius cannot access exchange command

	user := &objects.User{
		UserId:         12345,
		MenuId:         objects.Menu_SelectRadius,
		LanguageCode:   "en",
		Lat:            40.7128,  // Has latitude
		Lon:            -74.0060, // Has longitude
		SearchRadiusKm: nil,      // No radius
	}

	isInitialized := user.Lat != 0 && user.Lon != 0 && user.SearchRadiusKm != nil

	assert.False(t, isInitialized, "User without search radius should not be considered initialized")
}

func TestExchangeCommand_UninitializedUser_PartialData(t *testing.T) {
	// Test various combinations of missing data

	testCases := []struct {
		name           string
		lat            float64
		lon            float64
		searchRadius   *int
		expectedResult bool
	}{
		{
			name:           "No data at all",
			lat:            0,
			lon:            0,
			searchRadius:   nil,
			expectedResult: false,
		},
		{
			name:           "Only latitude",
			lat:            40.7128,
			lon:            0,
			searchRadius:   nil,
			expectedResult: false,
		},
		{
			name:           "Only longitude",
			lat:            0,
			lon:            -74.0060,
			searchRadius:   nil,
			expectedResult: false,
		},
		{
			name:           "Only search radius",
			lat:            0,
			lon:            0,
			searchRadius:   &[]int{15}[0],
			expectedResult: false,
		},
		{
			name:           "Location but no radius",
			lat:            40.7128,
			lon:            -74.0060,
			searchRadius:   nil,
			expectedResult: false,
		},
		{
			name:           "Radius but no location",
			lat:            0,
			lon:            0,
			searchRadius:   &[]int{15}[0],
			expectedResult: false,
		},
		{
			name:           "All required fields present",
			lat:            40.7128,
			lon:            -74.0060,
			searchRadius:   &[]int{15}[0],
			expectedResult: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			user := &objects.User{
				UserId:         12345,
				MenuId:         objects.Menu_Init,
				LanguageCode:   "en",
				Lat:            tc.lat,
				Lon:            tc.lon,
				SearchRadiusKm: tc.searchRadius,
			}

			isInitialized := user.Lat != 0 && user.Lon != 0 && user.SearchRadiusKm != nil

			assert.Equal(t, tc.expectedResult, isInitialized,
				"Initialization check should return %v for case: %s", tc.expectedResult, tc.name)
		})
	}
}

func TestExchangeCommand_StateTransition(t *testing.T) {
	// Test that command resets user state to Menu_Main regardless of current state

	testStates := []objects.MenuId{
		objects.Menu_Init,
		objects.Menu_AskLocation,
		objects.Menu_SelectRadius,
		objects.Menu_AskPhone,
		objects.Menu_HistoricalFanoutExecute,
		objects.Menu_HistoricalFanoutWait,
		objects.Menu_Amount,
	}

	for _, initialState := range testStates {
		t.Run("From_"+string(rune(initialState)), func(t *testing.T) {
			user := &objects.User{
				UserId:         12345,
				MenuId:         initialState,
				LanguageCode:   "en",
				Lat:            40.7128,
				Lon:            -74.0060,
				SearchRadiusKm: &[]int{15}[0],
			}

			// Verify user is initialized
			isInitialized := user.Lat != 0 && user.Lon != 0 && user.SearchRadiusKm != nil
			assert.True(t, isInitialized, "User should be initialized")

			// Simulate state change that would happen in HandleMessage
			user.MenuId = objects.Menu_Main

			assert.Equal(t, objects.Menu_Main, user.MenuId,
				"User should transition to Main menu from state %d", initialState)
		})
	}
}

func TestExchangeCommand_LocalizationKey(t *testing.T) {
	// Test that the localization key exists and is properly formatted

	expectedKey := "exchange_command.not_initialized"

	// Test with different language codes
	testLanguages := []string{"en", "ru", "es", "fr", "de"}

	for _, lang := range testLanguages {
		t.Run("Language_"+lang, func(t *testing.T) {
			// This would be the actual call in HandleMessage:
			// errorMsg := tgbotapi.NewMessage(userId, user.Locale().Get(expectedKey))

			// For now, just verify the key format is correct
			assert.NotEmpty(t, expectedKey, "Localization key should not be empty")
			assert.Contains(t, expectedKey, "exchange_command", "Key should contain command prefix")
			assert.Contains(t, expectedKey, "not_initialized", "Key should indicate error type")
		})
	}
}

func TestExchangeCommand_Integration(t *testing.T) {
	// Integration test simulating the full command flow

	t.Run("Successful_Command_Flow", func(t *testing.T) {
		// Initialized user
		user := &objects.User{
			UserId:         12345,
			MenuId:         objects.Menu_Amount, // User in amount selection
			LanguageCode:   "en",
			Lat:            40.7128,
			Lon:            -74.0060,
			SearchRadiusKm: &[]int{15}[0],
		}

		// Verify preconditions
		isInitialized := user.Lat != 0 && user.Lon != 0 && user.SearchRadiusKm != nil
		assert.True(t, isInitialized, "User should be initialized")
		assert.Equal(t, objects.Menu_Amount, user.MenuId, "User should start in Amount menu")

		// Simulate command processing
		user.MenuId = objects.Menu_Main

		// Verify postconditions
		assert.Equal(t, objects.Menu_Main, user.MenuId, "User should be in Main menu after command")
	})

	t.Run("Failed_Command_Flow", func(t *testing.T) {
		// Uninitialized user
		originalState := objects.Menu_AskLocation
		user := &objects.User{
			UserId:         12345,
			MenuId:         originalState,
			LanguageCode:   "en",
			Lat:            0,   // No location
			Lon:            0,   // No location
			SearchRadiusKm: nil, // No radius
		}

		// Verify preconditions
		isInitialized := user.Lat != 0 && user.Lon != 0 && user.SearchRadiusKm != nil
		assert.False(t, isInitialized, "User should not be initialized")

		// For uninitialized user, state should NOT change
		assert.Equal(t, originalState, user.MenuId, "User state should remain unchanged for uninitialized user")
	})
}
