package slicep

import (
	"reflect"
	"testing"

	"sealdice-core/utils/paginate"
)

func TestSliceAdapter(t *testing.T) {
	type args struct {
		source any
	}
	var tests []struct {
		name string
		args args
		want paginate.IAdapter
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Adapter(tt.args.source); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SliceAdapter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSlice_Length(t *testing.T) {
	type fields struct {
		src any
	}
	var tests []struct {
		name    string
		fields  fields
		want    int64
		wantErr bool
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Slice{
				src: tt.fields.src,
			}
			got, err := s.Length()
			if (err != nil) != tt.wantErr {
				t.Errorf("Length() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Length() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSlice_Slice(t *testing.T) {
	type fields struct {
		src any
	}
	type args struct {
		offset int64
		length int64
		dest   any
	}
	var tests []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Slice{
				src: tt.fields.src,
			}
			if err := s.Slice(tt.args.offset, tt.args.length, tt.args.dest); (err != nil) != tt.wantErr {
				t.Errorf("Slice() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_isPtr(t *testing.T) {
	type args struct {
		data any
	}
	var tests []struct {
		name string
		args args
		want bool
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPtr(tt.args.data); got != tt.want {
				t.Errorf("isPtr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isSlice(t *testing.T) {
	type args struct {
		data any
	}
	var tests []struct {
		name string
		args args
		want bool
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSlice(tt.args.data); got != tt.want {
				t.Errorf("isSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_makeSlice(t *testing.T) {
	type args struct {
		data   interface{}
		length int
		cap    int
	}
	var tests []struct {
		name    string
		args    args
		wantErr bool
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := makeSlice(tt.args.data, tt.args.length, tt.args.cap); (err != nil) != tt.wantErr {
				t.Errorf("makeSlice() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
