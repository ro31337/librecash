package repository

import (
	"librecash/objects"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateTimelineRecord(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	repo := NewRepository(db)

	// Create test users first
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
	err := repo.SaveUser(user1)
	assert.NoError(t, err)

	user2 := &objects.User{
		UserId:       789012,
		Username:     "testuser2",
		FirstName:    "Test",
		LastName:     "User2",
		LanguageCode: "en",
		MenuId:       objects.Menu_Init,
		Lat:          40.7589,
		Lon:          -73.9851,
	}
	err = repo.SaveUser(user2)
	assert.NoError(t, err)

	// Create test exchange
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

	// Create timeline record
	record := objects.NewTimelineRecord(exchange.ID, user2.UserId)
	err = repo.CreateTimelineRecord(record)
	assert.NoError(t, err)

	// Verify record was created
	assert.NotZero(t, record.ID)
	assert.Equal(t, exchange.ID, record.ExchangeID)
	assert.Equal(t, user2.UserId, record.RecipientUserID)
	assert.Equal(t, objects.TimelineStatusPending, record.Status)
	assert.False(t, record.IsDeleted)
	assert.Nil(t, record.TelegramMessageID)
}

func TestGetTimelineRecordsByExchange(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	repo := NewRepository(db)

	// Create test users and exchange
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
	err := repo.SaveUser(user1)
	assert.NoError(t, err)

	user2 := &objects.User{
		UserId:       789012,
		Username:     "testuser2",
		FirstName:    "Test",
		LastName:     "User2",
		LanguageCode: "en",
		MenuId:       objects.Menu_Init,
		Lat:          40.7589,
		Lon:          -73.9851,
	}
	err = repo.SaveUser(user2)
	assert.NoError(t, err)

	user3 := &objects.User{
		UserId:       345678,
		Username:     "testuser3",
		FirstName:    "Test",
		LastName:     "User3",
		LanguageCode: "en",
		MenuId:       objects.Menu_Init,
		Lat:          40.7505,
		Lon:          -73.9934,
	}
	err = repo.SaveUser(user3)
	assert.NoError(t, err)

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

	// Create multiple timeline records
	record1 := objects.NewTimelineRecord(exchange.ID, user2.UserId)
	err = repo.CreateTimelineRecord(record1)
	assert.NoError(t, err)

	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	record2 := objects.NewTimelineRecord(exchange.ID, user3.UserId)
	err = repo.CreateTimelineRecord(record2)
	assert.NoError(t, err)

	// Get timeline records
	records, err := repo.GetTimelineRecordsByExchange(exchange.ID)
	assert.NoError(t, err)
	assert.Len(t, records, 2)

	// Should be ordered by created_at DESC (newest first)
	assert.Equal(t, user3.UserId, records[0].RecipientUserID)
	assert.Equal(t, user2.UserId, records[1].RecipientUserID)
}

func TestUpdateTimelineRecord(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	repo := NewRepository(db)

	// Create test data
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
	err := repo.SaveUser(user1)
	assert.NoError(t, err)

	user2 := &objects.User{
		UserId:       789012,
		Username:     "testuser2",
		FirstName:    "Test",
		LastName:     "User2",
		LanguageCode: "en",
		MenuId:       objects.Menu_Init,
		Lat:          40.7589,
		Lon:          -73.9851,
	}
	err = repo.SaveUser(user2)
	assert.NoError(t, err)

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

	record := objects.NewTimelineRecord(exchange.ID, user2.UserId)
	err = repo.CreateTimelineRecord(record)
	assert.NoError(t, err)

	// Update with Telegram message ID
	telegramMessageID := 12345
	err = repo.UpdateTimelineRecord(record.ID, telegramMessageID, objects.TimelineStatusSent)
	assert.NoError(t, err)

	// Verify update
	records, err := repo.GetTimelineRecordsByExchange(exchange.ID)
	assert.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, objects.TimelineStatusSent, records[0].Status)
	assert.NotNil(t, records[0].TelegramMessageID)
	assert.Equal(t, telegramMessageID, *records[0].TelegramMessageID)
}

func TestMarkTimelineRecordsAsDeleted(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	repo := NewRepository(db)

	// Create test data
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
	err := repo.SaveUser(user1)
	assert.NoError(t, err)

	user2 := &objects.User{
		UserId:       789012,
		Username:     "testuser2",
		FirstName:    "Test",
		LastName:     "User2",
		LanguageCode: "en",
		MenuId:       objects.Menu_Init,
		Lat:          40.7589,
		Lon:          -73.9851,
	}
	err = repo.SaveUser(user2)
	assert.NoError(t, err)

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

	// Create timeline records
	record1 := objects.NewTimelineRecord(exchange.ID, user2.UserId)
	err = repo.CreateTimelineRecord(record1)
	assert.NoError(t, err)

	record2 := objects.NewTimelineRecord(exchange.ID, user1.UserId)
	err = repo.CreateTimelineRecord(record2)
	assert.NoError(t, err)

	// Mark as deleted
	err = repo.MarkTimelineRecordsAsDeleted(exchange.ID)
	assert.NoError(t, err)

	// Verify all records are marked as deleted
	records, err := repo.GetTimelineRecordsByExchange(exchange.ID)
	assert.NoError(t, err)
	assert.Len(t, records, 2)
	for _, record := range records {
		assert.True(t, record.IsDeleted)
		assert.NotNil(t, record.DeletedAt)
	}

	// Verify GetActiveTimelineRecordsByExchange returns empty
	activeRecords, err := repo.GetActiveTimelineRecordsByExchange(exchange.ID)
	assert.NoError(t, err)
	assert.Len(t, activeRecords, 0)
}

// Tests for PRD009: Exchange Deletion by Author - Repository Methods

func TestSoftDeleteExchangeTimeline(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewRepository(db)

	// Create test user
	user1 := &objects.User{
		UserId:       123456,
		Username:     "testuser1",
		FirstName:    "Test",
		LastName:     "User1",
		LanguageCode: "en",
		Lat:          40.7128,
		Lon:          -74.0060,
	}
	err := repo.SaveUser(user1)
	assert.NoError(t, err)

	user2 := &objects.User{
		UserId:       789012,
		Username:     "testuser2",
		FirstName:    "Test",
		LastName:     "User2",
		LanguageCode: "en",
		Lat:          40.7589,
		Lon:          -73.9851,
	}
	err = repo.SaveUser(user2)
	assert.NoError(t, err)

	// Create test exchange
	exchange := &objects.Exchange{
		UserID:            user1.UserId,
		ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
		Status:            objects.ExchangeStatusPosted,
		AmountUSD:         testIntPtr(50),
		Lat:               40.7128,
		Lon:               -74.0060,
	}
	err = repo.CreateExchange(exchange)
	assert.NoError(t, err)

	// Create multiple timeline records for the exchange
	record1 := objects.NewTimelineRecord(exchange.ID, user1.UserId)
	err = repo.CreateTimelineRecord(record1)
	assert.NoError(t, err)

	record2 := objects.NewTimelineRecord(exchange.ID, user2.UserId)
	err = repo.CreateTimelineRecord(record2)
	assert.NoError(t, err)

	// Verify records exist and are not deleted
	records, err := repo.GetTimelineRecordsByExchange(exchange.ID)
	assert.NoError(t, err)
	assert.Len(t, records, 2)
	for _, record := range records {
		assert.False(t, record.IsDeleted, "Records should not be deleted initially")
		assert.Nil(t, record.DeletedAt, "DeletedAt should be nil initially")
	}

	// Test SoftDeleteExchangeTimeline (which is an alias for MarkTimelineRecordsAsDeleted)
	err = repo.SoftDeleteExchangeTimeline(exchange.ID)
	assert.NoError(t, err)

	// Verify all records are now soft deleted
	records, err = repo.GetTimelineRecordsByExchange(exchange.ID)
	assert.NoError(t, err)
	assert.Len(t, records, 2)
	for _, record := range records {
		assert.True(t, record.IsDeleted, "Records should be marked as deleted")
		assert.NotNil(t, record.DeletedAt, "DeletedAt should be set")
	}

	// Verify GetActiveTimelineRecordsByExchange returns empty
	activeRecords, err := repo.GetActiveTimelineRecordsByExchange(exchange.ID)
	assert.NoError(t, err)
	assert.Len(t, activeRecords, 0, "Should return no active records after soft delete")
}

func TestSoftDeleteExchangeTimeline_NonExistentExchange(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewRepository(db)

	// Try to soft delete timeline records for non-existent exchange
	err := repo.SoftDeleteExchangeTimeline(999999)
	assert.NoError(t, err, "Should not error when deleting non-existent exchange timeline")

	// Verify no records exist
	records, err := repo.GetTimelineRecordsByExchange(999999)
	assert.NoError(t, err)
	assert.Len(t, records, 0, "Should return no records for non-existent exchange")
}

func TestGetTimelineRecordsByExchange_IncludesDeleted(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewRepository(db)

	// Create test user and exchange
	user1 := &objects.User{
		UserId:       123456,
		Username:     "testuser1",
		FirstName:    "Test",
		LastName:     "User1",
		LanguageCode: "en",
		Lat:          40.7128,
		Lon:          -74.0060,
	}
	err := repo.SaveUser(user1)
	assert.NoError(t, err)

	exchange := &objects.Exchange{
		UserID:            user1.UserId,
		ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
		Status:            objects.ExchangeStatusPosted,
		AmountUSD:         testIntPtr(50),
		Lat:               40.7128,
		Lon:               -74.0060,
	}
	err = repo.CreateExchange(exchange)
	assert.NoError(t, err)

	// Create timeline record
	record := objects.NewTimelineRecord(exchange.ID, user1.UserId)
	err = repo.CreateTimelineRecord(record)
	assert.NoError(t, err)

	// Verify record exists before deletion
	records, err := repo.GetTimelineRecordsByExchange(exchange.ID)
	assert.NoError(t, err)
	assert.Len(t, records, 1)
	assert.False(t, records[0].IsDeleted)

	// Soft delete the record
	err = repo.SoftDeleteExchangeTimeline(exchange.ID)
	assert.NoError(t, err)

	// Verify GetTimelineRecordsByExchange still returns the deleted record
	records, err = repo.GetTimelineRecordsByExchange(exchange.ID)
	assert.NoError(t, err)
	assert.Len(t, records, 1, "GetTimelineRecordsByExchange should include deleted records")
	assert.True(t, records[0].IsDeleted, "Record should be marked as deleted")

	// Verify GetActiveTimelineRecordsByExchange excludes the deleted record
	activeRecords, err := repo.GetActiveTimelineRecordsByExchange(exchange.ID)
	assert.NoError(t, err)
	assert.Len(t, activeRecords, 0, "GetActiveTimelineRecordsByExchange should exclude deleted records")
}

// Helper function for tests
func testIntPtr(i int) *int {
	return &i
}
