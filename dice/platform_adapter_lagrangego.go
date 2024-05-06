package dice

import (
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/LagrangeDev/LagrangeGo/client"
	"github.com/LagrangeDev/LagrangeGo/event"
	"github.com/LagrangeDev/LagrangeGo/info"
	lagMessage "github.com/LagrangeDev/LagrangeGo/message"
	"github.com/LagrangeDev/LagrangeGo/packets/oidb"
	"github.com/LagrangeDev/LagrangeGo/packets/wtlogin/qrcodeState"
	"github.com/LagrangeDev/LagrangeGo/utils"

	"sealdice-core/message"
)

var DefaultSignUrl = ``

func LoadSigInfo(filePath string) (*info.SigInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	var sigInfo info.SigInfo
	decoder := gob.NewDecoder(file)
	if err = decoder.Decode(&sigInfo); err != nil {
		return nil, err
	}
	// fmt.Printf("Loaded SigInfo: %+v\n", sigInfo)
	return &sigInfo, nil
}

func SaveSigInfo(filePath string, sigInfo *info.SigInfo) error {
	// fmt.Printf("Saving SigInfo: %+v\n", sigInfo)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	encoder := gob.NewEncoder(file)
	if err = encoder.Encode(sigInfo); err != nil {
		return err
	}

	return nil
}

func LoadDevice(path string) *info.DeviceInfo {
	data, err := os.ReadFile(path)
	if err != nil {
		deviceinfo := info.NewDeviceInfo(int(utils.RandU32()))
		_ = SaveDevice(deviceinfo, path)
		return deviceinfo
	}
	var dinfo info.DeviceInfo
	err = json.Unmarshal(data, &dinfo)
	if err != nil {
		deviceinfo := info.NewDeviceInfo(int(utils.RandU32()))
		_ = SaveDevice(deviceinfo, path)
		return deviceinfo
	}
	return &dinfo
}

func SaveDevice(deviceInfo *info.DeviceInfo, path string) error {
	data, err := json.Marshal(deviceInfo)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, data, 0666)
	if err != nil {
		return err
	}
	return nil
}

type PlatformAdapterLagrangeGo struct {
	Session       *IMSession       `yaml:"-" json:"-"`
	EndPoint      *EndPointInfo    `yaml:"-" json:"-"`
	UIN           uint32           `yaml:"uin" json:"uin"`
	CustomSignUrl string           `yaml:"signUrl" json:"signUrl"`
	QQClient      *client.QQClient `yaml:"-" json:"-"`
	CurState      int              `yaml:"-" json:"loginState"`

	QrcodeData []byte `yaml:"-" json:"-"`
	signUrl    string
	configDir  string
	sig        *info.SigInfo
}

func (pa *PlatformAdapterLagrangeGo) GetGroupInfoAsync(_ string) {}

func LagrangeGoMessageElementsToSealElements(elements []lagMessage.IMessageElement) []message.IMessageElement {
	var segment []message.IMessageElement
	for _, element := range elements {
		switch e := element.(type) {
		case *lagMessage.TextElement:
			segment = append(segment, &message.TextElement{Content: e.Content})
		case *lagMessage.AtElement:
			segment = append(segment, &message.AtElement{Target: strconv.FormatInt(int64(e.Target), 10)})
		case *lagMessage.GroupImageElement:
			segment = append(segment, &message.ImageElement{URL: e.Url})
		case *lagMessage.FriendImageElement:
			segment = append(segment, &message.ImageElement{URL: e.Url})
		case *lagMessage.FaceElement:
			segment = append(segment, &message.FaceElement{FaceID: strconv.Itoa(int(e.FaceID))})
		case *lagMessage.ReplyElement:
			segment = append(segment, &message.ReplyElement{
				ReplySeq: strconv.FormatInt(int64(e.ReplySeq), 10),
				Sender:   strconv.FormatInt(int64(e.Sender), 10),
				GroupID:  strconv.FormatInt(int64(e.GroupID), 10),
				Elements: LagrangeGoMessageElementsToSealElements(e.Elements),
			})
		}
	}
	return segment
}

func FormatLagrangeGoElementGroup(elements []message.IMessageElement) []lagMessage.IMessageElement {
	var segment []lagMessage.IMessageElement
	for _, element := range elements {
		switch e := element.(type) {
		case *message.TextElement:
			segment = append(segment, &lagMessage.TextElement{Content: e.Content})
		case *message.AtElement:
			target, _ := strconv.ParseInt(e.Target, 10, 32)
			segment = append(segment, &lagMessage.AtElement{Target: uint32(target)})
		case *message.ImageElement:
			data, err := io.ReadAll(e.File.Stream)
			if err != nil {
				continue
			}
			segment = append(segment, &lagMessage.GroupImageElement{Stream: data})
		case *message.FaceElement:
			faceID, _ := strconv.ParseInt(e.FaceID, 10, 16)
			segment = append(segment, &lagMessage.FaceElement{FaceID: uint16(faceID)})
		case *message.ReplyElement:
			replySeq, err := strconv.ParseInt(e.ReplySeq, 10, 32)
			if err != nil {
				continue
			}
			senderRaw := UserIDExtract(e.Sender)
			groupIDRaw := UserIDExtract(e.GroupID)
			sender, err := strconv.ParseInt(senderRaw, 10, 64)
			if err != nil {
				continue
			}
			groupID, err := strconv.ParseInt(groupIDRaw, 10, 64)
			if err != nil {
				continue
			}
			segment = append(segment, &lagMessage.ReplyElement{
				ReplySeq: int32(replySeq),
				Sender:   uint64(sender),
				GroupID:  uint64(groupID),
				Elements: FormatLagrangeGoElementGroup(e.Elements),
			})
		}
	}
	return segment
}

func FormatLagrangeGoElementPrivate(elements []message.IMessageElement) []lagMessage.IMessageElement {
	var segment []lagMessage.IMessageElement
	for _, element := range elements {
		switch e := element.(type) {
		case *message.TextElement:
			segment = append(segment, &lagMessage.TextElement{Content: e.Content})
		case *message.AtElement:
			target, _ := strconv.ParseInt(e.Target, 10, 32)
			segment = append(segment, &lagMessage.AtElement{Target: uint32(target)})
		case *message.ImageElement:
			data, err := io.ReadAll(e.File.Stream)
			if err != nil {
				continue
			}
			segment = append(segment, &lagMessage.FriendImageElement{Stream: data})
		case *message.FaceElement:
			faceID, _ := strconv.ParseInt(e.FaceID, 10, 16)
			segment = append(segment, &lagMessage.FaceElement{FaceID: uint16(faceID)})
		case *message.ReplyElement:
			replySeq, err := strconv.ParseInt(e.ReplySeq, 10, 32)
			if err != nil {
				continue
			}
			senderRaw := UserIDExtract(e.Sender)
			groupIDRaw := UserIDExtract(e.GroupID)
			sender, err := strconv.ParseInt(senderRaw, 10, 64)
			if err != nil {
				continue
			}
			groupID, err := strconv.ParseInt(groupIDRaw, 10, 64)
			if err != nil {
				continue
			}
			segment = append(segment, &lagMessage.ReplyElement{
				ReplySeq: int32(replySeq),
				Sender:   uint64(sender),
				GroupID:  uint64(groupID),
				Elements: FormatLagrangeGoElementPrivate(e.Elements),
			})
		}
	}
	return segment
}

func (pa *PlatformAdapterLagrangeGo) Serve() int {
	log := pa.Session.Parent.Logger
	if pa.CustomSignUrl == "" {
		// remember to inject the value of DefaultSignUrl in the build process
		//goland:noinspection GoBoolExpressions //nolint:gocritic
		if DefaultSignUrl == `` {
			panic("DefaultSignUrl is empty")
		}
		pa.signUrl = DefaultSignUrl
	} else {
		pa.signUrl = pa.CustomSignUrl
	}
	pa.EndPoint.ProtocolType = "LagrangeGo"
	appInfo := info.AppList["linux"]

	pa.configDir = filepath.Join(pa.Session.Parent.BaseConfig.DataDir, pa.EndPoint.RelWorkDir)
	// create config dir
	if _, err := os.Stat(pa.configDir); errors.Is(err, fs.ErrNotExist) {
		err = os.MkdirAll(pa.configDir, os.ModePerm)
		if err != nil {
			log.Errorf("create config dir failed: %v", err)
			return 1
		}
	} else if err != nil {
		log.Errorf("stat config dir failed: %v", err)
		return 1
	}

	deviceInfo := LoadDevice(pa.configDir + "/deviceinfo.json")
	log.Debugf("Loaded DeviceInfo: %+v\n", deviceInfo)
	err := SaveDevice(deviceInfo, pa.configDir+"/deviceinfo.json")
	if err != nil {
		log.Errorf("Save DeviceInfo failed: %v", err)
	}

	sigInfo, err := LoadSigInfo(pa.configDir + "/siginfo.gob")
	if err != nil {
		log.Errorf("Load SigInfo failed: %v", err)
		log.Infof("Generating new SigInfo...")
		pa.sig = info.NewSigInfo(8848)
	} else {
		pa.sig = sigInfo
		log.Debugf("Loaded SigInfo: %+v", sigInfo)
	}

	pa.CurState = StateCodeInLogin
	pa.EndPoint.State = 2
	pa.EndPoint.Enable = true
	pa.QQClient = client.NewQQclient(pa.UIN, pa.signUrl, appInfo, deviceInfo, pa.sig)
	err = pa.QQClient.Loop()
	if err != nil {
		log.Errorf("LagrangeGo Client loop failed: %v", err)
		return 1
	}
	go func() {
		for {
			time.Sleep(3 * time.Second)
			if pa.EndPoint.State == 3 || !pa.EndPoint.Enable || pa.EndPoint.State == 1 {
				break
			}
			result, err1 := pa.QQClient.GetQrcodeResult()
			if err1 == nil {
				log.Infof("QrcodeResult: %+v", result)
			} else {
				log.Errorf("GetQrcodeResult failed: %v", err1)
			}
			if result == qrcodeState.Confirmed {
				log.Infof("Qrcode confirmed\n")
				break
			}
			if result == qrcodeState.Expired || result == qrcodeState.Canceled {
				log.Errorf("Qrcode expired or canceled\n")
				pa.EndPoint.State = 3
				pa.EndPoint.Enable = false
				break
			}
			if result.Waitable() {
				qrcodeFile := pa.configDir + "/qrcode.png"
				qrdata, err2 := os.ReadFile(qrcodeFile)
				if err2 != nil {
					log.Errorf("ReadFile failed: %v", err2)
				} else {
					log.Infof("QrcodeData: %v", qrdata)
					pa.QrcodeData = qrdata
					pa.CurState = StateCodeInLoginQrCode
					break
				}
			}
		}
	}()
	_, err = pa.QQClient.Login("", pa.configDir+"/qrcode.png")
	if err != nil {
		log.Errorf("LagrangeGo Client login failed: %v", err)
		pa.EndPoint.State = 3
		pa.EndPoint.Enable = false
		return 1
	}

	log.Infof("LagrangeGo Client login success")

	pa.EndPoint.State = 1
	pa.CurState = StateCodeLoginSuccessed
	pa.EndPoint.Enable = true
	pa.EndPoint.UserID = fmt.Sprintf("QQ:%d", pa.sig.Uin)
	_ = pa.QQClient.RefreshFriendCache()

	// setup event handler
	pa.QQClient.GroupMessageEvent.Subscribe(func(client *client.QQClient, event *lagMessage.GroupMessage) {
		log.Debugf("GroupMessageEvent: %+v", event)
		if event.Sender.Uin == pa.UIN {
			return
		}
		msg := &Message{
			Time:        int64(event.Time),
			MessageType: "group",
			GroupID:     "QQ-Group:" + strconv.FormatInt(int64(event.GroupCode), 10),
			Platform:    "QQ",
			GroupName:   event.GroupName,
			Sender: SenderBase{
				Nickname: event.Sender.Nickname,
				UserID:   "QQ:" + strconv.FormatInt(int64(event.Sender.Uin), 10),
			},
		}
		msg.Segment = LagrangeGoMessageElementsToSealElements(event.Elements)
		pa.Session.ExecuteNew(pa.EndPoint, msg)
	})

	pa.QQClient.PrivateMessageEvent.Subscribe(func(client *client.QQClient, event *lagMessage.PrivateMessage) {
		if event.Sender.Uin == pa.UIN {
			return
		}
		msg := &Message{
			Time:        int64(event.Time),
			MessageType: "private",
			GroupID:     "",
			Platform:    "QQ",
			Sender: SenderBase{
				Nickname: event.Sender.Nickname,
				UserID:   "QQ:" + strconv.FormatInt(int64(event.Sender.Uin), 10),
			},
		}
		msg.Segment = LagrangeGoMessageElementsToSealElements(event.Elements)
		pa.Session.ExecuteNew(pa.EndPoint, msg)
	})

	pa.QQClient.GroupInvitedEvent.Subscribe(func(client *client.QQClient, event *event.GroupInvite) {
		log.Debugf("GroupInvitedEvent: %+v", event)
	})

	pa.QQClient.GroupMemberLeaveEvent.Subscribe(func(client *client.QQClient, event *event.GroupMemberDecrease) {
		log.Debugf("GroupLeaveEvent: %+v", event)
		if event.ExitType == 3 {
			// targetUin := pa.QQClient.GetUin(event.MemberUid, event.GroupUin)
			operatorUin := pa.QQClient.GetUin(event.OperatorUid, event.GroupUin)
			// if targetUin != pa.UIN {
			//	return
			// }
			ctx := &MsgContext{MessageType: "group", EndPoint: pa.EndPoint, Session: pa.Session, Dice: pa.Session.Parent}
			opUID := FormatDiceIDQQ(strconv.Itoa(int(operatorUin)))
			groupID := FormatDiceIDQQGroup(strconv.Itoa(int(event.GroupUin)))
			dm := pa.Session.Parent.Parent
			groupName := dm.TryGetGroupName(groupID)
			userName := dm.TryGetUserName(opUID)

			// Note: 从 gocq 抄过来的，为什么禁言就没有这一段呢？
			skip := false
			skipReason := ""
			banInfo, ok := ctx.Dice.BanList.GetByID(opUID)
			if ok {
				if banInfo.Rank == 30 {
					skip = true
					skipReason = "信任用户"
				}
			}
			if ctx.Dice.IsMaster(opUID) {
				skip = true
				skipReason = "Master"
			}

			var extra string
			if skip {
				extra = fmt.Sprintf("\n取消处罚，原因为%s", skipReason)
			} else {
				ctx.Dice.BanList.AddScoreByGroupKicked(opUID, groupID, ctx)
			}

			txt := fmt.Sprintf("被踢出群: 在QQ群组<%s>(%s)中被踢出，操作者:<%s>(%s)%s", groupName, groupID, userName, opUID, extra)
			log.Info(txt)
			ctx.Notice(txt)
		}
	})

	pa.QQClient.GroupMemberJoinRequestEvent.Subscribe(func(client *client.QQClient, event *event.GroupMemberJoinRequest) {
		log.Debugf("GroupMemberJoinRequestEvent: %+v", event)
	})

	pa.QQClient.GroupMemberJoinEvent.Subscribe(func(client *client.QQClient, event *event.GroupMemberIncrease) {
		log.Debugf("GroupMemberJoinEvent: %+v", event)
		_ = pa.QQClient.RefreshGroupMembersCache(event.GroupUin)
		ctx := &MsgContext{MessageType: "group", EndPoint: pa.EndPoint, Session: pa.Session, Dice: pa.Session.Parent}
		inviterID := FormatDiceIDQQ(strconv.Itoa(int(pa.QQClient.GetUin(event.InvitorUid, event.GroupUin))))
		msg := &Message{
			Time:        time.Now().Unix(),
			MessageType: "group",
			GroupID:     "QQ-Group:" + strconv.FormatInt(int64(event.GroupUin), 10),
			Platform:    "QQ",
			Sender: SenderBase{
				UserID: inviterID,
			},
		}
		newMemberUID := pa.QQClient.GetUin(event.MemberUid, event.GroupUin)
		// 自己加群
		if newMemberUID == pa.UIN {
			pa.Session.OnGroupJoined(ctx, msg)
		} else {
			// 其他人被邀请加群
			msg.Sender.UserID = FormatDiceIDQQ(strconv.Itoa(int(newMemberUID)))
			pa.Session.OnGroupMemberJoined(ctx, msg)
		}
	})

	pa.QQClient.GroupMuteEvent.Subscribe(func(client *client.QQClient, event *event.GroupMute) {
		ctx := &MsgContext{MessageType: "group", EndPoint: pa.EndPoint, Session: pa.Session, Dice: pa.Session.Parent}
		log.Debugf("GroupMuteEvent: %+v", event)
		targetUin := pa.QQClient.GetUin(event.TargetUid, event.GroupUin)
		operatorUin := pa.QQClient.GetUin(event.OperatorUid, event.GroupUin)
		if targetUin == pa.UIN {
			log.Debugf("Muted by %v", operatorUin)
		} else {
			return
		}
		// 解除禁言
		if event.Duration == 0 {
			return
		}
		dm := pa.Session.Parent.Parent
		opUID := FormatDiceIDQQ(strconv.Itoa(int(operatorUin)))
		groupID := FormatDiceIDQQGroup(strconv.Itoa(int(event.GroupUin)))
		groupName := dm.TryGetGroupName(groupID)
		userName := dm.TryGetUserName(opUID)
		ctx.Dice.BanList.AddScoreByGroupMuted(opUID, groupID, ctx)
		txt := fmt.Sprintf("被禁言: 在群组<%s>(%s)中被禁言，时长%d秒，操作者:<%s>(%s)", groupName, groupID, event.Duration, userName, opUID)
		log.Info(txt)
		ctx.Notice(txt)
	})

	err = SaveSigInfo(pa.configDir+"/siginfo.gob", pa.sig)
	if err != nil {
		log.Errorf("Save SigInfo failed: %v", err)
	}
	d := pa.Session.Parent
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	return 0
}

func (pa *PlatformAdapterLagrangeGo) DoRelogin() bool {
	log := pa.Session.Parent.Logger
	if pa.QQClient != nil {
		pa.QQClient.DisConnect()
		err := pa.QQClient.Connect()
		if err != nil {
			log.Errorf("LagrangeGo Reconnect failed: %v", err)
			return false
		}
		return true
	}
	return false
}

func (pa *PlatformAdapterLagrangeGo) SetEnable(enable bool) {
	pa.EndPoint.Enable = enable
	if enable {
		if pa.QQClient != nil {
			pa.QQClient.Stop()
			pa.QQClient = nil
		}
		pa.Serve()
	} else {
		if pa.QQClient != nil {
			pa.QQClient.Stop()
			pa.QQClient = nil
		}
		pa.EndPoint.State = 0
		pa.CurState = StateCodeInit
	}
}

func (pa *PlatformAdapterLagrangeGo) SendSegmentToGroup(ctx *MsgContext, groupID string, msg []message.IMessageElement, flag string) {
}

func (pa *PlatformAdapterLagrangeGo) SendSegmentToPerson(ctx *MsgContext, userID string, msg []message.IMessageElement, flag string) {
}

func (pa *PlatformAdapterLagrangeGo) SendToPerson(ctx *MsgContext, uid string, text string, flag string) {
	log := pa.Session.Parent.Logger
	if text == "" {
		log.Errorf("SendToPerson: text is empty")
		return
	}
	uidraw := UserIDExtract(uid)
	userCode, err := strconv.ParseInt(uidraw, 10, 64)
	if err != nil {
		log.Errorf("ParseInt failed: %v", err)
		return
	}
	elementsRaw := message.ConvertStringMessage(text)
	messageElem := FormatLagrangeGoElementPrivate(elementsRaw)
	_, err = pa.QQClient.SendPrivateMessage(uint32(userCode), messageElem)
	if err != nil {
		log.Errorf("SendToPerson failed: %v", err)
		return
	}
}

func (pa *PlatformAdapterLagrangeGo) SendToGroup(ctx *MsgContext, uid string, text string, flag string) {
	uidraw := UserIDExtract(uid)
	log := pa.Session.Parent.Logger
	if text == "" {
		log.Errorf("SendToGroup: text is empty")
		return
	}
	elementsRaw := message.ConvertStringMessage(text)
	groupCode, err := strconv.ParseInt(uidraw, 10, 64)
	if err != nil {
		log.Errorf("ParseInt failed: %v", err)
		return
	}
	messageElem := FormatLagrangeGoElementGroup(elementsRaw)
	_, err = pa.QQClient.SendGroupMessage(uint32(groupCode), messageElem)
	if err != nil {
		log.Errorf("SendGroupMessage failed: %v", err)
		return
	}
}

func (pa *PlatformAdapterLagrangeGo) SendFileToPerson(ctx *MsgContext, uid string, path string, flag string) {
	pa.SendToPerson(ctx, uid, fmt.Sprintf("[尝试发送文件: %s，但不支持]", filepath.Base(path)), flag)
}

func (pa *PlatformAdapterLagrangeGo) SendFileToGroup(ctx *MsgContext, uid string, path string, flag string) {
	pa.SendToGroup(ctx, uid, fmt.Sprintf("[尝试发送文件: %s，但不支持]", filepath.Base(path)), flag)
}

func (pa *PlatformAdapterLagrangeGo) QuitGroup(ctx *MsgContext, groupId string) {
	log := pa.Session.Parent.Logger
	groupCode, err := strconv.ParseInt(UserIDExtract(groupId), 10, 64)
	if err != nil {
		log.Errorf("ParseInt failed: %v", err)
		return
	}
	req, err := oidb.BuildGroupLeaveReq(uint32(groupCode))
	if err != nil {
		log.Errorf("BuildGroupLeaveReq failed: %v", err)
		return
	}
	response, err := pa.QQClient.SendOidbPacketAndWait(req)
	if err != nil {
		log.Errorf("QuitGroup failed: %v", err)
		return
	}
	_, err = oidb.ParseGroupLeaveResp(response.Data)
	if err != nil {
		log.Errorf("ParseGroupLeaveResp failed: %v", err)
	}
	log.Debugf("QuitGroup success")
}

func (pa *PlatformAdapterLagrangeGo) SetGroupCardName(ctx *MsgContext, name string) {
	log := pa.Session.Parent.Logger
	groupIdRaw := UserIDExtract(ctx.Group.GroupID)
	groupCode, err := strconv.ParseInt(groupIdRaw, 10, 64)
	if err != nil {
		log.Errorf("ParseInt failed: %v", err)
		return
	}
	uidRaw := UserIDExtract(ctx.Player.UserID)
	userCode, err := strconv.ParseInt(uidRaw, 10, 64)
	if err != nil {
		log.Errorf("ParseInt failed: %v", err)
		return
	}
	_ = pa.QQClient.RefreshGroupMembersCache(uint32(groupCode))
	req, err := oidb.BuildGroupRenameMemberReq(uint32(groupCode), pa.QQClient.GetUid(uint32(userCode), uint32(groupCode)), name)
	if err != nil {
		log.Errorf("BuildGroupRenameMemberReq failed: %v", err)
		return
	}
	response, err := pa.QQClient.SendOidbPacketAndWait(req)
	if err != nil {
		log.Errorf("SetGroupCardName failed: %v", err)
	}
	_, err = oidb.ParseGroupRenameMemberResp(response.Data)
	if err != nil {
		log.Errorf("ParseGroupRenameMemberResp failed: %v", err)
		return
	}
}

func (pa *PlatformAdapterLagrangeGo) MemberBan(_ string, _ string, _ int64) {}

func (pa *PlatformAdapterLagrangeGo) MemberKick(_ string, _ string) {}

func (pa *PlatformAdapterLagrangeGo) EditMessage(_ *MsgContext, _, _ string) {}

func (pa *PlatformAdapterLagrangeGo) RecallMessage(_ *MsgContext, _ string) {}
