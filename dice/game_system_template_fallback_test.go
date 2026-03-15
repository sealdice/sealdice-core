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

func TestMsgContext_loadAttrValueByName_SyncsBrokenDrivingKeyToDriveAuto(t *testing.T) {
	ctx, cleanup := newTemplateFallbackTestCtx(t, "coc7", map[string]*ds.VMValue{
		"驾驶": ds.NewIntVal(40),
	})
	defer cleanup()

	value := ctx.loadAttrValueByName("驾驶")
	if value == nil || value.ToString() != "40" {
		t.Fatalf("expected 驾驶 to resolve to 汽车驾驶 with value 40, got %v", value)
	}

	attrs := lo.Must(ctx.Dice.AttrsManager.LoadByCtx(ctx))
	if _, exists := attrs.LoadX("驾驶"); exists {
		t.Fatal("expected broken 驾驶 key to be removed after sync")
	}
	if value2, exists := attrs.LoadX("汽车驾驶"); !exists || value2.ToString() != "40" {
		t.Fatal("expected 汽车驾驶 to be written back after sync")
	}
}

func TestMsgContext_loadAttrValueByName_SyncsBrokenFirearmsKeyToCanonicalSkill(t *testing.T) {
	ctx, cleanup := newTemplateFallbackTestCtx(t, "coc7", map[string]*ds.VMValue{
		"手枪": ds.NewIntVal(55),
	})
	defer cleanup()

	value := ctx.loadAttrValueByName("手枪")
	if value == nil || value.ToString() != "55" {
		t.Fatalf("expected 手枪 to resolve to 射击:手枪 with value 55, got %v", value)
	}

	attrs := lo.Must(ctx.Dice.AttrsManager.LoadByCtx(ctx))
	if _, exists := attrs.LoadX("手枪"); exists {
		t.Fatal("expected old 手枪 key to be removed after sync")
	}
	if value2, exists := attrs.LoadX("射击:手枪"); !exists || value2.ToString() != "55" {
		t.Fatal("expected 射击:手枪 to be written back after sync")
	}
}

func TestMsgContext_loadAttrValueByName_ResolvesFirearmsSpecializationForms(t *testing.T) {
	ctx, cleanup := newTemplateFallbackTestCtx(t, "coc7", map[string]*ds.VMValue{
		"射击:步霰": ds.NewIntVal(66),
	})
	defer cleanup()

	for _, name := range []string{"步枪/霰弹枪", "步霰", "射击:步霰", "射击：步霰", "射击:步枪/霰弹枪", "射击：步枪/霰弹枪"} {
		value := ctx.loadAttrValueByName(name)
		if value == nil || value.ToString() != "66" {
			t.Fatalf("expected %s to resolve to 射击:步霰 with value 66, got %v", name, value)
		}
	}
}

func TestMsgContext_loadAttrValueByName_ResolvesDrivingSpecializationForms(t *testing.T) {
	ctx, cleanup := newTemplateFallbackTestCtx(t, "coc7", map[string]*ds.VMValue{
		"驾驶:飞行器": ds.NewIntVal(33),
	})
	defer cleanup()

	for _, name := range []string{"驾驶（飞行器）", "驾驶(飞行器)", "驾驶:飞行器", "驾驶：飞行器", "驾驶飞行器", "驾驶-飞行器"} {
		value := ctx.loadAttrValueByName(name)
		if value == nil || value.ToString() != "33" {
			t.Fatalf("expected %s to resolve to 驾驶:飞行器 with value 33, got %v", name, value)
		}
	}
}
