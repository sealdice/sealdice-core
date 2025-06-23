package upgrade

import (
	"time"

	"sealdice-core/utils/dboperator/engine"
)

type Upgrade struct {
	ID          string
	Description string
	// TODO: 有更好的想法吗，需要啥就从这里传是不是太抽象了
	// 或许可以在这放一个logger，这个logger会在使用时注入，这样会好看些
	Apply func(logf func(string), operator engine.DatabaseOperator) error
}

type UpgradeRecord struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	Logs      []string  `json:"logs"`
}

type Store interface {
	IsApplied(id string) (bool, error)
	SaveRecord(record UpgradeRecord) error
	LoadRecords() ([]UpgradeRecord, error)
}
