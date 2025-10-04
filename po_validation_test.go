package main

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/leonelquinteros/gotext"
)

// TestPOFilesCanBeParsed verifies that all .po files can be properly parsed by gotext
// This catches issues like unescaped quotes that prevent translations from loading
func TestPOFilesCanBeParsed(t *testing.T) {
	localeDir := "./locales/all"
	poFiles, err := filepath.Glob(filepath.Join(localeDir, "*.po"))
	if err != nil {
		t.Fatalf("Failed to find .po files: %v", err)
	}

	if len(poFiles) == 0 {
		t.Fatal("No .po files found in locales/all directory")
	}

	fmt.Printf("Testing critical translations across %d locale files\n", len(poFiles))

	totalFailures := 0
	for _, poFile := range poFiles {
		baseName := filepath.Base(poFile)
		langCode := baseName[:len(baseName)-3] // Remove .po extension

		// Test using gotext.Po (this is what the actual app uses)
		po := gotext.NewPo()
		po.ParseFile(poFile)

		failuresInFile := 0

		// Test specific keys that are critical and have had issues
		// Test 1: ask_location_menu.message (the problematic one)
		const keyLocationMsg = "ask_location_menu.message"
		translation1 := po.Get(keyLocationMsg)
		if translation1 == keyLocationMsg {
			failuresInFile++
			totalFailures++
			t.Errorf("❌ %s: Key '%s' cannot be loaded by gotext", langCode, keyLocationMsg)
		}

		// Test 2: init_menu.welcome (multiline message)
		const keyInitWelcome = "init_menu.welcome"
		translation2 := po.Get(keyInitWelcome)
		if translation2 == keyInitWelcome {
			failuresInFile++
			totalFailures++
			t.Errorf("❌ %s: Key '%s' cannot be loaded by gotext", langCode, keyInitWelcome)
		}

		// Test 3: language.changed (with parameter)
		const keyLangChanged = "language.changed"
		// Don't test with parameter - just check if key loads
		translation3 := po.Get(keyLangChanged)
		if translation3 == keyLangChanged {
			failuresInFile++
			totalFailures++
			t.Errorf("❌ %s: Key '%s' cannot be loaded by gotext", langCode, keyLangChanged)
		}

		if failuresInFile == 0 {
			fmt.Printf("✅ %s: Critical translations can be loaded by gotext\n", langCode)
		} else {
			fmt.Printf("❌ %s: %d critical translations CANNOT be loaded by gotext\n", langCode, failuresInFile)
		}
	}

	if totalFailures > 0 {
		t.Errorf("\n🔴 CRITICAL: %d translations cannot be loaded by gotext!", totalFailures)
		t.Error("This means users will see untranslated message keys instead of proper translations!")
	} else {
		fmt.Printf("\n✅ SUCCESS: All critical translations in all %d locales can be loaded by gotext\n", len(poFiles))
	}
}
