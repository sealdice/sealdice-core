package dice

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	logger "sealdice-core/logger"
)

func NewOfficialQQConnItem(appID, appSecret, uin string, onlyQQGuild bool) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.ID = uuid.New().String()
	conn.Platform = "QQ"
	conn.ProtocolType = "official"
	conn.Enable = false
	conn.RelWorkDir = "extra/official-qq-" + conn.ID
	conn.Adapter = &PlatformAdapterOfficialQQ{
		EndPoint:    conn,
		AppID:       appID,
		AppSecret:   appSecret,
		UIN:         uin,
		OnlyQQGuild: onlyQQGuild,
	}
	return conn
}

func ServerOfficialQQ(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	if ep.Platform == "QQ" && ep.ProtocolType == "official" {
		ep.BindRuntime(d.ImSession)
		// 连接状态、重试和日志统一由 ServeQQ 及其官方 QQ 分支负责。
		ServeQQ(d, ep)
	}
}

type DummyLogger struct {
	logger *zap.SugaredLogger
}

func NewDummyLogger() DummyLogger {
	return DummyLogger{
		logger: zap.S().Named(logger.LogKeyAdapter),
	}
}

const officialQQVerboseEnv = "SEALDICE_OFFICIAL_QQ_VERBOSE"

func officialQQVerbose() bool {
	return os.Getenv(officialQQVerboseEnv) == "1"
}

func (d DummyLogger) Debug(v ...interface{}) {
	if officialQQVerbose() {
		d.logger.Debug("official qq sdk: ", fmt.Sprint(v...))
	}
}

func (d DummyLogger) Info(v ...interface{}) {
	if officialQQVerbose() {
		d.logger.Info("official qq sdk: ", fmt.Sprint(v...))
	}
}

func (d DummyLogger) Warn(v ...interface{}) {
	d.logger.Warn(output(v...))
}

func (d DummyLogger) Error(v ...interface{}) {
	msg := fmt.Sprint(v...)
	if isOfficialQQReconnectMessage(msg) {
		d.logger.Warn("official qq sdk: " + friendlyOfficialQQReconnectMessage(msg))
		return
	}
	d.logger.Error(output(v...))
}

func (d DummyLogger) Debugf(format string, v ...interface{}) {
	if officialQQVerbose() {
		d.logger.Debugf("official qq sdk: "+format, v...)
	}
}

func (d DummyLogger) Infof(format string, v ...interface{}) {
	if officialQQVerbose() {
		d.logger.Infof("official qq sdk: "+format, v...)
	}
}

func (d DummyLogger) Warnf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	if isOfficialQQReconnectMessage(msg) {
		d.logger.Warn("official qq sdk: " + friendlyOfficialQQReconnectMessage(msg))
		return
	}
	d.logger.Warn(output(msg))
}

func (d DummyLogger) Errorf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	if isOfficialQQReconnectMessage(msg) {
		d.logger.Warn("official qq sdk: " + friendlyOfficialQQReconnectMessage(msg))
		return
	}
	d.logger.Error(output(msg))
}

// QQ 官方网关约每小时会通过 op=7 要求连接续期，旧 session 随后可能返回
// op=9 invalid session。这是正常的轮换流程，不应按 Error 报警。
func isOfficialQQReconnectMessage(msg string) bool {
	for _, token := range []string{
		"code:9000", "need reconnect", "code:9005", "invalid session",
		"4009", "Session timed out",
	} {
		if strings.Contains(msg, token) {
			return true
		}
	}
	return false
}

func friendlyOfficialQQReconnectMessage(msg string) string {
	if strings.Contains(msg, "正常续期") {
		return strings.TrimSpace(msg)
	}
	return "QQ 官方要求 WebSocket 重连（正常续期），海豹会自动恢复。详情: " + strings.TrimSpace(msg)
}

func (d DummyLogger) Sync() error {
	return nil
}

func output(v ...interface{}) string {
	pc, file, line, _ := runtime.Caller(3)
	file = filepath.Base(file)
	funcName := strings.TrimPrefix(filepath.Ext(runtime.FuncForPC(pc).Name()), ".")

	logFormat := "official qq sdk: %s:%d:%s " + fmt.Sprint(v...) + "\n"
	return fmt.Sprintf(logFormat, file, line, funcName)
}

// 链接现在原样发送；注释掉下面的正则防止以后要用
// 参考: https://gist.github.com/brydavis/0c7da92bd508195744708eeb2b54ac96
// var reUrlExtract = regexp.MustCompile(`(http:\/\/www\.|https:\/\/www\.|http:\/\/|https:\/\/|\/|\/\/)?[A-z0-9_-]*?[:]?[A-z0-9_-]*?[@]?[A-z0-9]+([\-\.]{1}[a-z0-9]+)*\.[a-z]{2,5}(:[0-9]{1,5})?(\/.*)?`)

// 不再使用，为了防止以后要用，暂时保留
func textLinkStrip(text string) string {
	return text
}
