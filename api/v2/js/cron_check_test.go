package js_test

import (
	"errors"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	. "sealdice-core/api/v2/js"
	"sealdice-core/dice"
)

func TestSetConfigsRejectsInvalidTaskWithoutSavingOtherFields(t *testing.T) {
	_, d, svc, _ := newDeadConfigService(t)
	task := d.ConfigManager.NewConfigItem("schedule", "0 8 * * *", "")
	task.Type = "task:cron"
	name := d.ConfigManager.NewConfigItem("name", "old", "")
	name.Type = "string"
	d.ConfigManager.RegisterPluginConfig("p", name, task)
	for _, value := range []interface{}{123, "not a cron", ""} {
		_, err := svc.SetConfigs(t.Context(), &SetConfigsReq{Body: JsSetConfigsReqBody{
			Name: "p", Config: map[string]interface{}{"name": "new", "schedule": value},
		}})
		var status huma.StatusError
		if !errors.As(err, &status) || status.GetStatus() != 400 {
			t.Fatalf("expected 400 for %v, got %v", value, err)
		}
		if name.Value != "old" || task.Value != "0 8 * * *" {
			t.Fatal("invalid configuration partially applied")
		}
	}
	resp, err := svc.SetConfigs(t.Context(), &SetConfigsReq{Body: JsSetConfigsReqBody{
		Name: "p", Config: map[string]interface{}{"name": "new", "schedule": "0 9 * * *"},
	}})
	if err != nil || !resp.Body.Item.Success || name.Value != "new" || task.Value != "0 9 * * *" {
		t.Fatalf("valid patch rejected: %v", err)
	}
	_, err = svc.ResetConfig(t.Context(), &ResetConfigReq{Body: JsResetConfigReqBody{Name: "p", Keys: []string{"name", "unknown"}}})
	var status huma.StatusError
	if !errors.As(err, &status) || status.GetStatus() != 400 || name.Value != "new" {
		t.Fatalf("invalid reset partially applied or not rejected: %v", err)
	}
}

func TestCheckCronExpressionAcceptsStandardCron(t *testing.T) {
	dm := &dice.DiceManager{}
	d := &dice.Dice{Parent: dm}
	dm.Dice = d

	svc := NewService(dm)
	resp, err := svc.CheckCron(t.Context(), &CheckCronReq{
		Body: JsCheckCronReqBody{Expr: "0 0 * * *"},
	})
	if err != nil {
		t.Fatalf("CheckCron returned error: %v", err)
	}
	if !resp.Body.Item.Valid {
		t.Fatal("Valid = false, want true")
	}
}

func TestCheckCronExpressionRejectsInvalidCron(t *testing.T) {
	dm := &dice.DiceManager{}
	d := &dice.Dice{Parent: dm}
	dm.Dice = d

	svc := NewService(dm)
	if _, err := svc.CheckCron(t.Context(), &CheckCronReq{
		Body: JsCheckCronReqBody{Expr: "not a cron"},
	}); err == nil {
		t.Fatal("CheckCron returned nil error for invalid expression")
	}
}

func TestCheckCronExpressionRejectsBlankCron(t *testing.T) {
	dm := &dice.DiceManager{}
	d := &dice.Dice{Parent: dm}
	dm.Dice = d

	svc := NewService(dm)
	if _, err := svc.CheckCron(t.Context(), &CheckCronReq{
		Body: JsCheckCronReqBody{Expr: "   "},
	}); err == nil {
		t.Fatal("CheckCron returned nil error for blank expression")
	}
}
