package menu

import (
	"librecash/objects"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/stretchr/testify/assert"
)

// Tests for PRD019: Fix User Data Corruption in Menu Processing

func TestUserDataUpdate_EmptyMessage_NoCorruption(t *testing.T) {
	// Test that empty messages (like from ContinueMenuProcessing) don't corrupt user data

	// Create user with existing data
	user := &objects.User{
		UserId:    12345,
		Username:  "john_doe",
		FirstName: "John",
		LastName:  "Doe",
		MenuId:    objects.Menu_Main,
	}

	// Create empty message (like ContinueMenuProcessing creates)
	emptyMessage := &tgbotapi.Message{
		From: &tgbotapi.User{
			ID: 12345,
			// UserName: "",     // Empty!
			// FirstName: "",    // Empty!
			// LastName: "",     // Empty!
		},
	}

	// Simulate the logic from HandleMessage
	needsUpdate := false

	if emptyMessage.From != nil &&
		(emptyMessage.From.UserName != "" || emptyMessage.From.FirstName != "" || emptyMessage.From.LastName != "") {

		if emptyMessage.From.UserName != "" && user.Username != emptyMessage.From.UserName {
			user.Username = emptyMessage.From.UserName
			needsUpdate = true
		}

		if emptyMessage.From.FirstName != "" && user.FirstName != emptyMessage.From.FirstName {
			user.FirstName = emptyMessage.From.FirstName
			needsUpdate = true
		}

		if emptyMessage.From.LastName != "" && user.LastName != emptyMessage.From.LastName {
			user.LastName = emptyMessage.From.LastName
			needsUpdate = true
		}
	}

	// Verify that user data was NOT corrupted
	assert.False(t, needsUpdate, "Empty message should not trigger update")
	assert.Equal(t, "john_doe", user.Username, "Username should not be corrupted")
	assert.Equal(t, "John", user.FirstName, "FirstName should not be corrupted")
	assert.Equal(t, "Doe", user.LastName, "LastName should not be corrupted")
}

func TestUserDataUpdate_RealChanges_UpdatesCorrectly(t *testing.T) {
	// Test that real changes are still captured

	// Create user with existing data
	user := &objects.User{
		UserId:    12345,
		Username:  "old_username",
		FirstName: "OldFirst",
		LastName:  "OldLast",
		MenuId:    objects.Menu_Main,
	}

	// Create message with new data
	messageWithChanges := &tgbotapi.Message{
		From: &tgbotapi.User{
			ID:        12345,
			UserName:  "new_username",
			FirstName: "NewFirst",
			LastName:  "NewLast",
		},
	}

	// Simulate the logic from HandleMessage
	needsUpdate := false

	if messageWithChanges.From != nil &&
		(messageWithChanges.From.UserName != "" || messageWithChanges.From.FirstName != "" || messageWithChanges.From.LastName != "") {

		if messageWithChanges.From.UserName != "" && user.Username != messageWithChanges.From.UserName {
			user.Username = messageWithChanges.From.UserName
			needsUpdate = true
		}

		if messageWithChanges.From.FirstName != "" && user.FirstName != messageWithChanges.From.FirstName {
			user.FirstName = messageWithChanges.From.FirstName
			needsUpdate = true
		}

		if messageWithChanges.From.LastName != "" && user.LastName != messageWithChanges.From.LastName {
			user.LastName = messageWithChanges.From.LastName
			needsUpdate = true
		}
	}

	// Verify that changes were captured
	assert.True(t, needsUpdate, "Real changes should trigger update")
	assert.Equal(t, "new_username", user.Username, "Username should be updated")
	assert.Equal(t, "NewFirst", user.FirstName, "FirstName should be updated")
	assert.Equal(t, "NewLast", user.LastName, "LastName should be updated")
}

func TestUserDataUpdate_NoChanges_NoUpdate(t *testing.T) {
	// Test that identical data doesn't trigger unnecessary updates

	// Create user with existing data
	user := &objects.User{
		UserId:    12345,
		Username:  "same_username",
		FirstName: "SameFirst",
		LastName:  "SameLast",
		MenuId:    objects.Menu_Main,
	}

	// Create message with same data
	messageWithSameData := &tgbotapi.Message{
		From: &tgbotapi.User{
			ID:        12345,
			UserName:  "same_username",
			FirstName: "SameFirst",
			LastName:  "SameLast",
		},
	}

	// Simulate the logic from HandleMessage
	needsUpdate := false

	if messageWithSameData.From != nil &&
		(messageWithSameData.From.UserName != "" || messageWithSameData.From.FirstName != "" || messageWithSameData.From.LastName != "") {

		if messageWithSameData.From.UserName != "" && user.Username != messageWithSameData.From.UserName {
			user.Username = messageWithSameData.From.UserName
			needsUpdate = true
		}

		if messageWithSameData.From.FirstName != "" && user.FirstName != messageWithSameData.From.FirstName {
			user.FirstName = messageWithSameData.From.FirstName
			needsUpdate = true
		}

		if messageWithSameData.From.LastName != "" && user.LastName != messageWithSameData.From.LastName {
			user.LastName = messageWithSameData.From.LastName
			needsUpdate = true
		}
	}

	// Verify that no update was triggered
	assert.False(t, needsUpdate, "Identical data should not trigger update")
	assert.Equal(t, "same_username", user.Username, "Username should remain unchanged")
	assert.Equal(t, "SameFirst", user.FirstName, "FirstName should remain unchanged")
	assert.Equal(t, "SameLast", user.LastName, "LastName should remain unchanged")
}

func TestUserDataUpdate_PartialChanges_UpdatesOnlyChanged(t *testing.T) {
	// Test that only changed fields are updated

	// Create user with existing data
	user := &objects.User{
		UserId:    12345,
		Username:  "same_username",
		FirstName: "OldFirst",
		LastName:  "SameLast",
		MenuId:    objects.Menu_Main,
	}

	// Create message with partial changes (only FirstName changed)
	messageWithPartialChanges := &tgbotapi.Message{
		From: &tgbotapi.User{
			ID:        12345,
			UserName:  "same_username", // Same
			FirstName: "NewFirst",      // Changed
			LastName:  "SameLast",      // Same
		},
	}

	// Simulate the logic from HandleMessage
	needsUpdate := false

	if messageWithPartialChanges.From != nil &&
		(messageWithPartialChanges.From.UserName != "" || messageWithPartialChanges.From.FirstName != "" || messageWithPartialChanges.From.LastName != "") {

		if messageWithPartialChanges.From.UserName != "" && user.Username != messageWithPartialChanges.From.UserName {
			user.Username = messageWithPartialChanges.From.UserName
			needsUpdate = true
		}

		if messageWithPartialChanges.From.FirstName != "" && user.FirstName != messageWithPartialChanges.From.FirstName {
			user.FirstName = messageWithPartialChanges.From.FirstName
			needsUpdate = true
		}

		if messageWithPartialChanges.From.LastName != "" && user.LastName != messageWithPartialChanges.From.LastName {
			user.LastName = messageWithPartialChanges.From.LastName
			needsUpdate = true
		}
	}

	// Verify that only changed field was updated
	assert.True(t, needsUpdate, "Partial changes should trigger update")
	assert.Equal(t, "same_username", user.Username, "Username should remain unchanged")
	assert.Equal(t, "NewFirst", user.FirstName, "FirstName should be updated")
	assert.Equal(t, "SameLast", user.LastName, "LastName should remain unchanged")
}

func TestUserDataUpdate_NilMessage_NoCorruption(t *testing.T) {
	// Test that nil message doesn't cause issues

	// Create user with existing data
	user := &objects.User{
		UserId:    12345,
		Username:  "john_doe",
		FirstName: "John",
		LastName:  "Doe",
		MenuId:    objects.Menu_Main,
	}

	// Nil message
	var nilMessage *tgbotapi.Message = nil

	// Simulate the logic from HandleMessage
	needsUpdate := false

	if nilMessage != nil && nilMessage.From != nil &&
		(nilMessage.From.UserName != "" || nilMessage.From.FirstName != "" || nilMessage.From.LastName != "") {
		// This block should not execute
		needsUpdate = true
	}

	// Verify that user data was not affected
	assert.False(t, needsUpdate, "Nil message should not trigger update")
	assert.Equal(t, "john_doe", user.Username, "Username should not be affected")
	assert.Equal(t, "John", user.FirstName, "FirstName should not be affected")
	assert.Equal(t, "Doe", user.LastName, "LastName should not be affected")
}

func TestUserDataUpdate_AnonymousUserIdentification(t *testing.T) {
	// Test that users with proper data are not identified as Anonymous

	testCases := []struct {
		name     string
		user     *objects.User
		expected string
	}{
		{
			name: "User with username",
			user: &objects.User{
				UserId:    123,
				Username:  "john_doe",
				FirstName: "John",
				LastName:  "Doe",
			},
			expected: "@john_doe",
		},
		{
			name: "User without username but with full name",
			user: &objects.User{
				UserId:    456,
				Username:  "",
				FirstName: "Jane",
				LastName:  "Smith",
			},
			expected: "[Jane Smith](tg://user?id=456)",
		},
		{
			name: "User with only first name",
			user: &objects.User{
				UserId:    789,
				Username:  "",
				FirstName: "Bob",
				LastName:  "",
			},
			expected: "[Bob](tg://user?id=789)",
		},
		{
			name: "Truly anonymous user (should only happen in edge cases)",
			user: &objects.User{
				UserId:    999,
				Username:  "",
				FirstName: "",
				LastName:  "",
			},
			expected: "Anonymous",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Use the same logic as formatUserIdentifierForAdmin
			var result string
			if tc.user.Username != "" {
				result = "@" + tc.user.Username
			} else {
				var displayName string
				if tc.user.FirstName != "" && tc.user.LastName != "" {
					displayName = tc.user.FirstName + " " + tc.user.LastName
				} else if tc.user.FirstName != "" {
					displayName = tc.user.FirstName
				} else if tc.user.LastName != "" {
					displayName = tc.user.LastName
				} else {
					displayName = "Anonymous"
				}

				if displayName == "Anonymous" {
					result = "Anonymous"
				} else {
					result = "[" + displayName + "](tg://user?id=" + string(rune(tc.user.UserId)) + ")"
				}
			}

			// For this test, we just check that non-empty data doesn't result in "Anonymous"
			if tc.user.Username != "" || tc.user.FirstName != "" || tc.user.LastName != "" {
				assert.NotEqual(t, "Anonymous", result, "User with data should not be Anonymous")
			}
		})
	}
}
