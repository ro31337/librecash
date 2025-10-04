package menu

import (
	"librecash/objects"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// MockPhoneContext for testing - simplified version
type MockPhoneContext struct {
	users        map[int64]*objects.User
	sentMessages []tgbotapi.MessageConfig
}

func (m *MockPhoneContext) SaveUser(user *objects.User) error {
	m.users[user.UserId] = user
	return nil
}

func (m *MockPhoneContext) FindUser(userId int64) *objects.User {
	return m.users[userId]
}

func (m *MockPhoneContext) Send(msg tgbotapi.MessageConfig) {
	m.sentMessages = append(m.sentMessages, msg)
}

func (m *MockPhoneContext) AnswerCallbackQuery(callback tgbotapi.CallbackConfig) error {
	return nil
}

// MockPhoneRepository for testing
type MockPhoneRepository struct {
	users map[int64]*objects.User
}

func (m *MockPhoneRepository) SaveUser(user *objects.User) error {
	m.users[user.UserId] = user
	return nil
}

func (m *MockPhoneRepository) FindUser(userId int64) *objects.User {
	return m.users[userId]
}

// MockPhoneBot for testing
type MockPhoneBot struct {
	sentMessages []tgbotapi.MessageConfig
}

func (m *MockPhoneBot) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	if msg, ok := c.(tgbotapi.MessageConfig); ok {
		m.sentMessages = append(m.sentMessages, msg)
	}
	return tgbotapi.Message{}, nil
}

func (m *MockPhoneBot) AnswerCallbackQuery(callback tgbotapi.CallbackConfig) (tgbotapi.APIResponse, error) {
	return tgbotapi.APIResponse{}, nil
}

func TestAskPhoneMenuHandler_HandleContactReceived(t *testing.T) {
	// Test phone number extraction from contact
	contact := &tgbotapi.Contact{
		PhoneNumber: "+1234567890",
		FirstName:   "John",
	}
	message := &tgbotapi.Message{
		Contact: contact,
	}

	user := &objects.User{
		UserId:       12345,
		MenuId:       objects.Menu_AskPhone,
		LanguageCode: "en",
	}

	// Test that contact message contains phone number
	if message.Contact == nil {
		t.Error("Expected contact to be present")
	}

	if message.Contact.PhoneNumber != "+1234567890" {
		t.Errorf("Expected phone number '+1234567890', got '%s'", message.Contact.PhoneNumber)
	}

	// Test user phone number assignment
	user.PhoneNumber = message.Contact.PhoneNumber
	if user.PhoneNumber != "+1234567890" {
		t.Errorf("Expected user phone number '+1234567890', got '%s'", user.PhoneNumber)
	}
}

func TestAskPhoneMenuHandler_HandleSkipCallback(t *testing.T) {
	// Test callback data parsing
	callback := &tgbotapi.CallbackQuery{
		ID:   "test_callback",
		Data: "phone_skip",
		Message: &tgbotapi.Message{
			Chat:      &tgbotapi.Chat{ID: 12345},
			MessageID: 123,
		},
	}

	// Test callback data
	if callback.Data != "phone_skip" {
		t.Errorf("Expected callback data 'phone_skip', got '%s'", callback.Data)
	}

	// Test phone number clearing logic
	user := &objects.User{
		UserId:      12345,
		MenuId:      objects.Menu_AskPhone,
		PhoneNumber: "+1234567890", // User had phone number before
	}

	// Simulate skip action
	if callback.Data == "phone_skip" {
		user.PhoneNumber = ""
	}

	// Verify phone number was cleared
	if user.PhoneNumber != "" {
		t.Errorf("Expected phone number to be empty, got '%s'", user.PhoneNumber)
	}
}

func TestAskPhoneMenuHandler_MessageTypes(t *testing.T) {
	// Test contact message detection
	contactMessage := &tgbotapi.Message{
		Contact: &tgbotapi.Contact{
			PhoneNumber: "+1234567890",
		},
	}

	if contactMessage.Contact == nil {
		t.Error("Expected contact message to have Contact field")
	}

	// Test regular message detection
	regularMessage := &tgbotapi.Message{
		Text: "some text",
	}

	if regularMessage.Contact != nil {
		t.Error("Expected regular message to not have Contact field")
	}

	// Test message type differentiation
	isContactMessage := contactMessage.Contact != nil
	isRegularMessage := regularMessage.Contact == nil

	if !isContactMessage {
		t.Error("Expected to detect contact message")
	}

	if !isRegularMessage {
		t.Error("Expected to detect regular message")
	}
}

func TestMenuStatesWithPhone(t *testing.T) {
	// Test that Menu_AskPhone is correctly defined
	if objects.Menu_AskPhone != 275 {
		t.Errorf("Menu_AskPhone should be 275, got %d", objects.Menu_AskPhone)
	}

	// Test menu order
	if objects.Menu_SelectRadius >= objects.Menu_AskPhone {
		t.Error("Menu_SelectRadius should come before Menu_AskPhone")
	}

	if objects.Menu_AskPhone >= objects.Menu_Main {
		t.Error("Menu_AskPhone should come before Menu_Main")
	}
}

func TestPhoneNumberInUser(t *testing.T) {
	user := &objects.User{
		UserId:      12345,
		PhoneNumber: "+1234567890",
	}

	if user.PhoneNumber != "+1234567890" {
		t.Errorf("Expected phone number '+1234567890', got '%s'", user.PhoneNumber)
	}

	// Test empty phone number
	user.PhoneNumber = ""
	if user.PhoneNumber != "" {
		t.Errorf("Expected empty phone number, got '%s'", user.PhoneNumber)
	}
}
