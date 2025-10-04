package fanout

import (
	"fmt"
	"librecash/context"
	"librecash/metrics"
	"librecash/objects"
	"librecash/rabbit"
	"log"
	"math"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/leonelquinteros/gotext"
)

type FanoutService struct {
	context *context.Context
}

// NewFanoutService creates a new fanout service instance
func NewFanoutService(context *context.Context) *FanoutService {
	return &FanoutService{
		context: context,
	}
}

// BroadcastExchange broadcasts an exchange offer to all nearby users
func (f *FanoutService) BroadcastExchange(exchange *objects.Exchange) error {
	log.Printf("[FANOUT] Broadcasting exchange %d to nearby users", exchange.ID)

	// 1. Get initiator's search radius
	initiator := f.context.Repo.FindUser(exchange.UserID)
	if initiator == nil {
		return fmt.Errorf("initiator user %d not found", exchange.UserID)
	}
	if initiator.SearchRadiusKm == nil {
		return fmt.Errorf("initiator user %d has no search radius set", exchange.UserID)
	}

	log.Printf("[FANOUT] Initiator %d has search radius %d km", exchange.UserID, *initiator.SearchRadiusKm)

	// TODO: TEMPORARY FOR TESTING - ALWAYS USE 90000 KM RADIUS
	testRadius := 90000
	log.Printf("[FANOUT] TESTING MODE: Overriding radius to %d km", testRadius)

	// 2. Find nearby users (including initiator for debugging/future features)
	nearbyUsers, err := f.context.Repo.FindUsersInRadius(
		exchange.Lat, exchange.Lon,
		testRadius, // *initiator.SearchRadiusKm,
	)
	if err != nil {
		return fmt.Errorf("failed to find nearby users: %v", err)
	}

	log.Printf("[FANOUT] Found %d nearby users for exchange %d", len(nearbyUsers), exchange.ID)

	// 3. Queue notification messages via RabbitMQ (users in main menu OR exchange author)
	for _, user := range nearbyUsers {
		// Send to users in main menu OR exchange author (needs delete button)
		if user.MenuId == objects.Menu_Main {
			if err := f.queueNotificationMessage(exchange, user, initiator); err != nil {
				log.Printf("[FANOUT] Failed to queue notification for user %d: %v", user.UserId, err)
				// Continue with other users even if one fails
			} else {
				log.Printf("[FANOUT] Queuing notification for user %d about exchange %d", user.UserId, exchange.ID)
			}
		} else if user.UserId == exchange.UserID {
			if err := f.queueNotificationMessage(exchange, user, initiator); err != nil {
				log.Printf("[FANOUT] Failed to queue notification for author %d: %v", user.UserId, err)
			} else {
				log.Printf("[FANOUT] Sending to author %d (own exchange %d, state: %d)",
					user.UserId, exchange.ID, user.MenuId)
			}
		} else {
			log.Printf("[FANOUT] Skipping user %d (not in main menu, state: %d)", user.UserId, user.MenuId)
		}
	}

	log.Printf("[FANOUT] Successfully queued notifications for exchange %d", exchange.ID)
	return nil
}

// queueNotificationMessage creates and queues a notification message for a specific user
func (f *FanoutService) queueNotificationMessage(exchange *objects.Exchange, recipient *objects.User, initiator *objects.User) error {
	log.Printf("[FANOUT] Queuing notification for user %d about exchange %d", recipient.UserId, exchange.ID)

	// Calculate distance between initiator and recipient
	distance := f.calculateDistance(exchange.Lat, exchange.Lon, recipient.Lat, recipient.Lon)
	distanceKm := int(math.Round(distance))

	// Build notification message
	messageText := f.buildNotificationMessage(exchange, recipient, distanceKm)

	// Create inline keyboard with appropriate button based on user role
	var keyboard tgbotapi.InlineKeyboardMarkup
	if recipient.UserId == exchange.UserID {
		// Author sees "Delete" button
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					recipient.Locale().Get("fanout.button_delete"),
					fmt.Sprintf("delete:%d", exchange.ID),
				),
			),
		)
	} else {
		// Recipients see "Show contact" button
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					recipient.Locale().Get("fanout.button_show_contact"),
					fmt.Sprintf("contact:%d", exchange.ID),
				),
			),
		)
	}

	// Create message config
	msg := tgbotapi.NewMessage(recipient.UserId, messageText)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "HTML"

	// Create exchange notification bag
	notificationBag := &rabbit.ExchangeNotificationBag{
		ExchangeID:      exchange.ID,
		RecipientUserID: recipient.UserId,
		Message:         msg,
		Priority:        100, // Medium priority for fanout notifications
	}

	// Queue the message
	err := f.context.RabbitPublish.PublishExchangeNotification(*notificationBag)

	// Record fanout message metric
	metrics.RecordFanoutMessage("exchange_notification", recipient.GetSupportedLanguageCode(), err == nil)

	return err
}

// buildNotificationMessage constructs the notification message text
func (f *FanoutService) buildNotificationMessage(exchange *objects.Exchange, recipient *objects.User, distanceKm int) string {
	locale := recipient.Locale()
	isAuthor := recipient.UserId == exchange.UserID

	// Header - different for author vs recipient
	var message string
	if isAuthor {
		message = locale.Get("fanout.author_notification_header") + "\n\n"
	} else {
		message = locale.Get("fanout.notification_header") + "\n\n"
	}

	// What they have and need - different for author vs recipient
	if exchange.ExchangeDirection == objects.ExchangeDirectionCashToCrypto {
		if isAuthor {
			message += locale.Get("fanout.author_notification_have_cash") + "\n"
			message += locale.Get("fanout.author_notification_need_crypto") + "\n"
		} else {
			message += locale.Get("fanout.notification_have_cash") + "\n"
			message += locale.Get("fanout.notification_need_crypto") + "\n"
		}
	} else {
		if isAuthor {
			message += locale.Get("fanout.author_notification_have_crypto") + "\n"
			message += locale.Get("fanout.author_notification_need_cash") + "\n"
		} else {
			message += locale.Get("fanout.notification_have_crypto") + "\n"
			message += locale.Get("fanout.notification_need_cash") + "\n"
		}
	}

	// Amount (same for both)
	if exchange.AmountUSD != nil {
		message += fmt.Sprintf(locale.Get("fanout.notification_amount"), *exchange.AmountUSD) + "\n"
	}

	// Distance - only for recipients, not for authors
	if !isAuthor {
		message += fmt.Sprintf(locale.Get("fanout.notification_distance"), distanceKm)
	}

	return message
}

// calculateDistance calculates the distance between two points in kilometers using Haversine formula
func (f *FanoutService) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	// Haversine formula
	dlat := lat2Rad - lat1Rad
	dlon := lon2Rad - lon1Rad

	a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

// BroadcastHistoricalExchanges broadcasts historical exchanges to a user who changed location
func (f *FanoutService) BroadcastHistoricalExchanges(userID int64, lat, lon float64) error {
	log.Printf("[FANOUT] Broadcasting historical exchanges to user %d at location (%f, %f)", userID, lat, lon)

	// 1. Get user and their search radius
	user := f.context.Repo.FindUser(userID)
	if user == nil {
		return fmt.Errorf("user %d not found", userID)
	}
	if user.SearchRadiusKm == nil {
		log.Printf("[FANOUT] User %d has no search radius set, skipping historical fanout", userID)
		return nil
	}

	log.Printf("[FANOUT] User %d has search radius %d km", userID, *user.SearchRadiusKm)

	// TODO: TEMPORARY FOR TESTING - ALWAYS USE 90000 KM RADIUS
	testRadius := 90000
	log.Printf("[FANOUT] TESTING MODE: Overriding historical radius to %d km", testRadius)

	// 2. Find historical exchanges in the area
	historicalExchanges, err := f.context.Repo.FindHistoricalExchangesInRadius(
		lat, lon,
		testRadius, // *user.SearchRadiusKm,
		userID,     // exclude user's own exchanges
	)
	if err != nil {
		return fmt.Errorf("failed to find historical exchanges: %v", err)
	}

	log.Printf("[HISTORICAL_FANOUT] Found %d historical exchanges for user %d", len(historicalExchanges), userID)

	// 3. Queue historical notification messages via RabbitMQ
	sentCount := 0
	for _, exchange := range historicalExchanges {
		if err := f.queueHistoricalNotificationMessage(exchange, user); err != nil {
			log.Printf("[HISTORICAL_FANOUT] Failed to queue historical notification for exchange %d: %v", exchange.ID, err)
			// Continue with other exchanges even if one fails
		} else {
			log.Printf("[HISTORICAL_FANOUT] Sending historical exchange %d to user %d (state: %d)",
				exchange.ID, userID, user.MenuId)
			sentCount++
		}
	}

	log.Printf("[HISTORICAL_FANOUT] Successfully queued %d historical notifications for user %d", sentCount, userID)
	return nil
}

// queueHistoricalNotificationMessage creates and queues a historical notification message
func (f *FanoutService) queueHistoricalNotificationMessage(exchange *objects.Exchange, recipient *objects.User) error {
	log.Printf("[FANOUT] Queuing historical notification for user %d about exchange %d", recipient.UserId, exchange.ID)

	// Get exchange author for distance calculation
	initiator := f.context.Repo.FindUser(exchange.UserID)
	if initiator == nil {
		return fmt.Errorf("exchange author %d not found", exchange.UserID)
	}

	// Calculate distance between initiator and recipient
	distance := f.calculateDistance(exchange.Lat, exchange.Lon, recipient.Lat, recipient.Lon)
	distanceKm := int(math.Round(distance))

	// Build historical notification message with time ago
	messageText := f.buildHistoricalNotificationMessage(exchange, recipient, distanceKm)

	// Create inline keyboard with "Show contact" button (recipients only get contact button for historical)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				recipient.Locale().Get("fanout.button_show_contact"),
				fmt.Sprintf("contact:%d", exchange.ID),
			),
		),
	)

	// Create message config
	msg := tgbotapi.NewMessage(recipient.UserId, messageText)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "HTML"

	// Create exchange notification bag with lower priority for historical
	notificationBag := &rabbit.ExchangeNotificationBag{
		ExchangeID:      exchange.ID,
		RecipientUserID: recipient.UserId,
		Message:         msg,
		Priority:        80, // Lower priority for historical notifications
	}

	// Queue the message
	err := f.context.RabbitPublish.PublishExchangeNotification(*notificationBag)

	// Record fanout message metric for historical
	metrics.RecordFanoutMessage("historical_notification", recipient.GetSupportedLanguageCode(), err == nil)

	return err
}

// buildHistoricalNotificationMessage constructs the historical notification message text
func (f *FanoutService) buildHistoricalNotificationMessage(exchange *objects.Exchange, recipient *objects.User, distanceKm int) string {
	locale := recipient.Locale()

	// Historical header with time ago
	timeAgo := f.formatTimeAgo(exchange.CreatedAt, LocaleWrapper{locale})
	message := fmt.Sprintf(locale.Get("fanout.historical_notification_header"), timeAgo) + "\n\n"

	// What they have and need (same as regular notification)
	if exchange.ExchangeDirection == objects.ExchangeDirectionCashToCrypto {
		message += locale.Get("fanout.notification_have_cash") + "\n"
		message += locale.Get("fanout.notification_need_crypto") + "\n"
	} else {
		message += locale.Get("fanout.notification_have_crypto") + "\n"
		message += locale.Get("fanout.notification_need_cash") + "\n"
	}

	// Amount (if specified)
	if exchange.AmountUSD != nil {
		message += fmt.Sprintf(locale.Get("fanout.notification_amount"), *exchange.AmountUSD) + "\n"
	}

	// Distance
	message += fmt.Sprintf(locale.Get("fanout.notification_distance"), distanceKm)

	return message
}

// LocaleInterface defines the interface for localization
type LocaleInterface interface {
	Get(key string) string
}

// LocaleWrapper wraps gotext.Po to implement LocaleInterface
type LocaleWrapper struct {
	po *gotext.Po
}

func (lw LocaleWrapper) Get(key string) string {
	switch key {
	case "time.minutes_ago":
		return lw.po.Get("time.minutes_ago")
	case "time.minutes_ago_1":
		return lw.po.Get("time.minutes_ago_1")
	case "time.hours_ago":
		return lw.po.Get("time.hours_ago")
	case "time.hours_ago_1":
		return lw.po.Get("time.hours_ago_1")
	case "time.days_ago":
		return lw.po.Get("time.days_ago")
	case "time.days_ago_1":
		return lw.po.Get("time.days_ago_1")
	case "time.weeks_ago":
		return lw.po.Get("time.weeks_ago")
	case "time.weeks_ago_1":
		return lw.po.Get("time.weeks_ago_1")
	default:
		return key
	}
}

// formatTimeAgo formats time difference in a localized, abbreviated way
func (f *FanoutService) formatTimeAgo(createdAt time.Time, locale LocaleInterface) string {
	// Use UTC for both times to avoid timezone issues
	now := time.Now().UTC()
	createdAtUTC := createdAt.UTC()
	diff := now.Sub(createdAtUTC)

	minutes := int(diff.Minutes())
	hours := int(diff.Hours())
	days := int(diff.Hours() / 24)
	weeks := days / 7

	if minutes < 60 {
		if minutes <= 1 {
			return locale.Get("time.minutes_ago_1")
		}
		template := locale.Get("time.minutes_ago")
		return fmt.Sprintf(template, minutes)
	} else if hours < 24 {
		if hours == 1 {
			return locale.Get("time.hours_ago_1")
		}
		template := locale.Get("time.hours_ago")
		return fmt.Sprintf(template, hours)
	} else if days < 7 {
		if days == 1 {
			return locale.Get("time.days_ago_1")
		}
		template := locale.Get("time.days_ago")
		return fmt.Sprintf(template, days)
	} else if days <= 30 {
		if weeks == 1 {
			return locale.Get("time.weeks_ago_1")
		}
		template := locale.Get("time.weeks_ago")
		return fmt.Sprintf(template, weeks)
	} else {
		// Should not happen as we filter out exchanges older than 30 days
		return locale.Get("time.weeks_ago_1") // fallback
	}
}
