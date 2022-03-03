package dice

import "strconv"

type VMValueType int

const (
	VMTypeInt64      VMValueType = 0
	VMTypeString     VMValueType = 1
	VMTypeBool       VMValueType = 2
	VMTypeExpression VMValueType = 3
	VMTypeNone       VMValueType = 4
)

type VMValue struct {
	TypeId VMValueType `json:"typeId"`
	Value  interface{} `json:"value"`
}

func (v *VMValue) ToString() string {
	switch v.TypeId {
	case VMTypeInt64:
		return strconv.FormatInt(v.Value.(int64), 10)
	case VMTypeString:
		return v.Value.(string)
	case VMTypeNone:
		return v.Value.(string)
	default:
		return "a value"
	}
}
