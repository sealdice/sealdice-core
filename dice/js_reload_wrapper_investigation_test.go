package dice //nolint:testpackage

import (
	"testing"

	"go.uber.org/zap"
)

func TestInvestigation_JsReloadWrapperRegistryKeepsWrapperButSwitchesRealExt(t *testing.T) {
	d := &Dice{
		Logger:         zap.NewNop().Sugar(),
		ExtRegistry:    new(SyncMap[string, *ExtInfo]),
		JsExtRegistry:  new(SyncMap[string, *ExtInfo]),
		ExtLoopManager: NewJsLoopManager(),
		GameSystemMap:  new(SyncMap[string, *GameSystemTemplate]),
		DirtyGroups:    new(SyncMap[string, int64]),
		CocExtraRules:  map[int]*CocRuleInfo{},
		ImSession:      &IMSession{ServiceAtNew: new(SyncMap[string, *GroupInfo])},
		ConfigManager:  NewConfigManager(t.TempDir() + "/plugin-configs.json"),
		BaseConfig:     BaseConfig{DataDir: t.TempDir()},
	}

	wrapper := &ExtInfo{
		Name:          "myPlugin",
		IsWrapper:     true,
		TargetName:    "myPlugin",
		IsJsExt:       true,
		JSLoopVersion: 1,
		dice:          d,
	}
	d.ExtList = []*ExtInfo{wrapper}
	d.ExtRegistry.Store("myPlugin", wrapper)
	d.ExtRegistry.Store("myplugin", wrapper)

	real1Called := 0
	real1 := &ExtInfo{
		Name:          "myPlugin",
		IsJsExt:       true,
		JSLoopVersion: 1,
		dice:          d,
		OnNotCommandReceived: func(_ *MsgContext, _ *Message) {
			real1Called++
		},
	}
	d.JsExtRegistry.Store("myPlugin", real1)

	group := newTestGroupInfo()
	group.SetActivatedExtList([]*ExtInfo{wrapper}, d)

	got1 := group.GetActivatedExtList(d)
	if len(got1) != 1 || got1[0] != wrapper {
		t.Fatalf("expected activated list to keep wrapper, got %#v", got1)
	}
	if gotReal := got1[0].GetRealExt(); gotReal != real1 {
		t.Fatalf("expected wrapper to resolve initial real ext, got %#v", gotReal)
	}
	got1[0].GetRealExt().OnNotCommandReceived(nil, nil)
	if real1Called != 1 {
		t.Fatalf("expected initial callback to run once, got %d", real1Called)
	}

	d.jsClear()

	if gotReal := wrapper.GetRealExt(); gotReal != nil {
		t.Fatalf("expected wrapper to resolve nil immediately after jsClear, got %#v", gotReal)
	}
	if _, ok := d.ExtRegistry.Load("myPlugin"); !ok {
		t.Fatalf("expected ExtRegistry to keep wrapper after jsClear")
	}
	if len(d.ExtList) != 1 || d.ExtList[0] != wrapper {
		t.Fatalf("expected ExtList to keep wrapper after jsClear")
	}

	real2Called := 0
	real2 := &ExtInfo{
		Name:          "myPlugin",
		IsJsExt:       true,
		JSLoopVersion: 2,
		dice:          d,
		OnNotCommandReceived: func(_ *MsgContext, _ *Message) {
			real2Called++
		},
	}
	d.JsExtRegistry.Store("myPlugin", real2)

	got2 := group.GetActivatedExtList(d)
	if len(got2) != 1 || got2[0] != wrapper {
		t.Fatalf("expected activated list to still keep wrapper after reload, got %#v", got2)
	}
	gotReal2 := got2[0].GetRealExt()
	if gotReal2 != real2 {
		t.Fatalf("expected wrapper to resolve new real ext after reload, got %#v", gotReal2)
	}

	gotReal2.OnNotCommandReceived(nil, nil)
	if real1Called != 1 {
		t.Fatalf("expected old callback count to stay 1, got %d", real1Called)
	}
	if real2Called != 1 {
		t.Fatalf("expected new callback count to be 1, got %d", real2Called)
	}
}

func TestInvestigation_SetBotOnAtGroupKeepsWrapperFromDefaultSettings(t *testing.T) {
	d := &Dice{
		Logger:      zap.NewNop().Sugar(),
		ExtRegistry: new(SyncMap[string, *ExtInfo]),
		BaseConfig:  BaseConfig{DataDir: t.TempDir()},
	}
	d.ImSession = &IMSession{Parent: d, ServiceAtNew: new(SyncMap[string, *GroupInfo])}

	wrapper := &ExtInfo{
		Name:       "myPlugin",
		IsWrapper:  true,
		TargetName: "myPlugin",
		IsJsExt:    true,
		dice:       d,
	}
	d.ExtList = []*ExtInfo{wrapper}
	d.Config.ExtDefaultSettings = []*ExtDefaultSettingItem{
		{
			Name:       "myPlugin",
			AutoActive: true,
			ExtItem:    wrapper,
		},
	}

	ctx := &MsgContext{
		Dice:    d,
		Session: d.ImSession,
		EndPoint: &EndPointInfo{
			EndPointInfoBase: EndPointInfoBase{UserID: "QQ:bot"},
		},
	}

	group := SetBotOnAtGroup(ctx, "QQ-Group:123")
	got := group.GetActivatedExtListRaw()
	if len(got) != 1 {
		t.Fatalf("expected one activated ext, got %d", len(got))
	}
	if got[0] != wrapper {
		t.Fatalf("expected SetBotOnAtGroup to cache wrapper from default settings, got %#v", got[0])
	}
}

func TestInvestigation_ActivatedListHoldingOldRealExtCausesVersionMismatch(t *testing.T) {
	d := &Dice{
		Logger:         zap.NewNop().Sugar(),
		ExtLoopManager: NewJsLoopManager(),
		ExtRegistry:    new(SyncMap[string, *ExtInfo]),
		JsExtRegistry:  new(SyncMap[string, *ExtInfo]),
	}

	// Simulate current loop version becoming 10 after many reloads.
	for range 10 {
		d.ExtLoopManager.SetLoop(nil)
	}

	wrapper := &ExtInfo{
		Name:       "billboard",
		IsWrapper:  true,
		TargetName: "billboard",
		IsJsExt:    true,
		dice:       d,
	}
	oldRealExt := &ExtInfo{
		Name:          "billboard",
		IsJsExt:       true,
		JSLoopVersion: 2,
		dice:          d,
	}
	newRealExt := &ExtInfo{
		Name:          "billboard",
		IsJsExt:       true,
		JSLoopVersion: 10,
		dice:          d,
	}
	d.ExtList = []*ExtInfo{wrapper}
	d.ExtRegistry.Store("billboard", wrapper)
	d.JsExtRegistry.Store("billboard", newRealExt)
	group := newTestGroupInfo()
	group.SetActivatedExtList([]*ExtInfo{oldRealExt}, d)
	group.ExtAppliedTime = 1

	got := group.GetActivatedExtList(d)
	if len(got) != 1 || got[0] != wrapper {
		t.Fatalf("expected activated list to normalize stale real ext to wrapper, got %#v", got)
	}
	if got[0].GetRealExt() != newRealExt {
		t.Fatalf("expected wrapper to resolve current real ext")
	}

	_, err := d.ExtLoopManager.GetLoop(got[0].JSLoopVersion)
	if err == nil {
		t.Fatalf("expected wrapper JSLoopVersion itself to be stale and mismatched")
	}
	_, err = d.ExtLoopManager.GetLoop(got[0].GetRealExt().JSLoopVersion)
	if err != nil {
		t.Fatalf("expected current real ext version to match active loop, got %v", err)
	}
}

func TestInvestigation_StaleRealExtCacheResolvesToCurrentRealExtAfterNormalization(t *testing.T) {
	d := &Dice{
		Logger:         zap.NewNop().Sugar(),
		ExtLoopManager: NewJsLoopManager(),
		ExtRegistry:    new(SyncMap[string, *ExtInfo]),
		JsExtRegistry:  new(SyncMap[string, *ExtInfo]),
	}
	for range 10 {
		d.ExtLoopManager.SetLoop(nil)
	}

	wrapper := &ExtInfo{
		Name:          "team",
		IsWrapper:     true,
		TargetName:    "team",
		IsJsExt:       true,
		JSLoopVersion: 10,
		dice:          d,
	}
	oldRealExt := &ExtInfo{
		Name:          "team",
		IsJsExt:       true,
		JSLoopVersion: 8,
		dice:          d,
	}
	newRealExt := &ExtInfo{
		Name:          "team",
		IsJsExt:       true,
		JSLoopVersion: 10,
		dice:          d,
	}
	d.ExtList = []*ExtInfo{wrapper}
	d.ExtRegistry.Store("team", wrapper)
	d.JsExtRegistry.Store("team", newRealExt)

	group := newTestGroupInfo()
	group.SetActivatedExtList([]*ExtInfo{oldRealExt}, d)
	group.ExtAppliedTime = 1

	activated := group.GetActivatedExtList(d)
	if len(activated) != 1 {
		t.Fatalf("expected one activated ext, got %d", len(activated))
	}
	if activated[0] != wrapper {
		t.Fatalf("expected stale cached real ext to normalize to wrapper")
	}
	resolvedRealExt := activated[0].GetRealExt()
	if resolvedRealExt != newRealExt {
		t.Fatalf("expected normalized wrapper to resolve to new real ext")
	}
	if resolvedRealExt == oldRealExt {
		t.Fatalf("should not resolve to stale real ext")
	}
}

func TestExtActiveWithRealExtNormalizesToWrapper(t *testing.T) {
	d := &Dice{
		Logger:        zap.NewNop().Sugar(),
		ExtRegistry:   new(SyncMap[string, *ExtInfo]),
		JsExtRegistry: new(SyncMap[string, *ExtInfo]),
	}

	wrapper := &ExtInfo{
		Name:       "polluted",
		IsWrapper:  true,
		TargetName: "polluted",
		IsJsExt:    true,
		dice:       d,
	}
	realExt := &ExtInfo{
		Name:    "polluted",
		IsJsExt: true,
		dice:    d,
	}
	d.ExtList = []*ExtInfo{wrapper}
	d.ExtRegistry.Store("polluted", wrapper)
	d.JsExtRegistry.Store("polluted", realExt)

	group := newTestGroupInfo()
	group.ExtAppliedTime = 1

	// This simulates a JS script calling ctx.group.ExtActive(seal.ext.find("polluted")).
	group.ExtActive(realExt)

	got := group.GetActivatedExtListRaw()
	if len(got) != 1 {
		t.Fatalf("expected one activated ext, got %d", len(got))
	}
	if got[0] != wrapper {
		t.Fatalf("expected activated list to normalize real ext to wrapper, got %#v", got[0])
	}
	if !got[0].IsWrapper {
		t.Fatalf("expected normalized entry to be wrapper")
	}
}
