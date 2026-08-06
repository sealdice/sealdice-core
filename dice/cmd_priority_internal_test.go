package dice

import (
	"encoding/json"
	"reflect"
	"testing"
)

func addTestGameSystem(d *Dice, name string, relatedExt ...string) {
	tmpl := &GameSystemTemplate{
		GameSystemTemplateV2: &GameSystemTemplateV2{
			Name: name,
			Commands: Commands{
				Set: SetConfig{RelatedExt: relatedExt},
			},
		},
	}
	tmpl.Init()
	if d.GameSystemMap == nil {
		d.GameSystemMap = new(SyncMap[string, *GameSystemTemplate])
	}
	d.GameSystemMap.Store(name, tmpl)
}

func TestCommandExtensionOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		system     string
		relatedExt []string
		exts       []*ExtInfo
		activated  []string
		want       []string
	}{
		{
			name:      "no selected system preserves activation order",
			exts:      []*ExtInfo{{Name: "cpr"}, {Name: "coc7"}, {Name: "dnd5e"}},
			activated: []string{"cpr", "coc7", "dnd5e"},
			want:      []string{"cpr", "coc7", "dnd5e"},
		},
		{
			name:       "coc promotes only coc extension",
			system:     "coc7",
			relatedExt: []string{"coc7"},
			exts:       []*ExtInfo{{Name: "cpr"}, {Name: "dnd5e"}, {Name: "coc7"}},
			activated:  []string{"cpr", "dnd5e", "coc7"},
			want:       []string{"coc7", "cpr", "dnd5e"},
		},
		{
			name:       "cpr tier preserves member order",
			system:     "cpr",
			relatedExt: []string{"coc7", "cpr"},
			exts:       []*ExtInfo{{Name: "dnd5e"}, {Name: "coc7"}, {Name: "cpr"}},
			activated:  []string{"cpr", "dnd5e", "coc7"},
			want:       []string{"cpr", "coc7", "dnd5e"},
		},
		{
			name:       "active-with chain joins system tier recursively",
			system:     "base",
			relatedExt: []string{"base"},
			exts: []*ExtInfo{
				{Name: "other"},
				{Name: "base"},
				{Name: "level1", ActiveWith: []string{"base"}},
				{Name: "level2", ActiveWith: []string{"level1"}},
			},
			activated: []string{"other", "level2", "base", "level1"},
			want:      []string{"level2", "base", "level1", "other"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := newTestDice(tt.exts)
			if tt.system != "" {
				addTestGameSystem(d, tt.system, tt.relatedExt...)
			}
			group := newTestGroupInfo()
			activated := make([]*ExtInfo, 0, len(tt.activated))
			for _, name := range tt.activated {
				activated = append(activated, d.ExtFind(name, false))
			}
			group.SetActivatedExtList(activated, d)
			group.System = tt.system

			got := extListToNames(commandExtensionOrder(group, d))
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("commandExtensionOrder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommandPriorityFollowsSelectedGameSystem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		system        string
		relatedExt    []string
		activated     []string
		command       string
		wantExtension string
	}{
		{
			name:          "coc ra wins over cpr",
			system:        "coc7",
			relatedExt:    []string{"coc7"},
			activated:     []string{"cpr", "coc7", "dnd5e"},
			command:       "ra",
			wantExtension: "coc7",
		},
		{
			name:          "cpr ra wins inside cpr tier",
			system:        "cpr",
			relatedExt:    []string{"coc7", "cpr"},
			activated:     []string{"cpr", "coc7", "dnd5e"},
			command:       "ra",
			wantExtension: "cpr",
		},
		{
			name:          "dnd st wins over generic coc st",
			system:        "dnd5e",
			relatedExt:    []string{"dnd5e"},
			activated:     []string{"coc7", "dnd5e", "cpr"},
			command:       "st",
			wantExtension: "dnd5e",
		},
		{
			name:          "cpr falls back to generic coc st in its tier",
			system:        "cpr",
			relatedExt:    []string{"coc7", "cpr"},
			activated:     []string{"dnd5e", "cpr", "coc7"},
			command:       "st",
			wantExtension: "coc7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			calls := map[string]int{}
			exts := make([]*ExtInfo, 0, 3)
			for _, name := range []string{"coc7", "dnd5e", "cpr"} {
				extName := name
				cmdMap := CmdMapCls{}
				if tt.command != "st" || name != "cpr" {
					cmdMap[tt.command] = &CmdItemInfo{
						Name: tt.command,
						Solve: func(_ *MsgContext, _ *Message, _ *CmdArgs) CmdExecuteResult {
							calls[extName]++
							return CmdExecuteResult{Matched: true, Solved: true}
						},
					}
				}
				exts = append(exts, &ExtInfo{
					Name:           name,
					CmdMap:         cmdMap,
					DefaultSetting: &ExtDefaultSettingItem{DisabledCommand: map[string]bool{}},
				})
			}

			d := newTestDice(exts)
			d.CmdMap = CmdMapCls{}
			addTestGameSystem(d, tt.system, tt.relatedExt...)
			group := newTestGroupInfo()
			activated := make([]*ExtInfo, 0, len(tt.activated))
			for _, name := range tt.activated {
				activated = append(activated, d.ExtFind(name, false))
			}
			group.SetActivatedExtList(activated, d)
			group.Active = true
			group.System = tt.system
			session := &IMSession{Parent: d}
			ctx := &MsgContext{
				Dice:      d,
				Session:   session,
				Group:     group,
				IsPrivate: true,
			}

			if !session.commandSolve(ctx, &Message{}, &CmdArgs{Command: tt.command}) {
				t.Fatal("expected command to be solved")
			}
			if calls[tt.wantExtension] != 1 {
				t.Fatalf("calls = %v, want only %s", calls, tt.wantExtension)
			}
			for name, count := range calls {
				if name != tt.wantExtension && count != 0 {
					t.Fatalf("calls = %v, unexpected execution by %s", calls, name)
				}
			}
		})
	}
}

func TestCommandPrioritySurvivesGroupReload(t *testing.T) {
	t.Parallel()

	coc := &ExtInfo{Name: "coc7"}
	cpr := &ExtInfo{Name: "cpr"}
	d := newTestDice([]*ExtInfo{coc, cpr})
	addTestGameSystem(d, "cpr", "coc7", "cpr")

	group := newTestGroupInfo()
	group.System = "cpr"
	group.SetActivatedExtList([]*ExtInfo{cpr, coc}, d)
	data, err := json.Marshal(group)
	if err != nil {
		t.Fatalf("marshal group: %v", err)
	}

	reloaded := &GroupInfo{}
	if err := json.Unmarshal(data, reloaded); err != nil {
		t.Fatalf("unmarshal group: %v", err)
	}
	got := extListToNames(commandExtensionOrder(reloaded, d))
	want := []string{"cpr", "coc7"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("command order after reload = %v, want %v", got, want)
	}
}

func TestCommandPrioritySurvivesJSWrapperRefresh(t *testing.T) {
	t.Parallel()

	coc := &ExtInfo{Name: "coc7"}
	cprWrapper := &ExtInfo{
		Name:           "cpr",
		TargetName:     "cpr",
		IsWrapper:      true,
		DefaultSetting: &ExtDefaultSettingItem{DisabledCommand: map[string]bool{}},
	}
	d := newTestDice([]*ExtInfo{coc, cprWrapper})
	d.JsExtRegistry = new(SyncMap[string, *ExtInfo])
	d.JsExtRegistry.Store("cpr", &ExtInfo{Name: "cpr", CmdMap: CmdMapCls{"ra": {Name: "old"}}})
	addTestGameSystem(d, "cpr", "coc7", "cpr")
	group := newTestGroupInfo()
	group.System = "cpr"
	group.SetActivatedExtList([]*ExtInfo{cprWrapper, coc}, d)

	d.JsExtRegistry.Store("cpr", &ExtInfo{Name: "cpr", CmdMap: CmdMapCls{"ra": {Name: "reloaded"}}})
	got := extListToNames(commandExtensionOrder(group, d))
	want := []string{"cpr", "coc7"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("command order after JS wrapper refresh = %v, want %v", got, want)
	}
	if gotItem := cprWrapper.GetCmdMap()["ra"]; gotItem == nil || gotItem.Name != "reloaded" {
		t.Fatalf("wrapper did not expose reloaded command: %#v", gotItem)
	}
}

func TestBuiltinGameSystemsRelateTheirRuleExtensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		file string
		want []string
	}{
		{file: "coc7.yaml", want: []string{"coc7"}},
		{file: "dnd5e.yaml", want: []string{"dnd5e"}},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			t.Parallel()

			tmpl, err := loadBuiltinTemplate(tt.file)
			if err != nil {
				t.Fatalf("load built-in template: %v", err)
			}
			if !reflect.DeepEqual(tmpl.SetConfig.RelatedExt, tt.want) {
				t.Fatalf("related extensions = %v, want %v", tmpl.SetConfig.RelatedExt, tt.want)
			}
		})
	}
}

func TestCommandReceivedHooksStillUseAllActivatedExtensions(t *testing.T) {
	t.Parallel()

	hookCalls := map[string]int{}
	newExt := func(name string) *ExtInfo {
		return &ExtInfo{
			Name: name,
			OnCommandReceived: func(_ *MsgContext, _ *Message, _ *CmdArgs) {
				hookCalls[name]++
			},
			DefaultSetting: &ExtDefaultSettingItem{DisabledCommand: map[string]bool{}},
		}
	}
	coc := newExt("coc7")
	coc.CmdMap = CmdMapCls{
		"st": {
			Name: "st",
			Solve: func(_ *MsgContext, _ *Message, _ *CmdArgs) CmdExecuteResult {
				return CmdExecuteResult{Matched: true, Solved: true}
			},
		},
	}
	lowPriority := newExt("unrelated")
	d := newTestDice([]*ExtInfo{coc, lowPriority})
	d.CmdMap = CmdMapCls{}
	addTestGameSystem(d, "fu", "coc7")
	group := newTestGroupInfo()
	group.Active = true
	group.System = "fu"
	group.SetActivatedExtList([]*ExtInfo{lowPriority, coc}, d)
	session := &IMSession{Parent: d}
	ctx := &MsgContext{Dice: d, Session: session, Group: group, IsPrivate: true}

	if !session.commandSolve(ctx, &Message{}, &CmdArgs{Command: "st"}) {
		t.Fatal("expected st command to be solved")
	}
	if hookCalls["coc7"] != 1 || hookCalls["unrelated"] != 1 {
		t.Fatalf("OnCommandReceived calls = %v, want every activated extension once", hookCalls)
	}
}
