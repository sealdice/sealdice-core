package emitter

// fork from https://github.com/nsxdevx/nsxbot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"

	socketio "github.com/PaienNate/pineutil/evsocket/v2"

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
	SetGroupAddRequest(ctx context.Context, flag string, subType string, approve bool, reason string) error
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
	RetCode int    `json:"retcode"`
	Data    T      `json:"data,omitempty"`
	Echo    string `json:"echo"`
}

func (r *Response[T]) UnmarshalJSON(data []byte) error {
	type rawResponse struct {
		Status       string `json:"status"`
		RetCode      int    `json:"retcode"`
		RetCodeCamel int    `json:"retCode"`
		Data         T      `json:"data,omitempty"`
		Echo         string `json:"echo"`
	}

	var aux rawResponse
	if err := sonic.Unmarshal(data, &aux); err != nil {
		return err
	}

	r.Status = aux.Status
	r.RetCode = aux.RetCode
	if r.RetCode == 0 && aux.RetCodeCamel != 0 {
		r.RetCode = aux.RetCodeCamel
	}
	r.Data = aux.Data
	r.Echo = aux.Echo
	return nil
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

func (e *emitterSocket) waitEchoAfterSend(ctx context.Context, action Action, echoId string, params any, send func() error) (Response[sonic.NoCopyRawMessage], error) {
	ctx, cancel := context.WithTimeout(ctx, EchoTimeOut)
	defer cancel()

	ch := make(chan Response[sonic.NoCopyRawMessage], 1)
	e.waiters.Store(echoId, ch)
	defer e.waiters.Delete(echoId)

	if err := send(); err != nil {
		return Response[sonic.NoCopyRawMessage]{}, wrapActionError(err, action, echoId, params)
	}

	select {
	case <-ctx.Done():
		return Response[sonic.NoCopyRawMessage]{}, wrapActionError(ctx.Err(), action, echoId, params)
	case resp := <-ch:
		return resp, nil
	}
}

func decodeResponse[R any](action Action, echoId string, params any, resp Response[sonic.NoCopyRawMessage]) (*R, error) {
	if strings.EqualFold(resp.Status, "failed") || resp.RetCode != 0 {
		return nil, fmt.Errorf("OneBot 动作执行失败: action=%s echo=%s params=%s response=%s",
			action, echoId, marshalForError(params), marshalForError(resp))
	}
	var res R
	if err := sonic.Unmarshal(resp.Data, &res); err != nil {
		return nil, fmt.Errorf("解析 OneBot 响应失败: action=%s echo=%s params=%s response=%s err=%w",
			action, echoId, marshalForError(params), marshalForError(resp), err)
	}
	return &res, nil
}

func (e *emitterSocket) SendPvtMsg(ctx context.Context, userId int64, msg schema.MessageChain) (*types.SendMsgRes, error) {
	params := types.SendPrivateMsgReq{
		UserId:  userId,
		Message: msg,
	}
	resp, err := doAction(ctx, e, ACTION_SEND_PRIVATE_MSG, params)
	if err != nil {
		return nil, err
	}
	return decodeResponse[types.SendMsgRes](ACTION_SEND_PRIVATE_MSG, resp.Echo, params, resp)
}

func (e *emitterSocket) SendGrMsg(ctx context.Context, groupId int64, msg schema.MessageChain) (*types.SendMsgRes, error) {
	params := types.SendGrMsgReq{
		GroupId: groupId,
		Message: msg,
	}
	resp, err := doAction(ctx, e, ACTION_SEND_GROUP_MSG, params)
	if err != nil {
		return nil, err
	}
	return decodeResponse[types.SendMsgRes](ACTION_SEND_GROUP_MSG, resp.Echo, params, resp)
}

func (e *emitterSocket) GetMsg(ctx context.Context, msgId int) (*types.GetMsgRes, error) {
	params := types.GetMsgReq{
		MessageId: msgId,
	}
	resp, err := doAction(ctx, e, ACTION_GET_MSG, params)
	if err != nil {
		return nil, err
	}
	return decodeResponse[types.GetMsgRes](ACTION_GET_MSG, resp.Echo, params, resp)
}

func (e *emitterSocket) DelMsg(ctx context.Context, msgId int) error {
	params := types.DelMsgReq{
		MessageId: msgId,
	}
	resp, err := doAction(ctx, e, ACTION_DELETE_MSG, params)
	if err != nil {
		return err
	}
	_, err = decodeResponse[any](ACTION_DELETE_MSG, resp.Echo, params, resp)
	return err
}

func (e *emitterSocket) GetLoginInfo(ctx context.Context) (*types.LoginInfo, error) {
	resp, err := doAction(ctx, e, ACTION_GET_LOGIN_INFO, nil)
	if err != nil {
		return nil, err
	}
	return decodeResponse[types.LoginInfo](ACTION_GET_LOGIN_INFO, resp.Echo, nil, resp)
}

func (e *emitterSocket) GetStrangerInfo(ctx context.Context, userId int64, noCache bool) (*types.StrangerInfo, error) {
	params := types.GetStrangerInfo{
		UserId:  userId,
		NoCache: noCache,
	}
	resp, err := doAction(ctx, e, ACTION_GET_STRANGER_INFO, params)
	if err != nil {
		return nil, err
	}
	return decodeResponse[types.StrangerInfo](ACTION_GET_STRANGER_INFO, resp.Echo, params, resp)
}

func (e *emitterSocket) GetStatus(ctx context.Context) (*types.Status, error) {
	resp, err := doAction(ctx, e, ACTION_GET_STATUS, nil)
	if err != nil {
		return nil, err
	}
	return decodeResponse[types.Status](ACTION_GET_STATUS, resp.Echo, nil, resp)
}

func (e *emitterSocket) GetVersionInfo(ctx context.Context) (*types.VersionInfo, error) {
	resp, err := doAction(ctx, e, ACTION_GET_VERSION_INFO, nil)
	if err != nil {
		return nil, err
	}
	return decodeResponse[types.VersionInfo](ACTION_GET_VERSION_INFO, resp.Echo, nil, resp)
}

func (e *emitterSocket) GetSelfId(_ context.Context) (int64, error) {
	return e.selfId, nil
}

func (e *emitterSocket) SetFriendAddRequest(ctx context.Context, flag string, approve bool, remark string) error {
	params := types.FriendAddReq{
		Flag:    flag,
		Approve: approve,
		Remark:  remark,
	}
	resp, err := doAction(ctx, e, ACTION_SET_FRIEND_ADD_REQUEST, params)
	if err != nil {
		return err
	}
	_, err = decodeResponse[any](ACTION_SET_FRIEND_ADD_REQUEST, resp.Echo, params, resp)
	return err
}

func (e *emitterSocket) SetGroupAddRequest(ctx context.Context, flag string, subType string, approve bool, reason string) error {
	params := types.GroupAddReq{
		Flag:    flag,
		SubType: subType,
		Approve: approve,
		Reason:  reason,
	}
	resp, err := doAction(ctx, e, ACTION_SET_GROUP_ADD_REQUEST, params)
	if err != nil {
		return err
	}
	_, err = decodeResponse[any](ACTION_SET_GROUP_ADD_REQUEST, resp.Echo, params, resp)
	return err
}

func (e *emitterSocket) SetGroupSpecialTitle(ctx context.Context, groupId int64, userId int64, specialTitle string, duration int) error {
	params := types.SpecialTitleReq{
		GroupId:      groupId,
		UserId:       userId,
		SpecialTitle: specialTitle,
	}
	resp, err := doAction(ctx, e, ACTION_SET_GROUP_SPECIAL_TITLE, params)
	if err != nil {
		return err
	}
	_, err = decodeResponse[any](ACTION_SET_GROUP_SPECIAL_TITLE, resp.Echo, params, resp)
	return err
}

// ADD 不存在于Onebot大典的内容

func (e *emitterSocket) QuitGroup(ctx context.Context, groupId int64) error {
	params := types.QuitGroupReq{
		GroupId: groupId,
	}
	resp, err := doAction(ctx, e, ACTION_QUIT_GROUP, params)
	if err != nil {
		return err
	}
	_, err = decodeResponse[any](ACTION_QUIT_GROUP, resp.Echo, params, resp)
	return err
}

func (e *emitterSocket) SetGroupCard(ctx context.Context, groupId int64, userId int64, card string) error {
	params := types.SetGroupCardReq{
		GroupId: groupId,
		UserId:  userId,
		Card:    card,
	}
	resp, err := doAction(ctx, e, ACTION_SET_GROUP_CARD, params)
	if err != nil {
		return err
	}
	_, err = decodeResponse[any](ACTION_SET_GROUP_CARD, resp.Echo, params, resp)
	return err
}

func (e *emitterSocket) GetGroupInfo(ctx context.Context, groupId int64, noCache bool) (*types.GroupInfo, error) {
	params := types.GetGroupInfoReq{
		GroupId: groupId,
		NoCache: noCache,
	}
	resp, err := doAction(ctx, e, ACTION_GET_GROUP_INFO, params)
	if err != nil {
		return nil, err
	}
	return decodeResponse[types.GroupInfo](ACTION_GET_GROUP_INFO, resp.Echo, params, resp)
}

func (e *emitterSocket) GetGroupMemberInfo(ctx context.Context, groupId int64, userId int64, noCache bool) (*types.GroupMemberInfo, error) {
	params := types.GetGroupMemberInfoReq{
		GroupId: groupId,
		UserId:  userId,
		NoCache: noCache,
	}
	resp, err := doAction(ctx, e, ACTION_GET_GROUP_MEMBER_INFO, params)
	if err != nil {
		return nil, err
	}
	return decodeResponse[types.GroupMemberInfo](ACTION_GET_GROUP_MEMBER_INFO, resp.Echo, params, resp)
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
	resp, err := e.waitEchoAfterSend(ctx, action, echoId, params, func() error {
		e.mu.Lock()
		defer e.mu.Unlock()
		return wsEmitWithEcho(e.conn, action, params, echoId)
	})
	if err != nil {
		return Response[sonic.NoCopyRawMessage]{}, err
	}
	return resp, nil
}

func wrapActionError(err error, action Action, echoId string, params any) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("等待 OneBot Echo 超时: action=%s echo=%s params=%s err=%w",
			action, echoId, marshalForError(params), err)
	}
	return fmt.Errorf("执行 OneBot 动作失败: action=%s echo=%s params=%s err=%w",
		action, echoId, marshalForError(params), err)
}

func marshalForError(value any) string {
	if value == nil {
		return "null"
	}
	data, err := sonic.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%#v", value)
	}
	return string(data)
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
