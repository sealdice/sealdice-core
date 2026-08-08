package dice

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/dop251/goja"
	ds "github.com/sealdice/dicescript"
)

func TestBuiltinTrpgExtensionOwnsRuleNeutralCommands(t *testing.T) {
	t.Parallel()

	d := newTestDice(nil)
	RegisterBuiltinExtTrpg(d)

	ext := d.ExtFind("trpg", false)
	if ext == nil {
		t.Fatal("trpg extension was not registered")
	}
	if !ext.AutoActive {
		t.Fatal("trpg extension must be active by default")
	}
	for _, command := range []string{"st", "sn", "team"} {
		if ext.CmdMap[command] == nil {
			t.Fatalf("trpg extension does not provide %s", command)
		}
	}
}

func TestSnAndTeamRegistrationsMovedToTrpg(t *testing.T) {
	d, _, _, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	if cmd := d.ExtFind("log", false).CmdMap["sn"]; cmd != nil {
		t.Fatal("log extension still provides sn")
	}
	if cmd := d.ExtFind("core", false).CmdMap["team"]; cmd != nil {
		t.Fatal("core extension still provides team")
	}
	trpg := d.ExtFind("trpg", false)
	if trpg.CmdMap["sn"] == nil || trpg.CmdMap["team"] == nil {
		t.Fatal("trpg extension must provide both sn and team")
	}
}

func newTrpgStTestContext(t *testing.T, d *Dice, ep *EndPointInfo, group *GroupInfo, userID string) *MsgContext {
	t.Helper()
	ctx := &MsgContext{
		Dice:                d,
		EndPoint:            ep,
		Session:             d.ImSession,
		Group:               group,
		Player:              &GroupPlayerInfo{Name: userID, UserID: userID},
		IsPrivate:           true,
		IsCurGroupBotOn:     true,
		IsCompatibilityTest: true,
	}
	SetTempVars(ctx, userID)
	return ctx
}

func solveTrpgStForTest(t *testing.T, cmd *CmdItemInfo, ctx *MsgContext, input string) CmdExecuteResult {
	t.Helper()
	args := strings.Fields(input)
	result := cmd.Solve(ctx, &Message{
		MessageType: "private",
		Sender:      SenderBase{UserID: ctx.Player.UserID, Nickname: ctx.Player.Name},
	}, &CmdArgs{
		Command:   "st",
		Args:      args,
		CleanArgs: input,
		RawArgs:   input,
	})
	if adapter, ok := ctx.EndPoint.Adapter.(*mockPlatformAdapter); ok {
		if _, received := adapter.waitForMsg(time.Second); !received {
			t.Fatalf("timed out waiting for .st reply to %q", input)
		}
	}
	return result
}

func trpgStAttrsSnapshot(t *testing.T, ctx *MsgContext) (map[string]string, string) {
	t.Helper()
	attrs, err := ctx.Dice.AttrsManager.LoadByCtx(ctx)
	if err != nil {
		t.Fatalf("load attrs: %v", err)
	}
	values := map[string]string{}
	attrs.Range(func(key string, value *ds.VMValue) bool {
		values[key] = value.ToRepr()
		return true
	})
	return values, attrs.SheetType
}

func TestTrpgStWithoutHooksMatchesGenericCocSt(t *testing.T) {
	d, ep, _, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	tmpl := &GameSystemTemplate{
		GameSystemTemplateV2: &GameSystemTemplateV2{
			Name:    "parity",
			Version: "1.0.0",
			Alias: Alias{
				"生命值":   {"hp"},
				"生命值上限": {"hpmax"},
			},
			Commands: Commands{Set: SetConfig{RelatedExt: []string{"trpg"}}},
		},
	}
	tmpl.Init()
	d.GameSystemMap.Store("parity", tmpl)
	group := newTestGroupInfo()
	group.GroupID = "QQ-Group:trpg-parity"
	group.Active = true
	group.System = "parity"

	cocCtx := newTrpgStTestContext(t, d, ep, group, "QQ:coc-st")
	trpgCtx := newTrpgStTestContext(t, d, ep, group, "QQ:trpg-st")
	cocSt := d.ExtFind("coc7", false).CmdMap["st"]
	trpgSt := d.ExtFind("trpg", false).CmdMap["st"]

	commands := []string{
		"hp10 hpmax20 敏捷3d1",
		"hp+2 敏捷-1",
		"&伤害=2+2",
		"伤害+1",
		"del 敏捷",
	}
	for _, command := range commands {
		solveTrpgStForTest(t, cocSt, cocCtx, command)
		solveTrpgStForTest(t, trpgSt, trpgCtx, command)
	}
	for _, item := range []struct {
		ctx *MsgContext
		cmd *CmdItemInfo
	}{
		{ctx: cocCtx, cmd: cocSt},
		{ctx: trpgCtx, cmd: trpgSt},
	} {
		attrs, err := d.AttrsManager.LoadByCtx(item.ctx)
		if err != nil {
			t.Fatalf("load computed attrs: %v", err)
		}
		metadata := &ds.ValueMap{}
		metadata.Store("base", ds.NewIntVal(10))
		attrs.Store("护甲", ds.NewComputedValRaw(&ds.ComputedData{Expr: "10", Attrs: metadata}))
		solveTrpgStForTest(t, item.cmd, item.ctx, "护甲+2")
		armor, _ := attrs.LoadX("护甲")
		if armor == nil || armor.TypeId != ds.VMTypeComputedValue {
			t.Fatalf("computed armor was overwritten: %#v", armor)
		}
	}

	cocValues, cocSheet := trpgStAttrsSnapshot(t, cocCtx)
	trpgValues, trpgSheet := trpgStAttrsSnapshot(t, trpgCtx)
	if !reflect.DeepEqual(trpgValues, cocValues) {
		t.Fatalf("trpg.st attrs = %#v, coc7.st attrs = %#v", trpgValues, cocValues)
	}
	if trpgSheet != cocSheet {
		t.Fatalf("trpg.st sheet = %q, coc7.st sheet = %q", trpgSheet, cocSheet)
	}
}

func TestTrpgStHooksClampAndRollbackAtomically(t *testing.T) {
	d, ep, _, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	tmpl := &GameSystemTemplate{
		GameSystemTemplateV2: &GameSystemTemplateV2{
			Name: "hooked",
			Alias: Alias{
				"生命值": {"hp"},
				"精神值": {"mp"},
			},
			Commands: Commands{Set: SetConfig{RelatedExt: []string{"trpg", "hooker"}}},
		},
	}
	tmpl.Init()
	d.GameSystemMap.Store("hooked", tmpl)

	hooker := &ExtInfo{
		Name: "hooker",
		TrpgStHooks: &TrpgStHooks{
			Systems: []string{"hooked"},
			AfterEvaluate: func(_ *TrpgStCommandEvent, operation *TrpgStOperation) {
				if operation.Kind == "delete" && operation.Name == "生命值" {
					operation.Skip = true
				}
				if operation.Name == "生命值" && operation.ProposedValue != nil && operation.ProposedValue.IntValue > 10 {
					operation.ProposedValue = trpgStValueFromVM(ds.NewIntVal(10))
				}
				if operation.Name == "精神值" && operation.ProposedValue != nil && operation.ProposedValue.IntValue == 99 {
					operation.RejectReason = "精神值测试拒绝"
				}
			},
			OnShow: func(event *TrpgStRenderEvent) {
				if event.Name == "生命值" {
					event.Text = "HP=" + event.Value.Repr
				}
			},
			OnExport: func(event *TrpgStRenderEvent) {
				if event.Name == "精神值" {
					event.Skip = true
				}
			},
		},
	}
	d.RegisterExtension(hooker)
	group := newTestGroupInfo()
	group.GroupID = "QQ-Group:trpg-hooks"
	group.Active = true
	group.System = "hooked"
	group.SetActivatedExtList([]*ExtInfo{hooker, d.ExtFind("trpg", false)}, d)
	ctx := newTrpgStTestContext(t, d, ep, group, "QQ:hook-target")
	cmd := d.ExtFind("trpg", false).CmdMap["st"]

	solveTrpgStForTest(t, cmd, ctx, "hp20 mp5")
	attrs, err := d.AttrsManager.LoadByCtx(ctx)
	if err != nil {
		t.Fatalf("load attrs: %v", err)
	}
	hp, _ := attrs.LoadX("生命值")
	if value, _ := hp.ReadInt(); value != 10 {
		t.Fatalf("clamped hp = %d, want 10", value)
	}
	mp, _ := attrs.LoadX("精神值")
	if value, _ := mp.ReadInt(); value != 5 {
		t.Fatalf("mp = %d, want 5", value)
	}

	runner := newTrpgStHookRunner(ctx, "hooked")
	event := &TrpgStCommandEvent{Actor: ctx, Target: ctx, System: "hooked"}
	shown, _, err := trpgStGetItemsForShow(ctx, tmpl, nil, 0, runner, event)
	if err != nil {
		t.Fatalf("render show items: %v", err)
	}
	if !slices.Contains(shown, "HP=10") {
		t.Fatalf("hooked show items = %v, want HP=10", shown)
	}
	exported, err := trpgStGetItemsForExport(ctx, tmpl, runner, event)
	if err != nil {
		t.Fatalf("render export items: %v", err)
	}
	for _, item := range exported {
		if strings.HasPrefix(item, "精神值:") {
			t.Fatalf("hooked export items = %v, 精神值 should be skipped", exported)
		}
	}

	solveTrpgStForTest(t, cmd, ctx, "del hp")
	hp, _ = attrs.LoadX("生命值")
	if value, _ := hp.ReadInt(); value != 10 {
		t.Fatalf("hp after skipped delete = %d, want 10", value)
	}

	solveTrpgStForTest(t, cmd, ctx, "hp7 mp99")
	hp, _ = attrs.LoadX("生命值")
	if value, _ := hp.ReadInt(); value != 10 {
		t.Fatalf("hp after rejected batch = %d, want rollback to 10", value)
	}
	mp, _ = attrs.LoadX("精神值")
	if value, _ := mp.ReadInt(); value != 5 {
		t.Fatalf("mp after rejected batch = %d, want rollback to 5", value)
	}
}

func TestTrpgStHooksCanBeRegisteredFromJavaScript(t *testing.T) {
	t.Parallel()

	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("jsbind", true))
	ext := &ExtInfo{Name: "js-hook"}
	if err := vm.Set("ext", ext); err != nil {
		t.Fatalf("set ext: %v", err)
	}
	if err := vm.Set("registerHooks", registerTrpgStHooks); err != nil {
		t.Fatalf("set registerHooks: %v", err)
	}
	if err := vm.Set("newIntValue", func(value int64) *TrpgStValue {
		return trpgStValueFromVM(ds.NewIntVal(ds.IntType(value)))
	}); err != nil {
		t.Fatalf("set newIntValue: %v", err)
	}
	if err := vm.Set("newOperation", func(kind, name string) *TrpgStOperation {
		return &TrpgStOperation{Kind: kind, Name: name}
	}); err != nil {
		t.Fatalf("set newOperation: %v", err)
	}

	_, err := vm.RunString(`
registerHooks(ext, {
  systems: ["fu"],
  afterParse: function (event) {
    const operation = newOperation("set", "幸运");
    operation.operand = newIntValue(7);
    event.operations = [operation].concat(event.operations);
  },
  afterEvaluate: function (_event, operation) {
    if (operation.name === "生命值" && operation.proposedValue.intValue > 10) {
      operation.proposedValue = newIntValue(10);
    }
  }
});`)
	if err != nil {
		t.Fatalf("register JS hooks: %v", err)
	}
	if ext.TrpgStHooks == nil || !ext.TrpgStHooks.appliesTo("fu") {
		t.Fatalf("JS hook registration was not stored: %#v", ext.TrpgStHooks)
	}
	event := &TrpgStCommandEvent{System: "fu", Operations: []*TrpgStOperation{{Kind: "set", Name: "生命值"}}}
	ext.TrpgStHooks.AfterParse(event)
	if len(event.Operations) != 2 || event.Operations[0].Name != "幸运" || event.Operations[0].Operand.IntValue != 7 {
		t.Fatalf("JS hook operations = %#v, want prepended 幸运=7", event.Operations)
	}
	operation := &TrpgStOperation{
		Name:          "生命值",
		ProposedValue: trpgStValueFromVM(ds.NewIntVal(20)),
	}
	ext.TrpgStHooks.AfterEvaluate(&TrpgStCommandEvent{System: "fu"}, operation)
	if operation.ProposedValue == nil || operation.ProposedValue.IntValue != 10 {
		t.Fatalf("JS hook proposed value = %#v, want int 10", operation.ProposedValue)
	}
}

func TestStProviderFollowsSelectedGameSystem(t *testing.T) {
	exts := []*ExtInfo{
		{Name: "coc7", CmdMap: CmdMapCls{"st": {Name: "coc7-st"}}},
		{Name: "dnd5e", CmdMap: CmdMapCls{"st": {Name: "dnd5e-st"}}},
		{Name: "trpg", CmdMap: CmdMapCls{"st": {Name: "trpg-st"}}},
		{Name: "some-plugin"},
	}
	d := newTestDice(exts)
	group := newTestGroupInfo()
	group.SetActivatedExtList(exts, d)

	tests := []struct {
		name       string
		system     string
		relatedExt []string
		want       []string
	}{
		{
			name:       "legacy coc template keeps coc st",
			system:     "legacy-coc",
			relatedExt: []string{"coc7", "some-plugin"},
			want:       []string{"coc7", "some-plugin", "dnd5e", "trpg"},
		},
		{
			name:       "dnd template keeps dnd st",
			system:     "dnd",
			relatedExt: []string{"dnd5e"},
			want:       []string{"dnd5e", "coc7", "trpg", "some-plugin"},
		},
		{
			name:       "new template selects trpg st",
			system:     "new-rule",
			relatedExt: []string{"trpg", "some-plugin"},
			want:       []string{"trpg", "some-plugin", "coc7", "dnd5e"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addTestGameSystem(d, tt.system, tt.relatedExt...)
			group.System = tt.system
			got := extListToNames(commandExtensionOrder(group, d))
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("command order = %v, want %v", got, tt.want)
			}
			if gotItem := commandExtensionOrder(group, d)[0].GetCmdMap()["st"]; gotItem == nil {
				t.Fatalf("first extension %s does not provide st", tt.want[0])
			}
		})
	}
}
