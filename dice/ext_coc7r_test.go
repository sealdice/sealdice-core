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

func TestCoc7RDerivedSkillFormulas(t *testing.T) {
	ch := map[string]int64{
		"力量": 60, "敏捷": 40, "体质": 50, "体型": 50,
		"外貌": 50, "智力": 70, "意志": 50, "教育": 65,
	}
	cases := map[string]int64{
		"斗殴":    30, // max(敏捷40, 力量60)/2
		"攀爬":    30,
		"说服":    10, // 外貌50/5
		"侦查":    35, // 智力70/2
		"图书馆使用": 32, // 教育65/2
		"斧":     12, // 力量60/5
		"急救":    32, // 教育65/2
	}
	for skill, want := range cases {
		fn := c7rDerivedSkills[skill]
		if fn == nil {
			t.Fatalf("缺少派生公式: %s", skill)
		}
		if got := fn(ch); got != want {
			t.Fatalf("%s = %d, want %d", skill, got, want)
		}
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

	// 9. 现实检定
	got = runCmd(t, d, ep, adapter, ".rv")
	if !strings.Contains(got, "现实检定") {
		t.Fatalf(".rv 输出异常: %q", got)
	}

	// 10. 圆桌先攻
	got = runCmd(t, d, ep, adapter, ".ri")
	if !strings.Contains(got, "回合计序") {
		t.Fatalf(".ri 输出异常: %q", got)
	}

	// 11. 技能成长取五：先过成长检定，通过后必须正好 +5
	before := skillValue(t, runCmd(t, d, ep, adapter, ".st show 侦查"), "侦查")
	passed := false
	for i := 0; i < 6; i++ {
		got = runCmd(t, d, ep, adapter, ".en5 侦查")
		if strings.Contains(got, "通过（取五）") {
			passed = true
			break
		}
		if !strings.Contains(got, "未通过") {
			t.Fatalf(".en5 输出异常: %q", got)
		}
	}
	if !passed {
		t.Fatal(".en5 连续六次成长检定未通过")
	}
	after := skillValue(t, runCmd(t, d, ep, adapter, ".st show 侦查"), "侦查")
	if after-before != 5 {
		t.Fatalf(".en5 应 +5，实际 %d → %d", before, after)
	}

	// 12. 状态
	got = runCmd(t, d, ep, adapter, ".c7r status")
	if !strings.Contains(got, "本篇章累计损失") {
		t.Fatalf(".c7r status 输出异常: %q", got)
	}

	// 13. 派生基础值
	got = runCmd(t, d, ep, adapter, ".c7r derive")
	if !strings.Contains(got, "技能基础值") && !strings.Contains(got, "未做改动") {
		t.Fatalf(".c7r derive 输出异常: %q", got)
	}

	// 14. 旧卡迁移
	runCmd(t, d, ep, adapter, ".st 话术40")
	got = runCmd(t, d, ep, adapter, ".c7r migrate")
	if !strings.Contains(got, "旧卡迁移完成") || !strings.Contains(got, "说服 ← 话术 40") {
		t.Fatalf(".c7r migrate 输出异常: %q", got)
	}
	got = runCmd(t, d, ep, adapter, ".st show 说服")
	if !strings.Contains(got, "40") {
		t.Fatalf("迁移后说服不是 40: %q", got)
	}

	// 15. 新版建卡指引
	got = runCmd(t, d, ep, adapter, ".c7r new")
	if !strings.Contains(got, "个人兴趣点") {
		t.Fatalf(".c7r new 输出异常: %q", got)
	}

	// 16. 重置篇章账本
	got = runCmd(t, d, ep, adapter, ".c7r reset")
	if !strings.Contains(got, "已清空") {
		t.Fatalf(".c7r reset 输出异常: %q", got)
	}

	// 17. 制卡
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

func TestCoc7RComputedReassureAndMythos(t *testing.T) {
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

	// 现实检定奖励骰
	got = runCmd(t, d, ep, adapter, ".rv b")
	if !strings.Contains(got, "奖励骰") {
		t.Fatalf(".rv b 未使用奖励骰: %q", got)
	}

	// 两次 5 点神话冲击，累计 10 点触发 Changed，必然获得克苏鲁神话技能
	runCmd(t, d, ep, adapter, ".sc 5/5 --mythos")
	runCmd(t, d, ep, adapter, ".c7r shock clear")
	runCmd(t, d, ep, adapter, ".sc 5/5 --mythos")
	got = runCmd(t, d, ep, adapter, ".c7r status")
	if strings.Contains(got, "神话冲击次数：0") {
		t.Fatalf("神话冲击未记账: %q", got)
	}
}

func TestCoc7RUnknownSkillAndWrongRule(t *testing.T) {
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
	got = runCmd(t, d, ep, adapter, ".en5 不存在的技能")
	if !strings.Contains(got, "未找到技能") {
		t.Fatalf(".en5 未知技能未报错: %q", got)
	}
}

func TestCoc7RSpecializationMinimums(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	t.Cleanup(cleanup)
	prepareC7RTestDB(t, d)

	runCmd(t, d, ep, adapter, ".set coc7r")
	runCmd(t, d, ep, adapter, ".st 斗殴60 斧15 射击:手枪50 射击:步霰10")

	got := runCmd(t, d, ep, adapter, ".c7r min")
	if !strings.Contains(got, "斗殴") && !strings.Contains(got, "格斗专精") {
		t.Fatalf(".c7r min 输出异常: %q", got)
	}
	// 斧 应被抬到 斗殴60 的一半 30；步霰 抬到手枪50 的一半 25
	show := runCmd(t, d, ep, adapter, ".st show 斧 射击:步霰")
	if !strings.Contains(show, "30") || !strings.Contains(show, "25") {
		t.Fatalf(".c7r min 未按半值下限修正: %q", show)
	}
}

func TestCoc7RRealityCheckLedger(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	t.Cleanup(cleanup)
	prepareC7RTestDB(t, d)

	runCmd(t, d, ep, adapter, ".set coc7r")
	// 理智 1、意志 5：现实检定几乎必然失败，且失败 1 点就达到 Changed 阈值
	runCmd(t, d, ep, adapter, ".st 意志5 理智1")

	var reply string
	for i := 0; i < 5; i++ {
		reply = runCmd(t, d, ep, adapter, ".rv")
		if strings.Contains(reply, "失败") {
			break
		}
	}
	if !strings.Contains(reply, "失败") {
		t.Fatalf(".rv 连续成功，无法验证失败路径: %q", reply)
	}
	if !strings.Contains(reply, "已 Changed") {
		t.Fatalf("现实检定的损失未计入 Changed 判定: %q", reply)
	}
	status := runCmd(t, d, ep, adapter, ".c7r status")
	if !strings.Contains(status, "本篇章累计损失 1") {
		t.Fatalf("现实检定的损失未计入篇章账本: %q", status)
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

func TestCoc7REn5RejectsNonSkill(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	t.Cleanup(cleanup)
	prepareC7RTestDB(t, d)

	runCmd(t, d, ep, adapter, ".set coc7r")
	runCmd(t, d, ep, adapter, ".st 力量50 意志50 理智50")

	for _, attr := range []string{"力量", "生命值上限", "理智"} {
		got := runCmd(t, d, ep, adapter, ".en5 "+attr)
		if !strings.Contains(got, "未找到技能") {
			t.Fatalf(".en5 不应接受属性 %s: %q", attr, got)
		}
	}
	// 属性值不能被改动
	show := runCmd(t, d, ep, adapter, ".st show 力量")
	if v := skillValue(t, show, "力量"); v != 50 {
		t.Fatalf(".en5 改动了属性: %q", show)
	}
}

func TestCoc7RResetKeepsMythosEvents(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	t.Cleanup(cleanup)
	prepareC7RTestDB(t, d)

	runCmd(t, d, ep, adapter, ".set coc7r")
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

func TestCoc7REn5RejectsNoGainSkills(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	t.Cleanup(cleanup)
	prepareC7RTestDB(t, d)

	runCmd(t, d, ep, adapter, ".set coc7r")
	runCmd(t, d, ep, adapter, ".st 克苏鲁神话5 信用评级30 理智50")

	for _, skill := range []string{"克苏鲁神话", "信用评级"} {
		got := runCmd(t, d, ep, adapter, ".en5 "+skill)
		if !strings.Contains(got, "不能通过成长检定") {
			t.Fatalf(".en5 不应提升 %s: %q", skill, got)
		}
	}
	show := runCmd(t, d, ep, adapter, ".st show 克苏鲁神话 信用评级")
	if v := skillValue(t, show, "克苏鲁神话"); v != 5 {
		t.Fatalf("克苏鲁神话被改动: %q", show)
	}
	if v := skillValue(t, show, "信用评级"); v != 30 {
		t.Fatalf("信用评级被改动: %q", show)
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
