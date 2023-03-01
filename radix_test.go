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
