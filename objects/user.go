package objects

import (
	"fmt"
	"log"
	"strings"

	"github.com/leonelquinteros/gotext"
)

type MenuId int

const (
	Menu_USComplianceCheck       MenuId = 50 // US persons compliance check (first menu)
	Menu_Blocked                 MenuId = 60 // Blocked users (terminal state)
	Menu_Init                    MenuId = 100
	Menu_AskLocation             MenuId = 200 // Ask for user's location
	Menu_SelectRadius            MenuId = 250 // Ask for search radius preference
	Menu_AskPhone                MenuId = 275 // Ask for phone number (optional)
	Menu_HistoricalFanoutExecute MenuId = 290 // Historical fanout execute menu (renamed from Menu_HistoricalFanout)
	Menu_HistoricalFanoutWait    MenuId = 295 // Historical fanout wait menu
	Menu_Main                    MenuId = 400 // Main exchange selection menu
	Menu_Amount                  MenuId = 500 // Select exchange amount
	Menu_Ban                     MenuId = 999999
)

type User struct {
	UserId         int64
	MenuId         MenuId
	Username       string
	FirstName      string
	LastName       string
	LanguageCode   string
	Lon            float64    // Longitude
	Lat            float64    // Latitude
	SearchRadiusKm *int       // Search radius in kilometers (nullable)
	PhoneNumber    string     // Phone number (optional)
	po             *gotext.Po // Direct Po object for translations
}

// GetSupportedLanguageCode returns the supported language code for the user
// Falls back to English if the user's language is not supported
func (u *User) GetSupportedLanguageCode() string {
	supportedLanguages := map[string]bool{
		"en":    true, // English
		"ru":    true, // Russian
		"id":    true, // Indonesian
		"pt":    true, // Portuguese (will match pt-br and pt-pt)
		"es":    true, // Spanish
		"hi":    true, // Hindi
		"tr":    true, // Turkish
		"ar":    true, // Arabic
		"vi":    true, // Vietnamese
		"fr":    true, // French
		"fa":    true, // Persian/Farsi
		"uk":    true, // Ukrainian
		"kk":    true, // Kazakh
		"it":    true, // Italian
		"de":    true, // German
		"he":    true, // Hebrew
		"th":    true, // Thai
		"my":    true, // Burmese
		"az":    true, // Azerbaijani
		"bg":    true, // Bulgarian
		"ro":    true, // Romanian
		"pl":    true, // Polish
		"zh":    true, // Chinese (fallback)
		"zh-CN": true, // Chinese Simplified
		"zh-TW": true, // Chinese Traditional (Taiwan)
		"zh-HK": true, // Chinese Traditional (Hong Kong)
		"fil":   true, // Filipino
	}

	// Chinese and Portuguese-specific mappings for case and script variations
	lang := strings.ToLower(u.LanguageCode)
	chineseMappings := map[string]string{
		"zh-cn":   "zh-CN",
		"zh-tw":   "zh-TW",
		"zh-hk":   "zh-HK",
		"zh-hans": "zh-CN", // Script → Simplified
		"zh-hant": "zh-TW", // Script → Traditional
	}

	portugueseMappings := map[string]string{
		"pt-br": "pt", // Brazil → Portuguese
		"pt-pt": "pt", // Portugal → Portuguese
		"pt-ao": "pt", // Angola → Portuguese
		"pt-mz": "pt", // Mozambique → Portuguese
	}

	if mapped, exists := chineseMappings[lang]; exists {
		return mapped
	}

	if mapped, exists := portugueseMappings[lang]; exists {
		return mapped
	}

	// Check exact match first (case-sensitive for non-Chinese)
	if supportedLanguages[u.LanguageCode] {
		return u.LanguageCode
	}

	// Check language family (e.g., pt-br -> pt)
	if len(u.LanguageCode) >= 2 {
		langFamily := u.LanguageCode[:2]
		if supportedLanguages[langFamily] {
			return langFamily
		}
	}

	// Chinese wildcard fallback for any other zh-* variants
	if strings.HasPrefix(lang, "zh-") {
		return "zh"
	}

	// Portuguese wildcard fallback for any other pt-* variants
	if strings.HasPrefix(lang, "pt-") {
		return "pt"
	}

	// Default to English
	log.Printf("[USER] Language '%s' not supported, defaulting to English", u.LanguageCode)
	return "en"
}

// GetLanguageName returns the human-readable name of the language
func (u *User) GetLanguageName() string {
	languageNames := map[string]string{
		"en":    "English",
		"ru":    "Russian",
		"id":    "Indonesian",
		"pt":    "Portuguese",
		"es":    "Spanish",
		"hi":    "Hindi",
		"tr":    "Turkish",
		"ar":    "Arabic",
		"vi":    "Vietnamese",
		"fr":    "French",
		"fa":    "Persian",
		"uk":    "Ukrainian",
		"kk":    "Kazakh",
		"it":    "Italian",
		"de":    "German",
		"he":    "Hebrew",
		"th":    "Thai",
		"my":    "Burmese",
		"az":    "Azerbaijani",
		"bg":    "Bulgarian",
		"ro":    "Romanian",
		"pl":    "Polish",
		"zh":    "Chinese",
		"zh-CN": "Chinese (Simplified)",
		"zh-TW": "Chinese (Traditional)",
		"zh-HK": "Chinese (Hong Kong)",
		"fil":   "Filipino",
	}

	lang := u.GetSupportedLanguageCode()
	if name, ok := languageNames[lang]; ok {
		return name
	}
	return "English"
}

// Locale returns the gotext Po for the user (not actually a Locale, but Po has the same Get() method)
func (u *User) Locale() *gotext.Po {
	if u.po == nil {
		lang := u.GetSupportedLanguageCode()
		log.Printf("[USER] Loading locale for user %d: %s", u.UserId, lang)

		// Use Po directly instead of Locale, since our files are in ./locales/all/*.po
		// not in the gotext expected structure of ./locales/LANG/LC_MESSAGES/DOMAIN.po
		u.po = gotext.NewPo()
		poFile := fmt.Sprintf("./locales/all/%s.po", lang)
		u.po.ParseFile(poFile)

		log.Printf("[USER] Loaded po file: %s", poFile)
	}
	return u.po
}
