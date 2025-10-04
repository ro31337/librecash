package menu

import (
	"librecash/context"
	"librecash/fanout"
	"librecash/metrics"
	"librecash/objects"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type HistoricalFanoutExecuteMenu struct{}

func NewHistoricalFanoutExecuteMenu() *HistoricalFanoutExecuteMenu {
	return &HistoricalFanoutExecuteMenu{}
}

func (h *HistoricalFanoutExecuteMenu) Handle(user *objects.User, context *context.Context, message *tgbotapi.Message) {
	log.Printf("[HISTORICAL_FANOUT_EXECUTE] Handling historical fanout execute for user %d", user.UserId)

	// Check if historical fanout should be triggered
	shouldFanout, err := context.Repo.ShouldTriggerHistoricalFanout(user.UserId)
	if err != nil {
		log.Printf("[HISTORICAL_FANOUT_EXECUTE] Error checking fanout trigger for user %d: %v", user.UserId, err)
		h.transitionToMain(user, context)
		return
	}

	if !shouldFanout {
		log.Printf("[HISTORICAL_FANOUT_EXECUTE] No changes detected for user %d, skipping fanout", user.UserId)
		h.transitionToMain(user, context)
		return
	}

	log.Printf("[HISTORICAL_FANOUT_EXECUTE] Changes detected for user %d, checking for historical exchanges", user.UserId)

	// Get user's search radius
	if user.SearchRadiusKm == nil {
		log.Printf("[HISTORICAL_FANOUT_EXECUTE] User %d has no search radius, skipping fanout", user.UserId)
		h.transitionToMain(user, context)
		return
	}

	// Find historical exchanges first to check if there are any
	historicalExchanges, err := context.Repo.FindHistoricalExchangesInRadius(
		user.Lat, user.Lon, *user.SearchRadiusKm, user.UserId,
	)
	if err != nil {
		log.Printf("[HISTORICAL_FANOUT_EXECUTE] Error finding historical exchanges for user %d: %v", user.UserId, err)
		h.transitionToMain(user, context)
		return
	}

	if len(historicalExchanges) == 0 {
		log.Printf("[HISTORICAL_FANOUT_EXECUTE] No historical exchanges found for user %d, skipping fanout", user.UserId)
		h.transitionToMain(user, context)
		return
	}

	// Broadcast historical exchanges
	fanoutService := fanout.NewFanoutService(context)
	err = fanoutService.BroadcastHistoricalExchanges(user.UserId, user.Lat, user.Lon)
	if err != nil {
		log.Printf("[HISTORICAL_FANOUT_EXECUTE] Error broadcasting historical exchanges for user %d: %v", user.UserId, err)
		h.transitionToMain(user, context)
		return
	}

	// Transition to wait menu to show continuation message
	log.Printf("[HISTORICAL_FANOUT_EXECUTE] Fanout completed, transitioning to wait menu for user %d", user.UserId)
	h.transitionToWait(user, context)
	return
}

func (h *HistoricalFanoutExecuteMenu) transitionToWait(user *objects.User, context *context.Context) {
	log.Printf("[HISTORICAL_FANOUT_EXECUTE] Transitioning user %d to historical fanout wait menu", user.UserId)

	// Use the global transition function to avoid circular dependency
	TransitionToHistoricalFanoutWait(context, user)
}

func (h *HistoricalFanoutExecuteMenu) transitionToMain(user *objects.User, context *context.Context) {
	log.Printf("[HISTORICAL_FANOUT_EXECUTE] Transitioning user %d to main menu", user.UserId)

	// Update user state to main menu
	oldMenuId := user.MenuId
	user.MenuId = objects.Menu_Main
	if err := context.Repo.SaveUser(user); err != nil {
		log.Printf("[HISTORICAL_FANOUT_EXECUTE] Error updating user state for user %d: %v", user.UserId, err)
		return
	}

	// Record menu transition metric
	metrics.RecordMenuTransition(oldMenuId, user.MenuId, user.GetSupportedLanguageCode())

	// НЕ вызываем mainHandler.Handle() - пусть menu loop сам вызовет handler для нового состояния
	log.Printf("[HISTORICAL_FANOUT_EXECUTE] State changed to Main menu, menu loop will handle it")
}

// TransitionToHistoricalFanoutExecute transitions user to historical fanout execute menu
func TransitionToHistoricalFanoutExecute(context *context.Context, user *objects.User) {
	log.Printf("[HISTORICAL_FANOUT_EXECUTE] Transitioning user %d to historical fanout execute menu", user.UserId)

	// Update user state
	oldMenuId := user.MenuId
	user.MenuId = objects.Menu_HistoricalFanoutExecute
	if err := context.Repo.SaveUser(user); err != nil {
		log.Printf("[HISTORICAL_FANOUT_EXECUTE] Error updating user state: %v", err)
		return
	}

	// Record menu transition metric
	metrics.RecordMenuTransition(oldMenuId, user.MenuId, user.GetSupportedLanguageCode())

	// НЕ вызываем handler.Handle() - пусть menu loop сам вызовет handler для нового состояния
	log.Printf("[HISTORICAL_FANOUT_EXECUTE] State changed to %d, menu loop will handle it", user.MenuId)
}
