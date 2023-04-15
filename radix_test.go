package radix

import (
	"testing"
)

type kv struct {
	key   string
	value interface{}
}

var basicKvs = []kv{
	{key: "apple", value: 10},
	{key: "mango", value: 20},
	{key: "manchester", value: 30},
	{key: "main", value: 40},
	{key: "mongodb", value: 50},
	{key: "mongoose", value: 60},
	{key: "app", value: 70},
	{key: "amsterdam", value: 80},
	{key: "everest", value: 90},
	{key: "docker", value: 100},
	{key: "dominoes", value: 110},
	{key: "duckduckgo", value: 120},
}

func TestTree_Add_Search(t *testing.T) {
	testTree(t, basicKvs)
}

func testTree(t *testing.T, kvs []kv) {
	tr := New()

	for _, kv := range kvs {
		tr.Add(kv.key, kv.value)
	}

	for _, kv := range kvs {
		v := tr.Search(kv.key)
		if kv.value != v {
			t.Fatalf("expected: %v, got: %v", kv.value, v)
		}
	}
}

func TestTree_Delete(t *testing.T) {
	tr := New()

	for _, kv := range basicKvs {
		tr.Add(kv.key, kv.value)
	}

	for _, kv := range basicKvs {
		v := tr.Search(kv.key)
		if v == nil {
			t.Fatalf("expected a value for %s", kv.key)
		}

		ok := tr.Delete(kv.key)
		if !ok {
			t.Fatalf("expected to delete %s", kv.key)
		}

		v = tr.Search(kv.key)
		if v != nil {
			t.Fatalf("expected %s to be deleted", kv.key)
		}
	}
}

func TestTree_DeletePrefix(t *testing.T) {
	tr := New()

	tt := map[string][]kv{
		"kube": {
			{
				key:   "kubernetes",
				value: 911,
			},
			{
				key:   "kubectl",
				value: "abc",
			},
		},
		"hash": {
			{
				key:   "hash",
				value: "foo",
			},
			{
				key:   "hashmap",
				value: true,
			},
			{
				key:   "hashicorp",
				value: 0xF5,
			},
		},
		"orders": {
			{
				key:   "orders/:id",
				value: "get orders by id",
			},
			{
				key:   "orders/create",
				value: "create new order",
			},
			{
				key:   "orders/all",
				value: "get all orders",
			},
			{
				key:   "orders/update/:id",
				value: "update order by id",
			},
			{
				key:   "orders/delete/:id",
				value: "delete order by id",
			},
		},
	}

	// Insert.
	for _, kvs := range tt {
		for _, kv := range kvs {
			tr.Add(kv.key, kv.value)
		}
	}

	// Assert insertion.
	for _, kvs := range tt {
		for _, kv := range kvs {
			v := tr.Search(kv.key)
			if v == nil {
				t.Fatalf("expected a value for %s", kv.key)
			}
		}
	}

	// Delete Prefix.
	for prefix, kvs := range tt {
		ok := tr.DeletePrefix(prefix)
		if !ok {
			t.Fatalf("expected to delete for prefix %s", prefix)
		}

		// Re-attempt.
		ok = tr.DeletePrefix(prefix)
		if ok {
			t.Fatalf("expected to have been deleted already for %s", prefix)
		}

		// Assert deletion.
		for _, kv := range kvs {
			v := tr.Search(kv.key)
			if v != nil {
				t.Fatalf("expected %s to have been deleted", kv.key)
			}
		}
	}
}

func TestTree_DFSWalk(t *testing.T) {
	tr := New()

	for _, kv := range basicKvs {
		tr.Add(kv.key, kv.value)
	}

	walked := make(map[string]interface{}, len(basicKvs))
	tr.DFSWalk(func(kv KV) {
		walked[kv.Key] = kv.Value
	})

	for _, kv := range basicKvs {
		value, ok := walked[kv.key]
		if !ok {
			t.Fatalf("expected to have walked over '%s'", kv.key)
		}
		if value != kv.value {
			t.Fatalf("for '%s' expected: %v, got: %v", kv.key, kv.value, value)
		}
	}
}

func TestTree_BFSWalk(t *testing.T) {
	tr := New()

	for _, kv := range basicKvs {
		tr.Add(kv.key, kv.value)
	}

	walked := make(map[string]interface{}, len(basicKvs))
	tr.BFSWalk(func(kv KV) {
		walked[kv.Key] = kv.Value
	})

	for _, kv := range basicKvs {
		value, ok := walked[kv.key]
		if !ok {
			t.Fatalf("expected to have walked over '%s'", kv.key)
		}
		if value != kv.value {
			t.Fatalf("for '%s' expected: %v, got: %v", kv.key, kv.value, value)
		}
	}
}
