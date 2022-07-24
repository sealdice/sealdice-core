package dice

type AttributeOrderOthers struct {
	SortBy string `yaml:"sortBy"` // time | Name | value desc
}

type AttributeOrder struct {
	Top    []string             `yaml:"top,flow"`
	Others AttributeOrderOthers `yaml:"others"`
}

type AttributeConfigs struct {
	Alias map[string][]string `yaml:"alias"`
	Order AttributeOrder      `yaml:"order"`
}

type DefaultValue struct {
	TypeId VMValueType `json:"typeId"`
	Value  interface{} `json:"value"`
}

type CharacterTemplate struct {
	// 属性
	// 技能
	// 默认值 AttrDefault
	// 同义词 synonyms
	AttrList      []string
	DefaultValues map[string]DefaultValue
}
