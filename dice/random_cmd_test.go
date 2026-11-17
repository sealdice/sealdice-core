package dice //nolint:testpackage

import (
	"strings"
	"testing"
	"time"
)

func TestGMEnableConfirmCode(t *testing.T) {
	const userID = "gm-confirm-user"

	code := issueGMEnableConfirmCode(userID)
	if len(code) != 4 {
		t.Fatalf("confirmation code length = %d, want 4", len(code))
	}
	if !consumeGMEnableConfirmCode(userID, code) {
		t.Fatal("expected confirmation code to be accepted")
	}
	if consumeGMEnableConfirmCode(userID, code) {
		t.Fatal("expected confirmation code to be single-use")
	}
}

func TestGMEnableConfirmCodeExpires(t *testing.T) {
	const userID = "gm-confirm-expired-user"

	code := issueGMEnableConfirmCode(userID)
	gmEnableConfirmCodes.Store(userID, &gmEnableConfirmState{
		Code:      code,
		ExpiresAt: time.Now().Add(-time.Second).Unix(),
	})
	if consumeGMEnableConfirmCode(userID, code) {
		t.Fatal("expected expired confirmation code to be rejected")
	}
}

func TestRandomModeHelpMentionsGMMemory(t *testing.T) {
	text := formatDiceRandomModeHelpText()
	for _, want := range []string{"国密源默认禁用", "32MB", ".randalgo set gm off", "NIST 模式会混入一段国密种子"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected %q in help text, got %q", want, text)
		}
	}
}
