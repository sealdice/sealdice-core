package dice

func trpgTextTemplates() TextTemplateWithWeight {
	return TextTemplateWithWeight{
		"属性设置_删除": {
			{`{$t玩家}的如下属性被成功删除:{$t属性列表}，失败{$t失败数量}项`, 1},
		},
		"属性设置_清除": {
			{`{$t玩家}的属性数据已经清除，共计{$t数量}条`, 1},
		},
		"属性设置_增减_单项": {
			{"{$t属性}: {$t旧值} ➯ {$t新值} ({$t增加或扣除}{$t表达式文本}={$t变化量})", 1},
		},
		"属性设置_增减": {
			{"{$t玩家}的属性变化:\n{$t变更列表}\n{TRPG:属性设置_保存提醒}", 1},
		},
		"属性设置_列出": {
			{"{$t玩家}的个人属性为:\n{$t属性信息}", 1},
		},
		"属性设置_列出_未发现记录": {
			{`未发现属性记录`, 1},
		},
		"属性设置_列出_隐藏提示": {
			{`\n注：{$t数量}条属性因<{$t判定值}被隐藏`, 1},
		},
		"属性设置": {
			{`{$t玩家}的{$t规则模板}属性录入完成，本次录入了{$t有效数量}条数据`, 1},
		},
		"属性设置_保存提醒": {
			{`{ $t当前绑定角色 ? '[√] 已绑卡' : '' }`, 1},
		},
		"属性设置_拒绝": {
			{`属性修改失败：{$t错误原因}`, 1},
		},
	}
}

func trpgTextTemplateHelp() TextTemplateHelpGroup {
	return TextTemplateHelpGroup{
		"属性设置_删除": {
			SubType: ".st rm A B C",
		},
		"属性设置_清除": {
			SubType: ".st clr",
		},
		"属性设置_增减": {
			SubType: ".st hp+1",
		},
		"属性设置_增减_单项": {
			SubType: ".st hp+1 mp-1",
		},
		"属性设置_列出": {
			SubType: ".st show",
		},
		"属性设置_列出_未发现记录": {
			SubType: ".st show",
		},
		"属性设置_列出_隐藏提示": {
			SubType: ".st show",
		},
		"属性设置": {
			SubType:         ".st 力量70",
			Vars:            []string{"$t玩家", "$t规则模板", "$t有效数量", "$t数量", "$t同义词数量"},
			ExampleCommands: []string{".st 力量70"},
		},
		"属性设置_保存提醒": {
			SubType:         ".st hp+1",
			ExampleCommands: []string{".st hp70", ".st hp+1"},
		},
		"属性设置_拒绝": {
			SubType: ".st",
			Vars:    []string{"$t错误原因"},
		},
	}
}
