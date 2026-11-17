package public_dice_test

import (
	"encoding/json"
	"strings"
	"testing"

	"sealdice-core/utils/public_dice"
)

func TestEndpointAppIDJSON(t *testing.T) {
	t.Parallel()

	official, err := json.Marshal(&public_dice.Endpoint{Platform: "QQ", UID: "OpenQQ:1", AppID: "123456789"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(official), `"appId":"123456789"`) {
		t.Fatalf("official endpoint JSON = %s", official)
	}

	ordinary, err := json.Marshal(&public_dice.Endpoint{Platform: "QQ", UID: "QQ:123456"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(ordinary), "appId") {
		t.Fatalf("ordinary endpoint JSON unexpectedly contains appId: %s", ordinary)
	}

	tick, err := json.Marshal(&public_dice.TickEndpoint{UID: "OpenQQ:1", AppID: "123456789"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(tick), `"appId":"123456789"`) {
		t.Fatalf("official tick JSON = %s", tick)
	}
}
