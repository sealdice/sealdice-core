package dice

import (
	"time"

	"github.com/google/uuid"
)

type AddOnebotEcho struct {
	Token      string
	ConnectURL string
}

func NewOnebotConnItem(v AddOnebotEcho) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.ID = uuid.New().String()
	conn.Platform = "QQ"
	conn.ProtocolType = "milky"
	conn.Enable = false
	conn.RelWorkDir = "extra/milky-" + conn.ID
	conn.Adapter = &PlatformAdapterOnebot{
		EndPoint:   conn,
		Token:      v.Token,
		ConnectURL: v.ConnectURL,
	}
	return conn
}

func ServePureOnebot(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	if ep.Platform == "QQ" {
		conn := ep.Adapter.(*PlatformAdapterOnebot)
		conn.EndPoint = ep
		conn.Session = d.ImSession
		d.Logger.Infof("Pure Onebot V11尝试连接")
		if conn.Serve() != 0 {
			d.Logger.Errorf("连接Pure Onebot V11失败")
			ep.State = 3
			d.LastUpdatedTime = time.Now().Unix()
			d.Save(false)
		}
	}
}
