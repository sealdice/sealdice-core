package dice

import (
	"time"

	"github.com/google/uuid"
)

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
		conn := ep.Adapter.(*PlatformAdapterKook)
		d.Logger.Infof("KOOK 尝试连接")
		if conn.Serve() != 0 {
			d.Logger.Errorf("连接KOOK服务失败")
			ep.State = 3
			d.LastUpdatedTime = time.Now().Unix()
			d.Save(false)
		}
	}
}
