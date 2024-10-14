package dice

import (
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"

	log "sealdice-core/utils/kratos"
)

func NewOfficialQQConnItem(appID uint64, token string, appSecret string, onlyQQGuild bool) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.ID = uuid.New().String()
	conn.Platform = "QQ"
	conn.ProtocolType = "official"
	conn.Enable = false
	conn.RelWorkDir = "extra/official-qq-" + conn.ID
	conn.Adapter = &PlatformAdapterOfficialQQ{
		EndPoint:    conn,
		AppID:       appID,
		Token:       token,
		AppSecret:   appSecret,
		OnlyQQGuild: onlyQQGuild,
	}
	return conn
}

func ServerOfficialQQ(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	if ep.Platform == "QQ" && ep.ProtocolType == "official" {
		conn := ep.Adapter.(*PlatformAdapterOfficialQQ)
		d.Logger.Infof("official qq 尝试连接")
		if conn.Serve() != 0 {
			d.Logger.Infof("official qq 连接失败")
			ep.State = 3
			ep.Enable = false
			d.LastUpdatedTime = time.Now().Unix()
			d.Save(false)
		}
	}
}

type DummyLogger struct {
	logger *log.Helper
}

func NewDummyLogger() DummyLogger {
	return DummyLogger{
		logger: log.NewHelper(log.With(log.GetLogger(), "caller", "officialQQ")),
	}
}

func (d DummyLogger) Debug(v ...interface{}) {
	d.logger.Debug(output(v...))
}

func (d DummyLogger) Info(v ...interface{}) {
	d.logger.Debug(output(v...))
}

func (d DummyLogger) Warn(v ...interface{}) {
	d.logger.Warn(output(v...))
}

func (d DummyLogger) Error(v ...interface{}) {
	d.logger.Error(output(v...))
}

func (d DummyLogger) Debugf(format string, v ...interface{}) {
	d.logger.Debug(output(fmt.Sprintf(format, v...)))
}

func (d DummyLogger) Infof(format string, v ...interface{}) {
	d.logger.Debug(output(fmt.Sprintf(format, v...)))
}

func (d DummyLogger) Warnf(format string, v ...interface{}) {
	d.logger.Warn(output(fmt.Sprintf(format, v...)))
}

func (d DummyLogger) Errorf(format string, v ...interface{}) {
	d.logger.Error(output(fmt.Sprintf(format, v...)))
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

// 参考: https://gist.github.com/brydavis/0c7da92bd508195744708eeb2b54ac96
var reUrlExtract = regexp.MustCompile(`(http:\/\/www\.|https:\/\/www\.|http:\/\/|https:\/\/|\/|\/\/)?[A-z0-9_-]*?[:]?[A-z0-9_-]*?[@]?[A-z0-9]+([\-\.]{1}[a-z0-9]+)*\.[a-z]{2,5}(:[0-9]{1,5})?(\/.*)?`)

func textLinkStrip(text string) string {
	urls := reUrlExtract.FindAllString(text, -1)

	// 在每个URL中将"."替换为"_"
	for _, url := range urls {
		replacedURL := strings.ReplaceAll(url, ".", "_")
		replacedURL = strings.ReplaceAll(replacedURL, "https://", "https_")
		replacedURL = strings.ReplaceAll(replacedURL, "http://", "http_")
		// replacedURL = strings.ReplaceAll(replacedURL, "/", "_")
		text = strings.ReplaceAll(text, url, replacedURL)
	}

	return text
}
