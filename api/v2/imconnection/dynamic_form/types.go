package dynamicform

// 校验类型：用于判定 data 的合法性
const (
	CheckTypeNull = iota
	CheckTypeNum
)

// 输入类型：用于描述表单项的数据类型与前端交互方式
const (
	InputTypeText   = 0
	InputTypeNum    = 1
	InputTypeBool   = 10
	InputTypeSelect = 12
)

// 是否必填
const (
	RequiredTrue  int8 = 1
	RequiredFalse int8 = 0
)

type Option struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// FormConfigItem 表单项定义
type FormConfigItem struct {
	Minimum     *int      `json:"minimum,omitempty"`
	Maximum     *int      `json:"maximum,omitempty"`
	Name        string    `json:"name"`
	InputType   int       `json:"input_type"`
	IsRequired  int8      `json:"is_required"`
	Default     string    `json:"default"`
	Placeholder string    `json:"placeholder"`
	FieldName   string    `json:"field_name"`
	CheckType   int       `json:"check_type"`
	SubOption   []*Option `json:"sub_option"`
	Sensitive   bool      `json:"sensitive,omitempty"`
	Readonly    bool      `json:"readonly,omitempty"`
}
