package dice

import (
	"testing"
	"time"
)

func TestCanReplyBlacklistedHelpMasterPerUserCooldown(t *testing.T) {
	var banList BanListInfo
	banList.Init()

	base := time.Unix(1_700_000_000, 0)

	if !banList.CanReplyBlacklistedHelpMaster("QQ:1001", base) {
		t.Fatal("expected first help-master reply for user to be allowed")
	}

	if banList.CanReplyBlacklistedHelpMaster("QQ:1001", base.Add(30*time.Minute)) {
		t.Fatal("expected same user to be throttled within cooldown")
	}

	if !banList.CanReplyBlacklistedHelpMaster("QQ:1002", base.Add(30*time.Minute)) {
		t.Fatal("expected different user to bypass another user's cooldown")
	}

	if !banList.CanReplyBlacklistedHelpMaster("QQ:1001", base.Add(blacklistedHelpMasterCooldown+time.Minute)) {
		t.Fatal("expected same user to be allowed after cooldown elapsed")
	}
}
