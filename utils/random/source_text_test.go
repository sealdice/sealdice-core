package random_test

import (
	"strings"
	"testing"

	randcore "sealdice-core/utils/random"
)

func TestModeCommandText_UsesRuntimeNoteFromSource(t *testing.T) {
	owner := randcore.NewEmptyGlobalOwner()
	src, err := randcore.NewSourceForMode(randcore.ModeNIST, nil)
	if err != nil {
		t.Fatalf("NewSourceForMode(nist): %v", err)
	}

	owner.RegisterSource(randcore.ModeNIST, src)
	if _, err := owner.SetActive(randcore.ModeNIST); err != nil {
		t.Fatalf("SetActive(nist): %v", err)
	}

	got := owner.ReportStatusText(randcore.ModeNIST)
	if !strings.Contains(got, "熵补充:") {
		t.Fatalf("expected runtime note in command text, got %q", got)
	}
}
