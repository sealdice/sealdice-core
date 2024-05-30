package dice

import (
	"strings"

	ds "github.com/sealdice/dicescript"
)

func (v *VMValue) ConvertToV2() *ds.VMValue {
	switch v.TypeID {
	case VMTypeInt64:
		return ds.VMValueNewInt(ds.IntType(v.Value.(int64)))
	case VMTypeString:
		return ds.VMValueNewStr(v.Value.(string))
	case VMTypeNone:
		return ds.VMValueNewNull()
	case VMTypeDNDComputedValue:
		oldCD := v.Value.(*VMDndComputedValueData)
		m := &ds.ValueMap{}
		base := oldCD.BaseValue.ConvertToV2()
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
		if oldCd.Attrs != nil {
			oldCd.Attrs.Range(func(key string, value *VMValue) bool {
				m.Store(key, value.ConvertToV2())
				return true
			})
		}
		cd := &ds.ComputedData{
			Expr:  oldCd.Expr,
			Attrs: m,
		}
		return ds.VMValueNewComputedRaw(cd)
	default:
		return ds.VMValueNewUndefined()
	}
}

func dsValueToRollVMv1(v *ds.VMValue) *VMValue {
	var v2 *VMValue
	switch v.TypeId {
	case ds.VMTypeInt:
		v2 = &VMValue{TypeID: VMTypeInt64, Value: int64(v.MustReadInt())}
	case ds.VMTypeFloat:
		v2 = &VMValue{TypeID: VMTypeInt64, Value: int64(v.MustReadFloat())}
	default:
		v2 = &VMValue{TypeID: VMTypeString, Value: v.ToString()}
	}
	return v2
}
