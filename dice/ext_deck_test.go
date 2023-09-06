package dice

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"
)

func TestDeck(t *testing.T) {
	m := make(map[string]int)
	lim := int(math.Pow(10.0, 4))
	for i := 1; i <= lim; i++ {
		var deck []string
		_ = json.Unmarshal([]byte(`["::10::1","::10::2","::10::3","::10::4","::10::5","::10::6","::40::7"]`), &deck)
		s := DeckToShuffleRandomPool(deck)
		k := s.Pick().(string)
		m[k] += 1

	}
	fmt.Println(m) // map[1:1002 2:988 3:954 4:1013 5:965 6:1016 7:4062]
}

func TestDeckLast(t *testing.T) {
	m := make(map[string]int)
	lim := int(math.Pow(10.0, 4))
	for i := 1; i <= lim; i++ {
		var deck []string
		_ = json.Unmarshal([]byte(`["1","2","3","4","5","6","7"]`), &deck)
		s := DeckToShuffleRandomPool(deck)
		var k string
		// 抽空牌组
		for range deck {
			k = s.Pick().(string)
		}
		// 最后一张是什么
		k = fmt.Sprintf("Last=%s", k)
		m[k] += 1
	}
	fmt.Println(m) // map[Last=1:1413 Last=2:1452 Last=3:1420 Last=4:1401 Last=5:1423 Last=6:1460 Last=7:1431]
}
