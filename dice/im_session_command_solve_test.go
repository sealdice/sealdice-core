//lint:file-ignore testpackage Tests need access to internal helpers and types
package dice //nolint:testpackage // tests rely on unexported helpers

import (
	"testing"

	"go.uber.org/zap"
)

func newGameSystemTemplateForTest(relatedExt ...string) *GameSystemTemplate {
	return &GameSystemTemplate{
		GameSystemTemplateV2: &GameSystemTemplateV2{
			Commands: Commands{
				Set: SetConfig{
					RelatedExt: relatedExt,
				},
			},
		},
	}
}

func newRuleSelectionTestContext(system string) *MsgContext {
	d := &Dice{
		GameSystemMap: new(SyncMap[string, *GameSystemTemplate]),
	}
	d.GameSystemMap.Store("coc7", newGameSystemTemplateForTest("coc7"))
	d.GameSystemMap.Store("dnd5e", newGameSystemTemplateForTest("dnd5e"))

	return &MsgContext{
		Dice:  d,
		Group: &GroupInfo{System: system},
	}
}

func TestSelectRulePluginCandidateByGroupSystem(t *testing.T) {
	t.Run("select candidate that matches group system", func(t *testing.T) {
		ctx := newRuleSelectionTestContext("dnd5e")
		candidates := []commandSolveCandidate{
			{Ext: &ExtInfo{Name: "coc7"}},
			{Ext: &ExtInfo{Name: "dnd5e"}},
		}

		selected, ok := selectRulePluginCandidateByGroupSystem(ctx, candidates)
		if !ok {
			t.Fatalf("expected rule candidate to be selected")
		}
		if selected.Ext == nil || selected.Ext.Name != "dnd5e" {
			t.Fatalf("expected dnd5e to be selected, got %+v", selected.Ext)
		}
	})

	t.Run("no match should keep conflict", func(t *testing.T) {
		ctx := newRuleSelectionTestContext("pf2e")
		candidates := []commandSolveCandidate{
			{Ext: &ExtInfo{Name: "coc7"}},
			{Ext: &ExtInfo{Name: "dnd5e"}},
		}

		if _, ok := selectRulePluginCandidateByGroupSystem(ctx, candidates); ok {
			t.Fatalf("expected unresolved conflict when group system has no related rule plugin")
		}
	})

	t.Run("mixed ordinary and rule plugins should keep conflict", func(t *testing.T) {
		ctx := newRuleSelectionTestContext("dnd5e")
		candidates := []commandSolveCandidate{
			{Ext: &ExtInfo{Name: "dnd5e"}},
			{Ext: &ExtInfo{Name: "reply"}},
		}

		if _, ok := selectRulePluginCandidateByGroupSystem(ctx, candidates); ok {
			t.Fatalf("expected unresolved conflict for mixed plugin types")
		}
	})

	t.Run("core plus rule plugin should keep conflict", func(t *testing.T) {
		ctx := newRuleSelectionTestContext("dnd5e")
		candidates := []commandSolveCandidate{
			{},
			{Ext: &ExtInfo{Name: "dnd5e"}},
		}

		if _, ok := selectRulePluginCandidateByGroupSystem(ctx, candidates); ok {
			t.Fatalf("expected unresolved conflict when core command is involved")
		}
	})
}

func newRawTestCmd(name string, hitCounter *int) *CmdItemInfo {
	return &CmdItemInfo{
		Name: name,
		Raw:  true,
		Solve: func(_ *MsgContext, _ *Message, _ *CmdArgs) CmdExecuteResult {
			(*hitCounter)++
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
}

func newCommandSolveTestSessionAndContext(system string, activatedExt []*ExtInfo) (*IMSession, *MsgContext) {
	d := &Dice{
		Logger:        zap.NewNop().Sugar(),
		CmdMap:        CmdMapCls{},
		GameSystemMap: new(SyncMap[string, *GameSystemTemplate]),
	}
	d.GameSystemMap.Store("coc7", newGameSystemTemplateForTest("coc7"))
	d.GameSystemMap.Store("dnd5e", newGameSystemTemplateForTest("dnd5e"))

	s := &IMSession{Parent: d}
	group := &GroupInfo{
		Active: true,
		System: system,
	}
	group.SetActivatedExtList(activatedExt, nil)

	ctx := &MsgContext{
		Dice:            d,
		Session:         s,
		Group:           group,
		IsCurGroupBotOn: true,
	}
	return s, ctx
}

func TestCommandSolveRuleConflictResolution(t *testing.T) {
	t.Run("resolve to current group rule plugin", func(t *testing.T) {
		commandName := "rolltest"
		dndHit := 0
		cocHit := 0

		extCoc := &ExtInfo{
			Name: "coc7",
			CmdMap: CmdMapCls{
				commandName: newRawTestCmd(commandName, &cocHit),
			},
			DefaultSetting: &ExtDefaultSettingItem{DisabledCommand: map[string]bool{}},
		}
		extDnd := &ExtInfo{
			Name: "dnd5e",
			CmdMap: CmdMapCls{
				commandName: newRawTestCmd(commandName, &dndHit),
			},
			DefaultSetting: &ExtDefaultSettingItem{DisabledCommand: map[string]bool{}},
		}

		session, ctx := newCommandSolveTestSessionAndContext("dnd5e", []*ExtInfo{extCoc, extDnd})
		result := session.commandSolve(ctx, &Message{Sender: SenderBase{Nickname: "tester"}}, &CmdArgs{Command: commandName})

		if result.Status != commandSolveSolved {
			t.Fatalf("expected solved status, got %v", result.Status)
		}
		if dndHit != 1 || cocHit != 0 {
			t.Fatalf("expected only dnd5e command to run, got dnd=%d coc=%d", dndHit, cocHit)
		}
	})

	t.Run("unmatched group rule keeps conflict", func(t *testing.T) {
		commandName := "rolltest"
		dndHit := 0
		cocHit := 0

		extCoc := &ExtInfo{
			Name: "coc7",
			CmdMap: CmdMapCls{
				commandName: newRawTestCmd(commandName, &cocHit),
			},
			DefaultSetting: &ExtDefaultSettingItem{DisabledCommand: map[string]bool{}},
		}
		extDnd := &ExtInfo{
			Name: "dnd5e",
			CmdMap: CmdMapCls{
				commandName: newRawTestCmd(commandName, &dndHit),
			},
			DefaultSetting: &ExtDefaultSettingItem{DisabledCommand: map[string]bool{}},
		}

		session, ctx := newCommandSolveTestSessionAndContext("pf2e", []*ExtInfo{extCoc, extDnd})
		result := session.commandSolve(ctx, &Message{Sender: SenderBase{Nickname: "tester"}}, &CmdArgs{Command: commandName})

		if result.Status != commandSolveConflict {
			t.Fatalf("expected conflict status, got %v", result.Status)
		}
		if dndHit != 0 || cocHit != 0 {
			t.Fatalf("expected no command execution on conflict, got dnd=%d coc=%d", dndHit, cocHit)
		}
	})
}
