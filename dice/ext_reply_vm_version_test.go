package dice

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestCustomReplyConfigVMVersionDetection(t *testing.T) {
	t.Parallel()

	testDice := &Dice{Logger: zap.NewNop().Sugar()}
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "legacy without marker", body: "enable: true\nitems: []\n", want: ""},
		{name: "marker only", body: ReplyVMVersionV2Marker + "\nenable: true\nitems: []\n", want: ReplyVMVersionV2},
		{name: "field only", body: "vmVersion: v2\nenable: true\nitems: []\n", want: ReplyVMVersionV2},
		{name: "marker with BOM", body: "\ufeff" + ReplyVMVersionV2Marker + "\nenable: true\nitems: []\n", want: ReplyVMVersionV2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "reply.yaml")
			if err := os.WriteFile(path, []byte(tt.body), 0o644); err != nil {
				t.Fatal(err)
			}

			config, err := CustomReplyConfigReadFromPath(testDice, path, "reply.yaml")
			if err != nil {
				t.Fatalf("CustomReplyConfigReadFromPath() error = %v", err)
			}
			if config.VMVersion != tt.want {
				t.Fatalf("VMVersion = %q, want %q", config.VMVersion, tt.want)
			}
		})
	}
}

func TestReplyConfigSavePreservesV2Declaration(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "reply.yaml")
	config := &ReplyConfig{VMVersion: ReplyVMVersionV2, Enable: true, Items: []*ReplyItem{}}
	if err := config.SaveToPath(path); err != nil {
		t.Fatalf("SaveToPath() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.HasPrefix(text, ReplyVMVersionV2Marker+"\n") {
		t.Fatalf("saved V2 file does not start with marker: %q", text)
	}
	if !strings.Contains(text, "vmVersion: v2\n") {
		t.Fatalf("saved V2 file does not contain vmVersion field: %q", text)
	}

	config.VMVersion = ReplyVMVersionV1
	if err := config.SaveToPath(path); err != nil {
		t.Fatalf("SaveToPath() legacy error = %v", err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text = string(data)
	if strings.Contains(text, ReplyVMVersionV2Marker) || strings.Contains(text, "vmVersion:") {
		t.Fatalf("legacy-compatible file unexpectedly contains a VM declaration: %q", text)
	}
}

func TestReplyConfigVMVersionForExecution(t *testing.T) {
	t.Parallel()

	testDice := &Dice{}
	legacy := &ReplyConfig{}
	explicitV2 := &ReplyConfig{VMVersion: ReplyVMVersionV2}

	testDice.Config.VMVersionForReply = ReplyVMVersionV1
	if got := legacy.vmVersionForExecution(testDice); got != ReplyVMVersionV1 {
		t.Fatalf("legacy file with global V1 resolved to %q", got)
	}

	testDice.Config.VMVersionForReply = ReplyVMVersionV2
	if got := legacy.vmVersionForExecution(testDice); got != ReplyVMVersionV2 {
		t.Fatalf("legacy file with global V2 resolved to %q", got)
	}

	testDice.Config.VMVersionForReply = ReplyVMVersionV1
	if got := explicitV2.vmVersionForExecution(testDice); got != ReplyVMVersionV2 {
		t.Fatalf("explicit V2 file resolved to %q", got)
	}
}
