package repository

import (
	"database/sql"
	"librecash/objects"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCheckContactRequestExists(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Database tests require PostgreSQL connection")
		return
	}
	defer db.Close()
	repo := NewRepository(db)

	// Clean up any existing data in correct order (child tables first)
	_, err := db.Exec("DELETE FROM contact_requests")
	assert.NoError(t, err)
	_, err = db.Exec("DELETE FROM timeline_records")
	assert.NoError(t, err)
	_, err = db.Exec("DELETE FROM exchanges")
	assert.NoError(t, err)
	_, err = db.Exec("DELETE FROM location_histories")
	assert.NoError(t, err)
	_, err = db.Exec("DELETE FROM users")
	assert.NoError(t, err)

	// Create test users first
	testUser1 := &objects.User{UserId: 123, Username: "requester", FirstName: "Test", LastName: "User", LanguageCode: "en"}
	testUser2 := &objects.User{UserId: 999, Username: "initiator", FirstName: "Test", LastName: "Initiator", LanguageCode: "en"}
	err = repo.SaveUser(testUser1)
	assert.NoError(t, err)
	err = repo.SaveUser(testUser2)
	assert.NoError(t, err)

	// Create test exchange
	amount := 50
	exchange := &objects.Exchange{
		UserID:            999,
		ExchangeDirection: "cash_to_crypto",
		Status:            "posted",
		AmountUSD:         &amount,
		Lat:               40.7128,
		Lon:               -74.006,
	}
	err = repo.CreateExchange(exchange)
	assert.NoError(t, err)
	exchangeID := exchange.ID

	// Test non-existent contact request
	exists, err := repo.CheckContactRequestExists(exchangeID, 123)
	assert.NoError(t, err)
	assert.False(t, exists, "Should not find non-existent contact request")

	// Create a contact request using the repository method
	err = repo.CreateContactRequest(exchangeID, 123, "requester", "Test", "User")
	assert.NoError(t, err)

	// Test existing contact request
	exists, err = repo.CheckContactRequestExists(exchangeID, 123)
	assert.NoError(t, err)
	assert.True(t, exists, "Should find existing contact request")

	// Test different user for same exchange
	exists, err = repo.CheckContactRequestExists(exchangeID, 456)
	assert.NoError(t, err)
	assert.False(t, exists, "Should not find contact request for different user")
}

func TestCreateContactRequest(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Database tests require PostgreSQL connection")
		return
	}
	defer db.Close()
	repo := NewRepository(db)

	// Clean up any existing data in correct order (child tables first)
	_, err := db.Exec("DELETE FROM contact_requests")
	assert.NoError(t, err)
	_, err = db.Exec("DELETE FROM timeline_records")
	assert.NoError(t, err)
	_, err = db.Exec("DELETE FROM exchanges")
	assert.NoError(t, err)
	_, err = db.Exec("DELETE FROM location_histories")
	assert.NoError(t, err)
	_, err = db.Exec("DELETE FROM users")
	assert.NoError(t, err)

	// Create test users first
	testUser1 := &objects.User{UserId: 789, Username: "requester", FirstName: "Test", LastName: "Requester", LanguageCode: "en"}
	testUser2 := &objects.User{UserId: 123, Username: "initiator", FirstName: "Test", LastName: "Initiator", LanguageCode: "en"}
	err = repo.SaveUser(testUser1)
	assert.NoError(t, err)
	err = repo.SaveUser(testUser2)
	assert.NoError(t, err)

	// Create test exchange
	amount := 50
	exchange := &objects.Exchange{
		UserID:            123,
		ExchangeDirection: "cash_to_crypto",
		Status:            "posted",
		AmountUSD:         &amount,
		Lat:               40.7128,
		Lon:               -74.006,
	}
	err = repo.CreateExchange(exchange)
	assert.NoError(t, err)
	exchangeID := exchange.ID

	// Test creating contact request
	err = repo.CreateContactRequest(exchangeID, 789, "requester", "Test", "Requester")
	assert.NoError(t, err)

	// Verify contact request was created
	exists, err := repo.CheckContactRequestExists(exchangeID, 789)
	assert.NoError(t, err)
	assert.True(t, exists, "Contact request should exist after creation")

	// Verify the data was stored correctly
	var username, firstName, lastName string
	var requestedAt time.Time
	err = db.QueryRow(`
		SELECT requester_username, requester_first_name, requester_last_name, requested_at
		FROM contact_requests
		WHERE exchange_id = $1 AND requester_user_id = 789
	`, exchangeID).Scan(&username, &firstName, &lastName, &requestedAt)
	assert.NoError(t, err)
	assert.Equal(t, "requester", username)
	assert.Equal(t, "Test", firstName)
	assert.Equal(t, "Requester", lastName)
	assert.True(t, time.Since(requestedAt) < time.Minute, "Request should be recent")

	// Test duplicate creation (should not error but also not create duplicate)
	err = repo.CreateContactRequest(exchangeID, 789, "requester", "Test", "Requester")
	assert.NoError(t, err, "Duplicate contact request should not error")

	// Verify still only one record exists
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM contact_requests
		WHERE exchange_id = $1 AND requester_user_id = 789
	`, exchangeID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "Should have exactly one contact request record")
}

func TestCreateContactRequestConcurrency(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Database tests require PostgreSQL connection")
		return
	}
	defer db.Close()
	repo := NewRepository(db)

	// Clean up any existing data
	_, err := db.Exec("DELETE FROM contact_requests")
	assert.NoError(t, err)
	_, err = db.Exec("DELETE FROM exchanges")
	assert.NoError(t, err)
	_, err = db.Exec("DELETE FROM location_histories")
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM users WHERE "userId" = 123`)
	assert.NoError(t, err)

	// Create test user and exchange
	_, err = db.Exec(`
		INSERT INTO users ("userId", "menuId", "username", "firstName", "lastName", "languageCode")
		VALUES (123, 1, 'testuser', 'Test', 'User', 'en')
	`)
	assert.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO exchanges (id, user_id, exchange_direction, status, lat, lon)
		VALUES (456, 123, 'cash_to_crypto', 'posted', 40.7128, -74.0060)
	`)
	assert.NoError(t, err)

	// Test concurrent contact request creation with REAL race condition
	// This tests the SELECT FOR UPDATE logic with multiple goroutines
	numGoroutines := 10
	done := make(chan error, numGoroutines)

	// Launch multiple goroutines trying to create the same contact request
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			// Each goroutine tries to create the same contact request
			err := repo.CreateContactRequest(456, 789, "concurrent_user", "Concurrent", "User")
			done <- err
		}(i)
	}

	// Collect all results
	var errors []error
	for i := 0; i < numGoroutines; i++ {
		err := <-done
		if err != nil {
			errors = append(errors, err)
		}
	}

	// All should complete without error (duplicates are handled gracefully)
	assert.Empty(t, errors, "No goroutines should have errors: %v", errors)

	// Verify only ONE record exists despite multiple concurrent attempts
	var count int
	err = db.QueryRow(
		`SELECT COUNT(*) FROM contact_requests WHERE exchange_id = 456 AND requester_user_id = 789`,
	).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "Should have exactly one contact request record despite %d concurrent attempts", numGoroutines)

	// Verify the record has correct data
	var username, firstName, lastName string
	err = db.QueryRow(`
		SELECT requester_username, requester_first_name, requester_last_name
		FROM contact_requests
		WHERE exchange_id = 456 AND requester_user_id = 789
	`).Scan(&username, &firstName, &lastName)
	assert.NoError(t, err)
	assert.Equal(t, "concurrent_user", username)
	assert.Equal(t, "Concurrent", firstName)
	assert.Equal(t, "User", lastName)
}

func TestCreateContactRequestSQLSyntax(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Database tests require PostgreSQL connection")
		return
	}
	defer db.Close()

	// Clean up in correct order
	_, err := db.Exec("DELETE FROM contact_requests")
	assert.NoError(t, err)
	_, err = db.Exec("DELETE FROM timeline_records")
	assert.NoError(t, err)
	_, err = db.Exec("DELETE FROM exchanges")
	assert.NoError(t, err)
	_, err = db.Exec("DELETE FROM location_histories")
	assert.NoError(t, err)
	_, err = db.Exec("DELETE FROM users")
	assert.NoError(t, err)

	repo := NewRepository(db)

	// Create test users first
	testUser1 := &objects.User{UserId: 123, Username: "testuser", FirstName: "Test", LastName: "User", LanguageCode: "en"}
	testUser2 := &objects.User{UserId: 999, Username: "initiator", FirstName: "Test", LastName: "Initiator", LanguageCode: "en"}
	err = repo.SaveUser(testUser1)
	assert.NoError(t, err)
	err = repo.SaveUser(testUser2)
	assert.NoError(t, err)

	// Create test exchange
	amount := 50
	exchange := &objects.Exchange{
		UserID:            999,
		ExchangeDirection: "cash_to_crypto",
		Status:            "posted",
		AmountUSD:         &amount,
		Lat:               40.7128,
		Lon:               -74.006,
	}
	err = repo.CreateExchange(exchange)
	assert.NoError(t, err)
	exchangeID := exchange.ID

	// Test that the SQL query with FOR UPDATE works correctly
	// This is the exact query from CreateContactRequest method
	tx, err := db.Begin()
	assert.NoError(t, err)
	defer tx.Rollback()

	// Test the SELECT FOR UPDATE query that was failing before
	var existingID sql.NullInt64
	err = tx.QueryRow(`
		SELECT id FROM contact_requests
		WHERE exchange_id = $1 AND requester_user_id = $2
		FOR UPDATE
	`, exchangeID, int64(123)).Scan(&existingID)

	// Should get sql.ErrNoRows, not a syntax error
	assert.Equal(t, sql.ErrNoRows, err, "Should get ErrNoRows for non-existent record, not syntax error")
	assert.False(t, existingID.Valid, "ID should not be valid for non-existent record")

	// Insert a record and test again
	_, err = tx.Exec(`
		INSERT INTO contact_requests (exchange_id, requester_user_id, requester_username, requester_first_name, requester_last_name)
		VALUES ($1, 123, 'testuser', 'Test', 'User')
	`, exchangeID)
	assert.NoError(t, err)

	// Now the SELECT FOR UPDATE should find the record
	err = tx.QueryRow(`
		SELECT id FROM contact_requests
		WHERE exchange_id = $1 AND requester_user_id = $2
		FOR UPDATE
	`, exchangeID, int64(123)).Scan(&existingID)

	assert.NoError(t, err, "SELECT FOR UPDATE should work without syntax errors")
	assert.True(t, existingID.Valid, "Should find the existing record")
	assert.Greater(t, existingID.Int64, int64(0), "ID should be positive")

	err = tx.Commit()
	assert.NoError(t, err)
}

func TestContactRequestTableConstraints(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Database tests require PostgreSQL connection")
		return
	}
	defer db.Close()

	// Clean up any existing data in correct order
	_, err := db.Exec("DELETE FROM contact_requests")
	assert.NoError(t, err)
	_, err = db.Exec("DELETE FROM timeline_records")
	assert.NoError(t, err)
	_, err = db.Exec("DELETE FROM exchanges")
	assert.NoError(t, err)
	_, err = db.Exec("DELETE FROM location_histories")
	assert.NoError(t, err)
	_, err = db.Exec("DELETE FROM users")
	assert.NoError(t, err)

	repo := NewRepository(db)

	// Create test users first
	testUser1 := &objects.User{UserId: 123, Username: "testuser", FirstName: "Test", LastName: "User", LanguageCode: "en"}
	testUser2 := &objects.User{UserId: 789, Username: "requester", FirstName: "Test", LastName: "Requester", LanguageCode: "en"}
	err = repo.SaveUser(testUser1)
	assert.NoError(t, err)
	err = repo.SaveUser(testUser2)
	assert.NoError(t, err)

	// Create test exchange
	amount := 50
	exchange := &objects.Exchange{
		UserID:            123,
		ExchangeDirection: "cash_to_crypto",
		Status:            "posted",
		AmountUSD:         &amount,
		Lat:               40.7128,
		Lon:               -74.006,
	}
	err = repo.CreateExchange(exchange)
	assert.NoError(t, err)
	exchangeID := exchange.ID

	// Test successful insert
	_, err = db.Exec(`
		INSERT INTO contact_requests (exchange_id, requester_user_id, requester_username, requester_first_name, requester_last_name)
		VALUES ($1, 789, 'requester', 'Test', 'Requester')
	`, exchangeID)
	assert.NoError(t, err)

	// Test unique constraint violation
	_, err = db.Exec(`
		INSERT INTO contact_requests (exchange_id, requester_user_id, requester_username, requester_first_name, requester_last_name)
		VALUES ($1, 789, 'requester', 'Test', 'Requester')
	`, exchangeID)
	assert.Error(t, err, "Should fail due to unique constraint")
	assert.Contains(t, err.Error(), "duplicate key", "Error should mention duplicate key")

	// Create another user for testing
	testUser3 := &objects.User{UserId: 888, Username: "user3", FirstName: "Test", LastName: "User3", LanguageCode: "en"}
	err = repo.SaveUser(testUser3)
	assert.NoError(t, err)

	// Test that NULL values are handled properly
	_, err = db.Exec(`
		INSERT INTO contact_requests (exchange_id, requester_user_id, requester_username, requester_first_name, requester_last_name)
		VALUES ($1, 888, NULL, 'Test', 'User')
	`, exchangeID)
	assert.NoError(t, err, "Should allow NULL username")
}

func TestContactRequestIndexes(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	// Verify that indexes exist
	indexes := []string{
		"idx_contact_requests_exchange_id",
		"idx_contact_requests_requester",
		"idx_contact_requests_requested_at",
	}

	for _, indexName := range indexes {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM pg_indexes 
				WHERE indexname = $1 AND tablename = 'contact_requests'
			)
		`, indexName).Scan(&exists)

		assert.NoError(t, err)
		assert.True(t, exists, "Index %s should exist", indexName)
	}
}
