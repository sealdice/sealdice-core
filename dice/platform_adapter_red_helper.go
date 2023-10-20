package dice

import (
	"time"

	"github.com/google/uuid"
)

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
	conn := ep.Adapter.(*PlatformAdapterRed)
	d.Logger.Infof("red 尝试连接")
	if conn.Serve() == 0 {
	} else {
		d.Logger.Errorf("连接 red 服务失败")
		ep.State = 3
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)
	}
}
