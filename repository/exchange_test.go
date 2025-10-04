package repository

import (
	"librecash/objects"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setupTestDBForExchange(t *testing.T) (*Repository, func()) {
	db := setupTestDB(t)
	if db == nil {
		return nil, func() {}
	}
	return NewRepository(db), func() { db.Close() }
}

func TestCreateExchange(t *testing.T) {
	repo, cleanup := setupTestDBForExchange(t)
	defer cleanup()
	if repo == nil {
		return
	}

	// Create a test user first
	user := &objects.User{
		UserId:       123456,
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		LanguageCode: "en",
		MenuId:       objects.Menu_Init,
		Lat:          40.7128,
		Lon:          -74.0060,
	}
	err := repo.SaveUser(user)
	assert.NoError(t, err)

	// Create an exchange
	exchange := &objects.Exchange{
		UserID:            123456,
		ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
		Status:            objects.ExchangeStatusInitiated,
		Lat:               40.7128,
		Lon:               -74.0060,
	}

	err = repo.CreateExchange(exchange)
	assert.NoError(t, err)
	assert.NotZero(t, exchange.ID)
	assert.NotZero(t, exchange.CreatedAt)
	assert.NotZero(t, exchange.UpdatedAt)
}

func TestGetExchangeByID(t *testing.T) {
	repo, cleanup := setupTestDBForExchange(t)
	defer cleanup()
	if repo == nil {
		return
	}

	// Create a test user first
	user := &objects.User{
		UserId:       123456,
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		LanguageCode: "en",
		MenuId:       objects.Menu_Init,
		Lat:          40.7128,
		Lon:          -74.0060,
	}
	err := repo.SaveUser(user)
	assert.NoError(t, err)

	// Create an exchange
	exchange := &objects.Exchange{
		UserID:            123456,
		ExchangeDirection: objects.ExchangeDirectionCryptoToCash,
		Status:            objects.ExchangeStatusInitiated,
		Lat:               40.7128,
		Lon:               -74.0060,
	}

	err = repo.CreateExchange(exchange)
	assert.NoError(t, err)

	// Retrieve the exchange
	retrieved, err := repo.GetExchangeByID(exchange.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, exchange.ID, retrieved.ID)
	assert.Equal(t, exchange.UserID, retrieved.UserID)
	assert.Equal(t, exchange.ExchangeDirection, retrieved.ExchangeDirection)
	assert.Equal(t, exchange.Status, retrieved.Status)
	assert.Equal(t, exchange.Lat, retrieved.Lat)
	assert.Equal(t, exchange.Lon, retrieved.Lon)

	// Test non-existent exchange
	notFound, err := repo.GetExchangeByID(99999)
	assert.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestGetUserExchanges(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	repo := NewRepository(db)

	// Clean up any existing data (in correct order due to foreign key constraints)
	_, err := db.Exec(`DELETE FROM contact_requests`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM timeline_records`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM exchanges`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM location_histories`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM location_histories`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM users`)
	assert.NoError(t, err)

	// Create test users
	user1 := &objects.User{
		UserId:       123456,
		Username:     "testuser1",
		FirstName:    "Test",
		LastName:     "User1",
		LanguageCode: "en",
		MenuId:       objects.Menu_Init,
		Lat:          40.7128,
		Lon:          -74.0060,
	}
	err = repo.SaveUser(user1)
	assert.NoError(t, err)

	user2 := &objects.User{
		UserId:       789012,
		Username:     "testuser2",
		FirstName:    "Test",
		LastName:     "User2",
		LanguageCode: "en",
		MenuId:       objects.Menu_Init,
		Lat:          40.7128,
		Lon:          -74.0060,
	}
	err = repo.SaveUser(user2)
	assert.NoError(t, err)

	// Create exchanges for user1
	exchange1 := &objects.Exchange{
		UserID:            123456,
		ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
		Status:            objects.ExchangeStatusInitiated,
		Lat:               40.7128,
		Lon:               -74.0060,
	}
	err = repo.CreateExchange(exchange1)
	assert.NoError(t, err)

	// Sleep to ensure different timestamps
	time.Sleep(10 * time.Millisecond)

	exchange2 := &objects.Exchange{
		UserID:            123456,
		ExchangeDirection: objects.ExchangeDirectionCryptoToCash,
		Status:            objects.ExchangeStatusPosted,
		AmountUSD:         intPtr(100),
		Lat:               40.7128,
		Lon:               -74.0060,
	}
	err = repo.CreateExchange(exchange2)
	assert.NoError(t, err)

	// Create exchange for user2
	exchange3 := &objects.Exchange{
		UserID:            789012,
		ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
		Status:            objects.ExchangeStatusInitiated,
		Lat:               40.7128,
		Lon:               -74.0060,
	}
	err = repo.CreateExchange(exchange3)
	assert.NoError(t, err)

	// Get exchanges for user1
	user1Exchanges, err := repo.GetUserExchanges(123456)
	assert.NoError(t, err)
	assert.Len(t, user1Exchanges, 2)
	// Should be ordered by created_at DESC
	assert.Equal(t, exchange2.ID, user1Exchanges[0].ID)
	assert.Equal(t, exchange1.ID, user1Exchanges[1].ID)

	// Get exchanges for user2
	user2Exchanges, err := repo.GetUserExchanges(789012)
	assert.NoError(t, err)
	assert.Len(t, user2Exchanges, 1)
	assert.Equal(t, exchange3.ID, user2Exchanges[0].ID)

	// Note: Cannot test limit as GetUserExchanges doesn't take a limit parameter
	// The function returns all exchanges for a user

	// Test user with no exchanges
	noExchanges, err := repo.GetUserExchanges(999999)
	assert.NoError(t, err)
	assert.Len(t, noExchanges, 0)
}

func TestUpdateExchangeStatus(t *testing.T) {
	repo, cleanup := setupTestDBForExchange(t)
	defer cleanup()
	if repo == nil {
		return
	}

	// Create a test user first
	user := &objects.User{
		UserId:       123456,
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		LanguageCode: "en",
		MenuId:       objects.Menu_Init,
		Lat:          40.7128,
		Lon:          -74.0060,
	}
	err := repo.SaveUser(user)
	assert.NoError(t, err)

	// Create an exchange
	exchange := &objects.Exchange{
		UserID:            123456,
		ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
		Status:            objects.ExchangeStatusInitiated,
		Lat:               40.7128,
		Lon:               -74.0060,
	}

	err = repo.CreateExchange(exchange)
	assert.NoError(t, err)

	originalUpdatedAt := exchange.UpdatedAt

	// Sleep to ensure different timestamp
	time.Sleep(10 * time.Millisecond)

	// Update the status
	err = repo.UpdateExchangeStatus(exchange.ID, objects.ExchangeStatusPosted)
	assert.NoError(t, err)

	// Retrieve and verify
	updated, err := repo.GetExchangeByID(exchange.ID)
	assert.NoError(t, err)
	assert.Equal(t, objects.ExchangeStatusPosted, updated.Status)
	assert.True(t, updated.UpdatedAt.After(originalUpdatedAt))

	// Test updating non-existent exchange
	err = repo.UpdateExchangeStatus(99999, objects.ExchangeStatusPosted)
	assert.NoError(t, err) // Should not error, just no rows affected
}

func TestExchangeWithAmount(t *testing.T) {
	repo, cleanup := setupTestDBForExchange(t)
	defer cleanup()
	if repo == nil {
		return
	}

	// Create a test user first
	user := &objects.User{
		UserId:       123456,
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		LanguageCode: "en",
		MenuId:       objects.Menu_Init,
		Lat:          40.7128,
		Lon:          -74.0060,
	}
	err := repo.SaveUser(user)
	assert.NoError(t, err)

	// Create an exchange with amount
	exchange := &objects.Exchange{
		UserID:            123456,
		ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
		Status:            objects.ExchangeStatusPosted,
		AmountUSD:         intPtr(50),
		Lat:               40.7128,
		Lon:               -74.0060,
	}

	err = repo.CreateExchange(exchange)
	assert.NoError(t, err)

	// Retrieve and verify
	retrieved, err := repo.GetExchangeByID(exchange.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved.AmountUSD)
	assert.Equal(t, 50, *retrieved.AmountUSD)
}

func TestExchangeGeography(t *testing.T) {
	repo, cleanup := setupTestDBForExchange(t)
	defer cleanup()
	if repo == nil {
		return
	}

	// Create a test user first
	user := &objects.User{
		UserId:       123456,
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		LanguageCode: "en",
		MenuId:       objects.Menu_Init,
		Lat:          40.7128,
		Lon:          -74.0060,
	}
	err := repo.SaveUser(user)
	assert.NoError(t, err)

	// Create an exchange with specific coordinates
	exchange := &objects.Exchange{
		UserID:            123456,
		ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
		Status:            objects.ExchangeStatusInitiated,
		Lat:               51.5074, // London
		Lon:               -0.1278, // London
	}

	err = repo.CreateExchange(exchange)
	assert.NoError(t, err)

	// Retrieve and verify coordinates
	retrieved, err := repo.GetExchangeByID(exchange.ID)
	assert.NoError(t, err)
	assert.InDelta(t, 51.5074, retrieved.Lat, 0.0001)
	assert.InDelta(t, -0.1278, retrieved.Lon, 0.0001)
}

func TestSoftDeleteExchange(t *testing.T) {
	repo, cleanup := setupTestDBForExchange(t)
	defer cleanup()
	if repo == nil {
		return
	}

	// Create a test user first
	user := &objects.User{
		UserId:       123456,
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		LanguageCode: "en",
		MenuId:       objects.Menu_Init,
		Lat:          40.7128,
		Lon:          -74.0060,
	}
	err := repo.SaveUser(user)
	assert.NoError(t, err)

	// Create an exchange
	exchange := &objects.Exchange{
		UserID:            123456,
		ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
		Status:            objects.ExchangeStatusPosted,
		Lat:               40.7128,
		Lon:               -74.0060,
	}

	err = repo.CreateExchange(exchange)
	assert.NoError(t, err)

	// Verify exchange exists and is not deleted
	retrieved, err := repo.GetExchangeByID(exchange.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.False(t, retrieved.IsDeleted)
	assert.Nil(t, retrieved.DeletedAt)

	// Soft delete the exchange
	err = repo.SoftDeleteExchange(exchange.ID)
	assert.NoError(t, err)

	// Verify exchange is no longer returned by GetExchangeByID (filtered out)
	deleted, err := repo.GetExchangeByID(exchange.ID)
	assert.NoError(t, err)
	assert.Nil(t, deleted)

	// Test soft deleting non-existent exchange (should not error)
	err = repo.SoftDeleteExchange(99999)
	assert.NoError(t, err)
}

func TestGetActiveExchanges(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	repo := NewRepository(db)

	// Clean up any existing data (in correct order due to foreign key constraints)
	_, err := db.Exec(`DELETE FROM contact_requests`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM timeline_records`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM exchanges`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM location_histories`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM location_histories`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM users`)
	assert.NoError(t, err)

	// Create a test user first
	user := &objects.User{
		UserId:       123456,
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		LanguageCode: "en",
		MenuId:       objects.Menu_Init,
		Lat:          40.7128,
		Lon:          -74.0060,
	}
	err = repo.SaveUser(user)
	assert.NoError(t, err)

	// Create multiple exchanges
	exchange1 := &objects.Exchange{
		UserID:            123456,
		ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
		Status:            objects.ExchangeStatusPosted,
		Lat:               40.7128,
		Lon:               -74.0060,
	}
	err = repo.CreateExchange(exchange1)
	assert.NoError(t, err)

	exchange2 := &objects.Exchange{
		UserID:            123456,
		ExchangeDirection: objects.ExchangeDirectionCryptoToCash,
		Status:            objects.ExchangeStatusPosted,
		Lat:               40.7128,
		Lon:               -74.0060,
	}
	err = repo.CreateExchange(exchange2)
	assert.NoError(t, err)

	// Get all active exchanges (should return both)
	activeExchanges, err := repo.GetActiveExchanges()
	assert.NoError(t, err)
	assert.Len(t, activeExchanges, 2)

	// Soft delete one exchange
	err = repo.SoftDeleteExchange(exchange1.ID)
	assert.NoError(t, err)

	// Get active exchanges again (should return only one)
	activeExchanges, err = repo.GetActiveExchanges()
	assert.NoError(t, err)
	assert.Len(t, activeExchanges, 1)
	assert.Equal(t, exchange2.ID, activeExchanges[0].ID)
}

func TestGetExchangeByIDWithDeleted(t *testing.T) {
	repo, cleanup := setupTestDBForExchange(t)
	defer cleanup()
	if repo == nil {
		return
	}

	// Create a test user first
	user := &objects.User{
		UserId:       123456,
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		LanguageCode: "en",
		MenuId:       objects.Menu_Init,
		Lat:          40.7128,
		Lon:          -74.0060,
	}
	err := repo.SaveUser(user)
	assert.NoError(t, err)

	// Create an exchange
	exchange := &objects.Exchange{
		UserID:            123456,
		ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
		Status:            objects.ExchangeStatusPosted,
		Lat:               40.7128,
		Lon:               -74.0060,
	}

	err = repo.CreateExchange(exchange)
	assert.NoError(t, err)

	// Verify exchange can be retrieved
	retrieved, err := repo.GetExchangeByID(exchange.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)

	// Soft delete the exchange
	err = repo.SoftDeleteExchange(exchange.ID)
	assert.NoError(t, err)

	// Verify GetExchangeByID returns nil for deleted exchange
	deleted, err := repo.GetExchangeByID(exchange.ID)
	assert.NoError(t, err)
	assert.Nil(t, deleted)
}

func TestGetUserExchangesWithDeleted(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	repo := NewRepository(db)

	// Clean up any existing data (in correct order due to foreign key constraints)
	_, err := db.Exec(`DELETE FROM contact_requests`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM timeline_records`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM exchanges`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM location_histories`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM users`)
	assert.NoError(t, err)

	// Create a test user first
	user := &objects.User{
		UserId:       123456,
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		LanguageCode: "en",
		MenuId:       objects.Menu_Init,
		Lat:          40.7128,
		Lon:          -74.0060,
	}
	err = repo.SaveUser(user)
	assert.NoError(t, err)

	// Create multiple exchanges for the user
	exchange1 := &objects.Exchange{
		UserID:            123456,
		ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
		Status:            objects.ExchangeStatusPosted,
		Lat:               40.7128,
		Lon:               -74.0060,
	}
	err = repo.CreateExchange(exchange1)
	assert.NoError(t, err)

	exchange2 := &objects.Exchange{
		UserID:            123456,
		ExchangeDirection: objects.ExchangeDirectionCryptoToCash,
		Status:            objects.ExchangeStatusPosted,
		Lat:               40.7128,
		Lon:               -74.0060,
	}
	err = repo.CreateExchange(exchange2)
	assert.NoError(t, err)

	// Get user exchanges (should return both)
	userExchanges, err := repo.GetUserExchanges(123456)
	assert.NoError(t, err)
	assert.Len(t, userExchanges, 2)

	// Soft delete one exchange
	err = repo.SoftDeleteExchange(exchange1.ID)
	assert.NoError(t, err)

	// Get user exchanges again (should return only one)
	userExchanges, err = repo.GetUserExchanges(123456)
	assert.NoError(t, err)
	assert.Len(t, userExchanges, 1)
	assert.Equal(t, exchange2.ID, userExchanges[0].ID)
}

func TestFindHistoricalExchangesInRadius(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	repo := NewRepository(db)

	// Clean up any existing data
	_, err := db.Exec(`DELETE FROM contact_requests`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM timeline_records`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM exchanges`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM location_histories`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM users`)
	assert.NoError(t, err)

	// Create test users
	user1 := &objects.User{
		UserId:       123456,
		Username:     "user1",
		FirstName:    "Test",
		LastName:     "User1",
		LanguageCode: "en",
		MenuId:       objects.Menu_Init,
		Lat:          40.7128,
		Lon:          -74.0060,
	}
	err = repo.SaveUser(user1)
	assert.NoError(t, err)

	user2 := &objects.User{
		UserId:       789012,
		Username:     "user2",
		FirstName:    "Test",
		LastName:     "User2",
		LanguageCode: "en",
		MenuId:       objects.Menu_Init,
		Lat:          40.7200,
		Lon:          -74.0100,
	}
	err = repo.SaveUser(user2)
	assert.NoError(t, err)

	user3 := &objects.User{
		UserId:       345678,
		Username:     "user3",
		FirstName:    "Test",
		LastName:     "User3",
		LanguageCode: "en",
		MenuId:       objects.Menu_Init,
		Lat:          40.7300,
		Lon:          -74.0200,
	}
	err = repo.SaveUser(user3)
	assert.NoError(t, err)

	// Create historical exchanges (older than current time)
	now := time.Now()

	// Exchange from 2 days ago (should be found)
	exchange1 := &objects.Exchange{
		UserID:            123456,
		ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
		Status:            objects.ExchangeStatusPosted,
		Lat:               40.7128,
		Lon:               -74.0060,
		IsDeleted:         false,
		CreatedAt:         now.Add(-48 * time.Hour),
		UpdatedAt:         now.Add(-48 * time.Hour),
	}
	err = repo.CreateExchange(exchange1)
	assert.NoError(t, err)

	// Exchange from 1 day ago (should be found in 3-day search)
	exchange2 := &objects.Exchange{
		UserID:            789012,
		ExchangeDirection: objects.ExchangeDirectionCryptoToCash,
		Status:            objects.ExchangeStatusPosted,
		Lat:               40.7200,
		Lon:               -74.0100,
		IsDeleted:         false,
		CreatedAt:         now.Add(-24 * time.Hour),
		UpdatedAt:         now.Add(-24 * time.Hour),
	}
	err = repo.CreateExchange(exchange2)
	assert.NoError(t, err)

	// Deleted exchange (should NOT be found)
	exchange3 := &objects.Exchange{
		UserID:            345678,
		ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
		Status:            objects.ExchangeStatusPosted,
		Lat:               40.7300,
		Lon:               -74.0200,
		IsDeleted:         true,
		DeletedAt:         &now,
		CreatedAt:         now.Add(-24 * time.Hour),
		UpdatedAt:         now.Add(-24 * time.Hour),
	}
	err = repo.CreateExchange(exchange3)
	assert.NoError(t, err)

	// Exchange with status 'initiated' (should NOT be found)
	exchange4 := &objects.Exchange{
		UserID:            345678,
		ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
		Status:            objects.ExchangeStatusInitiated,
		Lat:               40.7300,
		Lon:               -74.0200,
		IsDeleted:         false,
		CreatedAt:         now.Add(-12 * time.Hour),
		UpdatedAt:         now.Add(-12 * time.Hour),
	}
	err = repo.CreateExchange(exchange4)
	assert.NoError(t, err)

	// Test finding historical exchanges
	newUserID := int64(999999) // New user who doesn't have exchanges
	historicalExchanges, err := repo.FindHistoricalExchangesInRadius(
		40.7150, -74.0080, // Near the test exchanges
		10,        // 10 km radius
		newUserID, // exclude this user's exchanges
	)
	assert.NoError(t, err)
	assert.Len(t, historicalExchanges, 2, "Should find 2 active posted exchanges")

	// Verify the exchanges are sorted by creation time (oldest first)
	assert.True(t, historicalExchanges[0].CreatedAt.Before(historicalExchanges[1].CreatedAt),
		"Exchanges should be sorted by creation time (oldest first)")

	// Test excluding user's own exchanges
	ownExchanges, err := repo.FindHistoricalExchangesInRadius(
		40.7150, -74.0080,
		10,
		123456, // exclude user1's exchanges
	)
	assert.NoError(t, err)
	assert.Len(t, ownExchanges, 1, "Should exclude user's own exchanges")
	assert.Equal(t, int64(789012), ownExchanges[0].UserID, "Should only return other users' exchanges")
}

func TestFindHistoricalExchangesInRadiusTimeFrames(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	repo := NewRepository(db)

	// Clean up any existing data
	_, err := db.Exec(`DELETE FROM contact_requests`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM timeline_records`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM exchanges`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM location_histories`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM users`)
	assert.NoError(t, err)

	// Create test user
	user := &objects.User{
		UserId:       123456,
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		LanguageCode: "en",
		MenuId:       objects.Menu_Init,
		Lat:          40.7128,
		Lon:          -74.0060,
	}
	err = repo.SaveUser(user)
	assert.NoError(t, err)

	// Create exchange from 10 days ago (should be found in 30-day timeframe)
	now := time.Now()
	exchange := &objects.Exchange{
		UserID:            123456,
		ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
		Status:            objects.ExchangeStatusPosted,
		Lat:               40.7128,
		Lon:               -74.0060,
		IsDeleted:         false,
		CreatedAt:         now.Add(-240 * time.Hour), // 10 days ago
		UpdatedAt:         now.Add(-240 * time.Hour),
	}
	err = repo.CreateExchange(exchange)
	assert.NoError(t, err)

	// Test finding historical exchanges (should find the 10-day-old exchange in 30-day timeframe)
	newUserID := int64(999999)
	historicalExchanges, err := repo.FindHistoricalExchangesInRadius(
		40.7150, -74.0080,
		10,
		newUserID,
	)
	assert.NoError(t, err)
	assert.Len(t, historicalExchanges, 1, "Should find exchange in 30-day timeframe")
}

func TestFindHistoricalExchangesInRadiusOldExchanges(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	repo := NewRepository(db)

	// Clean up any existing data
	_, err := db.Exec(`DELETE FROM contact_requests`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM timeline_records`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM exchanges`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM location_histories`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM users`)
	assert.NoError(t, err)

	// Create test user
	user := &objects.User{
		UserId:       123456,
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		LanguageCode: "en",
		MenuId:       objects.Menu_Init,
		Lat:          40.7128,
		Lon:          -74.0060,
	}
	err = repo.SaveUser(user)
	assert.NoError(t, err)

	// Create exchange from 35 days ago (should NOT be found - older than 30 days)
	now := time.Now()
	oldExchange := &objects.Exchange{
		UserID:            123456,
		ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
		Status:            objects.ExchangeStatusPosted,
		Lat:               40.7128,
		Lon:               -74.0060,
		IsDeleted:         false,
		CreatedAt:         now.Add(-35 * 24 * time.Hour), // 35 days ago
		UpdatedAt:         now.Add(-35 * 24 * time.Hour),
	}
	err = repo.CreateExchange(oldExchange)
	assert.NoError(t, err)

	// Test finding historical exchanges (should NOT find the 35-day-old exchange)
	newUserID := int64(999999)
	historicalExchanges, err := repo.FindHistoricalExchangesInRadius(
		40.7150, -74.0080,
		10,
		newUserID,
	)
	assert.NoError(t, err)
	assert.Len(t, historicalExchanges, 0, "Should not find exchanges older than 30 days")
}

func TestLocationHistory(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	repo := NewRepository(db)

	// Clean up any existing data
	_, err := db.Exec(`DELETE FROM contact_requests`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM timeline_records`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM exchanges`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM location_histories`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM users`)
	assert.NoError(t, err)

	// Create test user
	user := &objects.User{
		UserId:       123456,
		Username:     "testuser",
		FirstName:    "Test",
		LastName:     "User",
		LanguageCode: "en",
		MenuId:       objects.Menu_Init,
		Lat:          40.7128,
		Lon:          -74.0060,
	}
	err = repo.SaveUser(user)
	assert.NoError(t, err)

	// Test CreateLocationHistory
	err = repo.CreateLocationHistory(123456, 15)
	assert.NoError(t, err)

	// Test UpdateLocationHistory
	err = repo.UpdateLocationHistory(123456, 40.7150, -74.0080)
	assert.NoError(t, err)

	// Test ShouldTriggerHistoricalFanout - first record should always trigger
	shouldTrigger, err := repo.ShouldTriggerHistoricalFanout(123456)
	assert.NoError(t, err)
	assert.True(t, shouldTrigger, "First location history record should trigger fanout")

	// Create second location history record with same data
	err = repo.CreateLocationHistory(123456, 15)
	assert.NoError(t, err)
	err = repo.UpdateLocationHistory(123456, 40.7150, -74.0080)
	assert.NoError(t, err)

	// Should not trigger fanout (same data)
	shouldTrigger, err = repo.ShouldTriggerHistoricalFanout(123456)
	assert.NoError(t, err)
	assert.False(t, shouldTrigger, "Same location and radius should not trigger fanout")

	// Create third record with different radius
	err = repo.CreateLocationHistory(123456, 25)
	assert.NoError(t, err)
	err = repo.UpdateLocationHistory(123456, 40.7150, -74.0080)
	assert.NoError(t, err)

	// Should trigger fanout (different radius)
	shouldTrigger, err = repo.ShouldTriggerHistoricalFanout(123456)
	assert.NoError(t, err)
	assert.True(t, shouldTrigger, "Different radius should trigger fanout")

	// Create fourth record with different location
	err = repo.CreateLocationHistory(123456, 25)
	assert.NoError(t, err)
	err = repo.UpdateLocationHistory(123456, 40.7200, -74.0100)
	assert.NoError(t, err)

	// Should trigger fanout (different location)
	shouldTrigger, err = repo.ShouldTriggerHistoricalFanout(123456)
	assert.NoError(t, err)
	assert.True(t, shouldTrigger, "Different location should trigger fanout")
}

// Helper function
func intPtr(i int) *int {
	return &i
}
