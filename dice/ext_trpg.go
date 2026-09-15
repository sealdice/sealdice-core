package dice

// RegisterBuiltinExtTrpg registers rule-neutral TRPG commands.
func RegisterBuiltinExtTrpg(d *Dice) {
	d.RegisterExtension(&ExtInfo{
		Name:        "trpg",
		Version:     "1.0.0",
		Brief:       "通用TRPG规则与人物卡指令",
		Author:      "SealDice-Team",
		AutoActive:  true,
		Official:    true,
		GetDescText: GetExtensionDesc,
		CmdMap: CmdMapCls{
			"st":   getCmdTrpgSt(),
			"sn":   getCmdTrpgSn(d),
			"team": cmdTeam,
		},
	})
}
