package dice

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type AppVersionInfo struct {
	AppName         string `json:"app_name"`
	ProtocolVersion string `json:"protocol_version"`
	AppVersion      string `json:"app_version"`
}

type LinkerMode string

// linkermode分为string和array
const (
	LinkerModeString LinkerMode = "string"
	LinkerModeArray  LinkerMode = "array"
)

// echo 事件 定义的类型
const (
	// GetLoginInfo 获取登录信息
	GetLoginInfo = "get_login_info"
	// GetGroupInfo 获取群信息
	GetGroupInfo = "get_group_info"
	// 获取版本信息
	GetVersionInfo = "get_version_info"
)

// 自己定义的TOPIC列表
const (
	TopicHandleAddNewFriends = "handle_add_friends"
	TopicHandleInviteToGroup = "handle_invite_group" // 只是为了获取群信息……
	TopicHandleAddNewGroup   = "handle_add_group"
)

// ONEBOT事件对应码表
// OnebotEventPostTypeCode
const (
	OnebotEventPostTypeMessage = "onebot_message"
	// OnebotEventPostTypeCodeMessageSent 是bot发出的消息
	OnebotEventPostTypeMessageSent = "onebot_message_sent"
	OnebotEventPostTypeRequest     = "onebot_request"
	OnebotEventPostTypeNotice      = "onebot_notice"
	OnebotEventPostTypeMetaEvent   = "onebot_meta_event"
)

const (
	OnebotReceiveMessage = "onebot_echo"
)

// Model层 TODO: 直接放model层是不是挺好的

type FriendRequestResponse struct {
	Flag    string `json:"flag"`    // 对应 req.Get("flag").String()
	Approve bool   `json:"approve"` // 对应 passQuestion && passblackList
	Remark  string `json:"remark"`  // 空字符串
	Reason  string `json:"reason"`  // 对应 extra
}

type GroupRequestResponse struct {
	Flag      string `json:"flag"`       // 对应 req.Get("flag").String()
	Approve   bool   `json:"approve"`    // 对应 passQuestion && passblackList
	Reason    string `json:"reason"`     // 对应 extra
	GroupName string `json:"group_name"` // 群组名
	GroupID   string `json:"group_id"`   // 群组ID
	UserID    string `json:"user_id"`    // 邀请人ID
	SubType   string `json:"sub_type"`
}

// Model层 End

// OBQQUidType copied from platform_adapter_gocq
type OBQQUidType int

const (
	OBQQUidPerson        OBQQUidType = 1
	OBQQUidGroup         OBQQUidType = 2
	OBQQUidChannelPerson OBQQUidType = 3
	OBQQUidChannelGroup  OBQQUidType = 4
)

func (pa *PlatformAdapterPureOnebot11) mustExtractID(id string) (int64, OBQQUidType) {
	if strings.HasPrefix(id, "QQ:") {
		num, _ := strconv.ParseInt(id[len("QQ:"):], 10, 64)
		return num, OBQQUidPerson
	}
	if strings.HasPrefix(id, "QQ-Group:") {
		num, _ := strconv.ParseInt(id[len("QQ-Group:"):], 10, 64)
		return num, OBQQUidGroup
	}
	if strings.HasPrefix(id, "PG-QQ:") {
		num, _ := strconv.ParseInt(id[len("PG-QQ:"):], 10, 64)
		return num, OBQQUidPerson
	}
	return 0, 0
}

// 讲获取的数据转换为海豹内的Message对象
func (p *PlatformAdapterPureOnebot11) convertStringMessage(operator gjson.Result) *Message {
	msg := new(Message)

	msg.Time = operator.Get("time").Int()
	msg.MessageType = operator.Get("message_type").String()
	msg.Message = operator.Get("message").String()
	// 看上去就是原本的替换策略
	msg.Message = strings.ReplaceAll(msg.Message, "&#91;", "[")
	msg.Message = strings.ReplaceAll(msg.Message, "&#93;", "]")
	msg.Message = strings.ReplaceAll(msg.Message, "&amp;", "&")
	msg.RawID = operator.Get("message_id").String()
	msg.Platform = "QQ"

	if msg.MessageType == "" {
		msg.MessageType = "private"
	}

	// 这两段代码什么情况？尝试取值？
	if operator.Get("data").Exists() && operator.Get("data.group_id").Exists() {
		msg.GroupID = FormatDiceIDQQGroup(operator.Get("data.group_id").String())
	}
	if operator.Get("group_id").Exists() {
		if msg.MessageType == "private" {
			msg.MessageType = "group"
		}
		msg.GroupID = FormatDiceIDQQGroup(operator.Get("group_id").String())
	}
	sender := operator.Get("sender")
	if sender.Exists() {
		msg.Sender.Nickname = sender.Get("nickname").String()
		// 如果用户有群昵称，且群昵称不是空的的情况
		if sender.Get("card").Exists() && sender.Get("card").String() != "" {
			msg.Sender.Nickname = sender.Get("card").String()
		}
		msg.Sender.GroupRole = sender.Get("role").String()
		msg.Sender.UserID = FormatDiceIDQQ(sender.Get("user_id").String())
	}
	return msg
}

// 将OB11的Array数据转换为string字符串
func (p *PlatformAdapterPureOnebot11) parseOB11ArrayToStringMessage(parseContent gjson.Result) (gjson.Result, error) {
	arrayContent := parseContent.Get("message").Array()
	cqMessage := strings.Builder{}

	for _, i := range arrayContent {
		typeStr := i.Get("type").String()
		dataObj := i.Get("data")
		switch typeStr {
		case "text":
			cqMessage.WriteString(dataObj.Get("text").String())
		case "image":
			// 兼容NC情况, 此时file字段只有文件名, 完整URL在url字段
			if !hasURLScheme(dataObj.Get("file").String()) && hasURLScheme(dataObj.Get("url").String()) {
				cqMessage.WriteString(fmt.Sprintf("[CQ:image,file=%v]", dataObj.Get("url").String()))
			} else {
				cqMessage.WriteString(fmt.Sprintf("[CQ:image,file=%v]", dataObj.Get("file").String()))
			}
		case "face":
			// 兼容四叶草，移除 .(string)。自动获取的信息表示此类型为 float64，这是go解析的问题
			cqMessage.WriteString(fmt.Sprintf("[CQ:face,id=%v]", dataObj.Get("id").String()))
		case "record":
			// 兼容NC情况, 此时file字段只有文件名, 完整路径在path字段
			if !hasURLScheme(dataObj.Get("file").String()) && dataObj.Get("path").String() != "" {
				cqMessage.WriteString(fmt.Sprintf("[CQ:record,file=%v]", dataObj.Get("path").String()))
			} else {
				cqMessage.WriteString(fmt.Sprintf("[CQ:record,file=%v]", dataObj.Get("file").String()))
			}
		case "at":
			cqMessage.WriteString(fmt.Sprintf("[CQ:at,qq=%v]", dataObj.Get("qq").String()))
		case "poke":
			cqMessage.WriteString("[CQ:poke]")
		case "reply":
			cqMessage.WriteString(fmt.Sprintf("[CQ:reply,id=%v]", dataObj.Get("id").String()))
		}
	}
	// 赋值对应的Message
	tempStr, err := sjson.Set(parseContent.String(), "message", cqMessage.String())
	if err != nil {
		return gjson.Result{}, err
	}
	return gjson.Parse(tempStr), nil
}

// 检查加好友是否成功
func (p *PlatformAdapterPureOnebot11) checkMultiFriendAddVerify(comment string, toMatch string) bool {
	// 根据GPT的描述，这里干的事情是：从评论中提取回答内容，并与目标字符串进行逐项匹配，最终决定是否接受。
	// 我只是从木落那里拆了过来，太热闹了。
	var willAccept bool
	re := regexp.MustCompile(`\n回答:([^\n]+)`)
	m := re.FindAllStringSubmatch(comment, -1)
	// 要匹配的是空，说明不验证
	if toMatch == "" {
		willAccept = true
		return willAccept
	}
	var items []string
	for _, i := range m {
		items = append(items, i[1])
	}

	re2 := regexp.MustCompile(`\s+`)
	m2 := re2.Split(toMatch, -1)

	if len(m2) == len(items) {
		ok := true
		for i := range m2 {
			if m2[i] != items[i] {
				ok = false
				break
			}
		}
		willAccept = ok
	}
	return willAccept
}

// 检查是否在用户黑名单
func (p *PlatformAdapterPureOnebot11) checkPassBlackList(userId string, ctx *MsgContext) bool {
	// 检查 userId 是否有效
	if userId == "" {
		return true // 如果 userId 为空，默认通过黑名单检查
	}

	// 调用 FormatDiceIDQQ 并检查返回值是否有效
	uid := FormatDiceIDQQ(userId)
	if uid == "" {
		return true // 如果格式化后 ID 为空，默认通过黑名单检查
	}

	// 获取禁用信息
	banInfo, ok := ctx.Dice.Config.BanList.GetByID(uid)
	if !ok || banInfo == nil {
		return true // 如果用户不在黑名单中，默认通过
	}

	// 检查禁用等级和行为配置
	if banInfo.Rank == BanRankBanned && ctx.Dice.Config.BanList.BanBehaviorRefuseInvite {
		return false // 如果用户被禁用且配置拒绝邀请，则返回 false
	}

	return true // 其他情况默认通过
}

func (p *PlatformAdapterPureOnebot11) checkPassBlackListGroup(userId string, groupID string, ctx *MsgContext) (bool, string) {
	userAllow := p.checkPassBlackList(userId, ctx)
	if !userAllow {
		return false, "邀请人在黑名单上"
	}
	// TODO: 拿捏不准，这里的ID是被拼过的？
	banInfo, ok := ctx.Dice.Config.BanList.GetByID(groupID)
	if ok {
		if banInfo.Rank == BanRankBanned {
			return false, "群黑名单"
		}
	}
	return true, ""
}
