package panicHandler

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/pkg/errors"
)

// this function is needed so that we don't return out of the goroutine when it panics
func tryOnce(errorHandler Errorer, f func()) {
	defer recoverPanics(errorHandler)
	f()
}

func recoverPanics(errorHandler Errorer) {
	if p := recover(); p != nil {
		result := identifyPanic()
		errorHandler.Error("遇到崩溃: ", errors.Errorf("panic: %v \n\n goroutine 遇到崩溃: \n\n 调用栈为: %s \n完整调用栈为 %v \n", p, result, string(debug.Stack())))
	}
}

func identifyPanic() string {
	var name, file string
	var line int
	var pc [16]uintptr

	n := runtime.Callers(3, pc[:])
	for _, pc := range pc[:n] {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		file, line = fn.FileLine(pc)
		name = fn.Name()
		if !strings.HasPrefix(name, "runtime.") {
			break
		}
	}

	switch {
	case name != "":
		return fmt.Sprintf("%v:%v", name, line)
	case file != "":
		return fmt.Sprintf("%v:%v", file, line)
	}

	return fmt.Sprintf("pc:%x", pc)
}
