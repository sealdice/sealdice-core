package v161_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	v161 "sealdice-core/migrate/v2/v161"

	"gopkg.in/yaml.v3"
)

func TestV161CopyDiceMastersToNoticeIDs(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "serve.yaml")
	content := []byte(`
diceMasters:
  - QQ:10001
  - UI:1001
  - QQ:10001
  - QQ:30003
noticeIds:
  - QQ:20002
  - UI:1001
  - QQ:30003:disable
commandPrefix:
  - .
`)
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	added, err := v161.V161CopyDiceMastersToNoticeIDs(configPath)
	if err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	if added != 1 {
		t.Fatalf("期望新增 1 个通知目标，实际新增 %d 个", added)
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		NoticeIDs     []string `yaml:"noticeIds"`
		CommandPrefix []string `yaml:"commandPrefix"`
	}
	if err = yaml.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	want := []string{"QQ:20002", "UI:1001", "QQ:30003:disable", "QQ:10001:only=send"}
	if !reflect.DeepEqual(got.NoticeIDs, want) {
		t.Fatalf("通知列表不符，期望 %v，实际 %v", want, got.NoticeIDs)
	}
	if !reflect.DeepEqual(got.CommandPrefix, []string{"."}) {
		t.Fatalf("无关配置不应丢失，实际 commandPrefix=%v", got.CommandPrefix)
	}
}

func TestV161CopyDiceMastersToNoticeIDsMissingConfigIsNoOp(t *testing.T) {
	added, err := v161.V161CopyDiceMastersToNoticeIDs(filepath.Join(t.TempDir(), "serve.yaml"))
	if err != nil {
		t.Fatalf("新安装缺少 serve.yaml 时应直接跳过: %v", err)
	}
	if added != 0 {
		t.Fatalf("缺少配置文件时新增数量应为 0，实际 %d", added)
	}
}
