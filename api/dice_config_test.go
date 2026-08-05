package api //nolint:testpackage

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"sealdice-core/dice"
)

func TestDiceConfigSetAcceptsCRNGRandomMode(t *testing.T) {
	t.Chdir(t.TempDir())

	previousDice := myDice
	previousDM := dm
	t.Cleanup(func() {
		myDice = previousDice
		dm = previousDM
	})

	testDice := &dice.Dice{
		BaseConfig: dice.BaseConfig{DataDir: "."},
		Config:     dice.NewConfig(nil),
		Logger:     zap.NewNop().Sugar(),
	}
	manager := &dice.DiceManager{
		Dice:        []*dice.Dice{testDice},
		JustForTest: true,
	}
	token := "test-token"
	manager.AccessTokens.Store(token, true)
	testDice.Parent = manager
	myDice = testDice
	dm = manager

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/sd-api/configs/dice", strings.NewReader(`{"diceRandomMode":"crng"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Token", token)
	rec := httptest.NewRecorder()

	if err := DiceConfigSet(e.NewContext(req, rec)); err != nil {
		t.Fatalf("DiceConfigSet() error = %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := testDice.Config.DiceRandomMode; got != string(dice.DiceRandomModeCRNG) {
		t.Fatalf("DiceRandomMode = %q, want %q", got, dice.DiceRandomModeCRNG)
	}
}
