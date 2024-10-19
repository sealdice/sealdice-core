//go:build !cgo
// +build !cgo

package model

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/glebarez/go-sqlite"
	"github.com/qustavo/sqlhooks/v2"

	log "sealdice-core/utils/kratos"
)

// 覆盖驱动名 sqlite3 会导致 panic, 因此需要创建新的驱动.
//
// database/sql/sql.go:51
const zapDriverName = "sqlite3-log"

func InitZapHook(log *log.Helper) {
	hook := &zapHook{Helper: log, IsPrintSQLDuration: true}
	sql.Register(zapDriverName, sqlhooks.Wrap(new(sqlite.Driver), hook))
}

// make sure zapHook implement all sqlhooks interface.
var _ interface {
	sqlhooks.Hooks
	sqlhooks.OnErrorer
} = (*zapHook)(nil)

// zapHook 使用 zap 记录 SQL 查询和参数
type zapHook struct {
	*log.Helper

	// 是否打印 SQL 耗时
	IsPrintSQLDuration bool
}

// sqlDurationKey 是 context.valueCtx Key
type sqlDurationKey struct{}

func buildQueryArgsFields(query string, args ...interface{}) []interface{} {
	if len(args) == 0 {
		return []interface{}{"查询", query}
	}
	return []interface{}{"查询", query, "参数", args}
}

func (z *zapHook) Before(ctx context.Context, _ string, _ ...interface{}) (context.Context, error) {
	if z == nil || z.Helper == nil {
		return ctx, nil
	}

	if z.IsPrintSQLDuration {
		ctx = context.WithValue(ctx, (*sqlDurationKey)(nil), time.Now())
	}
	return ctx, nil
}

func (z *zapHook) After(ctx context.Context, query string, args ...interface{}) (context.Context, error) {
	if z == nil || z.Helper == nil {
		return ctx, nil
	}

	var durationField string
	if v, ok := ctx.Value((*sqlDurationKey)(nil)).(time.Time); ok {
		durationField = fmt.Sprintf("%v", time.Since(v))
	}

	z.Debugf("SQL 执行后: %v 耗时: %v 秒", buildQueryArgsFields(query, args...), durationField)
	return ctx, nil
}

func (z *zapHook) OnError(_ context.Context, err error, query string, args ...interface{}) error {
	if z == nil || z.Helper == nil {
		return nil
	}
	z.Errorf("SQL 执行出错日志 %v 豹错为: %v", buildQueryArgsFields(query, args...), err)
	return nil
}
