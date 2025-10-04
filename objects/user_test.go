package objects

import (
	"testing"
)

func TestGetSupportedLanguageCode(t *testing.T) {
	tests := []struct {
		name     string
		langCode string
		expected string
	}{
		// Original 10 languages
		{"English", "en", "en"},
		{"Russian", "ru", "ru"},
		{"Indonesian", "id", "id"},
		{"Portuguese", "pt", "pt"},
		{"Portuguese Brazil", "pt-br", "pt"},
		{"Spanish", "es", "es"},
		{"Hindi", "hi", "hi"},
		{"Turkish", "tr", "tr"},
		{"Arabic", "ar", "ar"},
		{"Vietnamese", "vi", "vi"},
		{"French", "fr", "fr"},

		// New 17 languages
		{"Persian", "fa", "fa"},
		{"Ukrainian", "uk", "uk"},
		{"Kazakh", "kk", "kk"},
		{"Italian", "it", "it"},
		{"German", "de", "de"},
		{"Hebrew", "he", "he"},
		{"Thai", "th", "th"},
		{"Burmese", "my", "my"},
		{"Azerbaijani", "az", "az"},
		{"Bulgarian", "bg", "bg"},
		{"Romanian", "ro", "ro"},
		{"Polish", "pl", "pl"},
		{"Chinese Simplified", "zh-CN", "zh-CN"},
		{"Chinese Traditional Taiwan", "zh-TW", "zh-TW"},
		{"Chinese Traditional HK", "zh-HK", "zh-HK"},
		{"Chinese Fallback", "zh", "zh"},

		// Chinese variant tests
		{"Chinese Simplified lowercase", "zh-cn", "zh-CN"},
		{"Chinese Traditional TW lowercase", "zh-tw", "zh-TW"},
		{"Chinese HK lowercase", "zh-hk", "zh-HK"},
		{"Chinese Simplified script", "zh-hans", "zh-CN"},
		{"Chinese Traditional script", "zh-hant", "zh-TW"},
		{"Chinese Unknown variant", "zh-mo", "zh"},

		// Portuguese variant tests
		{"Portuguese Brazil", "pt-br", "pt"},
		{"Portuguese Brazil lowercase", "pt-br", "pt"},
		{"Portuguese Portugal", "pt-pt", "pt"},
		{"Portuguese Angola", "pt-ao", "pt"},
		{"Portuguese Mozambique", "pt-mz", "pt"},
		{"Portuguese Unknown variant", "pt-xyz", "pt"},
		{"Filipino", "fil", "fil"},

		// Unsupported languages
		{"Japanese", "ja", "en"},
		{"Korean", "ko", "en"},
		{"Unknown", "xyz", "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				LanguageCode: tt.langCode,
			}
			result := user.GetSupportedLanguageCode()
			if result != tt.expected {
				t.Errorf("GetSupportedLanguageCode() for %s = %v, want %v", tt.langCode, result, tt.expected)
			}
		})
	}
}

func TestGetLanguageName(t *testing.T) {
	tests := []struct {
		name         string
		langCode     string
		expectedName string
	}{
		// Original languages
		{"English", "en", "English"},
		{"Russian", "ru", "Russian"},
		{"Indonesian", "id", "Indonesian"},
		{"Portuguese", "pt", "Portuguese"},
		{"Spanish", "es", "Spanish"},
		{"Hindi", "hi", "Hindi"},
		{"Turkish", "tr", "Turkish"},
		{"Arabic", "ar", "Arabic"},
		{"Vietnamese", "vi", "Vietnamese"},
		{"French", "fr", "French"},

		// New languages
		{"Persian", "fa", "Persian"},
		{"Ukrainian", "uk", "Ukrainian"},
		{"Kazakh", "kk", "Kazakh"},
		{"Italian", "it", "Italian"},
		{"German", "de", "German"},
		{"Hebrew", "he", "Hebrew"},
		{"Thai", "th", "Thai"},
		{"Burmese", "my", "Burmese"},
		{"Azerbaijani", "az", "Azerbaijani"},
		{"Bulgarian", "bg", "Bulgarian"},
		{"Romanian", "ro", "Romanian"},
		{"Polish", "pl", "Polish"},
		{"Chinese", "zh", "Chinese"},
		{"Chinese Simplified", "zh-CN", "Chinese (Simplified)"},
		{"Chinese Traditional TW", "zh-TW", "Chinese (Traditional)"},
		{"Chinese Traditional HK", "zh-HK", "Chinese (Hong Kong)"},
		{"Filipino", "fil", "Filipino"},

		// Unsupported (should return English)
		{"Unknown", "xyz", "English"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				LanguageCode: tt.langCode,
			}
			result := user.GetLanguageName()
			if result != tt.expectedName {
				t.Errorf("GetLanguageName() for %s = %v, want %v", tt.langCode, result, tt.expectedName)
			}
		})
	}
}
