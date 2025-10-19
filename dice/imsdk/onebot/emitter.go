package emitter

// fork from https://github.com/nsxdevx/nsxbot

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"

	socketio "github.com/PaienNate/pineutil/evsocket"

	"sealdice-core/dice/imsdk/onebot/schema"
	"sealdice-core/dice/imsdk/onebot/types"
)

var _ Emitter = (*EmitterEVSocket)(nil)

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
}

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

type EmitterEVSocket struct {
	mu     sync.Mutex
	conn   *socketio.WebsocketWrapper
	echo   chan Response[sonic.NoCopyRawMessage]
	selfId int64
}

func NewEVEmitter(conn *socketio.WebsocketWrapper, echo chan Response[sonic.NoCopyRawMessage]) *EmitterEVSocket {
	emitter := &EmitterEVSocket{
		conn: conn,
		echo: echo,
	}
	return emitter
}

func (e *EmitterEVSocket) SetSelfId(_ context.Context, selfId int64) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.selfId = selfId
	return nil
}

func (e *EmitterEVSocket) SendPvtMsg(ctx context.Context, userId int64, msg schema.MessageChain) (*types.SendMsgRes, error) {
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
	return wsWait[types.SendMsgRes](ctx, echoId, e.echo)
}

func (e *EmitterEVSocket) SendGrMsg(ctx context.Context, groupId int64, msg schema.MessageChain) (*types.SendMsgRes, error) {
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
	return wsWait[types.SendMsgRes](ctx, echoId, e.echo)
}

func (e *EmitterEVSocket) GetMsg(ctx context.Context, msgId int) (*types.GetMsgRes, error) {
	e.mu.Lock()
	echoId, err := wsAction(e.conn, ACTION_GET_MSG, types.GetMsgReq{
		MessageId: msgId,
	})
	if err != nil {
		e.mu.Unlock()
		return nil, err
	}
	e.mu.Unlock()
	return wsWait[types.GetMsgRes](ctx, echoId, e.echo)
}

func (e *EmitterEVSocket) DelMsg(ctx context.Context, msgId int) error {
	e.mu.Lock()
	echoId, err := wsAction[any](e.conn, ACTION_DELETE_MSG, types.DelMsgReq{
		MessageId: msgId,
	})
	if err != nil {
		e.mu.Unlock()
		return err
	}
	e.mu.Unlock()
	_, err = wsWait[any](ctx, echoId, e.echo)
	return err
}

func (e *EmitterEVSocket) GetLoginInfo(ctx context.Context) (*types.LoginInfo, error) {
	e.mu.Lock()
	echoId, err := wsAction[any](e.conn, ACTION_GET_LOGIN_INFO, nil)
	if err != nil {
		e.mu.Unlock()
		return nil, err
	}
	e.mu.Unlock()
	return wsWait[types.LoginInfo](ctx, echoId, e.echo)
}

func (e *EmitterEVSocket) GetStrangerInfo(ctx context.Context, userId int64, noCache bool) (*types.StrangerInfo, error) {
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
	return wsWait[types.StrangerInfo](ctx, echoId, e.echo)
}

func (e *EmitterEVSocket) GetStatus(ctx context.Context) (*types.Status, error) {
	e.mu.Lock()
	echoId, err := wsAction[any](e.conn, ACTION_GET_STATUS, nil)
	if err != nil {
		e.mu.Unlock()
		return nil, err
	}
	e.mu.Unlock()
	return wsWait[types.Status](ctx, echoId, e.echo)
}

func (e *EmitterEVSocket) GetVersionInfo(ctx context.Context) (*types.VersionInfo, error) {
	e.mu.Lock()
	echoId, err := wsAction[any](e.conn, ACTION_GET_VERSION_INFO, nil)
	if err != nil {
		e.mu.Unlock()
		return nil, err
	}
	e.mu.Unlock()
	return wsWait[types.VersionInfo](ctx, echoId, e.echo)
}

func (e *EmitterEVSocket) GetSelfId(_ context.Context) (int64, error) {
	return e.selfId, nil
}

func (e *EmitterEVSocket) SetFriendAddRequest(ctx context.Context, flag string, approve bool, remark string) error {
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
	_, err = wsWait[any](ctx, echoId, e.echo)
	return err
}

func (e *EmitterEVSocket) SetGroupAddRequest(ctx context.Context, flag string, approve bool, reason string) error {
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
	_, err = wsWait[any](ctx, echoId, e.echo)
	return err
}

func (e *EmitterEVSocket) SetGroupSpecialTitle(ctx context.Context, groupId int64, userId int64, specialTitle string, duration int) error {
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
	_, err = wsWait[any](ctx, echoId, e.echo)
	return err
}

// ADD 不存在于Onebot大典的内容

func (e *EmitterEVSocket) QuitGroup(ctx context.Context, groupId int64) error {
	e.mu.Lock()
	echoId, err := wsAction(e.conn, ACTION_QUIT_GROUP, types.QuitGroupReq{
		GroupId: groupId,
	})
	if err != nil {
		e.mu.Unlock()
		return err
	}
	e.mu.Unlock()
	_, err = wsWait[any](ctx, echoId, e.echo)
	return err
}

func (e *EmitterEVSocket) SetGroupCard(ctx context.Context, groupId int64, userId int64, card string) error {
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
	_, err = wsWait[any](ctx, echoId, e.echo)
	return err
}

func (e *EmitterEVSocket) GetGroupInfo(ctx context.Context, groupId int64, noCache bool) (*types.GroupInfo, error) {
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
	return wsWait[types.GroupInfo](ctx, echoId, e.echo)
}

func (e *EmitterEVSocket) GetGroupMemberInfo(ctx context.Context, groupId int64, userId int64, noCache bool) (*types.GroupMemberInfo, error) {
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
	return wsWait[types.GroupMemberInfo](ctx, echoId, e.echo)
}

func (e *EmitterEVSocket) Raw(ctx context.Context, action Action, params any) ([]byte, error) {
	e.mu.Lock()
	echoId, err := wsAction(e.conn, action, params)
	if err != nil {
		e.mu.Unlock()
		return nil, err
	}
	e.mu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, EchoTimeOut)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case echo := <-e.echo:
			if !strings.EqualFold(echoId, echo.Echo) {
				e.echo <- echo
				continue
			}
			return sonic.Marshal(echo)
		}
	}
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

func wsWait[R any](ctx context.Context, echoId string, echoChan chan Response[sonic.NoCopyRawMessage]) (*R, error) {
	ctx, cancel := context.WithTimeout(ctx, EchoTimeOut)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case echo := <-echoChan:
			if !strings.EqualFold(echoId, echo.Echo) {
				echoChan <- echo
				continue
			}
			if strings.EqualFold("failed", echo.Status) {
				return nil, fmt.Errorf("action failed, rawdata: %x, please see onebot logs", echo.Status)
			}
			var res R
			if err := sonic.Unmarshal(echo.Data, &res); err != nil {
				return nil, err
			}
			return &res, nil
		}
	}
}
