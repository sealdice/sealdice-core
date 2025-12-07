//lint:file-ignore testpackage Tests need access to internal helpers and types
package dice //nolint:testpackage // tests rely on unexported helpers

import (
	"strings"
	"testing"
)

// TestExtActivateCompanionDetection 测试扩展激活时伴随扩展的检测逻辑
// 这个测试验证了 builtin_commands.go 中修复的 bug：
// 当用户明确激活的扩展恰好也是某个扩展的伴随扩展时，不应该被标记为"伴随扩展"
func TestExtActivateCompanionDetection(t *testing.T) {
	tests := []struct {
		name              string
		extensions        []*ExtInfo
		userActivateNames []string // 模拟用户输入的扩展名
		expectedCompanion []string // 期望被识别为伴随扩展的名称
	}{
		{
			name: "单个扩展有伴随扩展",
			extensions: []*ExtInfo{
				{Name: "ext1"},
				{Name: "companion1", ActiveWith: []string{"ext1"}},
			},
			userActivateNames: []string{"ext1"},
			expectedCompanion: []string{"companion1"},
		},
		{
			name: "多个扩展各有不同的伴随扩展",
			extensions: []*ExtInfo{
				{Name: "ext1"},
				{Name: "ext2"},
				{Name: "companion1", ActiveWith: []string{"ext1"}},
				{Name: "companion2", ActiveWith: []string{"ext2"}},
			},
			userActivateNames: []string{"ext1", "ext2"},
			expectedCompanion: []string{"companion1", "companion2"},
		},
		{
			name: "多个扩展共享同一个伴随扩展",
			extensions: []*ExtInfo{
				{Name: "ext1"},
				{Name: "ext2"},
				{Name: "shared", ActiveWith: []string{"ext1", "ext2"}},
			},
			userActivateNames: []string{"ext1", "ext2"},
			expectedCompanion: []string{"shared"},
		},
		{
			name: "用户明确激活的扩展恰好是伴随扩展 - 这是修复的 bug",
			extensions: []*ExtInfo{
				{Name: "ext1"},
				{Name: "ext2", ActiveWith: []string{"ext1"}},
			},
			userActivateNames: []string{"ext1", "ext2"}, // 用户同时激活 ext1 和 ext2
			expectedCompanion: []string{},               // ext2 不应该被识别为伴随扩展
		},
		{
			name: "复杂场景：部分是用户激活，部分是伴随激活",
			extensions: []*ExtInfo{
				{Name: "ext1"},
				{Name: "ext2"},
				{Name: "companion1", ActiveWith: []string{"ext1"}},
				{Name: "companion2", ActiveWith: []string{"ext2"}},
			},
			userActivateNames: []string{"ext1", "ext2"},             // 用户明确激活 ext1 和 ext2
			expectedCompanion: []string{"companion1", "companion2"}, // 只有真正的伴随扩展
		},
		{
			name: "大小写不敏感测试",
			extensions: []*ExtInfo{
				{Name: "Ext1"},
				{Name: "Ext2", ActiveWith: []string{"Ext1"}},
			},
			userActivateNames: []string{"ext1", "EXT2"}, // 用户输入不同大小写
			expectedCompanion: []string{},               // Ext2 不应该被识别为伴随扩展
		},
		{
			name: "使用别名激活且没有伴随扩展",
			extensions: []*ExtInfo{
				{Name: "MyExt", Aliases: []string{"alias1"}},
			},
			userActivateNames: []string{"alias1"},
			expectedCompanion: []string{},
		},
		{
			name: "使用别名激活且存在伴随扩展",
			extensions: []*ExtInfo{
				{Name: "MyExt", Aliases: []string{"alias1"}},
				{Name: "Companion", ActiveWith: []string{"MyExt"}},
			},
			userActivateNames: []string{"alias1"},
			expectedCompanion: []string{"Companion"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试环境
			dice := newTestDice(tt.extensions)
			dice.rebuildActiveWithGraph() // 构建伴随激活图
			group := newTestGroupInfo()

			// 记录激活前的扩展列表（模拟 builtin_commands.go 中的逻辑）
			beforeActivate := make(map[string]bool)
			for _, ext := range group.ActivatedExtList {
				beforeActivate[ext.Name] = true
			}
			userActivatedNames := make(map[string]bool)

			// 激活用户指定的扩展（模拟循环激活）
			var activatedExtNames []string
			for _, extName := range tt.userActivateNames {
				extName = strings.ToLower(extName)
				if ext := dice.ExtFind(extName, false); ext != nil {
					activatedExtNames = append(activatedExtNames, ext.Name)
					userActivatedNames[ext.Name] = true
					group.RemoveFromInactivated(ext.Name)
					group.ExtActive(ext)
				}
			}

			// 检查新激活的伴随扩展（模拟 builtin_commands.go 中的检测逻辑）
			var companionExtNames []string
			for _, ext := range group.ActivatedExtList {
				if !beforeActivate[ext.Name] && !userActivatedNames[ext.Name] {
					companionExtNames = append(companionExtNames, ext.Name)
				}
			}

			// 验证结果
			if len(companionExtNames) != len(tt.expectedCompanion) {
				t.Errorf("伴随扩展数量不匹配: 期望 %d 个 %v, 实际 %d 个 %v",
					len(tt.expectedCompanion), tt.expectedCompanion,
					len(companionExtNames), companionExtNames)
				return
			}

			// 验证所有期望的伴随扩展都被检测到
			expectedSet := make(map[string]bool)
			for _, name := range tt.expectedCompanion {
				expectedSet[name] = true
			}

			for _, name := range companionExtNames {
				if !expectedSet[name] {
					t.Errorf("意外的伴随扩展: %s (期望: %v, 实际: %v)",
						name, tt.expectedCompanion, companionExtNames)
				}
				delete(expectedSet, name)
			}

			// 检查是否有遗漏的伴随扩展
			for name := range expectedSet {
				t.Errorf("遗漏的伴随扩展: %s (期望: %v, 实际: %v)",
					name, tt.expectedCompanion, companionExtNames)
			}

			t.Logf("✓ 用户激活: %v, 检测到的伴随扩展: %v",
				activatedExtNames, companionExtNames)
		})
	}
}

// TestExtActivateChainedCompanions 测试链式伴随扩展的检测
// 例如：ext1 -> ext2 -> companion，用户只激活 ext1，ext2 和 companion 都是伴随扩展
func TestExtActivateChainedCompanions(t *testing.T) {
	tests := []struct {
		name              string
		extensions        []*ExtInfo
		userActivateNames []string
		expectedCompanion []string
	}{
		{
			name: "链式伴随扩展：ext1 -> ext2 -> companion",
			extensions: []*ExtInfo{
				{Name: "ext1"},
				{Name: "ext2", ActiveWith: []string{"ext1"}},
				{Name: "companion", ActiveWith: []string{"ext2"}},
			},
			userActivateNames: []string{"ext1"},
			expectedCompanion: []string{"ext2", "companion"},
		},
		{
			name: "链式伴随扩展，但用户明确激活了中间节点",
			extensions: []*ExtInfo{
				{Name: "ext1"},
				{Name: "ext2", ActiveWith: []string{"ext1"}},
				{Name: "companion", ActiveWith: []string{"ext2"}},
			},
			userActivateNames: []string{"ext1", "ext2"}, // 用户明确激活 ext1 和 ext2
			expectedCompanion: []string{"companion"},    // 只有 companion 是伴随扩展
		},
		{
			name: "链式伴随扩展，用户激活所有节点",
			extensions: []*ExtInfo{
				{Name: "ext1"},
				{Name: "ext2", ActiveWith: []string{"ext1"}},
				{Name: "companion", ActiveWith: []string{"ext2"}},
			},
			userActivateNames: []string{"ext1", "ext2", "companion"}, // 用户激活所有
			expectedCompanion: []string{},                            // 没有伴随扩展
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dice := newTestDice(tt.extensions)
			dice.rebuildActiveWithGraph() // 构建伴随激活图
			group := newTestGroupInfo()

			beforeActivate := make(map[string]bool)
			for _, ext := range group.ActivatedExtList {
				beforeActivate[ext.Name] = true
			}
			userActivatedNames := make(map[string]bool)

			for _, extName := range tt.userActivateNames {
				extName = strings.ToLower(extName)
				if ext := dice.ExtFind(extName, false); ext != nil {
					userActivatedNames[ext.Name] = true
					group.RemoveFromInactivated(ext.Name)
					group.ExtActive(ext)
				}
			}

			var companionExtNames []string
			for _, ext := range group.ActivatedExtList {
				if !beforeActivate[ext.Name] && !userActivatedNames[ext.Name] {
					companionExtNames = append(companionExtNames, ext.Name)
				}
			}

			// 验证伴随扩展数量
			if len(companionExtNames) != len(tt.expectedCompanion) {
				t.Errorf("伴随扩展数量不匹配: 期望 %v, 实际 %v",
					tt.expectedCompanion, companionExtNames)
				return
			}

			// 验证所有期望的伴随扩展都被检测到
			expectedSet := make(map[string]bool)
			for _, name := range tt.expectedCompanion {
				expectedSet[name] = true
			}

			for _, name := range companionExtNames {
				if !expectedSet[name] {
					t.Errorf("意外的伴随扩展: %s", name)
				}
			}

			t.Logf("✓ 激活链: %v -> 伴随扩展: %v",
				tt.userActivateNames, companionExtNames)
		})
	}
}
