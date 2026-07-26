package api

import "sealdice-core/dice"

func cloneTextTemplateWithWeightDict(src dice.TextTemplateWithWeightDict) dice.TextTemplateWithWeightDict {
	if src == nil {
		return nil
	}

	dst := make(dice.TextTemplateWithWeightDict, len(src))
	for groupName, group := range src {
		dst[groupName] = cloneTextTemplateWithWeight(group)
	}
	return dst
}

func cloneTextTemplateWithWeight(src dice.TextTemplateWithWeight) dice.TextTemplateWithWeight {
	if src == nil {
		return nil
	}

	dst := make(dice.TextTemplateWithWeight, len(src))
	for keyName, items := range src {
		dst[keyName] = cloneTextTemplateItems(items)
	}
	return dst
}

func cloneTextTemplateItems(src []dice.TextTemplateItem) []dice.TextTemplateItem {
	if src == nil {
		return nil
	}

	dst := make([]dice.TextTemplateItem, len(src))
	for index, item := range src {
		dst[index] = append(dice.TextTemplateItem(nil), item...)
	}
	return dst
}

func cloneTextTemplateWithHelpDict(src dice.TextTemplateWithHelpDict) dice.TextTemplateWithHelpDict {
	if src == nil {
		return nil
	}

	dst := make(dice.TextTemplateWithHelpDict, len(src))
	for groupName, group := range src {
		dst[groupName] = cloneTextTemplateHelpGroup(group)
	}
	return dst
}

func cloneTextTemplateHelpGroup(src dice.TextTemplateHelpGroup) dice.TextTemplateHelpGroup {
	if src == nil {
		return nil
	}

	dst := make(dice.TextTemplateHelpGroup, len(src))
	for keyName, item := range src {
		if item == nil {
			dst[keyName] = nil
			continue
		}
		cloned := *item
		cloned.Filename = append([]string(nil), item.Filename...)
		cloned.Origin = cloneTextTemplateItems(item.Origin)
		cloned.Vars = append([]string(nil), item.Vars...)
		cloned.Commands = append([]string(nil), item.Commands...)
		cloned.ExampleCommands = append([]string(nil), item.ExampleCommands...)
		dst[keyName] = &cloned
	}
	return dst
}
