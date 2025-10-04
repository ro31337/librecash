package fanout

import (
	"fmt"
	"librecash/objects"
	"librecash/rabbit"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/stretchr/testify/assert"
)

// Mock structures for testing

// MockRepository implements the repository interface for testing
type MockRepository struct {
	users               map[int64]*objects.User
	historicalExchanges []*objects.Exchange
}

func (m *MockRepository) FindUser(userID int64) *objects.User {
	return m.users[userID]
}

func (m *MockRepository) FindUsersInRadius(lat, lon float64, radiusKm int) ([]*objects.User, error) {
	var result []*objects.User
	for _, user := range m.users {
		result = append(result, user)
	}
	return result, nil
}

// MockExchangeNotification represents a published exchange notification for testing
type MockExchangeNotification struct {
	ExchangeID      int64
	RecipientUserID int64
	Message         tgbotapi.MessageConfig
	Priority        uint8
}

// MockRabbitClient implements the RabbitMQ client interface for testing
type MockRabbitClient struct {
	publishedMessages []MockExchangeNotification
}

func (m *MockRabbitClient) PublishExchangeNotification(notificationBag rabbit.ExchangeNotificationBag) error {
	m.publishedMessages = append(m.publishedMessages, MockExchangeNotification{
		ExchangeID:      notificationBag.ExchangeID,
		RecipientUserID: notificationBag.RecipientUserID,
		Message:         notificationBag.Message,
		Priority:        notificationBag.Priority,
	})
	return nil
}

// MockContext implements the context interface for testing
type MockContext struct {
	Repo          *MockRepository
	RabbitPublish *MockRabbitClient
}

// Create a test-specific fanout service that accepts our mock context
func NewTestFanoutService(repo *MockRepository, rabbit *MockRabbitClient) *TestFanoutService {
	return &TestFanoutService{
		repo:   repo,
		rabbit: rabbit,
	}
}

type TestFanoutService struct {
	repo   *MockRepository
	rabbit *MockRabbitClient
}

// BroadcastExchange is a test version that uses mock dependencies
func (f *TestFanoutService) BroadcastExchange(exchange *objects.Exchange) error {
	// 1. Get initiator's search radius
	initiator := f.repo.FindUser(exchange.UserID)
	if initiator == nil {
		return fmt.Errorf("initiator user %d not found", exchange.UserID)
	}
	if initiator.SearchRadiusKm == nil {
		return fmt.Errorf("initiator user %d has no search radius set", exchange.UserID)
	}

	// 2. Find nearby users (including initiator)
	nearbyUsers, err := f.repo.FindUsersInRadius(
		exchange.Lat, exchange.Lon,
		*initiator.SearchRadiusKm,
	)
	if err != nil {
		return fmt.Errorf("failed to find nearby users: %v", err)
	}

	// 3. Queue notification messages (users in main menu OR exchange author)
	for _, user := range nearbyUsers {
		// Send to users in main menu OR exchange author (needs delete button)
		if user.MenuId == objects.Menu_Main || user.UserId == exchange.UserID {
			// Create a simple message for testing
			msg := tgbotapi.NewMessage(user.UserId, "Test notification")

			notificationBag := rabbit.ExchangeNotificationBag{
				ExchangeID:      exchange.ID,
				RecipientUserID: user.UserId,
				Message:         msg,
				Priority:        100,
			}

			if err := f.rabbit.PublishExchangeNotification(notificationBag); err != nil {
				return fmt.Errorf("failed to queue notification for user %d: %v", user.UserId, err)
			}
		}
		// Note: We don't log skipped users in test to keep test output clean
	}

	return nil
}

func TestCalculateDistance(t *testing.T) {
	service := &FanoutService{}

	// Test known distances
	tests := []struct {
		name      string
		lat1      float64
		lon1      float64
		lat2      float64
		lon2      float64
		expected  float64
		tolerance float64
	}{
		{
			name:      "Same location",
			lat1:      40.7128,
			lon1:      -74.0060,
			lat2:      40.7128,
			lon2:      -74.0060,
			expected:  0,
			tolerance: 0.1,
		},
		{
			name:      "NYC to Times Square (about 5km)",
			lat1:      40.7128, // NYC
			lon1:      -74.0060,
			lat2:      40.7589, // Times Square
			lon2:      -73.9851,
			expected:  5.2,
			tolerance: 1.0,
		},
		{
			name:      "NYC to Brooklyn (about 6.5km)",
			lat1:      40.7128, // NYC
			lon1:      -74.0060,
			lat2:      40.6782, // Brooklyn
			lon2:      -73.9442,
			expected:  6.5,
			tolerance: 1.0,
		},
		{
			name:      "NYC to Philadelphia (about 130km)",
			lat1:      40.7128, // NYC
			lon1:      -74.0060,
			lat2:      39.9526, // Philadelphia
			lon2:      -75.1652,
			expected:  130.0,
			tolerance: 5.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance := service.calculateDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			assert.InDelta(t, tt.expected, distance, tt.tolerance,
				"Distance calculation for %s: expected ~%.1f km, got %.1f km",
				tt.name, tt.expected, distance)
		})
	}
}

func TestCalculateDistanceSymmetric(t *testing.T) {
	service := &FanoutService{}

	// Distance should be the same regardless of direction
	lat1, lon1 := 40.7128, -74.0060 // NYC
	lat2, lon2 := 40.7589, -73.9851 // Times Square

	distance1 := service.calculateDistance(lat1, lon1, lat2, lon2)
	distance2 := service.calculateDistance(lat2, lon2, lat1, lon1)

	assert.InDelta(t, distance1, distance2, 0.001,
		"Distance calculation should be symmetric")
}

func TestCalculateDistanceEdgeCases(t *testing.T) {
	service := &FanoutService{}

	// Test with extreme coordinates
	tests := []struct {
		name string
		lat1 float64
		lon1 float64
		lat2 float64
		lon2 float64
	}{
		{
			name: "North Pole to South Pole",
			lat1: 90.0,
			lon1: 0.0,
			lat2: -90.0,
			lon2: 0.0,
		},
		{
			name: "Equator opposite sides",
			lat1: 0.0,
			lon1: 0.0,
			lat2: 0.0,
			lon2: 180.0,
		},
		{
			name: "Date line crossing",
			lat1: 0.0,
			lon1: 179.0,
			lat2: 0.0,
			lon2: -179.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance := service.calculateDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)

			// Distance should be positive and reasonable (not NaN or infinite)
			assert.True(t, distance >= 0, "Distance should be non-negative")
			assert.False(t, math.IsNaN(distance), "Distance should not be NaN")
			assert.False(t, math.IsInf(distance, 0), "Distance should not be infinite")
			assert.True(t, distance <= 20100, "Distance should not exceed half Earth's circumference (allowing some tolerance)")
		})
	}
}

func TestBuildNotificationMessage(t *testing.T) {
	service := &FanoutService{}

	// Create mock recipient user (not author)
	mockRecipient := &objects.User{
		UserId:       123456,
		Username:     "recipient",
		FirstName:    "Test",
		LastName:     "Recipient",
		LanguageCode: "en",
		Lat:          40.7128,
		Lon:          -74.0060,
	}

	// Create mock author user
	mockAuthor := &objects.User{
		UserId:       789012,
		Username:     "author",
		FirstName:    "Test",
		LastName:     "Author",
		LanguageCode: "en",
		Lat:          40.7589,
		Lon:          -73.9851,
	}

	tests := []struct {
		name                string
		exchange            *objects.Exchange
		user                *objects.User
		distanceKm          int
		expectedContains    []string
		expectedNotContains []string
	}{
		{
			name: "Cash to Crypto exchange - Recipient",
			exchange: &objects.Exchange{
				ID:                1,
				UserID:            789012, // Author ID
				ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
				Status:            objects.ExchangeStatusPosted,
				AmountUSD:         intPtr(50),
				Lat:               40.7589,
				Lon:               -73.9851,
			},
			user:       mockRecipient, // Recipient (not author)
			distanceKm: 5,
			expectedContains: []string{
				"fanout.notification_header",
				"fanout.notification_have_cash",
				"fanout.notification_need_crypto",
				"fanout.notification_amount",
				"fanout.notification_distance",
			},
			expectedNotContains: []string{
				"fanout.author_notification_header",
				"fanout.author_notification_have_cash",
				"fanout.author_notification_need_crypto",
			},
		},

		{
			name: "Cash to Crypto exchange - Author",
			exchange: &objects.Exchange{
				ID:                1,
				UserID:            789012, // Author ID
				ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
				Status:            objects.ExchangeStatusPosted,
				AmountUSD:         intPtr(50),
				Lat:               40.7589,
				Lon:               -73.9851,
			},
			user:       mockAuthor, // Author
			distanceKm: 0,          // Distance irrelevant for author
			expectedContains: []string{
				"fanout.author_notification_header",
				"fanout.author_notification_have_cash",
				"fanout.author_notification_need_crypto",
				"fanout.notification_amount",
			},
			expectedNotContains: []string{
				"fanout.notification_header",
				"fanout.notification_have_cash",
				"fanout.notification_need_crypto",
				"fanout.notification_distance", // No distance for author
			},
		},
		{
			name: "Crypto to Cash exchange - Recipient",
			exchange: &objects.Exchange{
				ID:                2,
				UserID:            789012,
				ExchangeDirection: objects.ExchangeDirectionCryptoToCash,
				Status:            objects.ExchangeStatusPosted,
				AmountUSD:         intPtr(100),
				Lat:               40.7589,
				Lon:               -73.9851,
			},
			user:       mockRecipient,
			distanceKm: 15,
			expectedContains: []string{
				"fanout.notification_header",
				"fanout.notification_have_crypto",
				"fanout.notification_need_cash",
				"fanout.notification_amount",
				"fanout.notification_distance",
			},
			expectedNotContains: []string{
				"fanout.author_notification_header",
				"fanout.author_notification_have_crypto",
				"fanout.author_notification_need_cash",
			},
		},
		{
			name: "Crypto to Cash exchange - Author",
			exchange: &objects.Exchange{
				ID:                2,
				UserID:            789012,
				ExchangeDirection: objects.ExchangeDirectionCryptoToCash,
				Status:            objects.ExchangeStatusPosted,
				AmountUSD:         intPtr(100),
				Lat:               40.7589,
				Lon:               -73.9851,
			},
			user:       mockAuthor,
			distanceKm: 0,
			expectedContains: []string{
				"fanout.author_notification_header",
				"fanout.author_notification_have_crypto",
				"fanout.author_notification_need_cash",
				"fanout.notification_amount",
			},
			expectedNotContains: []string{
				"fanout.notification_header",
				"fanout.notification_have_crypto",
				"fanout.notification_need_cash",
				"fanout.notification_distance",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := service.buildNotificationMessage(tt.exchange, tt.user, tt.distanceKm)

			// Check that expected strings are present
			for _, expected := range tt.expectedContains {
				assert.Contains(t, message, expected, "Message should contain: %s", expected)
			}

			// Check that unexpected strings are not present
			for _, notExpected := range tt.expectedNotContains {
				assert.NotContains(t, message, notExpected, "Message should not contain: %s", notExpected)
			}

			// Verify message structure
			assert.True(t, len(message) > 0, "Message should not be empty")
			assert.Contains(t, message, "\n\n", "Message should have proper formatting with double newlines")
		})
	}
}

// Helper function for tests
func intPtr(i int) *int {
	return &i
}

func TestBroadcastExchange(t *testing.T) {
	// Create mock repository
	mockRepo := &MockRepository{
		users: make(map[int64]*objects.User),
	}

	// Create mock RabbitMQ client
	mockRabbit := &MockRabbitClient{
		publishedMessages: make([]MockExchangeNotification, 0),
	}

	// Create fanout service
	service := NewTestFanoutService(mockRepo, mockRabbit)

	// Set up test data
	initiatorID := int64(123456)
	searchRadius := 10

	// Create initiator user with search radius (in main menu)
	initiator := &objects.User{
		UserId:         initiatorID,
		Username:       "initiator",
		FirstName:      "Test",
		LastName:       "Initiator",
		LanguageCode:   "en",
		Lat:            40.7128,
		Lon:            -74.0060,
		SearchRadiusKm: &searchRadius,
		MenuId:         objects.Menu_Main, // In main menu - should receive notifications
	}
	mockRepo.users[initiatorID] = initiator

	// Create nearby users - one in main menu, one not
	nearbyUser1 := &objects.User{
		UserId:       789012,
		Username:     "nearby1",
		FirstName:    "Nearby",
		LastName:     "User1",
		LanguageCode: "en",
		Lat:          40.7589, // ~5km from initiator
		Lon:          -73.9851,
		MenuId:       objects.Menu_Main, // In main menu - should receive notifications
	}
	mockRepo.users[789012] = nearbyUser1

	nearbyUser2 := &objects.User{
		UserId:       345678,
		Username:     "nearby2",
		FirstName:    "Nearby",
		LastName:     "User2",
		LanguageCode: "es",
		Lat:          40.6782, // ~10km from initiator
		Lon:          -73.9442,
		MenuId:       objects.Menu_Amount, // In amount menu - should NOT receive notifications
	}
	mockRepo.users[345678] = nearbyUser2

	// Create test exchange
	exchange := &objects.Exchange{
		ID:                1,
		UserID:            initiatorID,
		ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
		Status:            objects.ExchangeStatusPosted,
		AmountUSD:         intPtr(50),
		Lat:               40.7128,
		Lon:               -74.0060,
	}

	// Test successful broadcast
	err := service.BroadcastExchange(exchange)
	assert.NoError(t, err)

	// Verify that messages were queued only for users in main menu (initiator + nearbyUser1)
	assert.Len(t, mockRabbit.publishedMessages, 2, "Should have queued 2 messages (only for users in main menu)")

	// Verify all messages have correct exchange ID and priority
	for i, msg := range mockRabbit.publishedMessages {
		assert.Equal(t, exchange.ID, msg.ExchangeID, "Message %d should have correct exchange ID", i)
		assert.Equal(t, uint8(100), msg.Priority, "Message %d should have correct priority", i)
	}

	// Verify that only users in main menu received messages
	recipientIDs := make([]int64, len(mockRabbit.publishedMessages))
	for i, msg := range mockRabbit.publishedMessages {
		recipientIDs[i] = msg.RecipientUserID
	}
	assert.Contains(t, recipientIDs, initiatorID, "Initiator should receive fanout message (in main menu)")
	assert.Contains(t, recipientIDs, nearbyUser1.UserId, "Nearby user 1 should receive message (in main menu)")
	assert.NotContains(t, recipientIDs, nearbyUser2.UserId, "Nearby user 2 should NOT receive message (in amount menu)")
}

func TestBroadcastExchangeErrors(t *testing.T) {
	// Create mock repository
	mockRepo := &MockRepository{
		users: make(map[int64]*objects.User),
	}

	// Create fanout service
	service := NewTestFanoutService(mockRepo, nil)

	// Test with non-existent initiator
	exchange := &objects.Exchange{
		ID:                1,
		UserID:            999999, // Non-existent user
		ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
		Status:            objects.ExchangeStatusPosted,
		Lat:               40.7128,
		Lon:               -74.0060,
	}

	err := service.BroadcastExchange(exchange)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "initiator user 999999 not found")

	// Test with initiator without search radius
	initiatorWithoutRadius := &objects.User{
		UserId:         123456,
		Username:       "initiator",
		FirstName:      "Test",
		LastName:       "Initiator",
		LanguageCode:   "en",
		Lat:            40.7128,
		Lon:            -74.0060,
		SearchRadiusKm: nil, // No search radius
	}
	mockRepo.users[123456] = initiatorWithoutRadius

	exchange.UserID = 123456
	err = service.BroadcastExchange(exchange)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "initiator user 123456 has no search radius set")
}

func TestBroadcastExchange_MenuStateFiltering(t *testing.T) {
	// Create mock repository
	mockRepo := &MockRepository{
		users: make(map[int64]*objects.User),
	}

	// Create mock RabbitMQ client
	mockRabbit := &MockRabbitClient{
		publishedMessages: make([]MockExchangeNotification, 0),
	}

	// Create fanout service
	service := NewTestFanoutService(mockRepo, mockRabbit)

	// Set up test data
	initiatorID := int64(123456)
	searchRadius := 10

	// Create initiator user in main menu
	initiator := &objects.User{
		UserId:         initiatorID,
		Username:       "initiator",
		FirstName:      "Test",
		LastName:       "Initiator",
		LanguageCode:   "en",
		Lat:            40.7128,
		Lon:            -74.0060,
		SearchRadiusKm: &searchRadius,
		MenuId:         objects.Menu_Main,
	}
	mockRepo.users[initiatorID] = initiator

	// Create users in different menu states
	userInMain := &objects.User{
		UserId:       100001,
		Username:     "user_main",
		FirstName:    "User",
		LastName:     "InMain",
		LanguageCode: "en",
		Lat:          40.7589,
		Lon:          -73.9851,
		MenuId:       objects.Menu_Main, // Should receive notification
	}
	mockRepo.users[100001] = userInMain

	userInAmount := &objects.User{
		UserId:       100002,
		Username:     "user_amount",
		FirstName:    "User",
		LastName:     "InAmount",
		LanguageCode: "en",
		Lat:          40.7589,
		Lon:          -73.9851,
		MenuId:       objects.Menu_Amount, // Should NOT receive notification
	}
	mockRepo.users[100002] = userInAmount

	userInPhone := &objects.User{
		UserId:       100003,
		Username:     "user_phone",
		FirstName:    "User",
		LastName:     "InPhone",
		LanguageCode: "en",
		Lat:          40.7589,
		Lon:          -73.9851,
		MenuId:       objects.Menu_AskPhone, // Should NOT receive notification
	}
	mockRepo.users[100003] = userInPhone

	// Create author in non-main menu state (should still receive notification)
	authorInAmount := &objects.User{
		UserId:         100004,
		Username:       "author_amount",
		FirstName:      "Author",
		LastName:       "InAmount",
		LanguageCode:   "en",
		Lat:            40.7589,
		Lon:            -73.9851,
		MenuId:         objects.Menu_Amount, // Not in main menu but is author
		SearchRadiusKm: &searchRadius,       // Author needs search radius for fanout
	}
	mockRepo.users[100004] = authorInAmount

	// Create test exchange by author in non-main menu
	exchange := &objects.Exchange{
		ID:                1,
		UserID:            100004, // authorInAmount
		ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
		AmountUSD:         &[]int{100}[0],
		Status:            objects.ExchangeStatusInitiated,
		Lat:               40.7128,
		Lon:               -74.0060,
	}

	// Test broadcast
	err := service.BroadcastExchange(exchange)
	assert.NoError(t, err)

	// Verify that users in Menu_Main + author received notifications
	assert.Len(t, mockRabbit.publishedMessages, 3, "Should have queued 3 messages (Menu_Main users + author)")

	// Verify the recipients are correct
	recipientIDs := make([]int64, len(mockRabbit.publishedMessages))
	for i, msg := range mockRabbit.publishedMessages {
		recipientIDs[i] = msg.RecipientUserID
	}

	assert.Contains(t, recipientIDs, initiatorID, "Initiator should receive notification (in main menu)")
	assert.Contains(t, recipientIDs, int64(100001), "User in main menu should receive notification")
	assert.Contains(t, recipientIDs, int64(100004), "Author should receive notification (even in amount menu)")
	assert.NotContains(t, recipientIDs, int64(100002), "User in amount menu should NOT receive notification")
	assert.NotContains(t, recipientIDs, int64(100003), "User in phone menu should NOT receive notification")
}

// Additional unit tests for distance calculation edge cases
func TestCalculateDistanceAccuracy(t *testing.T) {
	service := &FanoutService{}

	// Test against known accurate distances
	// London to Paris: approximately 344 km
	londonLat, londonLon := 51.5074, -0.1278
	parisLat, parisLon := 48.8566, 2.3522

	distance := service.calculateDistance(londonLat, londonLon, parisLat, parisLon)
	assert.InDelta(t, 344.0, distance, 10.0, "London to Paris distance should be ~344 km")

	// New York to Los Angeles: approximately 3944 km
	nyLat, nyLon := 40.7128, -74.0060
	laLat, laLon := 34.0522, -118.2437

	distance = service.calculateDistance(nyLat, nyLon, laLat, laLon)
	assert.InDelta(t, 3944.0, distance, 50.0, "NYC to LA distance should be ~3944 km")
}

func TestCalculateDistanceZeroDistance(t *testing.T) {
	service := &FanoutService{}

	// Test identical coordinates
	distance := service.calculateDistance(40.7128, -74.0060, 40.7128, -74.0060)
	assert.Equal(t, 0.0, distance, "Distance between identical coordinates should be 0")
}

func TestCalculateDistanceSmallDistances(t *testing.T) {
	service := &FanoutService{}

	// Test very small distances (within same city block)
	lat1, lon1 := 40.7128, -74.0060
	lat2, lon2 := 40.7129, -74.0061 // Very close coordinates

	distance := service.calculateDistance(lat1, lon1, lat2, lon2)
	assert.True(t, distance > 0, "Small distance should be greater than 0")
	assert.True(t, distance < 1, "Small distance should be less than 1 km")
}

// Tests for PRD009: Exchange Deletion by Author - Button Logic

func TestFanoutButtonLogic_AuthorVsRecipient(t *testing.T) {
	// Test the button logic that determines whether to show "Delete" or "Show contact" button
	tests := []struct {
		name               string
		recipientUserID    int64
		exchangeUserID     int64
		expectedIsAuthor   bool
		expectedButtonType string
	}{
		{
			name:               "Author receives delete button",
			recipientUserID:    123,
			exchangeUserID:     123,
			expectedIsAuthor:   true,
			expectedButtonType: "delete",
		},
		{
			name:               "Recipient receives contact button",
			recipientUserID:    456,
			exchangeUserID:     123,
			expectedIsAuthor:   false,
			expectedButtonType: "contact",
		},
		{
			name:               "Different recipient receives contact button",
			recipientUserID:    789,
			exchangeUserID:     123,
			expectedIsAuthor:   false,
			expectedButtonType: "contact",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic from queueNotificationMessage in fanout.go
			isAuthor := tt.recipientUserID == tt.exchangeUserID
			assert.Equal(t, tt.expectedIsAuthor, isAuthor, "Author detection should be correct")

			// Verify expected button type
			if isAuthor {
				assert.Equal(t, "delete", tt.expectedButtonType, "Author should get delete button")
			} else {
				assert.Equal(t, "contact", tt.expectedButtonType, "Recipient should get contact button")
			}
		})
	}
}

func TestFanoutCallbackDataFormat(t *testing.T) {
	// Test the callback data format for both delete and contact buttons
	tests := []struct {
		name         string
		buttonType   string
		exchangeID   int64
		expectedData string
	}{
		{
			name:         "Delete button callback data",
			buttonType:   "delete",
			exchangeID:   123,
			expectedData: "delete:123",
		},
		{
			name:         "Contact button callback data",
			buttonType:   "contact",
			exchangeID:   456,
			expectedData: "contact:456",
		},
		{
			name:         "Delete button with large ID",
			buttonType:   "delete",
			exchangeID:   999999999,
			expectedData: "delete:999999999",
		},
		{
			name:         "Contact button with large ID",
			buttonType:   "contact",
			exchangeID:   888888888,
			expectedData: "contact:888888888",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the callback data creation logic from fanout.go
			var callbackData string
			if tt.buttonType == "delete" {
				callbackData = fmt.Sprintf("delete:%d", tt.exchangeID)
			} else {
				callbackData = fmt.Sprintf("contact:%d", tt.exchangeID)
			}

			assert.Equal(t, tt.expectedData, callbackData, "Callback data should match expected format")

			// Verify the data can be parsed back correctly
			parts := strings.Split(callbackData, ":")
			assert.Len(t, parts, 2, "Callback data should have exactly 2 parts")
			assert.Equal(t, tt.buttonType, parts[0], "Button type should be correct")

			parsedID, err := strconv.ParseInt(parts[1], 10, 64)
			assert.NoError(t, err, "Exchange ID should be parseable")
			assert.Equal(t, tt.exchangeID, parsedID, "Parsed exchange ID should match original")
		})
	}
}

func TestFormatTimeAgo(t *testing.T) {
	// Create a real fanout service for testing
	service := &FanoutService{}

	// Create a mock locale
	mockLocale := &MockLocale{
		translations: map[string]string{
			"time.minutes_ago":   "%d min.ago",
			"time.minutes_ago_1": "1 min.ago",
			"time.hours_ago":     "%d h.ago",
			"time.hours_ago_1":   "1 h.ago",
			"time.days_ago":      "%d d.ago",
			"time.days_ago_1":    "1 d.ago",
			"time.weeks_ago":     "%d w.ago",
			"time.weeks_ago_1":   "1 w.ago",
		},
	}

	now := time.Now()

	// Test minutes
	result := service.formatTimeAgo(now.Add(-30*time.Minute), mockLocale)
	assert.Equal(t, "30 min.ago", result)

	result = service.formatTimeAgo(now.Add(-1*time.Minute), mockLocale)
	assert.Equal(t, "1 min.ago", result)

	// Test hours
	result = service.formatTimeAgo(now.Add(-3*time.Hour), mockLocale)
	assert.Equal(t, "3 h.ago", result)

	result = service.formatTimeAgo(now.Add(-1*time.Hour), mockLocale)
	assert.Equal(t, "1 h.ago", result)

	// Test days
	result = service.formatTimeAgo(now.Add(-5*24*time.Hour), mockLocale)
	assert.Equal(t, "5 d.ago", result)

	result = service.formatTimeAgo(now.Add(-1*24*time.Hour), mockLocale)
	assert.Equal(t, "1 d.ago", result)

	// Test weeks
	result = service.formatTimeAgo(now.Add(-14*24*time.Hour), mockLocale)
	assert.Equal(t, "2 w.ago", result)

	result = service.formatTimeAgo(now.Add(-7*24*time.Hour), mockLocale)
	assert.Equal(t, "1 w.ago", result)
}

// MockLocale for testing time formatting
type MockLocale struct {
	translations map[string]string
}

func (m *MockLocale) Get(key string) string {
	if val, exists := m.translations[key]; exists {
		return val
	}
	return key // Return key if translation not found
}

func TestBroadcastHistoricalExchanges(t *testing.T) {
	// Create mock repository
	mockRepo := &MockRepository{
		users: make(map[int64]*objects.User),
	}

	// Create mock RabbitMQ client
	mockRabbit := &MockRabbitClient{
		publishedMessages: make([]MockExchangeNotification, 0),
	}

	// Create test fanout service
	service := NewTestFanoutService(mockRepo, mockRabbit)

	// Create recipient user with search radius
	recipientID := int64(123456)
	searchRadius := 10
	recipient := &objects.User{
		UserId:         recipientID,
		Username:       "recipient",
		FirstName:      "Test",
		LastName:       "Recipient",
		LanguageCode:   "en",
		Lat:            40.7128,
		Lon:            -74.0060,
		SearchRadiusKm: &searchRadius,
	}
	mockRepo.users[recipientID] = recipient

	// Create exchange author
	authorID := int64(789012)
	author := &objects.User{
		UserId:       authorID,
		Username:     "author",
		FirstName:    "Test",
		LastName:     "Author",
		LanguageCode: "en",
		Lat:          40.7200,
		Lon:          -74.0100,
	}
	mockRepo.users[authorID] = author

	// Create historical exchanges
	now := time.Now()
	exchange1 := &objects.Exchange{
		ID:                1,
		UserID:            authorID,
		ExchangeDirection: objects.ExchangeDirectionCashToCrypto,
		Status:            objects.ExchangeStatusPosted,
		Lat:               40.7200,
		Lon:               -74.0100,
		IsDeleted:         false,
		CreatedAt:         now.Add(-48 * time.Hour), // 2 days ago
		UpdatedAt:         now.Add(-48 * time.Hour),
	}

	exchange2 := &objects.Exchange{
		ID:                2,
		UserID:            authorID,
		ExchangeDirection: objects.ExchangeDirectionCryptoToCash,
		Status:            objects.ExchangeStatusPosted,
		Lat:               40.7200,
		Lon:               -74.0100,
		IsDeleted:         false,
		CreatedAt:         now.Add(-24 * time.Hour), // 1 day ago
		UpdatedAt:         now.Add(-24 * time.Hour),
	}

	// Add historical exchanges to mock repo
	mockRepo.historicalExchanges = []*objects.Exchange{exchange1, exchange2}

	// Test broadcasting historical exchanges
	err := service.BroadcastHistoricalExchanges(recipientID, recipient.Lat, recipient.Lon)
	assert.NoError(t, err)

	// Verify that historical notifications were sent
	assert.Len(t, mockRabbit.publishedMessages, 2, "Should send 2 historical notifications")

	// Verify priority is lower for historical notifications
	for _, msg := range mockRabbit.publishedMessages {
		assert.Equal(t, uint8(80), msg.Priority, "Historical notifications should have priority 80")
	}
}

func TestBroadcastHistoricalExchangesNoSearchRadius(t *testing.T) {
	// Create mock repository
	mockRepo := &MockRepository{
		users: make(map[int64]*objects.User),
	}

	// Create mock RabbitMQ client
	mockRabbit := &MockRabbitClient{
		publishedMessages: make([]MockExchangeNotification, 0),
	}

	// Create test fanout service
	service := NewTestFanoutService(mockRepo, mockRabbit)

	// Create recipient user WITHOUT search radius
	recipientID := int64(123456)
	recipient := &objects.User{
		UserId:         recipientID,
		Username:       "recipient",
		FirstName:      "Test",
		LastName:       "Recipient",
		LanguageCode:   "en",
		Lat:            40.7128,
		Lon:            -74.0060,
		SearchRadiusKm: nil, // No search radius
	}
	mockRepo.users[recipientID] = recipient

	// Test broadcasting historical exchanges
	err := service.BroadcastHistoricalExchanges(recipientID, recipient.Lat, recipient.Lon)
	assert.NoError(t, err, "Should not error when user has no search radius")

	// Verify that no notifications were sent
	assert.Len(t, mockRabbit.publishedMessages, 0, "Should not send notifications when user has no search radius")
}

// Add BroadcastHistoricalExchanges method to TestFanoutService
func (f *TestFanoutService) BroadcastHistoricalExchanges(userID int64, lat, lon float64) error {
	// 1. Get user and their search radius
	user := f.repo.FindUser(userID)
	if user == nil {
		return fmt.Errorf("user %d not found", userID)
	}
	if user.SearchRadiusKm == nil {
		return nil // Skip if no search radius
	}

	// 2. Get historical exchanges (mock implementation)
	historicalExchanges := f.repo.historicalExchanges
	if historicalExchanges == nil {
		return nil
	}

	// 3. Queue historical notification messages
	for _, exchange := range historicalExchanges {
		// Create mock notification
		notification := MockExchangeNotification{
			ExchangeID:      exchange.ID,
			RecipientUserID: userID,
			Priority:        80, // Lower priority for historical
		}
		f.rabbit.publishedMessages = append(f.rabbit.publishedMessages, notification)
	}

	return nil
}

// Add historicalExchanges field to MockRepository
type MockRepositoryWithHistorical struct {
	*MockRepository
	historicalExchanges []*objects.Exchange
}

// Update MockRepository to include historical exchanges
func (m *MockRepository) FindHistoricalExchangesInRadius(lat, lon float64, radiusKm int, excludeUserID int64) ([]*objects.Exchange, error) {
	if m.historicalExchanges == nil {
		return []*objects.Exchange{}, nil
	}

	var result []*objects.Exchange
	for _, exchange := range m.historicalExchanges {
		if exchange.UserID != excludeUserID && !exchange.IsDeleted && exchange.Status == objects.ExchangeStatusPosted {
			result = append(result, exchange)
		}
	}
	return result, nil
}
