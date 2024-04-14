package dice

import (
	"strconv"
	"time"

	"github.com/google/uuid"
)

func NewLagrangeGoConnItem(uin uint32) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.ID = uuid.New().String()
	conn.Platform = "QQ"
	conn.ProtocolType = "LagrangeGo"
	conn.Enable = false
	conn.RelWorkDir = "extra/LagrangeGo" + "-qq" + strconv.Itoa(int(uin))
	conn.Adapter = &PlatformAdapterLagrangeGo{
		EndPoint: conn,
		UIN:      uin,
	}
	return conn
}

func ServeLagrangeGo(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	conn := ep.Adapter.(*PlatformAdapterLagrangeGo)
	d.Logger.Infof("LagrangeGo 尝试连接")
	if conn.Serve() == 0 {
	} else {
		d.Logger.Errorf("连接 LagrangeGo 失败")
		ep.State = 3
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)
	}
}
