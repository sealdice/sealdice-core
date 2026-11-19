package dice

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	ds "github.com/sealdice/dicescript"
)

// coc7r（克苏鲁的呼唤 第七版·守秘人伴侣）规则扩展
//
// 本扩展只改写 .sc 的结算流程，并提供一个 .c7r 入口管理状态与可选规则开关：
//   - .sc  成功仍按规则书投损失；失败必触发不自觉反应；单次损失≥5（Changed 后≥10）
//     投 INT 决定是否 Shock；单篇累计损失达 POW/5 转为 Changed；理智归零为 Lost。
//   - .c7r 篇章账本、Shock 状态、可选规则开关、旧卡迁移。
//
// 现实检定、圆桌先攻、成长取五都不新增指令：
// 现实检定等价于 .sc 0/1（同伴安抚成功时 .sc b 0/1），成长取五等价于 .en <技能>+5，
// 圆桌先攻收在 .c7r ri 下面，避免占用顶层指令名。
//
// 判定、属性读写、房规索引沿用 coc7 扩展的既有实现，避免出现两套不一致的规则。

const (
	c7rExtName        = "coc7r"
	c7rStateKeyPrefix = "coc7r|state|"
	c7rRulesKeyPrefix = "coc7r|rules|"
)

// coc7rState 记录单个篇章内的理智账本，按 群 + 角色卡 维度保存。
type coc7rState struct {
	ChapterLoss  int64 `json:"chapterLoss"`  // 本篇章累计理智损失
	Changed      bool  `json:"changed"`      // 是否已 Changed
	MythosEvents int   `json:"mythosEvents"` // 神话来源的 Shock/Changed 次数，跨篇章保留
	Shock        bool  `json:"shock"`        // 是否处于 Shock（期间不再损失理智）
	ShockRounds  int64 `json:"shockRounds"`  // 触发时掷出的 1D10 轮数，仅供提示
}

// coc7rRules 记录单个群的可选规则开关。
// sanity 默认开启：重写的理智章节是本规则集的主干，关掉后 .sc 回到七版流程。
// mythos 默认关闭：神话冲击提升克苏鲁神话技能属于可选条款，只由 .sc --mythos 触发。
type coc7rRules struct {
	Sanity bool `json:"sanity"`
	Mythos bool `json:"mythos"`
}

func c7rDefaultRules() coc7rRules {
	return coc7rRules{Sanity: true, Mythos: false}
}

func c7rRulesKey(groupID string) string {
	return c7rRulesKeyPrefix + groupID
}

// c7rLocks 按账本键串行化理智的读改写，避免同一角色的并发检定互相覆盖。
var c7rLocks sync.Map // string -> *sync.Mutex

func c7rLock(key string) func() {
	v, _ := c7rLocks.LoadOrStore(key, &sync.Mutex{})
	mu, _ := v.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func c7rStateKey(groupID, cardID string) string {
	return c7rStateKeyPrefix + groupID + "|" + cardID
}

// c7rCardID 取当前绑定角色卡的 ID；读不到卡时退化为玩家 ID。
func c7rCardID(ctx *MsgContext) string {
	if ctx == nil {
		return ""
	}
	if ctx.Dice != nil {
		if attrs, err := ctx.Dice.AttrsManager.LoadByCtx(ctx); err == nil && attrs != nil && attrs.ID != "" {
			return attrs.ID
		}
	}
	if ctx.Player != nil {
		return ctx.Player.UserID
	}
	return ""
}

// c7rStateKeyFor 返回当前上下文对应的账本键。
func c7rStateKeyFor(ctx *MsgContext) string {
	if ctx == nil || ctx.Group == nil {
		return ""
	}
	return c7rStateKey(ctx.Group.GroupID, c7rCardID(ctx))
}

func c7rLoadState(d *Dice, key string) coc7rState {
	var st coc7rState
	if d == nil || key == "" {
		return st
	}
	ext := d.ExtFind(c7rExtName, false)
	if ext == nil {
		return st
	}
	raw, err := ext.StorageGet(key)
	if err != nil || raw == "" {
		return st
	}
	_ = json.Unmarshal([]byte(raw), &st)
	return st
}

func c7rSaveState(d *Dice, key string, st coc7rState) {
	if d == nil || key == "" {
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
	_ = ext.StorageSet(key, string(raw))
}

func c7rLoadRules(d *Dice, groupID string) coc7rRules {
	rules := c7rDefaultRules()
	if d == nil || groupID == "" {
		return rules
	}
	ext := d.ExtFind(c7rExtName, false)
	if ext == nil {
		return rules
	}
	raw, err := ext.StorageGet(c7rRulesKey(groupID))
	if err != nil || raw == "" {
		return rules
	}
	if err = json.Unmarshal([]byte(raw), &rules); err != nil {
		return c7rDefaultRules()
	}
	return rules
}

func c7rSaveRules(d *Dice, groupID string, rules coc7rRules) error {
	if d == nil || groupID == "" {
		return fmt.Errorf("缺少群信息")
	}
	ext := d.ExtFind(c7rExtName, false)
	if ext == nil {
		return fmt.Errorf("coc7r 扩展未加载")
	}
	raw, err := json.Marshal(rules)
	if err != nil {
		return err
	}
	return ext.StorageSet(c7rRulesKey(groupID), string(raw))
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
		maxSan := 99 - c7rAttrInt(ctx, tmpl, "克苏鲁神话")
		if maxSan < 0 {
			maxSan = 0
		}
		if pow > maxSan {
			pow = maxSan
		}
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

// c7rBPTokenRe 匹配 .sc 开头的奖惩骰标记：b、b2、p、p3 …
var c7rBPTokenRe = regexp.MustCompile(`^([bBpP])(\d{0,2})$`)

// c7rParseDicePrefix 解析表达式开头的奖励骰/惩罚骰标记，返回剩余文本。
// 支持 b/b2/p/p2 这类写法，与 .ra 保持一致；同时写 b 和 p 会互相抵消。
func c7rParseDicePrefix(text string) (bonus, penalty int, rest string) {
	rest = strings.TrimSpace(text)
	for rest != "" {
		token := rest
		if idx := strings.IndexAny(rest, " \t"); idx >= 0 {
			token = rest[:idx]
		}
		m := c7rBPTokenRe.FindStringSubmatch(token)
		if m == nil {
			break
		}
		count := 1
		if m[2] != "" {
			if n, err := strconv.Atoi(m[2]); err == nil {
				count = n
			}
		}
		if count < 1 {
			count = 1
		}
		// 上限只是防止误输入，规则书里最多也就两三颗
		if count > 9 {
			count = 9
		}
		if strings.EqualFold(m[1], "b") {
			bonus += count
		} else {
			penalty += count
		}
		rest = strings.TrimSpace(rest[len(token):])
	}
	return bonus, penalty, rest
}

// c7rRollD100WithDice 掷 D100，按 COC 规则结算奖励骰/惩罚骰。
// 奖励骰多掷 count 颗十位骰取小，惩罚骰取大；个位骰只掷一次。
func c7rRollD100WithDice(ctx *MsgContext, bonus, penalty int) (int64, string) {
	net := bonus - penalty
	if net == 0 {
		v, ok := c7rEvalInt(ctx, "1d100", RollExtraFlags{DisableBlock: true})
		if !ok || v < 1 || v > 100 {
			v = 1
		}
		return v, ""
	}

	count := net
	label := "奖励骰"
	if count < 0 {
		count = -count
		label = "惩罚骰"
	}
	if count > 9 {
		count = 9
	}

	units := c7rRollDie(ctx, 10)
	if units == 10 {
		units = 0
	}
	tens := make([]string, 0, count+1)
	pick := int64(-1)
	for i := 0; i <= count; i++ {
		digit := c7rRollDie(ctx, 10)
		if digit == 10 {
			digit = 0
		}
		tens = append(tens, strconv.FormatInt(digit, 10))
		val := digit*10 + units
		if val == 0 {
			val = 100
		}
		if pick < 0 || (net > 0 && val < pick) || (net < 0 && val > pick) {
			pick = val
		}
	}
	return pick, fmt.Sprintf("%s×%d：十位 %s，个位 %d", label, count, strings.Join(tens, "/"), units)
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

// c7rCanModifyTarget 只有本人或骰主可以改动他人的角色卡。
func c7rCanModifyTarget(ctx, mctx *MsgContext) bool {
	if ctx == nil || mctx == nil || ctx.Player == nil || mctx.Player == nil {
		return false
	}
	if ctx.Player.UserID == mctx.Player.UserID {
		return true
	}
	return ctx.PrivilegeLevel >= 100
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
	rules := c7rLoadRules(mctx.Dice, mctx.Group.GroupID)

	stateKey := c7rStateKeyFor(mctx)
	unlock := c7rLock(stateKey)
	defer unlock()

	var st coc7rState
	if rules.Sanity {
		st = c7rLoadState(mctx.Dice, stateKey)
		if st.Shock {
			msgText := fmt.Sprintf("%s 正处于 Shock，期间不再损失理智，本次不结算。\n用 .c7r shock clear 结束（安抚成功或 1D10 轮结束）", mctx.Player.Name)
			ReplyToSender(mctx, msg, msgText)
			return CmdExecuteResult{Matched: true, Solved: true}
		}
	}

	argCap := cmdArgs.GetKwarg("cap")
	argHalf := cmdArgs.GetKwarg("half") != nil
	mythosSource := rules.Mythos && (cmdArgs.GetKwarg("mythos") != nil || cmdArgs.GetKwarg("m") != nil)
	bonusDice, penaltyDice, argText := c7rParseDicePrefix(cmdArgs.CleanArgs)

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
	if argCap != nil {
		if strings.TrimSpace(argCap.Value) == "" {
			ReplyToSender(mctx, msg, "损失上限要写成 --cap=N，例如 .sc 0/1D6 --cap=3")
			return CmdExecuteResult{Matched: true, Solved: true}
		}
		if n, err := strconv.ParseInt(argCap.Value, 10, 64); err == nil && n > 0 {
			lossCap = n
		}
	}

	san := c7rCurrentSan(mctx, tmpl)
	pow := c7rAttrInt(mctx, tmpl, "意志")
	intel := c7rAttrInt(mctx, tmpl, "智力")

	d100, diceDetail := c7rRollD100WithDice(mctx, bonusDice, penaltyDice)
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

	// sanity 关闭时回到七版流程：只结算损失并给出疯狂提示，不记账、不判 Shock/Changed/Lost
	if !rules.Sanity {
		if sanNew == 0 {
			b.WriteString(DiceFormatTmpl(mctx, "COC:提示_永久疯狂") + "\n")
		} else if loss >= 5 {
			b.WriteString(DiceFormatTmpl(mctx, "COC:提示_临时疯狂") + "\n")
		}
		ReplyToSender(mctx, msg, strings.TrimRight(b.String(), "\n"))
		if mctx.Player.AutoSetNameTemplate != "" {
			_, _ = SetPlayerGroupCardByTemplate(mctx, mctx.Player.AutoSetNameTemplate)
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	}

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

	// 失败必触发；成功但仍有理智损失（1-4 点）同样触发
	if !success || loss >= 1 {
		b.WriteString("不自觉反应：惊叫 / 掉落手中物品 / 失控移动 / 僵住 / 攻击（由守秘人指定其一）\n")
	}

	// 单次损失达标 → INT 检定决定是否 Shock
	shocked := false
	if loss >= shockThreshold {
		intRoll, intDetail := c7rRollD100WithDice(mctx, 0, 0)
		intSuccess := intRoll <= intel
		if intDetail != "" {
			intDetail = "（" + intDetail + "）"
		}
		if intSuccess {
			shocked = true
			rounds := c7rRollDie(mctx, 10)
			st.Shock = true
			st.ShockRounds = rounds
			fmt.Fprintf(&b, "损失 ≥%d，INT 检定：D100=%d/%d 成功 → 陷入 Shock，持续 1D10=%d 轮（用 .c7r shock clear 结束）\n", shockThreshold, intRoll, intel, rounds)
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
		if cm > 99 {
			cm = 99
		}
		VarSetValueInt64(mctx, "克苏鲁神话", cm)
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

	c7rSaveState(mctx.Dice, stateKey, st)

	ReplyToSender(mctx, msg, strings.TrimRight(b.String(), "\n"))
	if mctx.Player.AutoSetNameTemplate != "" {
		_, _ = SetPlayerGroupCardByTemplate(mctx, mctx.Player.AutoSetNameTemplate)
	}
	return CmdExecuteResult{Matched: true, Solved: true}
}

// c7rInitiativeText 圆桌式先攻：偶数向左手边，奇数向右手边。
func c7rInitiativeText(ctx *MsgContext) string {
	roll := c7rRollDie(ctx, 6)
	direction := "左手边"
	if roll%2 == 1 {
		direction = "右手边"
	}
	parity := "偶数"
	if roll%2 == 1 {
		parity = "奇数"
	}
	return fmt.Sprintf("回合计序：1D6=%d（%s）→ 由发起攻击者开始，沿%s依次行动；本轮确定方向后，整场战斗沿用", roll, parity, direction)
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
	case "rule":
		ReplyToSender(ctx, msg, c7rRuleCommand(ctx, cmdArgs))
	case "ri":
		ReplyToSender(ctx, msg, c7rInitiativeText(ctx))
	case "status":
		ReplyToSender(ctx, msg, c7rStatusText(ctx, cmdArgs))
	case "migrate":
		ReplyToSender(ctx, msg, c7rMigrateLegacy(ctx, cmdArgs))
	case "reset":
		ReplyToSender(ctx, msg, c7rResetState(ctx, cmdArgs))
	case "recover":
		ReplyToSender(ctx, msg, c7rRecoverChanged(ctx, cmdArgs))
	case "shock":
		ReplyToSender(ctx, msg, c7rShockCommand(ctx, cmdArgs))
	default:
		ReplyToSender(ctx, msg, c7rHelpText())
	}
	return CmdExecuteResult{Matched: true, Solved: true}
}

func c7rRuleLine(name string, on bool, desc string) string {
	state := "关"
	if on {
		state = "开"
	}
	return fmt.Sprintf("%s：%s  %s", name, state, desc)
}

// c7rRuleCommand 查看或切换可选规则。
func c7rRuleCommand(ctx *MsgContext, cmdArgs *CmdArgs) string {
	rules := c7rLoadRules(ctx.Dice, ctx.Group.GroupID)
	name := strings.ToLower(strings.TrimSpace(cmdArgs.GetArgN(2)))

	if name == "" {
		return strings.Join([]string{
			"可选规则：",
			c7rRuleLine("sanity", rules.Sanity, "重写的理智流程：Shock / Changed / Lost"),
			c7rRuleLine("mythos", rules.Mythos, "神话来源的冲击提升克苏鲁神话技能（.sc --mythos）"),
			"切换：.c7r rule <sanity|mythos> on|off",
		}, "\n")
	}

	var target *bool
	switch name {
	case "sanity", "理智":
		target = &rules.Sanity
	case "mythos", "神话":
		target = &rules.Mythos
	default:
		return fmt.Sprintf("未知的可选规则「%s」，可用：sanity、mythos", name)
	}

	action := strings.ToLower(strings.TrimSpace(cmdArgs.GetArgN(3)))
	switch action {
	case "on", "enable", "开", "启用":
		*target = true
	case "off", "disable", "关", "关闭":
		*target = false
	case "":
		state := "关"
		if *target {
			state = "开"
		}
		return fmt.Sprintf("可选规则 %s 当前为：%s", name, state)
	default:
		return fmt.Sprintf("用法：.c7r rule %s on|off", name)
	}

	if err := c7rSaveRules(ctx.Dice, ctx.Group.GroupID, rules); err != nil {
		return "保存可选规则失败：" + err.Error()
	}
	state := "关闭"
	if *target {
		state = "开启"
	}
	return fmt.Sprintf("已%s可选规则 %s", state, name)
}

func c7rScHelpText() string {
	return strings.Join([]string{
		"理智检定（守秘人伴侣版）：",
		".sc <成功损失>/<失败损失> // 例：.sc 0/1D6、.sc 1/1D4+1",
		".sc b 0/1 // 奖励骰，现实检定与同伴安抚成功都用这个；多个写 b2、b3",
		".sc p 0/1 // 惩罚骰，多个写 p2、p3；b 与 p 同时写会互相抵消",
		"  --cap=N   单次损失上限（用于“习惯恐惧”房规）",
		"  --half    损失减半",
		"  --mythos  本次冲击来自神话：Shock/Changed 时增加克苏鲁神话技能（需开启 mythos 可选规则）",
		"失败必触发不自觉反应；单次损失 ≥5（Changed 后 ≥10）投 INT，成功则陷入 Shock 1D10 轮",
		"关掉 sanity 可选规则后，.sc 回到七版流程：只结算损失并给出疯狂提示",
	}, "\n")
}

func c7rHelpText() string {
	return strings.Join([]string{
		"coc7r（守秘人伴侣）规则：",
		".set coc7r // 切换到本规则（切回七版用 .set coc7）",
		".sc <成功损失>/<失败损失> // 新版理智判定，可加 --cap=N、--half、--mythos",
		".c7r rule [名字] [on|off] // 查看或切换可选规则（sanity / mythos）",
		".c7r status [@人] // 篇章账本：累计损失、Changed、Shock、神话次数",
		".c7r ri // 圆桌式先攻：掷 1D6，偶数沿左手边、奇数沿右手边，整场战斗沿用",
		".c7r migrate [@人] // 合并旧卡的取悦/话术/恐吓等技能",
		".c7r reset [@人] // 开新篇章：清空篇章损失与 Shock（Changed 保留）",
		".c7r recover [@人] // 解除 Changed（需休整一周并改写背景）",
		".c7r shock [clear] // 查看或解除 Shock 状态",
		"",
		"建卡：职业列出的技能值直接抄到卡上（已含基础值）；个人兴趣点 = 教育 + 智力；",
		"宗教学按专精记录，多个信仰写成「宗教学(基督教)」这类独立键。",
		"现实检定用 .sc 0/1，同伴安抚成功写 .sc b 0/1（多颗奖励骰写 b2），失败由守秘人描述不自觉反应。",
		"成长取五用 .en <技能>+5（.en 默认成长量是 1d10，写死就是固定值），专精技能写全名，如 .en 射击:手枪+5。",
	}, "\n")
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
	st := c7rLoadState(mctx.Dice, c7rStateKeyFor(mctx))
	rules := c7rLoadRules(mctx.Dice, mctx.Group.GroupID)
	threshold := pow / 5
	if threshold < 1 {
		threshold = 1
	}
	changed := "否"
	if st.Changed {
		changed = "是"
	}
	shock := "否"
	if st.Shock {
		shock = fmt.Sprintf("是（约 %d 轮）", st.ShockRounds)
	}
	stateText := func(on bool) string {
		if on {
			return "开"
		}
		return "关"
	}
	ruleText := fmt.Sprintf("sanity %s、mythos %s", stateText(rules.Sanity), stateText(rules.Mythos))
	return fmt.Sprintf("%s 的 coc7r 状态：\n理智 %d/%d，意志 %d（Changed 阈值 %d）\n本篇章累计损失 %d\nChanged：%s\nShock：%s\n神话冲击次数：%d\n可选规则：%s",
		mctx.Player.Name, san, 99-cm, pow, threshold, st.ChapterLoss, changed, shock, st.MythosEvents, ruleText)
}

func c7rResetState(ctx *MsgContext, cmdArgs *CmdArgs) string {
	mctx := GetCtxProxyFirst(ctx, cmdArgs)
	if !c7rCanModifyTarget(ctx, mctx) {
		return "只能重置自己的篇章账本，或由骰主操作"
	}
	key := c7rStateKeyFor(mctx)
	st := c7rLoadState(mctx.Dice, key)
	st.ChapterLoss = 0
	st.Shock = false
	st.ShockRounds = 0
	c7rSaveState(mctx.Dice, key, st)
	changedTip := ""
	if st.Changed {
		changedTip = "；Changed 仍然保留，恢复要用 .c7r recover"
	}
	return fmt.Sprintf("已清空 %s 的篇章损失%s。神话冲击次数保留（%d 次），避免下次重复获得首次 +5",
		mctx.Player.Name, changedTip, st.MythosEvents)
}

// c7rRecoverChanged 恢复 Changed：书里要求休整一周并改写背景，这里只负责改状态。
func c7rRecoverChanged(ctx *MsgContext, cmdArgs *CmdArgs) string {
	mctx := GetCtxProxyFirst(ctx, cmdArgs)
	if !c7rCanModifyTarget(ctx, mctx) {
		return "只能恢复自己的角色，或由骰主操作"
	}
	key := c7rStateKeyFor(mctx)
	st := c7rLoadState(mctx.Dice, key)
	if !st.Changed {
		return fmt.Sprintf("%s 当前不是 Changed，无需恢复", mctx.Player.Name)
	}
	st.Changed = false
	st.Shock = false
	st.ShockRounds = 0
	c7rSaveState(mctx.Dice, key, st)
	return fmt.Sprintf("已解除 %s 的 Changed。前提是至少休整一周，并改写一条背景元素（标为 locked）", mctx.Player.Name)
}

// c7rShockCommand 查看或解除 Shock 状态。
func c7rShockCommand(ctx *MsgContext, cmdArgs *CmdArgs) string {
	mctx := GetCtxProxyFirst(ctx, cmdArgs)
	key := c7rStateKeyFor(mctx)
	st := c7rLoadState(mctx.Dice, key)
	sub := strings.ToLower(cmdArgs.GetArgN(2))

	switch sub {
	case "clear", "end", "off":
		if !c7rCanModifyTarget(ctx, mctx) {
			return "只能解除自己的 Shock，或由骰主操作"
		}
		if !st.Shock {
			return "当前不在 Shock"
		}
		st.Shock = false
		st.ShockRounds = 0
		c7rSaveState(mctx.Dice, key, st)
		return fmt.Sprintf("已解除 %s 的 Shock", mctx.Player.Name)
	default:
		if !st.Shock {
			return fmt.Sprintf("%s 当前不在 Shock", mctx.Player.Name)
		}
		return fmt.Sprintf("%s 正处于 Shock，触发时掷出 %d 轮；期间不损失理智。安抚成功或轮数结束后用 .c7r shock clear 解除",
			mctx.Player.Name, st.ShockRounds)
	}
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
	if !c7rCanModifyTarget(ctx, mctx) {
		return "只能迁移自己的角色卡，或由骰主操作"
	}
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
		// 目标技能不会低于模板基础值，避免合并后反而降级
		def := int64(tmpl.Attrs.Defaults[targetKey])
		best := def
		if v, exists := attrs.LoadX(targetKey); exists && v != nil && v.TypeId == ds.VMTypeInt {
			if cur := int64(v.Value.(ds.IntType)); cur > best {
				best = cur
			}
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
		if best > def {
			attrs.Store(targetKey, ds.NewIntVal(ds.IntType(best)))
		} else {
			attrs.Delete(targetKey)
		}
		lines = append(lines, fmt.Sprintf("%s ← %s，取最高值 %d（不低于基础值 %d）", targetKey, strings.Join(found, "、"), best, def))
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
		".sc b 0/1 // 奖励骰（b2、b3 为多颗）；惩罚骰用 p、p2",
		"  --cap N   单次损失上限（用于“习惯恐惧”房规）",
		"  --half    损失减半",
		"  --mythos  本次冲击来自神话（需开启 mythos 可选规则）",
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

	cmdHub := &CmdItemInfo{
		Name:      "c7r",
		ShortHelp: ".c7r help|rule|status|ri|migrate|reset // coc7r 规则辅助",
		Help:      c7rHelpText(),
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			return cmdC7RHub(ctx, msg, cmdArgs)
		},
	}

	theExt := &ExtInfo{
		Name:       c7rExtName,
		Version:    "1.1.0",
		Brief:      "克苏鲁的呼唤第七版·守秘人伴侣（Keeper Companion）规则支持",
		AutoActive: true,
		Author:     "ShiaNyaa",
		Official:   true,
		OnLoad:     func() {},
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
		},
		GetDescText: GetExtensionDesc,
		CmdMap: CmdMapCls{
			"c7r": cmdHub,
			// .sc 的入口在 coc7 扩展里按 group.System 分流；这里保留一份，
			// 以便 coc7 扩展被单独关闭时 coc7r 群仍可用。
			"sc": cmdSc,
		},
	}
	self.RegisterExtension(theExt)
}
