package dice

import (
	"strconv"
	"strings"

	ds "github.com/sealdice/dicescript"
)

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
	if vd.BaseValue.TypeID == VMTypeInt64 {
		return vd.BaseValue.Value.(int64), true
	}
	return 0, false
}

type ComputedData struct {
	Expr string `json:"expr"`

	/* 缓存数据 */
	Attrs *SyncMap[string, *VMValue] `json:"-"`
}

func (v *VMValue) ReadComputed() (*ComputedData, bool) {
	if v.TypeID == VMTypeComputedValue {
		return v.Value.(*ComputedData), true
	}
	return nil, false
}

func VMValueNewComputedRaw(computed *ComputedData) *VMValue {
	return &VMValue{TypeID: VMTypeComputedValue, Value: computed}
}

func VMValueNewComputed(expr string) *VMValue {
	return &VMValue{TypeID: VMTypeComputedValue, Value: &ComputedData{
		Expr: expr,
	}}
}

type VMValue struct {
	TypeID      VMValueType `json:"typeId"`
	Value       interface{} `json:"value"`
	ExpiredTime int64       `json:"expiredTime"`
}

func VMValueNew(typeID VMValueType, val interface{}) *VMValue {
	return &VMValue{
		TypeID: typeID,
		Value:  val,
	}
}

func (v *VMValue) AsBool() bool {
	switch v.TypeID {
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
	switch v.TypeID {
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
		// return "&(" + cd.Expr + ")"
	default:
		return "a value"
	}
}

func (v *VMValue) ReadInt64() (int64, bool) {
	if v.TypeID == VMTypeInt64 {
		return v.Value.(int64), true
	}
	return 0, false
}

func (v *VMValue) ReadString() (string, bool) {
	if v.TypeID == VMTypeString {
		return v.Value.(string), true
	}
	return "", false
}

func (v *VMValue) ComputedExecute(ctx *MsgContext, curDepth int64) (*VMResult, string, error) {
	cd, _ := v.ReadComputed()

	realV, detail, err := ctx.Dice.ExprEvalBase(cd.Expr, ctx, RollExtraFlags{vmDepth: curDepth + 1})

	return realV, detail, err
}

func (v *VMValue) ConvertToDiceScriptValue() *ds.VMValue {
	switch v.TypeID {
	case VMTypeInt64:
		return ds.VMValueNewInt(v.Value.(ds.IntType))
	case VMTypeString:
		return ds.VMValueNewStr(v.Value.(string))
	case VMTypeNone:
		return ds.VMValueNewNull()
	case VMTypeDNDComputedValue:
		oldCD := v.Value.(*VMDndComputedValueData)
		m := &ds.ValueMap{}
		base := oldCD.BaseValue.ConvertToDiceScriptValue()
		if base.TypeId == ds.VMTypeUndefined {
			base = ds.VMValueNewInt(0)
		}
		m.Store("base", base)
		expr := strings.ReplaceAll(oldCD.Expr, "$tVal", "this.base")
		expr = strings.ReplaceAll(expr, "熟练", "(熟练||0)")
		cd := &ds.ComputedData{
			Expr:  expr,
			Attrs: m,
		}
		return ds.VMValueNewComputedRaw(cd)
	case VMTypeComputedValue:
		oldCd, _ := v.ReadComputed()

		m := &ds.ValueMap{}
		oldCd.Attrs.Range(func(key string, value *VMValue) bool {
			m.Store(key, value.ConvertToDiceScriptValue())
			return true
		})
		cd := &ds.ComputedData{
			Expr:  oldCd.Expr,
			Attrs: m,
		}
		return ds.VMValueNewComputedRaw(cd)
	default:
		return ds.VMValueNewUndefined()
	}
}
