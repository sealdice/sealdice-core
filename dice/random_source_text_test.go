package dice //nolint:testpackage

import (
	"strings"
	"testing"
)

func TestFormatDiceRandomModeCommandText_UsesRuntimeNoteFromSource(t *testing.T) {
	src := &readerDiceSource{runtimeNote: "系统熵路径。"}

	got := formatDiceRandomModeCommandText(DiceRandomModeNIST, src)
	if !strings.Contains(got, "熵补充: 系统熵路径。") {
		t.Fatalf("expected runtime note in command text, got %q", got)
	}
}
