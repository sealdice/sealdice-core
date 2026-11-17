package epoch

import (
	"fmt"
	"os"
	"time"
)

// current 标识当前进程。客户端游标携带该值，用于识别服务重启后
// 从零开始的序号，避免把新进程的记录误判为已接收。
var current = fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())

func Current() string {
	return current
}
