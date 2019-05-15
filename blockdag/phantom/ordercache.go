// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package phantom

const (
	// We'll cache the order of a graph every interval of additions to the graph
	maxOrderCacheSize = 200
)

type orderCache struct {
	// Tips for OrderDAG calls
	tips [][]*node
	// blueSet for OrderDAG calls
	blueSet []*nodeSet
	// The cache of OrderDAG results, limited to maxOrderCacheSize length
	order [][]*node
}

func newOrderCache() *orderCache {
	return &orderCache{}
}

// add adds the values to the cache
func (oc *orderCache) add(tips []*node, blueSet *nodeSet, order []*node) {
	oc.tips = append(oc.tips, tips)
	oc.blueSet = append(oc.blueSet, blueSet)
	oc.order = append(oc.order, order)

	if oc.size() > maxOrderCacheSize {
		// Prune order cache size
		pruneFrom := oc.size() - maxOrderCacheSize

		oc.tips = oc.tips[pruneFrom:]
		oc.blueSet = oc.blueSet[pruneFrom:]
		oc.order = oc.order[pruneFrom:]
	}
}

// canUseCache returns true if the cached order data is old enough to be considered safe to use
// as a continuation point of calculating dag order.
func (oc *orderCache) canUseCache() bool {
	if oc.size() >= maxOrderCacheSize {
		return true
	}

	return false
}

// returns a graph made from the oldest dag order, with a virtual node at the end,
// whose parents are the tips of the oldest dag order.
func (oc *orderCache) getOldestVirtual() *Graph {
	vg := NewGraph()

	order := oc.oldestOrder()
	tips := oc.oldestTips()
	if order == nil || tips == nil {
		return nil
	}

	for _, n := range order {
		vg.nodes[n.id] = n
	}

	vnode := newNode("VIRTUAL")
	vg.AddNode(vnode)
	for _, tip := range tips {
		vg.AddEdge(vnode, tip)
	}

	return vg
}

// oldestBlueSet returns the oldest blue set cached from OrderDAG calls
func (oc *orderCache) oldestBlueSet() *nodeSet {
	if oc.size() == 0 {
		return nil
	}

	return oc.blueSet[0]
}

// oldestOrder returns the oldest dag order cached from OrderDAG calls
func (oc *orderCache) oldestOrder() []*node {
	if oc.size() == 0 {
		return nil
	}

	return oc.order[0]
}

// oldestTips returns the oldest tips cached from OrderDAG calls
func (oc *orderCache) oldestTips() []*node {
	if oc.size() == 0 {
		return nil
	}

	return oc.tips[0]
}

// size returns the number of cached OrderDAG results
func (oc *orderCache) size() int {
	return len(oc.order)
}