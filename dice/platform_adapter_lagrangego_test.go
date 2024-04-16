package dice

import (
	"testing"
	"time"
)

func TestAdapterLogin(t *testing.T) {
	t.Skip("Skip the test") // remove this line to run the test
	t.Log("Adapter login test started")
	conn := &PlatformAdapterLagrangeGo{
		Session:   &IMSession{},
		EndPoint:  &EndPointInfo{},
		UIN:       0,
		signUrl:   "",
		configDir: "../data/default/extra/lagrangeGoTest",
	}
	conn.Serve()
	t.Log("Adapter login test finished")
	time.Sleep(5 * time.Second)
	// conn.SendToGroup(&MsgContext{}, "QQ-Group:", "Hello, LagrangeGo!", "")
	t.Log("Message sent to group")
	time.Sleep(5 * time.Second)
	// conn.SendToPerson(&MsgContext{}, "QQ:", "Hello, LagrangeGo!", "")
	t.Log("Message sent to person")
	time.Sleep(5000 * time.Second)
}
