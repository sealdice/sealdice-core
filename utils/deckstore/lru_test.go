package deckstore

import "testing"

func TestLRUByteCap(t *testing.T) {
	c := newByteLRU[[]byte](10)

	c.put("k1", []byte("aaaaa"), 5)
	c.put("k2", []byte("bbbbb"), 5)
	if _, ok := c.get("k1"); !ok {
		t.Fatal("k1 should be present")
	}
	if _, ok := c.get("k2"); !ok {
		t.Fatal("k2 should be present")
	}

	// 再放入 5 字节：总量 15 > 10，应逐出最旧的 k1。
	c.put("k3", []byte("ccccc"), 5)
	if _, ok := c.get("k1"); ok {
		t.Fatal("k1 should have been evicted (oldest)")
	}
	if _, ok := c.get("k2"); !ok {
		t.Fatal("k2 should remain")
	}
	if _, ok := c.get("k3"); !ok {
		t.Fatal("k3 should remain")
	}

	// 重新访问 k2（变为最近使用），再放入 k4 应逐出 k3 而非 k2。
	c.get("k2")
	c.put("k4", []byte("ddddd"), 5)
	if _, ok := c.get("k3"); ok {
		t.Fatal("k3 should have been evicted")
	}
	if _, ok := c.get("k2"); !ok {
		t.Fatal("k2 should remain (recently used)")
	}
	if _, ok := c.get("k4"); !ok {
		t.Fatal("k4 should remain")
	}
}
