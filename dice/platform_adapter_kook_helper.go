package dice

import "github.com/google/uuid"

func NewKookConnItem(token string) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.ID = uuid.New().String()
	conn.Platform = "KOOK"
	conn.ProtocolType = ""
	conn.Enable = false
	conn.RelWorkDir = "extra/kook-" + conn.ID
	conn.Adapter = &PlatformAdapterKook{
		EndPoint: conn,
		Token:    token,
	}
	return conn
}

func ServeKook(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	if ep.Platform == "KOOK" {
		ep.BindRuntime(d.ImSession)
		d.Logger.Infof("KOOK 尝试连接")
		_ = StartEndpointLifecycle(d, ep)
	}
}
