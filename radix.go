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
