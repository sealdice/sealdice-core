package dice

import (
	"time"

	"github.com/google/uuid"
)

type AddMilkyEcho struct {
	Token       string
	WsGateway   string
	RestGateway string
}

func NewMilkyConnItem(v AddMilkyEcho) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.ID = uuid.New().String()
	conn.Platform = "QQ"
	conn.ProtocolType = "milky"
	conn.Enable = false
	conn.RelWorkDir = "extra/milky-" + conn.ID
	conn.Adapter = &PlatformAdapterMilky{
		EndPoint:    conn,
		Token:       v.Token,
		WsGateway:   v.WsGateway,
		RestGateway: v.RestGateway,
	}
	return conn
}

func ServeMilky(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	if ep.Platform == "QQ" {
		conn := ep.Adapter.(*PlatformAdapterMilky)
		conn.EndPoint = ep
		conn.Session = d.ImSession
		ep.Session = d.ImSession
		d.Logger.Infof("Milky 尝试连接")
		if conn.Serve() != 0 {
			d.Logger.Errorf("连接Milky失败")
			ep.State = 3
			d.LastUpdatedTime = time.Now().Unix()
			d.Save(false)
		}
	}
}
