package menu

import (
	"reflect"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/golang/mock/gomock"
)

// MockBotAPI is a mock of BotAPI interface
type MockBotAPI struct {
	ctrl     *gomock.Controller
	recorder *MockBotAPIMockRecorder
}

// MockBotAPIMockRecorder is the mock recorder for MockBotAPI
type MockBotAPIMockRecorder struct {
	mock *MockBotAPI
}

// NewMockBotAPI creates a new mock instance
func NewMockBotAPI(ctrl *gomock.Controller) *MockBotAPI {
	mock := &MockBotAPI{ctrl: ctrl}
	mock.recorder = &MockBotAPIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockBotAPI) EXPECT() *MockBotAPIMockRecorder {
	return m.recorder
}

// Send mocks base method
func (m *MockBotAPI) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Send", c)
	ret0, _ := ret[0].(tgbotapi.Message)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Send indicates an expected call of Send
func (mr *MockBotAPIMockRecorder) Send(c interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockBotAPI)(nil).Send), c)
}

// Request mocks base method
func (m *MockBotAPI) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Request", c)
	ret0, _ := ret[0].(*tgbotapi.APIResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Request indicates an expected call of Request
func (mr *MockBotAPIMockRecorder) Request(c interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Request", reflect.TypeOf((*MockBotAPI)(nil).Request), c)
}

// GetUpdatesChan mocks base method
func (m *MockBotAPI) GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUpdatesChan", config)
	ret0, _ := ret[0].(tgbotapi.UpdatesChannel)
	return ret0
}

// GetUpdatesChan indicates an expected call of GetUpdatesChan
func (mr *MockBotAPIMockRecorder) GetUpdatesChan(config interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUpdatesChan", reflect.TypeOf((*MockBotAPI)(nil).GetUpdatesChan), config)
}

// StopReceivingUpdates mocks base method
func (m *MockBotAPI) StopReceivingUpdates() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "StopReceivingUpdates")
}

// StopReceivingUpdates indicates an expected call of StopReceivingUpdates
func (mr *MockBotAPIMockRecorder) StopReceivingUpdates() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StopReceivingUpdates", reflect.TypeOf((*MockBotAPI)(nil).StopReceivingUpdates))
}
