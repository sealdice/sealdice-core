//lint:file-ignore testpackage Tests need access to internal helpers and types
package dice

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// TestStringSetMarshalYAML 验证序列化输出为排序后的切片，并处理 nil
func TestStringSetMarshalYAML(t *testing.T) {
	t.Run("nil set becomes empty slice", func(t *testing.T) {
		var s StringSet
		out, err := s.MarshalYAML()
		if err != nil {
			t.Fatalf("MarshalYAML failed: %v", err)
		}
		list, ok := out.([]string)
		if !ok {
			t.Fatalf("MarshalYAML should return []string, got %T", out)
		}
		if len(list) != 0 {
			t.Fatalf("expected empty slice, got %v", list)
		}
	})

	t.Run("non-nil set is sorted", func(t *testing.T) {
		s := StringSet{"beta": {}, "alpha": {}, "gamma": {}}
		out, err := s.MarshalYAML()
		if err != nil {
			t.Fatalf("MarshalYAML failed: %v", err)
		}
		list, ok := out.([]string)
		if !ok {
			t.Fatalf("MarshalYAML should return []string, got %T", out)
		}
		expected := []string{"alpha", "beta", "gamma"}
		if !reflect.DeepEqual(list, expected) {
			t.Fatalf("expected %v, got %v", expected, list)
		}
	})
}

// TestStringSetUnmarshalYAML 验证反序列化能正确构建集合
func TestStringSetUnmarshalYAML(t *testing.T) {
	var s StringSet
	data := "- alpha\n- beta\n"
	if err := yaml.Unmarshal([]byte(data), &s); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if len(s) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(s))
	}
	if _, ok := s["alpha"]; !ok {
		t.Fatalf("alpha should exist in set")
	}
	if _, ok := s["beta"]; !ok {
		t.Fatalf("beta should exist in set")
	}
}

// TestRegisterExtensionUpdatesRegistry 验证注册流程会更新 registry/版本号
func TestRegisterExtensionUpdatesRegistry(t *testing.T) {
	d := &Dice{
		Logger: zap.NewNop().Sugar(),
		BaseConfig: BaseConfig{
			DataDir: t.TempDir(),
		},
	}

	ext := &ExtInfo{
		Name:    "Sample",
		Aliases: []string{"AliasOne"},
	}
	d.RegisterExtension(ext)

	if ext.dice != d {
		t.Fatalf("extension dice pointer not set")
	}
	if len(d.ExtList) != 1 {
		t.Fatalf("expected ExtList length 1, got %d", len(d.ExtList))
	}
	if d.ExtRegistryVersion != 1 {
		t.Fatalf("expected registry version 1, got %d", d.ExtRegistryVersion)
	}
	if d.ActiveWithGraph != nil {
		t.Fatalf("ActiveWithGraph should be nil after registration to force rebuild")
	}
	if got, ok := d.ExtRegistry.Load("AliasOne"); !ok || got != ext {
		t.Fatalf("alias lookup failed: %v, %v", got, ok)
	}
	if got, ok := d.ExtRegistry.Load("aliasone"); !ok || got != ext {
		t.Fatalf("case-insensitive alias lookup failed")
	}
}

// TestRegisterExtensionDetectsCollision 验证名称/别名冲突会 panic
func TestRegisterExtensionDetectsCollision(t *testing.T) {
	d := &Dice{Logger: zap.NewNop().Sugar()}
	d.RegisterExtension(&ExtInfo{Name: "main"})

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when registering duplicate name/alias, got none")
		}
	}()

	// 再注册一个包含 main 别名的扩展应 panic
	d.RegisterExtension(&ExtInfo{
		Name:    "another",
		Aliases: []string{"main"},
	})
}

// TestDiceDataDirHelpers 验证与数据目录相关的辅助函数
func TestDiceDataDirHelpers(t *testing.T) {
	temp := t.TempDir()
	d := &Dice{
		BaseConfig: BaseConfig{
			DataDir: temp,
		},
	}

	dir := d.GetExtDataDir("demo")
	expectedDir := filepath.Join(temp, "extensions", "demo")
	if filepath.ToSlash(dir) != filepath.ToSlash(expectedDir) {
		t.Fatalf("expected dir %s, got %s", expectedDir, dir)
	}
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		t.Fatalf("expected directory to exist: err=%v, info=%v", err, fi)
	}

	cfgPath := d.GetExtConfigFilePath("demo", "conf.yaml")
	expectedCfg := filepath.Join(expectedDir, "conf.yaml")
	if filepath.ToSlash(cfgPath) != filepath.ToSlash(expectedCfg) {
		t.Fatalf("expected config path %s, got %s", expectedCfg, cfgPath)
	}

	dataPath := d.GetDiceDataPath("custom.dat")
	expectedData := filepath.Join(temp, "custom.dat")
	if filepath.ToSlash(dataPath) != filepath.ToSlash(expectedData) {
		t.Fatalf("expected data path %s, got %s", expectedData, dataPath)
	}
}

// TestExtActiveBySnapshotOrderReload 验证复用快照重载单个扩展时会恢复原先的激活状态
func TestExtActiveBySnapshotOrderReload(t *testing.T) {
	dice := newTestDice([]*ExtInfo{
		{Name: "ext1"},
		{Name: "ext2"},
	})
	group := newTestGroupInfo()

	// 初始激活顺序：先 ext1，再 ext2（ext2 优先级更高）
	group.ExtActive(dice.ExtFind("ext1", false))
	group.ExtActive(dice.ExtFind("ext2", false))
	if !stringSliceEqual(extListToNames(group.ActivatedExtList), []string{"ext2", "ext1"}) {
		t.Fatalf("unexpected initial order: %v", extListToNames(group.ActivatedExtList))
	}

	// 保存快照并清空激活列表，模拟扩展被卸载但快照仍保留
	snapshot := append([]string(nil), group.ExtActiveListSnapshot...)
	group.ActivatedExtList = nil
	group.ExtActiveListSnapshot = snapshot

	// 仅重新加载 ext1，Expect 它会根据快照被恢复
	group.ExtActiveBySnapshotOrder(dice.ExtFind("ext1", false), false)

	if len(group.ActivatedExtList) != 1 || group.ActivatedExtList[0].Name != "ext1" {
		t.Fatalf("expected ext1 to be reactivated via snapshot, got %v",
			extListToNames(group.ActivatedExtList))
	}

	// 快照也应更新为当前激活顺序
	if !stringSliceEqual(group.ExtActiveListSnapshot, []string{"ext1"}) {
		t.Fatalf("snapshot should update after reload, got %v", group.ExtActiveListSnapshot)
	}
}

// 创建测试用的轻量化 Dice 结构
func newTestDice(exts []*ExtInfo) *Dice {
	d := &Dice{
		ExtList:         exts,
		ExtRegistry:     new(SyncMap[string, *ExtInfo]),
		ActiveWithGraph: nil, // 会自动构建
		Logger:          zap.NewNop().Sugar(),
	}

	// 注册扩展到 registry
	for _, ext := range exts {
		ext.dice = d
		d.ExtRegistry.Store(ext.Name, ext)
		for _, alias := range ext.Aliases {
			d.ExtRegistry.Store(alias, ext)
		}
	}

	return d
}

// 创建测试用的 GroupInfo
func newTestGroupInfo() *GroupInfo {
	return &GroupInfo{
		ActivatedExtList:  []*ExtInfo{},
		InactivatedExtSet: make(StringSet),
	}
}

// TestBuildActiveWithGraph 测试反向图构建
func TestBuildActiveWithGraph(t *testing.T) {
	tests := []struct {
		name     string
		exts     []*ExtInfo
		expected map[string][]string // target -> followers
	}{
		{
			name: "单个伴随关系",
			exts: []*ExtInfo{
				{Name: "main"},
				{Name: "companion", ActiveWith: []string{"main"}},
			},
			expected: map[string][]string{
				"main": {"companion"},
			},
		},
		{
			name: "多个伴随同一个主扩展",
			exts: []*ExtInfo{
				{Name: "main"},
				{Name: "companion1", ActiveWith: []string{"main"}},
				{Name: "companion2", ActiveWith: []string{"main"}},
			},
			expected: map[string][]string{
				"main": {"companion1", "companion2"},
			},
		},
		{
			name: "链式伴随",
			exts: []*ExtInfo{
				{Name: "main"},
				{Name: "level1", ActiveWith: []string{"main"}},
				{Name: "level2", ActiveWith: []string{"level1"}},
			},
			expected: map[string][]string{
				"main":   {"level1"},
				"level1": {"level2"},
			},
		},
		{
			name: "一个扩展跟随多个",
			exts: []*ExtInfo{
				{Name: "main1"},
				{Name: "main2"},
				{Name: "companion", ActiveWith: []string{"main1", "main2"}},
			},
			expected: map[string][]string{
				"main1": {"companion"},
				"main2": {"companion"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := BuildActiveWithGraph(tt.exts)

			// 验证每个预期的关系
			for target, expectedFollowers := range tt.expected {
				followers, ok := graph.Load(target)
				if !ok {
					t.Errorf("目标扩展 %s 在图中不存在", target)
					continue
				}

				if len(followers) != len(expectedFollowers) {
					t.Errorf("扩展 %s 的伴随扩展数量不匹配: got %d, want %d",
						target, len(followers), len(expectedFollowers))
					continue
				}

				// 检查每个预期的 follower 是否存在
				followerSet := make(map[string]bool)
				for _, f := range followers {
					followerSet[f] = true
				}

				for _, expected := range expectedFollowers {
					if !followerSet[expected] {
						t.Errorf("扩展 %s 的伴随扩展缺失: %s", target, expected)
					}
				}
			}
		})
	}
}

// TestCollectChainedNames 测试伴随扩展收集
func TestCollectChainedNames(t *testing.T) {
	tests := []struct {
		name     string
		exts     []*ExtInfo
		base     string
		expected []string
	}{
		{
			name: "单层伴随",
			exts: []*ExtInfo{
				{Name: "main"},
				{Name: "companion", ActiveWith: []string{"main"}},
			},
			base:     "main",
			expected: []string{"companion"},
		},
		{
			name: "两层链式伴随",
			exts: []*ExtInfo{
				{Name: "main"},
				{Name: "level1", ActiveWith: []string{"main"}},
				{Name: "level2", ActiveWith: []string{"level1"}},
			},
			base:     "main",
			expected: []string{"level2", "level1"}, // 拓扑顺序：先 level2，再 level1
		},
		{
			name: "多个直接伴随",
			exts: []*ExtInfo{
				{Name: "main"},
				{Name: "companion1", ActiveWith: []string{"main"}},
				{Name: "companion2", ActiveWith: []string{"main"}},
			},
			base:     "main",
			expected: []string{"companion1", "companion2"},
		},
		{
			name: "无伴随扩展",
			exts: []*ExtInfo{
				{Name: "main"},
				{Name: "other"},
			},
			base:     "main",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := BuildActiveWithGraph(tt.exts)
			logger := zap.NewNop().Sugar()
			result := collectChainedNames(logger, graph, tt.base, maxChainDepth)

			if len(result) != len(tt.expected) {
				t.Errorf("收集到的伴随扩展数量不匹配: got %d, want %d\ngot: %v\nwant: %v",
					len(result), len(tt.expected), result, tt.expected)
				return
			}

			// 转换为集合进行比较（因为顺序可能不同）
			resultSet := make(map[string]bool)
			for _, name := range result {
				resultSet[name] = true
			}

			for _, expected := range tt.expected {
				if !resultSet[expected] {
					t.Errorf("期望的伴随扩展 %s 未被收集", expected)
				}
			}
		})
	}
}

// TestExtActivateWithCompanion 测试扩展激活（包含伴随扩展）
func TestExtActivateWithCompanion(t *testing.T) {
	tests := []struct {
		name                string
		exts                []*ExtInfo
		activateExtName     string
		expectedActivated   []string // 预期被激活的扩展名（按优先级从高到低）
		expectedInactivated map[string]bool
	}{
		{
			name: "激活主扩展，伴随扩展自动激活",
			exts: []*ExtInfo{
				{Name: "main"},
				{Name: "companion", ActiveWith: []string{"main"}},
			},
			activateExtName:   "main",
			expectedActivated: []string{"companion", "main"}, // companion 优先级更高
		},
		{
			name: "链式激活",
			exts: []*ExtInfo{
				{Name: "main"},
				{Name: "level1", ActiveWith: []string{"main"}},
				{Name: "level2", ActiveWith: []string{"level1"}},
			},
			activateExtName:   "main",
			expectedActivated: []string{"level2", "level1", "main"},
		},
		{
			name: "多个伴随扩展",
			exts: []*ExtInfo{
				{Name: "main"},
				{Name: "companion1", ActiveWith: []string{"main"}},
				{Name: "companion2", ActiveWith: []string{"main"}},
			},
			activateExtName:   "main",
			expectedActivated: []string{"companion1", "companion2", "main"}, // companion 们在前
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dice := newTestDice(tt.exts)
			group := newTestGroupInfo()

			// 激活指定扩展
			ext := dice.ExtFind(tt.activateExtName, false)
			if ext == nil {
				t.Fatalf("找不到扩展: %s", tt.activateExtName)
			}

			group.ExtActive(ext)

			// 验证激活的扩展数量
			if len(group.ActivatedExtList) != len(tt.expectedActivated) {
				t.Errorf("激活的扩展数量不匹配: got %d, want %d\ngot: %v",
					len(group.ActivatedExtList), len(tt.expectedActivated),
					extListToNames(group.ActivatedExtList))
				return
			}

			// 验证所有预期的扩展都被激活（使用集合比较，不关心具体顺序）
			activatedSet := make(map[string]bool)
			for _, ext := range group.ActivatedExtList {
				activatedSet[ext.Name] = true
			}

			for _, expectedName := range tt.expectedActivated {
				if !activatedSet[expectedName] {
					t.Errorf("预期激活的扩展 %s 未被激活", expectedName)
				}
			}

			// 验证主扩展（tt.activateExtName）在所有伴随扩展之后
			// （伴随扩展优先级应该更高，所以在列表前面）
			mainExtIndex := -1
			for i, ext := range group.ActivatedExtList {
				if ext.Name == tt.activateExtName {
					mainExtIndex = i
					break
				}
			}

			if mainExtIndex == -1 {
				t.Errorf("主扩展 %s 未被激活", tt.activateExtName)
			} else {
				// 检查主扩展之前的扩展是否都是伴随扩展
				for i := range group.ActivatedExtList[:mainExtIndex] {
					extName := group.ActivatedExtList[i].Name
					if extName == tt.activateExtName {
						t.Errorf("主扩展 %s 不应该出现在位置 %d（应该在所有伴随扩展之后）", tt.activateExtName, i)
					}
				}
			}

			// 验证 InactivatedExtSet
			for _, activatedExt := range group.ActivatedExtList {
				if group.IsExtInactivated(activatedExt.Name) {
					t.Errorf("扩展 %s 不应该在 InactivatedExtSet 中", activatedExt.Name)
				}
			}
		})
	}
}

// TestExtDeactivateWithCompanion 测试扩展关闭（包含伴随扩展）
func TestExtDeactivateWithCompanion(t *testing.T) {
	tests := []struct {
		name              string
		exts              []*ExtInfo
		activateFirst     []string // 先激活这些扩展
		deactivateExtName string   // 然后关闭这个扩展
		expectedRemaining []string // 预期剩余的扩展
	}{
		{
			name: "关闭主扩展，伴随扩展自动关闭",
			exts: []*ExtInfo{
				{Name: "main"},
				{Name: "companion", ActiveWith: []string{"main"}},
			},
			activateFirst:     []string{"main"},
			deactivateExtName: "main",
			expectedRemaining: []string{},
		},
		{
			name: "链式关闭",
			exts: []*ExtInfo{
				{Name: "main"},
				{Name: "level1", ActiveWith: []string{"main"}},
				{Name: "level2", ActiveWith: []string{"level1"}},
			},
			activateFirst:     []string{"main"},
			deactivateExtName: "main",
			expectedRemaining: []string{},
		},
		{
			name: "只关闭伴随扩展，主扩展保留",
			exts: []*ExtInfo{
				{Name: "main"},
				{Name: "companion", ActiveWith: []string{"main"}},
			},
			activateFirst:     []string{"main"},
			deactivateExtName: "companion",
			expectedRemaining: []string{"main"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dice := newTestDice(tt.exts)
			group := newTestGroupInfo()

			// 先激活指定的扩展
			for _, extName := range tt.activateFirst {
				ext := dice.ExtFind(extName, false)
				if ext == nil {
					t.Fatalf("找不到扩展: %s", extName)
				}
				group.ExtActive(ext)
			}

			// 关闭指定扩展
			group.ExtInactiveByName(tt.deactivateExtName)

			// 验证剩余扩展
			if len(group.ActivatedExtList) != len(tt.expectedRemaining) {
				t.Errorf("剩余扩展数量不匹配: got %d, want %d\ngot: %v\nwant: %v",
					len(group.ActivatedExtList), len(tt.expectedRemaining),
					extListToNames(group.ActivatedExtList), tt.expectedRemaining)
				return
			}

			remainingSet := make(map[string]bool)
			for _, ext := range group.ActivatedExtList {
				remainingSet[ext.Name] = true
			}

			for _, expectedName := range tt.expectedRemaining {
				if !remainingSet[expectedName] {
					t.Errorf("预期剩余的扩展 %s 未找到", expectedName)
				}
			}
		})
	}
}

// TestCircularDependencyDetection 测试循环依赖检测
func TestCircularDependencyDetection(t *testing.T) {
	// 注意：循环依赖会被检测并跳过，不会导致死循环
	exts := []*ExtInfo{
		{Name: "a", ActiveWith: []string{"b"}},
		{Name: "b", ActiveWith: []string{"a"}},
	}

	dice := newTestDice(exts)
	graph := dice.activeWithGraph()
	logger := zap.NewNop().Sugar()

	// 收集 a 的伴随扩展（应该不会死循环）
	result := collectChainedNames(logger, graph, "a", maxChainDepth)

	// 应该只收集到 b（因为检测到循环后会跳过）
	if len(result) > 1 {
		t.Errorf("循环依赖检测失败，收集到了过多的扩展: %v", result)
	}
}

// TestCommandPriority 测试指令优先级：晚开启的扩展优先级更高
func TestCommandPriority(t *testing.T) {
	tests := []struct {
		name              string
		exts              []*ExtInfo
		activateOrder     []string // 按顺序激活这些扩展
		expectedListOrder []string // 预期的 ActivatedExtList 顺序（从前到后）
	}{
		{
			name: "先开扩展1，再开扩展2，扩展2优先级更高",
			exts: []*ExtInfo{
				{Name: "ext1"},
				{Name: "ext2"},
			},
			activateOrder:     []string{"ext1", "ext2"},
			expectedListOrder: []string{"ext2", "ext1"}, // ext2 在前，优先级高
		},
		{
			name: "三个扩展，按顺序开启",
			exts: []*ExtInfo{
				{Name: "ext1"},
				{Name: "ext2"},
				{Name: "ext3"},
			},
			activateOrder:     []string{"ext1", "ext2", "ext3"},
			expectedListOrder: []string{"ext3", "ext2", "ext1"},
		},
		{
			name: "主扩展和伴随扩展，伴随扩展优先级高于主扩展",
			exts: []*ExtInfo{
				{Name: "main"},
				{Name: "companion", ActiveWith: []string{"main"}},
			},
			activateOrder:     []string{"main"},              // 只激活 main，companion 自动激活
			expectedListOrder: []string{"companion", "main"}, // companion 优先级更高
		},
		{
			name: "先开主扩展，再开另一个独立扩展",
			exts: []*ExtInfo{
				{Name: "main"},
				{Name: "companion", ActiveWith: []string{"main"}},
				{Name: "other"},
			},
			activateOrder:     []string{"main", "other"},
			expectedListOrder: []string{"other", "companion", "main"}, // other 最晚开，优先级最高
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dice := newTestDice(tt.exts)
			group := newTestGroupInfo()

			// 按顺序激活扩展
			for _, extName := range tt.activateOrder {
				ext := dice.ExtFind(extName, false)
				if ext == nil {
					t.Fatalf("找不到扩展: %s", extName)
				}
				group.ExtActive(ext)
			}

			// 验证 ActivatedExtList 的顺序
			if len(group.ActivatedExtList) != len(tt.expectedListOrder) {
				t.Errorf("激活的扩展数量不匹配: got %d, want %d\ngot: %v\nwant: %v",
					len(group.ActivatedExtList), len(tt.expectedListOrder),
					extListToNames(group.ActivatedExtList), tt.expectedListOrder)
				return
			}

			for i, expectedName := range tt.expectedListOrder {
				actualName := group.ActivatedExtList[i].Name
				if actualName != expectedName {
					t.Errorf("位置 %d 的扩展不匹配: got %s, want %s\n完整顺序: %v",
						i, actualName, expectedName, extListToNames(group.ActivatedExtList))
				}
			}

			// 额外验证：列表前面的扩展应该比后面的优先级高
			t.Logf("激活顺序（优先级从高到低）: %v", extListToNames(group.ActivatedExtList))
		})
	}
}

// TestReloadExtensionKeepsPriority 测试重载扩展保持原有优先级
func TestReloadExtensionKeepsPriority(t *testing.T) {
	dice := newTestDice([]*ExtInfo{
		{Name: "ext1"},
		{Name: "ext2"},
	})
	group := newTestGroupInfo()

	// 第一步：按顺序激活 ext1, ext2
	group.ExtActive(dice.ExtFind("ext1", false))
	group.ExtActive(dice.ExtFind("ext2", false))

	// 验证初始顺序：[ext2, ext1]（ext2 优先级高）
	expectedOrder1 := []string{"ext2", "ext1"}
	actualOrder1 := extListToNames(group.ActivatedExtList)
	if !stringSliceEqual(actualOrder1, expectedOrder1) {
		t.Errorf("初始激活后顺序不匹配:\ngot:  %v\nwant: %v", actualOrder1, expectedOrder1)
	}

	t.Logf("初始激活顺序（优先级从高到低）: %v", actualOrder1)
	t.Logf("快照内容: %v", group.ExtActiveListSnapshot)

	// 第二步：模拟脚本热重载
	// 保存快照（模拟持久化到磁盘）
	savedSnapshot := make([]string, len(group.ExtActiveListSnapshot))
	copy(savedSnapshot, group.ExtActiveListSnapshot)

	// 清空激活列表（模拟扩展从内存中卸载）
	group.ActivatedExtList = nil

	// 恢复快照（模拟从磁盘加载）
	group.ExtActiveListSnapshot = savedSnapshot

	t.Logf("模拟重载：快照恢复为 %v", group.ExtActiveListSnapshot)

	// 使用 ExtActiveBatchBySnapshotOrder 重新激活所有扩展（模拟脚本重载）
	ext1 := dice.ExtFind("ext1", false)
	ext2 := dice.ExtFind("ext2", false)
	group.ExtActiveBatchBySnapshotOrder([]*ExtInfo{ext1, ext2}, nil)

	// 验证重载后顺序：仍然是 [ext2, ext1]（ext2 优先级仍然高于 ext1）
	expectedOrder2 := []string{"ext2", "ext1"}
	actualOrder2 := extListToNames(group.ActivatedExtList)
	if !stringSliceEqual(actualOrder2, expectedOrder2) {
		t.Errorf("重载 ext1 后顺序不匹配:\ngot:  %v\nwant: %v", actualOrder2, expectedOrder2)
	}

	t.Logf("✓ 重载扩展后保持原有优先级，顺序: %v", actualOrder2)
}

// TestReopenExtensionPriority 测试重复开启扩展会提升优先级
func TestReopenExtensionPriority(t *testing.T) {
	dice := newTestDice([]*ExtInfo{
		{Name: "ext1"},
		{Name: "ext2"},
		{Name: "ext3"},
	})
	group := newTestGroupInfo()

	// 第一轮：按顺序激活 ext1, ext2, ext3
	group.ExtActive(dice.ExtFind("ext1", false))
	group.ExtActive(dice.ExtFind("ext2", false))
	group.ExtActive(dice.ExtFind("ext3", false))

	// 预期顺序：[ext3, ext2, ext1]
	expectedOrder1 := []string{"ext3", "ext2", "ext1"}
	actualOrder1 := extListToNames(group.ActivatedExtList)
	if !stringSliceEqual(actualOrder1, expectedOrder1) {
		t.Errorf("第一轮激活后顺序不匹配:\ngot:  %v\nwant: %v", actualOrder1, expectedOrder1)
	}

	// 第二轮：重新激活 ext1（应该提升到最前面）
	group.ExtActive(dice.ExtFind("ext1", false))

	// 预期顺序：[ext1, ext3, ext2]（ext1 被移到最前）
	expectedOrder2 := []string{"ext1", "ext3", "ext2"}
	actualOrder2 := extListToNames(group.ActivatedExtList)
	if !stringSliceEqual(actualOrder2, expectedOrder2) {
		t.Errorf("重新激活 ext1 后顺序不匹配:\ngot:  %v\nwant: %v", actualOrder2, expectedOrder2)
	}

	t.Logf("✓ 重新激活扩展成功提升优先级")
}

// 辅助函数：将 ExtInfo 列表转换为名称列表
func extListToNames(exts []*ExtInfo) []string {
	names := make([]string, len(exts))
	for i, ext := range exts {
		if ext != nil {
			names[i] = ext.Name
		}
	}
	return names
}

// TestCompanionFollowsMainExtension 测试伴随扩展随主扩展开关（场景3）
func TestCompanionFollowsMainExtension(t *testing.T) {
	dice := newTestDice([]*ExtInfo{
		{Name: "main"},
		{Name: "companion", ActiveWith: []string{"main"}},
	})
	group := newTestGroupInfo()

	// 第一步：开启主扩展，伴随扩展自动开启
	group.ExtActive(dice.ExtFind("main", false))

	if len(group.ActivatedExtList) != 2 {
		t.Fatalf("开启主扩展后应该有2个扩展，实际: %d", len(group.ActivatedExtList))
	}

	t.Logf("✓ 开启主扩展后：%v", extListToNames(group.ActivatedExtList))

	// 第二步：关闭主扩展，伴随扩展自动关闭
	group.ExtInactive(dice.ExtFind("main", false))

	if len(group.ActivatedExtList) != 0 {
		t.Errorf("关闭主扩展后应该没有扩展，实际: %v", extListToNames(group.ActivatedExtList))
	}

	t.Logf("✓ 关闭主扩展后：所有扩展已关闭")
}

// TestReopenMainExtensionReopensCompanion 测试重新开启主扩展会重新开启已关闭的伴随扩展（场景4）
func TestReopenMainExtensionReopensCompanion(t *testing.T) {
	dice := newTestDice([]*ExtInfo{
		{Name: "main"},
		{Name: "companion", ActiveWith: []string{"main"}},
	})
	group := newTestGroupInfo()

	// 第一步：开启主扩展（伴随扩展自动开启）
	group.ExtActive(dice.ExtFind("main", false))

	initialOrder := extListToNames(group.ActivatedExtList)
	t.Logf("初始状态：%v", initialOrder)

	if len(group.ActivatedExtList) != 2 {
		t.Fatalf("开启主扩展后应该有2个扩展，实际: %d", len(group.ActivatedExtList))
	}

	// 第二步：用户手动关闭伴随扩展
	group.ExtInactive(dice.ExtFind("companion", false))

	afterCloseCompanion := extListToNames(group.ActivatedExtList)
	t.Logf("手动关闭伴随扩展后：%v", afterCloseCompanion)

	if len(group.ActivatedExtList) != 1 || group.ActivatedExtList[0].Name != "main" {
		t.Errorf("关闭伴随扩展后应该只剩主扩展，实际: %v", afterCloseCompanion)
	}

	// 验证 companion 被标记为手动关闭
	if !group.IsExtInactivated("companion") {
		t.Error("companion 应该被标记为手动关闭（在 InactivatedExtSet 中）")
	}

	// 第三步：重新开启主扩展（用户执行 .ext main on）
	group.ExtActive(dice.ExtFind("main", false))

	finalOrder := extListToNames(group.ActivatedExtList)
	t.Logf("重新开启主扩展后：%v", finalOrder)

	// 验证：伴随扩展应该重新开启
	if len(group.ActivatedExtList) != 2 {
		t.Errorf("重新开启主扩展后应该有2个扩展，实际: %d, %v",
			len(group.ActivatedExtList), finalOrder)
	}

	// 验证两个扩展都存在
	hasMain := false
	hasCompanion := false
	for _, ext := range group.ActivatedExtList {
		if ext.Name == "main" {
			hasMain = true
		}
		if ext.Name == "companion" {
			hasCompanion = true
		}
	}

	if !hasMain {
		t.Error("主扩展 main 未被激活")
	}
	if !hasCompanion {
		t.Error("伴随扩展 companion 未被重新激活")
	}

	// 验证 companion 不再被标记为手动关闭
	if group.IsExtInactivated("companion") {
		t.Error("companion 不应该再被标记为手动关闭")
	}

	t.Logf("✓ 重新开启主扩展成功重新激活伴随扩展")
}

// 辅助函数：比较两个字符串切片是否相等
func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
