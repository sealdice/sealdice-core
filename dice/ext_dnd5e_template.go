package dice

var _dnd5eTmpl = &GameSystemTemplate{
	Name:        "dnd5e",
	FullName:    "龙与地下城5E",
	Authors:     []string{"木落"},
	Version:     "1.0.0",
	UpdatedTime: "20240615",
	TemplateVer: "1.0",

	SetConfig: SetConfig{
		DiceExpr:   "d20",
		Keys:       []string{"dnd", "dnd5e"},
		EnableTip:  "已切换至20面骰，并自动开启dnd5e扩展",
		RelatedExt: []string{"dnd5e"},
	},

	NameTemplate: map[string]NameTemplateItem{
		"dnd": {
			Template: "{$t玩家_RAW} HP{hp}/{hpmax} AC{ac} DC{dc} PP{pp}",
			HelpText: "自动设置dnd名片",
		},
	},

	PreloadCode: "func skillShowAs(val) {" +
		"  return `{val}{&val.base ? `[{&val.base}]`}`" +
		"}" +
		"func skillShowAsKey(key, val) {" +
		"  return `{key}{&val.factor ? '*'+ (&val.factor == 1 ? '' : str(&val.factor)) }`" +
		"}" + `
func pbCalc(base, factor, ab) {
	return base + (ab??0)/2 - 5 + floor((熟练??0) * (factor??0));
}
`,

	AttrConfig: AttrConfig{
		Top:    []string{"力量", "敏捷", "体质", "体型", "魅力", "智力", "感知", "hp", "ac", "熟练"},
		SortBy: "Name",
		Ignores: []string{
			"DSS", "DSF", // 死亡豁免的两个标记
			"hpmax",
		},
		ShowAs: map[string]string{
			"hp": "{hp}/{hpmax}",

			"运动": "{skillShowAs(&运动)}",

			"体操": "{skillShowAs(&体操)}",
			"巧手": "{skillShowAs(&巧手)}",
			"隐匿": "{skillShowAs(&隐匿)}",

			"调查": "{skillShowAs(&调查)}",
			"奥秘": "{skillShowAs(&奥秘)}",
			"历史": "{skillShowAs(&历史)}",
			"自然": "{skillShowAs(&自然)}",
			"宗教": "{skillShowAs(&宗教)}",

			"察觉": "{skillShowAs(&察觉)}",
			"洞悉": "{skillShowAs(&洞悉)}",
			"驯兽": "{skillShowAs(&驯兽)}",
			"医药": "{skillShowAs(&医药)}",
			"求生": "{skillShowAs(&求生)}",

			"游说": "{skillShowAs(&游说)}",
			"欺瞒": "{skillShowAs(&欺瞒)}",
			"威吓": "{skillShowAs(&威吓)}",
			"表演": "{skillShowAs(&表演)}",
		},
		ShowAsKey: map[string]string{
			"运动": "{skillShowAsKey('运动', &运动)}",

			"体操": "{skillShowAsKey('体操', &体操)}",
			"巧手": "{skillShowAsKey('巧手', &巧手)}",
			"隐匿": "{skillShowAsKey('隐匿', &隐匿)}",

			"调查": "{skillShowAsKey('调查', &调查)}",
			"奥秘": "{skillShowAsKey('奥秘', &奥秘)}",
			"历史": "{skillShowAsKey('历史', &历史)}",
			"自然": "{skillShowAsKey('自然', &自然)}",
			"宗教": "{skillShowAsKey('宗教', &宗教)}",

			"察觉": "{skillShowAsKey('察觉', &察觉)}",
			"洞悉": "{skillShowAsKey('洞悉', &洞悉)}",
			"驯兽": "{skillShowAsKey('驯兽', &驯兽)}",
			"医药": "{skillShowAsKey('医药', &医药)}",
			"求生": "{skillShowAsKey('求生', &求生)}",

			"游说": "{skillShowAsKey('游说', &游说)}",
			"欺瞒": "{skillShowAsKey('欺瞒', &欺瞒)}",
			"威吓": "{skillShowAsKey('威吓', &威吓)}",
			"表演": "{skillShowAsKey('表演', &表演)}",
		},
	},

	Defaults: map[string]int64{
		"熟练": 2,
	},
	DefaultsComputed: map[string]string{},

	Alias: map[string][]string{
		"力量": {"str", "Strength"},
		"敏捷": {"dex", "Dexterity"},
		"体质": {"con", "Constitution", "體質", "體魄", "体魄"},
		"智力": {"int", "Intelligence"},
		"感知": {"wis", "Wisdom"},
		"魅力": {"cha", "Charisma"},

		"ac":    {"AC", "护甲等级", "护甲值", "护甲", "護甲等級", "護甲值", "護甲", "装甲", "裝甲"},
		"hp":    {"HP", "生命值", "生命", "血量", "体力", "體力", "耐久值"},
		"hpmax": {"HPMAX", "生命值上限", "生命上限", "血量上限", "耐久上限"},
		"dc":    {"DC", "难度等级", "法术豁免", "難度等級", "法術豁免"},
		"hd":    {"HD", "生命骰"},
		"pp":    {"PP", "被动察觉", "被动感知", "被動察覺", "被动感知", "PW"},

		"熟练": {"熟练加值", "熟練", "熟練加值"},
		"体型": {"siz", "size", "體型", "体型", "体形", "體形"},

		// 技能
		"运动": {"Athletics", "運動"},

		"体操": {"Acrobatics", "杂技", "特技", "體操", "雜技", "特技動作", "特技动作"},
		"巧手": {"Sleight of Hand", "上手把戲", "上手把戏"},
		"隐匿": {"Stealth", "隱匿", "潜行", "潛行"},

		"调查": {"Investigation", "調查"},
		"奥秘": {"Arcana", "奧秘"},
		"历史": {"History", "歷史"},
		"自然": {"Nature"},
		"宗教": {"Religion"},

		"察觉": {"Perception", "察覺", "觉察", "覺察"},
		"洞悉": {"Insight", "洞察", "察言觀色", "察言观色"},
		"驯兽": {"Animal Handling", "馴獸", "驯养", "馴養", "動物馴服", "動物馴養", "动物驯服", "动物驯养"},
		"医药": {"Medicine", "醫藥", "医疗", "醫療"},
		"求生": {"Survival", "生存"},

		"游说": {"Persuasion", "说服", "话术", "遊說", "說服", "話術"},
		"欺瞒": {"Deception", "唬骗", "欺诈", "欺骗", "诈骗", "欺瞞", "唬騙", "欺詐", "欺騙", "詐騙"},
		"威吓": {"Intimidation", "恐吓", "威嚇", "恐嚇"},
		"表演": {"Performance"},
	},
}
