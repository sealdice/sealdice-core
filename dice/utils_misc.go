package dice

import "strconv"

type VMValueType int

const (
	VMTypeInt64            VMValueType = 0
	VMTypeString           VMValueType = 1
	VMTypeBool             VMValueType = 2
	VMTypeExpression       VMValueType = 3
	VMTypeNone             VMValueType = 4
	VMTypeDNDComputedValue VMValueType = 5  // 旧computed
	VMTypeComputedValue    VMValueType = 15 // 新computed
)

type VMDndComputedValueData struct {
	BaseValue VMValue `json:"base_value"`
	Expr      string  `json:"expr"`
}

func (vd *VMDndComputedValueData) SetValue(v *VMValue) {
	vd.BaseValue = *v
}

func (vd *VMDndComputedValueData) ReadBaseInt64() (int64, bool) {
	if vd.BaseValue.TypeId == VMTypeInt64 {
		return vd.BaseValue.Value.(int64), true
	}
	return 0, false
}

type ComputedData struct {
	Expr string `json:"expr"`

	/* 缓存数据 */
	Attrs *SyncMap[string, *VMValue] `json:"-"`
	//codeIndex int
	//code      []ByteCode
}

func (v *VMValue) ReadComputed() (*ComputedData, bool) {
	if v.TypeId == VMTypeComputedValue {
		return v.Value.(*ComputedData), true
	}
	return nil, false
}

func VMValueNewComputedRaw(computed *ComputedData) *VMValue {
	return &VMValue{TypeId: VMTypeComputedValue, Value: computed}
}

func VMValueNewComputed(expr string) *VMValue {
	return &VMValue{TypeId: VMTypeComputedValue, Value: &ComputedData{
		Expr: expr,
	}}
}

type VMValue struct {
	TypeId      VMValueType `json:"typeId"`
	Value       interface{} `json:"value"`
	ExpiredTime int64       `json:"expiredTime"`
}

func VMValueNew(typeId VMValueType, val interface{}) *VMValue {
	return &VMValue{
		TypeId: typeId,
		Value:  val,
	}
}

func (v *VMValue) AsBool() bool {
	switch v.TypeId {
	case VMTypeInt64:
		return v.Value != int64(0)
	case VMTypeString:
		return v.Value != ""
	case VMTypeNone:
		return false
	case VMTypeDNDComputedValue:
		vd := v.Value.(*VMDndComputedValueData)
		return vd.BaseValue.AsBool()
	case VMTypeComputedValue:
		return true
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
	case VMTypeDNDComputedValue:
		vd := v.Value.(*VMDndComputedValueData)
		return vd.BaseValue.ToString() + "=> (" + vd.Expr + ")"
	case VMTypeComputedValue:
		cd, _ := v.ReadComputed()
		return cd.Expr
		//return "&(" + cd.Expr + ")"
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

func (v *VMValue) ComputedExecute(ctx *MsgContext) (*VmResult, string) {
	cd, _ := v.ReadComputed()

	realV, detail, err := ctx.Dice.ExprEvalBase(cd.Expr, ctx, RollExtraFlags{})
	if err != nil {
		return nil, detail
	}

	return realV, detail
}
