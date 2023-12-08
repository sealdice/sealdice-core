package dice

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"sealdice-core/utils"

	"github.com/sacOO7/gowebsocket"
	"github.com/samber/lo"
)

type oneBotCommand struct {
	Action string      `json:"action"`
	Params interface{} `json:"params"`
	Echo   interface{} `json:"echo"`
}

type QQUidType int

const (
	QQUidPerson        QQUidType = 1
	QQUidGroup         QQUidType = 2
	QQUidChannelPerson QQUidType = 3
	QQUidChannelGroup  QQUidType = 4
)

func (pa *PlatformAdapterGocq) mustExtractID(id string) (int64, QQUidType) {
	if strings.HasPrefix(id, "QQ:") {
		num, _ := strconv.ParseInt(id[len("QQ:"):], 10, 64)
		return num, QQUidPerson
	}
	if strings.HasPrefix(id, "QQ-Group:") {
		num, _ := strconv.ParseInt(id[len("QQ-Group:"):], 10, 64)
		return num, QQUidGroup
	}
	if strings.HasPrefix(id, "PG-QQ:") {
		num, _ := strconv.ParseInt(id[len("PG-QQ:"):], 10, 64)
		return num, QQUidPerson
	}
	return 0, 0
}

func (pa *PlatformAdapterGocq) mustExtractChannelID(id string) (string, QQUidType) {
	if strings.HasPrefix(id, "QQ-CH:") {
		return id[len("QQ-CH:"):], QQUidChannelPerson
	}
	if strings.HasPrefix(id, "QQ-CH-Group:") {
		return id[len("QQ-CH-Group:"):], QQUidChannelGroup
	}
	return "", 0
}

// GetGroupInfoAsync 异步获取群聊信息
func (pa *PlatformAdapterGocq) GetGroupInfoAsync(groupID string) {
	type GroupMessageParams struct {
		GroupID int64 `json:"group_id"`
	}
	realGroupID, idType := pa.mustExtractID(groupID)
	if idType != QQUidGroup {
		return
	}

	a, _ := json.Marshal(oneBotCommand{
		"get_group_info",
		GroupMessageParams{
			realGroupID,
		},
		-2,
	})

	socketSendText(pa.Socket, string(a))
}

type OnebotGroupInfo struct {
	GroupID         int64  `json:"group_id"`          // 群号
	GroupName       string `json:"group_name"`        // 群名称
	GroupMemo       string `json:"group_memo"`        // 群备注
	GroupCreateTime uint32 `json:"group_create_time"` // 群创建时间
	GroupLevel      uint32 `json:"group_level"`       // 群等级
	MemberCount     int32  `json:"member_count"`      // 成员数
	MaxMemberCount  int32  `json:"max_member_count"`  // 最大成员数（群容量）
}

// GetGroupInfo 获取群聊信息
func (pa *PlatformAdapterGocq) GetGroupInfo(groupID string) *OnebotGroupInfo {
	type GroupMessageParams struct {
		GroupID int64 `json:"group_id"`
	}
	realGroupID, idType := pa.mustExtractID(groupID)
	if idType != QQUidGroup {
		return nil
	}

	echo := pa.getCustomEcho()
	a, _ := json.Marshal(oneBotCommand{
		"get_group_info",
		GroupMessageParams{
			realGroupID,
		},
		echo,
	})

	data := &OnebotGroupInfo{}
	err := pa.waitEcho2(echo, data, func(emi *echoMapInfo) {
		emi.echoOverwrite = -2 // 强制覆盖为获取群信息，与之前兼容
		socketSendText(pa.Socket, string(a))
	})
	if err == nil {
		return data
	}
	return nil
}

func socketSendText(socket *gowebsocket.Socket, s string) {
	defer func() {
		if r := recover(); r != nil { //nolint
			// core.GetLogger().Error(r)
		}
	}()

	if socket != nil {
		socket.SendText(s)
	}
}

// 不知道为什么，使用这个时候发不出话
func socketSendBinary(socket *gowebsocket.Socket, data []byte) { //nolint
	defer func() {
		if r := recover(); r != nil { //nolint
			// core.GetLogger().Error(r)
		}
	}()

	if socket != nil {
		socket.SendBinary(data)
	}
}

func doSleepQQ(ctx *MsgContext) {
	if ctx.Dice != nil {
		d := ctx.Dice
		offset := d.Config.MessageDelayRangeEnd - d.Config.MessageDelayRangeStart
		time.Sleep(time.Duration((d.Config.MessageDelayRangeStart + rand.Float64()*offset) * float64(time.Second)))
	} else {
		time.Sleep(time.Duration((0.4 + rand.Float64()/2) * float64(time.Second)))
	}
}

func (pa *PlatformAdapterGocq) SendToPerson(ctx *MsgContext, userID string, text string, flag string) {
	rawID, idType := pa.mustExtractID(userID)

	if idType != QQUidPerson {
		return
	}

	for _, i := range ctx.Dice.ExtList {
		if i.OnMessageSend != nil {
			i.callWithJsCheck(ctx.Dice, func() {
				i.OnMessageSend(ctx, &Message{
					Message:     text,
					MessageType: "private",
					Platform:    pa.EndPoint.Platform,
					Sender: SenderBase{
						Nickname: pa.EndPoint.Nickname,
						UserID:   pa.EndPoint.UserID,
					},
				}, flag)
			})
		}
	}

	type GroupMessageParams struct {
		MessageType string `json:"message_type"`
		UserID      int64  `json:"user_id"`
		Message     string `json:"message"`
	}

	text = textAssetsConvert(text)
	texts := textSplit(text)
	for _, subText := range texts {
		a, _ := json.Marshal(oneBotCommand{
			Action: "send_msg",
			Params: GroupMessageParams{
				MessageType: "private",
				UserID:      rawID,
				Message:     subText,
			},
		})
		doSleepQQ(ctx)
		socketSendText(pa.Socket, string(a))
	}
}

func (pa *PlatformAdapterGocq) SendToGroup(ctx *MsgContext, groupID string, text string, flag string) {
	if groupID == "" {
		return
	}
	rawID, idType := pa.mustExtractID(groupID)
	if idType == 0 {
		pa.SendToChannelGroup(ctx, groupID, text, flag)
		return
	}

	if ctx.Session.ServiceAtNew[groupID] != nil {
		for _, i := range ctx.Session.ServiceAtNew[groupID].ActivatedExtList {
			if i.OnMessageSend != nil {
				i.callWithJsCheck(ctx.Dice, func() {
					i.OnMessageSend(ctx, &Message{
						Message:     text,
						MessageType: "group",
						Platform:    pa.EndPoint.Platform,
						GroupID:     groupID,
						Sender: SenderBase{
							Nickname: pa.EndPoint.Nickname,
							UserID:   pa.EndPoint.UserID,
						},
					}, flag)
				})
			}
		}
	}

	type GroupMessageParams struct {
		GroupID int64  `json:"group_id"`
		Message string `json:"message"`
	}

	type GroupArrMessageParams struct {
		GroupID int64         `json:"group_id"`
		Message []interface{} `json:"message"` // 消息内容，原则上是OneBotV11MsgItem但是实际很杂说不清
	}

	text = textAssetsConvert(text)
	texts := textSplit(text)

	for index, subText := range texts {
		var a []byte
		if pa.useArrayMessage {
			a, _ = json.Marshal(oneBotCommand{
				Action: "send_group_msg",
				Params: GroupArrMessageParams{
					GroupID: rawID,
					Message: OneBot11CqMessageToArrayMessage(subText),
				},
			})
		} else {
			a, _ = json.Marshal(oneBotCommand{
				Action: "send_group_msg",
				Params: GroupMessageParams{
					rawID,
					subText, // "golang client test",
				},
			})
		}

		if len(texts) > 1 && index != 0 {
			doSleepQQ(ctx)
		}
		socketSendText(pa.Socket, string(a))
	}
}

func (pa *PlatformAdapterGocq) SendFileToPerson(ctx *MsgContext, userID string, path string, _ string) {
	rawID, idType := pa.mustExtractID(userID)
	if idType != QQUidPerson {
		return
	}

	dice := pa.Session.Parent
	// 路径可以是 http/base64/本地路径，但 gocq 的文件上传只支持本地文件，所以临时下载到本地
	fileName, temp, err := dice.ExtractLocalTempFile(path)
	defer func(name string) {
		_ = os.Remove(name)
	}(temp.Name())

	if err != nil {
		dice.Logger.Errorf("尝试发送文件[path=%s]出错: %s", path, err.Error())
		return
	}

	type uploadPrivateFileParams struct {
		UserID int64  `json:"user_id"`
		File   string `json:"file"`
		Name   string `json:"name"`
	}
	a, _ := json.Marshal(oneBotCommand{
		Action: "upload_private_file",
		Params: uploadPrivateFileParams{
			UserID: rawID,
			File:   temp.Name(),
			Name:   fileName,
		},
	})
	doSleepQQ(ctx)
	socketSendText(pa.Socket, string(a))
}

func (pa *PlatformAdapterGocq) SendFileToGroup(ctx *MsgContext, groupID string, path string, flag string) {
	if groupID == "" {
		return
	}
	rawID, idType := pa.mustExtractID(groupID)
	if idType == 0 {
		// qq频道尚不支持文件发送，降级
		pa.SendToChannelGroup(ctx, groupID, fmt.Sprintf("[尝试发送文件: %s，但不支持]", filepath.Base(path)), flag)
		return
	}

	dice := pa.Session.Parent
	// 路径可以是 http/base64/本地路径，但 gocq 的文件上传只支持本地文件，所以临时下载到本地
	fileName, temp, err := dice.ExtractLocalTempFile(path)
	defer func(name string) {
		_ = os.Remove(name)
	}(temp.Name())

	if err != nil {
		dice.Logger.Errorf("尝试发送文件[path=%s]出错: %s", path, err.Error())
		return
	}

	type uploadGroupFileParams struct {
		GroupID int64  `json:"group_id"`
		File    string `json:"file"`
		Name    string `json:"name"`
	}
	a, _ := json.Marshal(oneBotCommand{
		Action: "upload_group_file",
		Params: uploadGroupFileParams{
			GroupID: rawID,
			File:    temp.Name(),
			Name:    fileName,
		},
	})
	doSleepQQ(ctx)
	socketSendText(pa.Socket, string(a))
}

// SetGroupAddRequest 同意加群
func (pa *PlatformAdapterGocq) SetGroupAddRequest(flag string, subType string, approve bool, reason string) {
	type DetailParams struct {
		Flag    string `json:"flag"`
		SubType string `json:"sub_type"`
		Approve bool   `json:"approve"`
		Reason  string `json:"reason"`
	}

	a, _ := json.Marshal(oneBotCommand{
		Action: "set_group_add_request",
		Params: DetailParams{
			Flag:    flag,
			SubType: subType,
			Approve: approve,
			Reason:  reason,
		},
	})

	socketSendText(pa.Socket, string(a))
}

func (pa *PlatformAdapterGocq) getCustomEcho() string {
	if pa.customEcho > -10 {
		pa.customEcho = -10
	}
	pa.customEcho--
	return strconv.FormatInt(pa.customEcho, 10)
}

func (pa *PlatformAdapterGocq) waitEcho(echo any, beforeWait func()) *MessageQQ {
	// pa.echoList = append(pa.echoList, )
	ch := make(chan *MessageQQ, 1)

	if pa.echoMap == nil {
		pa.echoMap = new(SyncMap[any, chan *MessageQQ])
	}

	// 注: 之所以这样是因为echo是json.RawMessage
	e := lo.Must(json.Marshal(echo))
	pa.echoMap.Store(string(e), ch)

	beforeWait()
	return <-ch
}

func (pa *PlatformAdapterGocq) waitEcho2(echo any, value interface{}, beforeWait func(emi *echoMapInfo)) error {
	if pa.echoMap2 == nil {
		pa.echoMap2 = new(SyncMap[any, *echoMapInfo])
	}

	emi := &echoMapInfo{ch: make(chan string, 1)}
	beforeWait(emi)

	pa.echoMap2.Store(echo, emi)
	val := <-emi.ch
	if val == "" {
		return errors.New("超时")
	}
	return json.Unmarshal([]byte(val), value)
}

// GetGroupMemberInfo 获取群成员信息
func (pa *PlatformAdapterGocq) GetGroupMemberInfo(groupID string, userID string) *OnebotUserInfo {
	type DetailParams struct {
		GroupID string `json:"group_id"`
		UserID  string `json:"user_id"`
		NoCache bool   `json:"no_cache"`
	}

	echo := pa.getCustomEcho()

	a, _ := json.Marshal(oneBotCommand{
		Action: "get_group_member_info",
		Params: DetailParams{
			GroupID: groupID,
			UserID:  userID,
			NoCache: false,
		},
		Echo: echo,
	})

	msg := pa.waitEcho(echo, func() {
		socketSendText(pa.Socket, string(a))
	})

	d := msg.Data
	if msg.Data == nil {
		return &OnebotUserInfo{}
	}

	return &OnebotUserInfo{
		Nickname: d.Nickname,
		UserID:   string(d.UserID),
		GroupID:  string(d.GroupID),
		Card:     d.Card,
	}
}

// GetStrangerInfo 获取陌生人信息
func (pa *PlatformAdapterGocq) GetStrangerInfo(userID string) *OnebotUserInfo {
	type DetailParams struct {
		UserID  string `json:"user_id"`
		NoCache bool   `json:"no_cache"`
	}

	echo := pa.getCustomEcho()

	a, _ := json.Marshal(oneBotCommand{
		Action: "get_stranger_info",
		Params: DetailParams{
			UserID:  userID,
			NoCache: false,
		},
		Echo: echo,
	})

	msg := pa.waitEcho(echo, func() {
		socketSendText(pa.Socket, string(a))
	})

	d := msg.Data
	if msg.Data == nil {
		return &OnebotUserInfo{}
	}

	return &OnebotUserInfo{
		Nickname: d.Nickname,
		UserID:   string(d.UserID),
	}
}

func (pa *PlatformAdapterGocq) SetGroupCardName(ctx *MsgContext, name string) {
	groupIDRaw, idType := pa.mustExtractID(ctx.Group.GroupID)
	if idType != QQUidGroup {
		return
	}
	userIDRaw, idType := pa.mustExtractID(ctx.Player.UserID)
	if idType != QQUidPerson {
		return
	}

	pa.SetGroupCardNameBase(groupIDRaw, userIDRaw, name)
}

// SetGroupCardNameBase 设置群名片
func (pa *PlatformAdapterGocq) SetGroupCardNameBase(groupID int64, userID int64, card string) {
	type DetailParams struct {
		GroupID int64  `json:"group_id"`
		UserID  int64  `json:"user_id"`
		Card    string `json:"card"`
	}

	a, _ := json.Marshal(struct {
		Action string       `json:"action"`
		Params DetailParams `json:"params"`
	}{
		"set_group_card",
		DetailParams{
			GroupID: groupID,
			UserID:  userID,
			Card:    card,
		},
	})

	socketSendText(pa.Socket, string(a))
}

func (pa *PlatformAdapterGocq) SetFriendAddRequest(flag string, approve bool, remark string, reaseon string) {
	type DetailParams struct {
		Flag    string `json:"flag"`
		Remark  string `json:"remark"` // 备注名
		Approve bool   `json:"approve"`
		Reason  string `json:"reason"`
	}

	a, _ := json.Marshal(struct {
		Action string       `json:"action"`
		Params DetailParams `json:"params"`
	}{
		"set_friend_add_request",
		DetailParams{
			Flag:    flag,
			Approve: approve,
			Remark:  remark,
			Reason:  reaseon,
		},
	})

	socketSendText(pa.Socket, string(a))
}

func (pa *PlatformAdapterGocq) DeleteFriend(_ *MsgContext, id string) {
	friendID, idType := pa.mustExtractID(id)
	if idType != QQUidPerson {
		return
	}
	type GroupMessageParams struct {
		FriendID int64 `json:"friend_id"`
	}

	a, _ := json.Marshal(oneBotCommand{
		Action: "friend_id",
		Params: GroupMessageParams{
			friendID,
		},
	})

	socketSendText(pa.Socket, string(a))
}

func (pa *PlatformAdapterGocq) QuitGroup(_ *MsgContext, id string) {
	groupID, idType := pa.mustExtractID(id)
	if idType != QQUidGroup {
		return
	}
	type GroupMessageParams struct {
		GroupID   int64 `json:"group_id"`
		IsDismiss bool  `json:"is_dismiss"`
	}

	a, _ := json.Marshal(oneBotCommand{
		Action: "set_group_leave",
		Params: GroupMessageParams{
			groupID,
			false,
		},
	})

	socketSendText(pa.Socket, string(a))
}

func (pa *PlatformAdapterGocq) MemberBan(_ string, _ string, _ int64) {}

func (pa *PlatformAdapterGocq) MemberKick(_ string, _ string) {}

func (pa *PlatformAdapterGocq) GetLoginInfo() {
	a, _ := json.Marshal(struct {
		Action string `json:"action"`
		Echo   int64  `json:"echo"`
	}{
		Action: "get_login_info",
		Echo:   -1,
	})

	socketSendText(pa.Socket, string(a))
}

func textSplit(input string) []string {
	re := regexp.MustCompile(`\[CQ:poke,(.+?)]`) // [img:] 或 [图:]
	m := re.FindAllStringIndex(input, -1)

	// 注: 临时方案，后期对CQ码在消息转换时进行统一处理
	var poke []string
	if m != nil {
		for i := len(m) - 1; i >= 0; i-- {
			span := m[i]
			poke = append(poke, input[span[0]:span[1]])
			input = input[0:span[0]] + input[span[1]:]
		}
	}

	splits := utils.SplitLongText(input, 2000)
	splits = append(splits, poke...)

	return splits
}

// 以下都是CQ码处理
// 相对路径不能在这里实现，不然遇到嵌套调用无法展开
func textAssetsConvert(s string) string {
	solve2 := func(text string) string {
		re := regexp.MustCompile(`\[(img|图|文本|text|语音|voice|视频|video):(.+?)]`) // [img:] 或 [图:]
		m := re.FindStringSubmatch(text)
		if m != nil {
			fn := m[2]
			cqType := "image"
			if m[1] == "voice" || m[1] == "语音" {
				cqType = "record"
			}
			if m[1] == "video" || m[1] == "视频" {
				cqType = "video"
			}

			if strings.HasPrefix(fn, "file://") || strings.HasPrefix(fn, "http://") || strings.HasPrefix(fn, "https://") {
				u, err := url.Parse(fn)
				if err != nil {
					return text
				}
				cq := CQCommand{
					Type: cqType,
					Args: map[string]string{"file": u.String(), "cache": "0"},
				}
				return cq.Compile()
			}

			afn, err := filepath.Abs(fn)
			if err != nil {
				return text // 不是文件路径，不管
			}
			cwd, _ := os.Getwd()
			if !strings.HasPrefix(afn, cwd) && !strings.HasPrefix(afn, os.TempDir()) {
				return "[图片/文件指向的不是当前程序目录或临时文件目录，已禁止]"
			}

			if _, err := os.Stat(afn); errors.Is(err, os.ErrNotExist) {
				return "[找不到图片/文件]"
			}
			// 这里使用绝对路径，windows上gocqhttp会裁掉一个斜杠，所以我这里加一个
			if runtime.GOOS == `windows` {
				afn = "/" + afn
			}
			u := url.URL{
				Scheme: "file",
				Path:   afn,
			}
			cq := CQCommand{
				Type: cqType,
				Args: map[string]string{"file": u.String()},
			}
			return cq.Compile()
		}
		return text
	}

	solve := func(cq *CQCommand) {
		// if cq.Type == "image" || cq.Type == "voice" {
		fn, exists := cq.Args["file"]
		_, urlExists := cq.Args["url"]

		if exists {
			if urlExists {
				// 另一个问题，这个会爆出路径
				// .text [CQ:image,file=eff8428fa4034480d20631e0e37d10d2.image,url=http://]
				return
			}
			if strings.HasPrefix(fn, "file://") || strings.HasPrefix(fn, "http://") || strings.HasPrefix(fn, "https://") || strings.HasPrefix(fn, "base64://") {
				return
			}
			if strings.HasSuffix(fn, ".image") && len(fn) == 32+6 {
				// 举例
				// [CQ:image,file=eff8428fa4034480d20631e0e37d10d2.image,subType=1,url=https://gchat.qpic.cn/gchatpic_new/303451945/4186699433-2282331144-EFF8428FA4034480D20631E0E37D10D2/0?term=2&is_origin=0]
				return
			}

			afn, err := filepath.Abs(fn)
			if err != nil {
				return // 不是文件路径，不管
			}
			cwd, _ := os.Getwd()

			if strings.HasPrefix(afn, cwd) || strings.HasPrefix(afn, os.TempDir()) {
				if _, err := os.Stat(afn); errors.Is(err, os.ErrNotExist) {
					cq.Overwrite = "[CQ码找不到文件]"
				} else {
					// 这里使用绝对路径，windows上gocqhttp会裁掉一个斜杠，所以我这里加一个
					if runtime.GOOS == `windows` {
						afn = "/" + afn
					}
					u := url.URL{
						Scheme: "file",
						Path:   afn,
					}
					cq.Args["file"] = u.String()
				}
			} else {
				cq.Overwrite = "[CQ码读取的不是当前目录文件或临时文件，可能是恶意行为，已禁止]"
			}
		}
		//}
	}

	text := strings.ReplaceAll(s, `\n`, "\n")
	text = ImageRewrite(text, solve2)
	return CQRewrite(text, solve)
}
