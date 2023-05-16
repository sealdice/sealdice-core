package dice

import "github.com/google/uuid"

func NewMinecraftConnItem(url string) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.Id = uuid.New().String()
	conn.Platform = "MC"
	conn.ProtocolType = ""
	conn.Enable = false
	conn.RelWorkDir = "extra/minecraft-" + conn.Id
	conn.Adapter = &PlatformAdapterMinecraft{
		EndPoint:   conn,
		ConnectUrl: url,
	}
	return conn
}

func ServeMinecraft(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	if ep.Platform == "MC" {
		conn := ep.Adapter.(*PlatformAdapterMinecraft)
		d.Logger.Infof("Minecraft 尝试连接")
		conn.Serve()
	}
}
