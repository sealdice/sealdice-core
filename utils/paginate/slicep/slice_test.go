package slicep_test

import (
	"testing"

	"sealdice-core/utils/paginate/slicep"
)

func TestSliceAdapter(t *testing.T) {
	src := []int{1, 2, 3}
	adapter := slicep.Adapter(src)
	if adapter == nil {
		t.Fatal("Adapter returned nil")
	}
	length, err := adapter.Length()
	if err != nil {
		t.Fatalf("Length returned error: %v", err)
	}
	if length != 3 {
		t.Fatalf("Length = %d, want 3", length)
	}
}

func TestSlice_Length(t *testing.T) {
	adapter := slicep.Adapter([]string{"a", "b"})
	got, err := adapter.Length()
	if err != nil {
		t.Fatalf("Length returned error: %v", err)
	}
	if got != 2 {
		t.Fatalf("Length = %d, want 2", got)
	}
}

func TestSlice_Slice(t *testing.T) {
	adapter := slicep.Adapter([]string{"a", "b", "c"})
	var dest []string
	if err := adapter.Slice(1, 2, &dest); err != nil {
		t.Fatalf("Slice returned error: %v", err)
	}
	if len(dest) != 2 || dest[0] != "b" || dest[1] != "c" {
		t.Fatalf("Slice result = %#v, want [b c]", dest)
	}
}

func TestSliceImplementsPaginateAdapter(t *testing.T) {
	var _ = slicep.Adapter([]int{1})
}

func TestAdapterRejectsPointerInput(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for pointer input")
		}
	}()
	src := []int{1, 2, 3}
	_ = slicep.Adapter(&src)
}

func TestSliceHandlesEmptyRange(t *testing.T) {
	adapter := slicep.Adapter([]string{"a"})
	var dest []string
	if err := adapter.Slice(2, 1, &dest); err != nil {
		t.Fatalf("Slice returned error: %v", err)
	}
	if len(dest) != 0 {
		t.Fatalf("Slice result length = %d, want 0", len(dest))
	}
}
