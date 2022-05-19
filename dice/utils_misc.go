package dice

import "strconv"

type VMValueType int

const (
	VMTypeInt64         VMValueType = 0
	VMTypeString        VMValueType = 1
	VMTypeBool          VMValueType = 2
	VMTypeExpression    VMValueType = 3
	VMTypeNone          VMValueType = 4
	VMTypeComputedValue VMValueType = 5
)

type VMComputedValueData struct {
	BaseValue VMValue `json:"base_value"`
	Expr      string  `json:"expr"`
}

func (cv *VMComputedValueData) SetValue(v *VMValue) {
	cv.BaseValue = *v
}

func (vd *VMComputedValueData) ReadBaseInt64() (int64, bool) {
	if vd.BaseValue.TypeId == VMTypeInt64 {
		return vd.BaseValue.Value.(int64), true
	}
	return 0, false
}

type VMValue struct {
	TypeId VMValueType `json:"typeId"`
	Value  interface{} `json:"value"`
}

func (v *VMValue) AsBool() bool {
	switch v.TypeId {
	case VMTypeInt64:
		return v.Value != int64(0)
	case VMTypeString:
		return v.Value != ""
	case VMTypeNone:
		return false
	case VMTypeComputedValue:
		vd := v.Value.(*VMComputedValueData)
		return vd.BaseValue.AsBool()
	default:
		return false
	}
}

func (v *VMValue) ToString() string {
	switch v.TypeId {
	case VMTypeInt64:
		return strconv.FormatInt(v.Value.(int64), 10)
	case VMTypeString:
		return v.Value.(string)
	case VMTypeNone:
		return v.Value.(string)
	case VMTypeComputedValue:
		vd := v.Value.(*VMComputedValueData)
		return vd.BaseValue.ToString() + "=> (" + vd.Expr + ")"
	default:
		return "a value"
	}
}

func (v *VMValue) ReadInt64() (int64, bool) {
	if v.TypeId == VMTypeInt64 {
		return v.Value.(int64), true
	}
	return 0, false
}

func (v *VMValue) ReadString() (string, bool) {
	if v.TypeId == VMTypeString {
		return v.Value.(string), true
	}
	return "", false
}
