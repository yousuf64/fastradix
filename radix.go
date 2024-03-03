package radix

import (
	"sort"
	"strings"
	"unsafe"
)

type Tree[T comparable] struct {
	*node[T]
}

func New[T comparable]() *Tree[T] {
	return &Tree[T]{&node[T]{}}
}

func (t *Tree[T]) Add(key string, value T) {
	parent := t.node
	if key == "" {
		parent.value = value
		return
	}

LOOP:
	if c := key[0]; c >= parent.min && c <= parent.max {
		var nd *node[T]
		var index int
		for i, b := range []byte(parent.indices) {
			if c == b {
				nd = parent.children[i]
				index = i
				break
			}
		}
		if nd == nil {
			// Create here.
			goto FALLBACK
		}

		longest := longestCommonPrefix(key, nd.prefix)
		if longest == len(nd.prefix) {
			// Traversal.
			// pfx: /posts
			// seg: /posts|/upsert
			parent = nd
			key = key[len(nd.prefix):]
			if key != "" {
				goto LOOP
			}

			// Replace.
			nd.value = value
			return
		} else if longest == len(key) {
			// Expansion.
			// pfx: categories|/skus
			// seg: categories|
			branchNode := &node[T]{prefix: nd.prefix[:longest], value: value, children: make([]*node[T], 1)}
			nd.prefix = nd.prefix[longest:]
			branchNode.children[0] = nd
			branchNode.index()

			parent.children[index] = branchNode
			parent.index()
			return
		} else {
			// Collision.
			// pfx: cat|egories
			// seg: cat|woman
			newNode := &node[T]{prefix: key[longest:], value: value}
			branchNode := &node[T]{prefix: nd.prefix[:longest], children: make([]*node[T], 2)}
			nd.prefix = nd.prefix[longest:]
			branchNode.children[0] = nd
			branchNode.children[1] = newNode
			branchNode.index()

			parent.children[index] = branchNode
			//parent.index()
			return
		}
	}

FALLBACK:
	parent.children = append(parent.children, &node[T]{
		prefix: key,
		value:  value,
	})
	parent.index()
}

func longestCommonPrefix(s1, s2 string) int {
	maxLen := len(s1)
	if length := len(s2); length < maxLen {
		maxLen = length
	}

	i := 0
	for ; i < maxLen; i++ {
		if s1[i] != s2[i] {
			break
		}
	}
	return i
}

func (t *Tree[T]) Search(key string) (T, bool) {
	parent := t.node
	if key == "" {
		return parent.value, !parent.IsZero()
	}

LOOP:

	if c := key[0]; c >= parent.min && c <= parent.max {
		var nd *node[T]
		for i, b := range []byte(parent.indices) {
			if c == b {
				nd = parent.children[i]
			}
		}
		if nd == nil {
			return *new(T), false
		}

		if key == nd.prefix {
			// reached the end.
			return nd.value, !nd.IsZero()
		} else if strings.HasPrefix(key, nd.prefix) {
			// dfs into it.
			parent = nd
			key = key[len(parent.prefix):]
			goto LOOP
		}
	}

	return *new(T), false
}

func (t *Tree[T]) StartsWith(s string) (KVs []KV[T]) {
	prefix := ""
	parent := t.node

LOOP:
	if len(s) > 0 {
		if c := s[0]; c >= parent.min && c <= parent.max {
			var nd *node[T]
			for i, b := range []byte(parent.indices) {
				if c == b {
					nd = parent.children[i]
				}
			}

			if strings.HasPrefix(s, nd.prefix) {
				// dfs into it.
				parent = nd
				s = s[len(parent.prefix):]
				prefix = prefix + parent.prefix
				goto LOOP
			} else if strings.HasPrefix(nd.prefix, s) {
				parent = nd
				s = s[:0]
				prefix = prefix + parent.prefix
			}
		}
	}

	if len(s) > 0 {
		panic("not exhausted")
	}

	// Add last matched node if having a valid value.
	//if !parent.IsZero() {
	//	KVs = append(KVs, KV[T]{
	//		Key:   parent.prefix,
	//		Value: parent.value,
	//	})
	//}

	// DFS recursively and add KVs.
	parent.dfs(prefix[:len(prefix)-len(parent.prefix)], func(kv KV[T]) {
		KVs = append(KVs, kv)
	})
	return
}

func (t *Tree[T]) DeletePrefix(prefix string) bool {
	parent := t.node
	if prefix == "" {
		if hasValues := !parent.IsZero() || len(parent.children) > 0; hasValues {
			parent.value = *new(T)
			parent.children = nil
			parent.index()
			return true
		}
		return false
	}

LOOP:
	if c := prefix[0]; c >= parent.min && c <= parent.max {
		var nd *node[T]
		var index int
		for i, b := range []byte(parent.indices) {
			if c == b {
				nd = parent.children[i]
				index = i
			}
		}
		if nd == nil {
			return false
		}
		if strings.HasPrefix(nd.prefix, prefix) {
			parent.children = append(parent.children[:index], parent.children[index+1:]...)
			parent.index()

			// Should merge?
			if parent != t.node && len(parent.children) == 1 && parent.IsZero() {
				parent.prefix = parent.prefix + parent.children[0].prefix
				parent.value = parent.children[0].value
				parent.children = parent.children[0].children
				parent.index()
			}
			return true
		} else if strings.HasPrefix(prefix, nd.prefix) {
			parent = nd
			prefix = prefix[len(parent.prefix):]
			goto LOOP
		}
	}
	return false
}

func (t *Tree[T]) Delete(key string) bool {
	parent := t.node
	if key == "" {
		if !parent.IsZero() {
			parent.value = *new(T)
			return true
		}
		return false
	}

LOOP:
	if c := key[0]; c >= parent.min && c <= parent.max {
		var nd *node[T]
		var index int
		for i, b := range []byte(parent.indices) {
			if c == b {
				nd = parent.children[i]
				index = i
			}
		}
		if nd == nil {
			return false
		}
		if key == nd.prefix {
			// reached the end.
			if nd.IsZero() {
				return false
			}

			nd.value = *new(T)
			if len(nd.children) == 0 {
				// Remove node.
				parent.children = append(parent.children[:index], parent.children[index+1:]...)

				// Merge sibling to parent.
				if parent != t.node && len(parent.children) == 1 && parent.IsZero() {
					parent.prefix = parent.prefix + parent.children[0].prefix
					parent.value = parent.children[0].value
					parent.children = parent.children[0].children
				}

				parent.index()
			} else if len(nd.children) == 1 {
				// Merge child to node.
				nd.prefix = nd.prefix + nd.children[0].prefix
				nd.value = nd.children[0].value
				nd.children = nd.children[0].children
				nd.index()
			}
			return true
		} else if strings.HasPrefix(key, nd.prefix) {
			// dfs into it.
			parent = nd
			key = key[len(parent.prefix):]
			goto LOOP
		}
	}
	return false
}

func (t *Tree[T]) Has(key string) bool {
	_, ok := t.Search(key)
	return ok
}

type KV[T comparable] struct {
	Key   string
	Value T
}

func (t *Tree[T]) DFSWalk(f func(KV[T])) {
	node := t.node
	node.dfs(node.prefix, f)
}

func (n *node[T]) dfs(prefix string, fn func(kv KV[T])) {
	prefix = prefix + n.prefix

	if !n.IsZero() {
		fn(KV[T]{prefix, n.value})
	}

	for _, child := range n.children {
		child.dfs(prefix, fn)
	}
}

func (t *Tree[T]) BFSWalk(fn func(KV[T])) {
	prefixes := make([]string, 0, 32)
	prefixes = append(prefixes, t.node.prefix)
	nodes := make([]*node[T], 0, 32)
	nodes = append(nodes, t.node)

	for i := 0; ; i++ {
		if i >= len(nodes) {
			break
		}

		node := nodes[i]
		prefix := prefixes[i] + node.prefix
		if !node.IsZero() {
			fn(KV[T]{prefix, node.value})
		}

		nodes = append(nodes, node.children...)
		for k := 0; k < len(node.children); k++ {
			prefixes = append(prefixes, prefix)
		}
	}
}

type node[T comparable] struct {
	prefix   string
	value    T
	children []*node[T]
	indices  string
	min      uint8
	max      uint8
}

func (n *node[T]) IsZero() bool {
	return n.value == *new(T)
}

func (n *node[T]) index() {
	if len(n.children) == 0 {
		n.indices = ""
		n.min = 0
		n.max = 0
		return
	}

	// Sort children by prefix's first char.
	sort.Slice(n.children, func(i, j int) bool {
		return n.children[i].prefix[0] < n.children[j].prefix[0]
	})

	n.min = n.children[0].prefix[0]
	n.max = n.children[len(n.children)-1].prefix[0]

	indices := make([]byte, len(n.children))
	for i, child := range n.children {
		indices[i] = child.prefix[0]
	}
	n.indices = unsafeBytesToString(indices)
}

func unsafeBytesToString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}
