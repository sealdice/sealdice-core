package dice

import "github.com/google/uuid"

func NewSatoriConnItem(platform string, host string, port int, token string) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.ID = uuid.New().String()
	conn.Platform = platform
	conn.ProtocolType = "satori"
	conn.Enable = false
	conn.RelWorkDir = "extra/satori-" + platform + "-" + conn.ID
	conn.Adapter = &PlatformAdapterSatori{
		EndPoint: conn,
		Version:  SatoriProtocolVersion,
		Platform: platform,
		Host:     host,
		Port:     port,
		Token:    token,
	}
	return conn
}

func ServeSatori(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	d.Logger.Infof("satori 尝试连接")
	ep.BindRuntime(d.ImSession)
	_ = StartEndpointLifecycle(d, ep)
}
