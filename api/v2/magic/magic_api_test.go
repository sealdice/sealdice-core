package magic

import (
	"path/filepath"
	"testing"

	"sealdice-core/model/common/response"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInspectSourceRecognizesCurrentSQLiteSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "current.db")
	db := openSQLiteForTest(t, path)
	createTables(t, db,
		`create table attrs (id text primary key, data blob, attrs_type text, binding_sheet_id text, name text, owner_id text, sheet_type text, is_hidden integer, created_at integer, updated_at integer)`,
		`create table group_info (id text primary key, created_at integer, updated_at integer, data blob)`,
		`create table group_player_info (id integer primary key autoincrement, name text, user_id text, last_command_time integer, auto_set_name_template text, dice_side_num integer, created_at integer, updated_at integer, group_id text)`,
		`create table ban_info (id text primary key, ban_updated_at integer, updated_at integer, data blob)`,
		`create table endpoint_info (user_id text primary key, cmd_num integer, cmd_last_time integer, online_time integer, updated_at integer)`,
		`create table logs (id integer primary key autoincrement, name text, group_id text, created_at integer, updated_at integer, size integer, extra text, upload_url text, upload_time integer)`,
		`create table log_items (id integer primary key autoincrement, log_id integer, group_id text, nickname text, im_userid text, time integer, message text, is_dice integer, command_id integer, command_info text, raw_msg_id text, user_uniform_id text, removed integer, parent_id integer)`,
		`create table censor_log (id integer primary key autoincrement, msg_type text, user_id text, group_id text, content text, highest_level integer, created_at integer, sensitive_words text, clear_mark integer)`,
	)

	svc := NewService()
	resp, err := svc.InspectSource(t.Context(), &InspectReq{
		Body: struct {
			Source SourceProfile `json:"source"`
		}{
			Source: SourceProfile{
				Kind:       SourceKindSQLite,
				SQLitePath: path,
			},
		},
	})
	if err != nil {
		t.Fatalf("InspectSource() error = %v", err)
	}

	assertInspectResponse(t, resp, SourceStageCurrent, true, false)
}

func TestInspectSourceRecognizesLegacyV146SQLiteSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy-146.db")
	db := openSQLiteForTest(t, path)
	createTables(t, db,
		`create table attrs_user (id text primary key, updated_at integer, data blob)`,
		`create table attrs_group (id text primary key, updated_at integer, data blob)`,
		`create table attrs_group_user (id text primary key, updated_at integer, data blob)`,
		`create table group_info (id text primary key, created_at integer, updated_at integer, data blob)`,
	)

	svc := NewService()
	resp, err := svc.InspectSource(t.Context(), &InspectReq{
		Body: struct {
			Source SourceProfile `json:"source"`
		}{
			Source: SourceProfile{
				Kind:       SourceKindSQLite,
				SQLitePath: path,
			},
		},
	})
	if err != nil {
		t.Fatalf("InspectSource() error = %v", err)
	}

	assertInspectResponse(t, resp, SourceStageLegacyV146, false, true)
}

func assertInspectResponse(t *testing.T, resp *response.ItemResponse[InspectResult], stage SourceStage, direct bool, needsV150 bool) {
	t.Helper()

	if resp.Body.Item.Stage != stage {
		t.Fatalf("Stage = %q, want %q", resp.Body.Item.Stage, stage)
	}
	if resp.Body.Item.CanDirectMigrate != direct {
		t.Fatalf("CanDirectMigrate = %v, want %v", resp.Body.Item.CanDirectMigrate, direct)
	}
	if resp.Body.Item.RequiresV150Upgrade != needsV150 {
		t.Fatalf("RequiresV150Upgrade = %v, want %v", resp.Body.Item.RequiresV150Upgrade, needsV150)
	}
}

func openSQLiteForTest(t *testing.T, path string) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func createTables(t *testing.T, db *gorm.DB, stmts ...string) {
	t.Helper()

	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("exec %q: %v", stmt, err)
		}
	}
}
