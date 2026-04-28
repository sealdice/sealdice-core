package pgsql

import (
	"testing"

	"sealdice-core/utils/constant"
)

func TestPGSQLEngineType(t *testing.T) {
	if got := (&PGSQLEngine{}).Type(); got != constant.POSTGRESQL {
		t.Fatalf("PGSQLEngine.Type() = %q, want %q", got, constant.POSTGRESQL)
	}
}
