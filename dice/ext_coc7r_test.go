//nolint:testpackage
package dice

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"sealdice-core/model"
	"sealdice-core/utils/constant"
)

// ---------------------------------------------------------------------------
// 模板层测试：coc7r 必须准确反映 Keeper Companion 的更新技能表
// ---------------------------------------------------------------------------

func TestCoc7RTemplateBasics(t *testing.T) {
	tmpl, err := loadBuiltinTemplate("coc7r.yaml")
	if err != nil {
		t.Fatalf("load coc7r template: %v", err)
	}
	if tmpl.Name != "coc7r" {
		t.Fatalf("Name = %q, want coc7r", tmpl.Name)
	}

	wantDefaults := map[string]int{
		"说服":       10,
		"宗教学":      10,
		"生存":       10,
		"链锯":       15,
		"连枷":       15,
		"鞭":        15,
		"射击:重武器":   15,
		"射击:火焰喷射器": 15,
		"射击:机枪":    15,
		"数学":       1,
		"克苏鲁神话":    0,
	}
	for k, want := range wantDefaults {
		got, exists := tmpl.Attrs.Defaults[k]
		if !exists {
			t.Fatalf("缺少技能 %s", k)
		}
		if got != want {
			t.Fatalf("%s = %d, want %d", k, got, want)
		}
	}

	// 被合并/取消的技能不应再作为新卡技能出现
	for _, removed := range []string{"取悦", "话术", "恐吓", "学识", "沙漠", "海洋", "极地", "射击:冲锋枪"} {
		if _, exists := tmpl.Attrs.Defaults[removed]; exists {
			t.Fatalf("技能 %s 应已从 coc7r 默认表移除", removed)
		}
	}

	// 安抚由外貌推导
	if _, exists := tmpl.Attrs.DefaultsComputed["安抚"]; !exists {
		t.Fatal("安抚 应为计算属性（外貌/5）")
	}

	// 别名：新技能可被英文/同名别名索引
	if got := tmpl.GetAlias("Persuade"); got != "说服" {
		t.Fatalf("Persuade alias = %q, want 说服", got)
	}
	if got := tmpl.GetAlias("Reassure"); got != "安抚" {
		t.Fatalf("Reassure alias = %q, want 安抚", got)
	}

	if len(tmpl.SetConfig.RelatedExt) != 2 || tmpl.SetConfig.RelatedExt[0] != "coc7r" || tmpl.SetConfig.RelatedExt[1] != "coc7" {
		t.Fatalf("relatedExt = %v, want [coc7r coc7]", tmpl.SetConfig.RelatedExt)
	}
	if tmpl.SetConfig.DiceSides != 100 {
		t.Fatalf("diceSides = %d, want 100", tmpl.SetConfig.DiceSides)
	}
	foundKey := false
	for _, k := range tmpl.SetConfig.Keys {
		if k == "coc7r" {
			foundKey = true
		}
	}
	if !foundKey {
		t.Fatalf("set keys 缺少 coc7r: %v", tmpl.SetConfig.Keys)
	}
}

// ---------------------------------------------------------------------------
// 指令回归：真实跑一遍 ExecuteNew，覆盖 coc7r 全部对外指令
// ---------------------------------------------------------------------------

// prepareC7RTestDB 为测试库补建属性/群组表，mock 数据库默认是空库。
func prepareC7RTestDB(t *testing.T, d *Dice) {
	t.Helper()
	db := d.DBOperator.GetDataDB(constant.WRITE)
	if err := db.AutoMigrate(&model.AttributesItemModel{}, &model.GroupInfo{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	// 扩展存储会启动 buntdb 后台协程，测试结束前关闭，避免 goleak 报错
	t.Cleanup(func() {
		// 先等待异步指令任务收尾，再关扩展库，否则后台任务可能把库重新打开
		waitForTestMessageTasks(t, d)
		if ext := d.ExtFind(c7rExtName, false); ext != nil {
			_ = ext.StorageClose()
		} else {
			t.Logf("ext not found")
		}
	})
}

// skillValue 从 .st show 的回复里取出某个技能的数值。
func skillValue(t *testing.T, reply, skill string) int64 {
	t.Helper()
	re := regexp.MustCompile(regexp.QuoteMeta(skill) + `:(\d+)`)
	m := re.FindStringSubmatch(reply)
	if m == nil {
		t.Fatalf("回复里找不到技能 %s: %q", skill, reply)
	}
	v, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		t.Fatalf("解析技能值失败: %v", err)
	}
	return v
}

func drainReply(a *mockPlatformAdapter) {
	for {
		select {
		case <-a.msgCh:
		default:
			return
		}
	}
}

func runCmd(t *testing.T, d *Dice, ep *EndPointInfo, a *mockPlatformAdapter, text string) string {
	t.Helper()
	drainReply(a)
	d.ImSession.ExecuteNew(ep, newGroupMsg("QQ-Group:c7r", "QQ:777", text))
	msg, ok := a.waitForMsg(3 * time.Second)
	if !ok {
		t.Fatalf("指令 %q 没有回复", text)
	}
	return msg
}

func TestCoc7RCommandRegression(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	t.Cleanup(cleanup)
	prepareC7RTestDB(t, d)

	// 1. 切换规则
	got := runCmd(t, d, ep, adapter, ".set coc7r")
	if !strings.Contains(got, "coc7r") {
		t.Fatalf(".set coc7r 回复异常: %q", got)
	}
	group, ok := d.ImSession.ServiceAtNew.Load("QQ-Group:c7r")
	if !ok {
		t.Fatal("群信息未创建")
	}
	if group.System != "coc7r" {
		t.Fatalf("group.System = %q, want coc7r", group.System)
	}

	// 2. 设卡
	got = runCmd(t, d, ep, adapter, ".st 力量60 敏捷50 体质50 体型50 外貌50 智力60 意志50 教育60 幸运50 理智50")
	if strings.Contains(got, "错误") {
		t.Fatalf(".st 设卡失败: %q", got)
	}

	// 3. 展示属性
	got = runCmd(t, d, ep, adapter, ".st show")
	if !strings.Contains(got, "理智") || !strings.Contains(got, "力量") {
		t.Fatalf(".st show 输出异常: %q", got)
	}

	// 4. 普通检定
	got = runCmd(t, d, ep, adapter, ".ra 侦查")
	if !strings.Contains(got, "检定结果为") {
		t.Fatalf(".ra 侦查 输出异常: %q", got)
	}
	if !strings.Contains(got, "D100=") {
		t.Fatalf(".ra 侦查 未显示 D100: %q", got)
	}

	// 5. 困难检定
	got = runCmd(t, d, ep, adapter, ".ra 困难侦查")
	if !strings.Contains(got, "困难") && !strings.Contains(got, "D100=") {
		t.Fatalf(".ra 困难侦查 输出异常: %q", got)
	}

	// 6. 奖励骰：输出里必须出现奖励骰，而不是只把 50 当目标值
	got = runCmd(t, d, ep, adapter, ".ra b 50")
	if !strings.Contains(got, "奖励") {
		t.Fatalf(".ra b 50 未使用奖励骰: %q", got)
	}

	// 7. 新版理智判定：单次损失 5 点必然要求 INT 检定
	got = runCmd(t, d, ep, adapter, ".sc 5/5")
	if !strings.Contains(got, "理智检定") {
		t.Fatalf(".sc 输出异常: %q", got)
	}
	// 失败必触发；成功但掉 1-4 点同样触发。5/5 无论成败都会掉 5 点
	if !strings.Contains(got, "不自觉反应") {
		t.Fatalf(".sc 单次掉 5 点未提示不自觉反应: %q", got)
	}
	if !strings.Contains(got, "INT 检定") {
		t.Fatalf(".sc 损失 5 点未触发 INT 检定: %q", got)
	}

	// 8. 再来一次，累计损失达到 POW/5=10，应转为 Changed
	runCmd(t, d, ep, adapter, ".c7r shock clear")
	got = runCmd(t, d, ep, adapter, ".sc 5/5")
	if !strings.Contains(got, "已 Changed") {
		t.Fatalf(".sc 累计损失未触发 Changed: %q", got)
	}

	// 9. 圆桌先攻（收在 .c7r 下面，不占用顶层 .ri）
	got = runCmd(t, d, ep, adapter, ".c7r ri")
	if !strings.Contains(got, "回合计序") {
		t.Fatalf(".c7r ri 输出异常: %q", got)
	}

	// 10. 可选规则列表
	got = runCmd(t, d, ep, adapter, ".c7r rule")
	if !strings.Contains(got, "sanity") || !strings.Contains(got, "mythos") {
		t.Fatalf(".c7r rule 输出异常: %q", got)
	}

	// 11. 状态
	got = runCmd(t, d, ep, adapter, ".c7r status")
	if !strings.Contains(got, "本篇章累计损失") || !strings.Contains(got, "可选规则") {
		t.Fatalf(".c7r status 输出异常: %q", got)
	}

	// 12. 旧卡迁移
	runCmd(t, d, ep, adapter, ".st 话术40")
	got = runCmd(t, d, ep, adapter, ".c7r migrate")
	if !strings.Contains(got, "旧卡迁移完成") || !strings.Contains(got, "说服 ← 话术 40") {
		t.Fatalf(".c7r migrate 输出异常: %q", got)
	}
	got = runCmd(t, d, ep, adapter, ".st show 说服")
	if !strings.Contains(got, "40") {
		t.Fatalf("迁移后说服不是 40: %q", got)
	}

	// 13. 重置篇章账本
	got = runCmd(t, d, ep, adapter, ".c7r reset")
	if !strings.Contains(got, "已清空") {
		t.Fatalf(".c7r reset 输出异常: %q", got)
	}

	// 14. 制卡
	got = runCmd(t, d, ep, adapter, ".coc")
	if !strings.Contains(got, "人物作成") {
		t.Fatalf(".coc 输出异常: %q", got)
	}
}

func TestCoc7RSanityLostAndCap(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	t.Cleanup(cleanup)
	prepareC7RTestDB(t, d)

	runCmd(t, d, ep, adapter, ".set coc7r")
	runCmd(t, d, ep, adapter, ".st 意志50 智力50 理智6 克苏鲁神话0")

	// 单次损失 5 点，理智 6 → 1，未归零
	got := runCmd(t, d, ep, adapter, ".sc 5/5")
	if strings.Contains(got, "Lost") {
		t.Fatalf("理智未归零却判定 Lost: %q", got)
	}

	// 再扣一次，归零 → Lost（先解除可能的 Shock）
	runCmd(t, d, ep, adapter, ".c7r shock clear")
	got = runCmd(t, d, ep, adapter, ".sc 5/5")
	if !strings.Contains(got, "Lost") {
		t.Fatalf("理智归零未判定 Lost: %q", got)
	}
}

func TestCoc7RSanityCapKwarg(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	t.Cleanup(cleanup)
	prepareC7RTestDB(t, d)

	runCmd(t, d, ep, adapter, ".set coc7r")
	runCmd(t, d, ep, adapter, ".st 意志50 理智50")

	// 习惯恐惧：单次最多扣 2 点
	got := runCmd(t, d, ep, adapter, ".sc 6/6 --cap=2")
	if !strings.Contains(got, "-2") {
		t.Fatalf("--cap 未生效: %q", got)
	}
}

func TestCoc7RDoesNotHijackCoc7Sanity(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	t.Cleanup(cleanup)
	prepareC7RTestDB(t, d)

	// 经典七版下 .sc 仍走旧流程（不出现“不自觉反应”）
	got := runCmd(t, d, ep, adapter, ".set coc7")
	if !strings.Contains(got, "coc7") {
		t.Fatalf(".set coc7 输出异常: %q", got)
	}
	runCmd(t, d, ep, adapter, ".st 意志50 智力50 理智50")
	got = runCmd(t, d, ep, adapter, ".sc 0/1")
	if strings.Contains(got, "不自觉反应") {
		t.Fatalf("coc7 规则误用了 coc7r 理智流程: %q", got)
	}
	if !strings.Contains(got, "理智检定") {
		t.Fatalf("coc7 理智检定输出异常: %q", got)
	}
}

func TestCoc7RSwitchBackRequiresFormat(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	t.Cleanup(cleanup)
	prepareC7RTestDB(t, d)

	runCmd(t, d, ep, adapter, ".set coc7r")
	runCmd(t, d, ep, adapter, ".st 力量50 理智50")
	runCmd(t, d, ep, adapter, ".set coc7")

	// 卡还是 coc7r，直接被拦下
	got := runCmd(t, d, ep, adapter, ".ra 力量")
	if !strings.Contains(got, "阻止操作") {
		t.Fatalf("切回 coc7 后旧卡未被拦截: %q", got)
	}

	// 强制转卡后恢复
	got = runCmd(t, d, ep, adapter, ".st fmt")
	if strings.Contains(got, "阻止操作") {
		t.Fatalf(".st fmt 失败: %q", got)
	}
	got = runCmd(t, d, ep, adapter, ".ra 力量")
	if strings.Contains(got, "阻止操作") {
		t.Fatalf(".st fmt 后仍被拦截: %q", got)
	}
}

func TestCoc7RComputedReassure(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	t.Cleanup(cleanup)
	prepareC7RTestDB(t, d)

	runCmd(t, d, ep, adapter, ".set coc7r")
	runCmd(t, d, ep, adapter, ".st 外貌50 意志50 智力50 理智50")

	// 安抚 = 外貌/5 = 10，应可直接用于检定
	got := runCmd(t, d, ep, adapter, ".ra 安抚")
	if !strings.Contains(got, "/10") {
		t.Fatalf("安抚基础值应为 10: %q", got)
	}
}

// 可选规则开关：关闭 sanity 后 .sc 回到七版流程，既不提示不自觉反应，也不记账。
func TestCoc7RRuleSanityToggle(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	t.Cleanup(cleanup)
	prepareC7RTestDB(t, d)

	runCmd(t, d, ep, adapter, ".set coc7r")
	runCmd(t, d, ep, adapter, ".st 意志50 智力50 理智50")

	got := runCmd(t, d, ep, adapter, ".c7r rule")
	if !strings.Contains(got, "sanity：开") {
		t.Fatalf("sanity 应默认开启: %q", got)
	}
	if !strings.Contains(got, "mythos：关") {
		t.Fatalf("mythos 应默认关闭: %q", got)
	}

	if got = runCmd(t, d, ep, adapter, ".c7r rule sanity off"); !strings.Contains(got, "已关闭") {
		t.Fatalf(".c7r rule sanity off 输出异常: %q", got)
	}

	got = runCmd(t, d, ep, adapter, ".sc 5/5")
	if strings.Contains(got, "不自觉反应") || strings.Contains(got, "INT 检定") {
		t.Fatalf("sanity 关闭后仍走了新版流程: %q", got)
	}
	if !strings.Contains(got, "理智检定") {
		t.Fatalf("sanity 关闭后没有正常结算: %q", got)
	}
	status := runCmd(t, d, ep, adapter, ".c7r status")
	if !strings.Contains(status, "本篇章累计损失 0") {
		t.Fatalf("sanity 关闭后不应记账: %q", status)
	}

	if got = runCmd(t, d, ep, adapter, ".c7r rule sanity on"); !strings.Contains(got, "已开启") {
		t.Fatalf(".c7r rule sanity on 输出异常: %q", got)
	}
	got = runCmd(t, d, ep, adapter, ".sc 5/5")
	if !strings.Contains(got, "不自觉反应") {
		t.Fatalf("sanity 重新开启后未恢复新版流程: %q", got)
	}
}

// 奖惩骰解析：b2 / p2 与 .ra 保持一致，b 与 p 同时写会抵消。
func TestCoc7RBonusPenaltyDice(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	t.Cleanup(cleanup)
	prepareC7RTestDB(t, d)

	runCmd(t, d, ep, adapter, ".set coc7r")
	runCmd(t, d, ep, adapter, ".st 意志50 智力50 理智50")

	got := runCmd(t, d, ep, adapter, ".sc b2 0/1")
	if !strings.Contains(got, "奖励骰×2") {
		t.Fatalf(".sc b2 未按两颗奖励骰结算: %q", got)
	}

	got = runCmd(t, d, ep, adapter, ".sc p2 0/1")
	if !strings.Contains(got, "惩罚骰×2") {
		t.Fatalf(".sc p2 未按两颗惩罚骰结算: %q", got)
	}

	got = runCmd(t, d, ep, adapter, ".sc b p 0/1")
	if strings.Contains(got, "奖励骰") || strings.Contains(got, "惩罚骰") {
		t.Fatalf("b 与 p 同时写应互相抵消: %q", got)
	}
}

// 神话冲击需要显式开启 mythos 可选规则。
func TestCoc7RMythosRequiresToggle(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	t.Cleanup(cleanup)
	prepareC7RTestDB(t, d)

	runCmd(t, d, ep, adapter, ".set coc7r")
	// 意志 5 → Changed 阈值 1，一次 5 点损失必然 Changed
	runCmd(t, d, ep, adapter, ".st 意志5 智力50 理智50 克苏鲁神话0")

	got := runCmd(t, d, ep, adapter, ".sc 5/5 --mythos")
	if strings.Contains(got, "克苏鲁神话 +") {
		t.Fatalf("mythos 默认关闭时不应提升克苏鲁神话: %q", got)
	}

	runCmd(t, d, ep, adapter, ".c7r shock clear")
	runCmd(t, d, ep, adapter, ".c7r recover")
	runCmd(t, d, ep, adapter, ".c7r reset")
	if got = runCmd(t, d, ep, adapter, ".c7r rule mythos on"); !strings.Contains(got, "已开启") {
		t.Fatalf(".c7r rule mythos on 输出异常: %q", got)
	}

	got = runCmd(t, d, ep, adapter, ".sc 5/5 --mythos")
	if !strings.Contains(got, "克苏鲁神话 +5") {
		t.Fatalf("开启 mythos 后首次神话冲击应 +5: %q", got)
	}
}

func TestCoc7RUnknownRuleAndWrongRule(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	t.Cleanup(cleanup)
	prepareC7RTestDB(t, d)

	// 经典七版下 .c7r 会提示先切换规则
	got := runCmd(t, d, ep, adapter, ".set coc7")
	if !strings.Contains(got, "coc7") {
		t.Fatalf(".set coc7 输出异常: %q", got)
	}
	got = runCmd(t, d, ep, adapter, ".c7r status")
	if !strings.Contains(got, "set coc7r") {
		t.Fatalf(".c7r 在 coc7 群未提示切换: %q", got)
	}

	runCmd(t, d, ep, adapter, ".set coc7r")
	runCmd(t, d, ep, adapter, ".st fmt")
	got = runCmd(t, d, ep, adapter, ".c7r rule 不存在的规则 on")
	if !strings.Contains(got, "未知的可选规则") {
		t.Fatalf("未知可选规则未报错: %q", got)
	}
}

func TestCoc7RResetKeepsMythosEvents(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	t.Cleanup(cleanup)
	prepareC7RTestDB(t, d)

	runCmd(t, d, ep, adapter, ".set coc7r")
	runCmd(t, d, ep, adapter, ".c7r rule mythos on")
	// 意志 20 → Changed 阈值 4，一次 5 点损失就会 Changed 并触发神话增长
	runCmd(t, d, ep, adapter, ".st 意志20 智力50 理智50")

	first := runCmd(t, d, ep, adapter, ".sc 5/5 --mythos")
	if !strings.Contains(first, "克苏鲁神话 +5") {
		t.Fatalf("首次神话冲击应 +5: %q", first)
	}
	// reset 开新篇章：篇章损失清零，Changed 保留
	resetMsg := runCmd(t, d, ep, adapter, ".c7r reset")
	if !strings.Contains(resetMsg, "Changed 仍然保留") {
		t.Fatalf("reset 不应顺带解除 Changed: %q", resetMsg)
	}
	status := runCmd(t, d, ep, adapter, ".c7r status")
	if !strings.Contains(status, "Changed：是") {
		t.Fatalf("reset 后 Changed 应保留: %q", status)
	}

	// recover 解除 Changed 后再触发一次，神话仍按 +1 计
	runCmd(t, d, ep, adapter, ".c7r recover")
	second := runCmd(t, d, ep, adapter, ".sc 5/5 --mythos")
	if !strings.Contains(second, "克苏鲁神话 +1") {
		t.Fatalf("重置后不应重新获得首次 +5: %q", second)
	}
}

func TestCoc7RCapHint(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	t.Cleanup(cleanup)
	prepareC7RTestDB(t, d)

	runCmd(t, d, ep, adapter, ".set coc7r")
	runCmd(t, d, ep, adapter, ".st 意志50 理智50")

	got := runCmd(t, d, ep, adapter, ".sc 0/1D6 --cap 3")
	if !strings.Contains(got, "--cap=N") {
		t.Fatalf("--cap 缺少等号时应给出写法提示: %q", got)
	}
}

func TestCoc7RMigrateKeepsDefault(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	t.Cleanup(cleanup)
	prepareC7RTestDB(t, d)

	runCmd(t, d, ep, adapter, ".set coc7r")
	runCmd(t, d, ep, adapter, ".st 话术5")

	got := runCmd(t, d, ep, adapter, ".c7r migrate")
	if !strings.Contains(got, "不低于基础值 10") {
		t.Fatalf("迁移提示未说明基础值下限: %q", got)
	}
	show := runCmd(t, d, ep, adapter, ".st show 说服")
	if v := skillValue(t, show, "说服"); v != 10 {
		t.Fatalf("迁移后说服不应低于基础值 10，实际 %d", v)
	}
}

func TestCoc7RMovementFormula(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	t.Cleanup(cleanup)
	prepareC7RTestDB(t, d)

	runCmd(t, d, ep, adapter, ".set coc7r")

	cases := []struct {
		attrs string
		want  int64
	}{
		{".st 力量50 体型50 敏捷50", 7}, // 三项相等：7，不是旧公式的 8
		{".st 力量60", 8},           // 只有力量高于体型
		{".st 敏捷60", 9},           // 力量与敏捷都高于体型
	}
	for _, c := range cases {
		runCmd(t, d, ep, adapter, c.attrs)
		show := runCmd(t, d, ep, adapter, ".st show 移动力")
		if v := skillValue(t, show, "移动力"); v != c.want {
			t.Fatalf("%s 后 MOV 应为 %d，实际 %q", c.attrs, c.want, show)
		}
	}
}

// 成长取五直接复用 .en 的成长表达式：.en <技能>+5，专精技能也要能写全名。
func TestCoc7RTakeFiveViaEn(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	t.Cleanup(cleanup)
	prepareC7RTestDB(t, d)

	runCmd(t, d, ep, adapter, ".set coc7r")
	runCmd(t, d, ep, adapter, ".st 射击:手枪20")

	before := skillValue(t, runCmd(t, d, ep, adapter, ".st show 射击:手枪"), "射击:手枪")
	grown := false
	for i := 0; i < 10; i++ {
		runCmd(t, d, ep, adapter, ".en 射击:手枪+5")
		if v := skillValue(t, runCmd(t, d, ep, adapter, ".st show 射击:手枪"), "射击:手枪"); v > before {
			grown = true
			break
		}
	}
	if !grown {
		t.Fatal(".en 射击:手枪+5 连续十次都未成长，正则可能没吃掉冒号")
	}
	after := skillValue(t, runCmd(t, d, ep, adapter, ".st show 射击:手枪"), "射击:手枪")
	if after-before != 5 {
		t.Fatalf("取五应 +5，实际 %d → %d", before, after)
	}
}
func TestCoc7RShockBlocksSanityLoss(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	t.Cleanup(cleanup)
	prepareC7RTestDB(t, d)

	runCmd(t, d, ep, adapter, ".set coc7r")
	// 意志 99 → Changed 阈值 19，智力 99 让 INT 检定几乎必过，前三次不会 Changed
	runCmd(t, d, ep, adapter, ".st 意志99 智力99 理智90")

	shocked := false
	for i := 0; i < 3; i++ {
		got := runCmd(t, d, ep, adapter, ".sc 5/5")
		if strings.Contains(got, "陷入 Shock") {
			shocked = true
			break
		}
	}
	if !shocked {
		t.Fatal("三次 INT 检定都失败，无法验证 Shock 状态")
	}

	sanBefore := skillValue(t, runCmd(t, d, ep, adapter, ".st show 理智"), "理智")
	blocked := runCmd(t, d, ep, adapter, ".sc 5/5")
	if !strings.Contains(blocked, "正处于 Shock") {
		t.Fatalf("Shock 期间应拒绝结算: %q", blocked)
	}
	sanAfter := skillValue(t, runCmd(t, d, ep, adapter, ".st show 理智"), "理智")
	if sanBefore != sanAfter {
		t.Fatalf("Shock 期间仍掉了理智: %d → %d", sanBefore, sanAfter)
	}

	if got := runCmd(t, d, ep, adapter, ".c7r shock clear"); !strings.Contains(got, "已解除") {
		t.Fatalf(".c7r shock clear 输出异常: %q", got)
	}
	resumed := runCmd(t, d, ep, adapter, ".sc 5/5")
	if !strings.Contains(resumed, "（-5）") {
		t.Fatalf("解除 Shock 后应恢复扣理智: %q", resumed)
	}
}
