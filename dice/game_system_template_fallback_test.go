package dice

import (
	"testing"

	"github.com/samber/lo"
	ds "github.com/sealdice/dicescript"
)

func newTemplateFallbackTestCtx(t *testing.T, sheetType string, values map[string]*ds.VMValue) (*MsgContext, func()) {
	t.Helper()

	d, _, _, cleanup := newExecuteNewTestDice(t)
	ctx := &MsgContext{
		Dice:                d,
		Session:             d.ImSession,
		IsCompatibilityTest: true,
		Group: &GroupInfo{
			GroupID: "QQ-Group:114514",
			System:  "coc7",
		},
		Player: &GroupPlayerInfo{
			UserID: "QQ:1919810",
			Name:   "Tester",
		},
	}
	ctx.SystemTemplate = ctx.Group.GetCharTemplate(d)

	attrs := &AttributesItem{
		ID:        ctx.Group.GroupID + "-" + ctx.Player.UserID,
		valueMap:  &ds.ValueMap{},
		SheetType: sheetType,
	}
	for key, value := range values {
		attrs.Store(key, value)
	}
	d.AttrsManager.m.Store(attrs.ID, attrs)

	return ctx, cleanup
}

func TestGameSystemTemplate_GetShowValueAs_SyncsOldKeyOnSameSheetType(t *testing.T) {
	ctx, cleanup := newTemplateFallbackTestCtx(t, "coc7", map[string]*ds.VMValue{
		"db": ds.NewIntVal(42),
	})
	defer cleanup()

	value, err := ctx.SystemTemplate.GetShowValueAs(ctx, "DB")
	if err != nil {
		t.Fatalf("GetShowValueAs returned error: %v", err)
	}
	if got := value.ToString(); got != "42" {
		t.Fatalf("expected stored alias value 42, got %s", got)
	}

	attrs := lo.Must(ctx.Dice.AttrsManager.LoadByCtx(ctx))
	if version, exists := attrs.LoadX(attrsTemplateVersionKey); !exists || version.ToString() != "1.1.0" {
		t.Fatalf("expected %s to be set to 1.1.0 after sync", attrsTemplateVersionKey)
	}
}

func TestMsgContext_loadAttrValueByName_SyncsOldKeyOnSameSheetType(t *testing.T) {
	ctx, cleanup := newTemplateFallbackTestCtx(t, "coc7", map[string]*ds.VMValue{
		"语言": ds.NewIntVal(70),
	})
	defer cleanup()

	value := ctx.loadAttrValueByName("外语")
	if value == nil {
		t.Fatal("expected a value, got nil")
	}
	if got := value.ToString(); got != "70" {
		t.Fatalf("expected stored alias value 70, got %s", got)
	}
}

func TestAttrsManager_LoadByCtx_DoesNotSyncBeforeTemplateRead(t *testing.T) {
	ctx, cleanup := newTemplateFallbackTestCtx(t, "coc7", map[string]*ds.VMValue{
		"语言": ds.NewIntVal(70),
	})
	defer cleanup()

	attrs := lo.Must(ctx.Dice.AttrsManager.LoadByCtx(ctx))
	if _, exists := attrs.LoadX("语言"); !exists {
		t.Fatal("expected old key to remain before template-based read")
	}
	if _, exists := attrs.LoadX("外语"); exists {
		t.Fatal("did not expect new key to exist before template-based read")
	}
	if _, exists := attrs.LoadX(attrsTemplateVersionKey); exists {
		t.Fatalf("did not expect %s before template-based read", attrsTemplateVersionKey)
	}

	value := ctx.loadAttrValueByName("外语")
	if value == nil || value.ToString() != "70" {
		t.Fatalf("expected synced value 70 after template-based read, got %v", value)
	}
	if _, exists := attrs.LoadX("语言"); exists {
		t.Fatal("expected old key to be removed after template-based read")
	}
	if value2, exists := attrs.LoadX("外语"); !exists || value2.ToString() != "70" {
		t.Fatal("expected new key to be written back after template-based read")
	}
	if version, exists := attrs.LoadX(attrsTemplateVersionKey); !exists || version.ToString() != "1.1.0" {
		t.Fatalf("expected %s to be set to 1.1.0 after template-based read", attrsTemplateVersionKey)
	}
}

func TestGameSystemTemplate_GetRealValue_DoesNotFallbackAcrossSheetType(t *testing.T) {
	ctx, cleanup := newTemplateFallbackTestCtx(t, "dnd5e", map[string]*ds.VMValue{
		"语言": ds.NewIntVal(70),
	})
	defer cleanup()

	value, err := ctx.SystemTemplate.GetRealValue(ctx, "外语")
	if err != nil {
		t.Fatalf("GetRealValue returned error: %v", err)
	}
	if got := value.ToString(); got != "1" {
		t.Fatalf("expected coc7 default value 1 without cross-sheet fallback, got %s", got)
	}

	attrs := lo.Must(ctx.Dice.AttrsManager.LoadByCtx(ctx))
	if _, exists := attrs.LoadX(attrsTemplateVersionKey); exists {
		t.Fatalf("did not expect %s to be written for cross-sheet reads", attrsTemplateVersionKey)
	}
}

func TestGameSystemTemplate_GetRealValue_DoesNotResyncWhenVersionIsCurrent(t *testing.T) {
	ctx, cleanup := newTemplateFallbackTestCtx(t, "coc7", map[string]*ds.VMValue{
		attrsTemplateVersionKey: ds.NewStrVal("1.1.0"),
		"语言":                    ds.NewIntVal(70),
	})
	defer cleanup()

	value, err := ctx.SystemTemplate.GetRealValue(ctx, "外语")
	if err != nil {
		t.Fatalf("GetRealValue returned error: %v", err)
	}
	if got := value.ToString(); got != "1" {
		t.Fatalf("expected default value 1 when card version is already current, got %s", got)
	}

	attrs := lo.Must(ctx.Dice.AttrsManager.LoadByCtx(ctx))
	if _, exists := attrs.LoadX("语言"); !exists {
		t.Fatal("expected old key to remain when version is already current")
	}
	if _, exists := attrs.LoadX("外语"); exists {
		t.Fatal("did not expect new key when version is already current")
	}
}
