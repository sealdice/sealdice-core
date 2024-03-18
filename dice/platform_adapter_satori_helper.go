package dice

import (
	"time"

	"github.com/google/uuid"
)

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
	conn := ep.Adapter.(*PlatformAdapterSatori)
	d.Logger.Infof("satori 尝试连接")
	if conn.Serve() == 0 {
	} else {
		d.Logger.Errorf("连接 satori 服务失败")
		ep.State = 3
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)
	}
}
