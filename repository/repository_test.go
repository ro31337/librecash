package repository

import (
	"database/sql"
	_ "github.com/lib/pq"
	"librecash/objects"
	"testing"
)

// setupTestDB creates a real database connection for testing
func setupTestDB(t *testing.T) *sql.DB {
	// Connect to the test PostgreSQL instance (Docker port mapping)
	connStr := "host=localhost port=15433 user=librecash password=librecash dbname=librecash_test sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Logf("Failed to connect to test database: %v", err)
		t.Skip("Database tests require PostgreSQL connection")
		return nil
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		t.Logf("Failed to ping test database: %v", err)
		t.Skip("Database tests require PostgreSQL connection")
		return nil
	}

	return db
}

func TestUserSaveAndFind(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewRepository(db)

	// Create test user
	user := &objects.User{
		UserId:       12345,
		MenuId:       objects.Menu_Init,
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		LanguageCode: "en",
	}

	// Save user
	err := repo.SaveUser(user)
	if err != nil {
		t.Fatalf("Failed to save user: %v", err)
	}

	// Find user
	foundUser := repo.FindUser(12345)
	if foundUser == nil {
		t.Fatal("User not found")
	}

	// Verify user data
	if foundUser.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", foundUser.Username)
	}

	if foundUser.LanguageCode != "en" {
		t.Errorf("Expected language 'en', got '%s'", foundUser.LanguageCode)
	}
}

func TestCalloutTracking(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewRepository(db)

	userId := int64(12345)
	featureName := "test_feature"

	// Create dismissed_feature_callouts table if it doesn't exist
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS dismissed_feature_callouts (
			"userId" BIGINT NOT NULL,
			"featureName" VARCHAR(255) NOT NULL,
			"dismissedAt" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY ("userId", "featureName")
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Clean up any existing test data
	_, err = db.Exec(`DELETE FROM dismissed_feature_callouts WHERE "userId" = $1 AND "featureName" = $2`, userId, featureName)
	if err != nil {
		t.Fatalf("Failed to clean test data: %v", err)
	}

	// Initially should show callout
	show := repo.ShowCallout(userId, featureName)
	if !show {
		t.Error("Expected callout to show initially")
	}

	// Dismiss callout
	err = repo.DismissCallout(userId, featureName)
	if err != nil {
		t.Fatalf("Failed to dismiss callout: %v", err)
	}

	// Should not show after dismissal
	show = repo.ShowCallout(userId, featureName)
	if show {
		t.Error("Expected callout to be hidden after dismissal")
	}

	// Clean up test data
	_, err = db.Exec(`DELETE FROM dismissed_feature_callouts WHERE "userId" = $1 AND "featureName" = $2`, userId, featureName)
	if err != nil {
		t.Logf("Failed to clean up test data: %v", err)
	}
}
