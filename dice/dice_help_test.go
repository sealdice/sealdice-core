package dice //nolint:testpackage

import (
	"errors"
	"testing"

	"sealdice-core/dice/docengine"
)

func TestHelpManagerCloseWithoutSearchEngineDoesNotPanic(t *testing.T) {
	t.Parallel()

	var nilManager *HelpManager
	nilManager.Close()

	manager := &HelpManager{}
	manager.Close()
}

func TestHelpManagerGetHelpItemPageWithoutSearchEngineDoesNotPanic(t *testing.T) {
	t.Parallel()

	total, items := (&HelpManager{}).GetHelpItemPage(1, 10, "", "", "", "")
	if total != 0 {
		t.Fatalf("total = %d, want 0", total)
	}
	if len(items) != 0 {
		t.Fatalf("items = %#v, want empty", items)
	}
}

func TestHelpManagerSearchWithoutSearchEngineReturnsUnavailable(t *testing.T) {
	t.Parallel()

	ctx := &MsgContext{Group: &GroupInfo{}}
	_, _, _, _, err := (&HelpManager{}).Search(ctx, "test", false, 10, 1, "")
	if !errors.Is(err, docengine.ErrSearchEngineUnavailable) {
		t.Fatalf("Search() error = %v, want %v", err, docengine.ErrSearchEngineUnavailable)
	}
}
