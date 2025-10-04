package sender

import (
	"encoding/json"
	"librecash/context"
	"librecash/metrics"
	"librecash/objects"
	"librecash/rabbit"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/streadway/amqp"
)

type Sender struct {
	context *context.Context
}

func NewSender(context *context.Context) *Sender {
	log.Println("[SENDER] Creating new message sender")
	return &Sender{
		context: context,
	}
}

func (s *Sender) Handler(data []byte, headers amqp.Table) {
	// Check message type from headers
	if messageType, ok := headers["message_type"]; ok {
		switch messageType {
		case "exchange_notification":
			var messageBag rabbit.MessageBag
			if err := json.Unmarshal(data, &messageBag); err != nil {
				log.Printf("[SENDER] Failed to unmarshal exchange notification: %v", err)
				return
			}
			log.Printf("[SENDER] Processing exchange notification for chat %d with priority %d",
				messageBag.Message.ChatID, messageBag.Priority)
			s.handleExchangeNotification(&messageBag, headers)
			return
		case "callback_answer":
			var callbackBag rabbit.CallbackAnswerBag
			if err := json.Unmarshal(data, &callbackBag); err != nil {
				log.Printf("[SENDER] Failed to unmarshal callback answer: %v", err)
				return
			}
			log.Printf("[SENDER] Processing callback answer %s with priority %d",
				callbackBag.CallbackAnswer.CallbackQueryID, callbackBag.Priority)
			s.handleCallbackAnswer(&callbackBag)
			return
		case "edit_message":
			var editBag rabbit.EditMessageBag
			if err := json.Unmarshal(data, &editBag); err != nil {
				log.Printf("[SENDER] Failed to unmarshal edit message: %v", err)
				return
			}
			log.Printf("[SENDER] Processing message edit for message %d in chat %d with priority %d",
				editBag.EditMessage.MessageID, editBag.EditMessage.ChatID, editBag.Priority)
			s.handleEditMessage(&editBag)
			return
		}
	}

	// Handle regular message
	var messageBag rabbit.MessageBag
	if err := json.Unmarshal(data, &messageBag); err != nil {
		log.Printf("[SENDER] Failed to unmarshal regular message: %v", err)
		return
	}
	log.Printf("[SENDER] Processing regular message for chat %d with priority %d",
		messageBag.Message.ChatID, messageBag.Priority)
	s.handleRegularMessage(&messageBag)
}

func (s *Sender) handleRegularMessage(messageBag *rabbit.MessageBag) {
	log.Printf("[SENDER] Processing regular message for chat %d", messageBag.Message.ChatID)

	startTime := time.Now()

	// Send message via Telegram Bot API
	_, err := s.context.GetBot().Send(messageBag.Message)

	duration := time.Since(startTime)

	if err != nil {
		log.Printf("[SENDER] ERROR sending Telegram message to chat %d: %v (duration: %v)",
			messageBag.Message.ChatID, err, duration)
		// Record failed telegram message metric
		errorCode := "unknown"
		if err != nil {
			errorCode = strconv.Itoa(extractErrorCode(err))
		}
		metrics.RecordTelegramMessage("regular", "failed", errorCode)
	} else {
		log.Printf("[SENDER] Successfully sent message to chat %d (duration: %v)",
			messageBag.Message.ChatID, duration)
		// Record successful telegram message metric
		metrics.RecordTelegramMessage("regular", "sent", "none")
	}
}

func (s *Sender) handleExchangeNotification(messageBag *rabbit.MessageBag, headers amqp.Table) {
	log.Printf("[SENDER] Processing exchange notification for chat %d", messageBag.Message.ChatID)

	// Extract exchange information from headers
	exchangeID, ok := headers["exchange_id"].(int64)
	if !ok {
		log.Printf("[SENDER] ERROR: Invalid exchange_id in headers")
		return
	}

	recipientUserID, ok := headers["recipient_user_id"].(int64)
	if !ok {
		log.Printf("[SENDER] ERROR: Invalid recipient_user_id in headers")
		return
	}

	startTime := time.Now()

	// Send message via Telegram Bot API FIRST
	sentMessage, err := s.context.GetBot().Send(messageBag.Message)

	duration := time.Since(startTime)

	if err != nil {
		log.Printf("[SENDER] ERROR sending exchange notification to chat %d: %v (duration: %v)",
			messageBag.Message.ChatID, err, duration)

		// Record failed telegram message metric
		errorCode := "unknown"
		if err != nil {
			errorCode = strconv.Itoa(extractErrorCode(err))
		}
		metrics.RecordTelegramMessage("exchange_notification", "failed", errorCode)

		// Create timeline record with 'failed' status (no Telegram message ID)
		record := objects.NewTimelineRecord(exchangeID, recipientUserID)
		record.Status = objects.TimelineStatusFailed
		if createErr := s.context.Repo.CreateTimelineRecord(record); createErr != nil {
			log.Printf("[SENDER] ERROR creating failed timeline record: %v", createErr)
		}
	} else {
		log.Printf("[SENDER] Successfully sent exchange notification to chat %d (duration: %v)",
			messageBag.Message.ChatID, duration)

		// Record successful telegram message metric
		metrics.RecordTelegramMessage("exchange_notification", "sent", "none")

		// Create timeline record with Telegram message ID and 'sent' status
		record := objects.NewTimelineRecord(exchangeID, recipientUserID)
		record.TelegramMessageID = &sentMessage.MessageID
		record.Status = objects.TimelineStatusSent
		if createErr := s.context.Repo.CreateTimelineRecord(record); createErr != nil {
			log.Printf("[SENDER] ERROR creating sent timeline record: %v", createErr)
		}
	}
}

func (s *Sender) handleCallbackAnswer(callbackBag *rabbit.CallbackAnswerBag) {
	log.Printf("[SENDER] Processing callback answer %s", callbackBag.CallbackAnswer.CallbackQueryID)

	startTime := time.Now()

	// Send callback answer via Telegram Bot API
	_, err := s.context.GetBot().AnswerCallbackQuery(callbackBag.CallbackAnswer)

	duration := time.Since(startTime)

	if err != nil {
		log.Printf("[SENDER] ERROR answering callback query %s: %v (duration: %v)",
			callbackBag.CallbackAnswer.CallbackQueryID, err, duration)
		// Record failed telegram callback metric
		errorCode := "unknown"
		if err != nil {
			errorCode = strconv.Itoa(extractErrorCode(err))
		}
		metrics.RecordTelegramMessage("callback_answer", "failed", errorCode)
	} else {
		log.Printf("[SENDER] Successfully answered callback query %s (duration: %v)",
			callbackBag.CallbackAnswer.CallbackQueryID, duration)
		// Record successful telegram callback metric
		metrics.RecordTelegramMessage("callback_answer", "sent", "none")
	}
}

func (s *Sender) handleEditMessage(editBag *rabbit.EditMessageBag) {
	log.Printf("[SENDER] Processing message edit for message %d in chat %d",
		editBag.EditMessage.MessageID, editBag.EditMessage.ChatID)

	startTime := time.Now()

	// Send edit message via Telegram Bot API
	_, err := s.context.GetBot().Send(editBag.EditMessage)

	duration := time.Since(startTime)

	if err != nil {
		log.Printf("[SENDER] ERROR editing message %d in chat %d: %v (duration: %v)",
			editBag.EditMessage.MessageID, editBag.EditMessage.ChatID, err, duration)
		// Record failed telegram edit metric
		errorCode := "unknown"
		if err != nil {
			errorCode = strconv.Itoa(extractErrorCode(err))
		}
		metrics.RecordTelegramMessage("edit_message", "failed", errorCode)
	} else {
		log.Printf("[SENDER] Successfully edited message %d in chat %d (duration: %v)",
			editBag.EditMessage.MessageID, editBag.EditMessage.ChatID, duration)
		// Record successful telegram edit metric
		metrics.RecordTelegramMessage("edit_message", "sent", "none")
	}
}

func (s *Sender) Start() {
	log.Println("[SENDER] Starting message sender service")
	log.Println("[SENDER] Registering handler with RabbitMQ consumer")

	// Register the handler with RabbitMQ consumer
	// The rate limiting is handled in the RabbitClient
	s.context.RabbitConsume.RegisterHandler(s.Handler)

	log.Println("[SENDER] Message sender service started successfully")
}

// httpErrorCodeRegex matches HTTP status codes (4xx or 5xx) in error messages
// Uses negative lookbehind/lookahead to avoid matching phone numbers or other contexts
var httpErrorCodeRegex = regexp.MustCompile(`(?:^|\s|:|\(|-)([4-5]\d{2})(?:\s|$|:|!|\)|,)`)

// extractErrorCode extracts HTTP error code from Telegram API error using regex
func extractErrorCode(err error) int {
	if err == nil {
		return 200
	}

	// Use regex to find HTTP error codes (4xx or 5xx) in error message
	errStr := err.Error()
	matches := httpErrorCodeRegex.FindStringSubmatch(errStr)

	if len(matches) >= 2 {
		// Parse the captured HTTP code
		if code, parseErr := strconv.Atoi(matches[1]); parseErr == nil {
			return code
		}
	}

	return 0 // Unknown error - no HTTP code found
}
