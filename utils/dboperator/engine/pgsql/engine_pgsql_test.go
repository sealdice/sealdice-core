package pgsql_test

import (
	"testing"

	"sealdice-core/utils/constant"
	"sealdice-core/utils/dboperator/engine/pgsql"
)

func TestPGSQLEngineType(t *testing.T) {
	if got := (&pgsql.PGSQLEngine{}).Type(); got != constant.POSTGRESQL {
		t.Fatalf("PGSQLEngine.Type() = %q, want %q", got, constant.POSTGRESQL)
	}
}
