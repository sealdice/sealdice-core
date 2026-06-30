package dice //nolint:testpackage

import (
	"encoding/json"
	"testing"

	"go.uber.org/zap"
)

func TestGroupInfoMarshalJSONStoresActivatedExtAsNames(t *testing.T) {
	d := &Dice{
		Logger:      zap.NewNop().Sugar(),
		ExtRegistry: new(SyncMap[string, *ExtInfo]),
	}
	jsWrapper := &ExtInfo{
		Name:       "js-ext",
		IsWrapper:  true,
		TargetName: "js-ext",
		IsJsExt:    true,
		dice:       d,
	}
	builtin := &ExtInfo{
		Name: "builtin",
		dice: d,
	}
	d.ExtList = []*ExtInfo{jsWrapper, builtin}
	d.ExtRegistry.Store("js-ext", jsWrapper)
	d.ExtRegistry.Store("builtin", builtin)

	group := newTestGroupInfo()
	group.SetActivatedExtList([]*ExtInfo{jsWrapper, builtin}, d)

	data, err := json.Marshal(group)
	if err != nil {
		t.Fatalf("marshal group: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal raw json: %v", err)
	}

	rawList, ok := decoded["activatedExtList"].([]any)
	if !ok {
		t.Fatalf("activatedExtList should be a JSON array, got %T", decoded["activatedExtList"])
	}
	if len(rawList) != 2 {
		t.Fatalf("expected 2 activated ext names, got %d", len(rawList))
	}
	if rawList[0] != "js-ext" || rawList[1] != "builtin" {
		t.Fatalf("expected activated ext names [js-ext builtin], got %#v", rawList)
	}
}

func TestGroupInfoUnmarshalJSONSupportsLegacyObjectEntries(t *testing.T) {
	d := &Dice{
		Logger:        zap.NewNop().Sugar(),
		ExtRegistry:   new(SyncMap[string, *ExtInfo]),
		JsExtRegistry: new(SyncMap[string, *ExtInfo]),
	}
	jsWrapper := &ExtInfo{
		Name:       "js-ext",
		IsWrapper:  true,
		TargetName: "js-ext",
		IsJsExt:    true,
		dice:       d,
	}
	jsReal := &ExtInfo{
		Name:    "js-ext",
		IsJsExt: true,
		dice:    d,
	}
	builtin := &ExtInfo{
		Name: "builtin",
		dice: d,
	}
	d.ExtList = []*ExtInfo{jsWrapper, builtin}
	d.ExtRegistry.Store("js-ext", jsWrapper)
	d.ExtRegistry.Store("builtin", builtin)
	d.JsExtRegistry.Store("js-ext", jsReal)

	raw := `{"groupId":"QQ-Group:1","activatedExtList":[{"name":"js-ext"},{"name":"builtin"}]}`
	var group GroupInfo
	if err := json.Unmarshal([]byte(raw), &group); err != nil {
		t.Fatalf("unmarshal legacy group json: %v", err)
	}

	group.ExtAppliedTime = 1
	got := group.GetActivatedExtList(d)
	if len(got) != 2 {
		t.Fatalf("expected 2 activated exts, got %d", len(got))
	}
	if got[0] != jsWrapper || got[1] != builtin {
		t.Fatalf("expected legacy object entries to normalize to current refs, got %#v", got)
	}
}

func TestGroupInfoUnmarshalJSONSupportsStringEntries(t *testing.T) {
	d := &Dice{
		Logger:      zap.NewNop().Sugar(),
		ExtRegistry: new(SyncMap[string, *ExtInfo]),
	}
	ext1 := &ExtInfo{Name: "ext1", dice: d}
	ext2 := &ExtInfo{Name: "ext2", dice: d}
	d.ExtList = []*ExtInfo{ext1, ext2}
	d.ExtRegistry.Store("ext1", ext1)
	d.ExtRegistry.Store("ext2", ext2)

	raw := `{"groupId":"QQ-Group:2","activatedExtList":["ext1","ext2"]}`
	var group GroupInfo
	if err := json.Unmarshal([]byte(raw), &group); err != nil {
		t.Fatalf("unmarshal name-based group json: %v", err)
	}

	group.ExtAppliedTime = 1
	got := group.GetActivatedExtList(d)
	if len(got) != 2 {
		t.Fatalf("expected 2 activated exts, got %d", len(got))
	}
	if got[0] != ext1 || got[1] != ext2 {
		t.Fatalf("expected string entries to resolve to current refs, got %#v", got)
	}
}
