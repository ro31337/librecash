package menu

import (
	"librecash/context"
	"librecash/metrics"
	"librecash/objects"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type MainMenuHandler struct {
	user    *objects.User
	context *context.Context
}

func NewMainMenuHandler(c *context.Context, u *objects.User) *MainMenuHandler {
	return &MainMenuHandler{
		context: c,
		user:    u,
	}
}

func (handler *MainMenuHandler) Handle() {
	log.Printf("[MAIN_MENU] Showing main menu to user %d", handler.user.UserId)

	// Create inline keyboard with exchange options
	cashToCryptoBtn := tgbotapi.NewInlineKeyboardButtonData(
		handler.user.Locale().Get("main_menu.cash_to_crypto"),
		"main:cash_to_crypto",
	)
	cryptoToCashBtn := tgbotapi.NewInlineKeyboardButtonData(
		handler.user.Locale().Get("main_menu.crypto_to_cash"),
		"main:crypto_to_cash",
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(cashToCryptoBtn),
		tgbotapi.NewInlineKeyboardRow(cryptoToCashBtn),
	)

	msg := tgbotapi.NewMessage(handler.user.UserId, handler.user.Locale().Get("main_menu.message"))
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "HTML"

	handler.context.Send(msg)
}

// HandleCallback processes inline button callbacks for main menu
func HandleMainMenuCallback(c *context.Context, callback *tgbotapi.CallbackQuery, user *objects.User) {
	log.Printf("[MAIN_MENU] Processing callback: %s for user %d", callback.Data, user.UserId)

	// Parse callback data
	parts := strings.Split(callback.Data, ":")
	if len(parts) != 2 || parts[0] != "main" {
		log.Printf("[MAIN_MENU] Invalid callback data: %s", callback.Data)
		// Answer callback even for invalid data to remove loading animation
		callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
		c.AnswerCallbackQuery(callbackAnswer)
		return
	}

	var direction string
	var confirmationKey string

	switch parts[1] {
	case "cash_to_crypto":
		direction = objects.ExchangeDirectionCashToCrypto
		confirmationKey = "main_menu.confirmed_cash_to_crypto"
	case "crypto_to_cash":
		direction = objects.ExchangeDirectionCryptoToCash
		confirmationKey = "main_menu.confirmed_crypto_to_cash"
	default:
		log.Printf("[MAIN_MENU] Unknown exchange direction: %s", parts[1])
		// Answer callback even for unknown direction to remove loading animation
		callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
		c.AnswerCallbackQuery(callbackAnswer)
		return
	}

	// Create exchange history record
	exchange := objects.NewExchange(user.UserId, direction, user.Lat, user.Lon)
	if err := c.Repo.CreateExchange(exchange); err != nil {
		log.Printf("[MAIN_MENU] Error creating exchange record: %v", err)
		// Still continue to show confirmation
	} else {
		log.Printf("[MAIN_MENU] Created exchange record ID: %d", exchange.ID)
	}

	// Edit the message to show confirmation and remove keyboard
	var confirmationText string
	if confirmationKey == "main_menu.confirmed_cash_to_crypto" {
		confirmationText = user.Locale().Get("main_menu.confirmed_cash_to_crypto")
	} else {
		confirmationText = user.Locale().Get("main_menu.confirmed_crypto_to_cash")
	}
	editMsg := tgbotapi.NewEditMessageText(
		callback.Message.Chat.ID,
		callback.Message.MessageID,
		confirmationText,
	)
	editMsg.ParseMode = "HTML"
	c.EditMessage(editMsg)

	// Answer the callback to stop the loading animation
	callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
	if err := c.AnswerCallbackQuery(callbackAnswer); err != nil {
		log.Printf("[MAIN_MENU] Error answering callback: %v", err)
	}

	// Update user state to amount menu
	oldMenuId := user.MenuId
	user.MenuId = objects.Menu_Amount
	if err := c.Repo.SaveUser(user); err != nil {
		log.Printf("[MAIN_MENU] Error updating user state: %v", err)
	}

	// Record menu transition metric
	metrics.RecordMenuTransition(oldMenuId, user.MenuId, user.GetSupportedLanguageCode())

	// Transition to amount menu
	amountHandler := NewAmountMenuHandler(c, user)
	amountHandler.Handle()
}

// Helper function to show main menu after radius selection
func TransitionToMainMenu(c *context.Context, user *objects.User) {
	log.Printf("[MAIN_MENU] Transitioning user %d to main menu", user.UserId)

	// Update user state
	oldMenuId := user.MenuId
	user.MenuId = objects.Menu_Main
	if err := c.Repo.SaveUser(user); err != nil {
		log.Printf("[MAIN_MENU] Error updating user state: %v", err)
	}

	// Record menu transition metric
	metrics.RecordMenuTransition(oldMenuId, user.MenuId, user.GetSupportedLanguageCode())

	// НЕ вызываем mainHandler.Handle() - пусть menu loop сам вызовет handler для нового состояния
	log.Printf("[MAIN_MENU] State changed to Main menu, menu loop will handle it")
}
