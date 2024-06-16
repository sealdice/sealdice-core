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

	PreloadCode: "func showAs(val) {" +
		"return `{val ? `{val}[{&val.factor ? '*'+ (&val.factor == 1 ? '' : str(&val.factor))+','}{&val.base}]` : 0}`" +
		"}",

	AttrConfig: AttrConfig{
		Top: []string{"力量", "敏捷", "体质", "体型", "魅力", "智力", "感知", "hp", "ac", "熟练"},
		// SortBy: "Name",
		ShowAs: map[string]string{
			"运动": "{showAs(&运动)}",

			"体操": "{showAs(&体操)}",
			"巧手": "{showAs(&巧手)}",
			"隐匿": "{showAs(&隐匿)}",

			"调查": "{showAs(&调查)}",
			"奥秘": "{showAs(&奥秘)}",
			"历史": "{showAs(&历史)}",
			"自然": "{showAs(&自然)}",
			"宗教": "{showAs(&宗教)}",

			"察觉": "{showAs(&察觉)}",
			"洞悉": "{showAs(&洞悉)}",
			"驯兽": "{showAs(&驯兽)}",
			"医药": "{showAs(&医药)}",
			"求生": "{showAs(&求生)}",

			"游说": "{showAs(&游说)}",
			"欺瞒": "{showAs(&欺瞒)}",
			"威吓": "{showAs(&威吓)}",
			"表演": "{showAs(&表演)}",
		},
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
