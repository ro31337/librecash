package menu

import (
	"fmt"
	"librecash/context"
	"librecash/fanout"
	"librecash/metrics"
	"librecash/objects"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type AmountMenuHandler struct {
	user    *objects.User
	context *context.Context
}

func NewAmountMenuHandler(c *context.Context, u *objects.User) *AmountMenuHandler {
	return &AmountMenuHandler{
		context: c,
		user:    u,
	}
}

func (handler *AmountMenuHandler) Handle() {
	log.Printf("[AMOUNT_MENU] Showing amount menu to user %d", handler.user.UserId)

	// Create inline keyboard with amount options
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				handler.user.Locale().Get("amount_menu.button_5"),
				"amount:5",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				handler.user.Locale().Get("amount_menu.button_10"),
				"amount:10",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				handler.user.Locale().Get("amount_menu.button_15"),
				"amount:15",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				handler.user.Locale().Get("amount_menu.button_25"),
				"amount:25",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				handler.user.Locale().Get("amount_menu.button_50"),
				"amount:50",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				handler.user.Locale().Get("amount_menu.button_75"),
				"amount:75",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				handler.user.Locale().Get("amount_menu.button_100"),
				"amount:100",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				handler.user.Locale().Get("amount_menu.button_cancel"),
				"amount:cancel",
			),
		),
	)

	// Get user's radius for the message (default to 5 if not set)
	radius := 5
	if handler.user.SearchRadiusKm != nil {
		radius = *handler.user.SearchRadiusKm
	}
	messageText := fmt.Sprintf(
		handler.user.Locale().Get("amount_menu.message"),
		radius,
	)

	msg := tgbotapi.NewMessage(handler.user.UserId, messageText)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "HTML"

	handler.context.Send(msg)
}

// HandleCallback processes inline button callbacks for amount menu
func HandleAmountMenuCallback(c *context.Context, callback *tgbotapi.CallbackQuery, user *objects.User) {
	log.Printf("[AMOUNT_MENU] Processing callback: %s for user %d", callback.Data, user.UserId)

	// Parse callback data
	parts := strings.Split(callback.Data, ":")
	if len(parts) != 2 || parts[0] != "amount" {
		log.Printf("[AMOUNT_MENU] Invalid callback data: %s", callback.Data)
		// Answer callback even for invalid data to remove loading animation
		callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
		c.AnswerCallbackQuery(callbackAnswer)
		return
	}

	// Get the last exchange record for this user
	lastExchange, err := c.Repo.GetLastUserExchange(user.UserId)
	if err != nil {
		log.Printf("[AMOUNT_MENU] Error getting last exchange: %v", err)
		// Answer callback to remove loading animation
		callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
		c.AnswerCallbackQuery(callbackAnswer)
		return
	}
	if lastExchange == nil {
		log.Printf("[AMOUNT_MENU] No exchange record found for user %d", user.UserId)
		// Answer callback to remove loading animation
		callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
		c.AnswerCallbackQuery(callbackAnswer)
		return
	}

	var confirmationText string

	if parts[1] == "cancel" {
		// User canceled
		lastExchange.Status = objects.ExchangeStatusCanceled
		if err := c.Repo.UpdateExchange(lastExchange); err != nil {
			log.Printf("[AMOUNT_MENU] Error updating exchange to canceled: %v", err)
		}

		// Record listing cancellation metric
		metrics.RecordListing("canceled", lastExchange.ExchangeDirection, "0", user.GetSupportedLanguageCode())

		confirmationText = user.Locale().Get("amount_menu.canceled")
		log.Printf("[AMOUNT_MENU] User %d canceled amount selection", user.UserId)
	} else {
		// User selected an amount
		amount, err := strconv.Atoi(parts[1])
		if err != nil {
			log.Printf("[AMOUNT_MENU] Invalid amount: %s", parts[1])
			// Answer callback to remove loading animation
			callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
			c.AnswerCallbackQuery(callbackAnswer)
			return
		}

		// Update exchange record with amount and status
		lastExchange.AmountUSD = &amount
		lastExchange.Status = objects.ExchangeStatusPosted
		if err := c.Repo.UpdateExchange(lastExchange); err != nil {
			log.Printf("[AMOUNT_MENU] Error updating exchange with amount: %v", err)
		}

		// Record listing creation metric with USD amount
		amountStr := strconv.Itoa(amount)
		metrics.RecordListing("created", lastExchange.ExchangeDirection, amountStr, user.GetSupportedLanguageCode())

		confirmationText = fmt.Sprintf(
			user.Locale().Get("amount_menu.amount_selected"),
			amount,
		)
		log.Printf("[AMOUNT_MENU] User %d selected amount: $%d", user.UserId, amount)

		// Trigger fanout in background after showing confirmation to user
		go func() {
			fanoutService := fanout.NewFanoutService(c)
			if err := fanoutService.BroadcastExchange(lastExchange); err != nil {
				log.Printf("[AMOUNT_MENU] Fanout failed for exchange %d: %v", lastExchange.ID, err)
				// Fanout failure is non-critical, user already sees success
			} else {
				log.Printf("[AMOUNT_MENU] Fanout completed successfully for exchange %d", lastExchange.ID)
			}
		}()
	}

	// Edit the message to show confirmation and remove keyboard
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
		log.Printf("[AMOUNT_MENU] Error answering callback: %v", err)
	}

	// Update user state to main menu and show it
	oldMenuId := user.MenuId
	user.MenuId = objects.Menu_Main
	if err := c.Repo.SaveUser(user); err != nil {
		log.Printf("[AMOUNT_MENU] Error updating user state: %v", err)
	}

	// Record menu transition metric
	metrics.RecordMenuTransition(oldMenuId, user.MenuId, user.GetSupportedLanguageCode())

	// Show main menu directly
	log.Printf("[AMOUNT_MENU] Showing main menu after exchange posting")
	mainHandler := NewMainMenuHandler(c, user)
	mainHandler.Handle()
}
