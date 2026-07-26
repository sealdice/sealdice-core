package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"sealdice-core/dice"
)

func TestCustomTextSaveReturnsErrorAndRollsBackWhenSaveTextFails(t *testing.T) {
	dataDir := t.TempDir()
	templatePath := filepath.Join(dataDir, "configs", "text-template.yaml")
	if err := os.MkdirAll(templatePath, 0o755); err != nil {
		t.Fatalf("MkdirAll(templatePath) error = %v", err)
	}

	token := setupCustomTextSaveTestDice(t, dataDir)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/sd-api/configs/customText/save", strings.NewReader(
		`{"category":"核心","data":{}}`,
	))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Token", token)
	rec := httptest.NewRecorder()

	if err := customTextSave(e.NewContext(req, rec)); err != nil {
		t.Fatalf("customTextSave() error = %v", err)
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body=%q", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
	if _, exists := myDice.TextMapRaw["核心"]["原文案"]; !exists {
		t.Fatalf("TextMapRaw was not rolled back: %#v", myDice.TextMapRaw)
	}
	if _, exists := myDice.TextMapHelpInfo["核心"]["原文案"]; !exists {
		t.Fatalf("TextMapHelpInfo was not rolled back: %#v", myDice.TextMapHelpInfo)
	}
	if _, exists := myDice.TextMap["核心:原文案"]; !exists {
		t.Fatalf("TextMap was not rolled back: %#v", myDice.TextMap)
	}
}

func setupCustomTextSaveTestDice(t *testing.T, dataDir string) string {
	t.Helper()

	testDice := &dice.Dice{
		BaseConfig: dice.BaseConfig{DataDir: dataDir},
		Logger:     zap.NewNop().Sugar(),
		TextMapRaw: dice.TextTemplateWithWeightDict{
			"核心": dice.TextTemplateWithWeight{
				"原文案": []dice.TextTemplateItem{{"old", 1}},
			},
		},
		TextMapHelpInfo: dice.TextTemplateWithHelpDict{
			"核心": dice.TextTemplateHelpGroup{
				"原文案": &dice.TextTemplateHelpItem{
					Origin: []dice.TextTemplateItem{{"old", 1}},
				},
			},
		},
	}
	testDice.GenerateTextMap()

	manager := &dice.DiceManager{Dice: []*dice.Dice{testDice}}
	token := "test-token"
	manager.AccessTokens.Store(token, true)
	testDice.Parent = manager

	previousDice := myDice
	previousManager := dm
	myDice = testDice
	dm = manager
	t.Cleanup(func() {
		myDice = previousDice
		dm = previousManager
	})

	return token
}
