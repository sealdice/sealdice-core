package main

import (
	"encoding/json"
	"github.com/sacOO7/gowebsocket"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"time"
)

// 2022/02/03 11:47:42 Recieved message {"font":0,"message":"test","message_id":-487913662,"message_type":"private","post_type":"message","raw_message":"test","self_id":1001,"sender":{"age":0,"nickname":"鏈ㄨ惤","sex":"unknown","user_id":1002},"sub_type":"friend","target_id":1001,"time":1643860062,"user_id":1002}
// {"anonymous":null,"font":0,"group_id":111,"message":"qqq","message_id":884917177,"message_seq":1434,"message_type":"group","post_type":"message","raw_message":"qqq","self_id":1001,"sender":{"age":0,"area":"","card":"","level":"","nickname":"鏈ㄨ惤","role":"member","sex":"unknown","title":"","user_id":1002},"sub_type":"normal","time":1643863961,"user_id":1002}
// {"anonymous":null,"font":0,"group_id":111,"message":"[CQ:at,qq=1001]   .r test","message_id":888971055,"message_seq":1669,"message_type":"group","post_type":"message","raw_message":"[CQ:at,qq=1001]   .r test","self_id":1001,"sender":{"age":0,"area":"","card":"","level":"","nickname":"鏈ㄨ惤","role":"member","sex":"unknown","title":"","user_id":1002},"sub_type":"normal","time":1644127751,"user_id":1002}

func replyGroup(socket *gowebsocket.Socket, groupId int64, text string) {
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
	socket.SendText(string(a))
}

func getLoginInfo(socket gowebsocket.Socket) {
	a, _ := json.Marshal(struct {
		Action string `json:"action"`
	}{
		"get_login_info",
	})
	socket.SendText(string(a))
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

	// 个人信息
	Data *struct {
		Nickname string `json:"nickname"`
		UserId   int64  `json:"user_id"`
	} `json:"data"`
	Retcode int64 `json:"retcode"`
	//Status string `json:"status"`
}

type PlayerInfo struct {
	UserId int64 `yaml:"userId"`;
	Name string;
	ValueNumMap map[string]int64 `yaml:"valueNumMap"`;
	ValueStrMap map[string]string `yaml:"valueStrMap"`;
	RpToday int `yaml:"rpToday"`;
	RpTime string `yaml:"rpTime"`;
	lastUpdateTime int64 `yaml:"lastUpdateTime"`;

	// level int 权限
	DiceSideNum int `yaml:"diceSideNum"` // 面数，为0时等同于d100
}

type ServiceAtItem struct {
	Active bool `json:"active" yaml:"active"` // 需要能记住配置，故有此选项
	ActivatedExtList []*ExtInfo `yaml:"activatedExtList"` // 当前群开启的扩展列表
	Players map[int64]*PlayerInfo // 群员信息
}

type IMSession struct {
	Socket   *gowebsocket.Socket `yaml:"-"`
	Nickname string `yaml:"-"`
	UserId   int64 `yaml:"-"`
	parent   *Dice `yaml:"-"`

	ServiceAt map[int64]*ServiceAtItem `json:"serviceAt" yaml:"serviceAt"`
	//GroupId int64 `json:"group_id"`
}

func (s *IMSession) serve() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	session := s;
	socket := gowebsocket.New("ws://127.0.0.1:6700")
	session.Socket = &socket

	socket.OnConnected = func(socket gowebsocket.Socket) {
		log.Println("Connected to server")
		//  {"data":{"nickname":"闃斧鐗岃�佽檸鏈�","user_id":1001},"retcode":0,"status":"ok"}
		getLoginInfo(socket)
	}

	socket.OnConnectError = func(err error, socket gowebsocket.Socket) {
		log.Println("Recieved connect error ", err)
	}

	socket.OnTextMessage = func(message string, socket gowebsocket.Socket) {
		msg := new(Message)
		log.Println("Recieved message " + message)
		err := json.Unmarshal([]byte(message), msg)
		if err == nil {
			if msg.Data != nil && msg.Retcode == 0 {
				session.UserId = msg.Data.UserId
				session.Nickname = msg.Data.Nickname
				log.Println("User info received.")
				return
			}

			if msg.MessageType == "group" || msg.MessageType == "private" {
				msgInfo := CommandParse(msg.Message)
				//log.Println(msgInfo)

				//if msg.Sender.UserId == 303451945 || msg.Sender.UserId == 184023393 {
				if msg.MessageType == "group" {
					if msgInfo != nil {
						session.commandSolve(session, msg, msgInfo);
						//c, _ := json.Marshal(msgInfo)
						//text := fmt.Sprintf("指令测试，来自群%d - %s(%d)：参数 %s", msg.GroupId, msg.Sender.Nickname, msg.Sender.UserId, c);
						//replyGroup(Socket, 438115120, text)
					} else {
						//text := fmt.Sprintf("信息 来自群%d - %s(%d)：%s", msg.GroupId, msg.Sender.Nickname, msg.Sender.UserId, msg.Message);
						//replyGroup(Socket, 438115120, text)
					}
				}
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

	cmdArgs.AmIBeMentioned = false;
	for _, i := range cmdArgs.At {
		if i.UserId == session.UserId {
			cmdArgs.AmIBeMentioned = true;
			break;
		}
	}

	tryItemSolve := func (item *CmdItemInfo) bool {
		if item != nil {
			ret := item.solve(session, msg, cmdArgs);
			if ret.success {
				return true;
			}
		}
		return false;
	}

	item := session.parent.cmdMap[cmdArgs.Command];
	if tryItemSolve(item) {
		return
	};

	sa := session.ServiceAt[msg.GroupId];
	if sa != nil && sa.Active {
		for _, i := range sa.ActivatedExtList {
			item := i.cmdMap[cmdArgs.Command];
			if tryItemSolve(item) {
				return
			};
		}
	}
}
