//nolint:testpackage
package dice

import (
	"context"
	"errors"
	"testing"

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
	if confirmation == nil || confirmation.ConfirmCode == "" {
		t.Fatal("owner quit must require confirmation")
	}
	if permit, blocked := PrepareGroupQuit(ctx, testMilkyFallbackGroupID, operatorID, confirmation.ConfirmCode); permit != nil || blocked == nil {
		t.Fatal("confirmation must not allow quitting another group")
	}
	permit, blocked := PrepareGroupQuit(ctx, testMilkyOwnerGroupID, operatorID, confirmation.ConfirmCode)
	if permit == nil || blocked != nil {
		t.Fatal("valid confirmation must allow preparing the quit")
	}
	emitter.groupMemberInfoErr = errors.New("offline")
	if QuitGroupSafely(ctx, testMilkyOwnerGroupID, permit) {
		t.Fatal("confirmation must not bypass a failed final identity check")
	}
	emitter.groupMemberInfoErr = nil
	if !QuitGroupSafely(ctx, testMilkyOwnerGroupID, permit) {
		t.Fatal("explicit owner confirmation should allow quitting")
	}
	if QuitGroupSafely(ctx, testMilkyOwnerGroupID, permit) || len(emitter.quitGroups) != 1 {
		t.Fatalf("confirmation must be single use, quit requests: %v", emitter.quitGroups)
	}
}
