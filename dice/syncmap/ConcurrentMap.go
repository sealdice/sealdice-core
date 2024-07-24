package syncmap

import (
	cmap "github.com/smallnest/safemap"
)

// SyncMap 是一个线程安全的 map，提供了对并发读写的支持
// 它封装了 safemap 提供的 SafeMap 实现
type SyncMap[K comparable, V any] struct {
	m *cmap.SafeMap[K, V]
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

// MarshalJSON 序列化 SyncMap 为 JSON 格式
func (m *SyncMap[K, V]) MarshalJSON() ([]byte, error) {
	return m.m.MarshalJSON()
}

// UnmarshalJSON 反序列化 JSON 格式为 SyncMap
func (m *SyncMap[K, V]) UnmarshalJSON(b []byte) error {
	return m.m.UnmarshalJSON(b)
}
