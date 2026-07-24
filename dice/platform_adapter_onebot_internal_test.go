//nolint:testpackage
package dice

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	socketio "github.com/PaienNate/pineutil/evsocket/v2"
	"github.com/bytedance/sonic"
	loopfsm "github.com/looplab/fsm"
	"github.com/maypok86/otter"
	"github.com/panjf2000/ants/v2"
	"github.com/tidwall/gjson"

	emitter "sealdice-core/dice/imsdk/onebot"
	"sealdice-core/dice/imsdk/onebot/schema"
	emitterTypes "sealdice-core/dice/imsdk/onebot/types"
)

type onebotTestEmitter struct {
	friendReqCalls []friendReqCall
	groupReqCalls  []groupReqCall
	groupInfo      *emitterTypes.GroupInfo
	groupInfoErr   error
	groupReqCh     chan struct{}
	loginInfo      *emitterTypes.LoginInfo
	loginInfoErr   error
	sendPvtCh      chan time.Time
}

type friendReqCall struct {
	Flag    string
	Approve bool
	Remark  string
}

type groupReqCall struct {
	Flag    string
	Approve bool
	Reason  string
}

var _ emitter.Emitter = (*onebotTestEmitter)(nil)

func (m *onebotTestEmitter) SendPvtMsg(context.Context, int64, schema.MessageChain) (*emitterTypes.SendMsgRes, error) {
	if m.sendPvtCh != nil {
		select {
		case m.sendPvtCh <- time.Now():
		default:
		}
	}
	return &emitterTypes.SendMsgRes{}, nil
}

func (m *onebotTestEmitter) SendGrMsg(context.Context, int64, schema.MessageChain) (*emitterTypes.SendMsgRes, error) {
	return &emitterTypes.SendMsgRes{}, nil
}

func (m *onebotTestEmitter) GetMsg(context.Context, int) (*emitterTypes.GetMsgRes, error) {
	return &emitterTypes.GetMsgRes{}, nil
}

func (m *onebotTestEmitter) DelMsg(context.Context, int) error { return nil }

func (m *onebotTestEmitter) GetLoginInfo(context.Context) (*emitterTypes.LoginInfo, error) {
	if m.loginInfoErr != nil {
		return nil, m.loginInfoErr
	}
	if m.loginInfo != nil {
		return m.loginInfo, nil
	}
	return &emitterTypes.LoginInfo{}, nil
}

func (m *onebotTestEmitter) GetStrangerInfo(context.Context, int64, bool) (*emitterTypes.StrangerInfo, error) {
	return &emitterTypes.StrangerInfo{}, nil
}

func (m *onebotTestEmitter) GetStatus(context.Context) (*emitterTypes.Status, error) {
	return &emitterTypes.Status{}, nil
}

func (m *onebotTestEmitter) GetVersionInfo(context.Context) (*emitterTypes.VersionInfo, error) {
	return &emitterTypes.VersionInfo{}, nil
}

func (m *onebotTestEmitter) GetSelfId(context.Context) (int64, error) { return 0, nil }

func (m *onebotTestEmitter) SetSelfId(context.Context, int64) error { return nil }

func (m *onebotTestEmitter) SetFriendAddRequest(_ context.Context, flag string, approve bool, remark string) error {
	m.friendReqCalls = append(m.friendReqCalls, friendReqCall{
		Flag:    flag,
		Approve: approve,
		Remark:  remark,
	})
	return nil
}

func (m *onebotTestEmitter) SetGroupAddRequest(_ context.Context, flag string, approve bool, reason string) error {
	m.groupReqCalls = append(m.groupReqCalls, groupReqCall{
		Flag:    flag,
		Approve: approve,
		Reason:  reason,
	})
	if m.groupReqCh != nil {
		select {
		case m.groupReqCh <- struct{}{}:
		default:
		}
	}
	return nil
}

func (m *onebotTestEmitter) SetGroupSpecialTitle(context.Context, int64, int64, string, int) error {
	return nil
}

func (m *onebotTestEmitter) QuitGroup(context.Context, int64) error { return nil }

func (m *onebotTestEmitter) SetGroupCard(context.Context, int64, int64, string) error { return nil }

func (m *onebotTestEmitter) GetGroupInfo(context.Context, int64, bool) (*emitterTypes.GroupInfo, error) {
	if m.groupInfoErr != nil {
		return nil, m.groupInfoErr
	}
	if m.groupInfo == nil {
		return &emitterTypes.GroupInfo{}, nil
	}
	return m.groupInfo, nil
}

func (m *onebotTestEmitter) GetGroupMemberInfo(context.Context, int64, int64, bool) (*emitterTypes.GroupMemberInfo, error) {
	return &emitterTypes.GroupMemberInfo{}, nil
}

func (m *onebotTestEmitter) Raw(context.Context, emitter.Action, any) ([]byte, error) {
	return []byte{}, nil
}

func (m *onebotTestEmitter) HandleEcho(emitter.Response[sonic.NoCopyRawMessage]) {}

func (m *onebotTestEmitter) GetDroppedEchoCount() uint64 { return 0 }

func newPureOnebotTestAdapter(t *testing.T) (*Dice, *PlatformAdapterOnebot, *onebotTestEmitter, func()) {
	t.Helper()

	d, ep, _, cleanup := newExecuteNewTestDice(t)
	d.ExtList = nil
	pa := &PlatformAdapterOnebot{
		EndPoint: ep,
		ctx:      t.Context(),
		logger:   d.Logger,
	}
	pool, err := ants.NewPool(8)
	if err != nil {
		cleanup()
		t.Fatalf("create ants pool: %v", err)
	}
	pa.antPool = pool
	cache, err := otter.MustBuilder[string, *GroupCache](16).Build()
	if err != nil {
		pool.Release()
		cleanup()
		t.Fatalf("create group cache: %v", err)
	}
	pa.groupCache = &cache
	em := &onebotTestEmitter{
		groupReqCh: make(chan struct{}, 8),
		sendPvtCh:  make(chan time.Time, 8),
	}
	pa.sendEmitter = em

	ep.Platform = "QQ"
	ep.ProtocolType = "pureonebot"
	ep.Adapter = pa
	ep.Session = d.ImSession

	cleanupAll := func() {
		if pa.groupCache != nil {
			pa.groupCache.Close()
		}
		pool.Release()
		cleanup()
	}

	return d, pa, em, cleanupAll
}

func newPureOnebotRetryFSM(eventCh chan string) *loopfsm.FSM {
	return loopfsm.NewFSM(
		"connecting",
		loopfsm.Events{
			{Name: "connect_ok", Src: []string{"connecting"}, Dst: "connected"},
			{Name: "connect_fail", Src: []string{"connecting"}, Dst: "failed"},
		},
		loopfsm.Callbacks{
			"enter_connected": func(_ context.Context, _ *loopfsm.Event) {
				select {
				case eventCh <- "connect_ok":
				default:
				}
			},
			"enter_failed": func(_ context.Context, _ *loopfsm.Event) {
				select {
				case eventCh <- "connect_fail":
				default:
				}
			},
		},
	)
}

func TestPureOnebotFriendRequestUsesCanonicalUserIDForBlacklist(t *testing.T) {
	d, pa, em, cleanup := newPureOnebotTestAdapter(t)
	defer cleanup()

	d.Config.BanList.Map.Store("QQ:12345", &BanListInfoItem{
		ID:   "QQ:12345",
		Rank: BanRankBanned,
	})

	req := gjson.Parse(`{
		"post_type":"request",
		"request_type":"friend",
		"flag":"friend-flag",
		"user_id":"12345",
		"comment":"hi"
	}`)

	if err := pa.handleReqFriendAction(req, nil); err != nil {
		t.Fatalf("handleReqFriendAction returned error: %v", err)
	}

	if len(em.friendReqCalls) != 1 {
		t.Fatalf("expected one friend request action, got %d", len(em.friendReqCalls))
	}
	if em.friendReqCalls[0].Approve {
		t.Fatalf("expected banned inviter to be rejected, got %#v", em.friendReqCalls[0])
	}
}

func TestPureOnebotGroupInviteRejectStopsWithoutApprove(t *testing.T) {
	_, pa, em, cleanup := newPureOnebotTestAdapter(t)
	defer cleanup()

	req := gjson.Parse(`{
		"post_type":"request",
		"request_type":"group",
		"sub_type":"invite",
		"flag":"group-flag",
		"user_id":"55555",
		"group_id":"66666"
	}`)

	pa.EndPoint.Session.Parent.Config.RefuseGroupInvite = true

	if err := pa.handleReqGroupAction(req, nil); err != nil {
		t.Fatalf("handleReqGroupAction returned error: %v", err)
	}

	select {
	case <-em.groupReqCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for group request action")
	}

	if len(em.groupReqCalls) != 1 {
		t.Fatalf("expected one group request action, got %d", len(em.groupReqCalls))
	}
	if em.groupReqCalls[0].Approve {
		t.Fatalf("expected reject only, got %#v", em.groupReqCalls[0])
	}
}

func TestPureOnebotCheckPassBlackListGroupUsesInviterForTrustOnlyMode(t *testing.T) {
	d, _, _, cleanup := newPureOnebotTestAdapter(t)
	defer cleanup()

	ctx := &MsgContext{Dice: d}
	d.Config.TrustOnlyMode = true
	d.DiceMasters = []string{"QQ:55555"}
	d.Config.BanList.Map.Store("QQ-Group:66666", &BanListInfoItem{
		ID:   "QQ-Group:66666",
		Rank: BanRankNormal,
	})

	ok, reason := checkPassBlackListGroup("QQ:55555", "QQ-Group:66666", ctx)
	if !ok {
		t.Fatalf("expected master inviter to pass trust-only mode, got reason=%q", reason)
	}
}

func TestPureOnebotCheckBlackListUserRejectsBannedMaster(t *testing.T) {
	d, _, _, cleanup := newPureOnebotTestAdapter(t)
	defer cleanup()

	d.DiceMasters = []string{"QQ:55555"}
	d.Config.BanList.Map.Store("QQ:55555", &BanListInfoItem{
		ID:   "QQ:55555",
		Rank: BanRankBanned,
	})

	result := checkBlackList("QQ:55555", "user", "", &MsgContext{Dice: d})
	if result.Passed {
		t.Fatalf("expected banned master user check to be rejected, got %#v", result)
	}
	if result.Reason != "邀请人在黑名单上" {
		t.Fatalf("expected banned master reject reason, got %#v", result)
	}
}

func TestPureOnebotGetGroupInfoSyncUsesGroupBanInfo(t *testing.T) {
	d, pa, em, cleanup := newPureOnebotTestAdapter(t)
	defer cleanup()

	groupID := "QQ-Group:66666"
	group := &GroupInfo{
		GroupID:         groupID,
		GroupName:       "group",
		InviteUserID:    "QQ:55555",
		EnteredTime:     123,
		DiceIDExistsMap: new(SyncMap[string, bool]),
	}
	group.DiceIDExistsMap.Store(pa.EndPoint.UserID, true)
	d.ImSession.ServiceAtNew.Store(groupID, group)

	em.groupInfo = &emitterTypes.GroupInfo{
		GroupId:        66666,
		GroupName:      "group",
		MemberCount:    10,
		MaxMemberCount: 100,
	}

	d.Config.BanList.Map.Store(groupID, &BanListInfoItem{
		ID:      groupID,
		Rank:    BanRankBanned,
		Reasons: []string{"group banned"},
	})

	if got := pa.GetGroupInfoSync(groupID); got == nil {
		t.Fatalf("expected group info, got nil")
	}
}

func TestPureOnebotGroupInviteTrustOnlyNonMasterInviterRejected(t *testing.T) {
	d, pa, em, cleanup := newPureOnebotTestAdapter(t)
	defer cleanup()

	d.Config.TrustOnlyMode = true
	d.DiceMasters = []string{"QQ:99999"}
	d.Config.BanList.Map.Delete("QQ-Group:123456")

	req := gjson.Parse(`{
		"post_type":"request",
		"request_type":"group",
		"sub_type":"invite",
		"group_id":123456,
		"user_id":11111,
		"flag":"group-invite-flag"
	}`)

	if err := pa.handleReqGroupAction(req, nil); err != nil {
		t.Fatalf("handleReqGroupAction returned error: %v", err)
	}

	select {
	case <-em.groupReqCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for group request action")
	}

	if len(em.groupReqCalls) != 1 {
		t.Fatalf("expected one group request action, got %d", len(em.groupReqCalls))
	}

	call := em.groupReqCalls[0]
	if call.Approve {
		t.Fatalf("expected non-master inviter in trust-only mode to be rejected, got %#v", call)
	}
	if call.Flag != "group-invite-flag" {
		t.Fatalf("unexpected flag in groupReqCall, expected %q, got %q", "group-invite-flag", call.Flag)
	}
	if call.Reason != "只允许信任的人拉群" {
		t.Fatalf("expected trust-only reject reason, got %#v", call)
	}
}

func TestPureOnebotApplyClientAuthHeaderPreservesConfiguredToken(t *testing.T) {
	_, pa, _, cleanup := newPureOnebotTestAdapter(t)
	defer cleanup()

	pa.ConnectURL = "ws://127.0.0.1:12345"
	pa.Token = "test-token"
	options := socketio.ClientOptions{RequestHeader: http.Header{}}
	pa.applyClientAuthHeader(&options)
	if got := options.RequestHeader.Get("Authorization"); got != "test-token" {
		t.Fatalf("expected raw token header, got %q", got)
	}
}

func TestPureOnebotApplyClientAuthHeaderKeepsExplicitBearerScheme(t *testing.T) {
	_, pa, _, cleanup := newPureOnebotTestAdapter(t)
	defer cleanup()

	pa.ConnectURL = "ws://127.0.0.1:12345"
	pa.Token = "Bearer test-token"
	options := socketio.ClientOptions{RequestHeader: http.Header{}}
	pa.applyClientAuthHeader(&options)
	if got := options.RequestHeader.Get("Authorization"); got != "Bearer test-token" {
		t.Fatalf("expected explicit bearer header to be preserved, got %q", got)
	}
}

func TestPureOnebotAuthorizationHeaderMatchesConfiguredToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		config    string
		header    string
		wantMatch bool
	}{
		{
			name:      "empty config and empty header",
			config:    "",
			header:    "",
			wantMatch: true,
		},
		{
			name:      "empty config and non-empty header",
			config:    "",
			header:    "some-token",
			wantMatch: false,
		},
		{
			name:      "raw token matches raw header",
			config:    "test-token",
			header:    "test-token",
			wantMatch: true,
		},
		{
			name:      "raw token matches bearer header",
			config:    "test-token",
			header:    "Bearer test-token",
			wantMatch: true,
		},
		{
			name:      "bearer token matches raw header",
			config:    "Bearer test-token",
			header:    "test-token",
			wantMatch: true,
		},
		{
			name:      "bearer token matches bearer header",
			config:    "Bearer test-token",
			header:    "Bearer test-token",
			wantMatch: true,
		},
		{
			name:      "different token does not match",
			config:    "test-token",
			header:    "other-token",
			wantMatch: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := onebotAuthorizationMatches(tc.config, tc.header)
			if got != tc.wantMatch {
				t.Fatalf("onebotAuthorizationMatches(%q, %q) = %v, want %v", tc.config, tc.header, got, tc.wantMatch)
			}
		})
	}
}

func TestPureOnebotScheduleLoginInfoRetryInvokesCustomRetry(t *testing.T) {
	_, pa, _, cleanup := newPureOnebotTestAdapter(t)
	defer cleanup()

	calledCh := make(chan struct{}, 1)
	pa.loginInitRetry = func() {
		select {
		case calledCh <- struct{}{}:
		default:
		}
	}

	pa.scheduleLoginInfoRetry()

	select {
	case <-calledCh:
	case <-time.After(1 * time.Second):
		t.Fatal("expected custom loginInitRetry to be invoked")
	}
}

func TestPureOnebotScheduleLoginInfoRetryEmitsConnectFailOnLoginInfoError(t *testing.T) {
	_, pa, em, cleanup := newPureOnebotTestAdapter(t)
	defer cleanup()

	events := make(chan string, 2)
	sleepDurations := make(chan time.Duration, 1)
	pa.desiredEnabled = true
	pa.sm = newPureOnebotRetryFSM(events)
	pa.loginInitRetrySleep = func(delay time.Duration) {
		select {
		case sleepDurations <- delay:
		default:
		}
	}
	em.loginInfoErr = errors.New("boom")

	pa.scheduleLoginInfoRetry()

	select {
	case delay := <-sleepDurations:
		if delay != 3*time.Second {
			t.Fatalf("expected retry delay 3s, got %v", delay)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("expected retry sleep hook to be invoked")
	}

	select {
	case event := <-events:
		if event != "connect_fail" {
			t.Fatalf("expected connect_fail event, got %q", event)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("expected connect_fail event")
	}

	if got := pa.sm.Current(); got != "failed" {
		t.Fatalf("expected FSM to transition to failed, got %q", got)
	}
}

func TestPureOnebotScheduleLoginInfoRetryEmitsConnectOkAndSetsEndpoint(t *testing.T) {
	_, pa, em, cleanup := newPureOnebotTestAdapter(t)
	defer cleanup()

	events := make(chan string, 2)
	sleepDurations := make(chan time.Duration, 1)
	pa.desiredEnabled = true
	pa.sm = newPureOnebotRetryFSM(events)
	pa.loginInitRetrySleep = func(delay time.Duration) {
		select {
		case sleepDurations <- delay:
		default:
		}
	}
	em.loginInfo = &emitterTypes.LoginInfo{
		UserId:   123456,
		NickName: "test-nick",
	}

	pa.scheduleLoginInfoRetry()

	select {
	case delay := <-sleepDurations:
		if delay != 3*time.Second {
			t.Fatalf("expected retry delay 3s, got %v", delay)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("expected retry sleep hook to be invoked")
	}

	select {
	case event := <-events:
		if event != "connect_ok" {
			t.Fatalf("expected connect_ok event, got %q", event)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("expected connect_ok event")
	}

	if got := pa.sm.Current(); got != "connected" {
		t.Fatalf("expected FSM to transition to connected, got %q", got)
	}
	if pa.EndPoint.UserID != "QQ:123456" {
		t.Fatalf("expected EndPoint.UserID to be set, got %q", pa.EndPoint.UserID)
	}
	if pa.EndPoint.Nickname != "test-nick" {
		t.Fatalf("expected EndPoint.Nickname to be set, got %q", pa.EndPoint.Nickname)
	}
}

func TestPureOnebotScheduleLoginInfoRetryGuardsBeforeSleeping(t *testing.T) {
	tests := []struct {
		name      string
		configure func(pa *PlatformAdapterOnebot)
	}{
		{
			name: "disabled adapter",
			configure: func(pa *PlatformAdapterOnebot) {
				pa.desiredEnabled = false
			},
		},
		{
			name: "nil emitter",
			configure: func(pa *PlatformAdapterOnebot) {
				pa.sendEmitter = nil
			},
		},
		{
			name: "nil ctx",
			configure: func(pa *PlatformAdapterOnebot) {
				pa.ctx = nil
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, pa, _, cleanup := newPureOnebotTestAdapter(t)
			defer cleanup()

			events := make(chan string, 1)
			sleepDurations := make(chan time.Duration, 1)
			pa.desiredEnabled = true
			pa.sm = newPureOnebotRetryFSM(events)
			pa.loginInitRetrySleep = func(delay time.Duration) {
				select {
				case sleepDurations <- delay:
				default:
				}
			}
			tc.configure(pa)

			pa.scheduleLoginInfoRetry()

			select {
			case delay := <-sleepDurations:
				t.Fatalf("expected guard to stop retry before sleeping, got sleep duration %v", delay)
			case <-time.After(100 * time.Millisecond):
			}

			select {
			case event := <-events:
				t.Fatalf("expected no FSM event, got %q", event)
			case <-time.After(100 * time.Millisecond):
			}

			if got := pa.sm.Current(); got != "connecting" {
				t.Fatalf("expected FSM to remain connecting, got %q", got)
			}
		})
	}
}

func TestPureOnebotScheduleLoginInfoRetryRechecksGuardsAfterSleeping(t *testing.T) {
	_, pa, em, cleanup := newPureOnebotTestAdapter(t)
	defer cleanup()

	events := make(chan string, 1)
	sleepDurations := make(chan time.Duration, 1)
	pa.desiredEnabled = true
	pa.sm = newPureOnebotRetryFSM(events)
	pa.loginInitRetrySleep = func(delay time.Duration) {
		pa.desiredEnabled = false
		select {
		case sleepDurations <- delay:
		default:
		}
	}
	em.loginInfo = &emitterTypes.LoginInfo{
		UserId:   123456,
		NickName: "test-nick",
	}
	originalUserID := pa.EndPoint.UserID
	originalNickname := pa.EndPoint.Nickname

	pa.scheduleLoginInfoRetry()

	select {
	case delay := <-sleepDurations:
		if delay != 3*time.Second {
			t.Fatalf("expected retry delay 3s, got %v", delay)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("expected retry sleep hook to be invoked")
	}

	select {
	case event := <-events:
		t.Fatalf("expected no FSM event after desiredEnabled changed during sleep, got %q", event)
	case <-time.After(100 * time.Millisecond):
	}

	if got := pa.sm.Current(); got != "connecting" {
		t.Fatalf("expected FSM to remain connecting, got %q", got)
	}
	if pa.EndPoint.UserID != originalUserID {
		t.Fatalf("expected EndPoint.UserID to remain %q, got %q", originalUserID, pa.EndPoint.UserID)
	}
	if pa.EndPoint.Nickname != originalNickname {
		t.Fatalf("expected EndPoint.Nickname to remain %q, got %q", originalNickname, pa.EndPoint.Nickname)
	}
}

func TestPureOnebotHandleJoinGroupStoresInviterForSelfJoin(t *testing.T) {
	_, pa, _, cleanup := newPureOnebotTestAdapter(t)
	defer cleanup()

	req := gjson.Parse(`{
		"post_type":"notice",
		"notice_type":"group_increase",
		"group_id":"66666",
		"user_id":"54321",
		"self_id":"54321",
		"operator_id":"12345",
		"time": 1,
		"message": []
	}`)

	if err := pa.handleJoinGroupAction(req, nil); err != nil {
		t.Fatalf("handleJoinGroupAction returned error: %v", err)
	}

	group, ok := pa.EndPoint.Session.ServiceAtNew.Load("QQ-Group:66666")
	if !ok {
		t.Fatalf("expected group to be initialized")
	}
	if group.InviteUserID != "QQ:12345" {
		t.Fatalf("expected inviter to be preserved, got %q", group.InviteUserID)
	}
}
