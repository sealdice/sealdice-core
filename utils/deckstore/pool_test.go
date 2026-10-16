package deckstore_test

import (
	"math"
	"testing"

	"sealdice-core/utils/deckstore"
)

// fixedRand 依次返回预设序列（对 n 取模保证落在 [0,n) 内），用于确定性测试。
type fixedRand struct {
	vals []int64
	i    int
}

func (r *fixedRand) Int63n(n int64) int64 {
	if r.i < len(r.vals) {
		v := r.vals[r.i]
		r.i++
		return v % n
	}
	return 0
}

// countingRand 返回 0,1,2,...（对 n 取模），近似均匀地遍历区间。
type countingRand struct {
	i int64
}

func (r *countingRand) Int63n(n int64) int64 {
	v := r.i % n
	r.i++
	return v
}

func TestUniformPoolNoReplaceExhausts(t *testing.T) {
	p := deckstore.NewIndexPool(5, nil, &fixedRand{vals: []int64{7, 13, 42, 99, 1000, 3, 17}})

	seen := make([]bool, 5)
	for i := 0; i < 5; i++ {
		idx, ok := p.Pick(&fixedRand{})
		if !ok {
			t.Fatalf("第 %d 次 Pick 返回 false，期望 true", i)
		}
		if idx < 0 || idx >= 5 {
			t.Fatalf("Pick = %d，越界", idx)
		}
		if seen[idx] {
			t.Fatalf("idx %d 重复出现（不放回应各不相同）", idx)
		}
		seen[idx] = true
	}

	if _, ok := p.Pick(&fixedRand{}); ok {
		t.Fatal("第 6 次 Pick 应返回 false（已耗尽）")
	}
}

func TestWeightedPoolDistribution(t *testing.T) {
	const draws = 2000
	weights := []uint32{1, 1, 1, 99} // ::1::×3 + ::99::

	r := &countingRand{}
	heavy := 0 // 权重 99 的下标（idx==3）
	for i := 0; i < draws; i++ {
		p := deckstore.NewIndexPool(4, weights, r)
		idx, ok := p.Pick(r)
		if !ok {
			t.Fatalf("第 %d 次 Pick 返回 false", i)
		}
		if idx == 3 {
			heavy++
		}
	}

	got := float64(heavy) / draws
	want := 99.0 / 102.0
	if math.Abs(got-want) > 0.05 {
		t.Fatalf("重牌占比 = %.4f，期望 ≈ %.4f（±0.05）", got, want)
	}
}

func TestPoolDeterministicWithFixedRand(t *testing.T) {
	drawAll := func(p *deckstore.IndexPool, r *fixedRand) []int {
		var out []int
		for {
			idx, ok := p.Pick(r)
			if !ok {
				return out
			}
			out = append(out, idx)
		}
	}
	assertEq := func(a, b []int) {
		t.Helper()
		if len(a) != len(b) {
			t.Fatalf("长度不一致：%v vs %v", a, b)
		}
		for i := range a {
			if a[i] != b[i] {
				t.Fatalf("第 %d 个 idx 不一致：%v vs %v", i, a, b)
			}
		}
	}

	vals := []int64{7, 13, 42, 99, 1000, 3, 17, 5}

	// 均匀：置换在构造时洗出，相同 rand 序列应得到相同 idx 序列。
	a := deckstore.NewIndexPool(6, nil, &fixedRand{vals: vals})
	b := deckstore.NewIndexPool(6, nil, &fixedRand{vals: vals})
	assertEq(drawAll(a, &fixedRand{}), drawAll(b, &fixedRand{}))

	// 加权：Fenwick 树确定性构造，Pick 消耗相同 rand 序列应得到相同 idx 序列。
	w := []uint32{2, 5, 1, 10}
	c := deckstore.NewIndexPool(4, w, &fixedRand{vals: vals})
	d := deckstore.NewIndexPool(4, w, &fixedRand{vals: vals})
	assertEq(drawAll(c, &fixedRand{vals: vals}), drawAll(d, &fixedRand{vals: vals}))
}

func TestWeightedPoolExhausts(t *testing.T) {
	// 权重 [2,1,1]：3 个下标各抽出一次，第 4 次返回 false。
	r := &countingRand{}
	p := deckstore.NewIndexPool(3, []uint32{2, 1, 1}, r)

	seen := make([]bool, 3)
	for i := 0; i < 3; i++ {
		idx, ok := p.Pick(r)
		if !ok {
			t.Fatalf("第 %d 次 Pick 返回 false", i)
		}
		if seen[idx] {
			t.Fatalf("idx %d 重复（不放回应各不相同）", idx)
		}
		seen[idx] = true
	}
	if _, ok := p.Pick(r); ok {
		t.Fatal("第 4 次 Pick 应返回 false")
	}
}

func TestPoolEmptyOrZeroWeight(t *testing.T) {
	cases := []*deckstore.IndexPool{
		deckstore.NewIndexPool(0, nil, &fixedRand{}),
		deckstore.NewIndexPool(-1, nil, &fixedRand{}),
		deckstore.NewIndexPool(3, []uint32{0, 0, 0}, &fixedRand{}),
	}
	for i, p := range cases {
		if _, ok := p.Pick(&fixedRand{}); ok {
			t.Fatalf("case %d: Pick 应返回 false", i)
		}
	}
}
