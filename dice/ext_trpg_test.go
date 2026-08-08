package dice

import (
	"reflect"
	"testing"
)

func TestBuiltinTrpgExtensionIsOptIn(t *testing.T) {
	t.Parallel()

	d := newTestDice(nil)
	RegisterBuiltinExtTrpg(d)

	ext := d.ExtFind("trpg", false)
	if ext == nil {
		t.Fatal("trpg extension was not registered")
	}
	if ext.AutoActive {
		t.Fatal("trpg extension must remain opt-in")
	}
	if ext.CmdMap["st"] == nil {
		t.Fatal("trpg extension does not provide st")
	}
}

func TestStProviderFollowsSelectedGameSystem(t *testing.T) {
	t.Parallel()

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
