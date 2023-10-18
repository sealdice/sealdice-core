package dice

import "github.com/google/uuid"

func NewTelegramConnItem(token string, proxy string) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.ID = uuid.New().String()
	conn.Platform = "TG"
	conn.ProtocolType = ""
	conn.Enable = false
	conn.RelWorkDir = "extra/telegram-" + conn.ID
	conn.Adapter = &PlatformAdapterTelegram{
		EndPoint: conn,
		Token:    token,
		ProxyURL: proxy,
	}
	return conn
}

func ServeTelegram(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	if ep.Platform == "TG" {
		conn := ep.Adapter.(*PlatformAdapterTelegram)
		d.Logger.Infof("Telegram 尝试连接")
		conn.Serve()
	}
}
