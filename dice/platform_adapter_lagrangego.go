package dice

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/LagrangeDev/LagrangeGo/client"
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

func (pa *PlatformAdapterLagrangeGo) Serve() int {
	log := pa.Session.Parent.Logger
	if pa.CustomSignUrl == "" {
		// remember to inject the value of DefaultSignUrl in the build process
		// nolint:gocritic
		//goland:noinspection GoBoolExpressions
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
	// log.Infof("configDir: %s\n", pa.configDir)
	deviceInfo := LoadDevice(pa.configDir + "/deviceinfo.json")
	// log.Infof("Loaded DeviceInfo: %+v\n", deviceInfo)
	err := SaveDevice(deviceInfo, pa.configDir+"/deviceinfo.json")
	if err != nil {
		log.Errorf("Save DeviceInfo failed: %v", err)
	}

	// create config dir
	if _, err = os.Stat(pa.configDir); os.IsNotExist(err) {
		err = os.MkdirAll(pa.configDir, os.ModePerm)
		if err != nil {
			log.Errorf("create config dir failed: %v", err)
			return 1
		}
	}

	sigInfo, err := LoadSigInfo(pa.configDir + "/siginfo.gob")
	if err != nil {
		log.Errorf("Load SigInfo failed: %v", err)
		log.Infof("Generating new SigInfo...")
		pa.sig = info.NewSigInfo(8848)
	} else {
		pa.sig = sigInfo
		// log.Infof("Loaded SigInfo: %+v", sigInfo)
	}

	pa.CurState = StateCodeInLogin
	pa.EndPoint.State = 2
	pa.EndPoint.Enable = true
	pa.QQClient = client.NewQQclient(pa.UIN, pa.signUrl, appInfo, deviceInfo, pa.sig)
	pa.QQClient.Loop()
	go func() {
		for {
			time.Sleep(3 * time.Second)
			result, err1 := pa.QQClient.GetQrcodeResult()
			if err1 == nil {
				log.Infof("QrcodeResult: %+v", result)
			} else {
				log.Errorf("GetQrcodeResult failed: %v", err1)
			}
			if pa.EndPoint.State == 3 || !pa.EndPoint.Enable || pa.EndPoint.State == 1 {
				break
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
	// 上游会直接 panic 导致程序退出，而且还在另一个 goroutine 里，奶奶滴
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
	pa.QQClient.RefreshFriendCache()

	// setup event handler
	pa.QQClient.GroupMessageEvent.Subscribe(func(client *client.QQClient, event *lagMessage.GroupMessage) {
		// log.Infof("GroupMessageEvent: %+v\n", event)
		// log.Infof("GroupMessageEventSender: %+v\n", event.Sender)
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
		var segment []message.IMessageElement
		for _, element := range event.Elements {
			switch e := element.(type) {
			case *lagMessage.TextElement:
				segment = append(segment, &message.TextElement{Content: e.Content})
			case *lagMessage.AtElement:
				segment = append(segment, &message.AtElement{Target: strconv.FormatInt(int64(e.Target), 10)})
			case *lagMessage.GroupImageElement:
				// log.Infof("GroupImageElement: %+v\n", e)
				segment = append(segment, &message.ImageElement{URL: e.Url})
			case *lagMessage.ReplyElement:
				// log.Infof("ReplyElement: %d\n", e.ReplySeq)
			}
		}
		msg.Segment = segment
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
		var segment []message.IMessageElement
		for _, element := range event.Elements {
			switch e := element.(type) {
			case *lagMessage.TextElement:
				segment = append(segment, &message.TextElement{Content: e.Content})
			case *lagMessage.AtElement:
				segment = append(segment, &message.AtElement{Target: strconv.FormatInt(int64(e.Target), 10)})
			case *lagMessage.GroupImageElement:
				segment = append(segment, &message.ImageElement{URL: e.Url})
			case *lagMessage.ReplyElement:
			}
		}
		msg.Segment = segment
		pa.Session.ExecuteNew(pa.EndPoint, msg)
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
	messageElem := []lagMessage.IMessageElement{&lagMessage.TextElement{Content: text}}
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
	groupCode, err := strconv.ParseInt(uidraw, 10, 64)
	if err != nil {
		log.Errorf("ParseInt failed: %v", err)
		return
	}
	messageElem := []lagMessage.IMessageElement{&lagMessage.TextElement{Content: text}}
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

func (pa *PlatformAdapterLagrangeGo) QuitGroup(_ *MsgContext, _ string) {}

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
	pa.QQClient.RefreshGroupCache(uint32(groupCode))
	req, err := oidb.BuildGroupRenameMemberReq(uint32(groupCode), pa.QQClient.GetUidFromGroup(uint32(userCode), uint32(groupCode)), name)
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
