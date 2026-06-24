package service_test

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"sealdice-core/dice/service"
	"sealdice-core/model"
	"sealdice-core/utils/constant"
)

type logInfoTestOperator struct {
	db     *gorm.DB
	dbType string
}

func (o *logInfoTestOperator) Init(_ context.Context) error           { return nil }
func (o *logInfoTestOperator) Type() string                           { return o.dbType }
func (o *logInfoTestOperator) DBCheck()                               {}
func (o *logInfoTestOperator) GetDataDB(_ constant.DBMode) *gorm.DB   { return o.db }
func (o *logInfoTestOperator) GetLogDB(_ constant.DBMode) *gorm.DB    { return o.db }
func (o *logInfoTestOperator) GetCensorDB(_ constant.DBMode) *gorm.DB { return o.db }
func (o *logInfoTestOperator) Close()                                 {}

func TestLogGetInfoUsesSQLiteSequenceForRows(t *testing.T) {
	db := newLogInfoTestDB(t)
	seedLogInfoTestDB(t, db)

	if err := db.Exec("UPDATE sqlite_sequence SET seq = ? WHERE name = ?", 99, "logs").Error; err != nil {
		t.Fatalf("update logs sqlite sequence: %v", err)
	}
	if err := db.Exec("UPDATE sqlite_sequence SET seq = ? WHERE name = ?", 123, "log_items").Error; err != nil {
		t.Fatalf("update log_items sqlite sequence: %v", err)
	}

	got, err := service.LogGetInfo(&logInfoTestOperator{db: db, dbType: constant.SQLITE})
	if err != nil {
		t.Fatalf("LogGetInfo() error = %v", err)
	}

	want := []int{3, 4, 99, 123}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LogGetInfo() = %v, want %v", got, want)
	}
}

func TestLogGetInfoFallsBackToMaxIDWhenSQLiteSequenceMissing(t *testing.T) {
	db := newLogInfoTestDB(t)
	seedLogInfoTestDB(t, db)

	if err := db.Exec("DELETE FROM sqlite_sequence WHERE name IN (?, ?)", "logs", "log_items").Error; err != nil {
		t.Fatalf("delete sqlite sequence: %v", err)
	}

	got, err := service.LogGetInfo(&logInfoTestOperator{db: db, dbType: constant.SQLITE})
	if err != nil {
		t.Fatalf("LogGetInfo() error = %v", err)
	}

	want := []int{3, 4, 3, 4}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LogGetInfo() = %v, want %v", got, want)
	}
}

func TestLogGetInfoUsesMySQLTableRows(t *testing.T) {
	op, mock := newLogInfoMySQLTestOperator(t)
	maxLogIDQuery := regexp.QuoteMeta("SELECT MAX(id) FROM `logs`")
	maxItemIDQuery := regexp.QuoteMeta("SELECT MAX(id) FROM `log_items`")
	tableRowsQuery := regexp.QuoteMeta("SELECT TABLE_ROWS FROM information_schema.tables WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?")

	mock.ExpectQuery(maxLogIDQuery).WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(3))
	mock.ExpectQuery(tableRowsQuery).WithArgs("logs").WillReturnRows(sqlmock.NewRows([]string{"TABLE_ROWS"}).AddRow(88))
	mock.ExpectQuery(maxItemIDQuery).WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(4))
	mock.ExpectQuery(tableRowsQuery).WithArgs("log_items").WillReturnRows(sqlmock.NewRows([]string{"TABLE_ROWS"}).AddRow(99))

	got, err := service.LogGetInfo(op)
	if err != nil {
		t.Fatalf("LogGetInfo() error = %v", err)
	}

	want := []int{3, 4, 88, 99}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LogGetInfo() = %v, want %v", got, want)
	}
}

func TestLogGetInfoUsesPostgreSQLTableRows(t *testing.T) {
	op, mock := newLogInfoPostgreSQLTestOperator(t)
	maxLogIDQuery := regexp.QuoteMeta(`SELECT MAX(id) FROM "logs"`)
	maxItemIDQuery := regexp.QuoteMeta(`SELECT MAX(id) FROM "log_items"`)
	tableRowsQuery := regexp.QuoteMeta("SELECT reltuples::BIGINT FROM pg_class WHERE oid = $1::regclass")

	mock.ExpectQuery(maxLogIDQuery).WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(3))
	mock.ExpectQuery(tableRowsQuery).WithArgs("logs").WillReturnRows(sqlmock.NewRows([]string{"reltuples"}).AddRow(88))
	mock.ExpectQuery(maxItemIDQuery).WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(4))
	mock.ExpectQuery(tableRowsQuery).WithArgs("log_items").WillReturnRows(sqlmock.NewRows([]string{"reltuples"}).AddRow(99))

	got, err := service.LogGetInfo(op)
	if err != nil {
		t.Fatalf("LogGetInfo() error = %v", err)
	}

	want := []int{3, 4, 88, 99}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LogGetInfo() = %v, want %v", got, want)
	}
}

func TestLogGetInfoFallsBackToMaxIDWhenMetadataUnavailable(t *testing.T) {
	for _, dbType := range []string{constant.MYSQL, constant.POSTGRESQL, "pgsql"} {
		t.Run(dbType, func(t *testing.T) {
			db := newLogInfoTestDB(t)
			seedLogInfoTestDB(t, db)

			got, err := service.LogGetInfo(&logInfoTestOperator{db: db, dbType: dbType})
			if err != nil {
				t.Fatalf("LogGetInfo() error = %v", err)
			}

			want := []int{3, 4, 3, 4}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("LogGetInfo() = %v, want %v", got, want)
			}
		})
	}
}

func newLogInfoTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.ToSlash(filepath.Join(t.TempDir(), "log-info.db"))
	db, err := openLogInfoTestDB(dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if migrateErr := db.AutoMigrate(&model.LogInfo{}, &model.LogOneItem{}); migrateErr != nil {
		t.Fatalf("migrate log tables: %v", migrateErr)
	}
	sqlDB, dbErr := db.DB()
	if dbErr != nil {
		t.Fatalf("get sql db: %v", dbErr)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	return db
}

func newLogInfoMySQLTestOperator(t *testing.T) (*logInfoTestOperator, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	t.Cleanup(func() {
		if expectErr := mock.ExpectationsWereMet(); expectErr != nil {
			t.Errorf("unmet sql expectations: %v", expectErr)
		}
		_ = sqlDB.Close()
	})

	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open mysql mock db: %v", err)
	}
	return &logInfoTestOperator{db: db, dbType: constant.MYSQL}, mock
}

func newLogInfoPostgreSQLTestOperator(t *testing.T) (*logInfoTestOperator, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	t.Cleanup(func() {
		if expectErr := mock.ExpectationsWereMet(); expectErr != nil {
			t.Errorf("unmet sql expectations: %v", expectErr)
		}
		_ = sqlDB.Close()
	})

	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open postgresql mock db: %v", err)
	}
	return &logInfoTestOperator{db: db, dbType: constant.POSTGRESQL}, mock
}

func seedLogInfoTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	for i := 1; i <= 3; i++ {
		info := &model.LogInfo{
			Name:      fmt.Sprintf("log-%d", i),
			GroupID:   "group",
			CreatedAt: int64(i),
			UpdatedAt: int64(i),
		}
		if err := db.Create(info).Error; err != nil {
			t.Fatalf("create log info %d: %v", i, err)
		}
	}
	for i := 1; i <= 4; i++ {
		item := &model.LogOneItem{
			LogID:    1,
			GroupID:  "group",
			Nickname: fmt.Sprintf("nick-%d", i),
			Time:     int64(i),
			Message:  fmt.Sprintf("message-%d", i),
		}
		if err := db.Create(item).Error; err != nil {
			t.Fatalf("create log item %d: %v", i, err)
		}
	}
	if err := db.Where("id = ?", 2).Delete(&model.LogInfo{}).Error; err != nil {
		t.Fatalf("delete log info gap: %v", err)
	}
	if err := db.Where("id = ?", 2).Delete(&model.LogOneItem{}).Error; err != nil {
		t.Fatalf("delete log item gap: %v", err)
	}
}
