package slicep

import (
	"fmt"
	"reflect"

	"sealdice-core/utils/paginate"
)

func Adapter(source any) paginate.IAdapter {
	if isPtr(source) || !isSlice(source) {
		panic(fmt.Sprintf("expected slicep but got %s", reflect.TypeOf(source).Kind()))
	}
	return &Slice{src: source}
}

type Slice struct {
	src any
}

func (s Slice) Length() (int64, error) {
	return int64(reflect.ValueOf(s.src).Len()), nil
}

func (s Slice) Slice(offset, length int64, dest any) error {
	va := reflect.ValueOf(s.src)
	fullSize := int64(va.Len())
	needSize := length + offset
	if fullSize < needSize {
		length = fullSize - offset
	}
	lengthInt := int(length)
	if lengthInt <= 0 {
		lengthInt = 0
	}
	if err := makeSlice(dest, lengthInt, lengthInt); err != nil {
		return err
	}
	// 超出切片可以切割的范围
	if lengthInt == 0 {
		return nil
	}
	//防止切片需要切割的过多，导致的失败
	if needSize > fullSize {
		needSize = fullSize
	}
	vs := va.Slice(int(offset), int(needSize))
	vt := reflect.ValueOf(dest).Elem()
	for i := 0; i < vs.Len(); i++ {
		vt.Index(i).Set(reflect.ValueOf(vs.Index(i).Interface()))
	}
	return nil
}

func isPtr(data any) bool {
	t := reflect.TypeOf(data)
	return t.Kind() == reflect.Ptr
}
func isSlice(data any) bool {
	v := reflect.Indirect(reflect.ValueOf(data))
	return v.Kind() == reflect.Slice
}
func makeSlice(data interface{}, length, cap int) error {
	if !isPtr(data) {
		return fmt.Errorf("expected to be a ptr but got %T", data)
	}
	if !isSlice(data) {
		return fmt.Errorf("expected to be a slicep pointer but got %T", data)
	}
	ind := reflect.Indirect(reflect.ValueOf(data))
	typ := reflect.TypeOf(ind.Interface())
	reflect.ValueOf(data).Elem().Set(reflect.MakeSlice(typ, length, cap))
	return nil
}
