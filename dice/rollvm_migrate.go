package dice

import (
	ds "github.com/sealdice/dicescript"
	"strings"
)

func dsValueToRollVMv1(v *ds.VMValue) *VMValue {
	var v2 *VMValue
	switch v.TypeId {
	case ds.VMTypeInt:
		v2 = &VMValue{TypeId: VMTypeInt64, Value: v.MustReadInt()}
	case ds.VMTypeFloat:
		v2 = &VMValue{TypeId: VMTypeInt64, Value: int64(v.MustReadFloat())}
	default:
		v2 = &VMValue{TypeId: VMTypeString, Value: v.ToString()}
	}
	return v2
}

func (v *VMValue) ConvertToDiceScriptValue() *ds.VMValue {
	switch v.TypeId {
	case VMTypeInt64:
		return ds.VMValueNewInt(v.Value.(int64))
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
	}
	return ds.VMValueNewUndefined()
}
