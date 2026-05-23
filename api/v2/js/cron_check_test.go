package js_test

import (
	"testing"

	. "sealdice-core/api/v2/js"
	"sealdice-core/dice"
)

func TestCheckCronExpressionAcceptsStandardCron(t *testing.T) {
	dm := &dice.DiceManager{}
	d := &dice.Dice{Parent: dm}
	dm.Dice = []*dice.Dice{d}

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
	dm.Dice = []*dice.Dice{d}

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
	dm.Dice = []*dice.Dice{d}

	svc := NewService(dm)
	if _, err := svc.CheckCron(t.Context(), &CheckCronReq{
		Body: JsCheckCronReqBody{Expr: "   "},
	}); err == nil {
		t.Fatal("CheckCron returned nil error for blank expression")
	}
}
