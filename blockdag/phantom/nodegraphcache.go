// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package phantom

import (
	"sort"
	"sync"
)

const (
	// The maximum size of the NodeGraphCache
	maxNodeGraphCacheSize = 5000

	// How many adds we'll process before triggering expiry, when cache is full
	cacheFullExpireInterval = 100
)


type NodeGraphCacheEntry struct {
	// Having a pointer to the Node makes working with sorted entries easier.
	Node  *Node
	Graph *Graph
	hits  int
}

type NodeGraphCache struct {
	Name string
	// A map of a Node to its calculated past or future from Graph.getPast or Graph.getFuture
	cache map[*Node]*NodeGraphCacheEntry

	// This is used to help determine if we should trigger an expiry when the cache is full
	addsSinceLastExpire int

	sync.RWMutex
}

// Define an interface we can use to sort the NodeGraphCacheEntry elements by hits
type byHits []*NodeGraphCacheEntry
func (bh byHits) Len() int { return len(bh) }
func (bh byHits) Swap(i, j int) { bh[i], bh[j] = bh[j], bh[i] }
func (bh byHits) Less(i, j int) bool { return bh[i].hits < bh[j].hits }

// NewNodeGraphCache returns a NodeGraphCache, which can be used to help reduce repetitive
// computation for Node connectivity.
func NewNodeGraphCache(name string) *NodeGraphCache {
	return &NodeGraphCache{
		Name:  name,
		cache: make(map[*Node]*NodeGraphCacheEntry),
	}
}

// add adds an entry to the past cache
func (ngc *NodeGraphCache) add(n *Node, g *Graph) {
	size := len(ngc.cache)
	if size >= maxNodeGraphCacheSize && ngc.addsSinceLastExpire >= cacheFullExpireInterval {
		amt := (size - maxNodeGraphCacheSize) + 1
		ngc.expire(amt)
		ngc.addsSinceLastExpire = 0
	}

	ngc.cache[n] = &NodeGraphCacheEntry{
		Node:  n,
		Graph: g,
	}

	ngc.addsSinceLastExpire += 1
}

// Add adds an entry to the past cache
func (ngc *NodeGraphCache) Add(n *Node, g *Graph) {
	ngc.Lock()
	defer ngc.Unlock()

	ngc.add(n, g)
}

// delete deletes an entry from the cache
func (ngc *NodeGraphCache) delete(n *Node) {
	delete(ngc.cache, n)
}

// Delete deletes an entry from the cache
func (ngc *NodeGraphCache) Delete(n *Node) {
	ngc.Lock()
	defer ngc.Unlock()

	ngc.delete(n)
}

// expire removes a number of least-used entries from the cache
func (ngc *NodeGraphCache) expire(amt int) {
	if amt <= 0 {
		return
	}

	size := len(ngc.cache)
	if size < amt {
		amt = size
	}

	entries := make([]*NodeGraphCacheEntry, size)
	index := 0
	for _, e := range ngc.cache {
		entries[index] = e
		index += 1
	}

	sort.Sort(byHits(entries))

	for i := 0; i < amt; i++ {
		delete(ngc.cache, entries[i].Node)
	}
}

// Expire removes a number of least-used entries from the cache
func (ngc *NodeGraphCache) Expire(amt int) {
	ngc.Lock()
	defer ngc.Unlock()

	ngc.expire(amt)
}

// get returns a cache entry and bool indicating cache hit or miss. Entry is nil if not found.
func (ngc *NodeGraphCache) get(n *Node) (*NodeGraphCacheEntry, bool) {
	e, ok := ngc.cache[n]
	if !ok {
		return nil, false
	}

	e.hits += 1
	return e, true
}

// Get returns a cache entry and bool indicating cache hit or miss. Entry is nil if not found.
func (ngc *NodeGraphCache) Get(n *Node) (*NodeGraphCacheEntry, bool) {
	ngc.RLock()
	defer ngc.RUnlock()

	return ngc.get(n)
}


