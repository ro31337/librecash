package menu

import (
	"librecash/context"
	"librecash/metrics"
	"librecash/objects"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type USComplianceMenu struct{}

func NewUSComplianceMenu() *USComplianceMenu {
	return &USComplianceMenu{}
}

func (m *USComplianceMenu) Handle(user *objects.User, context *context.Context, message *tgbotapi.Message) {
	log.Printf("[US_COMPLIANCE] Showing compliance check for user %d", user.UserId)

	locale := user.Locale()

	// Create inline keyboard with Yes/No buttons
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				locale.Get("us_compliance.button_yes"),
				"us_compliance_yes",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				locale.Get("us_compliance.button_no"),
				"us_compliance_no",
			),
		),
	)

	// Send compliance question
	msg := tgbotapi.NewMessage(user.UserId, locale.Get("us_compliance.question"))
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "Markdown"

	context.Send(msg)
}

func (m *USComplianceMenu) HandleCallback(user *objects.User, context *context.Context, callback *tgbotapi.CallbackQuery) {
	log.Printf("[US_COMPLIANCE] Handling callback from user %d: %s", user.UserId, callback.Data)

	// Answer the callback to remove loading state
	callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
	context.AnswerCallbackQuery(callbackAnswer)

	switch callback.Data {
	case "us_compliance_yes":
		// User is US person or plans commercial use - block them
		log.Printf("[US_COMPLIANCE] User %d answered YES - blocking access", user.UserId)

		// Log compliance violation for audit
		log.Printf("[COMPLIANCE_AUDIT] User %d blocked: US person or commercial use", user.UserId)

		// Update user state to blocked
		oldMenuId := user.MenuId
		user.MenuId = objects.Menu_Blocked
		context.Repo.SaveUser(user)

		// Record menu transition metric
		metrics.RecordMenuTransition(oldMenuId, user.MenuId, user.GetSupportedLanguageCode())

		// Remove the inline keyboard by editing the message
		locale := user.Locale()
		editMsg := tgbotapi.NewEditMessageText(
			user.UserId,
			callback.Message.MessageID,
			locale.Get("us_compliance.question"),
		)
		editMsg.ParseMode = "HTML"
		context.EditMessage(editMsg)

		// Show blocked menu
		// Continue menu processing after state change
		ContinueMenuProcessing(context, user.UserId)

	case "us_compliance_no":
		// User is not US person and not for commercial use - proceed
		log.Printf("[US_COMPLIANCE] User %d answered NO - allowing access", user.UserId)

		// Log compliance approval for audit
		log.Printf("[COMPLIANCE_AUDIT] User %d approved: Non-US person, testing only", user.UserId)

		// Update user state to init menu
		oldMenuId := user.MenuId
		user.MenuId = objects.Menu_Init
		context.Repo.SaveUser(user)

		// Record menu transition metric
		metrics.RecordMenuTransition(oldMenuId, user.MenuId, user.GetSupportedLanguageCode())

		// Remove the inline keyboard by editing the message
		locale := user.Locale()
		editMsg := tgbotapi.NewEditMessageText(
			user.UserId,
			callback.Message.MessageID,
			locale.Get("us_compliance.question"),
		)
		editMsg.ParseMode = "HTML"
		context.EditMessage(editMsg)

		// Proceed to init menu (language selection)
		// Continue menu processing after state change
		ContinueMenuProcessing(context, user.UserId)

	default:
		log.Printf("[US_COMPLIANCE] Unknown callback data: %s", callback.Data)
	}
}
