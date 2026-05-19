package dice_test

import (
	"encoding/json"
	"sync"
	"testing"

	"sealdice-core/dice"

	"go.uber.org/zap"
)

// TestGetActivatedExtListRace 测试 GetActivatedExtList 的并发安全性
func TestGetActivatedExtListRace(t *testing.T) {
	// 创建一个模拟的 Dice 对象
	d := &dice.Dice{
		ExtList: []*dice.ExtInfo{
			{Name: "ext1", AutoActive: true},
			{Name: "ext2", AutoActive: true},
			{Name: "ext3", AutoActive: false},
		},
		ExtUpdateTime: 1,
	}
	d.Logger = zap.NewNop().Sugar()

	t.Run("concurrent_get_during_init", func(t *testing.T) {
		// 测试多个 goroutine 同时调用 GetActivatedExtList 进行初始化
		group := &dice.GroupInfo{
			GroupID: "test-group-1",
		}
		// 设置初始扩展列表
		group.SetActivatedExtList([]*dice.ExtInfo{
			{Name: "ext1"},
		}, nil)
		// 重置 ExtAppliedTime 为 0 以触发初始化
		group.ExtAppliedTime = 0

		var wg sync.WaitGroup
		const numGoroutines = 100

		for range numGoroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = group.GetActivatedExtList(d)
			}()
		}
		wg.Wait()
	})

	t.Run("concurrent_get_and_set", func(t *testing.T) {
		// 测试 GetActivatedExtList 和 SetActivatedExtList 的并发
		group := &dice.GroupInfo{
			GroupID: "test-group-2",
		}
		group.SetActivatedExtList([]*dice.ExtInfo{
			{Name: "ext1"},
		}, d)

		var wg sync.WaitGroup
		const numGoroutines = 50

		// 一半 goroutine 读取
		for range numGoroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = group.GetActivatedExtList(d)
			}()
		}

		// 一半 goroutine 写入
		for range numGoroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()
				group.SetActivatedExtList([]*dice.ExtInfo{
					{Name: "ext1"},
					{Name: "ext2"},
				}, d)
			}()
		}

		wg.Wait()
	})

	t.Run("concurrent_get_and_marshal", func(t *testing.T) {
		// 测试 GetActivatedExtList 和 MarshalJSON 的并发
		group := &dice.GroupInfo{
			GroupID: "test-group-3",
		}
		group.SetActivatedExtList([]*dice.ExtInfo{
			{Name: "ext1"},
			{Name: "ext2"},
		}, d)

		var wg sync.WaitGroup
		const numGoroutines = 50

		// 一半 goroutine 读取
		for range numGoroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = group.GetActivatedExtList(d)
			}()
		}

		// 一半 goroutine 序列化
		for range numGoroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = json.Marshal(group)
			}()
		}

		wg.Wait()
	})

	t.Run("concurrent_init_and_set", func(t *testing.T) {
		// 测试初始化期间的 Set 操作
		group := &dice.GroupInfo{
			GroupID: "test-group-4",
		}
		group.SetActivatedExtList([]*dice.ExtInfo{
			{Name: "ext1"},
		}, nil)
		// 重置以触发初始化
		group.ExtAppliedTime = 0

		var wg sync.WaitGroup
		const numGoroutines = 50

		// 一半 goroutine 触发初始化
		for range numGoroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = group.GetActivatedExtList(d)
			}()
		}

		// 一半 goroutine 写入
		for range numGoroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()
				group.SetActivatedExtList([]*dice.ExtInfo{
					{Name: "newExt"},
				}, d)
			}()
		}

		wg.Wait()
	})
}
