package slicep_test

import (
	"reflect"
	"testing"

	"sealdice-core/utils/paginate/slicep"
)

func TestAdapterLength(t *testing.T) {
	adapter := slicep.Adapter([]int{1, 2, 3})

	got, err := adapter.Length()
	if err != nil {
		t.Fatalf("Length returned error: %v", err)
	}
	if got != 3 {
		t.Fatalf("Length = %d, want 3", got)
	}
}

func TestAdapterSliceCopiesRequestedWindow(t *testing.T) {
	adapter := slicep.Adapter([]string{"a", "b", "c", "d"})
	var out []string

	if err := adapter.Slice(1, 2, &out); err != nil {
		t.Fatalf("Slice returned error: %v", err)
	}
	if !reflect.DeepEqual(out, []string{"b", "c"}) {
		t.Fatalf("Slice output = %#v, want [b c]", out)
	}
}

func TestAdapterSliceClampsBeyondEnd(t *testing.T) {
	adapter := slicep.Adapter([]int{1, 2, 3})
	var out []int

	if err := adapter.Slice(2, 10, &out); err != nil {
		t.Fatalf("Slice returned error: %v", err)
	}
	if !reflect.DeepEqual(out, []int{3}) {
		t.Fatalf("Slice output = %#v, want [3]", out)
	}
}

func TestAdapterRejectsNonSliceSource(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Adapter accepted non-slice source")
		}
	}()

	_ = slicep.Adapter(42)
}

func TestSliceRequiresPointerDestination(t *testing.T) {
	adapter := slicep.Adapter([]int{1})
	var out []int

	if err := adapter.Slice(0, 1, out); err == nil {
		t.Fatal("Slice accepted non-pointer destination")
	}
}
