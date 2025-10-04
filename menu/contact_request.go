package menu

import (
	"fmt"
	"librecash/context"
	"librecash/metrics"
	"librecash/objects"
	"librecash/rabbit"
	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/leonelquinteros/gotext"
)

// HandleContactRequestCallback processes "Show contact" button clicks
func HandleContactRequestCallback(c *context.Context, callback *tgbotapi.CallbackQuery, user *objects.User) {
	log.Printf("[CONTACT_REQUEST] Processing callback: %s for user %d", callback.Data, user.UserId)

	// Parse callback data
	parts := strings.Split(callback.Data, ":")
	if len(parts) != 2 || parts[0] != "contact" {
		log.Printf("[CONTACT_REQUEST] Invalid callback data: %s", callback.Data)
		// Answer callback even for invalid data to remove loading animation
		callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
		c.AnswerCallbackQuery(callbackAnswer)
		return
	}

	// Parse exchange ID
	exchangeID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		log.Printf("[CONTACT_REQUEST] Invalid exchange ID: %s", parts[1])
		// Answer callback even for invalid exchange ID to remove loading animation
		callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
		c.AnswerCallbackQuery(callbackAnswer)
		return
	}

	log.Printf("[CONTACT_REQUEST] User %d requesting contact for exchange %d", user.UserId, exchangeID)

	// Get exchange details
	exchange, err := c.Repo.GetExchangeByID(exchangeID)
	if err != nil {
		log.Printf("[CONTACT_REQUEST] Error getting exchange %d: %v", exchangeID, err)
		// Answer callback to remove loading animation
		callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
		c.AnswerCallbackQuery(callbackAnswer)
		return
	}
	if exchange == nil {
		log.Printf("[CONTACT_REQUEST] Exchange %d not found", exchangeID)
		// Answer callback to remove loading animation
		callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
		c.AnswerCallbackQuery(callbackAnswer)
		return
	}

	// Get initiator details
	initiator := c.Repo.FindUser(exchange.UserID)
	if initiator == nil {
		log.Printf("[CONTACT_REQUEST] Initiator user %d not found", exchange.UserID)
		// Answer callback to remove loading animation
		callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
		c.AnswerCallbackQuery(callbackAnswer)
		return
	}

	// Check for duplicate request first
	exists, err := c.Repo.CheckContactRequestExists(exchangeID, user.UserId)
	if err != nil {
		log.Printf("[CONTACT_REQUEST] Error checking contact request existence: %v", err)
		// Answer callback to remove loading animation
		callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
		c.AnswerCallbackQuery(callbackAnswer)
		return
	}

	// Determine user type for metrics
	userType := "returning"
	if exists {
		log.Printf("[CONTACT_REQUEST] Contact request already exists, showing existing contact info")
		userType = "existing_contact"
	} else {
		// Create new contact request
		err = c.Repo.CreateContactRequest(exchangeID, user.UserId, user.Username, user.FirstName, user.LastName)
		if err != nil {
			log.Printf("[CONTACT_REQUEST] Error creating contact request: %v", err)
			// Answer callback to remove loading animation
			callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
			c.AnswerCallbackQuery(callbackAnswer)
			return
		}
		log.Printf("[CONTACT_REQUEST] Created new contact request")
		userType = "new_contact"
	}

	// Record contact request metric
	metrics.RecordContactRequest(exchange.ExchangeDirection, user.GetSupportedLanguageCode(), userType)

	log.Printf("[CONTACT_REQUEST] Processing contact request")

	// Answer the callback to remove loading animation
	callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
	if err := c.AnswerCallbackQuery(callbackAnswer); err != nil {
		log.Printf("[CONTACT_REQUEST] Error answering callback: %v", err)
	}

	// 1. Edit requester's message to show contact info
	if err := editRequesterMessage(c, callback, user, initiator); err != nil {
		log.Printf("[CONTACT_REQUEST] Error editing requester message: %v", err)
		// Continue processing even if edit fails
	}

	// 2. Send notification to initiator (always, even for existing contacts)
	if err := sendInitiatorNotification(c, exchange, initiator, user); err != nil {
		log.Printf("[CONTACT_REQUEST] Error sending initiator notification: %v", err)
		// Continue processing even if notification fails
	}

	log.Printf("[CONTACT_REQUEST] Contact request processed successfully")
}

// editRequesterMessage edits the fanout message to show contact information
func editRequesterMessage(c *context.Context, callback *tgbotapi.CallbackQuery, requester *objects.User, initiator *objects.User) error {
	log.Printf("[CONTACT_REQUEST] Editing message for requester %d", requester.UserId)

	// Get current message text
	currentText := callback.Message.Text

	// Format contact info (include phone for contact requests)
	// Use requester's language for the contact info display
	contactInfo := formatUserIdentifier(initiator, true, requester.GetSupportedLanguageCode())
	log.Printf("[CONTACT_REQUEST] Contact info formatted: %s", contactInfo)
	contactLine := fmt.Sprintf("\n\n%s", requester.Locale().Get("contact_request.contact_info"))
	contactText := fmt.Sprintf(contactLine, contactInfo)
	log.Printf("[CONTACT_REQUEST] Contact text: %s", contactText)

	// Append contact to existing message
	newText := currentText + contactText
	log.Printf("[CONTACT_REQUEST] New message text: %s", newText)

	// Edit message with contact info and remove keyboard
	editMsg := tgbotapi.NewEditMessageText(
		callback.Message.Chat.ID,
		callback.Message.MessageID,
		newText,
	)
	editMsg.ParseMode = "HTML"
	// Remove the inline keyboard by setting it to nil
	editMsg.ReplyMarkup = nil

	// Note: We need to use EditMessageText, but RabbitMQ doesn't handle edits directly
	// For now, send edit directly (this is an exception to RabbitMQ rule for edits)
	err := c.EditMessage(editMsg)
	if err != nil {
		log.Printf("[CONTACT_REQUEST] Error editing message: %v", err)
		return err
	}

	log.Printf("[CONTACT_REQUEST] Successfully edited requester message")
	return nil
}

// sendInitiatorNotification sends notification to exchange initiator
func sendInitiatorNotification(c *context.Context, exchange *objects.Exchange, initiator *objects.User, requester *objects.User) error {
	log.Printf("[CONTACT_REQUEST] Sending notification to initiator %d", initiator.UserId)

	// Format requester identifier (include phone for notifications)
	// Use initiator's language for the notification display
	requesterInfo := formatUserIdentifier(requester, true, initiator.GetSupportedLanguageCode())

	// Create notification message
	notificationText := fmt.Sprintf(
		initiator.Locale().Get("contact_request.notification"),
		requesterInfo,
	)

	// Create message
	msg := tgbotapi.NewMessage(initiator.UserId, notificationText)
	msg.ParseMode = "HTML"

	// Send via RabbitMQ
	messageBag := rabbit.MessageBag{
		Message:  msg,
		Priority: 100, // Normal priority
	}

	err := c.RabbitPublish.PublishTgMessage(messageBag)
	if err != nil {
		log.Printf("[CONTACT_REQUEST] Error publishing initiator notification: %v", err)
		return err
	}

	log.Printf("[CONTACT_REQUEST] Successfully queued initiator notification")
	return nil
}

// formatUserIdentifier formats user identifier with priority: username > clickable name link
// Universal function that can optionally include phone number (PRD020)
// Updated for PRD021: Uses localized phone label instead of hardcoded "PHONE:"
func formatUserIdentifier(user *objects.User, includePhone bool, recipientLanguage string) string {
	var identifier string

	if user.Username != "" {
		identifier = fmt.Sprintf("@%s", user.Username)
	} else {
		// Create clickable user link using tg://user?id= for users without username
		// Build display name from available parts
		var displayName string
		if user.FirstName != "" && user.LastName != "" {
			displayName = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
		} else if user.FirstName != "" {
			displayName = user.FirstName
		} else if user.LastName != "" {
			displayName = user.LastName
		} else {
			displayName = "Anonymous"
		}

		// Return HTML link that will be clickable in Telegram (PRD020: HTML instead of Markdown)
		identifier = fmt.Sprintf(`<a href="tg://user?id=%d">%s</a>`,
			user.UserId, htmlEscapeString(displayName))
	}

	// Add phone number if requested (PRD021: Use localized phone label)
	if includePhone && user.PhoneNumber != "" {
		phoneLabel := getTranslation(recipientLanguage, "phone_label")
		identifier += "\n" + fmt.Sprintf(phoneLabel, user.PhoneNumber)
	}

	return identifier
}

// getTranslation gets a translation for a specific language code and key
// Helper function for PRD021 localization
func getTranslation(languageCode, key string) string {
	// Load locale file directly without caching to avoid conflicts
	po := gotext.NewPo()

	poFile := fmt.Sprintf("./locales/all/%s.po", languageCode)

	// Check if file exists (handle both app runtime and test runtime paths)
	if _, err := os.Stat(poFile); os.IsNotExist(err) {
		poFile = fmt.Sprintf("../locales/all/%s.po", languageCode)
	}

	po.ParseFile(poFile)

	// Use specific key to avoid linter issues with non-constant format strings
	if key == "phone_label" {
		return po.Get("phone_label")
	}

	// Fallback for other keys (though we only use phone_label for now)
	return key // Return key if not found
}

// htmlEscapeString escapes special characters for HTML
func htmlEscapeString(text string) string {
	// Basic HTML escaping for user names
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	text = strings.ReplaceAll(text, "\"", "&quot;")
	text = strings.ReplaceAll(text, "'", "&#39;")
	return text
}

// HandleDeleteExchangeCallback processes "Delete" button clicks from exchange authors
func HandleDeleteExchangeCallback(c *context.Context, callback *tgbotapi.CallbackQuery, user *objects.User) {
	log.Printf("[DELETE_EXCHANGE] Processing callback: %s for user %d", callback.Data, user.UserId)

	// Parse callback data
	parts := strings.Split(callback.Data, ":")
	if len(parts) != 2 || parts[0] != "delete" {
		log.Printf("[DELETE_EXCHANGE] Invalid callback data: %s", callback.Data)
		// Answer callback even for invalid data to remove loading animation
		callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
		c.AnswerCallbackQuery(callbackAnswer)
		return
	}

	// Parse exchange ID
	exchangeID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		log.Printf("[DELETE_EXCHANGE] Invalid exchange ID: %s", parts[1])
		// Answer callback even for invalid exchange ID to remove loading animation
		callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
		c.AnswerCallbackQuery(callbackAnswer)
		return
	}

	log.Printf("[DELETE_EXCHANGE] User %d requesting deletion of exchange %d", user.UserId, exchangeID)

	// Get exchange details
	exchange, err := c.Repo.GetExchangeByID(exchangeID)
	if err != nil {
		log.Printf("[DELETE_EXCHANGE] Error getting exchange %d: %v", exchangeID, err)
		// Answer callback to remove loading animation
		callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
		c.AnswerCallbackQuery(callbackAnswer)
		return
	}
	if exchange == nil {
		log.Printf("[DELETE_EXCHANGE] Exchange %d not found", exchangeID)
		// Answer callback to remove loading animation
		callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
		c.AnswerCallbackQuery(callbackAnswer)
		return
	}

	// Security check: Only exchange author can delete their exchange
	if user.UserId != exchange.UserID {
		log.Printf("[DELETE_EXCHANGE] Security violation: User %d tried to delete exchange %d owned by user %d",
			user.UserId, exchangeID, exchange.UserID)
		// Answer callback to remove loading animation
		callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
		c.AnswerCallbackQuery(callbackAnswer)
		return
	}

	log.Printf("[DELETE_EXCHANGE] Processing exchange deletion")

	// Answer the callback to remove loading animation
	callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
	if err := c.AnswerCallbackQuery(callbackAnswer); err != nil {
		log.Printf("[DELETE_EXCHANGE] Error answering callback: %v", err)
	}

	// 1. Soft delete the exchange itself
	if err := c.Repo.SoftDeleteExchange(exchangeID); err != nil {
		log.Printf("[DELETE_EXCHANGE] Error soft deleting exchange: %v", err)
		return
	}

	// Record listing deletion metric
	amountStr := "0"
	if exchange.AmountUSD != nil {
		amountStr = strconv.Itoa(*exchange.AmountUSD)
	}
	metrics.RecordListing("deleted", exchange.ExchangeDirection, amountStr, user.GetSupportedLanguageCode())

	// 2. Get all timeline records for this exchange
	timelineRecords, err := c.Repo.GetTimelineRecordsByExchange(exchangeID)
	if err != nil {
		log.Printf("[DELETE_EXCHANGE] Error getting timeline records: %v", err)
		return
	}

	// 3. Soft delete all timeline records
	if err := c.Repo.SoftDeleteExchangeTimeline(exchangeID); err != nil {
		log.Printf("[DELETE_EXCHANGE] Error soft deleting timeline records: %v", err)
		return
	}

	// 4. Edit author's message to show deletion confirmation
	if err := editAuthorMessage(c, user, timelineRecords, exchangeID); err != nil {
		log.Printf("[DELETE_EXCHANGE] Error editing author message: %v", err)
		// Continue processing even if edit fails
	}

	// 5. Edit all recipient messages to show deletion notification
	if err := editRecipientMessages(c, exchange, timelineRecords); err != nil {
		log.Printf("[DELETE_EXCHANGE] Error editing recipient messages: %v", err)
		// Continue processing even if edits fail
	}

	log.Printf("[DELETE_EXCHANGE] Exchange deletion processed successfully")
}

// editAuthorMessage edits the author's original message to show deletion confirmation
func editAuthorMessage(c *context.Context, author *objects.User, timelineRecords []*objects.TimelineRecord, exchangeID int64) error {
	log.Printf("[DELETE_EXCHANGE] Editing author message for user %d", author.UserId)

	// Find author's timeline record
	var authorRecord *objects.TimelineRecord
	for _, record := range timelineRecords {
		if record.RecipientUserID == author.UserId {
			authorRecord = record
			break
		}
	}

	if authorRecord == nil || authorRecord.TelegramMessageID == nil {
		log.Printf("[DELETE_EXCHANGE] Author's timeline record or message ID not found")
		return fmt.Errorf("author's message not found")
	}

	// Create confirmation text
	confirmationText := author.Locale().Get("delete_exchange.deleted_by_you")

	// Edit the message
	editMsg := tgbotapi.NewEditMessageText(
		author.UserId,
		*authorRecord.TelegramMessageID,
		confirmationText,
	)
	editMsg.ParseMode = "HTML"

	// Send edit via RabbitMQ
	if err := c.EditMessage(editMsg); err != nil {
		log.Printf("[DELETE_EXCHANGE] Error editing author message: %v", err)
		return err
	}

	log.Printf("[DELETE_EXCHANGE] Author message edited successfully")
	return nil
}

// editRecipientMessages edits all recipient messages to show deletion notification
func editRecipientMessages(c *context.Context, exchange *objects.Exchange, timelineRecords []*objects.TimelineRecord) error {
	log.Printf("[DELETE_EXCHANGE] Editing recipient messages")

	for _, record := range timelineRecords {
		// Skip the author (they already got their message edited)
		if record.RecipientUserID == exchange.UserID {
			continue
		}

		// Skip if no telegram message ID
		if record.TelegramMessageID == nil {
			log.Printf("[DELETE_EXCHANGE] No telegram message ID for recipient %d, skipping", record.RecipientUserID)
			continue
		}

		// Get recipient user
		recipient := c.Repo.FindUser(record.RecipientUserID)
		if recipient == nil {
			log.Printf("[DELETE_EXCHANGE] Recipient user %d not found, skipping", record.RecipientUserID)
			continue
		}

		// Create notification text
		notificationText := recipient.Locale().Get("delete_exchange.deleted_by_author")

		// Edit the message
		editMsg := tgbotapi.NewEditMessageText(
			recipient.UserId,
			*record.TelegramMessageID,
			notificationText,
		)
		editMsg.ParseMode = "HTML"

		// Send edit via RabbitMQ
		if err := c.EditMessage(editMsg); err != nil {
			log.Printf("[DELETE_EXCHANGE] Error editing message for recipient %d: %v", record.RecipientUserID, err)
			// Continue with other recipients even if one fails
			continue
		}

		log.Printf("[DELETE_EXCHANGE] Message edited for recipient %d", recipient.UserId)
	}

	log.Printf("[DELETE_EXCHANGE] All recipient messages edited")
	return nil
}
