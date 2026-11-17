package main

import (
	"testing"
	"time"

	"sealdice-core/dice"
	"sealdice-core/logger"
)

// 下载失败后监听必须继续工作：否则之后的每次升级请求都得不到回复，
// 调用方会一直卡在等待下载结果上，只有重启进程才能恢复。
func TestUpdateRequestListenAnswersRetriesAfterFailure(t *testing.T) {
	dm := &dice.DiceManager{
		Dice:           &dice.Dice{Logger: logger.M()},
		AppVersionCode: 100,
		// 线上版本比本地旧：downloadUpdate 在发起下载前就返回错误，不访问网络。
		AppVersionOnline:     &dice.VersionInfo{VersionLatestCode: 50},
		UpdateRequestChan:    make(chan struct{}, 1),
		UpdateDownloadedChan: make(chan string, 1),
	}
	go UpdateRequestListen(dm)

	for attempt := 1; attempt <= 3; attempt++ {
		dm.UpdateRequestChan <- struct{}{}
		select {
		case reason := <-dm.UpdateDownloadedChan:
			if reason == "" {
				t.Fatalf("attempt %d: got success, want the download failure reason", attempt)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("attempt %d: listener did not answer after an earlier failure", attempt)
		}
	}
}
