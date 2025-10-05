package rabbit

import (
	"encoding/json"
	"librecash/metrics"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/streadway/amqp"
	"go.uber.org/ratelimit"
)

type RabbitClient struct {
	url        string
	queueName  string
	connection *amqp.Connection
	channel    *amqp.Channel
}

type Handler func(data []byte, headers amqp.Table)

type MessageBag struct {
	Message  tgbotapi.MessageConfig
	Priority uint8 // 0..255
}

// CallbackAnswerBag represents a callback query answer
type CallbackAnswerBag struct {
	CallbackAnswer tgbotapi.CallbackConfig
	Priority       uint8 // Should always be 255 for instant response
}

// EditMessageBag represents a message edit operation
type EditMessageBag struct {
	EditMessage tgbotapi.EditMessageTextConfig
	Priority    uint8
}

// ExchangeNotificationBag represents a fanout notification message
type ExchangeNotificationBag struct {
	ExchangeID      int64
	RecipientUserID int64
	Message         tgbotapi.MessageConfig
	Priority        uint8
}

func NewRabbitClient(url string, queueName string) *RabbitClient {
	log.Printf("[RABBIT] Creating new RabbitMQ client for queue: %s", queueName)

	client := &RabbitClient{
		url:       url,
		queueName: queueName,
	}

	err := client.connect()
	if err != nil {
		log.Printf("[RABBIT] Initial connection failed: %v. Will retry...", err)
	}

	return client
}

func (c *RabbitClient) connect() error {
	log.Printf("[RABBIT] Connecting to RabbitMQ at %s", c.url)

	// Close existing connection if any
	if c.connection != nil && !c.connection.IsClosed() {
		c.connection.Close()
	}
	if c.channel != nil {
		c.channel.Close()
	}

	conn, err := amqp.Dial(c.url)
	if err != nil {
		return err
	}
	c.connection = conn

	ch, err := c.connection.Channel()
	if err != nil {
		c.connection.Close()
		return err
	}
	c.channel = ch

	// Declare queue with priority support
	args := amqp.Table{
		"x-max-priority": int32(10),
	}

	_, err = c.channel.QueueDeclare(
		c.queueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		args,  // arguments for priority queue
	)
	if err != nil {
		c.channel.Close()
		c.connection.Close()
		return err
	}

	log.Printf("[RABBIT] Connected successfully to queue: %s", c.queueName)
	return nil
}

func (c *RabbitClient) isConnectionOpen() bool {
	if c.connection == nil || c.connection.IsClosed() {
		return false
	}
	if c.channel == nil {
		return false
	}

	// Test channel by checking if we can get a queue (this will fail if channel is closed)
	_, err := c.channel.QueueDeclarePassive(
		c.queueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // args
	)

	// If queue doesn't exist or channel is closed, we'll get an error
	// For our purposes, any error means the channel isn't working properly
	return err == nil
}

func (c *RabbitClient) ensureConnection() error {
	if !c.isConnectionOpen() {
		log.Printf("[RABBIT] Connection is closed, attempting to reconnect...")
		return c.connect()
	}
	return nil
}

func (c *RabbitClient) PublishTgMessage(messageBag MessageBag) error {
	log.Printf("[RABBIT] Publishing message to user %d with priority %d",
		messageBag.Message.ChatID, messageBag.Priority)

	// Ensure we have a valid connection
	if err := c.ensureConnection(); err != nil {
		log.Printf("[RABBIT] Failed to establish connection: %v", err)
		// Record failed publish metric
		metrics.RecordRabbitMQMessage("published", c.queueName, false)
		return err
	}

	body, err := json.Marshal(messageBag)
	if err != nil {
		log.Printf("[RABBIT] Failed to marshal message: %v", err)
		return err
	}

	err = c.channel.Publish(
		"",          // exchange
		c.queueName, // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
			Priority:     messageBag.Priority,
		},
	)

	if err != nil {
		log.Printf("[RABBIT] Failed to publish message: %v", err)
		// Reset connection on publish error
		c.channel = nil
		c.connection = nil
		// Record failed publish metric
		metrics.RecordRabbitMQMessage("published", c.queueName, false)
		return err
	}

	log.Printf("[RABBIT] Message published successfully")
	// Record successful publish metric
	metrics.RecordRabbitMQMessage("published", c.queueName, true)
	return nil
}

// PublishCallbackAnswer publishes a callback query answer
func (c *RabbitClient) PublishCallbackAnswer(callbackBag CallbackAnswerBag) error {
	log.Printf("[RABBIT] Publishing callback answer %s with priority %d",
		callbackBag.CallbackAnswer.CallbackQueryID, callbackBag.Priority)

	// Ensure we have a valid connection
	if err := c.ensureConnection(); err != nil {
		log.Printf("[RABBIT] Failed to establish connection: %v", err)
		// Record failed publish metric
		metrics.RecordRabbitMQMessage("published", c.queueName, false)
		return err
	}

	body, err := json.Marshal(callbackBag)
	if err != nil {
		log.Printf("[RABBIT] Failed to marshal callback answer: %v", err)
		return err
	}

	err = c.channel.Publish(
		"",          // exchange
		c.queueName, // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
			Priority:     callbackBag.Priority,
			Headers: amqp.Table{
				"message_type": "callback_answer",
			},
		},
	)

	if err != nil {
		log.Printf("[RABBIT] Failed to publish callback answer: %v", err)
		// Reset connection on publish error
		c.channel = nil
		c.connection = nil
		// Record failed publish metric
		metrics.RecordRabbitMQMessage("published", c.queueName, false)
		return err
	}

	log.Printf("[RABBIT] Callback answer published successfully")
	// Record successful publish metric
	metrics.RecordRabbitMQMessage("published", c.queueName, true)
	return nil
}

// PublishEditMessage publishes a message edit operation
func (c *RabbitClient) PublishEditMessage(editBag EditMessageBag) error {
	log.Printf("[RABBIT] Publishing message edit for message %d in chat %d with priority %d",
		editBag.EditMessage.MessageID, editBag.EditMessage.ChatID, editBag.Priority)

	// Ensure we have a valid connection
	if err := c.ensureConnection(); err != nil {
		log.Printf("[RABBIT] Failed to establish connection: %v", err)
		return err
	}

	body, err := json.Marshal(editBag)
	if err != nil {
		log.Printf("[RABBIT] Failed to marshal edit message: %v", err)
		return err
	}

	err = c.channel.Publish(
		"",          // exchange
		c.queueName, // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
			Priority:     editBag.Priority,
			Headers: amqp.Table{
				"message_type": "edit_message",
			},
		},
	)

	if err != nil {
		log.Printf("[RABBIT] Failed to publish edit message: %v", err)
		// Reset connection on publish error
		c.channel = nil
		c.connection = nil
		return err
	}

	log.Printf("[RABBIT] Edit message published successfully")
	return nil
}

// PublishExchangeNotification publishes an exchange notification message
func (c *RabbitClient) PublishExchangeNotification(notificationBag ExchangeNotificationBag) error {
	log.Printf("[RABBIT] Publishing exchange notification for exchange %d to user %d with priority %d",
		notificationBag.ExchangeID, notificationBag.RecipientUserID, notificationBag.Priority)

	// Ensure we have a valid connection
	if err := c.ensureConnection(); err != nil {
		log.Printf("[RABBIT] Failed to establish connection: %v", err)
		return err
	}

	// Convert to regular MessageBag for now - we'll handle the exchange-specific data in the sender
	messageBag := MessageBag{
		Message:  notificationBag.Message,
		Priority: notificationBag.Priority,
	}

	body, err := json.Marshal(messageBag)
	if err != nil {
		log.Printf("[RABBIT] Failed to marshal exchange notification: %v", err)
		return err
	}

	err = c.channel.Publish(
		"",          // exchange
		c.queueName, // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
			Priority:     notificationBag.Priority,
			Headers: amqp.Table{
				"message_type":      "exchange_notification",
				"exchange_id":       notificationBag.ExchangeID,
				"recipient_user_id": notificationBag.RecipientUserID,
			},
		},
	)

	if err != nil {
		log.Printf("[RABBIT] Failed to publish exchange notification: %v", err)
		// Reset connection on publish error
		c.channel = nil
		c.connection = nil
		return err
	}

	log.Printf("[RABBIT] Exchange notification published successfully")
	return nil
}

func (c *RabbitClient) RegisterHandler(handler Handler) {
	log.Printf("[RABBIT] Registering message handler for queue: %s", c.queueName)

	// Rate limiter - 30 messages per second
	rl := ratelimit.New(30)

	go func() {
		for {
			// Ensure we have a valid connection
			if err := c.ensureConnection(); err != nil {
				log.Printf("[RABBIT] Reconnection failed: %v. Retrying in 5 seconds...", err)
				time.Sleep(5 * time.Second)
				continue
			}

			msgs, err := c.channel.Consume(
				c.queueName,
				"",    // consumer tag
				false, // auto-ack
				false, // exclusive
				false, // no-local
				false, // no-wait
				nil,   // args
			)

			if err != nil {
				log.Printf("[RABBIT] Failed to register consumer: %v", err)
				// Reset connection on consumer error
				c.channel = nil
				c.connection = nil
				time.Sleep(5 * time.Second)
				continue
			}

			log.Printf("[RABBIT] Consumer registered, waiting for messages...")

			for msg := range msgs {
				rl.Take() // Rate limiting

				log.Printf("[RABBIT] Processing message")
				handler(msg.Body, msg.Headers)

				if err := msg.Ack(false); err != nil {
					log.Printf("[RABBIT] Failed to acknowledge message: %v", err)
					// Record failed consume metric
					metrics.RecordRabbitMQMessage("consumed", c.queueName, false)
				} else {
					// Record successful consume metric
					metrics.RecordRabbitMQMessage("consumed", c.queueName, true)
				}
			}

			log.Printf("[RABBIT] Consumer channel closed, reconnecting...")
			// Reset connection for reconnection
			c.channel = nil
			c.connection = nil
		}
	}()
}

func (c *RabbitClient) Close() {
	log.Printf("[RABBIT] Closing RabbitMQ connection")
	if c.channel != nil {
		c.channel.Close()
	}
	if c.connection != nil {
		c.connection.Close()
	}
}
