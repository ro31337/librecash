package menu

import (
	"librecash/context"
	"librecash/metrics"
	"librecash/objects"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type HistoricalFanoutWaitMenu struct{}

func NewHistoricalFanoutWaitMenu() *HistoricalFanoutWaitMenu {
	return &HistoricalFanoutWaitMenu{}
}

func (h *HistoricalFanoutWaitMenu) Handle(user *objects.User, context *context.Context, message *tgbotapi.Message) {
	log.Printf("[WAIT_DEBUG] HistoricalFanoutWait.Handle() START: user.MenuId = %d", user.MenuId)
	log.Printf("[WAIT_DEBUG] Message details: Text='%s', Contact=%v, Location=%v",
		message.Text, message.Contact != nil, message.Location != nil)

	// Show continuation message with button
	log.Printf("[WAIT_DEBUG] About to show continuation message")
	h.showContinuationMessage(user, context)
	log.Printf("[WAIT_DEBUG] Continuation message sent")

	// Проверяем состояние после отправки сообщения
	userAfter := context.Repo.FindUser(user.UserId)
	log.Printf("[WAIT_DEBUG] User state after showContinuationMessage: MenuId = %d", userAfter.MenuId)

	log.Printf("[WAIT_DEBUG] HistoricalFanoutWait.Handle() END")
}

func (h *HistoricalFanoutWaitMenu) HandleCallback(user *objects.User, context *context.Context, callback *tgbotapi.CallbackQuery) {
	log.Printf("[HISTORICAL_FANOUT_WAIT] Handling callback from user %d: %s", user.UserId, callback.Data)

	if callback.Data == "historical_fanout:continue" {
		// Answer the callback
		callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
		context.AnswerCallbackQuery(callbackAnswer)

		// Transition to main menu
		h.transitionToMain(user, context)
		return
	}

	// Handle contact:X callbacks - forward to contact request handler
	if strings.HasPrefix(callback.Data, "contact:") {
		log.Printf("[HISTORICAL_FANOUT_WAIT] Forwarding contact callback to contact request handler")

		// Transition to main menu first
		h.transitionToMain(user, context)

		// Forward callback to contact request handler
		HandleContactRequestCallback(context, callback, user)
		return
	}

	log.Printf("[HISTORICAL_FANOUT_WAIT] Unknown callback data: %s", callback.Data)
}

func (h *HistoricalFanoutWaitMenu) showContinuationMessage(user *objects.User, context *context.Context) {
	log.Printf("[HISTORICAL_FANOUT_WAIT] Showing continuation message to user %d", user.UserId)

	locale := user.Locale()
	messageText := locale.Get("historical_fanout.message")

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				locale.Get("historical_fanout.button_continue"),
				"historical_fanout:continue",
			),
		),
	)

	msg := tgbotapi.NewMessage(user.UserId, messageText)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "HTML"

	// Send with priority 70 to ensure it comes after historical fanout messages (priority 80)
	context.SendWithPriority(msg, 70)
}

func (h *HistoricalFanoutWaitMenu) transitionToMain(user *objects.User, context *context.Context) {
	log.Printf("[HISTORICAL_FANOUT_WAIT] Transitioning user %d to main menu", user.UserId)

	// Update user state to main menu
	oldMenuId := user.MenuId
	user.MenuId = objects.Menu_Main
	if err := context.Repo.SaveUser(user); err != nil {
		log.Printf("[HISTORICAL_FANOUT_WAIT] Error updating user state for user %d: %v", user.UserId, err)
		return
	}

	// Record menu transition metric
	metrics.RecordMenuTransition(oldMenuId, user.MenuId, user.GetSupportedLanguageCode())

	// Continue menu processing after state change
	log.Printf("[HISTORICAL_FANOUT_WAIT] State changed to Main menu, continuing menu processing")
	ContinueMenuProcessing(context, user.UserId)
}

// TransitionToHistoricalFanoutWait transitions user to historical fanout wait menu
func TransitionToHistoricalFanoutWait(context *context.Context, user *objects.User) {
	log.Printf("[TRANSITION_DEBUG] TransitionToHistoricalFanoutWait START: user.MenuId = %d", user.MenuId)

	// Update user state
	oldMenuId := user.MenuId
	user.MenuId = objects.Menu_HistoricalFanoutWait
	log.Printf("[TRANSITION_DEBUG] Changed user.MenuId to %d", user.MenuId)

	if err := context.Repo.SaveUser(user); err != nil {
		log.Printf("[TRANSITION_DEBUG] Error saving user: %v", err)
		return
	}
	log.Printf("[TRANSITION_DEBUG] User saved successfully")

	// Record menu transition metric
	metrics.RecordMenuTransition(oldMenuId, user.MenuId, user.GetSupportedLanguageCode())

	// Проверяем состояние в БД
	userFromDB := context.Repo.FindUser(user.UserId)
	log.Printf("[TRANSITION_DEBUG] User from DB: MenuId = %d", userFromDB.MenuId)

	// НЕ вызываем handler.Handle() - пусть menu loop сам вызовет handler для нового состояния
	log.Printf("[TRANSITION_DEBUG] State changed to %d, menu loop will handle it", user.MenuId)

	log.Printf("[TRANSITION_DEBUG] TransitionToHistoricalFanoutWait END")
}
