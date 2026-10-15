//lint:file-ignore testpackage Tests need access to internal helpers and types
package dice //nolint:testpackage // tests rely on unexported helpers

import (
	"slices"
	"strings"
	"testing"
	"time"
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
			assertExtCommandCompanions(t, tt.extensions, tt.userActivateNames, tt.expectedCompanion)
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
			assertExtCommandCompanions(t, tt.extensions, tt.userActivateNames, tt.expectedCompanion)
		})
	}
}

func assertExtCommandCompanions(t *testing.T, extensions []*ExtInfo, userActivateNames, expectedCompanion []string) {
	t.Helper()
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	defer cleanup()
	for _, ext := range extensions {
		d.RegisterExtension(ext)
	}
	ctx, msg := newQuitCommandTestContext(t, d, ep, "QQ:9001", "QQ-Group:2001", "ExtTestGroup")
	ctx.Group.SetActivatedExtList(nil, nil)
	args := append(slices.Clone(userActivateNames), "on")
	result := d.CmdMap["ext"].Solve(ctx, msg, &CmdArgs{Command: "ext", Args: args})
	if !result.Matched || !result.Solved || result.ShowHelp {
		t.Fatalf("ext command returned unexpected result: %#v", result)
	}

	// ReplyToSender dispatches asynchronously through panicHandler.Once.
	// Wait for the adapter to record the reply before inspecting its buffers.
	if _, ok := adapter.waitForMsg(3 * time.Second); !ok {
		t.Fatal("timed out waiting for the ext command reply")
	}

	adapter.mu.Lock()
	replies := slices.Clone(adapter.groupMsgs)
	privateReplies := len(adapter.personMsgs)
	adapter.mu.Unlock()
	if len(replies) != 1 || privateReplies != 0 {
		t.Fatalf("expected one group reply, got group=%v private=%d", replies, privateReplies)
	}
	if !strings.HasPrefix(replies[0], "打开扩展 ") {
		t.Fatalf("expected extension activation reply, got %q", replies[0])
	}

	var gotCompanion []string
	for _, line := range strings.Split(replies[0], "\n") {
		if names, ok := strings.CutPrefix(line, "自动启用伴随扩展: "); ok {
			gotCompanion = append(gotCompanion, strings.Split(names, ", ")...)
		}
	}
	wantCompanion := slices.Clone(expectedCompanion)
	slices.Sort(gotCompanion)
	slices.Sort(wantCompanion)
	if !slices.Equal(gotCompanion, wantCompanion) {
		t.Fatalf("companion extensions in reply = %v, want %v; reply=%q", gotCompanion, wantCompanion, replies[0])
	}

	var gotActive, wantActive []string
	for _, ext := range ctx.Group.GetActivatedExtList(d) {
		gotActive = append(gotActive, ext.Name)
	}
	for _, ext := range extensions {
		wantActive = append(wantActive, ext.Name)
	}
	slices.Sort(gotActive)
	slices.Sort(wantActive)
	if !slices.Equal(gotActive, wantActive) {
		t.Fatalf("active extensions = %v, want %v", gotActive, wantActive)
	}
}
