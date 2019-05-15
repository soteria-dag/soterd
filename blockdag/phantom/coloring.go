// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package phantom

import (
	"container/list"
	"sort"
)

type orderedNodeSet struct {
	nodes map[*node]struct{}
	order map[int]*node
	counter int
}

func newOrderedNodeSet() *orderedNodeSet {
	return &orderedNodeSet {
		nodes: make(map[*node]struct{}),
		order: make(map[int]*node),
		counter: 0,
	}
}

func (ons *orderedNodeSet) add(node *node) {
	_, ok := ons.nodes[node]
	if !ok {
		ons.counter += 1
		ons.nodes[node] = keyExists
		ons.order[ons.counter] = node
	}
}

func (ons *orderedNodeSet) contains(node *node) bool {
	_, ok := ons.nodes[node]

	return ok
}

func (ons *orderedNodeSet) getNodes() []*node {
	var indexes = make([]int, 0, len(ons.order))
	var nodes = make([]*node, 0, len(ons.order))
	for k := range ons.order {
		indexes = append(indexes, k)
	}

	sort.Ints(indexes)

	for _, v := range indexes {
		nodes = append(nodes, ons.order[v])
	}
	return nodes
}

type BlueSetCache struct {
	cache map[*node]*nodeSet
}

func NewBlueSetCache() *BlueSetCache {
	return &BlueSetCache {
		cache: make(map[*node]*nodeSet),
	}
}

func (blueset *BlueSetCache) GetBlueNodes(n *node) []*node {

	set, ok := blueset.cache[n]
	if !ok {
		return nil
	}

	return set.elements()
}

// implements Algorithm 3 Selection of a blue set of Phantom paper
func calculateBlueSet(g *Graph, genesisNode *node, k int, blueSetCache *BlueSetCache) *nodeSet {
	blueSet := newNodeSet()

	// assumes genesisNode is in the graph
	// TODO(jenlouie): check and throw error
	// g.GetNodeById(genesisNode.GetId()) != nil

	if g.getSize() == 1 {
		if genesisNode != nil {
			blueSet.add(genesisNode)
		}
		return blueSet
	}

	tipToSet := make(map[*node]*nodeSet)

	for _, tipBlock := range g.getTips() {

		nodePast := g.getPast(tipBlock)

		var pastBlueSet *nodeSet
		if blueSetCache != nil {
			if _, ok := blueSetCache.cache[tipBlock]; !ok {
				pastBlueSet = calculateBlueSet(nodePast, genesisNode, k, blueSetCache)
				pastBlueSet.add(tipBlock)
				blueSetCache.cache[tipBlock] = pastBlueSet
			} else {
				pastBlueSet = blueSetCache.cache[tipBlock]
			}
			pastBlueSet = pastBlueSet.clone()
		} else {
			pastBlueSet = calculateBlueSet(nodePast, genesisNode, k, blueSetCache)
			pastBlueSet.add(tipBlock)
		}

		anticone := g.getAnticone(tipBlock)

		for _, node := range anticone.elements() {
			anticoneC := g.getAnticone(node)
			blueIntersect := anticoneC.intersection(pastBlueSet)
			if blueIntersect.size() <= k {
				pastBlueSet.add(node)
			}
		}
		tipToSet[tipBlock] = pastBlueSet
	}

	var setSize = 0
	for _, tip := range g.getTips() {
		var v = tipToSet[tip]
		//log.Debugf("Tip %s has blue set size %d", tip.GetId(), v.Size())
		if v.size() > setSize {
			setSize = v.size()
			blueSet = v
		}
	}

	return blueSet
}

// need to create a graph with a virtual node
func OrderDAG(g *Graph, genesisNode *node, k int, blueSetCache *BlueSetCache) []*node {

	g.RLock()
	defer g.RUnlock()

	todoQueue := list.New()
	seen := make(map[*node]struct{})
	orderingSet := newOrderedNodeSet()
	vg := g.getVirtual()
	var blueSet *nodeSet

	oldestCacheTips := g.orderCache.oldestTips()
	tips := g.getTips()

	// Determine if we should use orderCache
	if g.orderCache.canUseCache() {
		// Load the cached node order, from the oldest cache
		for _, n := range g.orderCache.oldestOrder() {
			orderingSet.add(n)
		}

		// Use the cached blueSet
		blueSet = g.orderCache.oldestBlueSet()

		// Start calculating from the tips of the cache, instead of genesis
		for _, n := range oldestCacheTips {
			todoQueue.PushBack(n)
			seen[n] = keyExists
		}
	} else {
		// Start ordering from genesis node
		blueSet = calculateBlueSet(vg, genesisNode, k, blueSetCache)
		todoQueue.PushBack(genesisNode)
		seen[genesisNode] = keyExists
	}

	for todoQueue.Len() > 0 {
		// pop from front of queue
		// and add to ordering
		elem := todoQueue.Front()
		todoQueue.Remove(elem)
		node := elem.Value.(*node)
		if node.GetId() == "VIRTUAL" {
			break
		}
		orderingSet.add(node)
		//log.Debugf("Added %s to order", node.GetId())

		children := node.getChildren()
		intersect := blueSet.intersection(children)
		anticone := vg.getAnticone(node)

		// for each child of node in the blue set
		for _, blueChild := range intersect.elements() {
			childPast := vg.getPast(blueChild)

			// get all node in its past that were in its parent's anticone
			// nodes topologically before child, but possibly not in the blue set
			// add to queue
			for _, anticoneNode := range anticone.elements() {
				//log.Debugf("%s in anticone of %s", anticoneNode.GetId(), node.GetId())
				if childPast.getNodeById(anticoneNode.GetId()) != nil {
					//log.Debugf("%s is in %s 's past", anticoneNode.GetId(), blueChild.GetId())
					_, ok := seen[anticoneNode]
					if !orderingSet.contains(anticoneNode) && !ok {
						todoQueue.PushBack(anticoneNode)
						seen[anticoneNode] = keyExists
						//log.Debugf("Adding %s to queue", anticoneNode.GetId())
					}
				}
			}
			// then add child to queue
			_, ok := seen[blueChild]
			if !ok {
				todoQueue.PushBack(blueChild)
				seen[blueChild] = keyExists
				//log.Debugf("Adding %s to queue", blueChild.GetId())
			}
		}
	}

	order := orderingSet.getNodes()

	// Cache the calculations made, to help improve ordering performance in future calls.
	g.orderCache.add(tips, blueSet, order)

	return order
}

// OrderDAGNoCache returns the order of nodes in the graph, without using ordering cache
func OrderDAGNoCache(g *Graph, genesisNode *node, k int, blueSetCache *BlueSetCache) []*node {

	g.RLock()
	defer g.RUnlock()

	todoQueue := list.New()
	seen := make(map[*node]struct{})
	orderingSet := newOrderedNodeSet()

	vg := g.getVirtual()
	blueSet := calculateBlueSet(vg, genesisNode, k, blueSetCache)
	todoQueue.PushBack(genesisNode)
	seen[genesisNode] = keyExists

	for todoQueue.Len() > 0 {
		// pop from front of queue
		// and add to ordering
		elem := todoQueue.Front()
		todoQueue.Remove(elem)
		node := elem.Value.(*node)
		if node.GetId() == "VIRTUAL" {
			break
		}
		orderingSet.add(node)
		//log.Debugf("Added %s to order", node.GetId())

		children := node.getChildren()
		intersect := blueSet.intersection(children)
		anticone := vg.getAnticone(node)

		// for each child of node in the blue set
		for _, blueChild := range intersect.elements() {
			childPast := vg.getPast(blueChild)

			// get all node in its past that were in its parent's anticone
			// nodes topologically before child, but possibly not in the blue set
			// add to queue
			for _, anticoneNode := range anticone.elements() {
				//log.Debugf("%s in anticone of %s", anticoneNode.GetId(), node.GetId())
				if childPast.getNodeById(anticoneNode.GetId()) != nil {
					//log.Debugf("%s is in %s 's past", anticoneNode.GetId(), blueChild.GetId())
					_, ok := seen[anticoneNode]
					if !orderingSet.contains(anticoneNode) && !ok {
						todoQueue.PushBack(anticoneNode)
						seen[anticoneNode] = keyExists
						//log.Debugf("Adding %s to queue", anticoneNode.GetId())
					}
				}
			}
			// then add child to queue
			_, ok := seen[blueChild]
			if !ok {
				todoQueue.PushBack(blueChild)
				seen[blueChild] = keyExists
				//log.Debugf("Adding %s to queue", blueChild.GetId())
			}
		}
	}

	return orderingSet.getNodes()
}