package dboperator_test

import (
	"context"
	"testing"

	"sealdice-core/utils/constant"
	"sealdice-core/utils/dboperator"
)

func TestNewDatabaseOperatorMissingExternalDSNDoesNotPanic(t *testing.T) {
	for _, databaseType := range []string{constant.MYSQL, constant.POSTGRESQL} {
		t.Run(databaseType, func(t *testing.T) {
			t.Setenv("DB_TYPE", databaseType)
			t.Setenv("DB_DSN", "")
			if _, err := dboperator.NewDatabaseOperator(context.Background()); err == nil {
				t.Fatal("NewDatabaseOperator() unexpectedly succeeded")
			}
		})
	}
}
