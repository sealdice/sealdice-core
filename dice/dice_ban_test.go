package dice_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"sealdice-core/dice"
)

func TestCanReplyBlacklistedHelpMasterPerUserCooldown(t *testing.T) {
	var banList dice.BanListInfo
	banList.Init()
	banList.HelpMasterCooldownMinutes = 10

	base := time.Unix(1_700_000_000, 0)

	if !banList.CanReplyBlacklistedHelpMaster("QQ:1001", base) {
		t.Fatal("expected first help-master reply for user to be allowed")
	}

	if banList.CanReplyBlacklistedHelpMaster("QQ:1001", base.Add(5*time.Minute)) {
		t.Fatal("expected same user to be throttled within cooldown")
	}

	if !banList.CanReplyBlacklistedHelpMaster("QQ:1002", base.Add(5*time.Minute)) {
		t.Fatal("expected different user to bypass another user's cooldown")
	}

	if !banList.CanReplyBlacklistedHelpMaster("QQ:1001", base.Add(11*time.Minute)) {
		t.Fatal("expected same user to be allowed after cooldown elapsed")
	}
}

func TestCanNotifyBlacklistedUserPerGroupAndUserCooldown(t *testing.T) {
	var banList dice.BanListInfo
	banList.Init()
	banList.BanNotifyIntervalMinutes = 10

	base := time.Unix(1_700_000_000, 0)

	if !banList.CanNotifyBlacklistedUser("QQ-Group:1001", "QQ:2001", base) {
		t.Fatal("expected first notice to be allowed")
	}
	if banList.CanNotifyBlacklistedUser("QQ-Group:1001", "QQ:2001", base.Add(5*time.Minute)) {
		t.Fatal("expected same user in same group to be throttled")
	}
	if !banList.CanNotifyBlacklistedUser("QQ-Group:1002", "QQ:2001", base.Add(5*time.Minute)) {
		t.Fatal("expected the same user in another group to have an independent cooldown")
	}
	if !banList.CanNotifyBlacklistedUser("QQ-Group:1001", "QQ:2002", base.Add(5*time.Minute)) {
		t.Fatal("expected another user in the same group to have an independent cooldown")
	}
	if !banList.CanNotifyBlacklistedUser("QQ-Group:1001", "QQ:2001", base.Add(11*time.Minute)) {
		t.Fatal("expected notice to be allowed after cooldown elapsed")
	}
}

func TestCanNotifyBlacklistedUserWithoutCooldown(t *testing.T) {
	var banList dice.BanListInfo
	banList.Init()
	banList.BanNotifyIntervalMinutes = -1

	now := time.Unix(1_700_000_000, 0)
	if !banList.CanNotifyBlacklistedUser("QQ-Group:1001", "QQ:2001", now) {
		t.Fatal("expected first notice to be allowed")
	}
	if !banList.CanNotifyBlacklistedUser("QQ-Group:1001", "QQ:2001", now) {
		t.Fatal("expected every notice to be allowed when cooldown is disabled")
	}
}

func TestCanNotifyBlacklistedUserConcurrent(t *testing.T) {
	var banList dice.BanListInfo
	banList.Init()

	const goroutines = 32
	var allowed atomic.Int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	now := time.Unix(1_700_000_000, 0)

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if banList.CanNotifyBlacklistedUser("QQ-Group:1001", "QQ:2001", now) {
				allowed.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if got := allowed.Load(); got != 1 {
		t.Fatalf("expected exactly one concurrent notice, got %d", got)
	}
}
