package random

import (
	"encoding/binary"
	"hash/fnv"
	"os"
	"runtime"
	"time"
	"unsafe"
)

// generateRandSeed 混合时间戳、对象地址、进程ID和堆栈信息生成初始种子。
func generateRandSeed() uint64 {
	timestamp := time.Now().UnixNano()

	type tempObj struct{ val int }
	obj := tempObj{val: 42}
	objPtr := uint64(uintptr(unsafe.Pointer(&obj)))

	pid := uint64(os.Getpid())

	buf := make([]byte, 1024)
	n := runtime.Stack(buf, true)
	stackInfo := buf[:n]
	h := fnv.New64a()

	_ = binary.Write(h, binary.LittleEndian, timestamp)
	_ = binary.Write(h, binary.LittleEndian, objPtr)
	_ = binary.Write(h, binary.LittleEndian, pid)
	_, _ = h.Write(stackInfo)

	return h.Sum64()
}
