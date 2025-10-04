package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestAllLocalesHaveTranslations checks that all locale files have actual translations
// and not just untranslated message keys
func TestAllLocalesHaveTranslations(t *testing.T) {
	// Pattern to detect untranslated message keys
	// Matches strings like "ask_location_menu.message" or "init_menu.welcome"
	// These are lowercase letters, underscores, dots - typical message key format
	untranslatedPattern := regexp.MustCompile(`^[a-z_\.]{3,1000}$`)

	// Get all locale files
	localeDir := "./locales/all"
	poFiles, err := filepath.Glob(filepath.Join(localeDir, "*.po"))
	if err != nil {
		t.Fatalf("Failed to find .po files: %v", err)
	}

	if len(poFiles) == 0 {
		t.Fatal("No .po files found in locales/all directory")
	}

	// Use en.po as reference for all keys
	referencePath := filepath.Join(localeDir, "en.po")
	referenceEntries, err := parsePOFile(referencePath)
	if err != nil {
		t.Fatalf("Failed to parse reference file en.po: %v", err)
	}

	// Remove empty msgid if present (header entry)
	delete(referenceEntries, "")

	// Get all keys from reference
	var testKeys []string
	for key := range referenceEntries {
		testKeys = append(testKeys, key)
	}

	t.Logf("Testing %d translation keys across all locales", len(testKeys))

	// Track all failures
	var failures []string
	totalKeysChecked := 0
	totalMissingTranslations := 0
	localesWithIssues := make(map[string]int)

	for _, poFile := range poFiles {
		baseName := filepath.Base(poFile)
		langCode := strings.TrimSuffix(baseName, ".po")

		entries, err := parsePOFile(poFile)
		if err != nil {
			t.Errorf("Failed to parse %s: %v", baseName, err)
			continue
		}

		// Remove empty msgid if present (header entry)
		delete(entries, "")

		fmt.Printf("\n=== Testing locale: %s ===\n", langCode)

		missingInThisLocale := 0
		for _, key := range testKeys {
			totalKeysChecked++

			// Get the translation
			translation, exists := entries[key]

			if !exists {
				// Key doesn't exist at all
				totalMissingTranslations++
				missingInThisLocale++

				failureMsg := fmt.Sprintf("Locale %s: Key '%s' is missing entirely",
					langCode, key)
				failures = append(failures, failureMsg)
				fmt.Printf("  ❌ %s\n", failureMsg)
			} else if translation == "" {
				// Empty translation
				totalMissingTranslations++
				missingInThisLocale++

				failureMsg := fmt.Sprintf("Locale %s: Key '%s' has empty translation",
					langCode, key)
				failures = append(failures, failureMsg)
				fmt.Printf("  ❌ %s\n", failureMsg)
			} else if untranslatedPattern.MatchString(translation) {
				// Translation looks like an untranslated key
				totalMissingTranslations++
				missingInThisLocale++

				failureMsg := fmt.Sprintf("Locale %s: Key '%s' returned untranslated value: '%s'",
					langCode, key, translation)
				failures = append(failures, failureMsg)
				fmt.Printf("  ❌ %s\n", failureMsg)
			} else {
				// Show first 50 chars of successful translation for verification
				truncated := translation
				// Replace newlines with spaces for display
				truncated = strings.ReplaceAll(truncated, "\\n", " ")
				if len(truncated) > 50 {
					truncated = truncated[:47] + "..."
				}
				fmt.Printf("  ✅ %s: %s\n", key, truncated)
			}
		}

		if missingInThisLocale > 0 {
			localesWithIssues[langCode] = missingInThisLocale
			fmt.Printf("  🔴 Missing translations in %s: %d/%d\n", langCode, missingInThisLocale, len(testKeys))
		} else {
			fmt.Printf("  ✅ All translations present in %s!\n", langCode)
		}
	}

	// Report summary
	fmt.Printf("\n=== SUMMARY ===\n")
	fmt.Printf("Total locales tested: %d\n", len(poFiles))
	fmt.Printf("Total keys checked: %d\n", totalKeysChecked)
	fmt.Printf("Total missing translations: %d\n", totalMissingTranslations)
	if totalKeysChecked > 0 {
		fmt.Printf("Failure rate: %.2f%%\n", float64(totalMissingTranslations)/float64(totalKeysChecked)*100)
	}

	if len(localesWithIssues) > 0 {
		fmt.Printf("\nLocales with missing translations:\n")
		for locale, count := range localesWithIssues {
			fmt.Printf("  - %s: %d missing\n", locale, count)
		}
	}

	// Fail the test if any translations are missing
	if len(failures) > 0 {
		fmt.Printf("\n=== ALL FAILURES ===\n")
		for _, failure := range failures {
			fmt.Println(failure)
		}
		t.Errorf("\n🔴 Test FAILED: Found %d missing translations across %d locales",
			totalMissingTranslations, len(localesWithIssues))
	} else {
		fmt.Printf("\n✅ All translations are present in all locales!\n")
	}
}
