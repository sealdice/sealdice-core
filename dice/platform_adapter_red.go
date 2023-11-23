package dice

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fy0/lockfree"
	"github.com/gorilla/websocket"
	"github.com/samber/lo"
)

type PlatformAdapterRed struct {
	Session     *IMSession    `yaml:"-" json:"-"`
	EndPoint    *EndPointInfo `yaml:"-" json:"-"`
	DiceServing bool          `yaml:"-"` // 是否正在连接中

	Host       string `yaml:"host" json:"host"`
	Port       int    `yaml:"port" json:"port"`
	Token      string `yaml:"token" json:"token"`
	RedVersion string `yaml:"-" json:"redVersion"`

	wsUrl   *url.URL
	httpUrl *url.URL

	conn      *websocket.Conn
	memberMap map[string]map[string]*GroupMember
}

type RedPack[T interface{}] struct {
	Type    string `json:"type"`
	Payload *T     `json:"payload"`
}

type RedConnectReq struct {
	Token string `json:"token"`
}

type RedConnectResp struct {
	Version  string `json:"version"`
	Name     string `json:"name"`
	AuthData *struct {
		Account     string `json:"account"`
		MainAccount string `json:"mainAccount"`
		Uin         string `json:"uin"`
		Uid         string `json:"uid"`
		NickName    string `json:"nickName"`
		Gender      int    `json:"gender"`
		Age         int    `json:"age"`
		FaceUrl     string `json:"faceUrl"`
		A2          string `json:"a2"`
		D2          string `json:"d2"`
		D2key       string `json:"d2Key"`
	} `json:"authData"`
}

type RedMessageRecv []*RedMessage

type RedChatType int

const (
	PersonChat RedChatType = 1
	GroupChat  RedChatType = 2
)

type RedPeer struct {
	ChatType RedChatType `json:"chatType"` // Group: 2
	PeerUin  string      `json:"peerUin"`
	GuildId  string      `json:"guildId,omitempty"` // 一直为 Null
}

type RedMessageSend struct {
	Peer     *RedPeer      `json:"peer"`
	Elements []*RedElement `json:"elements"`
}

type RedMessage struct {
	MsgID               string        `json:"msgId"`
	MsgRandom           string        `json:"msgRandom"`
	MsgSeq              string        `json:"msgSeq"`
	CntSeq              string        `json:"cntSeq"`
	ChatType            int           `json:"chatType"`
	MsgType             int           `json:"msgType"`
	SubMsgType          int           `json:"subMsgType"`
	SendType            int           `json:"sendType"`
	SenderUid           string        `json:"senderUid"`
	PeerUid             string        `json:"peerUid"`
	ChannelId           string        `json:"channelId"`
	GuildId             string        `json:"guildId"`
	GuildCode           string        `json:"guildCode"`
	FromUid             string        `json:"fromUid"`
	FromAppid           string        `json:"fromAppid"`
	MsgTime             string        `json:"msgTime"`
	MsgMeta             string        `json:"msgMeta"`
	SendStatus          int           `json:"sendStatus"`
	SendMemberName      string        `json:"sendMemberName"`
	SendNickName        string        `json:"sendNickName"`
	GuildName           string        `json:"guildName"`
	ChannelName         string        `json:"channelName"`
	Elements            []*RedElement `json:"elements"`
	Records             []interface{} `json:"records"`
	EmojiLikesList      []interface{} `json:"emojiLikesList"`
	CommentCnt          string        `json:"commentCnt"`
	DirectMsgFlag       int           `json:"directMsgFlag"`
	DirectMsgMembers    []interface{} `json:"directMsgMembers"`
	PeerName            string        `json:"peerName"`
	Editable            bool          `json:"editable"`
	AvatarMeta          string        `json:"avatarMeta"`
	AvatarPendant       string        `json:"avatarPendant"`
	FeedId              string        `json:"feedId"`
	RoleId              string        `json:"roleId"`
	TimeStamp           string        `json:"timeStamp"`
	IsImportMsg         bool          `json:"isImportMsg"`
	AtType              int           `json:"atType"`
	RoleType            int           `json:"roleType"`
	FromChannelRoleInfo *RedRoleInfo  `json:"fromChannelRoleInfo"`
	FromGuildRoleInfo   *RedRoleInfo  `json:"fromGuildRoleInfo"`
	LevelRoleInfo       *RedRoleInfo  `json:"levelRoleInfo"`
	RecallTime          string        `json:"recallTime"`
	IsOnlineMsg         bool          `json:"isOnlineMsg"`
	GeneralFlags        string        `json:"generalFlags"`
	ClientSeq           string        `json:"clientSeq"`
	NameType            int           `json:"nameType"`
	AvatarFlag          int           `json:"avatarFlag"`
	SenderUin           string        `json:"senderUin"`
	PeerUin             string        `json:"peerUin"`
}

type RedElement struct {
	ElementType     int             `json:"elementType,omitempty"`
	ElementId       string          `json:"elementId,omitempty"`
	ExtBufForUI     string          `json:"extBufForUI,omitempty"`
	PicElement      *RedPicElement  `json:"picElement,omitempty"`
	TextElement     *RedTextElement `json:"textElement,omitempty"`
	ArkElement      interface{}     `json:"arkElement,omitempty"`
	AvRecordElement interface{}     `json:"avRecordElement,omitempty"`
	CalendarElement interface{}     `json:"calendarElement,omitempty"`
	FaceElement     interface{}     `json:"faceElement,omitempty"`
	FileElement     *RedFileElement `json:"fileElement,omitempty"`
	GiphyElement    interface{}     `json:"giphyElement,omitempty"`
	GrayTipElement  *struct {
		XmlElement          *RedXMLElement `json:"xmlElement,omitempty"`
		AioOpGrayTipElement interface{}    `json:"aioOpGrayTipElement,omitempty"`
		BlockGrayTipElement interface{}    `json:"blockGrayTipElement,omitempty"`
		BuddyElement        interface{}    `json:"buddyElement,omitempty"`
		BuddyNotifyElement  interface{}    `json:"buddyNotifyElement,omitempty"`
		EmojiReplyElement   interface{}    `json:"emojiReplyElement,omitempty"`
		EssenceElement      interface{}    `json:"essenceElement,omitempty"`
		FeedMsgElement      interface{}    `json:"feedMsgElement,omitempty"`
		FileReceiptElement  interface{}    `json:"fileReceiptElement,omitempty"`
		GroupElement        interface{}    `json:"groupElement,omitempty"`
		GroupNotifyElement  interface{}    `json:"groupNotifyElement,omitempty"`
		JsonGrayTipElement  interface{}    `json:"jsonGrayTipElement,omitempty"`
		LocalGrayTipElement interface{}    `json:"localGrayTipElement,omitempty"`
		ProclamationElement interface{}    `json:"proclamationElement,omitempty"`
		RevokeElement       interface{}    `json:"revokeElement,omitempty"`
		SubElementType      interface{}    `json:"subElementType,omitempty"`
	} `json:"grayTipElement,omitempty"`
	InlineKeyboardElement  interface{} `json:"inlineKeyboardElement,omitempty"`
	LiveGiftElement        interface{} `json:"liveGiftElement,omitempty"`
	MarkdownElement        interface{} `json:"markdownElement,omitempty"`
	MarketFaceElement      interface{} `json:"marketFaceElement,omitempty"`
	MultiForwardMsgElement interface{} `json:"multiForwardMsgElement,omitempty"`
	PttElement             interface{} `json:"pttElement,omitempty"`
	ReplyElement           interface{} `json:"replyElement,omitempty"`
	StructLongMsgElement   interface{} `json:"structLongMsgElement,omitempty"`
	TextGiftElement        interface{} `json:"textGiftElement,omitempty"`
	VideoElement           interface{} `json:"videoElement,omitempty"`
	WalletElement          interface{} `json:"walletElement,omitempty"`
	YoloGameResultElement  interface{} `json:"yoloGameResultElement,omitempty"`
}

type RedXMLElement struct {
	BusiType    string `json:"busiType,omitempty"`
	BusiId      string `json:"busiId,omitempty"`
	C2cType     int    `json:"c2CType,omitempty"`
	ServiceType int    `json:"serviceType,omitempty"`
	CtrlFlag    int    `json:"ctrlFlag,omitempty"`
	Content     string `json:"content,omitempty"`
	TemplId     string `json:"templId,omitempty"`
	SeqId       string `json:"seqId,omitempty"`
	TemplParam  any    `json:"templParam,omitempty"`
	PbReserv    string `json:"pbReserv,omitempty"`
	Members     any    `json:"members,omitempty"`
}

type RedPicElement struct {
	PicSubType     int            `json:"picSubType,omitempty"`
	FileName       string         `json:"fileName,omitempty"`
	FileSize       string         `json:"fileSize,omitempty"`
	PicWidth       int64          `json:"picWidth,omitempty"`
	PicHeight      int64          `json:"picHeight,omitempty"`
	Original       bool           `json:"original,omitempty"`
	Md5HexStr      string         `json:"md5HexStr,omitempty"`
	SourcePath     string         `json:"sourcePath,omitempty"`
	ThumbPath      *RedThumbPath  `json:"thumbPath,omitempty"`
	TransferStatus int            `json:"transferStatus,omitempty"`
	Progress       int            `json:"progress,omitempty"`
	PicType        int            `json:"picType,omitempty"`
	InvalidState   int            `json:"invalidState,omitempty"`
	FileUuid       string         `json:"fileUuid,omitempty"`
	FileSubId      string         `json:"fileSubId,omitempty"`
	ThumbFileSize  int            `json:"thumbFileSize,omitempty"`
	Summary        string         `json:"summary,omitempty"`
	EmojiAd        *RedEmojiAd    `json:"emojiAd,omitempty"`
	EmojiMall      *RedEmojiMall  `json:"emojiMall,omitempty"`
	EmojiZplan     *RedEmojiZplan `json:"emojiZplan,omitempty"`
	OriginImageUrl string         `json:"originImageUrl,omitempty"`
}

type RedEmojiAd struct {
	Url  string `json:"url,omitempty"`
	Desc string `json:"desc,omitempty"`
}

type RedEmojiMall struct {
	PackageId int `json:"packageId,omitempty"`
	EmojiId   int `json:"emojiId,omitempty"`
}

type RedEmojiZplan struct {
	ActionId         int    `json:"actionId,omitempty"`
	ActionName       string `json:"actionName,omitempty"`
	ActionType       int    `json:"actionType,omitempty"`
	PlayerNumber     int    `json:"playerNumber,omitempty"`
	PeerUid          string `json:"peerUid,omitempty"`
	BytesReserveInfo string `json:"bytesReserveInfo,omitempty"`
}

type RedThumbPath struct {
}

type RedTextElement struct {
	Content        string `json:"content,omitempty"`
	AtType         int    `json:"atType,omitempty"`
	AtUid          string `json:"atUid,omitempty"`
	AtTinyId       string `json:"atTinyId,omitempty"`
	AtNtUid        string `json:"atNtUid,omitempty"`
	SubElementType int    `json:"subElementType,omitempty"`
	AtChannelId    string `json:"atChannelId,omitempty"`
	AtRoleId       string `json:"atRoleId,omitempty"`
	AtRoleColor    int    `json:"atRoleColor,omitempty"`
	AtRoleName     string `json:"atRoleName,omitempty"`
	NeedNotify     int    `json:"needNotify,omitempty"`
	AtNtUin        string `json:"atNtUin,omitempty"`
}

type RedFileElement struct {
	FileMd5       string      `json:"fileMd5,omitempty"`
	FileSize      int64       `json:"fileSize,omitempty"`
	FileName      string      `json:"fileName,omitempty"`
	FilePath      string      `json:"filePath,omitempty"`
	PicWidth      int64       `json:"picWidth,omitempty"`
	PicHeight     int64       `json:"picHeight,omitempty"`
	PicThumbPath  interface{} `json:"thumbPath,omitempty"`
	File10MMd5    string      `json:"file10MMd5,omitempty"`
	FileSha       string      `json:"fileSha,omitempty"`
	FileSha3      string      `json:"fileSha3,omitempty"`
	FileUuid      string      `json:"fileUuid,omitempty"`
	FileSubId     string      `json:"fileSubId,omitempty"`
	ThumbFileSize int64       `json:"thumbFileSize,omitempty"`
}

type RedRoleInfo struct {
	RoleId string `json:"roleId"`
	Name   string `json:"name"`
	Color  int    `json:"color"`
}

type Friend struct {
	Qid               string `json:"qid"`
	Uin               string `json:"uin"` // QQ 号
	Nick              string `json:"nick"`
	Remark            string `json:"remark"`
	LongNick          string `json:"longNick"`
	AvatarUrl         string `json:"avatarUrl"`
	Birthday_year     int    `json:"birthday_Year"`
	Birthday_month    int    `json:"birthday_Month"`
	Birthday_day      int    `json:"birthday_Day"`
	Sex               int    `json:"sex"` // 性别
	TopTime           string `json:"topTime"`
	IsBlock           bool   `json:"isBlock"` // 是否拉黑
	IsMsgDisturb      bool   `json:"isMsgDisturb"`
	IsSpecialCareOpen bool   `json:"isSpecialCareOpen"`
	IsSpecialCareZone bool   `json:"isSpecialCareZone"`
	RingId            string `json:"ringId"`
	Status            int    `json:"status"`
	ExtStatus         int    `json:"extStatus"`
	CategoryId        int    `json:"categoryId"`
	OnlyChat          bool   `json:"onlyChat"`
	QzoneNotWatch     bool   `json:"qzoneNotWatch"`
	QzoneNotWatched   bool   `json:"qzoneNotWatched"`
	VipFlag           bool   `json:"vipFlag"`
	YearVipFlag       bool   `json:"yearVipFlag"`
	SvipFlag          bool   `json:"svipFlag"`
	VipLevel          int    `json:"vipLevel"`
	Category          string `json:"category"` // 分组信息
}

type Group struct {
	GroupCode               string `json:"groupCode"`   // 群号
	MaxMember               int    `json:"maxMember"`   // 最大人数
	MemberCount             int    `json:"memberCount"` // 成员人数
	GroupName               string `json:"groupName"`   // 群名
	GroupStatus             int    `json:"groupStatus"`
	MemberRole              int    `json:"memberRole"` // 群成员角色
	IsTop                   bool   `json:"isTop"`
	ToppedTimestamp         string `json:"toppedTimestamp"`
	PrivilegeFlag           int    `json:"privilegeFlag"` // 群权限
	IsConf                  bool   `json:"isConf"`
	HasModifyConfGroupFace  bool   `json:"hasModifyConfGroupFace"`
	HasModifyConfGroupName  bool   `json:"hasModifyConfGroupName"`
	RemarkName              string `json:"remarkName"`
	HasMemo                 bool   `json:"hasMemo"`
	GroupShutupExpireTime   string `json:"groupShutupExpireTime"`
	PersonShutupExpireTime  string `json:"personShutupExpireTime"`
	DiscussToGroupUin       string `json:"discussToGroupUin"`
	DiscussToGroupMaxMsgSeq int    `json:"discussToGroupMaxMsgSeq"`
	DiscussToGroupTime      int    `json:"discussToGroupTime"`
}

type GroupMember struct {
	Uid        string `json:"uid"`
	Qid        string `json:"qid"`
	Uin        string `json:"uin"`
	Nick       string `json:"nick"`
	Remark     string `json:"remark"`
	CardType   int    `json:"cardType"`
	CardName   string `json:"cardName"`
	Role       int    `json:"role"` // 2-普通成员 3-管理 4-群主
	AvatarPath string `json:"avatarPath"`
	ShutUpTime int64  `json:"shutUpTime"`
	IsDelete   bool   `json:"isDelete"`
}

func (pa *PlatformAdapterRed) Serve() int {
	ep := pa.EndPoint
	s := pa.Session
	log := s.Parent.Logger
	dm := pa.Session.Parent.Parent

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	wsUrl := url.URL{
		Scheme: "ws",
		Host:   fmt.Sprintf("%s:%d", pa.Host, pa.Port),
	}
	httpUrl := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", pa.Host, pa.Port),
		Path:   "/api",
	}
	log.Infof("connecting to %s", wsUrl.String())
	conn, resp, err := websocket.DefaultDialer.Dial(wsUrl.String(), nil)
	if err != nil {
		log.Error("dial:", err)
		pa.EndPoint.State = 3
		return 1
	}
	defer resp.Body.Close()
	defer func(conn *websocket.Conn) {
		_ = conn.Close()
	}(conn)
	pa.conn = conn
	pa.EndPoint.State = 2

	// 鉴权
	auth := &RedPack[RedConnectReq]{
		Type:    "meta::connect",
		Payload: &RedConnectReq{pa.Token},
	}
	authData, _ := json.Marshal(auth)
	err = conn.WriteMessage(websocket.TextMessage, authData)
	if err != nil {
		log.Error("auth failed:", err)
		pa.EndPoint.State = 3
		return 1
	}
	_, authRespData, err := conn.ReadMessage()
	if err != nil {
		log.Error("auth failed:", err)
		pa.EndPoint.State = 3
		return 1
	}
	var authResp RedPack[RedConnectResp]
	err = json.Unmarshal(authRespData, &authResp)
	if err != nil {
		log.Error("auth failed:", err)
		pa.EndPoint.State = 3
		return 1
	}
	log.Debugf("red auth resp:%+v", authResp)

	pa.wsUrl = &wsUrl
	pa.httpUrl = &httpUrl
	pa.RedVersion = authResp.Payload.Version
	pa.EndPoint.State = 1

	// 获得用户信息
	botInfo := pa.getBotInfo()
	ep.Nickname = botInfo.Name
	ep.UserID = botInfo.SelfId
	d := pa.Session.Parent
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	pa.Session.Parent.Logger.Infof("red 连接成功，账号<%s>(%s)", pa.EndPoint.Nickname, pa.EndPoint.UserID)

	// 获得好友列表
	refreshFriends := func() {
		friends := pa.getFriends()
		for _, friend := range friends {
			dm.UserNameCache.Set(friend.Uin, &GroupNameCacheItem{
				Name: friend.Nick,
				time: time.Now().Unix(),
			})
		}
	}
	go refreshFriends()
	// 获得群列表
	pa.GetGroupInfoAsync("")

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			msgType, msgData, err := conn.ReadMessage()
			if err != nil {
				interrupt <- os.Interrupt
				return
			}
			switch msgType {
			case websocket.TextMessage:
				log.Debugf("message=%s", msgData)

				var msgRowMap map[string]interface{}
				err := json.Unmarshal(msgData, &msgRowMap)
				if err != nil {
					log.Errorf("recv parse error: %s, rowData: %s", err, msgData)
				}
				if msgType, ok := msgRowMap["type"]; ok && msgType != "message::send::reply" {
					var msgRow RedPack[RedMessageRecv]
					_ = json.Unmarshal(msgData, &msgRow)
					log.Debug("recv: %+v", msgRow)
					for _, msg := range *msgRow.Payload {
						pa.Session.Execute(pa.EndPoint, pa.decodeMessage(msg), false)
					}
				}
			case websocket.BinaryMessage:
			case websocket.CloseMessage:
				log.Debug("server close")
				pa.conn = nil
				done <- struct{}{}
			case websocket.PingMessage:
			case websocket.PongMessage:
			}
		}
	}()

	for {
		select {
		case <-done:
		case <-interrupt:
			log.Debug("red interrupt")

			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			_ = pa.conn.Close()

			select {
			case <-done:
			case <-time.After(time.Second):
			}
		}
	}
}

func (pa *PlatformAdapterRed) DoRelogin() bool {
	pa.Session.Parent.Logger.Infof("正在启用 red 连接……")
	pa.EndPoint.State = 0
	pa.EndPoint.Enable = false
	if pa.conn != nil {
		_ = pa.conn.Close()
	}
	pa.conn = nil
	return pa.Serve() == 0
}

func (pa *PlatformAdapterRed) SetEnable(enable bool) {
	d := pa.Session.Parent
	e := pa.EndPoint
	if enable {
		e.Enable = true
		pa.DiceServing = false

		if pa.conn == nil {
			go ServeQQ(d, e)
		}
	} else {
		e.State = 0
		e.Enable = false
		if pa.conn != nil {
			_ = pa.conn.Close()
			pa.conn = nil
		}
	}
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
}

func (pa *PlatformAdapterRed) QuitGroup(_ *MsgContext, id string) {
	log := pa.Session.Parent.Logger
	log.Warnf("red: 尝试退出群组(%s)，但尚不支持该功能", id)
}

func (pa *PlatformAdapterRed) SendToPerson(ctx *MsgContext, uid string, text string, flag string) {
	rowId, chatType := pa.mustExtractId(uid)
	if chatType != PersonChat {
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
				},
					flag)
			})
		}
	}

	texts := textSplit(text)
	for _, subText := range texts {
		doSleepQQ(ctx)
		pa.sendRow(&RedMessageSend{
			Peer: &RedPeer{
				ChatType: PersonChat,
				PeerUin:  strconv.FormatInt(rowId, 10),
			},
			Elements: pa.encodeMessage(ctx, subText),
		})
	}
}

func (pa *PlatformAdapterRed) SendToGroup(ctx *MsgContext, groupId string, text string, flag string) {
	rowId, chatType := pa.mustExtractId(groupId)
	if chatType != GroupChat {
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
						GroupID:     groupId,
						Sender: SenderBase{
							Nickname: pa.EndPoint.Nickname,
							UserID:   pa.EndPoint.UserID,
						},
					}, flag)
				})
			}
		}
	}

	texts := textSplit(text)
	for _, subText := range texts {
		doSleepQQ(ctx)
		pa.sendRow(&RedMessageSend{
			Peer: &RedPeer{
				ChatType: GroupChat,
				PeerUin:  strconv.FormatInt(rowId, 10),
			},
			Elements: pa.encodeMessage(ctx, subText),
		})
	}
}

func (pa *PlatformAdapterRed) sendRow(redMsg *RedMessageSend) {
	if pa.conn != nil {
		log := pa.Session.Parent.Logger
		conn := pa.conn

		param := &RedPack[RedMessageSend]{
			Type:    "message::send",
			Payload: redMsg,
		}
		data, _ := json.Marshal(param)
		err := conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Errorf("send msg failed: %s", err.Error())
		}
	}
}

func (pa *PlatformAdapterRed) mustExtractId(id string) (int64, RedChatType) {
	if strings.HasPrefix(id, "QQ:") {
		num, _ := strconv.ParseInt(id[len("QQ:"):], 10, 64)
		return num, PersonChat
	}
	if strings.HasPrefix(id, "QQ-Group:") {
		num, _ := strconv.ParseInt(id[len("QQ-Group:"):], 10, 64)
		return num, GroupChat
	}
	return 0, 0
}

func (pa *PlatformAdapterRed) SetGroupCardName(ctx *MsgContext, name string) {
	log := pa.Session.Parent.Logger
	log.Warn("red: 尚未实现该功能")
}

func (pa *PlatformAdapterRed) SendFileToPerson(ctx *MsgContext, uid string, path string, flag string) {
	pa.SendToPerson(ctx, uid, fmt.Sprintf("[CQ:file,file=%s]", path), flag)
}

func (pa *PlatformAdapterRed) SendFileToGroup(ctx *MsgContext, uid string, path string, flag string) {
	pa.SendToGroup(ctx, uid, fmt.Sprintf("[CQ:file,file=%s]", path), flag)
}

func (pa *PlatformAdapterRed) MemberBan(_ string, _ string, _ int64) {}

func (pa *PlatformAdapterRed) MemberKick(_ string, _ string) {}

func (pa *PlatformAdapterRed) GetGroupInfoAsync(_ string) {
	// 触发更新群信息
	d := pa.Session.Parent
	dm := d.Parent
	ep := pa.EndPoint
	s := pa.Session
	session := s
	if pa.memberMap == nil {
		pa.memberMap = make(map[string]map[string]*GroupMember)
	}

	refreshMembers := func(group *Group) {
		groupID := formatDiceIDRedGroup(group.GroupCode)
		members := pa.getMemberList(group.GroupCode, group.MemberCount)
		groupInfo := session.ServiceAtNew[groupID]
		groupMemberMap := make(map[string]*GroupMember)
		for _, member := range members {
			userID := formatDiceIDRed(member.Uin)
			groupMemberMap[userID] = member
			if groupInfo != nil {
				p := groupInfo.PlayerGet(d.DBData, userID)
				if p == nil {
					name := member.CardName
					if name == "" {
						name = member.Nick
					}
					p = &GroupPlayerInfo{
						Name:          name,
						UserID:        userID,
						ValueMapTemp:  lockfree.NewHashMap(),
						UpdatedAtTime: 0,
					}
					groupInfo.Players.Store(userID, p)
				}
			}
		}
		pa.memberMap[groupID] = groupMemberMap
	}

	refresh := func() {
		groups := pa.getGroups()
		for _, group := range groups {
			if group != nil {
				groupId := formatDiceIDRedGroup(group.GroupCode)
				dm.GroupNameCache.Set(groupId, &GroupNameCacheItem{
					Name: group.GroupName,
					time: time.Now().Unix(),
				})

				groupInfo := session.ServiceAtNew[groupId]
				if groupInfo == nil {
					// 新检测到群
					ctx := &MsgContext{
						Session:  session,
						EndPoint: ep,
						Dice:     session.Parent,
					}
					SetBotOnAtGroup(ctx, groupId)
				}

				// 触发群成员更新
				go refreshMembers(group)

				groupRecord := groupInfo
				if groupRecord != nil {
					if group.MemberCount == 0 {
						diceId := ep.UserID
						if _, exists := groupRecord.DiceIDExistsMap.Load(diceId); exists {
							// 不在群里了，更新信息
							groupRecord.DiceIDExistsMap.Delete(diceId)
							groupRecord.UpdatedAtTime = time.Now().Unix()
						}
					} else if groupRecord.GroupName != group.GroupName {
						// 更新群名
						groupRecord.GroupName = group.GroupName
						groupRecord.UpdatedAtTime = time.Now().Unix()
					}
				}
			}
		}
	}
	go refresh()
}

type BotInfo struct {
	SelfId string
	Name   string
}

func (pa *PlatformAdapterRed) getBotInfo() *BotInfo {
	data, _ := pa.httpDo("GET", "getSelfProfile", nil, nil)
	var body map[string]interface{}
	_ = json.Unmarshal(data, &body)

	return &BotInfo{
		SelfId: formatDiceIDRed(body["uin"].(string)),
		Name:   body["nick"].(string),
	}
}

func (pa *PlatformAdapterRed) getFriends() []*Friend {
	data, _ := pa.httpDo("GET", "bot/friends", nil, nil)
	var friends []*Friend
	_ = json.Unmarshal(data, &friends)
	return friends
}

func (pa *PlatformAdapterRed) getGroups() []*Group {
	data, _ := pa.httpDo("GET", "bot/groups", nil, nil)
	var groups []*Group
	_ = json.Unmarshal(data, &groups)
	return groups
}

func (pa *PlatformAdapterRed) getMemberList(group string, size int) []*GroupMember {
	if strings.HasPrefix(group, "QQ-Group:") {
		group = group[len("QQ-Group"):]
	}
	groupID, err := strconv.ParseInt(group, 10, 64)
	if err != nil {
		pa.Session.Parent.Logger.Error("red 获取群成员失败", err)
	}
	paramData, err := json.Marshal(map[string]interface{}{
		"group": groupID,
		"size":  size,
	})
	if err != nil {
		pa.Session.Parent.Logger.Error("red 获取群成员失败", err)
	}
	data, err := pa.httpDo("POST", "group/getMemberList", nil, bytes.NewBuffer(paramData))
	if err != nil {
		pa.Session.Parent.Logger.Error("red 获取群成员失败", err)
	}
	type memberInfo struct {
		Index  int          `json:"index"`
		Detail *GroupMember `json:"detail"`
	}
	var body []memberInfo
	_ = json.Unmarshal(data, &body)
	members := lo.Map(body, func(item memberInfo, _ int) *GroupMember {
		return item.Detail
	})
	return members
}

type RedRichMediaResp struct {
	Md5        string              `json:"md5"`
	FileSize   int64               `json:"fileSize"`
	FilePath   string              `json:"filePath"`
	NtFilePath string              `json:"ntFilePath"`
	ImageInfo  *RedImageUploadInfo `json:"imageInfo"`
}

type RedImageUploadInfo struct {
	Width  int64  `json:"width"`
	Height int64  `json:"height"`
	Type   string `json:"type"`
	Mime   string `json:"mime"`
	WUnits string `json:"wUnits"`
	HUnits string `json:"hUnits"`
}

func (pa *PlatformAdapterRed) getFile(msgID string, chatType RedChatType, peerUid string, elementID string) map[string]interface{} {
	paramData, _ := json.Marshal(map[string]interface{}{
		"msgId":        msgID,
		"chatType":     chatType,
		"peerUid":      peerUid,
		"elementId":    elementID,
		"thumbSize":    0,
		"downloadType": 2,
	})
	data, _ := pa.httpDo("POST", "message/fetchRichMedia", nil, bytes.NewBuffer(paramData))
	var body map[string]interface{}
	_ = json.Unmarshal(data, &body)
	return body
}

func (pa *PlatformAdapterRed) uploadFile(file string, reader io.Reader) *RedRichMediaResp {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", filepath.Base(file))
	_, _ = io.Copy(part, reader)
	_ = writer.Close()

	data, _ := pa.httpDo("POST", "upload", map[string]string{"Content-Type": writer.FormDataContentType()}, body)
	var resp RedRichMediaResp
	_ = json.Unmarshal(data, &resp)
	return &resp
}

func (pa *PlatformAdapterRed) httpDo(method, action string, headers map[string]string, body io.Reader) ([]byte, error) {
	client := http.Client{}
	request, _ := http.NewRequest(method, pa.httpUrl.String()+"/"+action, body)
	request.Header.Add("Authorization", "Bearer "+pa.Token)
	if len(headers) != 0 {
		for key, value := range headers {
			request.Header.Add(key, value)
		}
	}
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// encodeMessage 将带 cq code 的内容转换为 red 所需的格式
func (pa *PlatformAdapterRed) encodeMessage(ctx *MsgContext, content string) []*RedElement {
	dice := pa.Session.Parent
	elems := dice.ConvertStringMessage(content)
	var redElems []*RedElement
	for _, elem := range elems {
		switch e := elem.(type) {
		case *TextElement:
			redElems = append(redElems, &RedElement{
				ElementType: 1,
				TextElement: &RedTextElement{Content: e.Content},
			})
		case *AtElement:
			redElems = append(redElems, &RedElement{
				ElementType: 1,
				TextElement: &RedTextElement{
					AtType:  2,
					AtNtUin: e.Target,
					Content: fmt.Sprintf("@%s", e.Target),
				},
			})
		case *ImageElement:
			fi := e.file
			resp := pa.uploadFile(fi.File, fi.Stream)
			redElem := RedElement{
				ElementType: 2,
				PicElement: &RedPicElement{
					Original:   true,
					Md5HexStr:  resp.Md5,
					FileSize:   strconv.FormatInt(resp.FileSize, 10),
					FileName:   fi.File,
					SourcePath: resp.NtFilePath,
				},
			}
			if resp.ImageInfo != nil {
				redElem.PicElement.PicWidth = resp.ImageInfo.Width
				redElem.PicElement.PicHeight = resp.ImageInfo.Height
			}
			redElems = append(redElems, &redElem)
		case *FileElement:
			resp := pa.uploadFile(e.File, e.Stream)
			redElems = append(redElems, &RedElement{
				ElementType: 3,
				FileElement: &RedFileElement{
					FileMd5:       resp.Md5,
					FileSize:      resp.FileSize,
					FileName:      e.File,
					FilePath:      resp.NtFilePath,
					PicWidth:      0,
					PicHeight:     0,
					PicThumbPath:  nil,
					File10MMd5:    "",
					FileSha:       "",
					FileSha3:      "",
					FileUuid:      "",
					FileSubId:     "",
					ThumbFileSize: 750,
				},
			})
		}
	}
	return redElems
}

// decodeMessage 将 red 格式的信息解析成海豹所需格式
func (pa *PlatformAdapterRed) decodeMessage(message *RedMessage) *Message {
	log := pa.Session.Parent.Logger
	msg := new(Message)
	if t, err := strconv.ParseInt(message.MsgTime, 10, 64); err == nil {
		msg.Time = t
	} else {
		log.Errorf("red 消息 msgTime 解析错误，err=%s, row str=%s", err, message.MsgTime)
	}
	msg.RawID = message.MsgID
	var content string
	uid := message.SenderUin
	if uid == "" {
		uid = message.SenderUid
	}
	for _, element := range message.Elements {
		switch element.ElementType {
		case 1:
			// 文本
			switch element.TextElement.AtType {
			case 1:
				// at 全体
				content += "[CQ:at,qq=all]"
			case 2:
				// at 某人
				id := element.TextElement.AtNtUin
				if id == "" {
					id = element.TextElement.AtUid
				}
				content += fmt.Sprintf("[CQ:at,qq=%s]", id)
			default:
				content += element.TextElement.Content
			}
		case 2:
			// 图片
			u := element.PicElement.OriginImageUrl
			id := uid
			if id == "" {
				id = message.SenderUid
			}
			var gid string
			if message.ChatType == int(GroupChat) {
				gid = message.PeerUin
				if gid == "" {
					gid = message.PeerUin
				}
			}
			fUuid := element.PicElement.FileUuid

			if u == "" {
				if message.ChatType == int(GroupChat) {
					content += fmt.Sprintf(
						"[CQ:image,file=%s,subType=0,url=https://gchat.qpic.cn/gchatpic_new/%s/%s-%s-%s/0]",
						filepath.Base(element.PicElement.SourcePath),
						id,
						gid,
						fUuid,
						strings.ToUpper(element.PicElement.Md5HexStr),
					)
				} else {
					content += fmt.Sprintf(
						"[CQ:image,file=%s,url=https://c2cpicdw.qpic.cn/offpic_new/%s/%s/0]",
						filepath.Base(element.PicElement.SourcePath),
						id,
						fUuid,
					)
				}
			} else if strings.Contains(u, "&rkey") {
				// 下载图片
				resp := pa.getFile(message.MsgID, RedChatType(message.ChatType), message.PeerUin, element.ElementId)
				data := resp["data"].([]byte)
				dataBase64 := base64.StdEncoding.EncodeToString(data)
				content += fmt.Sprintf("[CQ:image,file=base64://%s,type=show,id=40000]", dataBase64)
			} else {
				content += fmt.Sprintf("[CQ:image,file=https://c2cpicdw.qpic.cn%s,type=show,id=40000]", u)
			}
		case 4:
		// TODO: 语音
		case 6:
			// 表情
			faceElement := element.FaceElement.(map[string]interface{})
			faceIndex := faceElement["faceIndex"].(float64)
			content += fmt.Sprintf("[CQ:face,id=%d]", int(faceIndex))
		case 7:
			// TODO: 引用
		}
	}
	msg.Message = content
	msg.Platform = "QQ"

	send := SenderBase{}
	send.UserID = formatDiceIDRed(uid)
	if message.ChatType == 1 {
		// 私聊消息
		msg.MessageType = "private"
		dm := pa.Session.Parent.Parent
		if nick, ok := dm.UserNameCache.Get(uid); ok {
			nameInfo := nick.(*GroupNameCacheItem)
			send.Nickname = nameInfo.Name
		}
		if send.Nickname == "" {
			send.Nickname = "<未知用户>"
		}
	} else {
		msg.MessageType = "group"
		msg.GroupID = formatDiceIDRedGroup(message.PeerUid)
		send.Nickname = message.SendNickName
		if send.Nickname == "" {
			send.Nickname = message.SendMemberName
		}
		groupMemberInfo := pa.memberMap[msg.GroupID]
		if groupMemberInfo != nil {
			memberInfo := groupMemberInfo[send.UserID]
			if memberInfo != nil {
				if memberInfo.Role == 4 {
					send.GroupRole = "owner"
				} else if memberInfo.Role == 3 {
					send.GroupRole = "admin"
				}
			}
		}
		if message.SendNickName != "" {
			send.Nickname = message.SendNickName
		} else if message.SendMemberName != "" {
			send.Nickname = message.SendMemberName
		}
	}
	msg.Sender = send

	return msg
}

func formatDiceIDRed(diceRed string) string {
	return fmt.Sprintf("QQ:%s", diceRed)
}

func formatDiceIDRedGroup(diceRed string) string {
	return fmt.Sprintf("QQ-Group:%s", diceRed)
}
