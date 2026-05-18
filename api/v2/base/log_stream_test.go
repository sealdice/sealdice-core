package base

import (
	"testing"

	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
)

func TestRegisterLogStreamRouteSkipsEmptyDiceManager(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RegisterLogStreamRoute panicked with empty DiceManager: %v", r)
		}
	}()

	RegisterLogStreamRoute(echo.New(), &dice.DiceManager{})
}
