package dynamicform

// 校验类型：用于判定 data 的合法性
const (
	CheckTypeNull = iota
	CheckTypeNum
)

// 输入类型：用于描述表单项的数据类型与前端交互方式
const (
	InputTypeText      = 0
	InputTypeNum       = 1
	InputTypeDate      = 4
	InputTypeSin       = 5
	InputTypeMul       = 6
	InputTypeBool      = 10
	InputTypeDateRange = 11
	InputTypeSelect    = 12
)

// 是否必填
const (
	RequiredTrue  int8 = 1
	RequiredFalse int8 = 0
)

type SizeRange struct {
	Min int64 `json:"min"`
	Max int64 `json:"max"`
}

type Option struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type RangeValue struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

// FormConfigItem 表单项定义
type FormConfigItem struct {
	ID           uint64      `json:"id"`
	Name         string      `json:"name"`
	InputType    int         `json:"input_type"`
	IsRequired   int8        `json:"is_required"`
	Default      string      `json:"default"`
	Placeholder  string      `json:"placeholder"`
	TableName    string      `json:"table_name,omitempty"`
	FieldName    string      `json:"field_name,omitempty"`
	Tag          string      `json:"tag,omitempty"`
	CheckType    int         `json:"check_type"`
	SizeRange    *SizeRange  `json:"size_range"`
	Hint         string      `json:"hint"`
	ErrMsg       string      `json:"err_msg"`
	SubOption    []*Option   `json:"sub_option"`
	OptionsURL   string      `json:"options_url,omitempty"`
	DefaultRange *RangeValue `json:"default_range,omitempty"`
}

type FormConfigItems []*FormConfigItem

// SubmitFormItem 提交项（id 来自表单定义；data 为用户输入）
type SubmitFormItem struct {
	ID   uint64 `json:"id"`
	Data string `json:"data"`
}

type SubmitFormItems []*SubmitFormItem
