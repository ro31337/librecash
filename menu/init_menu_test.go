package menu

import (
	"librecash/objects"
	"testing"
)

func TestLanguageDetection(t *testing.T) {
	tests := []struct {
		name         string
		languageCode string
		expected     string
		expectedName string
	}{
		{"English", "en", "en", "English"},
		{"Russian", "ru", "ru", "Russian"},
		{"Spanish", "es", "es", "Spanish"},
		{"Portuguese Brazil", "pt-br", "pt", "Portuguese"},
		{"Portuguese Portugal", "pt-pt", "pt", "Portuguese"},
		{"Indonesian", "id", "id", "Indonesian"},
		{"Hindi", "hi", "hi", "Hindi"},
		{"Turkish", "tr", "tr", "Turkish"},
		{"Arabic", "ar", "ar", "Arabic"},
		{"Vietnamese", "vi", "vi", "Vietnamese"},
		{"French", "fr", "fr", "French"},
		{"German", "de", "de", "German"},
		{"Chinese", "zh", "zh", "Chinese"},
		{"Empty", "", "en", "English"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &objects.User{
				UserId:       123,
				LanguageCode: tt.languageCode,
			}

			got := user.GetSupportedLanguageCode()
			if got != tt.expected {
				t.Errorf("GetSupportedLanguageCode() = %v, want %v", got, tt.expected)
			}

			gotName := user.GetLanguageName()
			if gotName != tt.expectedName {
				t.Errorf("GetLanguageName() = %v, want %v", gotName, tt.expectedName)
			}
		})
	}
}

func TestMenuStates(t *testing.T) {
	// Test that menu states are correctly defined
	if objects.Menu_Init != 100 {
		t.Errorf("Menu_Init should be 100, got %d", objects.Menu_Init)
	}

	if objects.Menu_AskLocation != 200 {
		t.Errorf("Menu_AskLocation should be 200, got %d", objects.Menu_AskLocation)
	}

	if objects.Menu_Ban != 999999 {
		t.Errorf("Menu_Ban should be 999999, got %d", objects.Menu_Ban)
	}
}

func TestFormatUserIdentifierForAdmin(t *testing.T) {
	tests := []struct {
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
			name: "User without username - full name",
			user: &objects.User{
				UserId:    456,
				Username:  "",
				FirstName: "Jane",
				LastName:  "Smith",
			},
			expected: `<a href="tg://user?id=456">Jane Smith</a>`,
		},
		{
			name: "User without username - first name only",
			user: &objects.User{
				UserId:    789,
				Username:  "",
				FirstName: "Bob",
				LastName:  "",
			},
			expected: `<a href="tg://user?id=789">Bob</a>`,
		},
		{
			name: "User without username - last name only",
			user: &objects.User{
				UserId:    101112,
				Username:  "",
				FirstName: "",
				LastName:  "Wilson",
			},
			expected: `<a href="tg://user?id=101112">Wilson</a>`,
		},
		{
			name: "User without username or names",
			user: &objects.User{
				UserId:    131415,
				Username:  "",
				FirstName: "",
				LastName:  "",
			},
			expected: `<a href="tg://user?id=131415">Anonymous</a>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use universal function with includePhone=false for admin
			result := formatUserIdentifier(tt.user, false, "en")
			if result != tt.expected {
				t.Errorf("formatUserIdentifier(user, false) = %v, want %v", result, tt.expected)
			}

			// Ensure no user ID is shown in the display name (should not contain "(ID: ")
			if tt.user.Username == "" && result != tt.expected {
				t.Errorf("Admin identifier should not contain user ID in display name")
			}
		})
	}
}
