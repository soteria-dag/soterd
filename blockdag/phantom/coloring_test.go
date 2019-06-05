// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package phantom

import (
	"fmt"
	"reflect"
	"testing"
)

// Figure 3 in PHANTOM paper
func createGraph() *Graph {
	var g = NewGraph()
	g.AddNodeById("GENESIS")
	g.AddNodeById("B")
	g.AddNodeById("C")
	g.AddNodeById("D")
	g.AddNodeById("E")
	g.AddNodeById("F")
	g.AddNodeById("H")
	g.AddNodeById("I")
	g.AddNodeById("J")
	g.AddNodeById("K")
	g.AddNodeById("L")
	g.AddNodeById("M")

	g.AddEdgeById("B", "GENESIS")
	g.AddEdgeById("C", "GENESIS")
	g.AddEdgeById("D", "GENESIS")
	g.AddEdgeById("E", "GENESIS")

	g.AddEdgesById("F", []string{"B", "C"})
	g.AddEdgesById("H", []string{"C", "D", "E"})
	g.AddEdgeById("I", "E")

	g.AddEdgesById("J", []string{"F", "H"})
	g.AddEdgesById("K", []string{"B", "H", "I"})
	g.AddEdgesById("L", []string{"D", "I"})
	g.AddEdgesById("M", []string{"F", "K"})

	return g
}

func createFigure4DAG() *Graph {
	var g = NewGraph()
	g.AddNodeById("GENESIS")
	g.AddNodeById("B")
	g.AddNodeById("C")
	g.AddNodeById("D")
	g.AddNodeById("E")
	g.AddNodeById("F")
	g.AddNodeById("H")
	g.AddNodeById("I")
	g.AddNodeById("J")
	g.AddNodeById("K")
	g.AddNodeById("L")
	g.AddNodeById("M")
	g.AddNodeById("N")
	g.AddNodeById("O")
	g.AddNodeById("P")
	g.AddNodeById("Q")
	g.AddNodeById("R")
	g.AddNodeById("S")
	g.AddNodeById("T")
	g.AddNodeById("U")

	g.AddEdgeById("B", "GENESIS")
	g.AddEdgeById("C", "GENESIS")
	g.AddEdgeById("D", "GENESIS")
	g.AddEdgeById("E", "GENESIS")

	g.AddEdgesById("F", []string{"B", "C"})
	g.AddEdgesById("H", []string{"E"})
	g.AddEdgesById("I", []string{"C", "D"})

	g.AddEdgesById("J", []string{"D", "F"})

	g.AddEdgesById("K", []string{"E", "I", "J"})
	g.AddEdgesById("L", []string{"F"})
	g.AddEdgesById("M", []string{"K", "L"})
	g.AddEdgesById("N", []string{"D", "H"})

	g.AddEdgesById("O", []string{"K"})
	g.AddEdgesById("P", []string{"K"})
	g.AddEdgesById("Q", []string{"N"})
	g.AddEdgesById("R", []string{"N","O", "P"})

	g.AddEdgesById("S", []string{"Q"})
	g.AddEdgesById("T", []string{"S"})
	g.AddEdgesById("U", []string{"T"})

	return g
}

// nodeSame returns true if the nodes are the same
// NOTE(cedric): We only check the children, so that we don't loop checking between parents and children forever.
// We're assuming that we will be calling the *Same functions across an entire set of nodes/graph.
func nodeSame(a, b *Node) (string, bool) {
	aId := a.GetId()
	bId := b.GetId()

	if aId != bId {
		return fmt.Sprintf("node ids differ; got %s, want %s", bId, aId), false
	}

	aChildren := a.getChildren()
	bChildren := b.getChildren()

	reason, same := nodeSetSame(aChildren, bChildren)
	if !same {
		return fmt.Sprintf("node %s children differ between a and b: %s", aId, reason), false
	}

	return "", true
}

// nodeSetSame returns true if the nodeSets are the same
func nodeSetSame(a, b *nodeSet) (string, bool) {
	checked := make(map[string]struct{})

	// Check if corresponding nodes between nodeSets are the same
	for i, n := range a.elements() {
		id := n.GetId()

		if !b.contains(n) {
			return fmt.Sprintf("node %s in a but not b", id), false
		}

		var index int
		var found bool
		var bNodes = b.elements()
		for j, bn := range bNodes {
			if n == bn || id == bn.GetId() {
				index = j
				found = true
				break
			}
		}

		if !found {
			return fmt.Sprintf("node %s in a and b, but not found in order of b", id), false
		}

		if i != index {
			return fmt.Sprintf("node %s order not same between a and b; got %d, want %d", id, index, i), false
		}

		// Check that nodes are the same
		reason, same := nodeSame(n, bNodes[i])
		if !same {
			return fmt.Sprintf("node %s differs between a and b; %s", id, reason), false
		}

		checked[id] = keyExists
	}

	for _, n := range b.elements() {
		id := n.GetId()
		if _, ok := checked[id]; ok {
			continue
		}

		if !a.contains(n) {
			return fmt.Sprintf("node %s in b but not a", id), false
		}
	}
	return "", true
}

// orderedNodeSetSame returns true if the nodeSets are the same
func orderedNodeSetSame(a, b *orderedNodeSet) (string, bool) {
	checked := make(map[string]struct{})

	// Check if corresponding nodes between nodeSets are the same
	for i, n := range a.elements() {
		id := n.GetId()

		if !b.contains(n) {
			return fmt.Sprintf("node %s in a but not b", id), false
		}

		var index int
		var found bool
		var bNodes = b.elements()
		for j, bn := range bNodes {
			if n == bn || id == bn.GetId() {
				index = j
				found = true
				break
			}
		}

		if !found {
			return fmt.Sprintf("node %s in a and b, but not found in order of b", id), false
		}

		if i != index {
			return fmt.Sprintf("node %s order not same between a and b; got %d, want %d", id, index, i), false
		}

		// Check that nodes are the same
		reason, same := nodeSame(n, bNodes[i])
		if !same {
			return fmt.Sprintf("node %s differs between a and b; %s", id, reason), false
		}

		checked[id] = keyExists
	}

	for _, n := range b.elements() {
		id := n.GetId()
		if _, ok := checked[id]; ok {
			continue
		}

		if !a.contains(n) {
			return fmt.Sprintf("node %s in b but not a", id), false
		}
	}
	return "", true
}

// blueSetSame returns true if the BlueSetCaches are the same
func blueSetSame(a, b *BlueSetCache) (string, bool) {
	// Check if the caches have the same nodes
	aIds := make(map[string]*Node)
	bIds := make(map[string]*Node)
	for n := range a.cache {
		aIds[n.GetId()] = n
	}
	for n := range b.cache {
		bIds[n.GetId()] = n
	}

	// Check if nodes are the same
	for id, n := range aIds {
		bn, exists := bIds[id]
		if !exists {
			return fmt.Sprintf("node %s in a but not b", id), false
		}

		reason, same := nodeSame(n, bn)
		if !same {
			return fmt.Sprintf("node %s differs between a and b\n\t\t%s", id, reason), false
		}

		// Check if nodeSets are the same
		reason, same = nodeSetSame(a.cache[n], b.cache[bn])
		if !same {
			return fmt.Sprintf("node %s orderedNodeSet differs between a and b\n\t\t%s", id, reason), false
		}
	}

	for id := range bIds {
		_, exists := aIds[id]
		if !exists {
			return fmt.Sprintf("node %s b but not a", id), false
		}
	}

	return "", true
}


func TestCalculateBlueSet(t *testing.T) {
	var graph = createGraph()
	var genesis = graph.GetNodeById("GENESIS")
	var blueSet = calculateBlueSet(graph, genesis, 0, nil)

	var expected = []string{"C", "GENESIS", "H", "K", "M"}

	if !reflect.DeepEqual(expected, GetIds(blueSet.elements())) {
		t.Errorf("Incorrect blue set for k = 0. Expecting %v, got %v",
			expected, GetIds(blueSet.elements()))
	}

	blueSet = calculateBlueSet(graph, genesis, 3, nil)
	expected = []string{"B", "C", "D", "F", "GENESIS", "H", "J", "K", "M"}

	if !reflect.DeepEqual(expected, GetIds(blueSet.elements())) {
		t.Errorf("Incorrect blue set for k = 3. Expecting %v, got %v",
			expected, GetIds(blueSet.elements()))
	}
}

func TestOrderDAGColoring(t *testing.T) {
	var steps = []struct{
		node string
		parents []string
	}{
		{node: "GENESIS"},
		{node: "B", parents: []string{"GENESIS"}},
		{node: "C", parents: []string{"GENESIS"}},
		{node: "D", parents: []string{"GENESIS"}},
		{node: "E", parents: []string{"GENESIS"}},
		{node: "F", parents: []string{"B", "C"}},
		{node: "H", parents: []string{"C", "D", "E"}},
		{node: "I", parents: []string{"E"}},
		{node: "J", parents: []string{"F", "H"}},
		{node: "K", parents: []string{"B", "H", "I"}},
		{node: "L", parents: []string{"D", "I"}},
		{node: "M", parents: []string{"F", "K"}},
	}

	genStep := func(i int, g1, g2 *Graph) {
		step := steps[i]
		// Make sure that all graphs have a reference to the same node pointer, and not
		// duplicate Nodes with different pointers.
		n := newNode(step.node)

		g1.addNode(n)
		g2.addNode(n)

		if len(step.parents) == 0 {
			return
		}

		// We don't need to call AddEdgesById for g2 because addEdgeById modifies the Node type,
		// which was already done in the call to g1.AddEdgesById().
		// All we need to do, to keep the graphs consistent is to remove the same nodes from
		// tips that was done in the call to g1.AddEdgesById().
		before := g1.tips.elements()
		g1.AddEdgesById(step.node, step.parents)
		after := g1.tips.elements()

		bTips := newOrderedNodeSet()
		for _, tip := range before {
			bTips.add(tip)
		}

		aTips := newOrderedNodeSet()
		for _, tip := range after {
			aTips.add(tip)
		}

		removeTips := bTips.difference(aTips)
		for _, tip := range removeTips.elements() {
			g2.tips.remove(tip)
		}
	}

	cacheGraph := NewGraph()
	orderCache := NewOrderCache()
	cacheBlueSet := NewBlueSetCache()
	var cacheGenesis *Node

	noCacheGraph := NewGraph()
	noCacheBlueSet := NewBlueSetCache()
	var noCacheGenesis *Node

	for i, _ := range steps {
		genStep(i, cacheGraph, noCacheGraph)

		if i == 0 {
			cacheGenesis = cacheGraph.GetNodeById("GENESIS")
			noCacheGenesis = noCacheGraph.GetNodeById("GENESIS")
		}

		cacheTips, cacheOrder, err := OrderDAG(cacheGraph, cacheGenesis, 3, cacheBlueSet, int32(i), orderCache)
		if err != nil {
			t.Fatalf("Step %d OrderDAG with orderCache failed: %s", i, err)
		}
		noCacheTips, noCacheOrder, err := OrderDAG(noCacheGraph, noCacheGenesis, 3, noCacheBlueSet, int32(i), nil)
		if err != nil {
			t.Fatalf("Step %d OrderDAG no orderCache failed: %s", i, err)
		}

		if !reflect.DeepEqual(GetIds(cacheTips), GetIds(noCacheTips)) {
			t.Fatalf("Step %d OrderDAG tips not the same between cache and no-cache:\n\tcacheTips %s\tnoCacheTips %s",
				i, GetIds(cacheTips), GetIds(noCacheTips))
		}

		// Order should always be the same in this case, because we aren't generating enough generations of the
		// graph to trigger caching hits.
		if !reflect.DeepEqual(GetIds(cacheOrder), GetIds(noCacheOrder)) {
			t.Fatalf("Step %d OrderDAG order not the same between cache and no-cache:\n\tcacheOrder %s\tnoCacheOrder %s",
				i, GetIds(cacheOrder), GetIds(noCacheOrder))
		}

		reason, ok := blueSetSame(cacheBlueSet, noCacheBlueSet)
		if !ok {

			fmt.Println("cacheBlueSet")
			fmt.Print(cacheBlueSet.String())
			fmt.Println("noCacheBlueSet")
			fmt.Print(noCacheBlueSet.String())

			t.Fatalf("Step %d OrderDAG BlueSetCache not the same between cache and no-cache:\n\t%s", i, reason)
		}
	}
}

func TestFigure4BlueSet(t *testing.T) {
	var graph = createFigure4DAG()
	var genesis = graph.GetNodeById("GENESIS")

	var blueSetCache = NewBlueSetCache()
	var blueSet = calculateBlueSet(graph, genesis, 3, blueSetCache)
	var expected = []string{"B", "C", "D", "E", "F", "GENESIS", "I", "J", "K", "M", "O", "P", "R"}

	if !reflect.DeepEqual(expected, GetIds(blueSet.elements())) {
		t.Errorf("Incorrect blue set for figure 4,  k = 3. Expecting %v, got %v",
			expected, GetIds(blueSet.elements()))
	}

	var _, orderedNodes, err = OrderDAG(graph, genesis, 3, blueSetCache, int32(-1), nil)
	if err != nil {
		t.Fatalf("Failed to sort dag: %s", err)
	}

	expected = []string{"GENESIS", "B", "C", "D", "E", "F", "I", "J", "K", "L", "M",
		"O", "P", "H", "N", "Q", "R", "S", "T", "U"}

	if !reflect.DeepEqual(expected, GetIds(orderedNodes)) {
		t.Errorf("Incorrect ordering for figure 4,  k = 3. Expecting %v, got %v",
			expected, GetIds(orderedNodes))
	}
}

func TestBlueSetCache_Add(t *testing.T) {
	a := newNode("A")
	b := newNode("B")
	ns := newNodeSet()
	ns.add(b)

	blueSetCache := NewBlueSetCache()

	if len(blueSetCache.cache) != 0 {
		t.Errorf("wrong cache size; got %d, want %d", len(blueSetCache.cache), 0)
	}

	blueSetCache.Add(a, ns)

	if len(blueSetCache.cache) != 1 {
		t.Errorf("wrong cache size; got %d, want %d", len(blueSetCache.cache), 1)
	}

	cacheNs, ok := blueSetCache.cache[a]
	if !ok {
		t.Errorf("failed to create new cache entry for %s", a.GetId())
	}

	if !reflect.DeepEqual(cacheNs, ns) {
		t.Errorf("cache entry different from what was added")
	}

	// Test cache entry expiry  due to Add()
	for i := 0; i < maxBlueSetCacheSize; i++ {
		n := newNode(fmt.Sprintf("X%d", i))
		blueSetCache.Add(n, nil)
	}

	if len(blueSetCache.cache) != maxBlueSetCacheSize {
		t.Errorf("wrong cache size; got %d, want %d", len(blueSetCache.cache), maxBlueSetCacheSize)
	}
}

func TestBlueSetCache_Expire(t *testing.T) {
	a := newNode("A")
	b := newNode("B")
	ns := newNodeSet()
	ns.add(b)

	blueSetCache := NewBlueSetCache()
	blueSetCache.Add(a, ns)

	blueSetCache.Expire(0)

	if len(blueSetCache.cache) != 1 {
		t.Errorf("wrong cache size; got %d, want %d", len(blueSetCache.cache), 1)
	}

	blueSetCache.Expire(-1)

	if len(blueSetCache.cache) != 1 {
		t.Errorf("wrong cache size; got %d, want %d", len(blueSetCache.cache), 1)
	}

	blueSetCache.Expire(1)

	if len(blueSetCache.cache) != 0 {
		t.Errorf("wrong cache size; got %d, want %d", len(blueSetCache.cache), 0)
	}

	blueSetCache.Add(a, ns)

	blueSetCache.Expire(len(blueSetCache.cache) * 2)

	if len(blueSetCache.cache) != 0 {
		t.Errorf("wrong cache size; got %d, want %d", len(blueSetCache.cache), 0)
	}
}

func TestBlueSetCache_GetBlueNodes(t *testing.T) {
	a := newNode("A")
	b := newNode("B")
	c := newNode("C")
	ns := newNodeSet()
	ns.add(b)
	ns.add(c)

	blueSetCache := NewBlueSetCache()
	blueSetCache.Add(a, ns)

	nodes := blueSetCache.GetBlueNodes(a)

	if len(nodes) != ns.size() {
		t.Errorf("wrong blue nodes size; got %d, want %d", len(nodes), ns.size())
	}

	if !reflect.DeepEqual(GetIds(nodes), GetIds(ns.elements())) {
		t.Errorf("nodes for cache hit don't match; got %v, want %v", GetIds(nodes), GetIds(ns.elements()))
	}

	nodes = blueSetCache.GetBlueNodes(c)
	if nodes != nil {
		t.Errorf("nodes should be nil for non-existent cache entry %s", c.GetId())
	}

	nodes = blueSetCache.GetBlueNodes(nil)
	if nodes != nil {
		t.Errorf("nodes should be nil for non-existent cache entry %v", nil)
	}
}

func TestBlueSetCache_GetBlueSet(t *testing.T) {
	a := newNode("A")
	b := newNode("B")
	c := newNode("C")
	ns := newNodeSet()
	ns.add(b)
	ns.add(c)

	blueSetCache := NewBlueSetCache()
	blueSetCache.Add(a, ns)

	set := blueSetCache.GetBlueSet(a)

	if set.size() != ns.size() {
		t.Errorf("wrong blueSet size; got %d, want %d", set.size(), ns.size())
	}

	if !reflect.DeepEqual(GetIds(set.elements()), GetIds(ns.elements())) {
		t.Errorf("cache-hit doesn't match blueSet; got %v, want %v", GetIds(set.elements()), GetIds(ns.elements()))
	}

	set = blueSetCache.GetBlueSet(c)
	if set != nil {
		t.Errorf("blueSet should be nil for non-existent cache entry %s", c.GetId())
	}

	set = blueSetCache.GetBlueSet(nil)
	if set != nil {
		t.Errorf("blueSet should be nil for non-existent cache entry %v", nil)
	}
}

func TestBlueSetCache_InCache(t *testing.T) {
	a := newNode("A")
	b := newNode("B")
	c := newNode("C")
	ns := newNodeSet()
	ns.add(b)
	ns.add(c)

	blueSetCache := NewBlueSetCache()
	blueSetCache.Add(a, ns)

	ok := blueSetCache.InCache(a)
	if !ok {
		t.Errorf("cache miss for %s", a.GetId())
	}

	ok = blueSetCache.InCache(c)
	if ok {
		t.Errorf("cache hit for %s", c.GetId())
	}

	ok = blueSetCache.InCache(nil)
	if ok {
		t.Errorf("cache hit for %v", nil)
	}
}