package syncmap

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/puzpuzpuz/xsync/v3"
)

// SyncMap 是一个线程安全的 map，提供了对并发读写的支持
// 它封装了 xsync 提供的 Map 实现
// BenchMark:
// BenchmarkXsyncMapReadsOnly
// BenchmarkXsyncMapReadsOnly-12                     320460              3805 ns/op
// BenchmarkXsyncMapReadsWithWrites
// BenchmarkXsyncMapReadsWithWrites-12               119052             11138 ns/op
// BenchmarkGoSyncMapReadsOnly
// BenchmarkGoSyncMapReadsOnly-12                     78333             15131 ns/op
// BenchmarkGoSyncMapReadsWithWrites
// BenchmarkGoSyncMapReadsWithWrites-12               40977             31248 ns/op
// BenchmarkSafeMapReadsOnly
// BenchmarkSafeMapReadsOnly-12                       12453             90218 ns/op
// BenchmarkSafeMapReadsWithWrites
// BenchmarkSafeMapReadsWithWrites-12                  7665            157391 ns/op
// 按木落要求，保留这个文件，但没任何地方在用它。
type SyncMap[K comparable, V any] struct {
	m    *xsync.MapOf[K, V]
	once sync.Once
}

// ensureInitialized 确保 m 已初始化
func (m *SyncMap[K, V]) ensureInitialized() {
	m.once.Do(func() {
		if m.m == nil {
			m.m = xsync.NewMapOf[K, V]()
		}
	})
}

// NewSyncMap 创建一个新的 SyncMap 实例
func NewSyncMap[K comparable, V any]() *SyncMap[K, V] {
	return &SyncMap[K, V]{m: xsync.NewMapOf[K, V]()}
}

// Delete 删除指定的键及其值
func (m *SyncMap[K, V]) Delete(key K) {
	m.m.Delete(key)
}

// Load 返回指定键的值。如果键存在，则返回值和 true；否则返回零值和 false
func (m *SyncMap[K, V]) Load(key K) (value V, ok bool) {
	val, ok := m.m.Load(key)
	if ok {
		value = val
	}
	return
}

// Exists 检查指定的键是否存在
func (m *SyncMap[K, V]) Exists(key K) bool {
	_, exists := m.Load(key)
	return exists
}

func (m *SyncMap[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	actual, loaded = m.m.LoadOrStore(key, value)
	return
}

// LoadAndDelete 加载并删除指定的键。如果键存在，则返回值和 true；否则返回零值和 false
func (m *SyncMap[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	val, ok := m.m.LoadAndDelete(key)
	if ok {
		value = val
	}
	return
}

// Store 存储指定的键和值
func (m *SyncMap[K, V]) Store(key K, value V) {
	m.m.Store(key, value)
}

// Len 返回 map 中的元素数量
func (m *SyncMap[K, V]) Len() int {
	return m.m.Size()
}

// Range 遍历 map 中的所有键值对，并对每个键值对调用提供的函数
// 如果函数返回 false，则停止遍历
func (m *SyncMap[K, V]) Range(f func(key K, value V) bool) {
	m.m.Range(func(key K, value V) bool {
		return f(key, value)
	})
}

// MarshalJSON 序列化 SyncMap 为 JSON 格式
func (m *SyncMap[K, V]) MarshalJSON() ([]byte, error) {
	if m.m == nil {
		return nil, errors.New("SyncMap未初始化")
	}
	data := make(map[K]V)
	m.Range(func(key K, value V) bool {
		data[key] = value
		return true
	})
	return json.Marshal(data)
}

// UnmarshalJSON 反序列化 JSON 格式为 SyncMap
func (m *SyncMap[K, V]) UnmarshalJSON(b []byte) error {
	m.ensureInitialized()

	var data map[K]V
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}

	for key, value := range data {
		m.Store(key, value)
	}
	return nil
}
