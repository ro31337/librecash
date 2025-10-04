package menu

import (
	"librecash/context"
	"librecash/objects"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type BlockedMenu struct{}

func NewBlockedMenu() *BlockedMenu {
	return &BlockedMenu{}
}

func (m *BlockedMenu) Handle(user *objects.User, context *context.Context, message *tgbotapi.Message) {
	log.Printf("[BLOCKED] Showing blocked message for user %d", user.UserId)

	locale := user.Locale()

	// Send blocked message
	msg := tgbotapi.NewMessage(user.UserId, locale.Get("blocked.message"))
	msg.ParseMode = "Markdown"

	context.Send(msg)

	log.Printf("[BLOCKED] User %d remains in blocked state", user.UserId)
}

// BlockedMenu doesn't need HandleCallback since it has no inline buttons
// Users can only restart with /start command
