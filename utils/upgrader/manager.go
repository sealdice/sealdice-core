// upgrade/manager.go
package upgrade

import (
	"fmt"
	"sort"
	"time"

	"sealdice-core/utils/dboperator/engine"
)

type Manager struct {
	Upgrades []Upgrade
	Store    Store
	Database engine.DatabaseOperator
}

func (m *Manager) Register(up Upgrade) {
	m.Upgrades = append(m.Upgrades, up)
}

func (m *Manager) ApplyAll() error {
	sort.Slice(m.Upgrades, func(i, j int) bool {
		return m.Upgrades[i].ID < m.Upgrades[j].ID
	})

	for _, up := range m.Upgrades {
		applied, err := m.Store.IsApplied(up.ID)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		logs := []string{}
		logf := func(msg string) {
			logs = append(logs, msg)
		}

		start := time.Now()
		err = up.Apply(logf, m.Database)

		rec := UpgradeRecord{
			ID:        up.ID,
			Timestamp: start,
			Success:   err == nil,
			Message:   "成功",
			Logs:      logs,
		}
		if err != nil {
			rec.Message = err.Error()
		}

		if err2 := m.Store.SaveRecord(rec); err2 != nil {
			return fmt.Errorf("保存升级记录失败: %w", err)
		}

		if err != nil {
			return fmt.Errorf("因无法忽略的错误，升级 %s 失败: %w，请联系海豹开发者", up.ID, err)
		}
	}
	return nil
}
