package dice

import "github.com/google/uuid"

func NewTelegramConnItem(token string) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.Id = uuid.New().String()
	conn.Platform = "TELEGRAM"
	conn.ProtocolType = ""
	conn.Enable = false
	conn.RelWorkDir = "extra/telegram-" + conn.Id
	conn.Adapter = &PlatformAdapterTelegram{
		EndPoint: conn,
		Token:    token,
	}
	return conn
}

func DiceServeTelegram(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	if ep.Platform == "TELEGRAM" {
		conn := ep.Adapter.(*PlatformAdapterTelegram)
		d.Logger.Infof("Telegram 尝试连接")
		conn.Serve()
	}
}
