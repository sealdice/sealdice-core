package dice

import "github.com/google/uuid"

type AddOnebotEcho struct {
	Token         string
	ConnectURL    string
	ReverseURL    string
	ReverseSuffix string
	Mode          string
}

func NewOnebotConnItem(v AddOnebotEcho) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.ID = uuid.New().String()
	conn.Platform = "QQ"
	conn.ProtocolType = "pureonebot"
	conn.Enable = false
	conn.RelWorkDir = "extra/pureonebot-" + conn.ID // 也不知道干啥的
	if v.ReverseSuffix == "" {
		v.ReverseSuffix = "/ws"
	}
	conn.Adapter = &PlatformAdapterOnebot{
		EndPoint:      conn,
		Token:         v.Token,
		ConnectURL:    v.ConnectURL,
		ReverseSuffix: v.ReverseSuffix,
		ReverseUrl:    v.ReverseURL,
		Mode:          v.Mode,
	}
	return conn
}

func ServePureOnebot(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	if ep.Platform == "QQ" {
		ep.BindRuntime(d.ImSession)
		d.Logger.Infof("Pure Onebot V11尝试连接")
		_ = StartEndpointLifecycle(d, ep)
	}
}
