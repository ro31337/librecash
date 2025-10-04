package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// POEntry represents a single entry in a .po file
type POEntry struct {
	msgid  string
	msgstr string
}

// parsePOFile parses a .po file and returns all entries
func parsePOFile(filename string) (map[string]string, error) {
	entries := make(map[string]string)

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentMsgid string
	var currentMsgstr string
	inMsgid := false
	inMsgstr := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip comments and empty lines
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Handle msgid
		if strings.HasPrefix(trimmed, "msgid ") {
			// Save previous entry if exists
			if currentMsgid != "" {
				entries[currentMsgid] = currentMsgstr
			}

			// Start new entry
			currentMsgid = extractQuotedString(trimmed[6:])
			currentMsgstr = ""
			inMsgid = true
			inMsgstr = false
		} else if strings.HasPrefix(trimmed, "msgstr ") {
			// Handle msgstr
			currentMsgstr = extractQuotedString(trimmed[7:])
			inMsgid = false
			inMsgstr = true
		} else if strings.HasPrefix(trimmed, "\"") {
			// Handle continuation lines
			continuation := extractQuotedString(trimmed)
			if inMsgid {
				currentMsgid += continuation
			} else if inMsgstr {
				currentMsgstr += continuation
			}
		}
	}

	// Save last entry
	if currentMsgid != "" {
		entries[currentMsgid] = currentMsgstr
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

// extractQuotedString extracts the string from quotes
func extractQuotedString(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// TestLocaleConsistency ensures all locale files have the same keys
func TestLocaleConsistency(t *testing.T) {
	localesDir := "locales/all"

	// Get all .po files
	poFiles, err := filepath.Glob(filepath.Join(localesDir, "*.po"))
	if err != nil {
		t.Fatalf("Failed to find .po files: %v", err)
	}

	if len(poFiles) == 0 {
		t.Fatal("No .po files found in locales/all directory")
	}

	// Use en.po as reference
	referencePath := filepath.Join(localesDir, "en.po")
	referenceEntries, err := parsePOFile(referencePath)
	if err != nil {
		t.Fatalf("Failed to parse reference file en.po: %v", err)
	}

	// Remove empty msgid if present (header entry)
	delete(referenceEntries, "")

	t.Logf("Reference locale (en.po) has %d keys", len(referenceEntries))

	// Track all errors
	var allErrors []string

	// Check each locale file
	for _, poFile := range poFiles {
		baseName := filepath.Base(poFile)

		// Skip the reference file
		if baseName == "en.po" {
			continue
		}

		entries, err := parsePOFile(poFile)
		if err != nil {
			t.Errorf("Failed to parse %s: %v", baseName, err)
			continue
		}

		// Remove empty msgid if present (header entry)
		delete(entries, "")

		// Check for missing keys
		var missingKeys []string
		var emptyTranslations []string

		for msgid, msgstr := range referenceEntries {
			if _, exists := entries[msgid]; !exists {
				missingKeys = append(missingKeys, msgid)
			} else if entries[msgid] == "" && msgstr != "" {
				// Only flag as empty if reference also isn't empty
				emptyTranslations = append(emptyTranslations, msgid)
			}
		}

		// Check for extra keys
		var extraKeys []string
		for msgid := range entries {
			if _, exists := referenceEntries[msgid]; !exists {
				extraKeys = append(extraKeys, msgid)
			}
		}

		// Report errors for this locale
		if len(missingKeys) > 0 {
			errorMsg := fmt.Sprintf("\n%s is missing %d keys:", baseName, len(missingKeys))
			for _, key := range missingKeys {
				errorMsg += fmt.Sprintf("\n  - %s", key)
			}
			allErrors = append(allErrors, errorMsg)
		}

		if len(emptyTranslations) > 0 {
			errorMsg := fmt.Sprintf("\n%s has %d empty translations:", baseName, len(emptyTranslations))
			for _, key := range emptyTranslations {
				errorMsg += fmt.Sprintf("\n  - %s", key)
			}
			allErrors = append(allErrors, errorMsg)
		}

		if len(extraKeys) > 0 {
			errorMsg := fmt.Sprintf("\n%s has %d extra keys not in reference:", baseName, len(extraKeys))
			for _, key := range extraKeys {
				errorMsg += fmt.Sprintf("\n  - %s", key)
			}
			allErrors = append(allErrors, errorMsg)
		}

		// Log success for consistent files
		if len(missingKeys) == 0 && len(emptyTranslations) == 0 && len(extraKeys) == 0 {
			t.Logf("✓ %s is consistent with reference (has %d keys)", baseName, len(entries))
		}
	}

	// Report all errors at once
	if len(allErrors) > 0 {
		t.Errorf("\nLocale consistency check failed! Found %d locale(s) with issues:%s",
			len(allErrors), strings.Join(allErrors, "\n"))
		t.Logf("\nTotal locales checked: %d", len(poFiles)-1) // -1 for reference
		t.Fatal("Fix the above locale inconsistencies before proceeding")
	}

	t.Logf("\n✅ All %d locale files are consistent!", len(poFiles))
}

// TestRequiredTranslationKeys ensures critical keys exist in all locales
func TestRequiredTranslationKeys(t *testing.T) {
	// Define keys that MUST exist in all locales
	requiredKeys := []string{
		"init_menu.welcome",
		"admin.new_user",
		// Add new required keys here as we implement features
		// "ask_location_menu.message",
		// "ask_location_menu.next_button",
	}

	localesDir := "locales/all"
	poFiles, err := filepath.Glob(filepath.Join(localesDir, "*.po"))
	if err != nil {
		t.Fatalf("Failed to find .po files: %v", err)
	}

	var allErrors []string

	for _, poFile := range poFiles {
		baseName := filepath.Base(poFile)
		entries, err := parsePOFile(poFile)
		if err != nil {
			t.Errorf("Failed to parse %s: %v", baseName, err)
			continue
		}

		var missingRequired []string
		for _, key := range requiredKeys {
			if _, exists := entries[key]; !exists {
				missingRequired = append(missingRequired, key)
			} else if entries[key] == "" {
				missingRequired = append(missingRequired, key+" (empty)")
			}
		}

		if len(missingRequired) > 0 {
			errorMsg := fmt.Sprintf("\n%s missing required keys: %v", baseName, missingRequired)
			allErrors = append(allErrors, errorMsg)
		}
	}

	if len(allErrors) > 0 {
		t.Errorf("\nRequired translation keys check failed:%s", strings.Join(allErrors, "\n"))
	}
}
