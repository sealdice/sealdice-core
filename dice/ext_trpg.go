package dice

// RegisterBuiltinExtTrpg registers opt-in, rule-neutral TRPG commands.
//
// The extension deliberately stays inactive until a game-system template lists
// it in relatedExt. Existing templates therefore continue to resolve coc7.st or
// dnd5e.st exactly as before, while new templates can explicitly select this
// implementation instead.
func RegisterBuiltinExtTrpg(d *Dice) {
	d.RegisterExtension(&ExtInfo{
		Name:        "trpg",
		Version:     "1.0.0",
		Brief:       "通用TRPG规则与人物卡指令",
		Author:      "SealDice-Team",
		AutoActive:  false,
		Official:    true,
		GetDescText: GetExtensionDesc,
		CmdMap: CmdMapCls{
			"st": getCmdStBase(CmdStOverrideInfo{}),
		},
	})
}
