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
	if chAny, ok := e.waiters.Load(resp.Echo); ok {
		ch := chAny.(chan Response[sonic.NoCopyRawMessage])
		select {
		case ch <- resp:
		default:
		}
		return
	}
	atomic.AddUint64(&e.droppedEchoCount, 1)
}

func (e *emitterSocket) SetSelfId(_ context.Context, selfId int64) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.selfId = selfId
	return nil
}

func (e *emitterSocket) waitEcho(ctx context.Context, echoId string) (Response[sonic.NoCopyRawMessage], error) {
	ctx, cancel := context.WithTimeout(ctx, EchoTimeOut)
	defer cancel()

	ch := make(chan Response[sonic.NoCopyRawMessage], 1)
	e.waiters.Store(echoId, ch)
	defer e.waiters.Delete(echoId)

	select {
	case <-ctx.Done():
		return Response[sonic.NoCopyRawMessage]{}, ctx.Err()
	case resp := <-ch:
		return resp, nil
	}
}

func waitAndDecode[R any](ctx context.Context, e *emitterSocket, echoId string) (*R, error) {
	resp, err := e.waitEcho(ctx, echoId)
	if err != nil {
		return nil, err
	}
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
	e.mu.Lock()
	echoId, err := wsAction(e.conn, ACTION_SEND_PRIVATE_MSG, types.SendPrivateMsgReq{
		UserId:  userId,
		Message: msg,
	})
	if err != nil {
		e.mu.Unlock()
		return nil, err
	}
	e.mu.Unlock()
	return waitAndDecode[types.SendMsgRes](ctx, e, echoId)
}

func (e *emitterSocket) SendGrMsg(ctx context.Context, groupId int64, msg schema.MessageChain) (*types.SendMsgRes, error) {
	e.mu.Lock()
	echoId, err := wsAction(e.conn, ACTION_SEND_GROUP_MSG, types.SendGrMsgReq{
		GroupId: groupId,
		Message: msg,
	})
	if err != nil {
		e.mu.Unlock()
		return nil, err
	}
	e.mu.Unlock()
	return waitAndDecode[types.SendMsgRes](ctx, e, echoId)
}

func (e *emitterSocket) GetMsg(ctx context.Context, msgId int) (*types.GetMsgRes, error) {
	e.mu.Lock()
	echoId, err := wsAction(e.conn, ACTION_GET_MSG, types.GetMsgReq{
		MessageId: msgId,
	})
	if err != nil {
		e.mu.Unlock()
		return nil, err
	}
	e.mu.Unlock()
	return waitAndDecode[types.GetMsgRes](ctx, e, echoId)
}

func (e *emitterSocket) DelMsg(ctx context.Context, msgId int) error {
	e.mu.Lock()
	echoId, err := wsAction[any](e.conn, ACTION_DELETE_MSG, types.DelMsgReq{
		MessageId: msgId,
	})
	if err != nil {
		e.mu.Unlock()
		return err
	}
	e.mu.Unlock()
	_, err = waitAndDecode[any](ctx, e, echoId)
	return err
}

func (e *emitterSocket) GetLoginInfo(ctx context.Context) (*types.LoginInfo, error) {
	e.mu.Lock()
	echoId, err := wsAction[any](e.conn, ACTION_GET_LOGIN_INFO, nil)
	if err != nil {
		e.mu.Unlock()
		return nil, err
	}
	e.mu.Unlock()
	return waitAndDecode[types.LoginInfo](ctx, e, echoId)
}

func (e *emitterSocket) GetStrangerInfo(ctx context.Context, userId int64, noCache bool) (*types.StrangerInfo, error) {
	e.mu.Lock()
	echoId, err := wsAction(e.conn, ACTION_GET_STRANGER_INFO, types.GetStrangerInfo{
		UserId:  userId,
		NoCache: noCache,
	})
	if err != nil {
		e.mu.Unlock()
		return nil, err
	}
	e.mu.Unlock()
	return waitAndDecode[types.StrangerInfo](ctx, e, echoId)
}

func (e *emitterSocket) GetStatus(ctx context.Context) (*types.Status, error) {
	e.mu.Lock()
	echoId, err := wsAction[any](e.conn, ACTION_GET_STATUS, nil)
	if err != nil {
		e.mu.Unlock()
		return nil, err
	}
	e.mu.Unlock()
	return waitAndDecode[types.Status](ctx, e, echoId)
}

func (e *emitterSocket) GetVersionInfo(ctx context.Context) (*types.VersionInfo, error) {
	e.mu.Lock()
	echoId, err := wsAction[any](e.conn, ACTION_GET_VERSION_INFO, nil)
	if err != nil {
		e.mu.Unlock()
		return nil, err
	}
	e.mu.Unlock()
	return waitAndDecode[types.VersionInfo](ctx, e, echoId)
}

func (e *emitterSocket) GetSelfId(_ context.Context) (int64, error) {
	return e.selfId, nil
}

func (e *emitterSocket) SetFriendAddRequest(ctx context.Context, flag string, approve bool, remark string) error {
	e.mu.Lock()
	echoId, err := wsAction(e.conn, ACTION_SET_FRIEND_ADD_REQUEST, types.FriendAddReq{
		Flag:    flag,
		Approve: approve,
		Remark:  remark,
	})
	if err != nil {
		e.mu.Unlock()
		return err
	}
	e.mu.Unlock()
	_, err = waitAndDecode[any](ctx, e, echoId)
	return err
}

func (e *emitterSocket) SetGroupAddRequest(ctx context.Context, flag string, approve bool, reason string) error {
	e.mu.Lock()
	echoId, err := wsAction(e.conn, ACTION_SET_GROUP_ADD_REQUEST, types.GroupAddReq{
		Flag:    flag,
		Approve: approve,
		Reason:  reason,
	})
	if err != nil {
		e.mu.Unlock()
		return err
	}
	e.mu.Unlock()
	_, err = waitAndDecode[any](ctx, e, echoId)
	return err
}

func (e *emitterSocket) SetGroupSpecialTitle(ctx context.Context, groupId int64, userId int64, specialTitle string, duration int) error {
	e.mu.Lock()
	echoId, err := wsAction(e.conn, ACTION_SET_GROUP_SPECIAL_TITLE, types.SpecialTitleReq{
		GroupId:      groupId,
		UserId:       userId,
		SpecialTitle: specialTitle,
	})
	if err != nil {
		e.mu.Unlock()
		return err
	}
	e.mu.Unlock()
	_, err = waitAndDecode[any](ctx, e, echoId)
	return err
}

// ADD 不存在于Onebot大典的内容

func (e *emitterSocket) QuitGroup(ctx context.Context, groupId int64) error {
	e.mu.Lock()
	echoId, err := wsAction(e.conn, ACTION_QUIT_GROUP, types.QuitGroupReq{
		GroupId: groupId,
	})
	if err != nil {
		e.mu.Unlock()
		return err
	}
	e.mu.Unlock()
	_, err = waitAndDecode[any](ctx, e, echoId)
	return err
}

func (e *emitterSocket) SetGroupCard(ctx context.Context, groupId int64, userId int64, card string) error {
	e.mu.Lock()
	echoId, err := wsAction(e.conn, ACTION_SET_GROUP_CARD, types.SetGroupCardReq{
		GroupId: groupId,
		UserId:  userId,
		Card:    card,
	})
	if err != nil {
		e.mu.Unlock()
		return err
	}
	e.mu.Unlock()
	_, err = waitAndDecode[any](ctx, e, echoId)
	return err
}

func (e *emitterSocket) GetGroupInfo(ctx context.Context, groupId int64, noCache bool) (*types.GroupInfo, error) {
	e.mu.Lock()
	echoId, err := wsAction(e.conn, ACTION_GET_GROUP_INFO, types.GetGroupInfoReq{
		GroupId: groupId,
		NoCache: noCache,
	})
	if err != nil {
		e.mu.Unlock()
		return nil, err
	}
	e.mu.Unlock()
	return waitAndDecode[types.GroupInfo](ctx, e, echoId)
}

func (e *emitterSocket) GetGroupMemberInfo(ctx context.Context, groupId int64, userId int64, noCache bool) (*types.GroupMemberInfo, error) {
	e.mu.Lock()
	echoId, err := wsAction(e.conn, ACTION_GET_GROUP_MEMBER_INFO, types.GetGroupMemberInfoReq{
		GroupId: groupId,
		UserId:  userId,
		NoCache: noCache,
	})
	if err != nil {
		e.mu.Unlock()
		return nil, err
	}
	e.mu.Unlock()
	return waitAndDecode[types.GroupMemberInfo](ctx, e, echoId)
}

func (e *emitterSocket) Raw(ctx context.Context, action Action, params any) ([]byte, error) {
	e.mu.Lock()
	echoId, err := wsAction(e.conn, action, params)
	if err != nil {
		e.mu.Unlock()
		return nil, err
	}
	e.mu.Unlock()
	resp, err := e.waitEcho(ctx, echoId)
	if err != nil {
		return nil, err
	}
	return sonic.Marshal(resp)
}

func wsAction[P any](w *socketio.WebsocketWrapper, action string, params P) (string, error) {
	echoid := uuid.New().String()
	marshal, err := sonic.Marshal(Request[P]{
		Action: action,
		Echo:   echoid,
		Params: params,
	})
	if err != nil {
		return "", err
	}
	// 消息推入消息队列，等待发送
	w.Emit(marshal, socketio.TextMessage)
	return echoid, nil
}
