package log

// GoVNRErrorer 是封装普通logger的结构体
type GoVNRErrorer struct {
	logger *Helper
}

// NewLoggerWrapper 创建新的封装器
func GOVNRErrorWrapper(l *Helper) *GoVNRErrorer {
	return &GoVNRErrorer{logger: l}
}

// Error 实现Errorer接口
func (lw *GoVNRErrorer) Error(err error) {
	lw.logger.Error("Error occurred: %v", err)
}
