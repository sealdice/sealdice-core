package v2

import (
	_ "time"

	v120 "sealdice-core/migrate/v2/v120"
	v131 "sealdice-core/migrate/v2/v131"
	v141 "sealdice-core/migrate/v2/v141"
	v144 "sealdice-core/migrate/v2/v144"
	v150 "sealdice-core/migrate/v2/v150"
	v151 "sealdice-core/migrate/v2/v151"
	operator "sealdice-core/utils/dboperator/engine"
	upgrade "sealdice-core/utils/upgrader"
	"sealdice-core/utils/upgrader/store"
)

func InitUpgrader(operator operator.DatabaseOperator) error {
	storer := store.NewJSONStore("upgrade_metadata.json")
	mgr := &upgrade.Manager{Store: storer, Database: operator}
	// V120注册
	mgr.Register(v120.V120Migration)
	mgr.Register(v120.V120LogMessageMigration)
	// V131注册
	mgr.Register(v131.V131ConfigUpdateMigration)
	// V141注册
	mgr.Register(v141.V141ConfigUpdateMigration)
	// v144注册
	mgr.Register(v144.V144RemoveOldHelpDocMigration)
	// v150注册
	mgr.Register(v150.V150UpgradeAttrsMigration)
	mgr.Register(v150.V150FixGroupInfoMigration)
	// v151注册
	mgr.Register(v151.V151GORMCleanMigration)
	err := mgr.ApplyAll()
	if err != nil {
		return err
	}
	return nil
}
