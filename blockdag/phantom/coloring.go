// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package phantom

import (
	"container/list"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type BlueSetCache struct {
	cache map[*Node]*nodeSet
	sync.RWMutex
}

func NewBlueSetCache() *BlueSetCache {
	return &BlueSetCache {
		cache: make(map[*Node]*nodeSet),
	}
}

// add the node associated with the set to the cache
func (blueset *BlueSetCache) add(n *Node, set *nodeSet) {
	blueset.cache[n] = set
}

// Add the node associated with the set to the cache
func (blueset *BlueSetCache) Add(n *Node, set *nodeSet) {
	blueset.Lock()
	defer blueset.Unlock()

	blueset.add(n, set)
}

func (blueset *BlueSetCache) GetBlueNodes(n *Node) []*Node {
	blueset.RLock()
	defer blueset.RUnlock()

	set, ok := blueset.cache[n]
	if !ok {
		return nil
	}

	return set.elements()
}

func (blueset *BlueSetCache) getBlueSet(n *Node) *nodeSet {
	set, exists := blueset.cache[n]
	if !exists {
		return nil
	}

	return set
}

// GetBlueSet returns the nodeSet associated with the node
func (blueset *BlueSetCache) GetBlueSet(n *Node) *nodeSet {
	blueset.RLock()
	defer blueset.RUnlock()

	return blueset.getBlueSet(n)
}

// inCache returns true if the node is in the blueset cache
func (blueset *BlueSetCache) inCache(n *Node) bool {
	_, exists := blueset.cache[n]
	return exists
}

// InCache returns true if the node is in the blueset cache
func (blueset *BlueSetCache) InCache(n *Node) bool {
	blueset.RLock()
	defer blueset.RUnlock()

	return blueset.inCache(n)
}

// String returns a string representing the current cache state
func (blueSet *BlueSetCache) String() string {
	var sb strings.Builder

	nodes := make([]*Node, 0, len(blueSet.cache))

	for k := range blueSet.cache {
		nodes = append(nodes, k)
	}

	less := func(i, j int) bool {
		if nodes[i].GetId() < nodes[j].GetId() {
			return true
		} else {
			return false
		}
	}

	sort.Slice(nodes, less)

	for _, n := range nodes {
		ons, ok := blueSet.cache[n]
		if !ok {
			sb.WriteString(fmt.Sprintln(n.GetId(), "\t", "UNKNOWN"))
			continue
		}

		sb.WriteString(fmt.Sprintln(n.GetId(), "\t", GetIds(ons.elements())))
	}

	return sb.String()
}

// implements Algorithm 3 Selection of a blue set of Phantom paper
func calculateBlueSet(g *Graph, genesisNode *Node, k int, blueSetCache *BlueSetCache) *nodeSet {
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

	tipToSet := make(map[*Node]*nodeSet)

	for _, tipBlock := range g.getTips() {
		nodePast := g.getPast(tipBlock)

		var pastBlueSet *nodeSet
		if blueSetCache != nil && tipBlock.GetId() != "VIRTUAL" {
			if !blueSetCache.InCache(tipBlock) {
				pastBlueSet = calculateBlueSet(nodePast, genesisNode, k, blueSetCache)
				pastBlueSet.add(tipBlock)
				blueSetCache.Add(tipBlock, pastBlueSet)
			} else {
				pastBlueSet = blueSetCache.GetBlueSet(tipBlock)
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

// OrderDAG returns the graphs' tips and order
func OrderDAG(g *Graph, genesisNode *Node, k int, blueSetCache *BlueSetCache, minHeight int32, orderCache *OrderCache) ([]*Node, []*Node, error) {
	g.RLock()
	defer g.RUnlock()

	var todoQueue = list.New()
	var seen = make(map[*Node]struct{})
	var orderingSet = newOrderedNodeSet()

	var tips = g.getTips()
	var vg = g.getVirtual()
	vNode := vg.GetNodeById("VIRTUAL")
	defer func() {
		if vNode == nil {
			return
		}

		// When Graph.getVirtual() is called, a new VIRTUAL node is created and is connected to the current tips
		// as its parents.
		// We want to remove the link to this new VIRTUAL node from the tips after we've finished processing.
		//
		// If left in place, the VIRTUAL node would cause the todoQueue processing to exit before we expect it to,
		// the next time that OrderDAG is called.
		for _, n := range tips {
			vg.removeEdge(vNode, n)
		}
	}()

	// We calculate blueSet from tips down
	var blueSet = calculateBlueSet(vg, genesisNode, k, blueSetCache)

	var entry *OrderCacheEntry
	var cacheHit bool
	if orderCache != nil {
		entry, cacheHit = orderCache.Get(minHeight)
	}

	if cacheHit && entry != nil {
		// Load the cached node order
		for _, n := range entry.Order {
			orderingSet.add(n)
		}

		// Start calculating order from the tips of the cache up, instead of the genesis node
		for _, tn := range entry.Tips {
			todoQueue.PushBack(tn)
			seen[tn] = keyExists
		}
	} else {
		// Start calculating ordering from genesis node
		todoQueue.PushBack(genesisNode)
		seen[genesisNode] = keyExists
	}

	for todoQueue.Len() > 0 {
		// pop from front of queue
		// and add to ordering
		elem := todoQueue.Front()
		todoQueue.Remove(elem)
		node := elem.Value.(*Node)
		if node.GetId() == "VIRTUAL" {
			break
		}
		orderingSet.add(node)
		//log.Debugf("Added %s to order", node.GetId())

		children := node.getChildren()
		intersect := blueSet.intersection(children)
		anticone := vg.GetAnticone(node)

		// for each child of node in the blue set
		for _, blueChild := range intersect.elements() {
			childPast := vg.GetPast(blueChild)

			// get all node in its past that were in its parent's anticone
			// nodes topologically before child, but possibly not in the blue set
			// add to queue
			for _, anticoneNode := range anticone.elements() {
				//log.Debugf("%s in anticone of %s", anticoneNode.GetId(), node.GetId())
				if childPast.GetNodeById(anticoneNode.GetId()) != nil {
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

	if orderCache != nil {
		if _, ok := orderCache.CanAdd(minHeight); ok {
			err := orderCache.Add(minHeight, tips, orderingSet.elements())
			if err != nil {
				return tips, orderingSet.elements(), err
			}
		}
	}

	return tips, orderingSet.elements(), nil
}