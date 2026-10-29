package dice

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	ds "github.com/sealdice/dicescript"
)

// coc7r（克苏鲁的呼唤 第七版·守秘人伴侣）规则扩展
//
// 本扩展实现 CHA23197-H《Keeper Companion》(2026) 中与骰子/角色卡相关的部分：
//   - .sc  新版理智判定：失败必触发不自觉反应；单次损失≥5 时投 INT 决定是否 Shock；
//     Shock 持续 1D10 轮；神话来源的 Shock/Changed 会带来克苏鲁神话技能增长；
//     单个篇章累计损失达到 POW/5 时转为 Changed；理智归零为 Lost。
//   - .rv  现实检定（现实检定失败扣 1 点理智并触发不自觉反应）。
//   - .ri  圆桌式先攻（掷骰决定顺时针/逆时针）。
//   - .en5 技能成长取五（Take Five）。
//   - .c7r 规则辅助：状态、派生基础值、旧卡迁移、重置篇章损失。
//
// 判定、属性读写、房规索引沿用 coc7 扩展的既有实现，避免出现两套不一致的规则。

const (
	c7rExtName        = "coc7r"
	c7rStateKeyPrefix = "coc7r|state|"
)

// coc7rState 记录单个篇章内的理智账本，按 群 + 玩家 维度保存。
type coc7rState struct {
	ChapterLoss  int64 `json:"chapterLoss"`  // 本篇章累计理智损失
	Changed      bool  `json:"changed"`      // 是否已 Changed
	MythosEvents int   `json:"mythosEvents"` // 神话来源的 Shock/Changed 次数
}

func c7rStateKey(groupID, userID string) string {
	return c7rStateKeyPrefix + groupID + "|" + userID
}

func c7rLoadState(d *Dice, groupID, userID string) coc7rState {
	var st coc7rState
	if d == nil || groupID == "" || userID == "" {
		return st
	}
	ext := d.ExtFind(c7rExtName, false)
	if ext == nil {
		return st
	}
	raw, err := ext.StorageGet(c7rStateKey(groupID, userID))
	if err != nil || raw == "" {
		return st
	}
	_ = json.Unmarshal([]byte(raw), &st)
	return st
}

func c7rSaveState(d *Dice, groupID, userID string, st coc7rState) {
	if d == nil || groupID == "" || userID == "" {
		return
	}
	ext := d.ExtFind(c7rExtName, false)
	if ext == nil {
		return
	}
	raw, err := json.Marshal(st)
	if err != nil {
		return
	}
	_ = ext.StorageSet(c7rStateKey(groupID, userID), string(raw))
}

func c7rClearState(d *Dice, groupID, userID string) {
	if d == nil || groupID == "" || userID == "" {
		return
	}
	ext := d.ExtFind(c7rExtName, false)
	if ext == nil {
		return
	}
	// buntdb 没有单独的 Delete 封装，写入空状态等价于清空。
	_ = ext.StorageSet(c7rStateKey(groupID, userID), "")
}

// c7rIsGroupOnRule 判断当前群是否运行在 coc7r 规则下。
func c7rIsGroupOnRule(ctx *MsgContext) bool {
	return ctx != nil && ctx.Group != nil && ctx.Group.System == c7rExtName
}

// c7rAttrInt 读取角色卡上的整数值，读不到时返回 0。
func c7rAttrInt(ctx *MsgContext, tmpl *GameSystemTemplate, name string) int64 {
	if ctx == nil || tmpl == nil {
		return 0
	}
	v, err := tmpl.GetRealValue(ctx, tmpl.GetAlias(name))
	if err != nil || v == nil {
		return 0
	}
	if v.TypeId != ds.VMTypeInt {
		return 0
	}
	n, ok := v.Value.(ds.IntType)
	if !ok {
		return 0
	}
	return int64(n)
}

// c7rCurrentSan 读取当前理智；角色卡还没写理智时，按规则书以意志值初始化。
func c7rCurrentSan(ctx *MsgContext, tmpl *GameSystemTemplate) int64 {
	if ctx == nil || tmpl == nil {
		return 0
	}
	key := tmpl.GetAlias("理智")
	attrs, err := ctx.Dice.AttrsManager.LoadByCtx(ctx)
	if err == nil && attrs != nil {
		if _, exists := attrs.LoadX(key); exists {
			return c7rAttrInt(ctx, tmpl, "理智")
		}
	}
	pow := c7rAttrInt(ctx, tmpl, "意志")
	if pow > 0 {
		VarSetValueInt64(ctx, key, pow)
	}
	return pow
}

// c7rEvalInt 计算一个表达式并取整数值。
func c7rEvalInt(ctx *MsgContext, expr string, flags RollExtraFlags) (int64, bool) {
	if strings.TrimSpace(expr) == "" {
		return 0, true
	}
	r, _, err := DiceExprEvalBase(ctx, expr, flags)
	if err != nil || r == nil {
		return 0, false
	}
	if r.TypeId != ds.VMTypeInt {
		return 0, false
	}
	n, ok := r.Value.(ds.IntType)
	if !ok {
		return 0, false
	}
	return int64(n), true
}

// c7rRollDie 掷一个单骰，返回 1..sides。
func c7rRollDie(ctx *MsgContext, sides int64) int64 {
	v, ok := c7rEvalInt(ctx, fmt.Sprintf("1d%d", sides), RollExtraFlags{DisableBlock: true})
	if !ok || v < 1 {
		return 1
	}
	return v
}

// c7rRollD100 掷 D100，bonus 为真时按 COC 奖励骰规则额外掷十位骰取小。
func c7rRollD100(ctx *MsgContext, bonus bool) (int64, string) {
	if !bonus {
		v, ok := c7rEvalInt(ctx, "1d100", RollExtraFlags{DisableBlock: true})
		if !ok || v < 1 || v > 100 {
			v = 1
		}
		return v, ""
	}

	units := c7rRollDie(ctx, 10)
	if units == 10 {
		units = 0
	}
	tensA := c7rRollDie(ctx, 10)
	if tensA == 10 {
		tensA = 0
	}
	tensB := c7rRollDie(ctx, 10)
	if tensB == 10 {
		tensB = 0
	}
	a := tensA*10 + units
	b := tensB*10 + units
	if a == 0 {
		a = 100
	}
	if b == 0 {
		b = 100
	}
	pick := a
	if b < pick {
		pick = b
	}
	return pick, fmt.Sprintf("奖励骰：十位 %d/%d，个位 %d", tensA, tensB, units)
}

// c7rLossResult 解析并结算理智损失表达式。
type c7rLossResult struct {
	loss int64 // 实际扣除
}

func c7rEvalLoss(ctx *MsgContext, expr string, fumble bool, half bool, cap int64) (c7rLossResult, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		expr = "0"
	}
	v, ok := c7rEvalInt(ctx, expr, RollExtraFlags{BigFailDiceOn: fumble, DisableBlock: true})
	if !ok {
		return c7rLossResult{}, fmt.Errorf("无法解析理智损失表达式: %s", expr)
	}
	if v < 0 {
		v = 0
	}
	if half {
		v /= 2
	}
	if cap > 0 && v > cap {
		v = cap
	}
	return c7rLossResult{loss: v}, nil
}

// c7rRequireRule 非 coc7r 群时给出提示并返回 false。
func c7rRequireRule(ctx *MsgContext, msg *Message) bool {
	if c7rIsGroupOnRule(ctx) {
		return true
	}
	ReplyToSender(ctx, msg, "该指令属于 coc7r（守秘人伴侣）规则，先用 .set coc7r 切换")
	return false
}

// cmdScCoc7R 新版理智判定。
func cmdScCoc7R(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
	if !c7rIsGroupOnRule(ctx) {
		return CmdExecuteResult{Matched: true, Solved: false}
	}
	if len(cmdArgs.Args) == 0 || cmdArgs.IsArgEqual(1, "help") {
		ReplyToSender(ctx, msg, c7rScHelpText())
		return CmdExecuteResult{Matched: true, Solved: true}
	}

	mctx := GetCtxProxyFirst(ctx, cmdArgs)
	tmpl := cardRuleCheck(mctx, msg)
	if tmpl == nil {
		return CmdExecuteResult{Matched: true, Solved: true}
	}
	mctx.SystemTemplate = tmpl

	argText := cmdArgs.CleanArgs
	argCap := cmdArgs.GetKwarg("cap")
	argHalf := cmdArgs.GetKwarg("half") != nil
	mythosSource := cmdArgs.GetKwarg("mythos") != nil || cmdArgs.GetKwarg("m") != nil

	succExpr := "0"
	failExpr := argText
	if idx := strings.Index(argText, "/"); idx >= 0 {
		succExpr = strings.TrimSpace(argText[:idx])
		failExpr = strings.TrimSpace(argText[idx+1:])
	}
	if failExpr == "" {
		return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
	}

	lossCap := int64(0)
	if argCap != nil && argCap.Value != "" {
		if n, err := strconv.ParseInt(argCap.Value, 10, 64); err == nil && n > 0 {
			lossCap = n
		}
	}

	san := c7rCurrentSan(mctx, tmpl)
	pow := c7rAttrInt(mctx, tmpl, "意志")
	intel := c7rAttrInt(mctx, tmpl, "智力")

	d100, diceDetail := c7rRollD100(mctx, false)
	rank, _ := ResultCheck(mctx, mctx.Group.CocRuleIndex, d100, san, 0)
	success := rank > 0
	fumble := rank == -2

	lossExpr := succExpr
	if !success {
		lossExpr = failExpr
	}
	res, err := c7rEvalLoss(mctx, lossExpr, fumble, argHalf, lossCap)
	if err != nil {
		ReplyToSender(mctx, msg, err.Error())
		return CmdExecuteResult{Matched: true, Solved: true}
	}
	loss := res.loss

	sanNew := san - loss
	if sanNew < 0 {
		sanNew = 0
	}
	VarSetValueInt64(mctx, "理智", sanNew)

	st := c7rLoadState(mctx.Dice, mctx.Group.GroupID, mctx.Player.UserID)
	st.ChapterLoss += loss

	changedThreshold := pow / 5
	if changedThreshold < 1 {
		changedThreshold = 1
	}
	// Shock 阈值按“本次事件发生前”的状态判断：刚刚 Changed 的那一次仍按 5 点计算
	shockThreshold := int64(5)
	if st.Changed {
		shockThreshold = 10
	}
	newlyChanged := !st.Changed && st.ChapterLoss >= changedThreshold
	if newlyChanged {
		st.Changed = true
	}

	var b strings.Builder
	verdict := "成功"
	switch {
	case fumble:
		verdict = "大失败"
	case !success:
		verdict = "失败"
	}
	if diceDetail != "" {
		fmt.Fprintf(&b, "%s 的理智检定：D100=%d/%d %s（%s）\n", mctx.Player.Name, d100, san, verdict, diceDetail)
	} else {
		fmt.Fprintf(&b, "%s 的理智检定：D100=%d/%d %s\n", mctx.Player.Name, d100, san, verdict)
	}
	if loss > 0 {
		fmt.Fprintf(&b, "理智 %d → %d（-%d）\n", san, sanNew, loss)
	} else {
		fmt.Fprintf(&b, "理智 %d → %d（无损失）\n", san, sanNew)
	}

	// 失败必然触发不自觉反应
	if !success {
		b.WriteString("不自觉反应：惊叫 / 掉落手中物品 / 失控移动 / 僵住 / 攻击（由守秘人指定其一）\n")
	}

	// 单次损失达标 → INT 检定决定是否 Shock
	shocked := false
	if loss >= shockThreshold {
		intRoll, intDetail := c7rRollD100(mctx, false)
		intSuccess := intRoll <= intel
		if intDetail != "" {
			intDetail = "（" + intDetail + "）"
		}
		if intSuccess {
			shocked = true
			rounds := c7rRollDie(mctx, 10)
			fmt.Fprintf(&b, "损失 ≥%d，INT 检定：D100=%d/%d 成功 → 陷入 Shock，持续 1D10=%d 轮\n", shockThreshold, intRoll, intel, rounds)
			b.WriteString("Shock 表现（五选一）：Fight 攻击 / Flight 逃离 / Freeze 僵住 / Faint 昏倒 / Fawn 讨好\n")
			b.WriteString("Shock 期间不再损失理智；结束后进入 Aftershock，直到当天结束都容易出现妄想\n")
		} else {
			fmt.Fprintf(&b, "损失 ≥%d，INT 检定：D100=%d/%d 失败 → 大脑把恐惧压了下去，未陷入 Shock\n", shockThreshold, intRoll, intel)
		}
	}

	// 神话来源的 Shock / Changed 带来克苏鲁神话技能增长
	if mythosSource && (shocked || newlyChanged) {
		gain := int64(1)
		if st.MythosEvents == 0 {
			gain = 5
		}
		st.MythosEvents++
		cm := c7rAttrInt(mctx, tmpl, "克苏鲁神话") + gain
		VarSetValueInt64(mctx, "克苏鲁神话", cm)
		if cm > 99 {
			cm = 99
		}
		fmt.Fprintf(&b, "神话冲击：克苏鲁神话 +%d（当前 %d，理智上限 %d）\n", gain, cm, 99-cm)
		if sanNew > 99-cm {
			sanNew = 99 - cm
			VarSetValueInt64(mctx, "理智", sanNew)
			fmt.Fprintf(&b, "理智上限下降到 %d，当前理智随之调整为 %d\n", 99-cm, sanNew)
		}
	}

	if newlyChanged {
		fmt.Fprintf(&b, "本篇章累计理智损失 %d/%d → 已 Changed：出现妄想、获得一个恐惧症、Shock 阈值提高到 10，且无法恢复幸运；休整一周并改写背景可恢复\n", st.ChapterLoss, changedThreshold)
	} else if !st.Changed {
		fmt.Fprintf(&b, "本篇章累计理智损失 %d/%d\n", st.ChapterLoss, changedThreshold)
	}

	if sanNew == 0 {
		b.WriteString("理智归零：调查员 Lost，不再是玩家角色\n")
	}

	c7rSaveState(mctx.Dice, mctx.Group.GroupID, mctx.Player.UserID, st)

	ReplyToSender(mctx, msg, strings.TrimRight(b.String(), "\n"))
	if mctx.Player.AutoSetNameTemplate != "" {
		_, _ = SetPlayerGroupCardByTemplate(mctx, mctx.Player.AutoSetNameTemplate)
	}
	return CmdExecuteResult{Matched: true, Solved: true}
}

// cmdC7RRealityCheck 现实检定：成功看破幻觉，失败 -1 理智并触发不自觉反应。
func cmdC7RRealityCheck(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
	if cmdArgs.IsArgEqual(1, "help") {
		return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
	}
	if !c7rRequireRule(ctx, msg) {
		return CmdExecuteResult{Matched: true, Solved: true}
	}

	mctx := GetCtxProxyFirst(ctx, cmdArgs)
	tmpl := cardRuleCheck(mctx, msg)
	if tmpl == nil {
		return CmdExecuteResult{Matched: true, Solved: true}
	}
	mctx.SystemTemplate = tmpl

	// .rv b / .rv --b：同伴安抚成功带来的奖励骰
	bonus := false
	for _, arg := range cmdArgs.Args {
		if arg == "b" || arg == "B" || arg == "--bonus" {
			bonus = true
		}
	}
	if cmdArgs.GetKwarg("b") != nil || cmdArgs.GetKwarg("bonus") != nil {
		bonus = true
	}

	san := c7rCurrentSan(mctx, tmpl)
	d100, detail := c7rRollD100(mctx, bonus)
	success := d100 <= san

	var b strings.Builder
	verdict := "失败"
	if success {
		verdict = "成功"
	}
	if detail != "" {
		fmt.Fprintf(&b, "%s 的现实检定：D100=%d/%d %s（%s）\n", mctx.Player.Name, d100, san, verdict, detail)
	} else {
		fmt.Fprintf(&b, "%s 的现实检定：D100=%d/%d %s\n", mctx.Player.Name, d100, san, verdict)
	}

	if success {
		b.WriteString("看破幻觉，守秘人须如实描述角色真正感知到的东西；在再次损失理智前，妄想不再生效\n")
	} else {
		sanNew := san - 1
		if sanNew < 0 {
			sanNew = 0
		}
		VarSetValueInt64(mctx, "理智", sanNew)
		fmt.Fprintf(&b, "理智 %d → %d（-1），幻觉未被识破\n", san, sanNew)
		b.WriteString("不自觉反应：惊叫 / 掉落手中物品 / 失控移动 / 僵住 / 攻击（由守秘人指定其一）\n")
		if sanNew == 0 {
			b.WriteString("理智归零：调查员 Lost，不再是玩家角色\n")
		}
	}

	ReplyToSender(mctx, msg, strings.TrimRight(b.String(), "\n"))
	return CmdExecuteResult{Matched: true, Solved: true}
}

// cmdC7RInitiative 圆桌式先攻：偶数向左手边，奇数向右手边。
func cmdC7RInitiative(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
	if cmdArgs.IsArgEqual(1, "help") {
		return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
	}
	if !c7rRequireRule(ctx, msg) {
		return CmdExecuteResult{Matched: true, Solved: true}
	}
	roll := c7rRollDie(ctx, 6)
	direction := "左手边"
	if roll%2 == 1 {
		direction = "右手边"
	}
	parity := "偶数"
	if roll%2 == 1 {
		parity = "奇数"
	}
	text := fmt.Sprintf("回合计序：1D6=%d（%s）→ 由发起攻击者开始，沿%s依次行动；本轮确定方向后，整场战斗沿用", roll, parity, direction)
	ReplyToSender(ctx, msg, text)
	return CmdExecuteResult{Matched: true, Solved: true}
}

// cmdC7RTakeFive 技能成长取五：不投骰，直接 +5。
func cmdC7RTakeFive(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
	if len(cmdArgs.Args) < 1 || cmdArgs.IsArgEqual(1, "help") {
		return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
	}
	if !c7rRequireRule(ctx, msg) {
		return CmdExecuteResult{Matched: true, Solved: true}
	}

	mctx := GetCtxProxyFirst(ctx, cmdArgs)
	tmpl := cardRuleCheck(mctx, msg)
	if tmpl == nil {
		return CmdExecuteResult{Matched: true, Solved: true}
	}
	mctx.SystemTemplate = tmpl

	skillName := strings.TrimSpace(strings.Join(cmdArgs.Args, " "))
	// @ 信息不参与技能名
	if idx := strings.Index(skillName, "@"); idx >= 0 {
		skillName = strings.TrimSpace(skillName[:idx])
	}
	key := tmpl.GetAlias(skillName)
	if key == "" || !c7rSkillExists(mctx, tmpl, key) {
		ReplyToSender(mctx, msg, fmt.Sprintf("未找到技能「%s」", skillName))
		return CmdExecuteResult{Matched: true, Solved: true}
	}

	old := c7rAttrInt(mctx, tmpl, key)
	if old >= 99 {
		ReplyToSender(mctx, msg, fmt.Sprintf("「%s」已经是 %d，无需再成长", key, old))
		return CmdExecuteResult{Matched: true, Solved: true}
	}
	newVal := old + 5
	if newVal > 99 {
		newVal = 99
	}
	VarSetValueInt64(mctx, key, newVal)
	ReplyToSender(mctx, msg, fmt.Sprintf("%s 的「%s」成长（取五）：%d → %d", mctx.Player.Name, key, old, newVal))
	return CmdExecuteResult{Matched: true, Solved: true}
}

// c7rSkillExists 判断技能是否属于当前模板（默认值、计算值或已存值）。
func c7rSkillExists(ctx *MsgContext, tmpl *GameSystemTemplate, key string) bool {
	if tmpl == nil {
		return false
	}
	if _, exists := tmpl.Attrs.Defaults[key]; exists {
		return true
	}
	if _, exists := tmpl.Attrs.DefaultsComputed[key]; exists {
		return true
	}
	attrs, err := ctx.Dice.AttrsManager.LoadByCtx(ctx)
	if err != nil || attrs == nil {
		return false
	}
	_, exists := attrs.LoadX(key)
	return exists
}

// cmdC7RHub .c7r 规则辅助入口。
func cmdC7RHub(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
	if !c7rIsGroupOnRule(ctx) {
		ReplyToSender(ctx, msg, "当前群不是 coc7r 规则，先用 .set coc7r 切换")
		return CmdExecuteResult{Matched: true, Solved: true}
	}

	sub := strings.ToLower(cmdArgs.GetArgN(1))
	switch sub {
	case "", "help":
		ReplyToSender(ctx, msg, c7rHelpText())
	case "status":
		ReplyToSender(ctx, msg, c7rStatusText(ctx, cmdArgs))
	case "derive":
		ReplyToSender(ctx, msg, c7rDeriveBases(ctx, cmdArgs))
	case "migrate":
		ReplyToSender(ctx, msg, c7rMigrateLegacy(ctx, cmdArgs))
	case "new":
		ReplyToSender(ctx, msg, c7rNewInvestigator(ctx, cmdArgs))
	case "min":
		ReplyToSender(ctx, msg, c7rSpecializationMinimums(ctx, cmdArgs))
	case "reset":
		reply := c7rResetState(ctx, cmdArgs)
		ReplyToSender(ctx, msg, reply)
	default:
		ReplyToSender(ctx, msg, c7rHelpText())
	}
	return CmdExecuteResult{Matched: true, Solved: true}
}

func c7rScHelpText() string {
	return strings.Join([]string{
		"理智检定（守秘人伴侣版）：",
		".sc <成功损失>/<失败损失> // 例：.sc 0/1D6、.sc 1/1D4+1",
		"  --cap=N   单次损失上限（用于“习惯恐惧”房规）",
		"  --half    损失减半",
		"  --mythos  本次冲击来自神话：Shock/Changed 时增加克苏鲁神话技能（首次 +5，之后 +1）",
		"失败必触发不自觉反应；单次损失 ≥5（Changed 后 ≥10）投 INT，成功则陷入 Shock 1D10 轮",
	}, "\n")
}

func c7rHelpText() string {
	return strings.Join([]string{
		"coc7r（守秘人伴侣）规则扩展：",
		".set coc7r // 切换到本规则（切回七版用 .set coc7）",
		".sc <成功损失>/<失败损失> // 新版理智判定，可加 --cap N、--half、--mythos",
		".rv [b] // 现实检定，b 表示同伴安抚成功带来的奖励骰",
		".ri // 圆桌式先攻，掷骰决定行动方向",
		".en5 <技能> // 技能成长取五，直接 +5",
		".c7r status // 查看本篇章理智账本",
		".c7r derive // 按属性推导技能基础值（可选规则）",
		".c7r new // 新版建卡流程与个人兴趣点",
		".c7r min // 按专精下限修正格斗/射击技能（半值规则）",
		".c7r migrate // 合并旧卡的取悦/话术/恐吓等技能",
		".c7r reset // 清空本篇章理智账本",
	}, "\n")
}

func c7rNewInvestigator(ctx *MsgContext, cmdArgs *CmdArgs) string {
	mctx := GetCtxProxyFirst(ctx, cmdArgs)
	tmpl := mctx.Group.GetCharTemplate(mctx.Dice)
	if tmpl == nil {
		return "当前群没有可用的规则模板"
	}
	edu := c7rAttrInt(mctx, tmpl, "教育")
	intel := c7rAttrInt(mctx, tmpl, "智力")
	app := c7rAttrInt(mctx, tmpl, "外貌")
	lines := []string{
		"新版建卡流程：",
		"1. 选一个职业，直接把职业列出的技能值抄到卡上（已含基础值，不用再算点数）",
		"2. 个人兴趣点 = 教育 + 智力，可分配到任意技能（克苏鲁神话除外），可加在职业技能上",
		"3. 想在职业内部调整，只能在职业列出的技能之间挪点，且不能低于基础值",
		"4. 信用评级可以直接用技能点或个人兴趣点调整",
		fmt.Sprintf("按当前属性：教育 %d + 智力 %d = 个人兴趣点 %d；外貌 %d → 安抚基础值 %d", edu, intel, edu+intel, app, app/5),
		"可选：.c7r derive 按属性推导全部技能基础值（守秘人伴侣 p23-25 的可选规则）",
	}
	return strings.Join(lines, "\n")
}

// c7rFightingSkills / c7rFirearmsSkills 对应书中“专精下限”可选规则。
var (
	c7rFightingSkills = []string{"斗殴", "斧", "链锯", "连枷", "绞索", "矛", "剑", "鞭"}
	c7rFirearmsSkills = []string{"射击:弓", "射击:手枪", "射击:重武器", "射击:火焰喷射器", "射击:机枪", "射击:步霰"}
)

func c7rSpecializationMinimums(ctx *MsgContext, cmdArgs *CmdArgs) string {
	mctx := GetCtxProxyFirst(ctx, cmdArgs)
	tmpl := mctx.Group.GetCharTemplate(mctx.Dice)
	if tmpl == nil {
		return "当前群没有可用的规则模板"
	}
	attrs, err := mctx.Dice.AttrsManager.LoadByCtx(mctx)
	if err != nil || attrs == nil {
		return "读不到角色卡"
	}

	lines := make([]string, 0, 4)
	apply := func(group []string, label string) {
		highest := int64(0)
		for _, name := range group {
			if v := c7rAttrInt(mctx, tmpl, name); v > highest {
				highest = v
			}
		}
		if highest <= 0 {
			return
		}
		minVal := highest / 2
		raised := make([]string, 0, len(group))
		for _, name := range group {
			key := tmpl.GetAlias(name)
			cur := c7rAttrInt(mctx, tmpl, name)
			if cur >= minVal {
				continue
			}
			attrs.Store(key, ds.NewIntVal(ds.IntType(minVal)))
			raised = append(raised, fmt.Sprintf("%s %d→%d", key, cur, minVal))
		}
		if len(raised) > 0 {
			lines = append(lines, fmt.Sprintf("%s（最高 %d，下限 %d）：%s", label, highest, minVal, strings.Join(raised, "、")))
		}
	}
	apply(c7rFightingSkills, "格斗专精")
	apply(c7rFirearmsSkills, "射击专精")

	if len(lines) == 0 {
		return "格斗/射击专精都已在半值下限之上，无需修正"
	}
	return "已按专精下限规则修正：\n" + strings.Join(lines, "\n")
}

func c7rStatusText(ctx *MsgContext, cmdArgs *CmdArgs) string {
	mctx := GetCtxProxyFirst(ctx, cmdArgs)
	tmpl := mctx.Group.GetCharTemplate(mctx.Dice)
	if tmpl == nil {
		return "当前群没有可用的规则模板"
	}
	san := c7rCurrentSan(mctx, tmpl)
	pow := c7rAttrInt(mctx, tmpl, "意志")
	cm := c7rAttrInt(mctx, tmpl, "克苏鲁神话")
	st := c7rLoadState(mctx.Dice, mctx.Group.GroupID, mctx.Player.UserID)
	threshold := pow / 5
	if threshold < 1 {
		threshold = 1
	}
	changed := "否"
	if st.Changed {
		changed = "是"
	}
	return fmt.Sprintf("%s 的 coc7r 状态：\n理智 %d/%d，意志 %d（Changed 阈值 %d）\n本篇章累计损失 %d\nChanged：%s\n神话冲击次数：%d",
		mctx.Player.Name, san, 99-cm, pow, threshold, st.ChapterLoss, changed, st.MythosEvents)
}

func c7rResetState(ctx *MsgContext, cmdArgs *CmdArgs) string {
	mctx := GetCtxProxyFirst(ctx, cmdArgs)
	c7rClearState(mctx.Dice, mctx.Group.GroupID, mctx.Player.UserID)
	return fmt.Sprintf("已清空 %s 在本群的篇章理智账本", mctx.Player.Name)
}

// c7rDerivedSkills 可选规则：技能基础值由属性推导（守秘人伴侣 p23-25）。
// 只写入卡上没有显式设置过的技能，避免覆盖玩家已分配的技能点。
var c7rDerivedSkills = map[string]func(ch map[string]int64) int64{
	"会计":       func(ch map[string]int64) int64 { return ch["教育"] / 10 },
	"表演":       func(ch map[string]int64) int64 { return maxI64(ch["智力"], ch["意志"]) / 10 },
	"动物驯养":     func(ch map[string]int64) int64 { return ch["意志"] / 10 },
	"估价":       func(ch map[string]int64) int64 { return ch["教育"] / 10 },
	"斧":        func(ch map[string]int64) int64 { return ch["力量"] / 5 },
	"射击:弓":     func(ch map[string]int64) int64 { return ch["敏捷"] / 5 },
	"斗殴":       func(ch map[string]int64) int64 { return maxI64(ch["敏捷"], ch["力量"]) / 2 },
	"链锯":       func(ch map[string]int64) int64 { return ch["力量"] / 5 },
	"攀爬":       func(ch map[string]int64) int64 { return maxI64(ch["敏捷"], ch["力量"]) / 2 },
	"计算机使用":    func(ch map[string]int64) int64 { return ch["教育"] / 10 },
	"乔装":       func(ch map[string]int64) int64 { return ch["智力"] / 10 },
	"汽车驾驶":     func(ch map[string]int64) int64 { return ch["敏捷"] / 2 },
	"电气维修":     func(ch map[string]int64) int64 { return ch["教育"] / 5 },
	"美术":       func(ch map[string]int64) int64 { return ch["敏捷"] / 10 },
	"急救":       func(ch map[string]int64) int64 { return ch["教育"] / 2 },
	"连枷":       func(ch map[string]int64) int64 { return ch["敏捷"] / 5 },
	"射击:火焰喷射器": func(ch map[string]int64) int64 { return ch["敏捷"] / 5 },
	"伪造":       func(ch map[string]int64) int64 { return maxI64(ch["敏捷"], ch["教育"]) / 10 },
	"绞索":       func(ch map[string]int64) int64 { return ch["敏捷"] / 5 },
	"射击:手枪":    func(ch map[string]int64) int64 { return ch["敏捷"] / 2 },
	"射击:重武器":   func(ch map[string]int64) int64 { return ch["力量"] / 5 },
	"历史":       func(ch map[string]int64) int64 { return ch["教育"] / 10 },
	"跳跃":       func(ch map[string]int64) int64 { return ch["敏捷"] / 2 },
	"法律":       func(ch map[string]int64) int64 { return ch["教育"] / 10 },
	"图书馆使用":    func(ch map[string]int64) int64 { return ch["教育"] / 2 },
	"聆听":       func(ch map[string]int64) int64 { return ch["智力"] / 2 },
	"射击:机枪":    func(ch map[string]int64) int64 { return ch["力量"] / 5 },
	"机械维修":     func(ch map[string]int64) int64 { return ch["智力"] / 5 },
	"博物学":      func(ch map[string]int64) int64 { return ch["智力"] / 5 },
	"导航":       func(ch map[string]int64) int64 { return ch["智力"] / 5 },
	"神秘学":      func(ch map[string]int64) int64 { return ch["教育"] / 10 },
	"说服":       func(ch map[string]int64) int64 { return ch["外貌"] / 5 },
	"摄影":       func(ch map[string]int64) int64 { return maxI64(ch["敏捷"], ch["教育"]) / 10 },
	"心理学":      func(ch map[string]int64) int64 { return ch["智力"] / 5 },
	"宗教学":      func(ch map[string]int64) int64 { return ch["教育"] / 5 },
	"骑术":       func(ch map[string]int64) int64 { return ch["敏捷"] / 10 },
	"射击:步霰":    func(ch map[string]int64) int64 { return ch["敏捷"] / 2 },
	"妙手":       func(ch map[string]int64) int64 { return ch["敏捷"] / 5 },
	"矛":        func(ch map[string]int64) int64 { return ch["力量"] / 2 },
	"侦查":       func(ch map[string]int64) int64 { return ch["智力"] / 2 },
	"潜行":       func(ch map[string]int64) int64 { return ch["敏捷"] / 2 },
	"剑":        func(ch map[string]int64) int64 { return ch["敏捷"] / 2 },
	"生存":       func(ch map[string]int64) int64 { return ch["教育"] / 5 },
	"游泳":       func(ch map[string]int64) int64 { return ch["敏捷"] / 2 },
	"投掷":       func(ch map[string]int64) int64 { return ch["敏捷"] / 2 },
	"追踪":       func(ch map[string]int64) int64 { return ch["智力"] / 5 },
	"鞭":        func(ch map[string]int64) int64 { return ch["敏捷"] / 5 },
}

func maxI64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func c7rDeriveBases(ctx *MsgContext, cmdArgs *CmdArgs) string {
	mctx := GetCtxProxyFirst(ctx, cmdArgs)
	tmpl := mctx.Group.GetCharTemplate(mctx.Dice)
	if tmpl == nil {
		return "当前群没有可用的规则模板"
	}
	ch := map[string]int64{
		"力量": c7rAttrInt(mctx, tmpl, "力量"),
		"敏捷": c7rAttrInt(mctx, tmpl, "敏捷"),
		"体质": c7rAttrInt(mctx, tmpl, "体质"),
		"体型": c7rAttrInt(mctx, tmpl, "体型"),
		"外貌": c7rAttrInt(mctx, tmpl, "外貌"),
		"智力": c7rAttrInt(mctx, tmpl, "智力"),
		"意志": c7rAttrInt(mctx, tmpl, "意志"),
		"教育": c7rAttrInt(mctx, tmpl, "教育"),
	}
	attrs, err := mctx.Dice.AttrsManager.LoadByCtx(mctx)
	if err != nil || attrs == nil {
		return "读不到角色卡"
	}

	applied := make([]string, 0, len(c7rDerivedSkills))
	for skill, fn := range c7rDerivedSkills {
		key := tmpl.GetAlias(skill)
		if _, exists := attrs.LoadX(key); exists {
			continue // 玩家已分配过技能点
		}
		val := fn(ch)
		if val < 0 {
			val = 0
		}
		// 与基础值相同就不用写到卡上，默认值本来就是它
		if def, ok := tmpl.Attrs.Defaults[key]; ok && int64(def) == val {
			continue
		}
		attrs.Store(key, ds.NewIntVal(ds.IntType(val)))
		applied = append(applied, fmt.Sprintf("%s %d", key, val))
	}
	if len(applied) == 0 {
		return "所有可用派生技能都已有数值，未做改动"
	}
	sort.Strings(applied)
	// 每行 5 项，避免一长串糊在一起
	lines := make([]string, 0, len(applied)/5+1)
	for i := 0; i < len(applied); i += 5 {
		end := i + 5
		if end > len(applied) {
			end = len(applied)
		}
		lines = append(lines, strings.Join(applied[i:end], "、"))
	}
	return fmt.Sprintf("已按属性写入 %d 项技能基础值：\n%s", len(applied), strings.Join(lines, "\n"))
}

// c7rMigratePairs 旧技能 → 新技能的合并关系。
var c7rMigratePairs = []struct {
	target string
	from   []string
}{
	{"说服", []string{"取悦", "话术", "恐吓"}},
	{"神秘学", []string{"学识"}},
	{"生存", []string{"沙漠", "海洋", "极地"}},
}

func c7rMigrateLegacy(ctx *MsgContext, cmdArgs *CmdArgs) string {
	mctx := GetCtxProxyFirst(ctx, cmdArgs)
	tmpl := mctx.Group.GetCharTemplate(mctx.Dice)
	if tmpl == nil {
		return "当前群没有可用的规则模板"
	}
	attrs, err := mctx.Dice.AttrsManager.LoadByCtx(mctx)
	if err != nil || attrs == nil {
		return "读不到角色卡"
	}

	lines := make([]string, 0, 4)
	for _, pair := range c7rMigratePairs {
		targetKey := tmpl.GetAlias(pair.target)
		best := int64(-1)
		if v, exists := attrs.LoadX(targetKey); exists && v != nil && v.TypeId == ds.VMTypeInt {
			best = int64(v.Value.(ds.IntType))
		}
		found := make([]string, 0, 3)
		for _, old := range pair.from {
			v, exists := attrs.LoadX(old)
			if !exists || v == nil || v.TypeId != ds.VMTypeInt {
				continue
			}
			oldVal := int64(v.Value.(ds.IntType))
			found = append(found, fmt.Sprintf("%s %d", old, oldVal))
			if oldVal > best {
				best = oldVal
			}
			attrs.Delete(old)
		}
		if len(found) == 0 || best < 0 {
			continue
		}
		attrs.Store(targetKey, ds.NewIntVal(ds.IntType(best)))
		lines = append(lines, fmt.Sprintf("%s ← %s，取最高值 %d", targetKey, strings.Join(found, "、"), best))
	}

	if v, exists := attrs.LoadX("射击:冲锋枪"); exists && v != nil && v.TypeId == ds.VMTypeInt {
		oldVal := int64(v.Value.(ds.IntType))
		lines = append(lines, fmt.Sprintf("射击:冲锋枪 %d 保留未动：新版由 射击:手枪 或 射击:步霰 取代，请自行决定并入哪一项后删除", oldVal))
	}

	if len(lines) == 0 {
		return "没有需要合并的旧技能"
	}
	return "旧卡迁移完成：\n" + strings.Join(lines, "\n")
}

// RegisterBuiltinExtCoc7R 注册 coc7r 扩展。
func RegisterBuiltinExtCoc7R(self *Dice) {
	helpSc := strings.Join([]string{
		".sc <成功损失>/<失败损失> // 例：.sc 0/1D6",
		"  --cap N   单次损失上限（用于“习惯恐惧”房规）",
		"  --half    损失减半",
		"  --mythos  本次冲击来自神话（Shock/Changed 时增加克苏鲁神话技能）",
	}, "\n")

	cmdSc := &CmdItemInfo{
		Name:          "sc",
		ShortHelp:     helpSc,
		Help:          "理智检定（守秘人伴侣版）:\n" + helpSc,
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			return cmdScCoc7R(ctx, msg, cmdArgs)
		},
	}

	cmdReality := &CmdItemInfo{
		Name:          "rv",
		ShortHelp:     ".rv [b] // 现实检定",
		Help:          "现实检定:\n.rv // 投 D100 对抗当前理智\n.rv b // 同伴安抚成功，获得一个奖励骰\n成功看破幻觉；失败扣 1 点理智并触发不自觉反应",
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			return cmdC7RRealityCheck(ctx, msg, cmdArgs)
		},
	}

	cmdInit := &CmdItemInfo{
		Name:      "ri",
		ShortHelp: ".ri // 圆桌式先攻",
		Help:      "圆桌式先攻:\n.ri // 掷 1D6，偶数沿左手边、奇数沿右手边依次行动\n整场战斗沿用同一方向",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			return cmdC7RInitiative(ctx, msg, cmdArgs)
		},
	}

	cmdTakeFive := &CmdItemInfo{
		Name:          "en5",
		ShortHelp:     ".en5 <技能> // 技能成长取五",
		Help:          "技能成长取五:\n.en5 <技能> // 不投骰，技能直接 +5（上限 99）",
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			return cmdC7RTakeFive(ctx, msg, cmdArgs)
		},
	}

	cmdHub := &CmdItemInfo{
		Name:      "c7r",
		ShortHelp: ".c7r help|status|derive|migrate|reset // coc7r 规则辅助",
		Help:      c7rHelpText(),
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			return cmdC7RHub(ctx, msg, cmdArgs)
		},
	}

	theExt := &ExtInfo{
		Name:       c7rExtName,
		Version:    "1.0.0",
		Brief:      "克苏鲁的呼唤第七版·守秘人伴侣（Keeper Companion）规则支持",
		AutoActive: true,
		Author:     "ShiaNyaa",
		Official:   true,
		OnLoad:     func() {},
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
		},
		GetDescText: GetExtensionDesc,
		CmdMap: CmdMapCls{
			"rv":  cmdReality,
			"ri":  cmdInit,
			"en5": cmdTakeFive,
			"c7r": cmdHub,
			"sc":  cmdSc,
		},
	}
	self.RegisterExtension(theExt)
}
