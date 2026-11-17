package paginate_test

import (
	"math"
	"reflect"
	"testing"

	"sealdice-core/utils/paginate"
)

func TestSliceBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name       string
		page, size int
		want       []int
	}{
		{"first", 1, 2, []int{1, 2}},
		{"last", 2, 2, []int{3}},
		{"beyond-end", 3, 2, []int{}},
		{"negative-page", -1, 2, []int{1, 2}},
		{"default-size", 1, 0, []int{1, 2, 3}},
		{"all-items", 1, math.MaxInt, []int{1, 2, 3}},
		{"overflow", math.MaxInt, math.MaxInt, []int{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			items := []int{1, 2, 3}
			got := paginate.Slice(items, tc.page, tc.size)
			if !reflect.DeepEqual(got.List, tc.want) || got.Total != 3 {
				t.Fatalf("unexpected page: %+v", got)
			}
			if got.Page < 1 || got.PageSize < 1 {
				t.Fatalf("invalid metadata: %+v", got)
			}
			if len(got.List) > 0 {
				got.List[0] = 99
			}
			if items[0] != 1 {
				t.Fatal("page shares slice storage with source")
			}
		})
	}
	if got := paginate.Slice[int](nil, 1, 10); got.List == nil || got.Total != 0 {
		t.Fatalf("unexpected empty page: %+v", got)
	}
}
