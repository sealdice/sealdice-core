package dice

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/samber/lo"
	ds "github.com/sealdice/dicescript"
)

type trpgStSnapshot struct {
	values           *ds.ValueMap
	sheetType        string
	lastModifiedTime int64
	lastUsedTime     int64
	isSaved          bool
	playerName       string
	playerUpdatedAt  int64
}

func cloneTrpgStValueMap(src *ds.ValueMap) *ds.ValueMap {
	dst := &ds.ValueMap{}
	if src == nil {
		return dst
	}
	src.Range(func(key string, value *ds.VMValue) bool {
		if value != nil {
			value = value.Clone()
		}
		dst.Store(key, value)
		return true
	})
	return dst
}

func takeTrpgStSnapshot(attrs *AttributesItem, player *GroupPlayerInfo) trpgStSnapshot {
	snapshot := trpgStSnapshot{
		values:           cloneTrpgStValueMap(attrs.valueMap),
		sheetType:        attrs.SheetType,
		lastModifiedTime: attrs.LastModifiedTime,
		lastUsedTime:     attrs.LastUsedTime,
		isSaved:          attrs.IsSaved,
	}
	if player != nil {
		snapshot.playerName = player.Name
		snapshot.playerUpdatedAt = player.UpdatedAtTime
	}
	return snapshot
}

func (snapshot trpgStSnapshot) restore(attrs *AttributesItem, player *GroupPlayerInfo) {
	attrs.valueMap = snapshot.values
	attrs.SheetType = snapshot.sheetType
	attrs.LastModifiedTime = snapshot.lastModifiedTime
	attrs.LastUsedTime = snapshot.lastUsedTime
	attrs.IsSaved = snapshot.isSaved
	if player != nil {
		player.Name = snapshot.playerName
		player.UpdatedAtTime = snapshot.playerUpdatedAt
	}
}

type trpgStHookRunner struct {
	dice       *Dice
	extensions []*ExtInfo
}

func newTrpgStHookRunner(ctx *MsgContext, system string) trpgStHookRunner {
	return trpgStHookRunner{dice: ctx.Dice, extensions: trpgStHookExtensions(ctx, system)}
}

func (runner trpgStHookRunner) run(selectHook func(*TrpgStHooks) func()) error {
	for _, ext := range runner.extensions {
		hook := selectHook(ext.TrpgStHooks)
		if hook == nil {
			continue
		}
		if err := invokeTrpgStHook(runner.dice, ext, hook); err != nil {
			return err
		}
	}
	return nil
}

func (runner trpgStHookRunner) beforeCommand(event *TrpgStCommandEvent) error {
	return runner.run(func(hooks *TrpgStHooks) func() {
		if hooks.BeforeCommand == nil {
			return nil
		}
		return func() { hooks.BeforeCommand(event) }
	})
}

func (runner trpgStHookRunner) afterParse(event *TrpgStCommandEvent) error {
	return runner.run(func(hooks *TrpgStHooks) func() {
		if hooks.AfterParse == nil {
			return nil
		}
		return func() { hooks.AfterParse(event) }
	})
}

func (runner trpgStHookRunner) beforeApply(event *TrpgStCommandEvent, operation *TrpgStOperation) error {
	return runner.run(func(hooks *TrpgStHooks) func() {
		if hooks.BeforeApply == nil {
			return nil
		}
		return func() { hooks.BeforeApply(event, operation) }
	})
}

func (runner trpgStHookRunner) afterEvaluate(event *TrpgStCommandEvent, operation *TrpgStOperation) error {
	return runner.run(func(hooks *TrpgStHooks) func() {
		if hooks.AfterEvaluate == nil {
			return nil
		}
		return func() { hooks.AfterEvaluate(event, operation) }
	})
}

func (runner trpgStHookRunner) beforeCommit(event *TrpgStCommandEvent) error {
	return runner.run(func(hooks *TrpgStHooks) func() {
		if hooks.BeforeCommit == nil {
			return nil
		}
		return func() { hooks.BeforeCommit(event) }
	})
}

func (runner trpgStHookRunner) afterCommit(event *TrpgStCommandEvent) {
	for _, ext := range runner.extensions {
		if ext.TrpgStHooks.AfterCommit == nil {
			continue
		}
		if err := invokeTrpgStHook(runner.dice, ext, func() { ext.TrpgStHooks.AfterCommit(event) }); err != nil {
			runner.dice.Logger.Error(err)
		}
	}
}

func (runner trpgStHookRunner) onShow(event *TrpgStRenderEvent) error {
	return runner.run(func(hooks *TrpgStHooks) func() {
		if hooks.OnShow == nil {
			return nil
		}
		return func() { hooks.OnShow(event) }
	})
}

func (runner trpgStHookRunner) onExport(event *TrpgStRenderEvent) error {
	return runner.run(func(hooks *TrpgStHooks) func() {
		if hooks.OnExport == nil {
			return nil
		}
		return func() { hooks.OnExport(event) }
	})
}

func trpgStFormatCard(mctx *MsgContext, tmpl *GameSystemTemplate, attrs *AttributesItem) {
	if tmpl != nil {
		cmdStCharFormat1(mctx, tmpl, attrs.valueMap)
	}
	attrs.SetSheetType(mctx.Group.System)
}

func trpgStCurrentValue(mctx *MsgContext, tmpl *GameSystemTemplate, attrs *AttributesItem, name string) (*ds.VMValue, bool) {
	current, _ := attrs.valueMap.Load(name)
	if current == nil {
		current, _, _, _ = tmpl.GetDefaultValue(name)
	}
	if current == nil {
		return ds.NewIntVal(0), true
	}
	if current.TypeId != ds.VMTypeComputedValue {
		return current, true
	}

	computedData, _ := current.ReadComputed()
	if computedData != nil && computedData.Attrs != nil {
		if base, ok := computedData.Attrs.Load("base"); ok {
			return base, false
		}
	}

	mctx.CreateVmIfNotExists()
	computed := current.ComputedExecute(mctx.vm, nil)
	if mctx.vm.Error != nil || computed == nil {
		mctx.vm.Error = nil
		return ds.NewIntVal(0), true
	}
	return computed, true
}

func trpgStReject(event *TrpgStCommandEvent, operation *TrpgStOperation, fallback error) error {
	if reason := trpgStRejectReason(event, operation); reason != "" {
		return errors.New(reason)
	}
	return fallback
}

func trpgStRunBeforeCommit(runner trpgStHookRunner, event *TrpgStCommandEvent) error {
	if err := runner.beforeCommit(event); err != nil {
		return err
	}
	if reason := trpgStRejectReason(event, nil); reason != "" {
		return errors.New(reason)
	}
	return nil
}

func trpgStReplyRejected(ctx *MsgContext, msg *Message, err error) {
	VarSetValueStr(ctx, "$t错误原因", err.Error())
	text := DiceFormatTmpl(ctx, "TRPG:属性设置_拒绝")
	if text == "" {
		text = "属性修改失败：" + err.Error()
	}
	ReplyToSender(ctx, msg, text)
}

func trpgStPlanSetAndMod(setItems, modItems []*stSetOrModInfoItem) []*TrpgStOperation {
	operations := make([]*TrpgStOperation, 0, len(setItems)+len(modItems))
	for _, item := range setItems {
		operations = append(operations, &TrpgStOperation{
			Kind:    "set",
			Name:    item.name,
			Operand: trpgStValueFromVM(item.value),
			Extra:   trpgStValueFromVM(item.extra),
		})
	}
	for _, item := range modItems {
		operations = append(operations, &TrpgStOperation{
			Kind:       "mod",
			Name:       item.name,
			Operator:   item.op,
			Expression: item.expr,
			Operand:    trpgStValueFromVM(item.value),
		})
	}
	return operations
}

func trpgStApplySet(
	mctx *MsgContext,
	tmpl *GameSystemTemplate,
	attrs *AttributesItem,
	runner trpgStHookRunner,
	event *TrpgStCommandEvent,
	operation *TrpgStOperation,
) (bool, error) {
	if err := runner.beforeApply(event, operation); err != nil {
		return false, err
	}
	if err := trpgStReject(event, operation, nil); err != nil {
		return false, err
	}
	if operation.Skip {
		return false, nil
	}

	operation.Name = tmpl.GetAlias(operation.Name)
	oldValue, _ := attrs.valueMap.Load(operation.Name)
	operation.OldValue = trpgStValueFromVM(oldValue)
	value, err := operation.Operand.toVMValue()
	if err != nil {
		return false, err
	}
	operation.ProposedValue = trpgStValueFromVM(value)
	if err = runner.afterEvaluate(event, operation); err != nil {
		return false, err
	}
	if err = trpgStReject(event, operation, nil); err != nil {
		return false, err
	}
	if operation.Skip {
		return false, nil
	}
	value, err = operation.ProposedValue.toVMValue()
	if err != nil {
		return false, err
	}

	def := tmpl.GetDefaultValueEx(mctx, operation.Name)
	if ds.ValueEqual(value, def, true) && oldValue == nil && operation.Extra == nil {
		return false, nil
	}
	attrs.Store(operation.Name, value)
	return true, nil
}

func trpgStApplyMod(
	mctx *MsgContext,
	tmpl *GameSystemTemplate,
	attrs *AttributesItem,
	runner trpgStHookRunner,
	event *TrpgStCommandEvent,
	operation *TrpgStOperation,
) (*ds.VMValue, *ds.VMValue, error) {
	if err := runner.beforeApply(event, operation); err != nil {
		return nil, nil, err
	}
	if err := trpgStReject(event, operation, nil); err != nil {
		return nil, nil, err
	}
	if operation.Skip {
		return nil, nil, nil
	}

	operation.Name = tmpl.GetAlias(operation.Name)
	oldValue, storeResult := trpgStCurrentValue(mctx, tmpl, attrs, operation.Name)
	operand, err := operation.Operand.toVMValue()
	if err != nil {
		return nil, nil, err
	}
	mctx.CreateVmIfNotExists()
	var proposed *ds.VMValue
	switch operation.Operator {
	case "+":
		proposed = oldValue.OpAdd(mctx.vm, operand)
	case "-", "-=":
		proposed = oldValue.OpSub(mctx.vm, operand)
	default:
		return nil, nil, fmt.Errorf("不支持的属性修改操作符: %s", operation.Operator)
	}
	if proposed == nil {
		proposed = oldValue
	}
	operation.OldValue = trpgStValueFromVM(oldValue)
	operation.ProposedValue = trpgStValueFromVM(proposed)
	if err = runner.afterEvaluate(event, operation); err != nil {
		return nil, nil, err
	}
	if err = trpgStReject(event, operation, nil); err != nil {
		return nil, nil, err
	}
	if operation.Skip {
		return nil, nil, nil
	}
	proposed, err = operation.ProposedValue.toVMValue()
	if err != nil {
		return nil, nil, err
	}
	if storeResult {
		attrs.Store(operation.Name, proposed)
	}
	return oldValue, proposed, nil
}

func trpgStTrySimplifiedCardName(ctx, mctx *MsgContext, tmpl *GameSystemTemplate, input string) string {
	re := regexp.MustCompile(`^(([^\s\-#]{1,25})([-#]))([^=\s\d(\[{\-+]+\d+)`)
	matches := re.FindStringSubmatch(input)
	if len(matches) == 0 {
		return input
	}

	flag := matches[3]
	name := matches[2]
	valueText := matches[4]
	isName := flag == "#"
	if !isName {
		isName = true
		result, _, err := DiceExprEvalBase(ctx, valueText, RollExtraFlags{})
		if err == nil && result.GetRestInput() == "" && result.IsCalculated() {
			isName = false
		}
		if isName {
			valueName := tmpl.GetAlias(name)
			if _, _, _, exists := tmpl.GetDefaultValueEx0(mctx, valueName); exists {
				isName = false
			}
		}
	}
	if !isName {
		return input
	}

	input = input[len(matches[1]):]
	VarSetValueStr(mctx, "$t旧昵称", fmt.Sprintf("<%s>", mctx.Player.Name))
	VarSetValueStr(mctx, "$t旧昵称_RAW", mctx.Player.Name)
	mctx.Player.Name = name
	VarSetValueStr(mctx, "$t玩家", fmt.Sprintf("<%s>", name))
	VarSetValueStr(mctx, "$t玩家_RAW", name)
	mctx.Player.UpdatedAtTime = time.Now().Unix()
	return input
}

func getCmdTrpgSt() *CmdItemInfo {
	help := "属性修改指令，支持分支指令如下:\n" +
		".st show // 展示个人属性\n" +
		".st show <属性1> <属性2> ... // 展示特定的属性数值\n" +
		".st show <数字> // 展示高于<数字>的属性，如.st show 30\n" +
		".st clr // 清除属性\n" +
		".st fmt // 强制转卡为当前规则(改变卡片类型，转换同义词)\n" +
		".st del <属性1> <属性2> ... // 删除属性，可多项，以空格间隔\n" +
		".st export // 导出\n" +
		".st help // 帮助\n" +
		".st <属性><值> // 例：.st 敏捷50 力量3d6*5\n" +
		".st &<属性>=<式子> // 例：.st &手枪=1d6\n" +
		".st <属性>±<表达式> // 例：.st 敏捷+2 hp+1d3"

	return &CmdItemInfo{
		Name:          "st",
		ShortHelp:     strings.TrimPrefix(help, "属性修改指令，支持分支指令如下:\n"),
		Help:          help,
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult { //nolint:gocyclo,maintidx
			cmdArgs.ChopPrefixToArgsWith("del", "rm", "show", "list", "export")
			value := cmdArgs.GetArgN(1)
			mctx := GetCtxProxyFirst(ctx, cmdArgs)
			attrs := lo.Must(ctx.Dice.AttrsManager.LoadByCtx(mctx))
			cardType := ReadCardType(mctx)
			tmpl := ctx.Group.GetCharTemplate(ctx.Dice)
			ctx.SystemTemplate = tmpl
			mctx.SystemTemplate = tmpl

			tmplShow := tmpl
			if cardType != tmplShow.Name {
				if cardTemplate, _ := ctx.Dice.GameSystemMap.Load(cardType); cardTemplate != nil {
					tmplShow = cardTemplate
				}
			}
			mctx.Eval(tmpl.InitScript, nil)
			if tmplShow != tmpl {
				mctx.Eval(tmplShow.InitScript, nil)
			}

			event := &TrpgStCommandEvent{
				Actor: ctx, Target: mctx, Message: msg, Args: cmdArgs, System: tmpl.Name,
			}
			runner := newTrpgStHookRunner(mctx, tmpl.Name)
			if err := runner.beforeCommand(event); err != nil {
				trpgStReplyRejected(mctx, msg, err)
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if reason := trpgStRejectReason(event, nil); reason != "" {
				trpgStReplyRejected(mctx, msg, errors.New(reason))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if event.Handled {
				if event.Reply != "" {
					ReplyToSender(mctx, msg, event.Reply)
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			switch value {
			case "help":
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}

			case "show", "list":
				pickItems, limit := cmdStGetPickItemAndLimit(tmplShow, cmdArgs)
				items, dropped, err := trpgStGetItemsForShow(mctx, tmplShow, pickItems, limit, runner, event)
				if err != nil {
					ReplyToSender(mctx, msg, err.Error())
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				itemsPerLine := tmplShow.Commands.St.Show.ItemsPerLine
				if itemsPerLine <= 1 {
					itemsPerLine = 4
				}
				var info strings.Builder
				for index, item := range items {
					info.WriteString(item)
					if (index+1)%itemsPerLine == 0 {
						info.WriteString("\n")
					} else {
						info.WriteString("\t")
					}
				}
				if info.Len() == 0 {
					info.WriteString(DiceFormatTmpl(mctx, "TRPG:属性设置_列出_未发现记录"))
				}
				if limit > 0 {
					VarSetValueInt64(mctx, "$t数量", int64(dropped))
					VarSetValueInt64(mctx, "$t判定值", limit)
					info.WriteString(DiceFormatTmpl(mctx, "TRPG:属性设置_列出_隐藏提示"))
				}
				VarSetValueStr(mctx, "$t属性信息", info.String())
				extra := ReadCardTypeEx(mctx, tmpl.Name)
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "TRPG:属性设置_列出")+extra)

			case "export":
				items, err := trpgStGetItemsForExport(mctx, tmpl, runner, event)
				if err != nil {
					ReplyToSender(mctx, msg, err.Error())
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				var info strings.Builder
				info.WriteString("导出结果：\n.st clr\n.st ")
				for _, item := range items {
					info.WriteString(item)
					info.WriteString(" ")
				}
				if playerName := DiceFormat(mctx, "{$t玩家_RAW}"); playerName != "" {
					info.WriteString("\n.nn ")
					info.WriteString(playerName)
				}
				out := info.String()
				if len(items) == 0 {
					out = DiceFormatTmpl(mctx, "TRPG:属性设置_列出_未发现记录")
				}
				ReplyToSender(mctx, msg, out)

			case "del", "rm":
				snapshot := takeTrpgStSnapshot(attrs, mctx.Player)
				for _, name := range cmdArgs.Args[1:] {
					event.Operations = append(event.Operations, &TrpgStOperation{Kind: "delete", Name: tmpl.GetAlias(name)})
				}
				if err := runner.afterParse(event); err != nil {
					snapshot.restore(attrs, mctx.Player)
					trpgStReplyRejected(mctx, msg, err)
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				var deleted, failed []string
				for _, operation := range event.Operations {
					if err := runner.beforeApply(event, operation); err != nil || trpgStRejectReason(event, operation) != "" {
						if err == nil {
							err = errors.New(trpgStRejectReason(event, operation))
						}
						snapshot.restore(attrs, mctx.Player)
						trpgStReplyRejected(mctx, msg, err)
						return CmdExecuteResult{Matched: true, Solved: true}
					}
					if operation.Skip {
						continue
					}
					operation.Name = tmpl.GetAlias(operation.Name)
					old, exists := attrs.LoadX(operation.Name)
					operation.OldValue = trpgStValueFromVM(old)
					if err := runner.afterEvaluate(event, operation); err != nil || trpgStRejectReason(event, operation) != "" {
						if err == nil {
							err = errors.New(trpgStRejectReason(event, operation))
						}
						snapshot.restore(attrs, mctx.Player)
						trpgStReplyRejected(mctx, msg, err)
						return CmdExecuteResult{Matched: true, Solved: true}
					}
					if operation.Skip {
						continue
					}
					if exists {
						deleted = append(deleted, operation.Name)
						attrs.Delete(operation.Name)
					} else {
						failed = append(failed, operation.Name)
					}
				}
				if err := trpgStRunBeforeCommit(runner, event); err != nil {
					snapshot.restore(attrs, mctx.Player)
					trpgStReplyRejected(mctx, msg, err)
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				runner.afterCommit(event)
				VarSetValueStr(mctx, "$t属性列表", strings.Join(deleted, " "))
				VarSetValueInt64(mctx, "$t失败数量", int64(len(failed)))
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "TRPG:属性设置_删除")+event.ReplySuffix)

			case "clr", "clear":
				snapshot := takeTrpgStSnapshot(attrs, mctx.Player)
				event.Operations = []*TrpgStOperation{{Kind: "clear"}}
				if err := runner.afterParse(event); err != nil {
					snapshot.restore(attrs, mctx.Player)
					trpgStReplyRejected(mctx, msg, err)
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				operation := event.Operations[0]
				if err := runner.beforeApply(event, operation); err != nil || trpgStRejectReason(event, operation) != "" {
					if err == nil {
						err = errors.New(trpgStRejectReason(event, operation))
					}
					snapshot.restore(attrs, mctx.Player)
					trpgStReplyRejected(mctx, msg, err)
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				if err := runner.afterEvaluate(event, operation); err != nil || trpgStRejectReason(event, operation) != "" {
					if err == nil {
						err = errors.New(trpgStRejectReason(event, operation))
					}
					snapshot.restore(attrs, mctx.Player)
					trpgStReplyRejected(mctx, msg, err)
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				num := 0
				if !operation.Skip {
					num = attrs.Clear()
					attrs.SetSheetType("")
				}
				if err := trpgStRunBeforeCommit(runner, event); err != nil {
					snapshot.restore(attrs, mctx.Player)
					trpgStReplyRejected(mctx, msg, err)
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				runner.afterCommit(event)
				VarSetValueInt64(mctx, "$t数量", int64(num))
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "TRPG:属性设置_清除")+event.ReplySuffix)

			case "fmt", "format":
				snapshot := takeTrpgStSnapshot(attrs, mctx.Player)
				event.Operations = []*TrpgStOperation{{Kind: "format"}}
				if err := runner.afterParse(event); err != nil {
					snapshot.restore(attrs, mctx.Player)
					trpgStReplyRejected(mctx, msg, err)
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				operation := event.Operations[0]
				if err := runner.beforeApply(event, operation); err != nil || trpgStRejectReason(event, operation) != "" {
					if err == nil {
						err = errors.New(trpgStRejectReason(event, operation))
					}
					snapshot.restore(attrs, mctx.Player)
					trpgStReplyRejected(mctx, msg, err)
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				if err := runner.afterEvaluate(event, operation); err != nil || trpgStRejectReason(event, operation) != "" {
					if err == nil {
						err = errors.New(trpgStRejectReason(event, operation))
					}
					snapshot.restore(attrs, mctx.Player)
					trpgStReplyRejected(mctx, msg, err)
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				if !operation.Skip {
					trpgStFormatCard(mctx, tmpl, attrs)
				}
				if err := trpgStRunBeforeCommit(runner, event); err != nil {
					snapshot.restore(attrs, mctx.Player)
					trpgStReplyRejected(mctx, msg, err)
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				runner.afterCommit(event)
				ReplyToSender(mctx, msg, "角色卡片类型被强制修改为: "+ctx.Group.System+event.ReplySuffix)

			default:
				if cardType != "" && cardType != mctx.Group.System {
					ReplyToSender(mctx, msg, fmt.Sprintf("阻止操作：当前卡规则为 %s，群规则为 %s。\n为避免损坏此人物卡，请先更换角色卡，或使用.st fmt强制转卡", cardType, mctx.Group.System))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				snapshot := takeTrpgStSnapshot(attrs, mctx.Player)
				trpgStFormatCard(mctx, tmpl, attrs)
				input := trpgStTrySimplifiedCardName(ctx, mctx, tmpl, cmdArgs.CleanArgs)
				_, setItems, modItems, err := cmdStReadOrMod(mctx, tmpl, input)
				if err != nil {
					snapshot.restore(attrs, mctx.Player)
					ctx.Dice.Logger.Info("trpg.st 格式错误: ", err.Error())
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}
				restInput := mctx.vm.RestInput
				event.Operations = trpgStPlanSetAndMod(setItems, modItems)
				if err = runner.afterParse(event); err != nil || trpgStRejectReason(event, nil) != "" {
					if err == nil {
						err = errors.New(trpgStRejectReason(event, nil))
					}
					snapshot.restore(attrs, mctx.Player)
					trpgStReplyRejected(mctx, msg, err)
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				var text string
				validSetCount := int64(0)
				setCount := int64(0)
				modCount := 0
				commandInfo := map[string]any{
					"cmd": "st", "rule": cardType, "pcName": mctx.Player.Name, "items": []any{},
				}
				var modTexts, appendedTexts []string
				characterName := attrs.Name
				for _, operation := range event.Operations {
					switch operation.Kind {
					case "set":
						setCount++
						stored, applyErr := trpgStApplySet(mctx, tmpl, attrs, runner, event, operation)
						if applyErr != nil {
							err = applyErr
							break
						}
						if !operation.Skip && (stored || operation.Extra != nil) {
							validSetCount++
						}
					case "mod":
						oldValue, newValue, applyErr := trpgStApplyMod(mctx, tmpl, attrs, runner, event, operation)
						if applyErr != nil {
							err = applyErr
							break
						}
						if oldValue == nil || newValue == nil {
							continue
						}
						modCount++
						isIncrease := operation.Operator == "+"
						commandInfo["items"] = append(commandInfo["items"].([]any), map[string]any{
							"type": "mod", "attr": operation.Name, "modExpr": operation.Expression,
							"valOld": oldValue, "valNew": newValue, "isInc": isIncrease, "op": operation.Operator,
						})
						VarSetValueStr(mctx, "$t属性", operation.Name)
						VarSetValue(mctx, "$t旧值", oldValue)
						VarSetValue(mctx, "$t新值", newValue)
						operand, _ := operation.Operand.toVMValue()
						VarSetValue(mctx, "$t变化量", operand)
						if isIncrease {
							VarSetValueStr(mctx, "$t增加或扣除", "增加")
						} else {
							VarSetValueStr(mctx, "$t增加或扣除", "扣除")
						}
						VarSetValueStr(mctx, "$t表达式文本", operation.Expression)
						VarSetValueStr(mctx, "$t当前绑定角色", characterName)
						modTexts = append(modTexts, DiceFormatTmpl(mctx, "TRPG:属性设置_增减_单项"))
						if operation.AppendText != "" {
							appendedTexts = append(appendedTexts, operation.AppendText)
						}
					default:
						err = fmt.Errorf("不支持的trpg.st操作类型: %s", operation.Kind)
					}
					if err != nil {
						break
					}
				}
				if err == nil {
					err = trpgStRunBeforeCommit(runner, event)
				}
				if err != nil {
					snapshot.restore(attrs, mctx.Player)
					trpgStReplyRejected(mctx, msg, err)
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				if setCount > 0 {
					VarSetValueInt64(mctx, "$t数量", setCount)
					VarSetValueInt64(mctx, "$t有效数量", validSetCount)
					VarSetValueInt64(mctx, "$t同义词数量", 0)
					text = DiceFormatTmpl(mctx, "TRPG:属性设置")
					SetCardType(mctx, tmpl.Name)
				}
				if modCount > 0 {
					VarSetValueStr(mctx, "$t变更列表", strings.Join(modTexts, "\n"))
					text = DiceFormatTmpl(mctx, "TRPG:属性设置_增减")
					if len(appendedTexts) > 0 {
						text = strings.TrimSpace(text) + strings.Join(appendedTexts, "\n")
					}
					ctx.CommandInfo = commandInfo
					attrs.SetModified()
					SetCardType(mctx, tmpl.Name)
				}
				if restInput != "" {
					text += "\n解析失败: " + restInput
				}
				if snapshot.playerName != mctx.Player.Name && mctx.Group != nil {
					mctx.Group.MarkDirty(mctx.Dice)
				}
				runner.afterCommit(event)
				ReplyToSender(mctx, msg, text+event.ReplySuffix)
			}

			if ctx.Player.AutoSetNameTemplate != "" {
				_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
}
