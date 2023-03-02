package radix

import (
	"sort"
	"strings"
	"unsafe"
)

type Tree struct {
	*node
}

func New() *Tree {
	return &Tree{&node{}}
}

func (t *Tree) Add(key string, value interface{}) {
	parent := t.node
	if key == "" {
		parent.value = value
		return
	}

LOOP:
	if c := key[0]; c >= parent.min && c <= parent.max {
		var nd *node
		var index int
		for i, b := range []byte(parent.indices) {
			if c == b {
				nd = parent.children[i]
				index = i
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
			branchNode := &node{prefix: nd.prefix[:longest], value: value, children: make([]*node, 1)}
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
			newNode := &node{prefix: key[longest:], value: value}
			branchNode := &node{prefix: nd.prefix[:longest], children: make([]*node, 2)}
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
	parent.children = append(parent.children, &node{
		prefix: key,
		value:  value,
	})
	parent.index()
}

func longestCommonPrefix(s1, s2 string) int {
	max := len(s1)
	if length := len(s2); length < max {
		max = length
	}

	i := 0
	for ; i < max; i++ {
		if s1[i] != s2[i] {
			break
		}
	}
	return i
}

func (t *Tree) Search(key string) interface{} {
	parent := t.node
	if key == "" {
		return parent.value
	}

LOOP:

	if c := key[0]; c >= parent.min && c <= parent.max {
		var nd *node
		for i, b := range []byte(parent.indices) {
			if c == b {
				nd = parent.children[i]
			}
		}
		if nd == nil {
			return nil
		}

		if key == nd.prefix {
			// reached the end.
			return nd.value
		} else if strings.HasPrefix(key, nd.prefix) {
			// dfs into it.
			parent = nd
			key = key[len(parent.prefix):]
			goto LOOP
		}
	}

	return nil
}

func (t *Tree) DeletePrefix(prefix string) bool {
	parent := t.node
	if prefix == "" {
		if hasValues := parent.value != nil || len(parent.children) > 0; hasValues {
			parent.value = nil
			parent.children = nil
			parent.index()
			return true
		}
		return false
	}

LOOP:
	if c := prefix[0]; c >= parent.min && c <= parent.max {
		var nd *node
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
			if parent != t.node && len(parent.children) == 1 && parent.value == nil {
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

func (t *Tree) Delete(key string) bool {
	parent := t.node
	if key == "" {
		if parent.value != nil {
			parent.value = nil
			return true
		}
		return false
	}

LOOP:
	if c := key[0]; c >= parent.min && c <= parent.max {
		var nd *node
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
			if nd.value == nil {
				return false
			}

			nd.value = nil
			if len(nd.children) == 0 {
				// Remove node.
				parent.children = append(parent.children[:index], parent.children[index+1:]...)

				// Merge sibling to parent.
				if parent != t.node && len(parent.children) == 1 && parent.value == nil {
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

func (t *Tree) Has(key string) bool {
	return t.Search(key) != nil
}

type KV struct {
	Key   string
	Value interface{}
}

func (t *Tree) DFSWalk(f func(KV)) {
	node := t.node
	node.dfs(node.prefix, f)
}

func (n *node) dfs(prefix string, f func(kv KV)) {
	prefix = prefix + n.prefix

	if n.value != nil {
		f(KV{prefix, n.value})
	}

	for _, child := range n.children {
		child.dfs(prefix, f)
	}
}

type node struct {
	prefix   string
	value    interface{}
	children []*node
	indices  string
	min      uint8
	max      uint8
}

func (n *node) index() {
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
	return *(*string)(unsafe.Pointer(&b))
}
