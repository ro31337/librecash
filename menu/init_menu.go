package menu

import (
	"fmt"
	"librecash/context"
	"librecash/metrics"
	"librecash/objects"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type InitMenuHandler struct {
	context *context.Context
	user    *objects.User
}

func NewInitMenu() *InitMenuHandler {
	return &InitMenuHandler{}
}

func (handler *InitMenuHandler) Handle(user *objects.User, context *context.Context, message *tgbotapi.Message) {
	startTime := time.Now()
	log.Printf("[INIT_MENU] Handling /start for user %d", user.UserId)

	handler.context = context
	handler.user = user

	// Send welcome message in user's language
	handler.sendWelcomeMessage()

	// Update user state to ask for search radius first
	oldMenuId := user.MenuId
	user.MenuId = objects.Menu_SelectRadius
	context.Repo.SaveUser(user)

	// Record menu transition metric
	metrics.RecordMenuTransition(oldMenuId, user.MenuId, user.GetSupportedLanguageCode())

	duration := time.Since(startTime)
	log.Printf("[INIT_MENU] User %d initialization complete (duration: %v)", user.UserId, duration)
}

func (handler *InitMenuHandler) sendWelcomeMessage() {
	log.Printf("[INIT_MENU] Sending welcome message to user %d in language: %s",
		handler.user.UserId, handler.user.GetSupportedLanguageCode())

	// Get the language name for display
	languageName := handler.user.GetLanguageName()

	// Format the welcome message with the language name
	welcomeTemplate := handler.user.Locale().Get("init_menu.welcome")
	welcomeMessage := fmt.Sprintf(welcomeTemplate, languageName)

	msg := tgbotapi.NewMessage(handler.user.UserId, welcomeMessage)
	msg.DisableWebPagePreview = true

	handler.context.Send(msg)

	log.Printf("[INIT_MENU] Welcome message sent to user %d", handler.user.UserId)
}
