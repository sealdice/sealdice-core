package emitter

// fork from https://github.com/nsxdevx/nsxbot

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"

	socketio "github.com/PaienNate/pineutil/evsocket"

	"sealdice-core/dice/imsdk/onebot/schema"
	"sealdice-core/dice/imsdk/onebot/types"
)

// EchoTimeOut ws onebot await echo message time out
var EchoTimeOut = 10 * time.Second

type Emitter interface {
	SendPvtMsg(ctx context.Context, userId int64, msg schema.MessageChain) (*types.SendMsgRes, error)
	SendGrMsg(ctx context.Context, groupId int64, msg schema.MessageChain) (*types.SendMsgRes, error)
	GetMsg(ctx context.Context, msgId int) (*types.GetMsgRes, error)
	DelMsg(ctx context.Context, msgId int) error
	GetLoginInfo(ctx context.Context) (*types.LoginInfo, error)
	GetStrangerInfo(ctx context.Context, userId int64, noCache bool) (*types.StrangerInfo, error)
	GetStatus(ctx context.Context) (*types.Status, error)
	GetVersionInfo(ctx context.Context) (*types.VersionInfo, error)
	GetSelfId(ctx context.Context) (int64, error)
	SetSelfId(ctx context.Context, selfId int64) error
	SetFriendAddRequest(ctx context.Context, flag string, approve bool, remark string) error
	SetGroupAddRequest(ctx context.Context, flag string, approve bool, reason string) error
	SetGroupSpecialTitle(ctx context.Context, groupId int64, userId int64, specialTitle string, duration int) error

	// 并非Onebot11大典的逻辑，是补充逻辑

	QuitGroup(ctx context.Context, groupId int64) error
	SetGroupCard(ctx context.Context, groupId int64, userId int64, card string) error
	GetGroupInfo(ctx context.Context, groupId int64, noCache bool) (*types.GroupInfo, error)
	GetGroupMemberInfo(ctx context.Context, groupId int64, userId int64, noCache bool) (*types.GroupMemberInfo, error)
	Raw(ctx context.Context, action Action, params any) ([]byte, error)

	HandleEcho(resp Response[sonic.NoCopyRawMessage])

	GetDroppedEchoCount() uint64
}

var _ Emitter = (*emitterSocket)(nil)

type Request[T any] struct {
	Echo   string `json:"echo"`
	Action Action `json:"action"`
	Params T      `json:"params,omitempty"`
}

type Response[T any] struct {
	Status  string `json:"status"`
	RetCode int    `json:"retCode"`
	Data    T      `json:"data,omitempty"`
	Echo    string `json:"echo"`
}

type emitterSocket struct {
	mu     sync.Mutex
	conn   *socketio.WebsocketWrapper
	selfId int64

	waiters sync.Map // map[string]chan Response[sonic.NoCopyRawMessage]

	droppedEchoCount uint64
}

func NewEVEmitter(conn *socketio.WebsocketWrapper) *emitterSocket {
	emitter := &emitterSocket{
		conn: conn,
	}
	return emitter
}

func (e *emitterSocket) HandleEcho(resp Response[sonic.NoCopyRawMessage]) {
	if resp.Echo == "" {
		atomic.AddUint64(&e.droppedEchoCount, 1)
		return
	}
	chAny, ok := e.waiters.Load(resp.Echo)
	if !ok {
		atomic.AddUint64(&e.droppedEchoCount, 1)
		return
	}
	ch, ok := chAny.(chan Response[sonic.NoCopyRawMessage])
	if !ok {
		atomic.AddUint64(&e.droppedEchoCount, 1)
		return
	}
	select {
	case ch <- resp:
	default:
		atomic.AddUint64(&e.droppedEchoCount, 1)
	}
}

func (e *emitterSocket) GetDroppedEchoCount() uint64 {
	return atomic.LoadUint64(&e.droppedEchoCount)
}

func (e *emitterSocket) SetSelfId(_ context.Context, selfId int64) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.selfId = selfId
	return nil
}

func (e *emitterSocket) waitEchoAfterSend(ctx context.Context, echoId string, send func() error) (Response[sonic.NoCopyRawMessage], error) {
	ctx, cancel := context.WithTimeout(ctx, EchoTimeOut)
	defer cancel()

	ch := make(chan Response[sonic.NoCopyRawMessage], 1)
	e.waiters.Store(echoId, ch)
	defer e.waiters.Delete(echoId)

	if err := send(); err != nil {
		return Response[sonic.NoCopyRawMessage]{}, err
	}

	select {
	case <-ctx.Done():
		return Response[sonic.NoCopyRawMessage]{}, ctx.Err()
	case resp := <-ch:
		return resp, nil
	}
}

func decodeResponse[R any](resp Response[sonic.NoCopyRawMessage]) (*R, error) {
	if strings.EqualFold("failed", resp.Status) {
		return nil, fmt.Errorf("action failed, status=%s retcode=%d", resp.Status, resp.RetCode)
	}
	var res R
	if err := sonic.Unmarshal(resp.Data, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (e *emitterSocket) SendPvtMsg(ctx context.Context, userId int64, msg schema.MessageChain) (*types.SendMsgRes, error) {
	resp, err := doAction(ctx, e, ACTION_SEND_PRIVATE_MSG, types.SendPrivateMsgReq{
		UserId:  userId,
		Message: msg,
	})
	if err != nil {
		return nil, err
	}
	return decodeResponse[types.SendMsgRes](resp)
}

func (e *emitterSocket) SendGrMsg(ctx context.Context, groupId int64, msg schema.MessageChain) (*types.SendMsgRes, error) {
	resp, err := doAction(ctx, e, ACTION_SEND_GROUP_MSG, types.SendGrMsgReq{
		GroupId: groupId,
		Message: msg,
	})
	if err != nil {
		return nil, err
	}
	return decodeResponse[types.SendMsgRes](resp)
}

func (e *emitterSocket) GetMsg(ctx context.Context, msgId int) (*types.GetMsgRes, error) {
	resp, err := doAction(ctx, e, ACTION_GET_MSG, types.GetMsgReq{
		MessageId: msgId,
	})
	if err != nil {
		return nil, err
	}
	return decodeResponse[types.GetMsgRes](resp)
}

func (e *emitterSocket) DelMsg(ctx context.Context, msgId int) error {
	resp, err := doAction(ctx, e, ACTION_DELETE_MSG, types.DelMsgReq{
		MessageId: msgId,
	})
	if err != nil {
		return err
	}
	_, err = decodeResponse[any](resp)
	return err
}

func (e *emitterSocket) GetLoginInfo(ctx context.Context) (*types.LoginInfo, error) {
	resp, err := doAction(ctx, e, ACTION_GET_LOGIN_INFO, nil)
	if err != nil {
		return nil, err
	}
	return decodeResponse[types.LoginInfo](resp)
}

func (e *emitterSocket) GetStrangerInfo(ctx context.Context, userId int64, noCache bool) (*types.StrangerInfo, error) {
	resp, err := doAction(ctx, e, ACTION_GET_STRANGER_INFO, types.GetStrangerInfo{
		UserId:  userId,
		NoCache: noCache,
	})
	if err != nil {
		return nil, err
	}
	return decodeResponse[types.StrangerInfo](resp)
}

func (e *emitterSocket) GetStatus(ctx context.Context) (*types.Status, error) {
	resp, err := doAction(ctx, e, ACTION_GET_STATUS, nil)
	if err != nil {
		return nil, err
	}
	return decodeResponse[types.Status](resp)
}

func (e *emitterSocket) GetVersionInfo(ctx context.Context) (*types.VersionInfo, error) {
	resp, err := doAction(ctx, e, ACTION_GET_VERSION_INFO, nil)
	if err != nil {
		return nil, err
	}
	return decodeResponse[types.VersionInfo](resp)
}

func (e *emitterSocket) GetSelfId(_ context.Context) (int64, error) {
	return e.selfId, nil
}

func (e *emitterSocket) SetFriendAddRequest(ctx context.Context, flag string, approve bool, remark string) error {
	resp, err := doAction(ctx, e, ACTION_SET_FRIEND_ADD_REQUEST, types.FriendAddReq{
		Flag:    flag,
		Approve: approve,
		Remark:  remark,
	})
	if err != nil {
		return err
	}
	_, err = decodeResponse[any](resp)
	return err
}

func (e *emitterSocket) SetGroupAddRequest(ctx context.Context, flag string, approve bool, reason string) error {
	resp, err := doAction(ctx, e, ACTION_SET_GROUP_ADD_REQUEST, types.GroupAddReq{
		Flag:    flag,
		Approve: approve,
		Reason:  reason,
	})
	if err != nil {
		return err
	}
	_, err = decodeResponse[any](resp)
	return err
}

func (e *emitterSocket) SetGroupSpecialTitle(ctx context.Context, groupId int64, userId int64, specialTitle string, duration int) error {
	resp, err := doAction(ctx, e, ACTION_SET_GROUP_SPECIAL_TITLE, types.SpecialTitleReq{
		GroupId:      groupId,
		UserId:       userId,
		SpecialTitle: specialTitle,
	})
	if err != nil {
		return err
	}
	_, err = decodeResponse[any](resp)
	return err
}

// ADD 不存在于Onebot大典的内容

func (e *emitterSocket) QuitGroup(ctx context.Context, groupId int64) error {
	resp, err := doAction(ctx, e, ACTION_QUIT_GROUP, types.QuitGroupReq{
		GroupId: groupId,
	})
	if err != nil {
		return err
	}
	_, err = decodeResponse[any](resp)
	return err
}

func (e *emitterSocket) SetGroupCard(ctx context.Context, groupId int64, userId int64, card string) error {
	resp, err := doAction(ctx, e, ACTION_SET_GROUP_CARD, types.SetGroupCardReq{
		GroupId: groupId,
		UserId:  userId,
		Card:    card,
	})
	if err != nil {
		return err
	}
	_, err = decodeResponse[any](resp)
	return err
}

func (e *emitterSocket) GetGroupInfo(ctx context.Context, groupId int64, noCache bool) (*types.GroupInfo, error) {
	resp, err := doAction(ctx, e, ACTION_GET_GROUP_INFO, types.GetGroupInfoReq{
		GroupId: groupId,
		NoCache: noCache,
	})
	if err != nil {
		return nil, err
	}
	return decodeResponse[types.GroupInfo](resp)
}

func (e *emitterSocket) GetGroupMemberInfo(ctx context.Context, groupId int64, userId int64, noCache bool) (*types.GroupMemberInfo, error) {
	resp, err := doAction(ctx, e, ACTION_GET_GROUP_MEMBER_INFO, types.GetGroupMemberInfoReq{
		GroupId: groupId,
		UserId:  userId,
		NoCache: noCache,
	})
	if err != nil {
		return nil, err
	}
	return decodeResponse[types.GroupMemberInfo](resp)
}

func (e *emitterSocket) Raw(ctx context.Context, action Action, params any) ([]byte, error) {
	resp, err := doAction(ctx, e, action, params)
	if err != nil {
		return nil, err
	}
	return sonic.Marshal(resp)
}

func doAction(ctx context.Context, e *emitterSocket, action string, params any) (Response[sonic.NoCopyRawMessage], error) {
	echoId := uuid.New().String()
	resp, err := e.waitEchoAfterSend(ctx, echoId, func() error {
		e.mu.Lock()
		defer e.mu.Unlock()
		return wsEmitWithEcho(e.conn, action, params, echoId)
	})
	if err != nil {
		return Response[sonic.NoCopyRawMessage]{}, err
	}
	return resp, nil
}

func wsEmitWithEcho(w *socketio.WebsocketWrapper, action string, params any, echoId string) error {
	marshal, err := sonic.Marshal(Request[any]{
		Action: action,
		Echo:   echoId,
		Params: params,
	})
	if err != nil {
		return err
	}
	w.Emit(marshal, socketio.TextMessage)
	return nil
}
