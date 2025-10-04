package menu

import (
	"librecash/context"
	"librecash/metrics"
	"librecash/objects"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type AskLocationMenuHandler struct {
	user    *objects.User
	context *context.Context
	message *tgbotapi.Message
}

func (handler *AskLocationMenuHandler) saveLocation() {
	handler.user.Lon = handler.message.Location.Longitude
	handler.user.Lat = handler.message.Location.Latitude
	log.Printf("[LOCATION] Saving location for user %d: lon=%f, lat=%f",
		handler.user.UserId, handler.user.Lon, handler.user.Lat)
	handler.context.Repo.SaveUser(handler.user)

	// Record geographic data metric
	metrics.RecordUserLocation(handler.user.Lat, handler.user.Lon, handler.user.GetSupportedLanguageCode())

	// Update geography column in database for PostGIS queries
	_, err := handler.context.Repo.UpdateUserLocation(handler.user.UserId, handler.user.Lon, handler.user.Lat)
	if err != nil {
		log.Printf("[LOCATION] Error updating user location: %v", err)
		return
	}

	// Update location history (PRD012)
	if err := handler.context.Repo.UpdateLocationHistory(handler.user.UserId, handler.user.Lat, handler.user.Lon); err != nil {
		log.Printf("[LOCATION] Error updating location history: %v", err)
		// Continue anyway - this is not critical for user flow
	}
}

func (handler *AskLocationMenuHandler) Handle(user *objects.User, context *context.Context, message *tgbotapi.Message) {
	log.Printf("[MENU] Ask location menu for user %d", user.UserId)

	handler.user = user
	handler.context = context
	handler.message = message

	// Check if we received a location
	if message.Location != nil {
		log.Printf("[LOCATION] Received location from user %d: %+v", user.UserId, message.Location)
		handler.saveLocation()

		// Transition to phone menu (PRD012: radius → location → phone → historical_fanout)
		oldMenuId := user.MenuId
		user.MenuId = objects.Menu_AskPhone
		context.Repo.SaveUser(user)

		// Record menu transition metric
		metrics.RecordMenuTransition(oldMenuId, user.MenuId, user.GetSupportedLanguageCode())

		// Remove the location keyboard
		removeKeyboard := tgbotapi.NewMessage(user.UserId, user.Locale().Get("ask_location_menu.location_received"))
		removeKeyboard.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		context.Send(removeKeyboard)

		log.Printf("[MENU] User %d transitioned to Menu_AskPhone after providing location", user.UserId)
		// The menu loop will automatically handle the transition to AskPhone
		return
	}

	// Show location request message with button
	log.Printf("[MENU] Showing location request to user %d", user.UserId)

	// Create location request button
	locationButton := tgbotapi.NewKeyboardButtonLocation(user.Locale().Get("ask_location_menu.next_button"))
	keyboard := tgbotapi.NewReplyKeyboard(
		[]tgbotapi.KeyboardButton{locationButton},
	)
	keyboard.OneTimeKeyboard = true
	keyboard.ResizeKeyboard = true

	msg := tgbotapi.NewMessage(user.UserId, user.Locale().Get("ask_location_menu.message"))
	msg.ReplyMarkup = keyboard

	context.Send(msg)
}

func NewAskLocationMenu() *AskLocationMenuHandler {
	return &AskLocationMenuHandler{}
}
