package dice

import (
	"fmt"
	"github.com/lonelyevil/kook"
	"github.com/lonelyevil/kook/log_adapter/plog"
	"github.com/phuslu/log"
	"strings"
	"time"
)

type PlatformAdapterKook struct {
	Session       *IMSession    `yaml:"-" json:"-"`
	Token         string        `yaml:"token" json:"token"`
	EndPoint      *EndPointInfo `yaml:"-" json:"-"`
	IntentSession *kook.Session `yaml:"-" json:"-"`
}

func (pa *PlatformAdapterKook) GetGroupInfoAsync(groupId string) {
	logger := pa.Session.Parent.Logger
	dm := pa.Session.Parent.Parent
	go pa.updateChannelNum()
	channel, err := pa.IntentSession.ChannelView(ExtractKookChannelId(groupId))
	if err != nil {
		logger.Errorf("获取Kook频道信息#{%s}时出错:{%s}", groupId, err.Error())
		return
	}
	dm.GroupNameCache.Set(groupId, &GroupNameCacheItem{
		channel.Name,
		time.Now().Unix(),
	})
	group := pa.Session.ServiceAtNew[groupId]
	if group != nil {
		group.GroupName = channel.Name
	}
}

func (pa *PlatformAdapterKook) updateChannelNum() {
	page := new(kook.PageSetting)
	guilds, _, _ := pa.IntentSession.GuildList(page)
	GroupNum := 0
	//guilds是bot加入的服务器list，channels是每个服务器里的频道（有权限访问的）
	for _, guild := range guilds {
		GroupNum += len(guild.Channels)
	}
	pa.EndPoint.GroupNum = int64(GroupNum)
}

func (pa *PlatformAdapterKook) updateGameStatus() {
	logger := pa.Session.Parent.Logger
	//gameupdate := new(kook.GameUpdate)
	//gameupdate.ID = int64(768222)
	//gameupdate.Icon = "https://img.kookapp.cn/assets/2022-12/DfYli1buyO0e80c0.png"
	//gameupdate.Name = "SealDice"
	//_, _ = pa.IntentSession.GameUpdate(gameupdate)
	err := pa.IntentSession.GameActivity(int64(768222))
	if err != nil {
		logger.Errorf("更新游戏状态时出错{%s}", err.Error())
		return
	}
}

func (pa *PlatformAdapterKook) Serve() int {
	//TODO: 写个子类继承他这个logger，太吵了
	l := log.Logger{
		Level:  log.TraceLevel,
		Writer: &log.ConsoleWriter{},
	}
	s := kook.New(pa.Token, plog.NewLogger(&l))
	s.AddHandler(func(ctx *kook.KmarkdownMessageContext) {
		if ctx.Common.Type != kook.MessageTypeKMarkdown || ctx.Extra.Author.Bot {
			return
		}
		pa.Session.Execute(pa.EndPoint, pa.toStdMessage(ctx), false)
	})
	err := s.Open()
	if err != nil {
		pa.Session.Parent.Logger.Errorf("与KOOK服务建立连接时出错")
		return 1
	}
	pa.Session.Parent.Logger.Infof("KOOK 连接成功")
	pa.IntentSession = s
	go pa.updateGameStatus()
	pa.EndPoint.State = 1
	pa.EndPoint.Enable = true
	self, _ := s.UserMe()
	pa.EndPoint.Nickname = self.Nickname
	pa.EndPoint.UserId = FormatDiceIdKook(self.ID)
	return 0
}

func (pa *PlatformAdapterKook) DoRelogin() bool {
	return false
}

func (pa *PlatformAdapterKook) SetEnable(enable bool) {
	if enable {
		pa.Session.Parent.Logger.Infof("正在启用KOOK服务……")
		if pa.IntentSession == nil {
			pa.Serve()
			return
		}
		err := pa.IntentSession.Open()
		if err != nil {
			pa.Session.Parent.Logger.Error("与KOOK服务进行连接时出错:", err)
			pa.EndPoint.State = 0
			pa.EndPoint.Enable = false
			return
		}
		pa.EndPoint.State = 1
		pa.EndPoint.Enable = true
	} else {
		pa.EndPoint.State = 0
		pa.EndPoint.Enable = false
		_ = pa.IntentSession.Close()
	}
}

func (pa *PlatformAdapterKook) SendToPerson(ctx *MsgContext, userId string, text string, flag string) {
	channel, err := pa.IntentSession.UserChatCreate(ExtractKookUserId(userId))
	if err != nil {
		return
	}
	dmc := &kook.DirectMessageCreate{
		ChatCode: channel.Code,
		MessageCreateBase: kook.MessageCreateBase{
			Content: text,
		},
	}
	_, err = pa.IntentSession.DirectMessageCreate(dmc)
	for _, i := range ctx.Dice.ExtList {
		if i.OnMessageSend != nil {
			i.OnMessageSend(ctx, "private", userId, text, flag)
		}
	}
}

func (pa *PlatformAdapterKook) SendToGroup(ctx *MsgContext, groupId string, text string, flag string) {
	_, err := pa.IntentSession.MessageCreate(&kook.MessageCreate{
		MessageCreateBase: kook.MessageCreateBase{
			TargetID: ExtractKookChannelId(groupId),
			Content:  text,
			Type:     kook.MessageTypeText,
		},
	})
	if err != nil {
		return
	}
	if ctx.Session.ServiceAtNew[groupId] != nil {
		for _, i := range ctx.Session.ServiceAtNew[groupId].ActivatedExtList {
			if i.OnMessageSend != nil {
				i.OnMessageSend(ctx, "group", groupId, text, flag)
			}
		}
	}
}

func FormatDiceIdKook(diceKook string) string {
	return fmt.Sprintf("KOOK:%s", diceKook)
}

func FormatDiceIdKookChannel(diceDiscord string) string {
	return fmt.Sprintf("KOOK-CH-Group:%s", diceDiscord)
}

func ExtractKookUserId(id string) string {
	if strings.HasPrefix(id, "KOOK:") {
		return id[len("KOOK:"):]
	}
	return id
}

func ExtractKookChannelId(id string) string {
	if strings.HasPrefix(id, "KOOK-CH-Group:") {
		return id[len("KOOK-CH-Group:"):]
	}
	return id
}

func (pa *PlatformAdapterKook) QuitGroup(ctx *MsgContext, id string) {}

func (pa *PlatformAdapterKook) SetGroupCardName(groupId string, userId string, name string) {}

func (pa *PlatformAdapterKook) toStdMessage(ctx *kook.KmarkdownMessageContext) *Message {
	msg := new(Message)
	msg.Time = ctx.Common.MsgTimestamp
	msg.RawId = ctx.Common.MsgID
	msg.Message = ctx.Common.Content
	msg.GroupId = FormatDiceIdKookChannel(ctx.Common.TargetID)
	msg.Platform = "KOOK"
	if ctx.Common.ChannelType == "PERSON" {
		msg.MessageType = "private"
	} else {
		msg.MessageType = "group"
	}
	send := new(SenderBase)
	send.UserId = FormatDiceIdKook(ctx.Common.AuthorID)
	send.Nickname = ctx.Extra.Author.Nickname
	send.GroupRole = ctx.Common.ChannelType
	msg.Sender = *send
	return msg
}
