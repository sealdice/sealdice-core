//nolint:testpackage
package dice

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sacOO7/gowebsocket"
	"go.uber.org/zap"

	emitterTypes "sealdice-core/dice/imsdk/onebot/types"
)

type groupQuitTestEmitter struct {
	*onebotTestEmitter
	role           string
	lookupError    error
	nilResponse    bool
	wrongUser      bool
	wrongGroup     bool
	missingProfile bool
	quitGroups     []int64
}

func (e *groupQuitTestEmitter) GetGroupMemberInfo(ctx context.Context, groupID, userID int64, noCache bool) (*emitterTypes.GroupMemberInfo, error) {
	e.groupMemberInfoCalls = append(e.groupMemberInfoCalls, groupMemberInfoCall{GroupID: groupID, UserID: userID, NoCache: noCache})
	if e.lookupError != nil || e.nilResponse {
		return nil, e.lookupError
	}
	if e.wrongUser {
		userID++
	}
	if e.wrongGroup {
		groupID++
	}
	member := &emitterTypes.GroupMemberInfo{GroupId: groupID, UserId: userID, Nickname: "Bot", Role: e.role}
	if e.missingProfile {
		member.Nickname = ""
	}
	return member, nil
}

func (e *groupQuitTestEmitter) QuitGroup(_ context.Context, groupID int64) error {
	e.quitGroups = append(e.quitGroups, groupID)
	return nil
}

func newGroupQuitTestContext(t *testing.T, role string) (*MsgContext, *groupQuitTestEmitter) {
	t.Helper()
	e := &groupQuitTestEmitter{onebotTestEmitter: &onebotTestEmitter{}, role: role}
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
		name           string
		role           string
		lookupError    error
		nilResponse    bool
		wrongUser      bool
		wrongGroup     bool
		missingProfile bool
		wantQuit       bool
	}{
		{name: "owner", role: "owner"},
		{name: "owner alias", role: " OWNER "},
		{name: "member", role: "member", wantQuit: true},
		{name: "admin", role: "admin", wantQuit: true},
		{name: "missing role"},
		{name: "unknown role", role: "unknown"},
		{name: "numeric role", role: "1"},
		{name: "lookup failed", lookupError: errors.New("offline")},
		{name: "nil response", nilResponse: true},
		{name: "another member", role: "member", wrongUser: true},
		{name: "another group", role: "member", wrongGroup: true},
		{name: "synthetic member fallback", role: "member", missingProfile: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, emitter := newGroupQuitTestContext(t, tt.role)
			emitter.lookupError = tt.lookupError
			emitter.nilResponse = tt.nilResponse
			emitter.wrongUser = tt.wrongUser
			emitter.wrongGroup = tt.wrongGroup
			emitter.missingProfile = tt.missingProfile
			announced := false
			if got := quitGroupForBlacklist(ctx, testMilkyOwnerGroupID, func() { announced = true }); got != tt.wantQuit {
				t.Fatalf("quit result = %v, want %v", got, tt.wantQuit)
			}
			if announced != tt.wantQuit || (len(emitter.quitGroups) == 1) != tt.wantQuit {
				t.Fatalf("announced = %v, quit requests = %v", announced, emitter.quitGroups)
			}
			for _, call := range emitter.groupMemberInfoCalls {
				if call.GroupID != testMilkyOwnerGroupQQID || call.UserID != testMilkyBotQQID || !call.NoCache {
					t.Fatalf("identity lookup must refresh the bot's membership: %#v", call)
				}
			}
		})
	}
}

func TestBlacklistQuitRechecksAfterFarewell(t *testing.T) {
	ctx, emitter := newGroupQuitTestContext(t, "member")
	if quitGroupForBlacklist(ctx, testMilkyOwnerGroupID, func() { emitter.role = "owner" }) {
		t.Fatal("ownership transferred during the farewell must cancel the quit")
	}
	if len(emitter.quitGroups) != 0 || len(emitter.groupMemberInfoCalls) != 2 {
		t.Fatalf("quit requests = %v, role checks = %d", emitter.quitGroups, len(emitter.groupMemberInfoCalls))
	}
}

func TestCheckBotGroupRoleLegacyOnebotValidatesFreshResponse(t *testing.T) {
	for _, tt := range []struct {
		name     string
		data     string
		status   string
		retcode  int
		wantRole string
	}{
		{name: "owner", data: `{"group_id":2010,"user_id":10010,"role":"owner"}`, wantRole: "owner"},
		{name: "admin", data: `{"group_id":2010,"user_id":10010,"role":"admin"}`, wantRole: "admin"},
		{name: "member", data: `{"group_id":2010,"user_id":10010,"role":"member","nickname":"Bot"}`, wantRole: "member"},
		{name: "member with join time", data: `{"group_id":2010,"user_id":10010,"role":"member","join_time":1}`, wantRole: "member"},
		{name: "synthetic member fallback", data: `{"group_id":2010,"user_id":10010,"role":"member","nickname":"","join_time":0}`},
		{name: "another user", data: `{"group_id":2010,"user_id":10011,"role":"admin"}`},
		{name: "another group", data: `{"group_id":2011,"user_id":10010,"role":"admin"}`},
		{name: "unknown role", data: `{"group_id":2010,"user_id":10010,"role":"unknown"}`},
		{name: "failed status", status: "failed", data: `{"group_id":2010,"user_id":10010,"role":"admin"}`},
		{name: "failed return code", retcode: 1, data: `{"group_id":2010,"user_id":10010,"role":"admin"}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				upgrader := websocket.Upgrader{}
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					t.Error(err)
					return
				}
				defer conn.Close()
				var request struct {
					Action string `json:"action"`
					Params struct {
						GroupID string `json:"group_id"`
						UserID  string `json:"user_id"`
						NoCache bool   `json:"no_cache"`
					} `json:"params"`
					Echo json.RawMessage `json:"echo"`
				}
				if err = conn.ReadJSON(&request); err != nil {
					t.Error(err)
					return
				}
				if request.Action != "get_group_member_info" || request.Params.GroupID != "2010" || request.Params.UserID != "10010" || !request.Params.NoCache {
					t.Errorf("identity query must refresh the selected bot's membership: %#v", request)
				}
				status := tt.status
				if status == "" {
					status = "ok"
				}
				if err = conn.WriteJSON(map[string]any{
					"status": status, "retcode": tt.retcode, "echo": request.Echo, "data": json.RawMessage(tt.data),
				}); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			conn, resp, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
			if resp != nil {
				_ = resp.Body.Close()
			}
			if err != nil {
				t.Fatal(err)
			}
			socket := gowebsocket.New("")
			socket.Conn = conn
			pa := &PlatformAdapterGocq{Socket: &socket}
			received := make(chan struct{})
			go func() {
				defer close(received)
				var reply MessageQQ
				if err := conn.ReadJSON(&reply); err != nil {
					t.Error(err)
					return
				}
				if waiter, ok := pa.echoMap.LoadAndDelete(string(reply.Echo)); ok {
					waiter <- &reply
				}
			}()
			defer func() {
				_ = conn.Close()
				<-received
			}()
			ctx := &MsgContext{EndPoint: &EndPointInfo{
				EndPointInfoBase: EndPointInfoBase{UserID: "QQ:10010", Platform: "QQ"}, Adapter: pa,
			}}
			role, ok := checkBotGroupRole(ctx, testMilkyOwnerGroupID)
			if ok != (tt.wantRole != "") || (ok && role != tt.wantRole) {
				t.Fatalf("identity lookup = (%q, %v), want role %q", role, ok, tt.wantRole)
			}
			if pa.echoMap.Len() != 0 {
				t.Fatal("completed identity query must release its echo waiter")
			}
		})
	}
}

func TestOwnerQuitConfirmationIsBoundAndSingleUse(t *testing.T) {
	ctx, emitter := newGroupQuitTestContext(t, "owner")
	const operatorID = "QQ:9001"
	_, confirmation := PrepareGroupQuit(ctx, testMilkyOwnerGroupID, operatorID, "")
	if confirmation == nil || !isDismissConfirmCode(confirmation.ConfirmCode) {
		t.Fatalf("expected an owner confirmation challenge, got %#v", confirmation)
	}
	key := getDismissConfirmKeyForGroup(ctx, operatorID, testMilkyOwnerGroupID)
	t.Cleanup(func() { dismissConfirmCodes.Delete(key) })
	if QuitGroupSafely(ctx, testMilkyOwnerGroupID, nil) {
		t.Fatal("automated quits must never use a pending manual confirmation")
	}
	for _, tt := range []struct {
		name     string
		groupID  string
		operator string
		code     string
	}{
		{name: "wrong group", groupID: testMilkyFallbackGroupID, operator: operatorID, code: confirmation.ConfirmCode},
		{name: "wrong operator", groupID: testMilkyOwnerGroupID, operator: "QQ:9002", code: confirmation.ConfirmCode},
		{name: "wrong code", groupID: testMilkyOwnerGroupID, operator: operatorID, code: "invalid"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			permit, blocked := PrepareGroupQuit(ctx, tt.groupID, tt.operator, tt.code)
			if permit != nil || blocked == nil {
				t.Fatal("an unrelated or invalid confirmation must not allow quitting")
			}
		})
	}
	permit, blocked := PrepareGroupQuit(ctx, testMilkyOwnerGroupID, operatorID, confirmation.ConfirmCode)
	if permit == nil || blocked != nil {
		t.Fatalf("expected valid confirmation to prepare the quit, got %#v", blocked)
	}
	otherEndpoint := *ctx.EndPoint
	otherEndpoint.UserID = "QQ:10020"
	otherContext := &MsgContext{EndPoint: &otherEndpoint, Dice: ctx.Dice}
	if QuitGroupSafely(otherContext, testMilkyOwnerGroupID, permit) {
		t.Fatal("a confirmation for another account must not be accepted")
	}
	if !QuitGroupSafely(ctx, testMilkyOwnerGroupID, permit) {
		t.Fatal("a valid owner confirmation should allow the explicit quit")
	}
	if QuitGroupSafely(ctx, testMilkyOwnerGroupID, permit) || len(emitter.quitGroups) != 1 {
		t.Fatalf("owner confirmation must be single use, quit requests: %v", emitter.quitGroups)
	}
}

func TestOwnerQuitStillRequiresSuccessfulIdentityLookup(t *testing.T) {
	ctx, emitter := newGroupQuitTestContext(t, "owner")
	const operatorID = "QQ:9001"
	_, confirmation := PrepareGroupQuit(ctx, testMilkyOwnerGroupID, operatorID, "")
	if confirmation == nil || confirmation.ConfirmCode == "" {
		t.Fatal("expected owner confirmation")
	}
	key := getDismissConfirmKeyForGroup(ctx, operatorID, testMilkyOwnerGroupID)
	t.Cleanup(func() { dismissConfirmCodes.Delete(key) })
	permit, blocked := PrepareGroupQuit(ctx, testMilkyOwnerGroupID, operatorID, confirmation.ConfirmCode)
	if blocked != nil {
		t.Fatalf("unexpected preparation failure: %#v", blocked)
	}
	emitter.lookupError = errors.New("identity lookup timed out")
	if QuitGroupSafely(ctx, testMilkyOwnerGroupID, permit) || len(emitter.quitGroups) != 0 {
		t.Fatal("confirmation must not bypass a failed final identity check")
	}
	if permit, blocked = PrepareGroupQuit(ctx, testMilkyOwnerGroupID, operatorID, confirmation.ConfirmCode); permit != nil || blocked == nil || blocked.ConfirmCode != "" {
		t.Fatal("failed identity lookups must not issue or accept owner confirmations")
	}
	emitter.lookupError = nil
	dismissConfirmCodes.Store(key, &dismissConfirmState{Code: confirmation.ConfirmCode, ExpiresAt: time.Now().Add(-time.Minute).Unix()})
	if permit, blocked = PrepareGroupQuit(ctx, testMilkyOwnerGroupID, operatorID, confirmation.ConfirmCode); permit != nil || blocked == nil {
		t.Fatal("expired owner confirmations must not be accepted")
	}
}

func TestBlacklistAndManualQuitKeepOtherPlatformsWorking(t *testing.T) {
	adapter := newMockPlatformAdapter()
	ctx := &MsgContext{EndPoint: &EndPointInfo{
		EndPointInfoBase: EndPointInfoBase{UserID: "TG:10010", Platform: "TG"}, Adapter: adapter,
	}}
	if !quitGroupForBlacklist(ctx, "TG-Group:2010", nil) {
		t.Fatal("other platforms must retain automatic quit support")
	}
	permit, blocked := PrepareGroupQuit(ctx, "TG-Group:2020", "TG:9001", "")
	if blocked != nil || !QuitGroupSafely(ctx, "TG-Group:2020", permit) || len(adapter.quitGroups) != 2 {
		t.Fatal("other platforms must retain manual quit support")
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
			emitter := &groupQuitTestEmitter{onebotTestEmitter: baseEmitter, role: "owner"}
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
