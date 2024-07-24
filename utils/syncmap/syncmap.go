package syncmap

import (
	"sync"

	cmap "github.com/smallnest/safemap"
)

// 在Go 1.9之前，go语言标准库中并没有实现并发map。
// 在Go 1.9中，引入了sync.Map。新的sync.Map与此concurrent-map有几个关键区别。
// 标准库中的sync.Map是专为append-only场景设计的。
// 因此，如果您想将Map用于一个类似内存数据库，那么使用我们的版本可能会受益。
// 译注:sync.Map在读多写少性能比较好，否则并发性能很差
// 实话说，我也不知道咱们到底是不是读多写少，不过反正下面的函数是做了兼容的，可以方便回退……

// SyncMap 是一个线程安全的 map，提供了对并发读写的支持
// 它封装了 safemap 提供的 SafeMap 实现
type SyncMap[K comparable, V any] struct {
	m    *cmap.SafeMap[K, V]
	once sync.Once
}

// ensureInitialized 确保 m 已初始化
func (m *SyncMap[K, V]) ensureInitialized() {
	m.once.Do(func() {
		if m.m == nil {
			m.m = cmap.New[K, V]()
		}
	})
}

// NewSyncMap 创建一个新的 SyncMap 实例
func NewSyncMap[K comparable, V any]() *SyncMap[K, V] {
	return &SyncMap[K, V]{m: cmap.New[K, V]()}
}

// Delete 删除指定的键及其值
func (m *SyncMap[K, V]) Delete(key K) {
	m.m.Remove(key)
}

// Load 返回指定键的值。如果键存在，则返回值和 true；否则返回零值和 false
func (m *SyncMap[K, V]) Load(key K) (value V, ok bool) {
	return m.m.Get(key)
}

// Exists 检查指定的键是否存在
func (m *SyncMap[K, V]) Exists(key K) bool {
	_, exists := m.Load(key)
	return exists
}

func (m *SyncMap[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	// 使用 Upsert 方法，回调函数中根据键是否存在来设置 loaded 标志
	actual = m.m.Upsert(key, value, func(exist bool, valueInMap V, newValue V) V {
		loaded = exist
		if exist {
			return valueInMap
		}
		return newValue
	})
	return
}

// LoadAndDelete 加载并删除指定的键。如果键存在，则返回值和 true；否则返回零值和 false
func (m *SyncMap[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	return m.m.Pop(key)
}

// Store 存储指定的键和值
func (m *SyncMap[K, V]) Store(key K, value V) {
	m.m.Set(key, value)
}

// Len 返回 map 中的元素数量
func (m *SyncMap[K, V]) Len() int {
	return m.m.Count()
}

// Range 遍历 map 中的所有键值对，并对每个键值对调用提供的函数
// 如果函数返回 false，则停止遍历
func (m *SyncMap[K, V]) Range(f func(key K, value V) bool) {
	m.m.IterCb(func(key K, value V) {
		if !f(key, value) {
			return
		}
	})
}

// 似乎除了这种情况以外，别的时候都能通过直接New一个来规避
// TODO: 如果全部都加上Once是否会影响性能呢？

// MarshalJSON 序列化 SyncMap 为 JSON 格式
func (m *SyncMap[K, V]) MarshalJSON() ([]byte, error) {
	// 怀疑是因为原本的实现方式下，sync.Map默认就是存在的不需要初始化
	// 而如果在这种情况下，默认m是不会被初始化的
	// 所以导致问题，或许应该得手动初始化一个？
	// TODO： 初始化应该不太对劲，有高人指点一下吗
	m.ensureInitialized()
	return m.m.MarshalJSON()
}

// UnmarshalJSON 反序列化 JSON 格式为 SyncMap
func (m *SyncMap[K, V]) UnmarshalJSON(b []byte) error {
	// 怀疑是因为原本的实现方式下，sync.Map默认就是存在的不需要初始化
	// 而如果在这种情况下，默认m是不会被初始化的
	// 所以导致问题，或许应该得手动初始化一个？
	// TODO： 初始化应该不太对劲，有高人指点一下吗
	m.ensureInitialized()
	return m.m.UnmarshalJSON(b)
}
