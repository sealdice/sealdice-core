package v161

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	operator "sealdice-core/utils/dboperator/engine"
	upgrade "sealdice-core/utils/upgrader"
)

const defaultServeConfigPath = "data/default/serve.yaml"

// V161CopyDiceMastersToNoticeIDs 将骰主 ID 一次性补入通知列表。
//
// 已有通知目标会被保留；完全相同的 ID 不会重复添加。此函数本身保持可重复执行，
// “仅执行一次”由升级框架的迁移记录保证。
func V161CopyDiceMastersToNoticeIDs(configPath string) (int, error) {
	content, err := os.ReadFile(filepath.Clean(configPath))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// 新安装此时还没有 serve.yaml，由运行期默认配置负责初始化通知列表。
			return 0, nil
		}
		return 0, err
	}

	var config struct {
		DiceMasters []string `yaml:"diceMasters"`
		NoticeIDs   []string `yaml:"noticeIds"`
	}
	if err = yaml.Unmarshal(content, &config); err != nil {
		return 0, err
	}

	existing := make(map[string]struct{}, len(config.NoticeIDs))
	for _, target := range config.NoticeIDs {
		if id := noticeTargetID(target); id != "" {
			existing[id] = struct{}{}
		}
	}

	added := 0
	for _, rawID := range config.DiceMasters {
		id := strings.TrimSpace(rawID)
		if id == "" {
			continue
		}
		if _, ok := existing[id]; ok {
			continue
		}
		config.NoticeIDs = append(config.NoticeIDs, id+":only=send")
		existing[id] = struct{}{}
		added++
	}
	if added == 0 {
		return 0, nil
	}

	var data map[string]any
	if err = yaml.Unmarshal(content, &data); err != nil {
		return 0, err
	}
	data["noticeIds"] = config.NoticeIDs

	modified, err := yaml.Marshal(data)
	if err != nil {
		return 0, err
	}
	if err = os.WriteFile(filepath.Clean(configPath), modified, 0o644); err != nil {
		return 0, err
	}
	return added, nil
}

// noticeTargetID 提取兼容旧格式通知目标中的实际 ID。
// 元数据只从末尾识别，避免截断 OpenQQ 等本身含有多个冒号的 ID。
func noticeTargetID(raw string) string {
	parts := strings.Split(strings.TrimSpace(raw), ":")
	end := len(parts)
	for end > 1 {
		suffix := strings.TrimSpace(parts[end-1])
		if suffix == "disable" || strings.HasPrefix(suffix, "only=") {
			end--
			continue
		}
		break
	}
	return strings.TrimSpace(strings.Join(parts[:end], ":"))
}

var V161NoticeIDsMigration = upgrade.Upgrade{
	ID: "011_V161NoticeIDsMigration",
	Description: `
# 升级说明
将骰主 ID 一次性补入通知列表，并仅开启 send 分类，避免升级后骰主收不到代发通知。
`,
	Apply: func(logf func(string), _ operator.DatabaseOperator) error {
		logf("[INFO] V161 通知列表迁移开始")
		added, err := V161CopyDiceMastersToNoticeIDs(defaultServeConfigPath)
		if err != nil {
			return fmt.Errorf("迁移骰主 ID 到通知列表失败: %w", err)
		}
		logf(fmt.Sprintf("[INFO] V161 通知列表迁移完成，新增 %d 个通知目标", added))
		return nil
	},
}
