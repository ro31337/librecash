package menu

import (
	"fmt"
	"librecash/context"
	"librecash/metrics"
	"librecash/objects"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// SelectRadiusMenu handles the search radius selection menu
type SelectRadiusMenu struct{}

func NewSelectRadiusMenu() *SelectRadiusMenu {
	return &SelectRadiusMenu{}
}

func (menu *SelectRadiusMenu) Handle(user *objects.User, context *context.Context, message *tgbotapi.Message) {
	log.Printf("[SELECT_RADIUS_MENU] Handling message from user %d", user.UserId)

	// Send the radius selection message with inline keyboard
	msgText := user.Locale().Get("select_radius_menu.message")

	// Create inline keyboard with three options
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				user.Locale().Get("select_radius_menu.big_city"),
				"radius_5",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				user.Locale().Get("select_radius_menu.suburbs"),
				"radius_15",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				user.Locale().Get("select_radius_menu.rural"),
				"radius_50",
			),
		),
	)

	msg := tgbotapi.NewMessage(user.UserId, msgText)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "Markdown"

	context.Send(msg)
}

// HandleCallback processes the inline button selection
func (menu *SelectRadiusMenu) HandleCallback(user *objects.User, context *context.Context, callback *tgbotapi.CallbackQuery) {
	log.Printf("[SELECT_RADIUS_MENU] Handling callback from user %d: %s", user.UserId, callback.Data)

	var radius int
	switch callback.Data {
	case "radius_5":
		radius = 5
	case "radius_15":
		radius = 15
	case "radius_50":
		radius = 50
	default:
		log.Printf("[SELECT_RADIUS_MENU] Unknown callback data: %s", callback.Data)
		return
	}

	// Update user's search radius in database
	if err := context.Repo.UpdateUserSearchRadius(user.UserId, radius); err != nil {
		log.Printf("[SELECT_RADIUS_MENU] Error updating search radius: %v", err)
		return
	}

	// Answer the callback to remove loading animation
	callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
	if err := context.AnswerCallbackQuery(callbackAnswer); err != nil {
		log.Printf("[SELECT_RADIUS_MENU] Error answering callback: %v", err)
	}

	// Edit the message to remove inline keyboard and show confirmation
	confirmText := fmt.Sprintf(user.Locale().Get("select_radius_menu.radius_confirmed"), radius)
	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, confirmText)

	context.EditMessage(editMsg)

	// Save radius preference
	user.SearchRadiusKm = &radius
	if err := context.Repo.SaveUser(user); err != nil {
		log.Printf("[SELECT_RADIUS_MENU] Error saving radius preference: %v", err)
		return
	}

	// Create location history record (PRD012)
	if err := context.Repo.CreateLocationHistory(user.UserId, radius); err != nil {
		log.Printf("[SELECT_RADIUS_MENU] Error creating location history: %v", err)
		// Continue anyway - this is not critical for user flow
	}

	// Transition to location menu (new order: radius first, then location)
	oldMenuId := user.MenuId
	user.MenuId = objects.Menu_AskLocation
	if err := context.Repo.SaveUser(user); err != nil {
		log.Printf("[SELECT_RADIUS_MENU] Error updating user state: %v", err)
		return
	}

	// Record menu transition metric
	metrics.RecordMenuTransition(oldMenuId, user.MenuId, user.GetSupportedLanguageCode())

	log.Printf("[MENU] User %d transitioned to Menu_AskLocation after radius selection", user.UserId)

	// Immediately show the location menu
	locationHandler := NewAskLocationMenu()
	locationHandler.Handle(user, context, &tgbotapi.Message{})
}
