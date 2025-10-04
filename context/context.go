package context

import (
	"librecash/config"
	"librecash/rabbit"
	"librecash/repository"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Context struct {
	bot           *tgbotapi.BotAPI // private - only accessible through methods
	Repo          *repository.Repository
	RabbitPublish *rabbit.RabbitClient // for publishing only
	RabbitConsume *rabbit.RabbitClient // for consuming only
	Config        *config.Config
}

// Send is a drop-in replacement for telegram Send method, posts with high priority
func (context *Context) Send(message tgbotapi.MessageConfig) {
	log.Printf("[CONTEXT] Sending message to user %d via RabbitMQ with high priority", message.ChatID)

	context.RabbitPublish.PublishTgMessage(rabbit.MessageBag{
		Message:  message,
		Priority: 220, // high priority for user messages, but lower than callbacks
	})
}

// SendWithPriority sends a message with specified priority through RabbitMQ
func (context *Context) SendWithPriority(message tgbotapi.MessageConfig, priority uint8) {
	log.Printf("[CONTEXT] Sending message to user %d via RabbitMQ with priority %d", message.ChatID, priority)

	context.RabbitPublish.PublishTgMessage(rabbit.MessageBag{
		Message:  message,
		Priority: priority,
	})
}

// AnswerCallbackQuery answers a callback query through RabbitMQ with highest priority
func (context *Context) AnswerCallbackQuery(callback tgbotapi.CallbackConfig) error {
	log.Printf("[CONTEXT] Sending callback answer %s via RabbitMQ with highest priority", callback.CallbackQueryID)

	return context.RabbitPublish.PublishCallbackAnswer(rabbit.CallbackAnswerBag{
		CallbackAnswer: callback,
		Priority:       255, // Highest priority for instant response
	})
}

// GetBot returns the bot instance - ONLY for sender package use
// This method should ONLY be used by the sender package for actual message sending
func (context *Context) GetBot() *tgbotapi.BotAPI {
	return context.bot
}

// SetBot sets the bot instance - used during initialization
func (context *Context) SetBot(bot *tgbotapi.BotAPI) {
	context.bot = bot
}

// EditMessage edits a message through RabbitMQ
func (context *Context) EditMessage(editMsg tgbotapi.EditMessageTextConfig) error {
	log.Printf("[CONTEXT] Sending message edit for message %d in chat %d via RabbitMQ", editMsg.MessageID, editMsg.ChatID)

	return context.RabbitPublish.PublishEditMessage(rabbit.EditMessageBag{
		EditMessage: editMsg,
		Priority:    200, // High priority for edits
	})
}
