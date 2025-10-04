package menu

import (
	"fmt"
	"librecash/context"
	"librecash/metrics"
	"librecash/objects"
	"librecash/rabbit"
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

	// Notify admin channel about new user
	handler.notifyAdminChannel()

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

func (handler *InitMenuHandler) notifyAdminChannel() {
	// Only notify once per user
	if !handler.context.Repo.ShowCallout(handler.user.UserId, "admin_channel_new_user_notification") {
		log.Printf("[INIT_MENU] Admin already notified about user %d", handler.user.UserId)
		return
	}

	log.Printf("[INIT_MENU] Notifying admin channel about new user %d", handler.user.UserId)

	// Create user identifier for admin notification (no phone for admin)
	// Use English for admin notifications
	userIdentifier := formatUserIdentifier(handler.user, false, "en")

	// Send notification to admin channel
	adminMessage := fmt.Sprintf("New user joined LibreCash: %s\nLanguage: %s",
		userIdentifier, handler.user.GetLanguageName())

	// Note: Using plain text for admin channel (not localized, as per requirements)
	msg := tgbotapi.NewMessage(handler.context.Config.Admin_Channel_Chat_Id, adminMessage)
	msg.ParseMode = "HTML"

	handler.context.RabbitPublish.PublishTgMessage(rabbit.MessageBag{
		Message:  msg,
		Priority: 200, // High priority for admin messages
	})

	// Mark as notified
	handler.context.Repo.DismissCallout(handler.user.UserId, "admin_channel_new_user_notification")

	log.Printf("[INIT_MENU] Admin channel notified about user %d", handler.user.UserId)
}
