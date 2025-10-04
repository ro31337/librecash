package menu

import (
	"librecash/objects"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHistoricalFanoutExecuteMenu(t *testing.T) {
	handler := NewHistoricalFanoutExecuteMenu()
	assert.NotNil(t, handler, "HistoricalFanoutExecuteMenu should be created")
}

func TestNewHistoricalFanoutWaitMenu(t *testing.T) {
	handler := NewHistoricalFanoutWaitMenu()
	assert.NotNil(t, handler, "HistoricalFanoutWaitMenu should be created")
}

func TestMenuConstants(t *testing.T) {
	// Test that menu constants are defined correctly
	assert.Equal(t, objects.MenuId(290), objects.Menu_HistoricalFanoutExecute, "Menu_HistoricalFanoutExecute should be 290")
	assert.Equal(t, objects.MenuId(295), objects.Menu_HistoricalFanoutWait, "Menu_HistoricalFanoutWait should be 295")

	// Test that they are in correct order
	assert.True(t, objects.Menu_AskPhone < objects.Menu_HistoricalFanoutExecute, "AskPhone should come before HistoricalFanoutExecute")
	assert.True(t, objects.Menu_HistoricalFanoutExecute < objects.Menu_HistoricalFanoutWait, "Execute should come before Wait")
	assert.True(t, objects.Menu_HistoricalFanoutWait < objects.Menu_Main, "Wait should come before Main")
}

func TestMenuFlow(t *testing.T) {
	// Test the expected menu flow order
	expectedFlow := []objects.MenuId{
		objects.Menu_Init,
		objects.Menu_AskLocation,
		objects.Menu_SelectRadius,
		objects.Menu_AskPhone,
		objects.Menu_HistoricalFanoutExecute,
		objects.Menu_HistoricalFanoutWait,
		objects.Menu_Main,
		objects.Menu_Amount,
	}

	for i := 1; i < len(expectedFlow); i++ {
		assert.True(t, expectedFlow[i-1] < expectedFlow[i],
			"Menu %d should come before menu %d", expectedFlow[i-1], expectedFlow[i])
	}
}
