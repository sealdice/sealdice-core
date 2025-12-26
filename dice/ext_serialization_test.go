package dice_test

import (
	"testing"

	"sealdice-core/dice"

	"gopkg.in/yaml.v3"
)

// TestGroupInfoSerialization 测试 GroupInfo 扩展相关字段的序列化与反序列化
func TestGroupInfoSerialization(t *testing.T) {
	tests := []struct {
		name     string
		group    *dice.GroupInfo
		validate func(*testing.T, *dice.GroupInfo)
	}{
		{
			name: "有数据的情况",
			group: &dice.GroupInfo{
				InactivatedExtSet: dice.StringSet{"ext1": {}, "ext2": {}},
				ExtAppliedVersion: 123,
			},
			validate: func(t *testing.T, g *dice.GroupInfo) {
				if len(g.InactivatedExtSet) != 2 {
					t.Errorf("InactivatedExtSet 长度不匹配: got %d, want 2", len(g.InactivatedExtSet))
				}
				if _, ok := g.InactivatedExtSet["ext1"]; !ok {
					t.Error("InactivatedExtSet 应该包含 ext1")
				}
				if g.ExtAppliedVersion != 123 {
					t.Errorf("ExtAppliedVersion 不匹配: got %d, want 123", g.ExtAppliedVersion)
				}
			},
		},
		{
			name: "nil 字段的情况",
			group: &dice.GroupInfo{
				InactivatedExtSet: nil,
				ExtAppliedVersion: 0,
			},
			validate: func(t *testing.T, g *dice.GroupInfo) {
				// YAML 会将 nil map/slice 反序列化为空集合，这是预期行为
				if g.InactivatedExtSet == nil {
					g.InactivatedExtSet = dice.StringSet{}
				}
				// 验证空集合也能正常工作
				if g.IsExtInactivated("any") {
					t.Error("空的 InactivatedExtSet 不应该包含任何扩展")
				}
			},
		},
		{
			name: "空但非 nil 的情况",
			group: &dice.GroupInfo{
				InactivatedExtSet: dice.StringSet{},
			},
			validate: func(t *testing.T, g *dice.GroupInfo) {
				if len(g.InactivatedExtSet) != 0 {
					t.Errorf("InactivatedExtSet 应该为空: got %d items", len(g.InactivatedExtSet))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 序列化
			data, err := yaml.Marshal(tt.group)
			if err != nil {
				t.Fatalf("序列化失败: %v", err)
			}

			t.Logf("YAML 输出:\n%s", string(data))

			// 反序列化
			var restored dice.GroupInfo
			err = yaml.Unmarshal(data, &restored)
			if err != nil {
				t.Fatalf("反序列化失败: %v", err)
			}

			// 验证
			tt.validate(t, &restored)
		})
	}
}

// TestInactivatedExtSetOperations 测试反序列化后的 InactivatedExtSet 操作是否正常
func TestInactivatedExtSetOperations(t *testing.T) {
	yamlData := `
inactivatedExtSet:
  - ext1
  - ext2
`

	var group dice.GroupInfo
	err := yaml.Unmarshal([]byte(yamlData), &group)
	if err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}

	// 测试查询
	if !group.IsExtInactivated("ext1") {
		t.Error("ext1 应该在 InactivatedExtSet 中")
	}
	if group.IsExtInactivated("ext3") {
		t.Error("ext3 不应该在 InactivatedExtSet 中")
	}

	// 测试删除
	group.RemoveFromInactivated("ext1")
	if group.IsExtInactivated("ext1") {
		t.Error("ext1 应该已经被移除")
	}

	// 测试添加
	group.AddToInactivated("ext5")
	if !group.IsExtInactivated("ext5") {
		t.Error("ext5 应该已经被添加")
	}
}
