package menu

import (
	"fmt"
	"librecash/objects"
	"testing"

	"github.com/leonelquinteros/gotext"
	"github.com/stretchr/testify/assert"
)

// TestLocalizedPhoneLabels tests that phone labels are properly localized
// across all supported languages
func TestLocalizedPhoneLabels(t *testing.T) {
	testCases := []struct {
		language      string
		expectedPhone string
	}{
		{"en", "Phone: +1234567890"},
		{"ru", "Телефон: +1234567890"},
		{"es", "Teléfono: +1234567890"},
		{"fr", "Téléphone: +1234567890"},
		{"de", "Telefon: +1234567890"},
		{"it", "Telefono: +1234567890"},
		{"pt", "Telefone: +1234567890"},
		{"hi", "फ़ोन: +1234567890"},
		{"tr", "Telefon: +1234567890"},
		{"ar", "هاتف: +1234567890"},
		{"vi", "Điện thoại: +1234567890"},
		{"zh", "电话: +1234567890"},
	}

	user := &objects.User{
		UserId:      123,
		Username:    "test_user",
		FirstName:   "Test",
		LastName:    "User",
		PhoneNumber: "+1234567890",
	}

	for _, tc := range testCases {
		t.Run("Language_"+tc.language, func(t *testing.T) {
			// Test formatUserIdentifier with phone included
			result := formatUserIdentifier(user, true, tc.language)

			// Should contain the username
			assert.Contains(t, result, "@test_user", "Should contain username")

			// Should contain the localized phone label
			assert.Contains(t, result, tc.expectedPhone, "Should contain localized phone label for %s", tc.language)

			// Should be in format: "@username\nPhone: +number"
			expected := "@test_user\n" + tc.expectedPhone
			assert.Equal(t, expected, result, "Should match expected format for %s", tc.language)
		})
	}
}

// TestContactLabelsCapitalization tests that contact labels use proper capitalization
func TestContactLabelsCapitalization(t *testing.T) {
	testCases := []struct {
		language        string
		expectedContact string
	}{
		{"en", "Contact: @test_user"},
		{"ru", "Контакт: @test_user"},
		{"es", "Contacto: @test_user"},
		{"fr", "Contact: @test_user"},
		{"de", "Kontakt: @test_user"},
	}

	for _, tc := range testCases {
		t.Run("Language_"+tc.language, func(t *testing.T) {
			// Use our getTranslation function to get the contact template
			// (This handles the path issues correctly)
			po := gotext.NewPo()
			poFile := fmt.Sprintf("../locales/all/%s.po", tc.language)
			po.ParseFile(poFile)

			contactTemplate := po.Get("contact_request.contact_info")

			// Should not be all caps (no "CONTACT:" or "КОНТАКТ:")
			assert.NotContains(t, contactTemplate, "CONTACT:", "Should not contain all-caps CONTACT")
			assert.NotContains(t, contactTemplate, "КОНТАКТ:", "Should not contain all-caps КОНТАКТ")

			// Should contain proper capitalization and placeholder
			if contactTemplate != "contact_request.contact_info" {
				assert.Contains(t, contactTemplate, "%s", "Should contain placeholder")
			}

			t.Logf("Language %s: contact template = '%s'", tc.language, contactTemplate)
		})
	}
}

// TestNoHardcodedLabels tests that no hardcoded labels remain in the code
func TestNoHardcodedLabels(t *testing.T) {
	user := &objects.User{
		UserId:      123,
		Username:    "test_user",
		PhoneNumber: "+1234567890",
	}

	// Test with phone included
	result := formatUserIdentifier(user, true, "en")

	// Should NOT contain hardcoded labels
	assert.NotContains(t, result, "PHONE:", "Should not contain hardcoded PHONE:")
	assert.NotContains(t, result, "CONTACT:", "Should not contain hardcoded CONTACT:")

	// Should contain proper localized labels
	assert.Contains(t, result, "Phone:", "Should contain localized Phone:")

	t.Logf("Result: %s", result)
}
