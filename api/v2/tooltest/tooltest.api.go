package tooltest

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"sealdice-core/dice"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
)

const (
	privateMode = "private"
	groupMode   = "group"
)

type Service struct {
	dice *dice.Dice
	dm   *dice.DiceManager

	mu                  sync.Mutex
	lastPrivateExecTime int64
	lastGroupExecTime   int64

	now      func() time.Time
	dispatch func(ep *dice.EndPointInfo, msg *dice.Message)
}

func NewService(dm *dice.DiceManager) *Service {
	s := &Service{
		dice: dm.GetDice(),
		dm:   dm,
		now:  time.Now,
	}
	s.dispatch = func(ep *dice.EndPointInfo, msg *dice.Message) {
		if s.dice == nil || s.dice.ImSession == nil || ep == nil || msg == nil {
			return
		}
		s.dice.ImSession.Execute(ep, msg, false)
	}
	return s
}

func (s *Service) Dice() *dice.Dice {
	return s.dice
}

func (s *Service) RegisterRoutes(grp *huma.Group) {
	huma.Get(grp, "/messages/pending", s.GetPendingMessages, func(o *huma.Operation) {
		o.Description = "获取并清空指令测试待收取消息"
		o.Summary = "获取指令测试待收取消息"
	})
	huma.Get(grp, "/commands", s.GetCommands, func(o *huma.Operation) {
		o.Description = "获取指令测试命令补全列表"
		o.Summary = "获取指令测试命令列表"
	})
}

func (s *Service) RegisterProtectedRoutes(grp *huma.Group) {
	huma.Post(grp, "/messages", s.PostMessage, func(o *huma.Operation) {
		o.Description = "发送一条指令测试消息。TODO: 后续改为接入 realtime 推送，移除 pending 轮询。"
		o.Summary = "发送指令测试消息"
	})
}

func (s *Service) PostMessage(_ context.Context, req *PostMessageReq) (*SimpleItemResponse, error) {
	if s.dice == nil || s.dice.UIEndpoint == nil || s.dice.UIEndpoint.Adapter == nil {
		return nil, huma.Error500InternalServerError("UI endpoint unavailable")
	}

	body := req.Body
	text := strings.TrimSpace(body.Text)
	if text == "" {
		return nil, huma.Error400BadRequest("text不能为空")
	}

	mode, err := normalizeMode(body.Mode)
	if err != nil {
		return nil, err
	}
	if err := s.checkRateLimit(mode); err != nil {
		return nil, err
	}

	msg := &dice.Message{
		MessageType: mode,
		Message:     text,
		Platform:    "UI",
		Sender: dice.SenderBase{
			Nickname: "User",
			UserID:   "UI:1001",
		},
	}
	if mode == groupMode {
		msg.GroupID = "UI-Group:2001"
		msg.GroupName = "UI-Group 2001"
		msg.Sender.UserID = "UI:1002"
		msg.Sender.GroupRole = "owner"
	}

	s.dispatch(s.dice.UIEndpoint, msg)
	return response.NewItemResponse(response.SimpleOK{Success: true}), nil
}

func (s *Service) GetPendingMessages(_ context.Context, _ *request.Empty) (*PendingMessagesItemResponse, error) {
	adapter, err := s.uiAdapter()
	if err != nil {
		return nil, err
	}

	items := make([]MessageItem, 0, len(adapter.RecentMessage))
	for _, item := range adapter.RecentMessage {
		items = append(items, MessageItem{
			UID:         item.UID,
			Message:     item.Message,
			MessageType: item.MessageType,
		})
	}
	adapter.RecentMessage = nil

	return response.NewItemResponse(PendingMessagesResp{Items: items}), nil
}

func (s *Service) GetCommands(_ context.Context, _ *request.Empty) (*CommandsItemResponse, error) {
	if s.dice == nil {
		return nil, huma.Error500InternalServerError("Dice instance is nil")
	}

	commands := make([]string, 0, len(s.dice.CmdMap))
	for key := range s.dice.CmdMap {
		commands = append(commands, key)
	}
	for _, ext := range s.dice.ExtList {
		if ext == nil {
			continue
		}
		for key := range ext.GetCmdMap() {
			commands = append(commands, key)
		}
	}
	sort.Sort(dice.ByLength(commands))

	return response.NewItemResponse(CommandsResp{Items: commands}), nil
}

func (s *Service) uiAdapter() (*dice.PlatformAdapterHTTP, error) {
	if s.dice == nil || s.dice.UIEndpoint == nil {
		return nil, huma.Error500InternalServerError("UI endpoint unavailable")
	}
	adapter, ok := s.dice.UIEndpoint.Adapter.(*dice.PlatformAdapterHTTP)
	if !ok || adapter == nil {
		return nil, huma.Error500InternalServerError("UI adapter unavailable")
	}
	return adapter, nil
}

func normalizeMode(mode string) (string, error) {
	switch strings.TrimSpace(mode) {
	case "", privateMode:
		return privateMode, nil
	case groupMode:
		return groupMode, nil
	default:
		return "", huma.Error400BadRequest("mode必须为private或group")
	}
}

func (s *Service) checkRateLimit(mode string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	interval := int64(500)
	if s.dm != nil && s.dm.JustForTest {
		interval = 80
	}

	now := s.now().UnixMilli()
	switch mode {
	case groupMode:
		if now-s.lastGroupExecTime < interval {
			return huma.Error400BadRequest("消息过于频繁")
		}
		s.lastGroupExecTime = now
	default:
		if now-s.lastPrivateExecTime < interval {
			return huma.Error400BadRequest("消息过于频繁")
		}
		s.lastPrivateExecTime = now
	}

	return nil
}
