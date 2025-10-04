package menu

import (
	"librecash/context"
	"librecash/metrics"
	"librecash/objects"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Handler interface {
	Handle(user *objects.User, context *context.Context, message *tgbotapi.Message)
}

func HandleMessage(context *context.Context, userId int64, message *tgbotapi.Message) {
	startTime := time.Now()
	log.Printf("[MENU] Handling message from user %d: '%s'", userId, message.Text)

	previousState := objects.Menu_Ban
	iterationCount := 0

	for isStateChanged(context, previousState, userId) {
		iterationCount++
		log.Printf("[MENU_DEBUG] === ITERATION %d START ===", iterationCount)
		log.Printf("[MENU_DEBUG] previousState = %d, checking isStateChanged...", previousState)

		user := context.Repo.FindUser(userId)

		// Init user if not present
		isNewUser := false
		if user == nil {
			log.Printf("[MENU_DEBUG] User is nil, creating new user")
			log.Printf("[MENU] Creating new user %d", userId)
			isNewUser = true
			user = &objects.User{
				UserId:       userId,
				MenuId:       objects.Menu_USComplianceCheck,
				LanguageCode: "en", // Default to English
			}
		}

		log.Printf("[MENU_DEBUG] Found user: MenuId = %d", user.MenuId)
		log.Printf("[MENU_DEBUG] isStateChanged result: %t (%d != %d)",
			user.MenuId != previousState, user.MenuId, previousState)

		// Save recent user information - only if data actually changed (PRD019)
		if message.From != nil &&
			(message.From.UserName != "" || message.From.FirstName != "" || message.From.LastName != "" ||
				(isNewUser && message.From.LanguageCode != "")) {

			needsUpdate := false

			// Only update username if it's not empty AND different
			if message.From.UserName != "" && user.Username != message.From.UserName {
				log.Printf("[MENU] Username changed for user %d: '%s' -> '%s'",
					userId, user.Username, message.From.UserName)
				user.Username = message.From.UserName
				needsUpdate = true
			}

			// Only update firstName if it's not empty AND different
			if message.From.FirstName != "" && user.FirstName != message.From.FirstName {
				log.Printf("[MENU] FirstName changed for user %d: '%s' -> '%s'",
					userId, user.FirstName, message.From.FirstName)
				user.FirstName = message.From.FirstName
				needsUpdate = true
			}

			// Only update lastName if it's not empty AND different
			if message.From.LastName != "" && user.LastName != message.From.LastName {
				log.Printf("[MENU] LastName changed for user %d: '%s' -> '%s'",
					userId, user.LastName, message.From.LastName)
				user.LastName = message.From.LastName
				needsUpdate = true
			}

			// Only update language for new users (not when user manually changed it)
			if isNewUser && message.From.LanguageCode != "" && user.LanguageCode != message.From.LanguageCode {
				log.Printf("[MENU] Setting language for new user %d: '%s' -> '%s'",
					userId, user.LanguageCode, message.From.LanguageCode)
				user.LanguageCode = message.From.LanguageCode
				needsUpdate = true
			}

			// Only save if something actually changed
			if needsUpdate {
				log.Printf("[MENU] Saving updated user info for %d", userId)
				context.Repo.SaveUser(user)
			} else {
				log.Printf("[MENU] No user data changes for %d", userId)
			}
		} else if message.From != nil {
			log.Printf("[MENU] Skipping user data update for %d (empty or unchanged data)", userId)
		}

		// Handle /start command
		if message.Text == "/start" {
			log.Printf("[MENU] User %d sent /start command", userId)
			oldMenuId := user.MenuId
			user.MenuId = objects.Menu_USComplianceCheck
			message.Text = ""
			context.Repo.SaveUser(user)

			// Record menu transition metric
			metrics.RecordMenuTransition(oldMenuId, user.MenuId, user.GetSupportedLanguageCode())

			// Record metrics (PRD022)
			userType := "returning"
			if isNewUser {
				metrics.RecordNewUser(user.GetSupportedLanguageCode())
				userType = "new"
			}
			metrics.RecordCommand("/start", user.GetSupportedLanguageCode(), userType)
		}

		// Handle /location command (PRD025)
		if strings.ToLower(message.Text) == "/location" {
			log.Printf("[MENU] User %d sent /location command", userId)
			oldMenuId := user.MenuId
			user.MenuId = objects.Menu_SelectRadius
			message.Text = ""
			context.Repo.SaveUser(user)

			// Record menu transition metric
			metrics.RecordMenuTransition(oldMenuId, user.MenuId, user.GetSupportedLanguageCode())

			// Record command metric
			metrics.RecordCommand("/location", user.GetSupportedLanguageCode(), "existing")

			log.Printf("[MENU] User %d transitioned to radius selection via /location command", userId)
		}

		// Handle /language command
		if message.Text == "/language" {
			log.Printf("[MENU] User %d sent /language command", userId)

			// Record command metric (PRD022)
			userType := "returning"
			if isNewUser {
				userType = "new"
			}
			metrics.RecordCommand("/language", user.GetSupportedLanguageCode(), userType)

			ShowLanguageSelection(user, context)
			return
		}

		// Handle /exchange command
		if message.Text == "/exchange" {
			log.Printf("[MENU] User %d sent /exchange command", userId)

			// Record command metric (PRD022)
			userType := "returning"
			if isNewUser {
				userType = "new"
			}
			metrics.RecordCommand("/exchange", user.GetSupportedLanguageCode(), userType)

			// Check if user has completed initialization
			isInitialized := user.Lat != 0 && user.Lon != 0 && user.SearchRadiusKm != nil

			if !isInitialized {
				log.Printf("[MENU] User %d tried /exchange but not initialized", userId)
				errorMsg := tgbotapi.NewMessage(userId, user.Locale().Get("exchange_command.not_initialized"))
				context.Send(errorMsg)
				return
			}

			log.Printf("[MENU] User %d accessing exchange menu via command", userId)

			// Reset to main menu state
			oldMenuId := user.MenuId
			user.MenuId = objects.Menu_Main
			context.Repo.SaveUser(user)

			// Record menu transition metric
			metrics.RecordMenuTransition(oldMenuId, user.MenuId, user.GetSupportedLanguageCode())

			// Show main menu
			mainHandler := NewMainMenuHandler(context, user)
			mainHandler.Handle()
			return
		}

		previousState = user.MenuId
		log.Printf("[MENU_DEBUG] Updated previousState to %d", previousState)

		var handler Handler

		switch user.MenuId {
		case objects.Menu_USComplianceCheck:
			handler = NewUSComplianceMenu()
		case objects.Menu_Blocked:
			handler = NewBlockedMenu()
		case objects.Menu_Init:
			handler = NewInitMenu()
		case objects.Menu_AskLocation:
			handler = NewAskLocationMenu()
		case objects.Menu_SelectRadius:
			handler = NewSelectRadiusMenu()
		case objects.Menu_AskPhone:
			handler = NewAskPhoneMenu()
		case objects.Menu_HistoricalFanoutExecute:
			handler = NewHistoricalFanoutExecuteMenu()
		case objects.Menu_HistoricalFanoutWait:
			handler = NewHistoricalFanoutWaitMenu()
		case objects.Menu_Main:
			// Show main menu
			log.Printf("[MENU] Showing main menu to user %d", userId)
			mainHandler := NewMainMenuHandler(context, user)
			mainHandler.Handle()
			return
		case objects.Menu_Amount:
			// Amount menu is shown via transition from main menu
			log.Printf("[MENU] User %d is in amount menu state", userId)
			return
		default:
			log.Printf("[MENU] Handler not implemented for menu with id %d", user.MenuId)
			return
		}

		log.Printf("[MENU_DEBUG] About to call handler.Handle() for menu %d", user.MenuId)
		if handler != nil {
			handler.Handle(user, context, message)
		}
		log.Printf("[MENU_DEBUG] handler.Handle() completed for menu %d", user.MenuId)

		// Проверяем состояние после Handle()
		userAfter := context.Repo.FindUser(userId)
		log.Printf("[MENU_DEBUG] User state after Handle(): MenuId = %d", userAfter.MenuId)

		// Important! Reset message to indicate it has been processed
		message = &tgbotapi.Message{}
		log.Printf("[MENU_DEBUG] === ITERATION %d END ===", iterationCount)
	}

	log.Printf("[MENU_DEBUG] Loop finished after %d iterations", iterationCount)

	duration := time.Since(startTime)
	log.Printf("[MENU] Message handling completed for user %d (duration: %v)", userId, duration)
}

func isStateChanged(context *context.Context, previousState objects.MenuId, userId int64) bool {
	log.Printf("[MENU_DEBUG] isStateChanged called: previousState = %d", previousState)

	user := context.Repo.FindUser(userId)

	if user == nil {
		log.Printf("[MENU_DEBUG] User is nil, returning true")
		return true
	}

	result := user.MenuId != previousState
	log.Printf("[MENU_DEBUG] isStateChanged result: %t (current: %d, previous: %d)",
		result, user.MenuId, previousState)

	return result
}

// ContinueMenuProcessing continues menu processing after a state change in callback
func ContinueMenuProcessing(context *context.Context, userId int64) {
	log.Printf("[MENU] Continuing menu processing for user %d after callback state change", userId)

	// Create an empty message to trigger menu processing
	emptyMessage := &tgbotapi.Message{
		From: &tgbotapi.User{
			ID: int(userId),
		},
	}

	// Call HandleMessage to continue processing with the new state
	HandleMessage(context, userId, emptyMessage)
}

// HandleCallback handles inline button callbacks
func HandleCallback(context *context.Context, userId int64, callback *tgbotapi.CallbackQuery) {
	log.Printf("[MENU] Handling callback from user %d: data=%s", userId, callback.Data)

	user := context.Repo.FindUser(userId)
	if user == nil {
		log.Printf("[MENU] User %d not found for callback", userId)
		return
	}

	// Handle language selection callbacks FIRST - they should work from any menu
	if strings.HasPrefix(callback.Data, "lang_") {
		// Handle language selection callback
		HandleLanguageSelection(user, context, callback)
		return
	}

	// Route to appropriate handler based on menu state and callback data
	if user.MenuId == objects.Menu_USComplianceCheck {
		handler := NewUSComplianceMenu()
		handler.HandleCallback(user, context, callback)

	} else if user.MenuId == objects.Menu_SelectRadius {
		handler := NewSelectRadiusMenu()
		handler.HandleCallback(user, context, callback)

	} else if user.MenuId == objects.Menu_HistoricalFanoutWait {
		handler := NewHistoricalFanoutWaitMenu()
		handler.HandleCallback(user, context, callback)

	} else if strings.HasPrefix(callback.Data, "main:") {
		// Handle main menu callbacks
		HandleMainMenuCallback(context, callback, user)
	} else if callback.Data == "historical_fanout:continue" {
		// Handle historical fanout continue button - user is already in main menu
		log.Printf("[MENU] Handling historical_fanout:continue for user %d (already in main menu)", user.UserId)

		// Answer the callback
		callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
		context.AnswerCallbackQuery(callbackAnswer)

		// Remove the continue button by editing the message (remove keyboard)
		locale := user.Locale()
		editMsg := tgbotapi.NewEditMessageText(
			user.UserId,
			callback.Message.MessageID,
			locale.Get("historical_fanout.message"), // Same message but without button
		)
		editMsg.ParseMode = "HTML"
		// No ReplyMarkup = no buttons
		context.EditMessage(editMsg)

		// Show main menu so user can proceed with exchanges
		log.Printf("[MENU] Showing main menu after continue button")
		mainHandler := NewMainMenuHandler(context, user)
		mainHandler.Handle()
	} else if strings.HasPrefix(callback.Data, "amount:") {
		// Handle amount menu callbacks
		HandleAmountMenuCallback(context, callback, user)
	} else if strings.HasPrefix(callback.Data, "contact:") {
		// Handle contact request callbacks
		HandleContactRequestCallback(context, callback, user)
	} else if strings.HasPrefix(callback.Data, "delete:") {
		// Handle delete exchange callbacks
		HandleDeleteExchangeCallback(context, callback, user)
	} else {
		log.Printf("[MENU] No callback handler for menu %d", user.MenuId)
	}
}
