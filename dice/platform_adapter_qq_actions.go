package dice

import (
	"encoding/json"
	"errors"
	"github.com/sacOO7/gowebsocket"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type oneBotCommand struct {
	Action string      `json:"action"`
	Params interface{} `json:"params"`
	Echo   int64       `json:"echo"`
}

type QQUidType int

const (
	QQUidPerson        QQUidType = 1
	QQUidGroup         QQUidType = 2
	QQUidChannelPerson QQUidType = 3
	QQUidChannelGroup  QQUidType = 4
)

func (pa *PlatformAdapterGocq) mustExtractId(id string) (int64, QQUidType) {
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

func (pa *PlatformAdapterGocq) mustExtractChannelId(id string) (string, QQUidType) {
	if strings.HasPrefix(id, "QQ-CH:") {
		return id[len("QQ-CH:"):], QQUidChannelPerson
	}
	if strings.HasPrefix(id, "QQ-CH-Group:") {
		return id[len("QQ-CH-Group:"):], QQUidChannelGroup
	}
	return "", 0
}

// GetGroupInfoAsync 异步获取群聊信息
func (pa *PlatformAdapterGocq) GetGroupInfoAsync(groupId string) {
	type GroupMessageParams struct {
		GroupId int64 `json:"group_id"`
	}
	realGroupId, type_ := pa.mustExtractId(groupId)
	if type_ != QQUidGroup {
		return
	}

	a, _ := json.Marshal(oneBotCommand{
		"get_group_info",
		GroupMessageParams{
			realGroupId,
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
func (pa *PlatformAdapterGocq) GetGroupInfo(groupId string) *OnebotGroupInfo {
	type GroupMessageParams struct {
		GroupId int64 `json:"group_id"`
	}
	realGroupId, type_ := pa.mustExtractId(groupId)
	if type_ != QQUidGroup {
		return nil
	}

	echo := pa.getCustomEcho()
	a, _ := json.Marshal(oneBotCommand{
		"get_group_info",
		GroupMessageParams{
			realGroupId,
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
			//core.GetLogger().Error(r)
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
			//core.GetLogger().Error(r)
		}
	}()

	if socket != nil {
		socket.SendBinary(data)
	}
}

func doSleepQQ(ctx *MsgContext) {
	if ctx.Dice != nil {
		d := ctx.Dice
		offset := d.MessageDelayRangeEnd - d.MessageDelayRangeStart
		time.Sleep(time.Duration((d.MessageDelayRangeStart + rand.Float64()*offset) * float64(time.Second)))
	} else {
		time.Sleep(time.Duration((0.4 + rand.Float64()/2) * float64(time.Second)))
	}
}

func (pa *PlatformAdapterGocq) SendToPerson(ctx *MsgContext, userId string, text string, flag string) {
	rawId, type_ := pa.mustExtractId(userId)

	if type_ != QQUidPerson {
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
						UserId:   pa.EndPoint.UserId,
					},
				},
					flag)
			})
		}
	}

	type GroupMessageParams struct {
		MessageType string `json:"message_type"`
		UserId      int64  `json:"user_id"`
		Message     string `json:"message"`
	}

	text = textAssetsConvert(text)
	texts := textSplit(text)
	for _, subText := range texts {
		a, _ := json.Marshal(oneBotCommand{
			Action: "send_msg",
			Params: GroupMessageParams{
				MessageType: "private",
				UserId:      rawId,
				Message:     subText,
			},
		})
		doSleepQQ(ctx)
		socketSendText(pa.Socket, string(a))
	}
}

func (pa *PlatformAdapterGocq) SendToGroup(ctx *MsgContext, groupId string, text string, flag string) {
	if groupId == "" {
		return
	}
	rawId, type_ := pa.mustExtractId(groupId)
	if type_ == 0 {
		pa.SendToChannelGroup(ctx, groupId, text, flag)
		return
	}

	if ctx.Session.ServiceAtNew[groupId] != nil {
		for _, i := range ctx.Session.ServiceAtNew[groupId].ActivatedExtList {
			if i.OnMessageSend != nil {
				i.callWithJsCheck(ctx.Dice, func() {
					i.OnMessageSend(ctx, &Message{
						Message:     text,
						MessageType: "group",
						Platform:    pa.EndPoint.Platform,
						GroupId:     groupId,
						Sender: SenderBase{
							Nickname: pa.EndPoint.Nickname,
							UserId:   pa.EndPoint.UserId,
						},
					}, flag)
				})
			}
		}
	}

	type GroupMessageParams struct {
		GroupId int64  `json:"group_id"`
		Message string `json:"message"`
	}

	text = textAssetsConvert(text)
	texts := textSplit(text)

	for index, subText := range texts {
		a, _ := json.Marshal(oneBotCommand{
			Action: "send_group_msg",
			Params: GroupMessageParams{
				rawId,
				subText, // "golang client test",
			},
		})

		if len(texts) > 1 && index != 0 {
			doSleepQQ(ctx)
		}
		socketSendText(pa.Socket, string(a))
	}
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

func (pa *PlatformAdapterGocq) getCustomEcho() int64 {
	if pa.customEcho > -10 {
		pa.customEcho = -10
	}
	pa.customEcho -= 1
	return pa.customEcho
}

func (pa *PlatformAdapterGocq) waitEcho(echo int64, beforeWait func()) *MessageQQ {
	//pa.echoList = append(pa.echoList, )
	ch := make(chan *MessageQQ, 1)

	if pa.echoMap == nil {
		pa.echoMap = new(SyncMap[int64, chan *MessageQQ])
	}
	pa.echoMap.Store(echo, ch)

	beforeWait()
	return <-ch
}

func (pa *PlatformAdapterGocq) waitEcho2(echo int64, value interface{}, beforeWait func(emi *echoMapInfo)) error {
	if pa.echoMap2 == nil {
		pa.echoMap2 = new(SyncMap[int64, *echoMapInfo])
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
func (pa *PlatformAdapterGocq) GetGroupMemberInfo(GroupId int64, UserId int64) *OnebotUserInfo {
	type DetailParams struct {
		GroupId int64 `json:"group_id"`
		UserId  int64 `json:"user_id"`
		NoCache bool  `json:"no_cache"`
	}

	echo := pa.getCustomEcho()

	a, _ := json.Marshal(oneBotCommand{
		Action: "get_group_member_info",
		Params: DetailParams{
			GroupId: GroupId,
			UserId:  UserId,
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
		UserId:   d.UserId,
		GroupId:  d.GroupId,
		Card:     d.Card,
	}
}

// GetStrangerInfo 获取陌生人信息
func (pa *PlatformAdapterGocq) GetStrangerInfo(UserId int64) *OnebotUserInfo {
	type DetailParams struct {
		UserId  int64 `json:"user_id"`
		NoCache bool  `json:"no_cache"`
	}

	echo := pa.getCustomEcho()

	a, _ := json.Marshal(oneBotCommand{
		Action: "get_stranger_info",
		Params: DetailParams{
			UserId:  UserId,
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
		UserId:   d.UserId,
	}
}

func (pa *PlatformAdapterGocq) SetGroupCardName(groupId string, userId string, name string) {
	groupIdRaw, type_ := pa.mustExtractId(groupId)
	if type_ != QQUidGroup {
		return
	}
	userIdRaw, type_ := pa.mustExtractId(userId)
	if type_ != QQUidPerson {
		return
	}

	pa.SetGroupCardNameBase(groupIdRaw, userIdRaw, name)
}

// SetGroupCardNameBase 设置群名片
func (pa *PlatformAdapterGocq) SetGroupCardNameBase(GroupId int64, UserId int64, Card string) {
	type DetailParams struct {
		GroupId int64  `json:"group_id"`
		UserId  int64  `json:"user_id"`
		Card    string `json:"card"`
	}

	a, _ := json.Marshal(struct {
		Action string       `json:"action"`
		Params DetailParams `json:"params"`
	}{
		"set_group_card",
		DetailParams{
			GroupId: GroupId,
			UserId:  UserId,
			Card:    Card,
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

func (pa *PlatformAdapterGocq) DeleteFriend(ctx *MsgContext, id string) {
	friendId, type_ := pa.mustExtractId(id)
	if type_ != QQUidPerson {
		return
	}
	type GroupMessageParams struct {
		FriendId int64 `json:"friend_id"`
	}

	a, _ := json.Marshal(oneBotCommand{
		Action: "friend_id",
		Params: GroupMessageParams{
			friendId,
		},
	})

	socketSendText(pa.Socket, string(a))
}

func (pa *PlatformAdapterGocq) QuitGroup(ctx *MsgContext, id string) {
	groupId, type_ := pa.mustExtractId(id)
	if type_ != QQUidGroup {
		return
	}
	type GroupMessageParams struct {
		GroupId   int64 `json:"group_id"`
		IsDismiss bool  `json:"is_dismiss"`
	}

	a, _ := json.Marshal(oneBotCommand{
		Action: "set_group_leave",
		Params: GroupMessageParams{
			groupId,
			false,
		},
	})

	socketSendText(pa.Socket, string(a))
}

func (pa *PlatformAdapterGocq) MemberBan(groupId string, userId string, last int64) {

}

func (pa *PlatformAdapterGocq) MemberKick(groupId string, userId string) {

}

func (pa *PlatformAdapterGocq) GetLoginInfo() {
	a, _ := json.Marshal(struct {
		Action string `json:"action"`
		Echo   int64  `json:"echo"`
	}{
		Action: "get_login_info",
		Echo:   -1,
	})

	//if s.Socket != nil {
	socketSendText(pa.Socket, string(a))
	//s.Socket.SendText(string(a))
	//}
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

	maxLen := 5000 // 以utf-8计算，1666个汉字
	var splits []string

	var l, r int
	for l, r = 0, maxLen; r < len(input); l, r = r, r+maxLen {
		for !utf8.RuneStart(input[r]) {
			r--
		}
		splits = append(splits, input[l:r])
	}
	splits = append(splits, input[l:])

	splits = append(splits, poke...)

	return splits
}

// 以下都是CQ码处理
// 相对路径不能在这里实现，不然遇到嵌套调用无法展开
func textAssetsConvert(s string) string {
	//var s2 string
	//raw := []byte(`"` + strings.Replace(s, `"`, `\"`, -1) + `"`)
	//err := json.Unmarshal(raw, &s2)
	//if err != nil {
	//	ctx.Dice.Logger.Info(err)
	//	return s
	//}

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
			if strings.HasPrefix(afn, cwd) {
				if _, err := os.Stat(afn); errors.Is(err, os.ErrNotExist) {
					return "[找不到图片/文件]"
				} else {
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
			} else {
				return "[图片/文件指向非当前程序目录，已禁止]"
			}
		}
		return text
	}

	solve := func(cq *CQCommand) {
		//if cq.Type == "image" || cq.Type == "voice" {
		fn, exists := cq.Args["file"]
		if exists {
			if strings.HasPrefix(fn, "file://") || strings.HasPrefix(fn, "http://") || strings.HasPrefix(fn, "https://") || strings.HasPrefix(fn, "base64://") {
				return
			}

			afn, err := filepath.Abs(fn)
			if err != nil {
				return // 不是文件路径，不管
			}
			cwd, _ := os.Getwd()

			if strings.HasPrefix(afn, cwd) {
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
				cq.Overwrite = "[CQ码读取非当前目录文件，可能是恶意行为，已禁止]"
			}
		}
		//}
	}

	text := strings.Replace(s, `\n`, "\n", -1)
	text = ImageRewrite(text, solve2)
	return CQRewrite(text, solve)
}
