package group

import (
	"context"
	"testing"

	groupm "sealdice-core/api/v2/model/group"
	"sealdice-core/dice"
	"sealdice-core/logger"
	areq "sealdice-core/model/api/req"
	"sealdice-core/model/common/request"
	"sealdice-core/message"
)

type fakePlatformAdapter struct {
	quitGroupIDs []string
}

func (f *fakePlatformAdapter) Serve() int { return 0 }
func (f *fakePlatformAdapter) DoRelogin() bool { return true }
func (f *fakePlatformAdapter) SetEnable(enable bool) {}
func (f *fakePlatformAdapter) QuitGroup(ctx *dice.MsgContext, ID string) {
	f.quitGroupIDs = append(f.quitGroupIDs, ID)
}
func (f *fakePlatformAdapter) SendToPerson(ctx *dice.MsgContext, userID string, text string, flag string) {}
func (f *fakePlatformAdapter) SendToGroup(ctx *dice.MsgContext, groupID string, text string, flag string) {}
func (f *fakePlatformAdapter) SetGroupCardName(ctx *dice.MsgContext, name string) {}
func (f *fakePlatformAdapter) SendSegmentToGroup(ctx *dice.MsgContext, groupID string, msg []message.IMessageElement, flag string) {
}
func (f *fakePlatformAdapter) SendSegmentToPerson(ctx *dice.MsgContext, userID string, msg []message.IMessageElement, flag string) {
}
func (f *fakePlatformAdapter) SendFileToPerson(ctx *dice.MsgContext, userID string, path string, flag string) {}
func (f *fakePlatformAdapter) SendFileToGroup(ctx *dice.MsgContext, groupID string, path string, flag string) {}
func (f *fakePlatformAdapter) MemberBan(groupID string, userID string, duration int64) {}
func (f *fakePlatformAdapter) MemberKick(groupID string, userID string) {}
func (f *fakePlatformAdapter) GetGroupInfoAsync(groupID string) {}
func (f *fakePlatformAdapter) EditMessage(ctx *dice.MsgContext, msgID, message string) {}
func (f *fakePlatformAdapter) RecallMessage(ctx *dice.MsgContext, msgID string) {}

func newTestGroupService(t *testing.T) *GroupService {
	t.Helper()

	d := &dice.Dice{
		Logger: logger.M(),
	}
	d.BaseConfig.Name = "group-test"
	d.BaseConfig.DataDir = t.TempDir()
	d.Config = dice.NewConfig(d)
	d.ImSession = &dice.IMSession{
		Parent:       d,
		EndPoints:    []*dice.EndPointInfo{},
		ServiceAtNew: new(dice.SyncMap[string, *dice.GroupInfo]),
		PendingQuits: new(dice.SyncMap[string, *dice.PendingQuitInfo]),
	}
	dm := &dice.DiceManager{
		Dice:        []*dice.Dice{d},
		JustForTest: true,
	}
	d.Parent = dm
	return NewGroupService(dm)
}

func newTestGroup(groupID string, name string, activeDiceIDs ...string) *dice.GroupInfo {
	g := &dice.GroupInfo{
		Active:           true,
		GroupID:          groupID,
		GroupName:        name,
		DiceIDActiveMap:  new(dice.SyncMap[string, bool]),
		DiceIDExistsMap:  new(dice.SyncMap[string, bool]),
		BotList:          new(dice.SyncMap[string, bool]),
		Players:          new(dice.SyncMap[string, *dice.GroupPlayerInfo]),
		PlayerGroups:     new(dice.SyncMap[string, []string]),
		InactivatedExtSet: dice.StringSet{},
	}
	for _, id := range activeDiceIDs {
		g.DiceIDExistsMap.Store(id, true)
		g.DiceIDActiveMap.Store(id, true)
	}
	return g
}

func TestGetGroupPageReturnsVisibleGroupsAndTmpExtList(t *testing.T) {
	svc := newTestGroupService(t)

	visible := newTestGroup("QQ-Group:1001", "Alpha", "QQ:1001")
	visible.SetActivatedExtList([]*dice.ExtInfo{{Name: "coc7"}, {Name: "reply"}}, nil)
	hiddenPG := newTestGroup("PG-Group:shadow", "PG", "QQ:1001")
	hiddenNoDice := newTestGroup("DISCORD-CH-Group:2001", "No Dice")

	svc.dice.ImSession.ServiceAtNew.Store(visible.GroupID, visible)
	svc.dice.ImSession.ServiceAtNew.Store(hiddenPG.GroupID, hiddenPG)
	svc.dice.ImSession.ServiceAtNew.Store(hiddenNoDice.GroupID, hiddenNoDice)

	resp, err := svc.GetGroupPage(context.Background(), request.NewRequestWrapper(areq.GroupPageRequest{
		PageInfo: request.PageInfo{Page: 1, PageSize: 10},
	}))
	if err != nil {
		t.Fatalf("GetGroupPage returned error: %v", err)
	}

	if len(resp.Body.Item.List) != 1 {
		t.Fatalf("group list length = %d, want 1", len(resp.Body.Item.List))
	}
	if resp.Body.Item.List[0].GroupID != visible.GroupID {
		t.Fatalf("group id = %q, want %q", resp.Body.Item.List[0].GroupID, visible.GroupID)
	}
	if got := resp.Body.Item.List[0].TmpExtList; len(got) != 2 || got[0] != "coc7" || got[1] != "reply" {
		t.Fatalf("tmpExtList = %#v, want [coc7 reply]", got)
	}
}

func TestGetPlatformsReturnsVisiblePlatformOptions(t *testing.T) {
	svc := newTestGroupService(t)
	svc.dice.ImSession.ServiceAtNew.Store("QQ-Group:1001", newTestGroup("QQ-Group:1001", "QQ", "QQ:1001"))
	svc.dice.ImSession.ServiceAtNew.Store("QQ-Group:1002", newTestGroup("QQ-Group:1002", "QQ2", "QQ:1001"))
	svc.dice.ImSession.ServiceAtNew.Store("DISCORD-CH-Group:2001", newTestGroup("DISCORD-CH-Group:2001", "Discord", "DISCORD:1001"))
	svc.dice.ImSession.ServiceAtNew.Store("PG-Group:shadow", newTestGroup("PG-Group:shadow", "PG", "QQ:1001"))

	resp, err := svc.GetPlatforms(context.Background(), &request.Empty{})
	if err != nil {
		t.Fatalf("GetPlatforms returned error: %v", err)
	}
	if len(resp.Body.Item) != 2 {
		t.Fatalf("platform options = %d, want 2", len(resp.Body.Item))
	}
	if resp.Body.Item[0].Value != "DISCORD-CH-GROUP" || resp.Body.Item[1].Value != "QQ-GROUP" {
		t.Fatalf("platform values = %#v", resp.Body.Item)
	}
}

func TestQuitGroupUsesExplicitDiceID(t *testing.T) {
	svc := newTestGroupService(t)

	adapter1 := &fakePlatformAdapter{}
	adapter2 := &fakePlatformAdapter{}
	ep1 := &dice.EndPointInfo{
		EndPointInfoBase: dice.EndPointInfoBase{ID: "ep-1", UserID: "QQ:1001", Platform: "QQ", Enable: true},
		Adapter:          adapter1,
	}
	ep2 := &dice.EndPointInfo{
		EndPointInfoBase: dice.EndPointInfoBase{ID: "ep-2", UserID: "QQ:1002", Platform: "QQ", Enable: true},
		Adapter:          adapter2,
	}
	svc.dice.ImSession.EndPoints = []*dice.EndPointInfo{ep1, ep2}

	group := newTestGroup("QQ-Group:1001", "Alpha", ep1.UserID, ep2.UserID)
	svc.dice.ImSession.ServiceAtNew.Store(group.GroupID, group)

	resp, err := svc.QuitGroup(context.Background(), request.NewRequestWrapper(groupm.QuitGroupRequest{
		GroupID: group.GroupID,
		DiceID:  ep2.UserID,
		Silence: true,
	}))
	if err != nil {
		t.Fatalf("QuitGroup returned error: %v", err)
	}
	if resp.Body.Message != "ok" {
		t.Fatalf("QuitGroup message = %q, want ok", resp.Body.Message)
	}
	if len(adapter1.quitGroupIDs) != 0 {
		t.Fatalf("adapter1 quit calls = %#v, want 0", adapter1.quitGroupIDs)
	}
	if len(adapter2.quitGroupIDs) != 1 || adapter2.quitGroupIDs[0] != group.GroupID {
		t.Fatalf("adapter2 quit calls = %#v, want %q", adapter2.quitGroupIDs, group.GroupID)
	}
	if group.DiceIDExistsMap.Exists(ep2.UserID) {
		t.Fatalf("diceId %q should be removed from exists map", ep2.UserID)
	}
	if !group.DiceIDExistsMap.Exists(ep1.UserID) {
		t.Fatalf("diceId %q should remain in exists map", ep1.UserID)
	}
}
