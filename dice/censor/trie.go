package censor

type node struct {
	end      bool
	content  string
	level    Level
	children map[rune]*node
}

func (n *node) insert(c rune) *node {
	if n.children == nil {
		n.children = make(map[rune]*node)
	}
	if cur, ok := n.children[c]; ok {
		return cur
	}
	n.children[c] = &node{}
	return n.children[c]
}

func (n *node) findChild(c rune) *node {
	if n.children == nil {
		return nil
	}
	if cur, ok := n.children[c]; ok {
		return cur
	}
	return nil
}

type trie struct {
	root *node
	size int
}

func newTire() *trie {
	return &trie{
		root: &node{},
	}
}

func (t *trie) Insert(key string, level Level) {
	cur := t.root
	for _, c := range key {
		cur = cur.insert(c)
	}
	cur.end = true
	cur.content = key
	cur.level = HigherLevel(level, cur.level)
	t.size++
}

func (t *trie) Match(text string) (sensitiveWords map[string]Level) {
	if t.root == nil {
		return nil
	}

	sensitiveWords = map[string]Level{}
	chars := []rune(text)
	for i := 0; i < len(chars); i++ {
		cur := t.root.findChild(chars[i])
		if cur == nil {
			continue
		}
		// 匹配到了前缀，开始检查是否完整匹配
		j := i + 1
		for ; cur != nil && j < len(chars); j++ {
			if cur.end {
				sensitiveWords[cur.content] = cur.level
				break
			}
			cur = cur.findChild(chars[j])
		}
		if cur != nil && j == len(chars) && cur.end {
			sensitiveWords[cur.content] = cur.level
		}
	}
	return sensitiveWords
}
