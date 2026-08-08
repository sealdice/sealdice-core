package dice

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	ds "github.com/sealdice/dicescript"
)

func trpgStGetItemsForShow(
	mctx *MsgContext,
	tmpl *GameSystemTemplate,
	pickItems map[string]int,
	limit int64,
	runner trpgStHookRunner,
	command *TrpgStCommandEvent,
) ([]string, int, error) {
	usePickItem := len(pickItems) > 0
	useLimit := limit > 0
	dropped := 0
	items := []string{}
	topNum, attrKeys := cmdStSortNamesByTmpl(mctx, tmpl, pickItems, limit, false)

	for index, key := range attrKeys {
		if !usePickItem && strings.HasPrefix(key, "$") {
			continue
		}
		if usePickItem {
			if _, ok := pickItems[key]; !ok {
				continue
			}
		}

		mctx.CreateVmIfNotExists()
		mctx.vm.Error = nil
		baseKey := key
		displayKey, err := tmpl.GetShowKeyAs(mctx, key)
		if err != nil {
			return nil, 0, errors.New("模板卡异常(key), 属性: " + key + "\n报错: " + err.Error())
		}
		value, err := tmpl.GetShowValueAs(mctx, baseKey)
		if err != nil {
			return nil, 0, errors.New("模板卡异常(value), 属性: " + displayKey + "\n报错: " + err.Error())
		}

		if index >= topNum && useLimit {
			compareValue := ds.IntType(0)
			if value.TypeId == ds.VMTypeInt {
				compareValue = value.MustReadInt()
			} else if value.TypeId == ds.VMTypeString {
				if parsed, parseErr := strconv.ParseInt(value.ToString(), 10, 64); parseErr == nil {
					compareValue = ds.IntType(parsed)
				}
			}
			if int64(compareValue) < limit {
				dropped++
				continue
			}
		}

		render := &TrpgStRenderEvent{
			Command: command,
			Name:    baseKey,
			Value:   trpgStValueFromVM(value),
			Text:    fmt.Sprintf("%s:%s", displayKey, value.ToString()),
		}
		if err = runner.onShow(render); err != nil {
			return nil, 0, err
		}
		if reason := trpgStRejectReason(command, nil); reason != "" {
			return nil, 0, errors.New(reason)
		}
		if !render.Skip && render.Text != "" {
			items = append(items, render.Text)
		}
	}
	return items, dropped, nil
}

func trpgStGetItemsForExport(
	mctx *MsgContext,
	tmpl *GameSystemTemplate,
	runner trpgStHookRunner,
	command *TrpgStCommandEvent,
) ([]string, error) {
	items := []string{}
	_, attrKeys := cmdStSortNamesByTmpl(mctx, tmpl, map[string]int{}, 0, true)
	for _, key := range attrKeys {
		if strings.HasPrefix(key, "$") {
			continue
		}
		value, err := tmpl.GetRealValue(mctx, key)
		if err != nil {
			return nil, errors.New("模板卡异常, 属性: " + key)
		}

		text := ""
		if value.TypeId == ds.VMTypeComputedValue {
			valueText := value.ToString()
			if len(valueText) >= 4 {
				text = fmt.Sprintf("&%s:%s", key, valueText[2:len(valueText)-1])
			}
		} else {
			text = fmt.Sprintf("%s:%s", key, value.ToRepr())
		}
		render := &TrpgStRenderEvent{
			Command: command,
			Name:    key,
			Value:   trpgStValueFromVM(value),
			Text:    text,
		}
		if err = runner.onExport(render); err != nil {
			return nil, err
		}
		if reason := trpgStRejectReason(command, nil); reason != "" {
			return nil, errors.New(reason)
		}
		if !render.Skip && render.Text != "" {
			items = append(items, render.Text)
		}
	}
	return items, nil
}
