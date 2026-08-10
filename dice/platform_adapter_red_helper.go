package dice

import "github.com/google/uuid"

func NewRedConnItem(host string, port int, token string) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.ID = uuid.New().String()
	conn.Platform = "QQ"
	conn.ProtocolType = "red"
	conn.Enable = false
	conn.RelWorkDir = "extra/red-" + conn.ID
	conn.Adapter = &PlatformAdapterRed{
		EndPoint: conn,
		Host:     host,
		Port:     port,
		Token:    token,
	}
	return conn
}

func ServeRed(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	d.Logger.Infof("red 尝试连接")
	ep.BindRuntime(d.ImSession)
	_ = StartEndpointLifecycle(d, ep)
}
