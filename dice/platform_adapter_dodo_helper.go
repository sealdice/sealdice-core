package dice

import "github.com/google/uuid"

func NewDodoConnItem(clientID string, token string) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.ID = uuid.New().String()
	conn.Platform = "DODO"
	conn.ProtocolType = ""
	conn.Enable = false
	conn.RelWorkDir = "extra/dodo-" + conn.ID
	conn.Adapter = &PlatformAdapterDodo{
		EndPoint:      conn,
		ClientID:      clientID,
		Token:         token,
		UserPermCache: new(SyncMap[string, *SyncMap[string, *GuildPermCacheItem]]),
	}
	return conn
}

func ServeDodo(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	if ep.Platform == "DODO" {
		ep.BindRuntime(d.ImSession)
		d.Logger.Infof("Dodo 尝试连接")
		_ = StartEndpointLifecycle(d, ep)
	}
}
