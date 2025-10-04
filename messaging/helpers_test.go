package messaging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHTMLMessage(t *testing.T) {
	chatID := int64(123456)
	text := "Hello <b>world</b>!"

	msg := NewHTMLMessage(chatID, text)

	assert.Equal(t, chatID, msg.ChatID)
	assert.Equal(t, text, msg.Text)
	assert.Equal(t, "HTML", msg.ParseMode)
}

func TestNewHTMLEditMessage(t *testing.T) {
	chatID := int64(123456)
	messageID := 789
	text := "Updated <i>text</i>"

	editMsg := NewHTMLEditMessage(chatID, messageID, text)

	assert.Equal(t, chatID, editMsg.ChatID)
	assert.Equal(t, messageID, editMsg.MessageID)
	assert.Equal(t, text, editMsg.Text)
	assert.Equal(t, "HTML", editMsg.ParseMode)
}

func TestHTMLMessageWithSpecialCharacters(t *testing.T) {
	// Test that HTML messages work with characters that break Markdown
	problematicTexts := []string{
		"User: @Rails_way",               // underscore
		"Message: user*name",             // asterisk
		"Bot: test[bot]",                 // brackets
		"Code: `user`",                   // backticks
		"Strike: ~through~",              // tilde
		"HTML: <script>alert()</script>", // HTML tags
	}

	for _, text := range problematicTexts {
		t.Run("Text: "+text, func(t *testing.T) {
			msg := NewHTMLMessage(123, text)
			assert.Equal(t, "HTML", msg.ParseMode)
			assert.Equal(t, text, msg.Text)
		})
	}
}
