package main

import (
	"encoding/json"
	"github.com/sacOO7/gowebsocket"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sort"
	"strings"
	"time"
)

// 2022/02/03 11:47:42 Recieved message {"font":0,"message":"test","message_id":-487913662,"message_type":"private","post_type":"message","raw_message":"test","self_id":1001,"sender":{"age":0,"nickname":"鏈ㄨ惤","sex":"unknown","user_id":1002},"sub_type":"friend","target_id":1001,"time":1643860062,"user_id":1002}
// {"anonymous":null,"font":0,"group_id":111,"message":"qqq","message_id":884917177,"message_seq":1434,"message_type":"group","post_type":"message","raw_message":"qqq","self_id":1001,"sender":{"age":0,"area":"","card":"","level":"","nickname":"鏈ㄨ惤","role":"member","sex":"unknown","title":"","user_id":1002},"sub_type":"normal","time":1643863961,"user_id":1002}
// {"anonymous":null,"font":0,"group_id":111,"message":"[CQ:at,qq=1001]   .r test","message_id":888971055,"message_seq":1669,"message_type":"group","post_type":"message","raw_message":"[CQ:at,qq=1001]   .r test","self_id":1001,"sender":{"age":0,"area":"","card":"","level":"","nickname":"鏈ㄨ惤","role":"member","sex":"unknown","title":"","user_id":1002},"sub_type":"normal","time":1644127751,"user_id":1002}

func replyPerson(s *IMSession, userId int64, text string) {
	replyPersonRaw(s, userId, text, "")
}

func replyPersonRaw(s *IMSession, userId int64, text string, flag string) {
	for _, i := range s.parent.extList {
		if i.OnMessageSend != nil {
			i.OnMessageSend(s, "private", userId, text, flag)
		}
	}
	time.Sleep(time.Duration((0.8 + rand.Float64()) * float64(time.Second)))

	type GroupMessageParams struct {
		MessageType string        `json:"message_type"`
		UserId int64             `json:"user_id"`
		Message string            `json:"message"`
	}

	a, _ := json.Marshal(struct {
		Action string             `json:"action"`
		Params GroupMessageParams `json:"params"`
	}{
		Action: "send_msg",
		Params: GroupMessageParams{
			MessageType: "private",
			UserId:      userId,
			Message:     text,
		},
	})

	s.Socket.SendText(string(a))
}

func GetGroupInfo(socket *gowebsocket.Socket, groupId int64) {
	type GroupMessageParams struct {
		GroupId int64  `json:"group_id"`
	}

	a, _ := json.Marshal(struct {
		Action string             `json:"action"`
		Params GroupMessageParams `json:"params"`
		Echo int64 `json:"echo"`
	}{
		"get_group_info",
		GroupMessageParams{
			groupId,
		},
		-2,
	})
	socket.SendText(string(a))
}


func quitGroup(s *IMSession, groupId int64) {
	type GroupMessageParams struct {
		GroupId int64  `json:"group_id"`
	}

	a, _ := json.Marshal(struct {
		Action string             `json:"action"`
		Params GroupMessageParams `json:"params"`
	}{
		"set_group_leave",
		GroupMessageParams{
			groupId,
		},
	})
	s.Socket.SendText(string(a))
}

func replyGroup(s *IMSession, groupId int64, text string) {
	replyGroupRaw(s, groupId, text, "")
}

func replyGroupRaw(s *IMSession, groupId int64, text string, flag string) {
	if s.ServiceAt[groupId] != nil {
		for _, i := range s.ServiceAt[groupId].ActivatedExtList {
			if i.OnMessageSend != nil {
				i.OnMessageSend(s, "group", groupId, text, flag)
			}
		}
	}

	time.Sleep(time.Duration((0.8 + rand.Float64()) * float64(time.Second)))

	type GroupMessageParams struct {
		GroupId int64  `json:"group_id"`
		Message string `json:"message"`
	}

	a, _ := json.Marshal(struct {
		Action string             `json:"action"`
		Params GroupMessageParams `json:"params"`
	}{
		"send_group_msg",
		GroupMessageParams{
			groupId,
			text, // "golang client test",
		},
	})
	s.Socket.SendText(string(a))
}

func replyToSenderRaw(s *IMSession, msg *Message, text string, flag string) {
	inGroup := msg.MessageType == "group"
	if inGroup {
		replyGroupRaw(s, msg.GroupId, text, flag)
	} else {
		replyPersonRaw(s, msg.Sender.UserId, text, flag)
	}
}

func replyToSender(s *IMSession, msg *Message, text string) {
	replyToSenderRaw(s, msg, text, "")
}

func (s *IMSession) GetLoginInfo() {
	a, _ := json.Marshal(struct {
		Action string `json:"action"`
		Echo int64 `json:"echo"`
	}{
		Action: "get_login_info",
		Echo: -1,
	})
	s.Socket.SendText(string(a))
}

type Sender struct {
	Age      int32  `json:"age"`
	Card     string `json:"card"`
	Nickname string `json:"nickname"`
	Role     string `json:"owner"`
	UserId   int64  `json:"user_id"`
}

type Message struct {
	MessageId     int64  `json:"message_id"`
	MessageType   string `json:"message_type"` // group
	Sender        Sender `json:"sender"`       // 发送者
	RawMessage    string `json:"raw_message"`
	Message       string `json:"message"` // 消息内容
	Time          int64  `json:"time"`    // 发送时间
	MetaEventType string `json:"meta_event_type"`
	GroupId       int64  `json:"group_id"` // 群号

	Data *struct {
		// 个人信息
		Nickname string `json:"nickname"`
		UserId   int64  `json:"user_id"`

		// 群信息
		GroupId       int64  `json:"group_id"` // 群号
		GroupCreateTime       uint32  `json:"group_create_time"` // 群号
		MemberCount int64 `json:"member_count"`
		GroupName string `json:"group_name"`
	} `json:"data"`
	Retcode int64 `json:"retcode"`
	//Status string `json:"status"`
	Echo int `json:"echo"`
}

type PlayerInfo struct {
	UserId         int64 `yaml:"userId"`
	Name           string
	ValueNumMap    map[string]int64  `yaml:"valueNumMap"`
	ValueStrMap    map[string]string `yaml:"valueStrMap"`
	RpToday        int               `yaml:"rpToday"`
	RpTime         string            `yaml:"rpTime"`
	lastUpdateTime int64             `yaml:"lastUpdateTime"`

	// level int 权限
	DiceSideNum    int `yaml:"diceSideNum"` // 面数，为0时等同于d100
	TempValueAlias *map[string][]string `yaml:"-"`
}

func (i *PlayerInfo) GetValueNameByAlias(s string, alias map[string][]string) string {
	name := s

	if alias == nil {
		alias = *i.TempValueAlias
	}

	for k, v := range alias {
		if strings.EqualFold(s, k) {
			break // 名字本身就是确定值，不用修改
		}
		for _, i := range v {
			if strings.EqualFold(s, i) {
				name = k
				break
			}
		}
	}

	return name
}

func (i *PlayerInfo) SetValueInt64(s string, sanNew int64, alias map[string][]string) {
	name := i.GetValueNameByAlias(s, alias)
	i.ValueNumMap[name] = sanNew
}

func (i *PlayerInfo) GetValueInt64(s string, alias map[string][]string) (int64, bool) {
	name := i.GetValueNameByAlias(s, alias)
	v, e := i.ValueNumMap[name]
	return v, e
}

type ServiceAtItem struct {
	Active bool `json:"active" yaml:"active"` // 需要能记住配置，故有此选项
	ActivatedExtList []*ExtInfo `yaml:"activatedExtList"` // 当前群开启的扩展列表
	Players map[int64]*PlayerInfo // 群员信息

	LogCurName  string   `yaml:"logCurFile"`
	LogOn       bool     `yaml:"logOn"`
	GroupId int64 `yaml:"groupId"`
	GroupName   string   `yaml:"groupName"`

	// http://www.antagonistes.com/files/CoC%20CheatSheet.pdf
	//RuleCriticalSuccessValue *int64 // 大成功值，1默认
	//RuleFumbleValue *int64 // 大失败值 96默认
}

func (i *ServiceAtItem) GetFumbleValue() int64 {
	return 96
}

func (i *ServiceAtItem) getCriticalSuccessValue() int64 {
	return 1
}



type IMSession struct {
	Socket   *gowebsocket.Socket `yaml:"-"`
	Nickname string `yaml:"-"`
	UserId   int64 `yaml:"userId"`
	parent   *Dice `yaml:"-"`

	ServiceAt map[int64]*ServiceAtItem `json:"serviceAt" yaml:"serviceAt"`
	CommandIndex int64 `yaml:"-"`
	//GroupId int64 `json:"group_id"`
}


type ByLength []string

func (s ByLength) Len() int {
	return len(s)
}
func (s ByLength) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByLength) Less(i, j int) bool {
	return len(s[i]) > len(s[j])
}

func (s *IMSession) serve() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	session := s
	socket := gowebsocket.New("ws://127.0.0.1:6700")
	session.Socket = &socket

	socket.OnConnected = func(socket gowebsocket.Socket) {
		log.Println("Connected to server")
		//  {"data":{"nickname":"闃斧鐗岃�佽檸鏈�","user_id":1001},"retcode":0,"status":"ok"}
		s.GetLoginInfo()
	}

	socket.OnConnectError = func(err error, socket gowebsocket.Socket) {
		log.Println("Recieved connect error ", err)
	}

	socket.OnTextMessage = func(message string, socket gowebsocket.Socket) {
		msg := new(Message)
		log.Println("Recieved message " + message)
		err := json.Unmarshal([]byte(message), msg)

		if err == nil {
			// 获得用户信息
			if msg.Echo == -1 {
				session.UserId = msg.Data.UserId
				session.Nickname = msg.Data.Nickname
				log.Println("User info received.")
				return
			}

			// 获得群信息
			if msg.Echo == -2 {
				group := session.ServiceAt[msg.Data.GroupId]
				if group != nil {
					group.GroupName = msg.Data.GroupName
					group.GroupId = msg.Data.GroupId
				}
				log.Println("Group info received: ", msg.Data.GroupName)
				return
			}

			// 处理命令
			if msg.MessageType == "group" || msg.MessageType == "private" {
				var cmdLst []string

				if s.parent.CommandCompatibleMode {
					for k := range session.parent.cmdMap {
						cmdLst = append(cmdLst, k)
					}

					sa := session.ServiceAt[msg.GroupId]
					if sa != nil && sa.Active {
						for _, i := range sa.ActivatedExtList {
							for k := range i.cmdMap {
								cmdLst = append(cmdLst, k)
							}
						}
					}
					sort.Sort(ByLength(cmdLst))
				}

				// 收到信息回调
				sa := session.ServiceAt[msg.GroupId]
				if sa != nil && sa.Active {
					for _, i := range sa.ActivatedExtList {
						if i.OnMessageReceived != nil {
							i.OnMessageReceived(session, msg)
						}
					}
				}

				msgInfo := CommandParse(msg.Message, s.parent.CommandCompatibleMode, cmdLst)
				//log.Println(msgInfo)

				//if msg.Sender.UserId == 303451945 || msg.Sender.UserId == 184023393 {
				//if msg.MessageType == "group" {
				if msgInfo != nil {
					f := func() {
						//defer func() {
						//	if r := recover(); r != nil {
						//		//  + fmt.Sprintf("%s", r)
						//		core.GetLogger().Error(r)
						//		fmt.Println(r)
						//		replyToSender(s, msg, "已从核心崩溃中恢复，请带指令联系开发者。注意不要重复发送本指令以免风控。")
						//	}
						//}()
						session.commandSolve(session, msg, msgInfo)
					}
					go f()

					//c, _ := json.Marshal(msgInfo)
					//text := fmt.Sprintf("指令测试，来自群%d - %s(%d)：参数 %s", msg.GroupId, msg.Sender.Nickname, msg.Sender.UserId, c);
					//replyGroup(Socket, 11, text)
				} else {
					//text := fmt.Sprintf("信息 来自群%d - %s(%d)：%s", msg.GroupId, msg.Sender.Nickname, msg.Sender.UserId, msg.Message);
					//replyGroup(Socket, 22, text)
				}
				//}
				//}
			}
		} else {
			log.Println("error" + err.Error())
		}
	}

	socket.OnBinaryMessage = func(data []byte, socket gowebsocket.Socket) {
		log.Println("Recieved binary data ", data)
	}

	socket.OnPingReceived = func(data string, socket gowebsocket.Socket) {
		log.Println("Recieved ping " + data)
	}

	socket.OnPongReceived = func(data string, socket gowebsocket.Socket) {
		log.Println("Recieved pong " + data)
	}

	socket.OnDisconnected = func(err error, socket gowebsocket.Socket) {
		log.Println("Disconnected from server ")
		return
	}

	socket.Connect()

	for {
		select {
		case <-interrupt:
			log.Println("interrupt")
			socket.Close()
			return
		}
	}
}

func (s *IMSession) commandSolve(session *IMSession, msg *Message, cmdArgs *CmdArgs) {
	//c, _ := json.Marshal(msgInfo)
	//text := fmt.Sprintf("指令测试，来自群%d - %s(%d)：参数 %s", msg.GroupId, msg.Sender.Nickname, msg.Sender.UserId, c);
	//replyGroup(Socket, 111, text)

	cmdArgs.AmIBeMentioned = false
	for _, i := range cmdArgs.At {
		if i.UserId == session.UserId {
			cmdArgs.AmIBeMentioned = true
			break
		}
	}

	tryItemSolve := func (item *CmdItemInfo) bool {
		if item != nil {
			ret := item.solve(session, msg, cmdArgs)
			if ret.success {
				return true
			}
		}
		return false
	}

	sa := session.ServiceAt[msg.GroupId]
	if sa != nil && sa.Active {
		for _, i := range sa.ActivatedExtList {
			if i.OnCommandReceived != nil {
				i.OnCommandReceived(session, msg, cmdArgs)
			}
		}
	}

	item := session.parent.cmdMap[cmdArgs.Command]
	if tryItemSolve(item) {
		return
	}

	if sa != nil && sa.Active {
		for _, i := range sa.ActivatedExtList {
			item := i.cmdMap[cmdArgs.Command]
			if tryItemSolve(item) {
				return
			}
		}
	}

	if msg.MessageType == "private" {
		for _, i := range session.parent.extList {
			if i.ActiveOnPrivate {
				item := i.cmdMap[cmdArgs.Command]
				if tryItemSolve(item) {
					return
				}
			}
		}
	}
}
