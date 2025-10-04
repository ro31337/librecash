package messaging

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// NewHTMLMessage creates a new message with HTML parsing mode
// This ensures consistent HTML usage across the entire codebase
func NewHTMLMessage(chatID int64, text string) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML" // Always HTML to avoid special character issues
	return msg
}

// NewHTMLEditMessage creates a new edit message with HTML parsing mode
func NewHTMLEditMessage(chatID int64, messageID int, text string) tgbotapi.EditMessageTextConfig {
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ParseMode = "HTML" // Always HTML to avoid special character issues
	return editMsg
}
