package deckstore

// IndexPool 是不放回的下标抽样池：
//   - 均匀组（weights 为 nil / 空）：构造时用 Fisher-Yates 洗出 [0,count) 的
//     随机置换，随后 Pick 依次弹出；耗尽返回 false。
//   - 加权组（weights 非空）：用 Fenwick 树维护剩余权重，每次 Pick 按权重
//     不放回抽取一个下标；耗尽返回 false。
//
// 随机源一律来自入参 Rand，不使用包级 math/rand。
type IndexPool struct {
	weighted bool
	perm     []int   // 均匀：洗好的置换，Pick 从末尾弹出
	weights  []int64 // 加权：每个下标剩余权重（抽出后置 0）
	tree     []int64 // 加权：Fenwick 树，长度 len(weights)+1
	total    int64   // 加权：剩余总权重
}

// NewIndexPool 构造一个不放回下标抽样池。count<=0 或权重总和为 0 时返回一个
// 空池（Pick 恒返回 false），不 panic。weights 为 nil / 空表示均匀。
func NewIndexPool(count int, weights []uint32, rand Rand) *IndexPool {
	if count <= 0 {
		return &IndexPool{}
	}
	if len(weights) == 0 {
		p := &IndexPool{perm: make([]int, count)}
		for i := range p.perm {
			p.perm[i] = i
		}
		for i := count - 1; i > 0; i-- {
			j := int(rand.Int63n(int64(i + 1)))
			p.perm[i], p.perm[j] = p.perm[j], p.perm[i]
		}
		return p
	}

	p := &IndexPool{
		weighted: true,
		weights:  make([]int64, count),
		tree:     make([]int64, count+1),
	}
	for i := range count {
		var w int64
		if i < len(weights) {
			w = int64(weights[i])
		} else {
			w = 1 // 缺省权重 1
		}
		p.weights[i] = w
		p.add(i, w)
	}
	if p.total <= 0 {
		return &IndexPool{}
	}
	return p
}

// Pick 不放回抽取一个下标；耗尽（或无有效权重）返回 (0, false)。
func (p *IndexPool) Pick(rand Rand) (int, bool) {
	if !p.weighted {
		if len(p.perm) == 0 {
			return 0, false
		}
		n := len(p.perm)
		idx := p.perm[n-1]
		p.perm = p.perm[:n-1]
		return idx, true
	}

	if p.total <= 0 {
		return 0, false
	}
	target := rand.Int63n(p.total) + 1
	idx := p.findByPrefix(target)
	if idx < 0 || idx >= len(p.weights) {
		return 0, false
	}
	w := p.weights[idx]
	if w <= 0 {
		return 0, false
	}
	p.weights[idx] = 0
	p.add(idx, -w)
	return idx, true
}

// add 在 Fenwick 树上对下标 index 增加 delta，并累加 total。
func (p *IndexPool) add(index int, delta int64) {
	p.total += delta
	for i := index + 1; i < len(p.tree); i += i & -i {
		p.tree[i] += delta
	}
}

// findByPrefix 返回包含第 target 个权重单位的下标（target 为 1-based）。
func (p *IndexPool) findByPrefix(target int64) int {
	idx := 0
	bit := 1
	for bit < len(p.tree) {
		bit <<= 1
	}
	for step := bit; step > 0; step >>= 1 {
		next := idx + step
		if next < len(p.tree) && p.tree[next] < target {
			target -= p.tree[next]
			idx = next
		}
	}
	return idx
}
