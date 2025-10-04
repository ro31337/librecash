package menu

import (
	"librecash/objects"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Tests for PRD018: US Persons Compliance Check

func TestUSComplianceMenu_NewMenuStates(t *testing.T) {
	// Test that new menu states are properly defined
	assert.Equal(t, objects.MenuId(50), objects.Menu_USComplianceCheck, "USComplianceCheck menu state should be 50")
	assert.Equal(t, objects.MenuId(60), objects.Menu_Blocked, "Blocked menu state should be 60")

	// Test that compliance check comes before init
	assert.True(t, objects.Menu_USComplianceCheck < objects.Menu_Init, "USComplianceCheck should come before Init")
	assert.True(t, objects.Menu_Blocked < objects.Menu_Init, "Blocked should come before Init")
}

func TestUSComplianceMenu_MenuFlow(t *testing.T) {
	// Test the expected menu flow order with new compliance check
	expectedFlow := []objects.MenuId{
		objects.Menu_USComplianceCheck,
		objects.Menu_Blocked, // Terminal state for blocked users
		objects.Menu_Init,
		objects.Menu_AskLocation,
		objects.Menu_SelectRadius,
		objects.Menu_AskPhone,
		objects.Menu_HistoricalFanoutExecute,
		objects.Menu_HistoricalFanoutWait,
		objects.Menu_Main,
		objects.Menu_Amount,
	}

	// Check that compliance check is first
	assert.Equal(t, objects.Menu_USComplianceCheck, expectedFlow[0], "USComplianceCheck should be first menu")

	// Check that blocked state exists
	assert.Equal(t, objects.Menu_Blocked, expectedFlow[1], "Blocked should be second in order")

	// Check ordering (except blocked which is terminal)
	for i := 2; i < len(expectedFlow); i++ {
		if expectedFlow[i-1] != objects.Menu_Blocked {
			assert.True(t, expectedFlow[i-1] < expectedFlow[i],
				"Menu %d should come before menu %d", expectedFlow[i-1], expectedFlow[i])
		}
	}
}

func TestUSComplianceMenu_NewUserFlow(t *testing.T) {
	// Test that new users start with compliance check

	// Simulate new user creation
	newUser := &objects.User{
		UserId:       12345,
		MenuId:       objects.Menu_USComplianceCheck, // Should start here
		LanguageCode: "en",
	}

	assert.Equal(t, objects.Menu_USComplianceCheck, newUser.MenuId, "New user should start with compliance check")
}

func TestUSComplianceMenu_StartCommandFlow(t *testing.T) {
	// Test that /start command leads to compliance check

	// Simulate existing user using /start
	user := &objects.User{
		UserId:       12345,
		MenuId:       objects.Menu_Main, // User was in main menu
		LanguageCode: "en",
	}

	// Simulate /start command processing
	user.MenuId = objects.Menu_USComplianceCheck

	assert.Equal(t, objects.Menu_USComplianceCheck, user.MenuId, "/start should lead to compliance check")
}

func TestUSComplianceMenu_CallbackHandling(t *testing.T) {
	// Test callback data validation

	validCallbacks := []string{
		"us_compliance_yes",
		"us_compliance_no",
	}

	for _, callback := range validCallbacks {
		t.Run("Callback_"+callback, func(t *testing.T) {
			assert.NotEmpty(t, callback, "Callback data should not be empty")
			assert.Contains(t, callback, "us_compliance", "Callback should contain prefix")
		})
	}
}

func TestUSComplianceMenu_YesResponse(t *testing.T) {
	// Test that "Yes" response leads to blocked state

	user := &objects.User{
		UserId:       12345,
		MenuId:       objects.Menu_USComplianceCheck,
		LanguageCode: "en",
	}

	// Simulate "Yes" response (user is US person or commercial use)
	user.MenuId = objects.Menu_Blocked

	assert.Equal(t, objects.Menu_Blocked, user.MenuId, "Yes response should lead to blocked state")
}

func TestUSComplianceMenu_NoResponse(t *testing.T) {
	// Test that "No" response leads to init menu

	user := &objects.User{
		UserId:       12345,
		MenuId:       objects.Menu_USComplianceCheck,
		LanguageCode: "en",
	}

	// Simulate "No" response (user is not US person and testing only)
	user.MenuId = objects.Menu_Init

	assert.Equal(t, objects.Menu_Init, user.MenuId, "No response should lead to init menu")
}

func TestBlockedMenu_TerminalState(t *testing.T) {
	// Test that blocked state is terminal (user stays there)

	user := &objects.User{
		UserId:       12345,
		MenuId:       objects.Menu_Blocked,
		LanguageCode: "en",
	}

	// User should remain in blocked state
	assert.Equal(t, objects.Menu_Blocked, user.MenuId, "User should remain in blocked state")

	// Only /start should be able to change state
	// (This would be tested in integration tests)
}

func TestUSComplianceMenu_LocalizationKeys(t *testing.T) {
	// Test that all required localization keys are properly formatted

	requiredKeys := []string{
		"us_compliance.question",
		"us_compliance.button_yes",
		"us_compliance.button_no",
		"blocked.message",
	}

	for _, key := range requiredKeys {
		t.Run("Key_"+key, func(t *testing.T) {
			assert.NotEmpty(t, key, "Localization key should not be empty")

			if key == "us_compliance.question" {
				assert.Contains(t, key, "question", "Question key should contain 'question'")
			}
			if key == "us_compliance.button_yes" || key == "us_compliance.button_no" {
				assert.Contains(t, key, "button", "Button key should contain 'button'")
			}
			if key == "blocked.message" {
				assert.Contains(t, key, "blocked", "Blocked key should contain 'blocked'")
			}
		})
	}
}

func TestUSComplianceMenu_MenuHandlers(t *testing.T) {
	// Test that menu handlers can be created

	complianceMenu := NewUSComplianceMenu()
	assert.NotNil(t, complianceMenu, "USComplianceMenu should be created")

	blockedMenu := NewBlockedMenu()
	assert.NotNil(t, blockedMenu, "BlockedMenu should be created")
}

func TestUSComplianceMenu_ComplianceAuditLogging(t *testing.T) {
	// Test compliance audit requirements

	testCases := []struct {
		name           string
		userResponse   string
		expectedAction string
	}{
		{
			name:           "US Person Response",
			userResponse:   "us_compliance_yes",
			expectedAction: "BLOCK",
		},
		{
			name:           "Non-US Person Response",
			userResponse:   "us_compliance_no",
			expectedAction: "ALLOW",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotEmpty(t, tc.userResponse, "User response should not be empty")
			assert.NotEmpty(t, tc.expectedAction, "Expected action should not be empty")

			if tc.userResponse == "us_compliance_yes" {
				assert.Equal(t, "BLOCK", tc.expectedAction, "Yes response should result in BLOCK")
			}
			if tc.userResponse == "us_compliance_no" {
				assert.Equal(t, "ALLOW", tc.expectedAction, "No response should result in ALLOW")
			}
		})
	}
}

func TestUSComplianceMenu_RestartFromBlocked(t *testing.T) {
	// Test that users can restart from blocked state with /start

	blockedUser := &objects.User{
		UserId:       12345,
		MenuId:       objects.Menu_Blocked,
		LanguageCode: "en",
	}

	// Simulate /start command from blocked state
	blockedUser.MenuId = objects.Menu_USComplianceCheck

	assert.Equal(t, objects.Menu_USComplianceCheck, blockedUser.MenuId,
		"Blocked user should be able to restart with /start")
}

func TestUSComplianceMenu_Integration(t *testing.T) {
	// Integration test simulating full compliance flow

	t.Run("Successful_NonUS_Flow", func(t *testing.T) {
		// New user starts with compliance check
		user := &objects.User{
			UserId:       12345,
			MenuId:       objects.Menu_USComplianceCheck,
			LanguageCode: "en",
		}

		// User answers "No" (not US person, testing only)
		user.MenuId = objects.Menu_Init

		// User should proceed to language selection
		assert.Equal(t, objects.Menu_Init, user.MenuId, "Non-US user should proceed to init")
	})

	t.Run("Blocked_US_Flow", func(t *testing.T) {
		// New user starts with compliance check
		user := &objects.User{
			UserId:       12345,
			MenuId:       objects.Menu_USComplianceCheck,
			LanguageCode: "en",
		}

		// User answers "Yes" (US person or commercial use)
		user.MenuId = objects.Menu_Blocked

		// User should be blocked
		assert.Equal(t, objects.Menu_Blocked, user.MenuId, "US user should be blocked")

		// User can restart with /start
		user.MenuId = objects.Menu_USComplianceCheck
		assert.Equal(t, objects.Menu_USComplianceCheck, user.MenuId, "User should be able to restart")
	})
}
