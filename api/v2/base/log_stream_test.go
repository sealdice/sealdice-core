package base_test

import (
	"testing"

	"github.com/gofiber/fiber/v2"

	base "sealdice-core/api/v2/base"
	"sealdice-core/dice"
)

func TestRegisterLogStreamRouteSkipsEmptyDiceManager(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RegisterLogStreamRoute panicked with empty DiceManager: %v", r)
		}
	}()

	base.RegisterLogStreamRoute(fiber.New(), &dice.DiceManager{})
}
