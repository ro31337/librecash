package menu

import (
	"fmt"
	"librecash/context"
	"librecash/objects"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// Define supported languages with their native names
var supportedLanguages = []struct {
	code string
	name string
}{
	{"en", "English"},
	{"ru", "Русский"},
	{"es", "Español"},
	{"pt", "Português"},
	{"fr", "Français"},
	{"de", "Deutsch"},
	{"it", "Italiano"},
	{"pl", "Polski"},
	{"uk", "Українська"},
	{"tr", "Türkçe"},
	{"ar", "العربية"},
	{"fa", "فارسی"},
	{"he", "עברית"},
	{"hi", "हिन्दी"},
	{"id", "Bahasa Indonesia"},
	{"vi", "Tiếng Việt"},
	{"th", "ไทย"},
	{"my", "မြန်မာ"},
	{"kk", "Қазақша"},
	{"az", "Azərbaycan"},
	{"bg", "Български"},
	{"ro", "Română"},
	{"fil", "Filipino"},
	{"zh", "中文"},
	{"zh-TW", "繁體中文(台灣)"},
	{"zh-HK", "繁體中文(香港)"},
	{"zh-CN", "简体中文"},
}

// ShowLanguageSelection displays the language selection menu with inline buttons
func ShowLanguageSelection(user *objects.User, context *context.Context) {
	log.Printf("[LANGUAGE] Showing language selection for user %d", user.UserId)

	// Get current language name
	currentLang := user.GetLanguageName()
	msgText := fmt.Sprintf("🌐 Select your language\nCurrent language: %s", currentLang)

	// Create inline keyboard with language options
	// We'll arrange them in 2 columns for better UX
	var rows [][]tgbotapi.InlineKeyboardButton

	// Create buttons in rows of 2
	for i := 0; i < len(supportedLanguages); i += 2 {
		var row []tgbotapi.InlineKeyboardButton

		// First button in the row
		btn1Text := supportedLanguages[i].name
		if supportedLanguages[i].code == user.LanguageCode {
			btn1Text = "✅ " + btn1Text
		}
		btn1 := tgbotapi.NewInlineKeyboardButtonData(btn1Text, "lang_"+supportedLanguages[i].code)
		row = append(row, btn1)

		// Second button in the row (if exists)
		if i+1 < len(supportedLanguages) {
			btn2Text := supportedLanguages[i+1].name
			if supportedLanguages[i+1].code == user.LanguageCode {
				btn2Text = "✅ " + btn2Text
			}
			btn2 := tgbotapi.NewInlineKeyboardButtonData(btn2Text, "lang_"+supportedLanguages[i+1].code)
			row = append(row, btn2)
		}

		rows = append(rows, row)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	msg := tgbotapi.NewMessage(user.UserId, msgText)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "Markdown"

	context.Send(msg)
}

// HandleLanguageSelection processes the language selection callback
func HandleLanguageSelection(user *objects.User, context *context.Context, callback *tgbotapi.CallbackQuery) {
	// Extract language code from callback data (format: "lang_XX")
	langCode := strings.TrimPrefix(callback.Data, "lang_")
	log.Printf("[LANGUAGE] User %d selected language: %s", user.UserId, langCode)

	// Update user's language
	user.LanguageCode = langCode
	if err := context.Repo.SaveUser(user); err != nil {
		log.Printf("[LANGUAGE] Error saving user language preference: %v", err)
		return
	}

	// Answer the callback to remove loading animation
	callbackAnswer := tgbotapi.NewCallback(callback.ID, "")
	if err := context.AnswerCallbackQuery(callbackAnswer); err != nil {
		log.Printf("[LANGUAGE] Error answering callback: %v", err)
	}

	// Find the native language name from our global languages list
	var langName string
	for _, lang := range supportedLanguages {
		if lang.code == langCode {
			langName = lang.name
			break
		}
	}

	if langName == "" {
		langName = "English" // Fallback
	}

	// Edit the message to show confirmation
	confirmText := fmt.Sprintf(user.Locale().Get("language.changed"), langName)
	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, confirmText)

	context.EditMessage(editMsg)

	log.Printf("[LANGUAGE] Successfully changed language for user %d to %s", user.UserId, langCode)

	// Regenerate the current menu in the new language
	RegenerateCurrentMenu(user, context)
}

// RegenerateCurrentMenu re-displays the current menu in the user's new language
func RegenerateCurrentMenu(user *objects.User, context *context.Context) {
	log.Printf("[LANGUAGE] Regenerating menu %d for user %d in language %s", user.MenuId, user.UserId, user.LanguageCode)

	// Create an empty message to trigger menu regeneration
	message := &tgbotapi.Message{
		From: &tgbotapi.User{ID: int(user.UserId)},
		Chat: &tgbotapi.Chat{ID: user.UserId},
		Text: "",
	}

	// Call the appropriate menu handler based on current menu state
	switch user.MenuId {
	case objects.Menu_USComplianceCheck:
		// Show US compliance check menu in new language
		log.Printf("[LANGUAGE] Regenerating US compliance check menu for user %d", user.UserId)
		complianceHandler := NewUSComplianceMenu()
		complianceHandler.Handle(user, context, message)
		return
	case objects.Menu_Blocked:
		// Show blocked menu in new language
		log.Printf("[LANGUAGE] Regenerating blocked menu for user %d", user.UserId)
		blockedHandler := NewBlockedMenu()
		blockedHandler.Handle(user, context, message)
		return
	case objects.Menu_Init:
		// Show init menu in new language
		log.Printf("[LANGUAGE] Regenerating init menu for user %d", user.UserId)
		initHandler := NewInitMenu()
		initHandler.Handle(user, context, message)
		return
	case objects.Menu_AskLocation:
		// Show ask location menu in new language
		log.Printf("[LANGUAGE] Regenerating ask location menu for user %d", user.UserId)
		locationHandler := NewAskLocationMenu()
		locationHandler.Handle(user, context, message)
		return
	case objects.Menu_SelectRadius:
		// Show select radius menu in new language
		log.Printf("[LANGUAGE] Regenerating select radius menu for user %d", user.UserId)
		radiusHandler := NewSelectRadiusMenu()
		radiusHandler.Handle(user, context, message)
		return
	case objects.Menu_Main:
		// Show main menu in new language
		log.Printf("[LANGUAGE] Regenerating main menu for user %d", user.UserId)
		mainHandler := NewMainMenuHandler(context, user)
		mainHandler.Handle()
		return
	case objects.Menu_AskPhone:
		// Show ask phone menu in new language
		log.Printf("[LANGUAGE] Regenerating ask phone menu for user %d", user.UserId)
		phoneHandler := NewAskPhoneMenu()
		phoneHandler.Handle(user, context, message)
		return
	case objects.Menu_Amount:
		// Show amount menu in new language
		log.Printf("[LANGUAGE] Regenerating amount menu for user %d", user.UserId)
		amountHandler := NewAmountMenuHandler(context, user)
		amountHandler.Handle()
		return
	case objects.Menu_HistoricalFanoutExecute:
		// Show historical fanout execute menu in new language
		log.Printf("[LANGUAGE] Regenerating historical fanout execute menu for user %d", user.UserId)
		fanoutExecuteHandler := NewHistoricalFanoutExecuteMenu()
		fanoutExecuteHandler.Handle(user, context, message)
		return
	case objects.Menu_HistoricalFanoutWait:
		// Show historical fanout wait menu in new language
		log.Printf("[LANGUAGE] Regenerating historical fanout wait menu for user %d", user.UserId)
		fanoutWaitHandler := NewHistoricalFanoutWaitMenu()
		fanoutWaitHandler.Handle(user, context, message)
		return
	default:
		log.Printf("[LANGUAGE] No handler for menu %d, skipping regeneration", user.MenuId)
		return
	}
}
