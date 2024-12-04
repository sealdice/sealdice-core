package paniclog

import (
	"fmt"
	"io"
	"os"
	"time"

	log "sealdice-core/utils/kratos"
)

func InitPanicLog() {
	// TODO: 全局写死写入在data目录，这东西几乎没有任何值得配置的
	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatalf("未发现data文件夹，且未能创建data文件夹，请检查写入权限: %v", err)
	}
	f, err := os.OpenFile("./data/panic.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	// Copied from https://github.com/rclone/rclone/tree/master/fs/log
	// 这里GPT说，因为使用了APPEND，所以保证了不需要使用SEEK。但是rclone既然这么用了，我决定相信rclone的处理。
	_, err = f.Seek(0, io.SeekEnd)
	if err != nil {
		log.Errorf("Failed to seek log file to end: %v", err)
	}
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	separator := fmt.Sprintf("\n-------- %s --------\n", currentTime)
	// 将分割线写入文件
	_, err = f.WriteString(separator)
	if err != nil {
		log.Fatalf("Failed to write separator to log file: %v", err)
	}
	redirectStderr(f)
}
