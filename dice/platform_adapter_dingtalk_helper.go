package dice

import (
	"github.com/google/uuid"
	"time"
)

func NewDingTalkConnItem(clientID string, token string, nickname string, robotCode string) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.ID = uuid.New().String()
	conn.Platform = "DINGTALK"
	conn.Nickname = nickname
	conn.ProtocolType = ""
	conn.Enable = false
	conn.RelWorkDir = "extra/dingtalk-" + conn.ID
	conn.Adapter = &PlatformAdapterDingTalk{
		EndPoint:  conn,
		ClientID:  clientID,
		Token:     token,
		RobotCode: robotCode,
	}
	return conn
}

func ServeDingTalk(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	if ep.Platform == "DINGTALK" {
		conn := ep.Adapter.(*PlatformAdapterDingTalk)
		d.Logger.Infof("Dingtalk 尝试连接")
		if conn.Serve() != 0 {
			d.Logger.Errorf("连接Dingtalk 失败")
			ep.State = 3
			ep.Enable = false
			d.LastUpdatedTime = time.Now().Unix()
			d.Save(false)
		}
	}
}
