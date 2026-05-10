//nolint:testpackage
package dice

import (
	"testing"

	"github.com/samber/lo"
	ds "github.com/sealdice/dicescript"
)

func newPlayerObjectTestCtx(t *testing.T, sheetType string, values map[string]*ds.VMValue) (*MsgContext, func()) {
	t.Helper()

	d, ep, _, cleanup := newExecuteNewTestDice(t)

	groupID := "QQ-Group:2233"
	userID := "QQ:4455"
	msg := newGroupMsg(groupID, userID, ".r 1")
	ctx := CreateTempCtx(ep, msg)
	group, player := GetPlayerInfoBySender(ctx, msg)
	group.System = "coc7"
	player.Name = "Tester"
	ctx.Group = group
	ctx.Player = player
	ctx.IsCompatibilityTest = true
	ctx.SystemTemplate = group.GetCharTemplate(d)

	attrs := &AttributesItem{
		ID:        groupID + "-" + userID,
		valueMap:  &ds.ValueMap{},
		SheetType: sheetType,
	}
	for key, value := range values {
		attrs.Store(key, value)
	}
	d.AttrsManager.m.Store(attrs.ID, attrs)

	return ctx, cleanup
}

func TestPlayerObject_IsInjectedAsPlayerAndActor(t *testing.T) {
	ctx, cleanup := newPlayerObjectTestCtx(t, "coc7", nil)
	defer cleanup()

	ctx.CreateVmIfNotExists()

	playerVal, ok := ctx.vm.Attrs.Load("player")
	if !ok || playerVal == nil || playerVal.TypeId != ds.VMTypeNativeObject {
		t.Fatalf("expected native object player, got %v (exists=%v)", playerVal, ok)
	}
	actorVal, ok := ctx.vm.Attrs.Load("actor")
	if !ok || actorVal == nil || actorVal.TypeId != ds.VMTypeNativeObject {
		t.Fatalf("expected native object actor, got %v (exists=%v)", actorVal, ok)
	}

	playerObj, _ := playerVal.ReadNativeObjectData()
	actorObj, _ := actorVal.ReadNativeObjectData()
	if playerObj != actorObj {
		t.Fatal("expected player and actor to point to the same native object")
	}
}

func TestPlayerObject_WriteAndReadCanonicalAttr(t *testing.T) {
	ctx, cleanup := newPlayerObjectTestCtx(t, "coc7", nil)
	defer cleanup()

	result := ctx.Eval("player.力量 = 70; player.力量", nil)
	if result.vm.Error != nil {
		t.Fatalf("Eval returned error: %v", result.vm.Error)
	}
	if got := result.ToString(); got != "70" {
		t.Fatalf("expected 70, got %s", got)
	}

	attrs := lo.Must(ctx.Dice.AttrsManager.LoadByCtx(ctx))
	if v, exists := attrs.LoadX("力量"); !exists || v.ToString() != "70" {
		t.Fatalf("expected current attrs to store 力量=70, got %v (exists=%v)", v, exists)
	}
}

func TestPlayerObject_AliasWriteStoresCanonicalKey(t *testing.T) {
	ctx, cleanup := newPlayerObjectTestCtx(t, "coc7", nil)
	defer cleanup()

	result := ctx.Eval("player.DEX = 60; player.敏捷", nil)
	if result.vm.Error != nil {
		t.Fatalf("Eval returned error: %v", result.vm.Error)
	}
	if got := result.ToString(); got != "60" {
		t.Fatalf("expected 60, got %s", got)
	}

	attrs := lo.Must(ctx.Dice.AttrsManager.LoadByCtx(ctx))
	if v, exists := attrs.LoadX("敏捷"); !exists || v.ToString() != "60" {
		t.Fatalf("expected canonical key 敏捷=60, got %v (exists=%v)", v, exists)
	}
	if _, exists := attrs.LoadX("DEX"); exists {
		t.Fatal("did not expect alias key DEX to be written")
	}
}

func TestPlayerObject_AliasReadReturnsStoredCanonicalValue(t *testing.T) {
	ctx, cleanup := newPlayerObjectTestCtx(t, "coc7", map[string]*ds.VMValue{
		"敏捷": ds.NewIntVal(70),
	})
	defer cleanup()

	result := ctx.Eval("player.DEX", nil)
	if result.vm.Error != nil {
		t.Fatalf("Eval returned error: %v", result.vm.Error)
	}
	if got := result.ToString(); got != "70" {
		t.Fatalf("expected 70, got %s", got)
	}
}

func TestPlayerObject_ItemGetAndSetUseAliasResolution(t *testing.T) {
	ctx, cleanup := newPlayerObjectTestCtx(t, "coc7", nil)
	defer cleanup()

	result := ctx.Eval("player['DEX'] = 55; player['敏捷']", nil)
	if result.vm.Error != nil {
		t.Fatalf("Eval returned error: %v", result.vm.Error)
	}
	if got := result.ToString(); got != "55" {
		t.Fatalf("expected 55, got %s", got)
	}

	attrs := lo.Must(ctx.Dice.AttrsManager.LoadByCtx(ctx))
	if v, exists := attrs.LoadX("敏捷"); !exists || v.ToString() != "55" {
		t.Fatalf("expected canonical key 敏捷=55, got %v (exists=%v)", v, exists)
	}
}

func TestPlayerObject_UsesTemplateDefaultWhenAttrMissing(t *testing.T) {
	ctx, cleanup := newPlayerObjectTestCtx(t, "coc7", nil)
	defer cleanup()

	result := ctx.Eval("player.外语", nil)
	if result.vm.Error != nil {
		t.Fatalf("Eval returned error: %v", result.vm.Error)
	}
	if got := result.ToString(); got != "1" {
		t.Fatalf("expected default value 1, got %s", got)
	}
}

func TestPlayerObject_ReturnsNullWhenAttrTrulyMissing(t *testing.T) {
	ctx, cleanup := newPlayerObjectTestCtx(t, "coc7", nil)
	defer cleanup()

	result := ctx.Eval("player.不存在属性", nil)
	if result.vm.Error != nil {
		t.Fatalf("Eval returned error: %v", result.vm.Error)
	}
	if result.TypeId != ds.VMTypeInt || result.ToString() != "0" {
		t.Fatalf("expected missing value to be 0, got type %v with value %s", result.TypeId, result.ToString())
	}
}

func TestPlayerObject_DirIncludesMethodsAndVisibleKeys(t *testing.T) {
	ctx, cleanup := newPlayerObjectTestCtx(t, "coc7", map[string]*ds.VMValue{
		"力量": ds.NewIntVal(80),
	})
	defer cleanup()

	result := ctx.Eval("dir(player)", nil)
	if result.vm.Error != nil {
		t.Fatalf("Eval returned error: %v", result.vm.Error)
	}

	dirArr, ok := result.ReadArray()
	if !ok {
		t.Fatalf("expected array result, got %s", result.GetTypeName())
	}

	items := map[string]bool{}
	for _, item := range dirArr.List {
		items[item.ToString()] = true
	}
	for _, key := range []string{"keys", "values", "items", "len", "has", "力量", "外语"} {
		if !items[key] {
			t.Fatalf("expected dir(player) to include %q", key)
		}
	}
}

func TestPlayerObject_DirWithNilContextIsSafe(t *testing.T) {
	object := newActorNativeObject(nil, "player")
	objectData, ok := object.ReadNativeObjectData()
	if !ok {
		t.Fatalf("expected native object, got %s", object.GetTypeName())
	}

	dir := objectData.DirFunc(ds.NewVM())
	items := map[string]bool{}
	for _, item := range dir {
		items[item.ToString()] = true
	}

	for _, key := range []string{"has", "items", "keys", "len", "values"} {
		if !items[key] {
			t.Fatalf("expected dir(player) to include %q with nil context", key)
		}
	}
}

func TestPlayerObject_ProvidesDictStyleMethods(t *testing.T) {
	ctx, cleanup := newPlayerObjectTestCtx(t, "coc7", map[string]*ds.VMValue{
		"力量": ds.NewIntVal(80),
	})
	defer cleanup()

	result := ctx.Eval("player.keys().len()", nil)
	if result.vm.Error != nil {
		t.Fatalf("Eval returned error: %v", result.vm.Error)
	}
	if result.TypeId != ds.VMTypeInt {
		t.Fatalf("expected integer len result, got %s", result.GetTypeName())
	}
	if result.MustReadInt() <= 0 {
		t.Fatal("expected player.keys().len() to be greater than 0")
	}
}

func TestPlayerObject_KeysDoesNotIncludeInjectedMethods(t *testing.T) {
	ctx, cleanup := newPlayerObjectTestCtx(t, "coc7", map[string]*ds.VMValue{
		"力量": ds.NewIntVal(80),
	})
	defer cleanup()

	result := ctx.Eval("player.keys()", nil)
	if result.vm.Error != nil {
		t.Fatalf("Eval returned error: %v", result.vm.Error)
	}

	keys, ok := result.ReadArray()
	if !ok {
		t.Fatalf("expected array result, got %s", result.GetTypeName())
	}

	items := map[string]bool{}
	for _, item := range keys.List {
		items[item.ToString()] = true
	}

	if !items["力量"] || !items["外语"] {
		t.Fatalf("expected keys() to include visible attrs, got %v", items)
	}
	for _, key := range []string{"keys", "values", "items", "len", "has"} {
		if items[key] {
			t.Fatalf("did not expect keys() to include injected method %q", key)
		}
	}
}

func TestPlayerObject_HasSupportsCanonicalAliasAndTemplateDefault(t *testing.T) {
	ctx, cleanup := newPlayerObjectTestCtx(t, "coc7", map[string]*ds.VMValue{
		"敏捷": ds.NewIntVal(80),
	})
	defer cleanup()

	result := ctx.Eval("[player.has('敏捷'), player.has('DEX'), player.has('外语')]", nil)
	if result.vm.Error != nil {
		t.Fatalf("Eval returned error: %v", result.vm.Error)
	}

	arr, ok := result.ReadArray()
	if !ok {
		t.Fatalf("expected array result, got %s", result.GetTypeName())
	}
	if len(arr.List) != 3 {
		t.Fatalf("expected 3 results, got %d", len(arr.List))
	}
	for idx, want := range []string{"1", "1", "1"} {
		if got := arr.List[idx].ToString(); got != want {
			t.Fatalf("expected result[%d] = %s, got %s", idx, want, got)
		}
	}
}

func TestPlayerObject_HasReturnsFalseForTrulyMissingAttr(t *testing.T) {
	ctx, cleanup := newPlayerObjectTestCtx(t, "coc7", nil)
	defer cleanup()

	result := ctx.Eval("player.has('不存在属性')", nil)
	if result.vm.Error != nil {
		t.Fatalf("Eval returned error: %v", result.vm.Error)
	}
	if got := result.ToString(); got != "0" {
		t.Fatalf("expected has() to return 0 for missing attr, got %s", got)
	}
}

func TestPlayerObject_DoesNotChangeTopLevelMissingValueBehavior(t *testing.T) {
	ctx, cleanup := newPlayerObjectTestCtx(t, "coc7", nil)
	defer cleanup()

	result := ctx.Eval("不存在属性", nil)
	if result.vm.Error != nil {
		t.Fatalf("Eval returned error: %v", result.vm.Error)
	}
	if got := result.ToString(); got != "0" {
		t.Fatalf("expected top-level missing value to remain 0, got %s", got)
	}
}
