// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package phantom

import (
	"fmt"
	"reflect"
	"testing"
)

func TestNodeGraphCache_Add(t *testing.T) {
	g := createGraph()
	f := g.GetNodeById("F")
	past := g.GetPast(f)

	ngc := NewNodeGraphCache("test")

	if len(ngc.cache) != 0 {
		t.Errorf("new node graph cache is not empty; got %d entries, want %d entries", len(ngc.cache), 0)
	}

	ngc.Add(f, past)

	if len(ngc.cache) != 1 {
		t.Errorf("wrong cache size; got %d, want %d", len(ngc.cache), 1)
	}

	e, ok := ngc.cache[f]
	if !ok {
		t.Errorf("failed to create new cache entry for %s", f.GetId())
	}

	if e.hits != 0 {
		t.Errorf("new cache entry should have no hits; got %d, want %d", e.hits, 0)
	}

	ngc.Delete(f)

	// Test cache expiry due to Add()
	nodes := make([]*Node, 0)
	for i := 0; i < maxNodeGraphCacheSize + 1; i++ {
		n := newNode(fmt.Sprintf("X%d", i))
		ngc.Add(n, nil)
		nodes = append(nodes, n)

		if i != 0 {
			// Get cache entries for other nodes aside from the first, so that we'll know which
			// cache entry should have been picked for expiry
			_, _ = ngc.Get(n)
			_, _ = ngc.Get(n)
		}
	}

	if len(ngc.cache) != maxNodeGraphCacheSize {
		t.Errorf("wrong cache size; got %d, want %d", len(ngc.cache), maxNodeGraphCacheSize)
	}

	e, ok = ngc.Get(nodes[0])
	if ok {
		fmt.Println("cache size", len(ngc.cache))
		t.Errorf("cache entry for node %s should have been expired", nodes[0].GetId())
	}

	if e != nil {
		t.Errorf("entry should be nil for expired cache entry %s", nodes[0].GetId())
	}

	for i := 1; i < len(nodes); i++ {
		e, ok = ngc.Get(nodes[i])
		if !ok {
			t.Errorf("cache miss for %s", nodes[i].GetId())
		}

		if !reflect.DeepEqual(e.Node, nodes[i]) {
			t.Errorf("cache entry %s doesn't match node %s", e.Node.GetId(), nodes[i].GetId())
		}
	}
}

func TestNodeGraphCache_Delete(t *testing.T) {
	g := createGraph()
	f := g.GetNodeById("F")
	h := g.GetNodeById("H")

	ngc := NewNodeGraphCache("test")

	ngc.Add(f, g.GetPast(f))
	ngc.Add(h, g.GetPast(h))

	ngc.Delete(f)

	e, ok := ngc.cache[f]
	if ok {
		t.Errorf("failed to delete cache entry for %s", f.GetId())
	}

	if e != nil {
		t.Errorf("entry should be nil for deleted cache entry %s", f.GetId())
	}

	if len(ngc.cache) != 1 {
		t.Errorf("wrong cache size; got %d, want %d", len(ngc.cache), 1)
	}

	ngc.Delete(nil)

	if len(ngc.cache) != 1 {
		t.Errorf("wrong cache size; got %d, want %d", len(ngc.cache), 1)
	}
}

func TestNodeGraphCache_Expire(t *testing.T) {
	g := createGraph()
	f := g.GetNodeById("F")
	past := g.GetPast(f)

	ngc := NewNodeGraphCache("test")
	ngc.Add(f, past)

	ngc.Expire(0)

	if len(ngc.cache) != 1 {
		t.Errorf("wrong cache size; got %d, want %d", len(ngc.cache), 1)
	}

	ngc.Expire(-1)

	if len(ngc.cache) != 1 {
		t.Errorf("wrong cache size; got %d, want %d", len(ngc.cache), 1)
	}

	ngc.Expire(1)

	if len(ngc.cache) != 0 {
		t.Errorf("wrong cache size; got %d, want %d", len(ngc.cache), 0)
	}

	ngc.Add(f, past)

	ngc.Expire(len(ngc.cache) * 2)

	if len(ngc.cache) != 0 {
		t.Errorf("wrong cache size; got %d, want %d", len(ngc.cache), 0)
	}
}

func TestNodeGraphCache_Get(t *testing.T) {
	g := createGraph()
	f := g.GetNodeById("F")
	past := g.GetPast(f)

	ngc := NewNodeGraphCache("test")

	ngc.Add(f, past)

	e, ok := ngc.Get(f)

	if !ok {
		t.Errorf("cache miss for %s", f.GetId())
	}

	if e.hits != 1 {
		t.Errorf("wrong number of cache hits; got %d, want %d", e.hits, 1)
	}

	if !reflect.DeepEqual(e.Node, f) {
		t.Errorf("cache entry %s doesn't match node %s", e.Node.GetId(), f.GetId())
	}

	if !reflect.DeepEqual(e.Graph, past) {
		t.Errorf("cache entry graph doesn't match past graph for %s", f.GetId())
	}

	banana := newNode("BANANA")
	e, ok = ngc.Get(banana)

	if ok {
		t.Errorf("hit for non-existent cache entry %s", banana.GetId())
	}

	if e != nil {
		t.Errorf("entry should be nil for non-existent cache entry %s", banana.GetId())
	}

	e, ok = ngc.Get(nil)
	if ok {
		t.Errorf("cache hit for non-existent cache entry %v", nil)
	}

	if e != nil {
		t.Errorf("entry should be nil for non-existent cache entry %v", nil)
	}
}


