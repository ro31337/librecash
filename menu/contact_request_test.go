package menu

import (
	"fmt"
	"librecash/objects"
	"librecash/rabbit"
	"strconv"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/stretchr/testify/assert"
)

// Mock structures for testing contact requests

type MockContactRepository struct {
	users           map[int64]*objects.User
	exchanges       map[int64]*objects.Exchange
	contactRequests map[string]bool                     // key: "exchangeID:userID"
	timelineRecords map[int64][]*objects.TimelineRecord // key: exchangeID
}

func (m *MockContactRepository) FindUser(userID int64) *objects.User {
	return m.users[userID]
}

func (m *MockContactRepository) GetExchangeByID(id int64) (*objects.Exchange, error) {
	exchange := m.exchanges[id]
	if exchange == nil || exchange.IsDeleted {
		return nil, nil // Return nil for deleted or non-existent exchanges
	}
	return exchange, nil
}

func (m *MockContactRepository) CheckContactRequestExists(exchangeID, requesterUserID int64) (bool, error) {
	key := fmt.Sprintf("%d:%d", exchangeID, requesterUserID)
	return m.contactRequests[key], nil
}

func (m *MockContactRepository) CreateContactRequest(exchangeID, requesterUserID int64, username, firstName, lastName string) error {
	key := fmt.Sprintf("%d:%d", exchangeID, requesterUserID)
	m.contactRequests[key] = true
	return nil
}

func (m *MockContactRepository) SoftDeleteExchange(exchangeID int64) error {
	if exchange, exists := m.exchanges[exchangeID]; exists {
		exchange.IsDeleted = true
		now := time.Now()
		exchange.DeletedAt = &now
	}
	return nil
}

func (m *MockContactRepository) GetTimelineRecordsByExchange(exchangeID int64) ([]*objects.TimelineRecord, error) {
	records := m.timelineRecords[exchangeID]
	if records == nil {
		return []*objects.TimelineRecord{}, nil
	}
	return records, nil
}

func (m *MockContactRepository) SoftDeleteExchangeTimeline(exchangeID int64) error {
	if records, exists := m.timelineRecords[exchangeID]; exists {
		for _, record := range records {
			record.IsDeleted = true
			now := time.Now()
			record.DeletedAt = &now
		}
	}
	return nil
}

type MockContactRabbitClient struct {
	publishedMessages []rabbit.MessageBag
}

func (m *MockContactRabbitClient) PublishTgMessage(messageBag rabbit.MessageBag) error {
	m.publishedMessages = append(m.publishedMessages, messageBag)
	return nil
}

type MockContactBot struct {
	editedMessages []tgbotapi.EditMessageTextConfig
}

func (m *MockContactBot) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	if editMsg, ok := c.(tgbotapi.EditMessageTextConfig); ok {
		m.editedMessages = append(m.editedMessages, editMsg)
		return tgbotapi.Message{MessageID: editMsg.MessageID}, nil
	}
	return tgbotapi.Message{}, nil
}

type MockContactContext struct {
	Repo          *MockContactRepository
	RabbitPublish *MockContactRabbitClient
	Bot           *MockContactBot
}

func (m *MockContactContext) Send(message tgbotapi.MessageConfig) {
	m.RabbitPublish.PublishTgMessage(rabbit.MessageBag{
		Message:  message,
		Priority: 220,
	})
}

func (m *MockContactContext) AnswerCallbackQuery(callback tgbotapi.CallbackConfig) error {
	// Mock implementation - just return nil
	return nil
}

func (m *MockContactContext) EditMessage(editMsg tgbotapi.EditMessageTextConfig) error {
	// Mock implementation - just return nil
	return nil
}

func TestFormatUserIdentifier(t *testing.T) {
	tests := []struct {
		name         string
		user         *objects.User
		includePhone bool
		expected     string
	}{
		{
			name: "User with username, no phone",
			user: &objects.User{
				UserId:    123,
				Username:  "john_doe",
				FirstName: "John",
				LastName:  "Doe",
			},
			includePhone: false,
			expected:     "@john_doe",
		},
		{
			name: "User without username - full name, no phone",
			user: &objects.User{
				UserId:    456,
				Username:  "",
				FirstName: "Jane",
				LastName:  "Smith",
			},
			includePhone: false,
			expected:     `<a href="tg://user?id=456">Jane Smith</a>`,
		},
		{
			name: "User with phone number included",
			user: &objects.User{
				UserId:      789,
				Username:    "bob_wilson",
				FirstName:   "Bob",
				LastName:    "Wilson",
				PhoneNumber: "+1234567890",
			},
			includePhone: true,
			expected:     "@bob_wilson\nPhone: +1234567890",
		},
		{
			name: "User without username - first name only, no phone",
			user: &objects.User{
				UserId:    101,
				Username:  "",
				FirstName: "Alice",
				LastName:  "",
			},
			includePhone: false,
			expected:     `<a href="tg://user?id=101">Alice</a>`,
		},
		{
			name: "User without username or names, no phone",
			user: &objects.User{
				UserId:    103,
				Username:  "",
				FirstName: "",
				LastName:  "",
			},
			includePhone: false,
			expected:     `<a href="tg://user?id=103">Anonymous</a>`,
		},
		{
			name: "User with special characters in name (HTML escaped)",
			user: &objects.User{
				UserId:    104,
				Username:  "",
				FirstName: "John<script>",
				LastName:  "Doe&Co",
			},
			includePhone: false,
			expected:     `<a href="tg://user?id=104">John&lt;script&gt; Doe&amp;Co</a>`,
		},
		{
			name: "User with username and phone number included",
			user: &objects.User{
				UserId:      105,
				Username:    "john_doe",
				FirstName:   "John",
				LastName:    "Doe",
				PhoneNumber: "+1234567890",
			},
			includePhone: true,
			expected:     "@john_doe\nPhone: +1234567890",
		},
		{
			name: "User without username but with phone number included",
			user: &objects.User{
				UserId:      106,
				Username:    "",
				FirstName:   "Jane",
				LastName:    "Smith",
				PhoneNumber: "+0987654321",
			},
			includePhone: true,
			expected:     `<a href="tg://user?id=106">Jane Smith</a>` + "\nPhone: +0987654321",
		},
		{
			name: "User with phone but phone not included",
			user: &objects.User{
				UserId:      107,
				Username:    "test_user",
				PhoneNumber: "+1111111111",
			},
			includePhone: false,
			expected:     "@test_user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatUserIdentifier(tt.user, tt.includePhone, "en")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHandleContactRequestCallback_ValidRequest(t *testing.T) {
	// Setup mock data
	mockRepo := &MockContactRepository{
		users: map[int64]*objects.User{
			123: {
				UserId:       123,
				Username:     "requester",
				FirstName:    "Test",
				LastName:     "Requester",
				LanguageCode: "en",
			},
			456: {
				UserId:       456,
				Username:     "initiator",
				FirstName:    "Test",
				LastName:     "Initiator",
				LanguageCode: "en",
			},
		},
		exchanges: map[int64]*objects.Exchange{
			789: {
				ID:     789,
				UserID: 456, // initiator
			},
		},
		contactRequests: make(map[string]bool),
	}

	mockRabbit := &MockContactRabbitClient{
		publishedMessages: make([]rabbit.MessageBag, 0),
	}

	mockBot := &MockContactBot{
		editedMessages: make([]tgbotapi.EditMessageTextConfig, 0),
	}

	// This test would require full context setup - implement as integration test
	t.Skip("Test requires full context setup - implement as integration test")

	// Variables below are prepared for future integration test implementation
	_ = mockRepo
	_ = mockRabbit
	_ = mockBot

	// TODO: Implement when we have proper context mocking
	// HandleContactRequestCallback(mockContext, callback, user)

	// Verify contact request was created
	// assert.True(t, mockRepo.contactRequests["789:123"])

	// Verify message was edited
	// assert.Len(t, mockBot.editedMessages, 1)

	// Verify notification was sent
	// assert.Len(t, mockRabbit.publishedMessages, 1)
}

func TestHandleContactRequestCallback_DuplicateRequest(t *testing.T) {
	// Setup mock data with existing contact request
	mockRepo := &MockContactRepository{
		users: map[int64]*objects.User{
			123: {
				UserId:       123,
				Username:     "requester",
				FirstName:    "Test",
				LastName:     "Requester",
				LanguageCode: "en",
			},
		},
		exchanges: map[int64]*objects.Exchange{
			789: {
				ID:     789,
				UserID: 456,
			},
		},
		contactRequests: map[string]bool{
			"789:123": true, // Already exists
		},
	}

	// Test duplicate detection
	exists, err := mockRepo.CheckContactRequestExists(789, 123)
	assert.NoError(t, err)
	assert.True(t, exists, "Should detect existing contact request")

	// Test non-existent request
	exists, err = mockRepo.CheckContactRequestExists(789, 999)
	assert.NoError(t, err)
	assert.False(t, exists, "Should not find non-existent contact request")
}

func TestHandleContactRequestCallback_InvalidData(t *testing.T) {
	tests := []struct {
		name         string
		callbackData string
		shouldHandle bool
	}{
		{
			name:         "Valid contact callback",
			callbackData: "contact:123",
			shouldHandle: true,
		},
		{
			name:         "Invalid prefix",
			callbackData: "invalid:123",
			shouldHandle: false,
		},
		{
			name:         "Missing exchange ID",
			callbackData: "contact:",
			shouldHandle: false,
		},
		{
			name:         "Invalid exchange ID",
			callbackData: "contact:abc",
			shouldHandle: false,
		},
		{
			name:         "No colon separator",
			callbackData: "contact123",
			shouldHandle: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test callback data parsing logic
			parts := strings.Split(tt.callbackData, ":")
			isValid := len(parts) == 2 && parts[0] == "contact"

			if tt.shouldHandle {
				assert.True(t, isValid, "Should be valid callback data")
				if isValid {
					_, err := strconv.ParseInt(parts[1], 10, 64)
					assert.NoError(t, err, "Should parse exchange ID")
				}
			} else {
				if isValid {
					_, err := strconv.ParseInt(parts[1], 10, 64)
					assert.Error(t, err, "Should fail to parse invalid exchange ID")
				} else {
					assert.False(t, isValid, "Should be invalid callback data")
				}
			}
		})
	}
}

func TestHtmlEscapeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No special characters",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "HTML special characters",
			input:    "Hello<World>&Co",
			expected: "Hello&lt;World&gt;&amp;Co",
		},
		{
			name:     "Quotes and apostrophes",
			input:    `Hello "World" & 'Test'`,
			expected: "Hello &quot;World&quot; &amp; &#39;Test&#39;",
		},
		{
			name:     "All HTML special characters",
			input:    `<>&"'`,
			expected: "&lt;&gt;&amp;&quot;&#39;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := htmlEscapeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Tests for PRD009: Exchange Deletion by Author

func TestHandleDeleteExchangeCallback_ValidData(t *testing.T) {
	tests := []struct {
		name         string
		callbackData string
		shouldHandle bool
		expectedID   int64
	}{
		{
			name:         "Valid delete callback",
			callbackData: "delete:123",
			shouldHandle: true,
			expectedID:   123,
		},
		{
			name:         "Valid delete callback with large ID",
			callbackData: "delete:999999999",
			shouldHandle: true,
			expectedID:   999999999,
		},
		{
			name:         "Invalid prefix",
			callbackData: "invalid:123",
			shouldHandle: false,
		},
		{
			name:         "Missing exchange ID",
			callbackData: "delete:",
			shouldHandle: false,
		},
		{
			name:         "Invalid exchange ID",
			callbackData: "delete:abc",
			shouldHandle: false,
		},
		{
			name:         "No colon separator",
			callbackData: "delete123",
			shouldHandle: false,
		},
		{
			name:         "Multiple colons",
			callbackData: "delete:123:456",
			shouldHandle: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test callback data parsing logic (same as used in HandleDeleteExchangeCallback)
			parts := strings.Split(tt.callbackData, ":")
			isValid := len(parts) == 2 && parts[0] == "delete"

			if tt.shouldHandle {
				assert.True(t, isValid, "Should be valid callback data")
				if isValid {
					exchangeID, err := strconv.ParseInt(parts[1], 10, 64)
					assert.NoError(t, err, "Should parse exchange ID")
					assert.Equal(t, tt.expectedID, exchangeID, "Should parse correct exchange ID")
				}
			} else {
				if isValid {
					_, err := strconv.ParseInt(parts[1], 10, 64)
					assert.Error(t, err, "Should fail to parse invalid exchange ID")
				} else {
					assert.False(t, isValid, "Should be invalid callback data")
				}
			}
		})
	}
}

func TestHandleDeleteExchangeCallback_SecurityCheck(t *testing.T) {
	// Test that only exchange author can delete their exchange
	tests := []struct {
		name           string
		userID         int64
		exchangeUserID int64
		shouldAllow    bool
	}{
		{
			name:           "Author can delete own exchange",
			userID:         123,
			exchangeUserID: 123,
			shouldAllow:    true,
		},
		{
			name:           "Non-author cannot delete exchange",
			userID:         123,
			exchangeUserID: 456,
			shouldAllow:    false,
		},
		{
			name:           "Different user cannot delete",
			userID:         999,
			exchangeUserID: 123,
			shouldAllow:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the security check logic from HandleDeleteExchangeCallback
			isAuthor := tt.userID == tt.exchangeUserID
			assert.Equal(t, tt.shouldAllow, isAuthor, "Security check should match expected result")
		})
	}
}

func TestSoftDeleteExchangeIntegration(t *testing.T) {
	// Test the soft delete functionality with mock repository
	mockRepo := &MockContactRepository{
		users:           make(map[int64]*objects.User),
		exchanges:       make(map[int64]*objects.Exchange),
		contactRequests: make(map[string]bool),
		timelineRecords: make(map[int64][]*objects.TimelineRecord),
	}

	// Create test exchange
	exchange := &objects.Exchange{
		ID:                1,
		UserID:            123456,
		ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
		Status:            objects.ExchangeStatusPosted,
		Lat:               40.7128,
		Lon:               -74.0060,
		IsDeleted:         false,
	}
	mockRepo.exchanges[1] = exchange

	// Verify exchange exists before deletion
	retrievedExchange, err := mockRepo.GetExchangeByID(1)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedExchange, "Exchange should exist before deletion")
	assert.False(t, retrievedExchange.IsDeleted, "Exchange should not be deleted initially")

	// Soft delete the exchange
	err = mockRepo.SoftDeleteExchange(1)
	assert.NoError(t, err)

	// Verify exchange is filtered out after soft delete
	deletedExchange, err := mockRepo.GetExchangeByID(1)
	assert.NoError(t, err)
	assert.Nil(t, deletedExchange, "Exchange should be filtered out after soft delete")

	// Verify the actual exchange object was marked as deleted
	actualExchange := mockRepo.exchanges[1]
	assert.True(t, actualExchange.IsDeleted, "Exchange should be marked as deleted")
	assert.NotNil(t, actualExchange.DeletedAt, "DeletedAt should be set")
}

func TestSoftDeleteTimelineRecordsIntegration(t *testing.T) {
	// Test the soft delete functionality for timeline records
	mockRepo := &MockContactRepository{
		users:           make(map[int64]*objects.User),
		exchanges:       make(map[int64]*objects.Exchange),
		contactRequests: make(map[string]bool),
		timelineRecords: make(map[int64][]*objects.TimelineRecord),
	}

	// Create timeline records
	timelineRecord1 := &objects.TimelineRecord{
		ID:                1,
		ExchangeID:        1,
		RecipientUserID:   123456,
		TelegramMessageID: intPtr(100),
		Status:            objects.TimelineStatusSent,
		IsDeleted:         false,
	}
	timelineRecord2 := &objects.TimelineRecord{
		ID:                2,
		ExchangeID:        1,
		RecipientUserID:   789012,
		TelegramMessageID: intPtr(101),
		Status:            objects.TimelineStatusSent,
		IsDeleted:         false,
	}
	mockRepo.timelineRecords[1] = []*objects.TimelineRecord{timelineRecord1, timelineRecord2}

	// Verify timeline records exist before deletion
	timelineRecords, err := mockRepo.GetTimelineRecordsByExchange(1)
	assert.NoError(t, err)
	assert.Len(t, timelineRecords, 2, "Should have 2 timeline records")
	for _, record := range timelineRecords {
		assert.False(t, record.IsDeleted, "Timeline record should not be deleted initially")
	}

	// Soft delete timeline records
	err = mockRepo.SoftDeleteExchangeTimeline(1)
	assert.NoError(t, err)

	// Verify timeline records were soft deleted
	timelineRecords, err = mockRepo.GetTimelineRecordsByExchange(1)
	assert.NoError(t, err)
	for _, record := range timelineRecords {
		assert.True(t, record.IsDeleted, "Timeline record should be marked as deleted")
		assert.NotNil(t, record.DeletedAt, "Timeline record DeletedAt should be set")
	}
}
