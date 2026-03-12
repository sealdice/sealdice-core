//nolint:testpackage
package utils

import (
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// --- Smoke tests for ParseRate ---

func TestParseRate_Integer(t *testing.T) {
	got, err := ParseRate("100")
	if err != nil {
		t.Fatalf("ParseRate(\"100\") returned error: %v", err)
	}
	if got != rate.Limit(100) {
		t.Errorf("ParseRate(\"100\") = %v, want 100", got)
	}
}

func TestParseRate_AtEvery(t *testing.T) {
	got, err := ParseRate("@every 1s")
	if err != nil {
		t.Fatalf("ParseRate(\"@every 1s\") returned error: %v", err)
	}
	want := rate.Every(time.Second)
	if got != want {
		t.Errorf("ParseRate(\"@every 1s\") = %v, want %v", got, want)
	}
}

func TestParseRate_AtEvery_Minutes(t *testing.T) {
	_, err := ParseRate("@every 5m")
	if err != nil {
		t.Fatalf("ParseRate(\"@every 5m\") returned error: %v", err)
	}
}

func TestParseRate_Invalid_String(t *testing.T) {
	_, err := ParseRate("not-a-number")
	if err == nil {
		t.Error("expected error for invalid rate string, got nil")
	}
}

func TestParseRate_Invalid_AtEvery(t *testing.T) {
	_, err := ParseRate("@every not-a-duration")
	if err == nil {
		t.Error("expected error for invalid duration in @every, got nil")
	}
}

// --- Smoke tests for SyncMap ---

func TestSyncMap_StoreLoad(t *testing.T) {
	var m SyncMap[string, int]
	m.Store("key", 42)
	v, ok := m.Load("key")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if v != 42 {
		t.Errorf("Load() = %v, want 42", v)
	}
}

func TestSyncMap_Load_Missing(t *testing.T) {
	var m SyncMap[string, int]
	_, ok := m.Load("nonexistent")
	if ok {
		t.Error("expected missing key to return ok=false")
	}
}

func TestSyncMap_Delete(t *testing.T) {
	var m SyncMap[string, string]
	m.Store("k", "v")
	m.Delete("k")
	_, ok := m.Load("k")
	if ok {
		t.Error("expected deleted key to be gone")
	}
}

func TestSyncMap_Exists(t *testing.T) {
	var m SyncMap[string, bool]
	m.Store("present", true)
	if !m.Exists("present") {
		t.Error("Exists() should return true for stored key")
	}
	if m.Exists("absent") {
		t.Error("Exists() should return false for missing key")
	}
}

func TestSyncMap_LoadOrStore(t *testing.T) {
	var m SyncMap[string, int]
	actual, loaded := m.LoadOrStore("key", 1)
	if loaded {
		t.Error("expected loaded=false on first call")
	}
	if actual != 1 {
		t.Errorf("first LoadOrStore = %v, want 1", actual)
	}

	actual, loaded = m.LoadOrStore("key", 2)
	if !loaded {
		t.Error("expected loaded=true on second call")
	}
	if actual != 1 {
		t.Errorf("second LoadOrStore = %v, want 1 (original value)", actual)
	}
}

func TestSyncMap_LoadAndDelete(t *testing.T) {
	var m SyncMap[string, int]
	m.Store("key", 99)
	v, loaded := m.LoadAndDelete("key")
	if !loaded {
		t.Error("expected loaded=true")
	}
	if v != 99 {
		t.Errorf("LoadAndDelete = %v, want 99", v)
	}
	_, ok := m.Load("key")
	if ok {
		t.Error("key should be deleted after LoadAndDelete")
	}
}

func TestSyncMap_Len(t *testing.T) {
	var m SyncMap[int, int]
	if m.Len() != 0 {
		t.Error("expected Len=0 for empty map")
	}
	m.Store(1, 10)
	m.Store(2, 20)
	m.Store(3, 30)
	if m.Len() != 3 {
		t.Errorf("expected Len=3, got %d", m.Len())
	}
}

func TestSyncMap_Range(t *testing.T) {
	var m SyncMap[string, int]
	m.Store("a", 1)
	m.Store("b", 2)
	m.Store("c", 3)
	count := 0
	m.Range(func(_ string, _ int) bool {
		count++
		return true
	})
	if count != 3 {
		t.Errorf("Range visited %d items, want 3", count)
	}
}

func TestSyncMap_Range_EarlyStop(t *testing.T) {
	var m SyncMap[string, int]
	for i := range 10 {
		m.Store(string(rune('a'+i)), i)
	}
	count := 0
	m.Range(func(_ string, _ int) bool {
		count++
		return count < 3 // stop after 3
	})
	if count != 3 {
		t.Errorf("Range with early stop visited %d items, want 3", count)
	}
}

func TestSyncMap_ConcurrentReadWrite(t *testing.T) {
	var m SyncMap[int, int]
	var wg sync.WaitGroup
	const goroutines = 20
	const ops = 100

	wg.Add(goroutines)
	for g := range goroutines {
		go func(id int) {
			defer wg.Done()
			for i := range ops {
				m.Store(id*ops+i, i)
				m.Load(id*ops + i)
			}
		}(g)
	}
	wg.Wait()
}

// --- Benchmark tests for SplitLongText ---

func BenchmarkSplitLongText_Short(b *testing.B) {
	text := strings.Repeat("Hello world. ", 10) // ~130 bytes, under limit
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SplitLongText(text, 500, DefaultSplitPaginationHint)
	}
}

func BenchmarkSplitLongText_Long(b *testing.B) {
	text := strings.Repeat("这是一段比较长的消息文本，需要被切分成多段发送。", 50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SplitLongText(text, 200, DefaultSplitPaginationHint)
	}
}

func BenchmarkSplitLongText_WithCQCode(b *testing.B) {
	cqCode := "[CQ:image,file=base64://" + strings.Repeat("A", 500) + "]"
	text := strings.Repeat("普通文字内容 ", 30) + cqCode + strings.Repeat(" 更多内容", 30)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SplitLongText(text, 300, DefaultSplitPaginationHint)
	}
}

func BenchmarkLenWithoutCQCode_NoCQ(b *testing.B) {
	text := strings.Repeat("plain text without any cq codes here. ", 20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lenWithoutCQCode(text)
	}
}

func BenchmarkLenWithoutCQCode_WithCQ(b *testing.B) {
	cqCode := "[CQ:image,file=test.jpg]"
	text := strings.Repeat("text "+cqCode+" more ", 20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lenWithoutCQCode(text)
	}
}

// --- Benchmark tests for SyncMap ---

func BenchmarkSyncMap_Store(b *testing.B) {
	var m SyncMap[int, int]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Store(i%1000, i)
	}
}

func BenchmarkSyncMap_Load_Hit(b *testing.B) {
	var m SyncMap[int, int]
	for i := range 1000 {
		m.Store(i, i*2)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.Load(i % 1000)
	}
}

func BenchmarkSyncMap_Load_Miss(b *testing.B) {
	var m SyncMap[int, int]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.Load(i)
	}
}
