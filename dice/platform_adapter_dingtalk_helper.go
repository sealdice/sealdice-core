package dice

import "github.com/google/uuid"

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
		ep.BindRuntime(d.ImSession)
		d.Logger.Infof("Dingtalk 尝试连接")
		_ = StartEndpointLifecycle(d, ep)
	}
}
