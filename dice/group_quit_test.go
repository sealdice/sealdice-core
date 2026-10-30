//nolint:testpackage
package dice

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	emitterTypes "sealdice-core/dice/imsdk/onebot/types"
)

type groupQuitTestEmitter struct {
	*onebotTestEmitter
	quitGroups []int64
}

func (e *groupQuitTestEmitter) QuitGroup(_ context.Context, groupID int64) error {
	e.quitGroups = append(e.quitGroups, groupID)
	return nil
}

func newGroupQuitTestContext(t *testing.T, role string) (*MsgContext, *groupQuitTestEmitter) {
	t.Helper()
	e := &groupQuitTestEmitter{onebotTestEmitter: &onebotTestEmitter{
		groupMemberInfo: &emitterTypes.GroupMemberInfo{Role: role},
	}}
	pa := &PlatformAdapterOnebot{ctx: t.Context(), sendEmitter: e}
	ep := &EndPointInfo{
		EndPointInfoBase: EndPointInfoBase{ID: t.Name(), UserID: "QQ:10010", Platform: "QQ", ProtocolType: "onebot"},
		Adapter:          pa,
	}
	pa.EndPoint = ep
	return &MsgContext{EndPoint: ep, Dice: &Dice{Logger: zap.NewNop().Sugar()}}, e
}

func TestBlacklistQuitRequiresConfirmedNonOwner(t *testing.T) {
	for _, tt := range []struct {
		name     string
		role     string
		err      error
		wantQuit bool
	}{
		{name: "owner", role: "owner"},
		{name: "member without profile fields", role: "member", wantQuit: true},
		{name: "admin", role: "admin", wantQuit: true},
		{name: "unknown role", role: "unknown"},
		{name: "empty role"},
		{name: "lookup failed", err: errors.New("offline")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, emitter := newGroupQuitTestContext(t, tt.role)
			emitter.groupMemberInfoErr = tt.err
			announced := false
			quit := quitGroupForBlacklist(ctx, testMilkyOwnerGroupID, func() { announced = true })
			if quit != tt.wantQuit || announced != tt.wantQuit || (len(emitter.quitGroups) > 0) != tt.wantQuit {
				t.Fatalf("quit = %v, announced = %v, requests = %v, want quit = %v", quit, announced, emitter.quitGroups, tt.wantQuit)
			}
		})
	}
}

func TestBlacklistQuitRechecksAfterFarewell(t *testing.T) {
	ctx, emitter := newGroupQuitTestContext(t, "member")
	if quitGroupForBlacklist(ctx, testMilkyOwnerGroupID, func() { emitter.groupMemberInfo.Role = "owner" }) || len(emitter.quitGroups) != 0 {
		t.Fatal("ownership transferred during the farewell must cancel the quit")
	}
}

func TestOwnerQuitConfirmation(t *testing.T) {
	ctx, emitter := newGroupQuitTestContext(t, "owner")
	const operatorID = "QQ:9001"
	key := getDismissConfirmKeyForGroup(ctx, operatorID, testMilkyOwnerGroupID)
	t.Cleanup(func() { dismissConfirmCodes.Delete(key) })
	_, confirmation := PrepareGroupQuit(ctx, testMilkyOwnerGroupID, operatorID, "")
	if confirmation == nil || !isDismissConfirmCode(confirmation.ConfirmCode) {
		t.Fatalf("expected an owner confirmation challenge, got %#v", confirmation)
	}
	for _, tt := range []struct {
		groupID  string
		operator string
		code     string
	}{
		{testMilkyFallbackGroupID, operatorID, confirmation.ConfirmCode},
		{testMilkyOwnerGroupID, "QQ:9002", confirmation.ConfirmCode},
		{testMilkyOwnerGroupID, operatorID, "invalid"},
	} {
		if permit, blocked := PrepareGroupQuit(ctx, tt.groupID, tt.operator, tt.code); permit != nil || blocked == nil {
			t.Fatal("a confirmation must match the group, operator and code")
		}
	}
	permit, blocked := PrepareGroupQuit(ctx, testMilkyOwnerGroupID, operatorID, confirmation.ConfirmCode)
	if permit == nil || blocked != nil || len(emitter.groupMemberInfoCalls) != 1 {
		t.Fatal("valid confirmations should prepare the quit without another role lookup")
	}
	otherEndpoint := *ctx.EndPoint
	otherEndpoint.UserID = "QQ:10020"
	if QuitGroupSafely(&MsgContext{EndPoint: &otherEndpoint, Dice: ctx.Dice}, testMilkyOwnerGroupID, permit) {
		t.Fatal("a confirmation for another account must not be accepted")
	}
	emitter.groupMemberInfoErr = errors.New("offline")
	if QuitGroupSafely(ctx, testMilkyOwnerGroupID, permit) || len(emitter.quitGroups) != 0 {
		t.Fatal("confirmation must not bypass a failed final identity check")
	}
	emitter.groupMemberInfoErr = nil
	if !QuitGroupSafely(ctx, testMilkyOwnerGroupID, permit) {
		t.Fatal("explicit owner confirmation should allow quitting")
	}
	if QuitGroupSafely(ctx, testMilkyOwnerGroupID, permit) || len(emitter.quitGroups) != 1 {
		t.Fatalf("confirmation must be single use, quit requests: %v", emitter.quitGroups)
	}
	dismissConfirmCodes.Store(key, &dismissConfirmState{Code: confirmation.ConfirmCode, ExpiresAt: time.Now().Add(-time.Minute).Unix()})
	if permit, blocked := PrepareGroupQuit(ctx, testMilkyOwnerGroupID, operatorID, confirmation.ConfirmCode); permit != nil || blocked == nil {
		t.Fatal("expired confirmations must not allow quitting")
	}
}

func TestBlacklistEntryPointsDoNotQuitOwners(t *testing.T) {
	for _, trigger := range []string{"blacklisted user", "blacklisted admin", "blacklisted group", "last-place punishment"} {
		t.Run(trigger, func(t *testing.T) {
			d, _, _, cleanup := newExecuteNewTestDice(t)
			defer cleanup()
			roleContext, emitter := newGroupQuitTestContext(t, "owner")
			ep := roleContext.EndPoint
			ep.Session = d.ImSession
			ctx, msg := newQuitCommandTestContext(t, d, ep, "QQ:9001", testMilkyOwnerGroupID, "OwnedGroup")
			ctx.PrivilegeLevel = 0
			banList := d.Config.BanList
			banList.BanBehaviorQuitIfAdmin = false
			banList.BanBehaviorQuitIfAdminSilentIfNotAdmin = false
			banList.BanBehaviorQuitPlaceImmediately = true
			userBan := &BanListInfoItem{ID: msg.Sender.UserID, Rank: BanRankBanned}
			switch trigger {
			case "blacklisted user", "blacklisted admin":
				ctx.PrivilegeLevel = -30
				banList.Map.Store(msg.Sender.UserID, userBan)
				if trigger == "blacklisted admin" {
					ctx.GroupRoleLevel = 50
					banList.BanBehaviorQuitIfAdmin = true
				}
			case "blacklisted group":
				banList.Map.Store(msg.GroupID, &BanListInfoItem{ID: msg.GroupID, Rank: BanRankBanned})
			case "last-place punishment":
				banList.Map.Store(msg.Sender.UserID, userBan)
				banList.BanBehaviorQuitLastPlace = true
				banList.NoticeCheck(msg.Sender.UserID, msg.GroupID, BanRankNormal, ctx)
			}
			if trigger != "last-place punishment" && !checkBan(ctx, msg) {
				t.Fatal("blacklisted messages must remain blocked even when quitting is prevented")
			}
			waitForTestMessageTasks(t, d)
			if len(emitter.groupMemberInfoCalls) == 0 || len(emitter.quitGroups) != 0 {
				t.Fatalf("expected owner check without quitting: lookups=%v quits=%v", emitter.groupMemberInfoCalls, emitter.quitGroups)
			}
			if !ctx.Group.DiceIDExistsMap.Exists(ep.UserID) {
				t.Fatal("prevented blacklist quit must preserve group membership")
			}
		})
	}
}

func TestBlacklistedInvitationDoesNotQuitOwner(t *testing.T) {
	for _, target := range []string{"inviter", "group"} {
		t.Run(target, func(t *testing.T) {
			d, pa, baseEmitter, cleanup := newPureOnebotTestAdapter(t)
			defer cleanup()
			baseEmitter.groupMemberInfo = &emitterTypes.GroupMemberInfo{Role: "owner"}
			emitter := &groupQuitTestEmitter{onebotTestEmitter: baseEmitter}
			pa.sendEmitter = emitter
			baseEmitter.groupInfo = &emitterTypes.GroupInfo{GroupId: 2010, GroupName: "OwnedGroup", MaxMemberCount: 100}
			ctx, _ := newQuitCommandTestContext(t, d, pa.EndPoint, "QQ:9001", testMilkyOwnerGroupID, "OwnedGroup")
			ctx.Group.InviteUserID = "QQ:9002"
			ctx.Group.EnteredTime = time.Now().Unix()
			uid := ctx.Group.InviteUserID
			if target == "group" {
				uid = ctx.Group.GroupID
			}
			d.Config.BanList.BanBehaviorRefuseInvite = true
			d.Config.BanList.Map.Store(uid, &BanListInfoItem{ID: uid, Rank: BanRankBanned, BanTime: ctx.Group.EnteredTime - 60})
			pa.GetGroupInfoSync(ctx.Group.GroupID)
			waitForTestMessageTasks(t, d)
			if len(emitter.groupMemberInfoCalls) == 0 || len(emitter.quitGroups) != 0 {
				t.Fatalf("blacklisted invitation must check ownership before quitting: lookups=%v quits=%v", emitter.groupMemberInfoCalls, emitter.quitGroups)
			}
		})
	}
}
