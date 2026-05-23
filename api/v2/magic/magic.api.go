package magic

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"sealdice-core/model/common/response"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) RegisterRoutes(grp *huma.Group) {
	huma.Post(grp, "/source/inspect", s.InspectSource, func(o *huma.Operation) {
		o.Description = "检查迁移源数据库结构"
		o.Summary = "检查迁移源数据库结构"
	})
}

func (s *Service) InspectSource(_ context.Context, req *InspectReq) (*response.ItemResponse[InspectResult], error) {
	result, err := inspectSource(req.Body.Source)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	return response.NewItemResponse(result), nil
}

func inspectSource(profile SourceProfile) (InspectResult, error) {
	switch profile.Kind {
	case SourceKindSQLite:
		return inspectSQLite(profile.SQLitePath)
	case SourceKindMySQL, SourceKindPostgreSQL:
		return InspectResult{}, errors.New("暂未实现服务端数据库源检查")
	default:
		return InspectResult{}, fmt.Errorf("unsupported source kind: %s", profile.Kind)
	}
}

func inspectSQLite(path string) (InspectResult, error) {
	if strings.TrimSpace(path) == "" {
		return InspectResult{}, errors.New("sqlite path is required")
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return InspectResult{}, err
	}

	fingerprint, err := loadSQLiteFingerprint(db)
	if err != nil {
		return InspectResult{}, err
	}

	result := InspectResult{
		Kind:        SourceKindSQLite,
		Fingerprint: fingerprint,
		Messages:    []string{},
	}

	switch {
	case isLegacyV146(fingerprint):
		result.Stage = SourceStageLegacyV146
		result.RequiresV150Upgrade = true
		result.Messages = append(result.Messages, "检测到 1.4.6 attrs 旧结构，必须先升级到 1.5.0 SQLite 结构。")
	case isCurrentSchema(fingerprint):
		result.Stage = SourceStageCurrent
		result.CanDirectMigrate = true
		result.Messages = append(result.Messages, "源结构已是当前标准结构，可直接进入迁移流程。")
	default:
		result.Stage = SourceStageUnknownLegacy
		result.Messages = append(result.Messages, "检测到非当前标准结构，但不属于可自动处理的 1.4.6 -> 1.5.0 路径。")
	}

	return result, nil
}

func loadSQLiteFingerprint(db *gorm.DB) (SchemaFingerprint, error) {
	type tableRow struct {
		Name string `gorm:"column:name"`
	}

	var rows []tableRow
	if err := db.Raw("SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name").Scan(&rows).Error; err != nil {
		return SchemaFingerprint{}, err
	}

	tableNames := make([]string, 0, len(rows))
	tableSet := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			continue
		}
		tableNames = append(tableNames, name)
		tableSet[name] = struct{}{}
	}
	sort.Strings(tableNames)

	return SchemaFingerprint{
		HasAttrs:                hasTable(tableSet, "attrs"),
		HasLegacyAttrsUser:      hasTable(tableSet, "attrs_user"),
		HasLegacyAttrsGroup:     hasTable(tableSet, "attrs_group"),
		HasLegacyAttrsGroupUser: hasTable(tableSet, "attrs_group_user"),
		HasGroupInfo:            hasTable(tableSet, "group_info"),
		HasGroupPlayerInfo:      hasTable(tableSet, "group_player_info"),
		HasBanInfo:              hasTable(tableSet, "ban_info"),
		HasEndpointInfo:         hasTable(tableSet, "endpoint_info"),
		HasLogs:                 hasTable(tableSet, "logs"),
		HasLogItems:             hasTable(tableSet, "log_items"),
		HasCensorLog:            hasTable(tableSet, "censor_log"),
		Tables:                  tableNames,
	}, nil
}

func hasTable(tableSet map[string]struct{}, name string) bool {
	_, ok := tableSet[name]
	return ok
}

func isLegacyV146(f SchemaFingerprint) bool {
	return f.HasLegacyAttrsUser && f.HasLegacyAttrsGroup && f.HasLegacyAttrsGroupUser
}

func isCurrentSchema(f SchemaFingerprint) bool {
	return f.HasAttrs &&
		f.HasGroupInfo &&
		f.HasGroupPlayerInfo &&
		f.HasBanInfo &&
		f.HasEndpointInfo &&
		f.HasLogs &&
		f.HasLogItems &&
		f.HasCensorLog &&
		!f.HasLegacyAttrsUser &&
		!f.HasLegacyAttrsGroup &&
		!f.HasLegacyAttrsGroupUser
}
