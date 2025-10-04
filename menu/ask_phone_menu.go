package menu

import (
	"librecash/context"
	"librecash/objects"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type AskPhoneMenuHandler struct {
	user    *objects.User
	context *context.Context
	message *tgbotapi.Message
}

func (handler *AskPhoneMenuHandler) savePhoneNumber() {
	if handler.message.Contact != nil {
		handler.user.PhoneNumber = handler.message.Contact.PhoneNumber
		log.Printf("[PHONE] Saving phone number for user %d: %s",
			handler.user.UserId, handler.user.PhoneNumber)
		handler.context.Repo.SaveUser(handler.user)
	}
}

func (handler *AskPhoneMenuHandler) Handle(user *objects.User, context *context.Context, message *tgbotapi.Message) {
	log.Printf("[ASKPHONE_DEBUG] AskPhone.Handle() START: user.MenuId = %d", user.MenuId)
	log.Printf("[ASKPHONE_DEBUG] Message details: Text='%s', Contact=%v",
		message.Text, message.Contact != nil)

	handler.user = user
	handler.context = context
	handler.message = message

	// Check if we received a contact (phone number)
	if message.Contact != nil {
		log.Printf("[PHONE] Received contact from user %d: %+v", user.UserId, message.Contact)
		handler.savePhoneNumber()

		// Remove the phone keyboard
		removeKeyboard := tgbotapi.NewMessage(user.UserId, user.Locale().Get("ask_phone_menu.phone_received"))
		removeKeyboard.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		context.Send(removeKeyboard)

		// Transition to historical fanout execute menu (PRD013)
		TransitionToHistoricalFanoutExecute(context, user)
		return
	}

	// Check if user clicked "Don't set" button
	if message.Text == user.Locale().Get("ask_phone_menu.skip_button") {
		log.Printf("[PHONE] User %d chose to skip phone number", user.UserId)

		// Set phone number to empty string (removes existing phone)
		user.PhoneNumber = ""
		if err := context.Repo.SaveUser(user); err != nil {
			log.Printf("[ASK_PHONE_MENU] Error saving user: %v", err)
			return
		}

		// Remove the phone keyboard
		removeKeyboard := tgbotapi.NewMessage(user.UserId, user.Locale().Get("ask_phone_menu.phone_skipped"))
		removeKeyboard.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		context.Send(removeKeyboard)

		// Transition to historical fanout execute menu (PRD013)
		TransitionToHistoricalFanoutExecute(context, user)
		return
	}

	// Show phone request message with buttons
	log.Printf("[ASKPHONE_DEBUG] About to show phone request")

	// Create regular keyboard with two buttons
	phoneButton := tgbotapi.NewKeyboardButtonContact(user.Locale().Get("ask_phone_menu.share_button"))
	skipButton := tgbotapi.NewKeyboardButton(user.Locale().Get("ask_phone_menu.skip_button"))

	keyboard := tgbotapi.NewReplyKeyboard(
		[]tgbotapi.KeyboardButton{phoneButton},
		[]tgbotapi.KeyboardButton{skipButton},
	)
	keyboard.OneTimeKeyboard = true
	keyboard.ResizeKeyboard = true

	// Send message with regular keyboard
	msg := tgbotapi.NewMessage(user.UserId, user.Locale().Get("ask_phone_menu.message"))
	msg.ReplyMarkup = keyboard

	context.Send(msg)
	log.Printf("[ASKPHONE_DEBUG] Phone request sent")

	// Проверяем состояние после отправки
	userAfter := context.Repo.FindUser(user.UserId)
	log.Printf("[ASKPHONE_DEBUG] User state after Send: MenuId = %d", userAfter.MenuId)

	log.Printf("[ASKPHONE_DEBUG] AskPhone.Handle() END")
}

func NewAskPhoneMenu() *AskPhoneMenuHandler {
	return &AskPhoneMenuHandler{}
}
