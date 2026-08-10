package tooltest

import (
	"context"
	"fmt"
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
	publish  func(name string, payload any)
}

const toolTestMessageEvent = "tooltest/message"

type defaultUITestProfile struct {
	ID        string
	Name      string
	Role      string
	AvatarKey string
}

type uiTestIdentity struct {
	Player *dice.GroupPlayerInfo
	Meta   *dice.UITestMember
}

var defaultUITestProfiles = []defaultUITestProfile{
	{ID: "UI:1001", Name: "普通用户", Role: UITestRoleMember, AvatarKey: "member"},
	{ID: "UI:1002", Name: "群主", Role: UITestRoleOwner, AvatarKey: "owner"},
	{ID: "UI:1003", Name: "管理员", Role: UITestRoleAdmin, AvatarKey: "admin"},
	{ID: "UI:1004", Name: "邀请人", Role: UITestRoleInviter, AvatarKey: "inviter"},
	{ID: "UI:1005", Name: "骰主", Role: UITestRoleMaster, AvatarKey: "master"},
	{ID: "UI:1006", Name: "普通用户 2", Role: UITestRoleMember, AvatarKey: "member-2"},
	{ID: "UI:1007", Name: "黑名单用户", Role: UITestRoleBlacklisted, AvatarKey: "blacklisted"},
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
	s.publish = func(_ string, _ any) {}
	return s
}

func (s *Service) Dice() *dice.Dice {
	return s.dice
}

func (s *Service) RegisterRoutes(grp *huma.Group) {
	huma.Get(grp, "/context", s.GetContext, func(o *huma.Operation) {
		o.Description = "获取指令测试的私聊或群聊上下文及持久化假人列表"
		o.Summary = "获取指令测试上下文"
	})
	huma.Get(grp, "/messages/pending", s.GetPendingMessages, func(o *huma.Operation) {
		o.Description = "获取并清空指令测试待收取消息"
		o.Summary = "获取指令测试待收取消息"
	})
	huma.Get(grp, "/commands", s.GetCommands, func(o *huma.Operation) {
		o.Description = "获取当前私聊或群聊上下文可用的指令补全列表"
		o.Summary = "获取指令测试命令列表"
	})
	huma.Get(grp, "/split-options", s.GetSplitOptions, func(o *huma.Operation) {
		o.Description = "获取指令测试回复分段选项"
		o.Summary = "获取指令测试分段选项"
	})
}

func (s *Service) RegisterProtectedRoutes(grp *huma.Group) {
	huma.Put(grp, "/profile", s.UpdateProfile, func(o *huma.Operation) {
		o.Description = "更新指令测试假人的持久化资料"
		o.Summary = "更新指令测试假人"
	})
	huma.Put(grp, "/context", s.UpdateContext, func(o *huma.Operation) {
		o.Description = "更新指令测试群组名称和群黑白名单测试状态"
		o.Summary = "更新指令测试上下文"
	})
	huma.Post(grp, "/messages", s.PostMessage, func(o *huma.Operation) {
		o.Description = "发送一条指令测试消息，并通过 realtime 推送结构化回复。"
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

	senderID := strings.TrimSpace(body.SenderID)
	if senderID == "" {
		if mode == groupMode {
			senderID = "UI:1002"
		} else {
			senderID = "UI:1001"
		}
	}
	if !strings.HasPrefix(senderID, "UI:") {
		return nil, huma.Error400BadRequest("senderId必须使用UI:前缀")
	}
	groupID := strings.TrimSpace(body.GroupID)
	if mode == groupMode {
		if groupID == "" {
			groupID = "UI-Group:2001"
		}
		if !strings.HasPrefix(groupID, "UI-Group:") {
			return nil, huma.Error400BadRequest("groupId必须使用UI-Group:前缀")
		}
	}
	group, profile, err := s.ensureProfile(mode, senderID, groupID)
	if err != nil {
		return nil, err
	}
	if !profile.Meta.Enabled {
		return nil, huma.Error400BadRequest("该测试身份已停用")
	}

	msg := &dice.Message{
		MessageType: mode,
		Message:     text,
		Platform:    "UI",
		Sender: dice.SenderBase{
			Nickname: profile.Player.Name,
			UserID:   senderID,
		},
		UITestReplySplitLen: body.MessageSplitLen,
		UITestRole:          profile.Meta.Role,
	}
	if mode == groupMode {
		msg.GroupID = group.GroupID
		msg.GroupName = group.GroupName
		msg.Sender.GroupRole = groupRoleForUITest(profile.Meta.Role)
		msg.UITestGroupAccess = group.UITest.Access
	}

	s.bindAdapter()
	s.publish(toolTestMessageEvent, dice.HTTPTestMessage{
		ID:             fmt.Sprintf("ui-input-%d", time.Now().UnixNano()),
		MessageType:    mode,
		ConversationID: conversationID(mode, senderID, groupID),
		GroupID:        groupID,
		GroupName:      groupName(group),
		Sender:         msg.Sender,
		SenderRole:     profile.Meta.Role,
		AvatarKey:      profile.Meta.AvatarKey,
		Direction:      "incoming",
		RawMessage:     text,
		Segments:       []dice.HTTPTestSegment{{Type: "text", Text: text}},
		Timestamp:      time.Now().UnixMilli(),
	})

	s.dispatch(s.dice.UIEndpoint, msg)
	return response.NewItemResponse(response.SimpleOK{Success: true}), nil
}

func (s *Service) GetPendingMessages(_ context.Context, _ *request.Empty) (*PendingMessagesItemResponse, error) {
	adapter, err := s.uiAdapter()
	if err != nil {
		return nil, err
	}

	recent := adapter.TakeRecentMessages()
	items := make([]MessageItem, 0, len(recent))
	for _, item := range recent {
		items = append(items, MessageItem{
			UID:         item.UID,
			Message:     item.Message,
			MessageType: item.MessageType,
		})
	}
	return response.NewItemResponse(PendingMessagesResp{Items: items}), nil
}

func (s *Service) GetContext(_ context.Context, req *ContextReq) (*ContextItemResponse, error) {
	mode, err := normalizeMode(req.Mode)
	if err != nil {
		return nil, err
	}
	senderID := strings.TrimSpace(req.SenderID)
	if senderID == "" {
		senderID = "UI:1001"
	}
	groupID := strings.TrimSpace(req.GroupID)
	if mode == groupMode && groupID == "" {
		groupID = "UI-Group:2001"
	}
	group, _, err := s.ensureProfile(mode, senderID, groupID)
	if err != nil {
		return nil, err
	}
	return response.NewItemResponse(s.contextResponse(mode, senderID, group)), nil
}

func (s *Service) UpdateProfile(_ context.Context, req *UpdateProfileReq) (*ContextItemResponse, error) {
	body := req.Body
	mode, err := normalizeMode(body.Mode)
	if err != nil {
		return nil, err
	}
	userID := strings.TrimSpace(body.UserID)
	if !strings.HasPrefix(userID, "UI:") {
		return nil, huma.Error400BadRequest("userId必须使用UI:前缀")
	}
	groupID := strings.TrimSpace(body.GroupID)
	if mode == groupMode && groupID == "" {
		groupID = "UI-Group:2001"
	}
	group, profile, err := s.ensureProfile(mode, userID, groupID)
	if err != nil {
		return nil, err
	}
	if body.Name != "" {
		profile.Player.Name = strings.TrimSpace(body.Name)
	}
	if body.Role != "" {
		if !isUITestRole(body.Role) {
			return nil, huma.Error400BadRequest("不支持的测试身份")
		}
		profile.Meta.Role = body.Role
	}
	if body.AvatarKey != "" {
		profile.Meta.AvatarKey = strings.TrimSpace(body.AvatarKey)
	}
	if body.Enabled != nil {
		profile.Meta.Enabled = *body.Enabled
	}
	markProfileDirty(s.dice, group, profile)
	return response.NewItemResponse(s.contextResponse(mode, userID, group)), nil
}

func (s *Service) UpdateContext(_ context.Context, req *UpdateContextReq) (*ContextItemResponse, error) {
	groupID := strings.TrimSpace(req.Body.GroupID)
	if !strings.HasPrefix(groupID, "UI-Group:") {
		return nil, huma.Error400BadRequest("groupId必须使用UI-Group:前缀")
	}
	if req.Body.GroupAccess != "" && req.Body.GroupAccess != "normal" && req.Body.GroupAccess != "blacklisted" && req.Body.GroupAccess != "trusted" {
		return nil, huma.Error400BadRequest("groupAccess必须为normal、blacklisted或trusted")
	}
	group, _, err := s.ensureProfile(groupMode, "UI:1002", groupID)
	if err != nil {
		return nil, err
	}
	if req.Body.GroupName != "" {
		group.GroupName = strings.TrimSpace(req.Body.GroupName)
	}
	group.UITest.Access = req.Body.GroupAccess
	group.MarkDirty(s.dice)
	return response.NewItemResponse(s.contextResponse(groupMode, "UI:1002", group)), nil
}

func (s *Service) GetCommands(_ context.Context, req *ContextReq) (*CommandsItemResponse, error) {
	if s.dice == nil {
		return nil, huma.Error500InternalServerError("Dice instance is nil")
	}
	mode := privateMode
	senderID := "UI:1001"
	groupID := ""
	if req != nil {
		var err error
		mode, err = normalizeMode(req.Mode)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(req.SenderID) != "" {
			senderID = strings.TrimSpace(req.SenderID)
		}
		groupID = strings.TrimSpace(req.GroupID)
	}
	if mode == groupMode && groupID == "" {
		groupID = "UI-Group:2001"
	}
	group, _, err := s.ensureProfile(mode, senderID, groupID)
	if err != nil {
		return nil, err
	}

	commands := make([]*CommandOption, 0, len(s.dice.CmdMap))
	seen := make(map[string]struct{})
	appendCommand := func(key string, item *dice.CmdItemInfo, source string) {
		key = strings.TrimSpace(key)
		if key == "" {
			return
		}
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		commands = append(commands, &CommandOption{
			Name:        key,
			Description: commandDescription(item),
			Source:      source,
		})
	}
	for key, item := range s.dice.CmdMap {
		appendCommand(key, item, "核心")
	}
	for _, ext := range group.GetActivatedExtList(s.dice) {
		if ext == nil {
			continue
		}
		source := strings.TrimSpace(ext.Name)
		if source == "" {
			source = "扩展"
		}
		for key, item := range ext.GetCmdMap() {
			appendCommand(key, item, source)
		}
	}
	sort.Slice(commands, func(i, j int) bool {
		if len(commands[i].Name) != len(commands[j].Name) {
			return len(commands[i].Name) > len(commands[j].Name)
		}
		return strings.ToLower(commands[i].Name) < strings.ToLower(commands[j].Name)
	})

	return response.NewItemResponse(CommandsResp{Items: commands}), nil
}

func commandDescription(item *dice.CmdItemInfo) string {
	if item == nil {
		return ""
	}
	for _, value := range []string{item.ShortHelp, item.Help} {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		value = strings.ReplaceAll(value, "\r\n", "\n")
		return strings.TrimSpace(strings.SplitN(value, "\n", 2)[0])
	}
	return ""
}

func (s *Service) GetSplitOptions(_ context.Context, _ *request.Empty) (*SplitOptionsItemResponse, error) {
	return response.NewItemResponse(SplitOptionsResp{
		DefaultKey: "qq",
		Options: []*SplitOption{
			{
				Key:             "short",
				Label:           fmt.Sprintf("短分段 %d", dice.UITestReplySplitLenShort),
				MessageSplitLen: dice.UITestReplySplitLenShort,
			},
			{
				Key:             "qq",
				Label:           fmt.Sprintf("QQ 分段 %d", dice.UITestReplySplitLenQQ),
				MessageSplitLen: dice.UITestReplySplitLenQQ,
			},
			{
				Key:             "unlimited",
				Label:           "无限",
				MessageSplitLen: dice.UITestReplySplitLenUnlimited,
			},
		},
	}), nil
}

type eventPublisher interface {
	Publish(name string, payload any)
}

func (s *Service) SetPublisher(publisher eventPublisher) {
	if publisher == nil {
		s.publish = func(_ string, _ any) {}
		return
	}
	s.publish = publisher.Publish
	s.bindAdapter()
}

func (s *Service) bindAdapter() {
	if s == nil || s.dice == nil || s.dice.UIEndpoint == nil {
		return
	}
	adapter, ok := s.dice.UIEndpoint.Adapter.(*dice.PlatformAdapterHTTP)
	if !ok || adapter == nil {
		return
	}
	if s.publish == nil {
		s.publish = func(_ string, _ any) {}
	}
	adapter.OnMessage = func(item dice.HTTPTestMessage) {
		s.publish(toolTestMessageEvent, item)
	}
}

func (s *Service) ensureProfile(mode string, userID string, groupID string) (*dice.GroupInfo, *uiTestIdentity, error) {
	if s.dice == nil || s.dice.ImSession == nil || s.dice.UIEndpoint == nil {
		return nil, nil, huma.Error500InternalServerError("UI endpoint unavailable")
	}
	if !strings.HasPrefix(userID, "UI:") {
		return nil, nil, huma.Error400BadRequest("userId必须使用UI:前缀")
	}
	contextID := groupID
	if mode == privateMode {
		contextID = "PG-" + userID
	} else if !strings.HasPrefix(contextID, "UI-Group:") {
		return nil, nil, huma.Error400BadRequest("groupId必须使用UI-Group:前缀")
	}

	ctx := &dice.MsgContext{
		Dice:     s.dice,
		Session:  s.dice.ImSession,
		EndPoint: s.dice.UIEndpoint,
	}
	group, exists := s.dice.ImSession.ServiceAtNew.Load(contextID)
	if !exists || group == nil {
		group = dice.SetBotOnAtGroup(ctx, contextID)
	}
	if group.GroupName == "" {
		if mode == groupMode {
			group.GroupName = "SealDice 测试群组"
		} else {
			group.GroupName = "与 " + userID + " 的私聊"
		}
		group.MarkDirty(s.dice)
	}
	if group.UITest == nil {
		group.UITest = &dice.UITestConfig{}
	}
	if group.UITest.Members == nil {
		group.UITest.Members = map[string]*dice.UITestMember{}
	}
	if !group.UITest.Initialized {
		for _, item := range defaultUITestProfiles {
			group.UITest.Members[item.ID] = &dice.UITestMember{
				Role:      item.Role,
				AvatarKey: item.AvatarKey,
				Enabled:   true,
			}
			ensurePlayer(group, s.dice, item.ID, item.Name)
		}
		group.UITest.Initialized = true
		group.MarkDirty(s.dice)
	}
	meta, ok := group.UITest.Members[userID]
	if !ok {
		meta = &dice.UITestMember{Role: UITestRoleMember, AvatarKey: "member", Enabled: true}
		group.UITest.Members[userID] = meta
		group.MarkDirty(s.dice)
	}
	player := ensurePlayer(group, s.dice, userID, userID)
	return group, &uiTestIdentity{Player: player, Meta: meta}, nil
}

func ensurePlayer(group *dice.GroupInfo, d *dice.Dice, userID string, defaultName string) *dice.GroupPlayerInfo {
	if group.Players == nil {
		group.Players = new(dice.SyncMap[string, *dice.GroupPlayerInfo])
	}
	if player, ok := group.Players.Load(userID); ok && player != nil {
		player.InGroup = true
		return player
	}
	var player *dice.GroupPlayerInfo
	if d != nil && d.DBOperator != nil {
		player = group.PlayerGet(d.DBOperator, userID)
	}
	if player == nil {
		player = &dice.GroupPlayerInfo{
			Name:          defaultName,
			UserID:        userID,
			InGroup:       true,
			ValueMapTemp:  nil,
			UpdatedAtTime: 1,
		}
		group.Players.Store(userID, player)
	}
	if player.Name == "" {
		player.Name = defaultName
		player.UpdatedAtTime = 1
	}
	player.InGroup = true
	return player
}

func markProfileDirty(d *dice.Dice, group *dice.GroupInfo, identity *uiTestIdentity) {
	if identity != nil && identity.Player != nil {
		identity.Player.UpdatedAtTime = 1
	}
	if group != nil {
		group.MarkDirty(d)
	}
}

func (s *Service) contextResponse(mode string, senderID string, group *dice.GroupInfo) ContextResp {
	members := make([]*ProfileItem, 0)
	if group != nil && group.UITest != nil {
		keys := make([]string, 0, len(group.UITest.Members))
		for key := range group.UITest.Members {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, userID := range keys {
			meta := group.UITest.Members[userID]
			if meta == nil {
				continue
			}
			player := ensurePlayer(group, s.dice, userID, userID)
			members = append(members, &ProfileItem{
				UserID:    userID,
				Name:      player.Name,
				Role:      meta.Role,
				AvatarKey: meta.AvatarKey,
				Enabled:   meta.Enabled,
			})
		}
	}
	groupID := ""
	groupName := ""
	access := ""
	if mode == groupMode && group != nil {
		groupID = group.GroupID
		groupName = group.GroupName
		if group.UITest != nil {
			access = group.UITest.Access
		}
	}
	return ContextResp{
		Mode:            mode,
		ConversationID:  conversationID(mode, senderID, groupID),
		GroupID:         groupID,
		GroupName:       groupName,
		GroupAccess:     access,
		CurrentSenderID: senderID,
		Members:         members,
		BotName:         "海豹核心",
		BotAvatarKey:    "seal",
		CommandPrefix:   append([]string{}, s.dice.CommandPrefix...),
	}
}

func conversationID(mode string, senderID string, groupID string) string {
	if mode == groupMode {
		return groupID
	}
	return "PG-" + senderID
}

func groupName(group *dice.GroupInfo) string {
	if group == nil {
		return ""
	}
	return group.GroupName
}

func groupRoleForUITest(role string) string {
	switch role {
	case UITestRoleOwner:
		return "owner"
	case UITestRoleAdmin:
		return "admin"
	default:
		return ""
	}
}

func isUITestRole(role string) bool {
	switch role {
	case UITestRoleOwner, UITestRoleAdmin, UITestRoleInviter, UITestRoleMaster, UITestRoleMember, UITestRoleBlacklisted:
		return true
	default:
		return false
	}
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
