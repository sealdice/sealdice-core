package dice

import "github.com/google/uuid"

// NewDiscordConnItem 本来没必要写这个的，但是不知道为啥依赖出问题
func NewDiscordConnItem(token string) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.Id = uuid.New().String()
	conn.Platform = "Discord"
	conn.ProtocolType = ""
	conn.Enable = false
	conn.RelWorkDir = "/extend"
	conn.Adapter = &PlatformAdapterDiscord{
		EndPoint: conn,
		Token:    token,
	}
	return conn
}

// ServeDiscord gocqhttp_helper 中有一个相同的待重构方法，为了避免阻碍重构，先不写在一起了
func ServeDiscord(d *Dice, ep *EndPointInfo) {
	if ep.Platform == "Discord" {
		conn := ep.Adapter.(*PlatformAdapterDiscord)
		d.Logger.Infof("Discord 尝试连接")
		if conn.Serve() == 0 {
			conn.DiceServing = true
			d.Logger.Infof("Discord 服务连接成功")
		}
	}
}
