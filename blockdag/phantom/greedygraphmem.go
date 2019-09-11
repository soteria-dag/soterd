// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package phantom

import (
	"fmt"
	"sort"
	"strings"
)

// kChain represents a chain of nodes that pass coloringRule2 for a given k value
type kChain struct {
	chain *OrderedStringSet
	minHeight int
}

// GreedyGraph represents a graph of nodes, using Greedy PHANTOM colouring
type GreedyGraphMem struct {
	// coloringK value is used by a large number of methods
	k int

	children map[string]*OrderedStringSet
	parents map[string]*OrderedStringSet
	height map[string]int

	// Coloring tip of main coloring chain
	coloringTip string
	// The main coloring chain
	coloringChain *OrderedStringSet
	// The main k-chain
	mainKChain *kChain

	// Used for coloring and ordering of graph
	bluePastOrder *ChainMap
	redPastOrder *ChainMap

	// Antipast is diffPast between virtual node and its coloring parent,
	// which is the coloring tip of the graph.
	blueAntiPastOrder *OrderMap
	redAntiPastOrder *OrderMap
	uncoloredUnorderedAntiPast *OrderMap

	// Aggregations of other data-structures
	// pastOrder = ChainMap(bluePastOrder, redPastOrder)
	pastOrder *ChainMap

	// antiPastOrder = ChainMap(blueAntiPastOrder, redAntiPastOrder)
	antiPastOrder *ChainMap

	// antiPast = ChainMap(antiPastOrder, uncoloredUnorderedAntiPast)
	antiPast *ChainMap

	// coloringOrder is virtual node's coloring of the entire graph.
	// coloringOrder = ChainMap(bluePastOrder, blueAntiPastOrder)
	coloringOrder *ChainMap

	// order is the virtual node's ordering of the entire graph.
	// order = ChainMap(pastOrder, antiPastOrder)
	order *ChainMap

	// TODO(cedric): Prune node-specific map entries after a certain height away from tips

	// blueCount tracks the number of blue nodes in a node's past
	blueCount map[string]int

	// coloringParents tracks coloring parent of nodes
	coloringParents map[string]string

	blueDiffPastOrder map[string]OrderMap
	redDiffPastOrder map[string]OrderMap

	selfOrder map[string]int
}

// Add adds a node to the graph with the given parents, then updates coloring and graph order
func (g *GreedyGraphMem) Add(id string, parents []string) (bool, error) {
	ok, err := g.AddNode(id)
	if err != nil {
		return ok, err
	}

	if len(parents) > 0 {
		ok, err := g.AddMultiEdge(id, parents)
		if err != nil {
			return ok, err
		}
	}

	// Calculate and cache the height of the node
	_, err = g.Height(id)
	if err != nil {
		return ok, err
	}

	// Update the coloring and order of the graph
	g.updateColoringIncrementally(id, parents)
	g.updateTopologicalOrderIncrementally(id)

	return ok, nil
}

// AddNode adds a node to the graph
// Returns true if the node was added.
func (g *GreedyGraphMem) AddNode(id string) (bool, error) {
	_, ok := g.parents[id]
	if ok {
		return false, nil
	}

	g.parents[id] = NewOrderedStringSet()
	g.children[id] = NewOrderedStringSet()

	return true, nil
}

// AddEdge adds an edge from node a to b. (b is parent of a)
// Returns true if the edge was added.
func (g *GreedyGraphMem) AddEdge(a, b string) (bool, error) {
	// Add node, if it hasn't been added already
	_, err := g.AddNode(a)
	if err != nil {
		return false, err
	}

	// Add edge, if it hasn't been added already
	_, err = g.AddNode(b)
	if err != nil {
		return false, err
	}

	parentsOfA, ok := g.parents[a]
	if !ok {
		return false, fmt.Errorf("failed to add edge from %s to %s; no node entry for %s", a, b, a)
	}

	childrenOfB, ok := g.children[b]
	if !ok {
		return false, fmt.Errorf("failed to add edge from %s to %s; no node entry for %s", a, b, b)
	}

	if parentsOfA.Contains(b) {
		// Edge already exists
		return false, nil
	}

	parentsOfA.Add(b)
	childrenOfB.Add(a)

	return true, nil
}

// AddMultiEdgeById adds an edge from node with id a to nodes in parents.
// Returns true if all edges were added.
func (g *GreedyGraphMem) AddMultiEdge(a string, parents []string) (bool, error) {
	added := 0
	for _, p := range parents {
		ok, err := g.AddEdge(a, p)
		if err != nil {
			return false, err
		}

		if ok {
			added += 1
		}
	}

	return added == len(parents), nil
}

// RemoveEdge removes an edge from node a to b.
// b is not parent of a
// a is not child of b
// Returns true if the edge was removed.
func (g *GreedyGraphMem) RemoveEdge(a, b string) bool {
	var inParent, inChild bool

	parents, ok := g.parents[a]
	if ok {
		inParent = parents.Contains(b)
		parents.Remove(b)
	}

	children, ok := g.children[b]
	if ok {
		inChild = children.Contains(a)
		children.Remove(a)
	}

	return inParent && inChild
}

// RemoveTip removes a tip from the graph.
func (g *GreedyGraphMem) RemoveTip(id string) error {
	children, ok := g.children[id]
	if !ok {
		// Attempting to remove a tip that doesn't exist
		return nil
	}

	if children.Size() > 0 {
		// This is not a tip, so we won't remove it
		return nil
	}

	// Removing the node triggers:
	// recalculation of coloringTip if this node was the coloringTip,
	// recalculation of main coloring chain if it was the tip of the main coloring chain
	//
	// It also removes:
	// any edges involving it,
	// its diffPastOrder,
	// its antiPastOrder,
	// its bluePast, redPast,
	// its blueCount,
	// its coloringParents info,
	// the node from the graph order,
	// the node's self-order,
	// the node from uncoloredUnorderedAntiPast.

	if g.coloringTip != "" && id == g.coloringTip {
		// Update coloring tip and main coloring chain
		tips, err := g.Tips()
		if err != nil {
			return fmt.Errorf("failed to remove node %s: %s", id, err)
		}

		otherTips := make([]string, 0)
		for _, t := range tips {
			if t != id {
				otherTips = append(otherTips, t)
			}
		}

		if len(otherTips) > 0 {
			// Set the coloring tip and main chain to one of the other tips.
			bluest := g.getBluest(otherTips...)
			g.coloringTip = bluest

			g.mainKChain = g.getKChain(bluest)
		} else {
			// This node is the only tip, so set the coloring tip and main chain
			// to one of its parents.
			parents, ok := g.parents[id]
			if !ok {
				return fmt.Errorf("failed to remove node %s; missing entry for node in g.parents", id)
			}

			if parents.Size() > 0 {
				bluest := g.getBluest(parents.Elements()...)
				g.coloringTip = bluest

				g.mainKChain = g.getKChain(bluest)
			}
		}
	}

	// Remove references to node everywhere
	delete(g.children, id)
	delete(g.parents, id)

	for _, children := range g.children {
		if children.Contains(id) {
			children.Remove(id)
		}
	}

	for _, parents := range g.parents {
		if parents.Contains(id) {
			parents.Remove(id)
		}
	}

	delete(g.height, id)
	g.uncoloredUnorderedAntiPast.Remove(id)
	delete(g.selfOrder, id)
	delete(g.coloringParents, id)
	delete(g.blueCount, id)
	g.redPastOrder.Remove(id)
	g.bluePastOrder.Remove(id)
	g.redAntiPastOrder.Remove(id)
	g.blueAntiPastOrder.Remove(id)
	delete(g.redDiffPastOrder, id)
	delete(g.blueDiffPastOrder, id)

	return nil
}

// NodeExists returns true if the node exists in the graph
func (g *GreedyGraphMem) NodeExists(id string) (bool, error) {
	_, ok := g.parents[id]
	return ok, nil
}

// Nodes returns all nodes in the graph
func (g *GreedyGraphMem) Nodes() []string {
	nodes := make([]string, len(g.parents))
	index := 0
	for n := range g.parents {
		nodes[index] = n
		index += 1
	}

	return nodes
}

// Tips returns the tip nodes of the graph
func (g *GreedyGraphMem) Tips() ([]string, error) {
	tips := make([]string, 0)
	for node, children := range g.children {
		if children.Size() == 0 {
			tips = append(tips, node)
		}
	}
	sort.Strings(tips)

	return tips, nil
}

// TipDiff returns the unioned antiPast of the given subTips; nodes from the future of
// those subTips to the tips of the graph, including non-adjacent nodes.
func (g *GreedyGraphMem) TipDiff(subTips []string) ([]string, error) {
	subTipPast := make(map[string]struct{})

	for _, tip := range subTips {
		subTipPast[tip] = keyExists

		past, err := g.Past(tip)
		if err != nil {
			return nil, err
		}

		for _, id := range past {
			_, ok := subTipPast[id]
			if !ok {
				subTipPast[id] = keyExists
			}
		}
	}

	// Include nodes not in any of the subTips past
	diff := make([]string, 0)
	for _, n := range g.Nodes() {
		_, ok := subTipPast[n]
		if !ok {
			diff = append(diff, n)
		}
	}

	return diff, nil
}

// Parents returns the parents of the node
func (g *GreedyGraphMem) Parents(id string) ([]string, error) {
	parents, ok := g.parents[id]
	if !ok {
		return nil, fmt.Errorf("missing node %s", id)
	}

	return parents.Elements(), nil
}

// Past returns the past of the node
func (g *GreedyGraphMem) Past(id string) ([]string, error) {
	parents, ok := g.parents[id]
	if !ok {
		return nil, fmt.Errorf("missing node %s", id)
	}

	if parents.Size() == 0 {
		return []string{}, nil
	}

	var past = parents.Clone()
	todo := newStringList()
	for _, p := range parents.Elements() {
		todo.push(p)
	}

	for todo.size() > 0 {
		node, ok := todo.shift()
		if !ok {
			return nil, fmt.Errorf("todo size > 0 but nothing shifted")
		}

		past.Add(node)

		parents, ok := g.parents[node]
		if !ok {
			continue
		}

		for _, p := range parents.Elements() {
			todo.push(p)
		}
	}

	return past.Elements(), nil
}

// Height returns the min, max height of the node
func (g *GreedyGraphMem) Height(id string) (int, error) {
	height, ok := g.height[id]
	if ok {
		return height, nil
	}

	parents, ok := g.parents[id]
	if !ok {
		return -1, fmt.Errorf("missing node %s", id)
	}

	if parents.Size() == 0 {
		return 0, nil
	}

	var maxHeight int
	for _, p := range parents.Elements() {
		pHeight, err := g.Height(p)
		if err != nil {
			return -1, fmt.Errorf("failed to get height of %s when evaluating parent %s: %s", id, p, err)
		}

		if pHeight > maxHeight {
			maxHeight = pHeight
		}
	}

	var nodeHeight = maxHeight + 1
	g.height[id] = nodeHeight

	return nodeHeight, nil
}

// Order returns the topological order of the graph
func (g *GreedyGraphMem) Order() ([]string, error) {
	var ordered = NewOrderedStringSet()
	var coloring = g.getColoring()

	// TODO(cedric): Define this as a loop instead of recursion
	// Define the function variable first, so that the function can reference itself when contents are defined.
	var getTopologicalOrder func([]string, string) ([]string, error)
	getTopologicalOrder = func(leaves []string, coloringParent string) ([]string, error) {
		var leafSet = NewOrderedStringSet(leaves...)

		leafSet = leafSet.Difference(ordered)
		if leafSet.Size() == 0 {
			// No remaining graph to order
			return nil, nil
		}

		blueLeafSet := leafSet.Intersection(coloring)
		var blueLeaves = blueLeafSet.Elements()
		sort.Strings(blueLeaves)
		redLeafSet := leafSet.Difference(blueLeafSet)
		var redLeaves = redLeafSet.Elements()
		sort.Strings(redLeaves)

		toSort := make([]string, 0)
		if coloringParent != "" {
			toSort = append(toSort, coloringParent)
		}
		toSort = append(toSort, blueLeaves...)
		toSort = append(toSort, redLeaves...)

		var topologicalOrder = NewOrderedStringSet()
		for _, leaf := range toSort {
			ordered.Add(leaf)

			parents, err := g.Parents(leaf)
			if err != nil {
				return nil, err
			}

			var leafOrder []string
			leafColoringParent, ok := g.coloringParents[leaf]
			if ok {
				leafOrder, err = getTopologicalOrder(parents, leafColoringParent)
			} else {
				leafOrder, err = getTopologicalOrder(parents, "")
			}

			leafOrder = append(leafOrder, leaf)
			for _, n := range leafOrder {
				topologicalOrder.Add(n)
			}
		}

		return topologicalOrder.Elements(), nil
	}

	tips, err := g.Tips()
	if err != nil {
		return nil, err
	}
	var order []string

	if g.coloringTip != "" {
		order, err = getTopologicalOrder(tips, g.coloringTip)
	} else {
		order, err = getTopologicalOrder(tips, "")
	}

	if err != nil {
		return nil, err
	}

	return order, nil
}

// ColoringOrder returns the coloring order of the graph
func (g *GreedyGraphMem) ColoringOrder() (*OrderedStringSet, error) {
	return g.coloringOrder.Keys(), nil
}

// ToString returns a string representation of the graph
func (g *GreedyGraphMem) ToString() (string, error) {
	var sb strings.Builder
	var expanded = make(map[string]bool)

	todo := newStringList()
	tips, err := g.Tips()
	if err != nil {
		return "", err
	}

	for _, tip := range tips {
		todo.push(tip)
	}

	for todo.size() > 0 {
		node, ok := todo.shift()
		if !ok {
			return "", fmt.Errorf("No element removed from front of list")
		}

		if !expanded[node] {
			parents, err := g.Parents(node)
			if err != nil {
				return "", err
			}

			size := len(parents)
			if size > 0 {
				// sort parents so order is always the same
				sort.Strings(parents)
				for _, p := range parents {
					todo.push(p)
					sb.WriteString(node + "->" + p + "\n")
				}
			} else {
				sb.WriteString(node + "\n")
			}
		}

		expanded[node] = true
	}

	return sb.String(), nil
}

// clearAntiPastOrder clears the data-structures that make up the antiPastOrder
func (g *GreedyGraphMem) clearAntiPastOrder() {
	// Replace the OrderMaps in antiPastOrder with new ones
	var blueAntiPastOrder = make(OrderMap)
	g.blueAntiPastOrder = &blueAntiPastOrder

	var redAntiPastOrder = make(OrderMap)
	g.redAntiPastOrder = &redAntiPastOrder

	for e := g.antiPastOrder.members.Front(); e != nil; e = e.Next() {
		_ = g.antiPastOrder.members.Remove(e)
	}

	g.antiPastOrder.members.PushBack(*g.blueAntiPastOrder)
	g.antiPastOrder.members.PushBack(*g.redAntiPastOrder)

	// Update coloring order
	e := g.coloringOrder.members.Back()
	_ = g.coloringOrder.members.Remove(e)
	g.coloringOrder.members.PushBack(*g.blueAntiPastOrder)
}

func (g *GreedyGraphMem) isBlue(id string) bool {
	g.updateAntiPastColoring()
	return g.coloringOrder.Contains(id)
}

func (g *GreedyGraphMem) getColoring() *OrderedStringSet {
	g.updateAntiPastColoring()
	return g.coloringOrder.Keys()
}

// sortByBluest returns a copy of the ids, from least-blue to most.
// The sorting function will sort the globalIds from least-blue to most.
// The tie-breaker when two globalIds are equally as blue, is that the node with the lower globalId will
// be considered more blue.
func (g *GreedyGraphMem) sortByBluest(ids ...string) []string {
	if len(ids) < 2 {
		// Not enough elements to sort, order remains the same
		return ids[:]
	}

	var sorted = make([]string, len(ids))
	copy(sorted, ids)

	less := func(i, j int) bool {
		var iNode = sorted[i]
		iCount, _ := g.blueCount[iNode]

		var jNode = sorted[j]
		jCount, _ := g.blueCount[jNode]

		if iCount < jCount {
			return true
		}

		if iCount == jCount && iNode > jNode {
			return true
		}

		return false
	}

	sort.Slice(sorted, less)

	return sorted
}

func (g *GreedyGraphMem) getBluest(ids ...string) string {
	sorted := g.sortByBluest(ids...)
	return sorted[len(sorted) - 1]
}

func (g *GreedyGraphMem) coloringRule2(chain *kChain, id string) bool {
	var coloringParent = id
	var ok = true
	for ok {
		height, _ := g.Height(coloringParent)
		if height < chain.minHeight {
			return false
		}

		if chain.chain.Contains(coloringParent) {
			return true
		}

		coloringParent, ok = g.coloringParents[coloringParent]
	}

	return false
}

func (g *GreedyGraphMem) colorNode(blueOrder, redOrder *OrderMap, chain *kChain, id string) {
	if g.coloringRule2(chain, id) {
		blueOrder.Add(id)
		if redOrder.Contains(id) {
			redOrder.Remove(id)
		}
	} else {
		redOrder.Add(id)
		if blueOrder.Contains(id) {
			blueOrder.Remove(id)
		}
	}
}

func (g *GreedyGraphMem) getKChain(id string) *kChain {
	var chain = NewOrderedStringSet()
	var minHeight = -1
	var blueCount = 0

	var coloringParent = id
	var ok = true
	for ok {
		if blueCount > g.k {
			break
		}

		chain.Add(coloringParent)
		height, _ := g.Height(coloringParent)
		minHeight = height
		blueDiffPastOrder, hasBlueDiffPastOrder := g.blueDiffPastOrder[coloringParent]
		if hasBlueDiffPastOrder {
			blueCount += len(blueDiffPastOrder)
		}

		coloringParent, ok = g.coloringParents[coloringParent]
	}

	return &kChain{
		chain: chain,
		minHeight: minHeight,
	}
}

func (g *GreedyGraphMem) getAntiPast(id string) *OrderedStringSet {
	if id == "" {
		return g.order.Keys()
	}

	if g.coloringTip != "" && id == g.coloringTip {
		return g.antiPast.Keys()
	}

	var positiveSets = make([]*OrderedStringSet, 0)
	var negativeSets = make([]*OrderedStringSet, 0)
	var intersection string

	// Walk from local tip to intersection with main coloring chain
	var coloringParent = id
	var ok = true
	for ok {
		var target *[]*OrderedStringSet
		if g.coloringChain.Contains(coloringParent) {
			intersection = coloringParent
			target = &positiveSets
		} else {
			target = &negativeSets
		}

		blue, hasBlue := g.blueDiffPastOrder[coloringParent]
		if hasBlue {
			order := NewOrderedStringSet(blue.Order()...)
			*target = append(*target, order)
		}

		red, hasRed := g.redDiffPastOrder[coloringParent]
		if hasRed {
			order := NewOrderedStringSet(red.Order()...)
			*target = append(*target, order)
		}


		if g.coloringChain.Contains(coloringParent) {
			break
		}

		coloringParent, ok = g.coloringParents[coloringParent]
	}

	// Walk from coloringTip to intersection
	// If the coloring tip isn't set, there is no work to do in this section
	coloringParent = g.coloringTip
	ok = g.coloringTip != ""
	for ok {
		var target *[]*OrderedStringSet
		if coloringParent == intersection {
			target = &negativeSets
		} else {
			target = &positiveSets
		}

		blue, hasBlue := g.blueDiffPastOrder[coloringParent]
		if hasBlue {
			order := NewOrderedStringSet(blue.Order()...)
			*target = append(*target, order)
		}

		red, hasRed := g.redDiffPastOrder[coloringParent]
		if hasRed {
			order := NewOrderedStringSet(red.Order()...)
			*target = append(*target, order)
		}

		if coloringParent == intersection {
			break
		}

		coloringParent, ok = g.coloringParents[coloringParent]
	}

	// Flatten the antipast and return it.
	// 1. antipast union positive sets
	// 2. antipast difference negative sets
	antiPast := g.antiPast.Keys()
	for _, set := range positiveSets {
		antiPast = antiPast.Union(set)
	}
	for _, set := range negativeSets {
		antiPast = antiPast.Difference(set)
	}

	return antiPast
}

func (g *GreedyGraphMem) updateDiffColoringOfNode(id string, parents []string) {
	var blueDiffPastOrder = make(OrderMap)
	var redDiffPastOrder = make(OrderMap)
	var chain = g.getKChain(id)
	var parentAntiPast *OrderedStringSet
	parent, ok := g.coloringParents[id]
	if ok {
		parentAntiPast = g.getAntiPast(parent)
	} else {
		parentAntiPast = g.getAntiPast("")
	}

	// Iterate over diff past and color all nodes according to node's coloring chain.
	// Because a node considers itself part of its antipast, it won't include itself in its coloring.
	var todo = newStringList()
	sort.Strings(parents)
	for _, n := range parents {
		todo.push(n)
	}

	for todo.size() > 0 {
		node, ok := todo.shift()
		if !ok {
			break
		}

		if blueDiffPastOrder.Contains(node) {
			continue
		}

		if redDiffPastOrder.Contains(node) {
			continue
		}

		if !parentAntiPast.Contains(node) {
			continue
		}

		parents, _ := g.Parents(node)
		sort.Strings(parents)
		for _, n := range parents {
			todo.unshift(n)
		}

		g.colorNode(&blueDiffPastOrder, &redDiffPastOrder, chain, node)
	}

	// Update coloring node with details
	g.blueDiffPastOrder[id] = blueDiffPastOrder
	g.redDiffPastOrder[id] = redDiffPastOrder

	blueCount, hasCount := g.blueCount[id]
	if hasCount {
		g.blueCount[id] = blueCount + len(blueDiffPastOrder)
	} else {
		g.blueCount[id] = len(blueDiffPastOrder)
	}
}

func (g *GreedyGraphMem) isABluerThanB(a, b string) bool {
	aCount, ok := g.blueCount[a]
	if !ok {
		return false
	}

	bCount, ok := g.blueCount[b]
	if !ok {
		return true
	}

	if aCount > bCount {
		return true
	}

	if aCount == bCount && a < b {
		return true
	}

	return false
}

func (g *GreedyGraphMem) isMaxColoringTip(id string) bool {
	if g.coloringTip == "" {
		return true
	}

	return g.isABluerThanB(id, g.coloringTip)
}

func (g *GreedyGraphMem) updatePastColoringAccordingTo(id string) {
	// uncoloredUnorderedAntiPast ∪ blueAntiPastOrder
	g.uncoloredUnorderedAntiPast.Add(g.blueAntiPastOrder.Order()...)

	// uncoloredUnorderedAntiPast ∪ redAntiPastOrder
	g.uncoloredUnorderedAntiPast.Add(g.redAntiPastOrder.Order()...)

	// TODO(cedric): This looks like a bug in Aviv's greedy phantom implementation.
	// Right after clearing antiPastOrder, we check if coloringTip is in blueAntiPastOrder.
	g.clearAntiPastOrder()

	// Add the tips to the antipast
	g.uncoloredUnorderedAntiPast.Add(id)

	var coloringTip = g.coloringTip
	var hasColoringTip = g.coloringTip != ""
	if hasColoringTip {
		if g.blueAntiPastOrder.Contains(coloringTip) {
			g.uncoloredUnorderedAntiPast.Add(coloringTip)
		}
	}

	var diffPastOrderings = make([]string, 0)

	// Walk from local tip to intersection with main coloring chain
	var intersection string
	var coloringParent = id
	var ok = true
	for ok {
		if g.coloringChain.Contains(coloringParent) {
			// Intersection is common for both old coloring chain and new one,
			// so there's no reason to make any modifications.
			intersection = coloringParent
			break
		}

		g.coloringChain.Add(coloringParent)

		diffPastOrderings = append(diffPastOrderings, coloringParent)

		coloringParent, ok = g.coloringParents[coloringParent]
	}

	// Walk from coloringTip to intersection
	coloringParent = coloringTip
	ok = hasColoringTip
	for ok {
		if coloringParent == intersection {
			// Intersection is common for both old coloring chain and new one,
			// so there's no reason to make any modifications.
			break
		}

		g.coloringChain.Remove(coloringParent)

		g.uncoloredUnorderedAntiPast.Add(g.bluePastOrder.Pop()...)
		g.uncoloredUnorderedAntiPast.Add(g.redPastOrder.Pop()...)

		coloringParent, ok = g.coloringParents[coloringParent]
	}

	// Flatten uncoloredUnorderedAntiPast
	// 1. union bluePastOrder (done in walk from coloringTip to intersection)
	// 2. union redPastOrder (done in walk from coloringTip to intersection)
	// 3. For each pair of diffPastOrderings
	// 	a) difference blueDiffPastOrder
	//  b) difference redDiffPastOrder
	for _, node := range diffPastOrderings {
		blueDiffPastOrder, ok := g.blueDiffPastOrder[node]
		if ok {
			g.bluePastOrder.members.PushBack(blueDiffPastOrder)
		}

		redDiffPastOrder, ok := g.redDiffPastOrder[node]
		if ok {
			g.redPastOrder.members.PushBack(redDiffPastOrder)
		}

		// uncoloredUnorderedAntiPast \ blueDiffPastOrder
		for n := range blueDiffPastOrder {
			g.uncoloredUnorderedAntiPast.Remove(n)
		}

		// uncoloredUnorderedAntiPast \ redDiffPastOrder
		for n := range redDiffPastOrder {
			g.uncoloredUnorderedAntiPast.Remove(n)
		}
	}

	// Set the new coloring tip
	g.coloringTip = id
}

func (g *GreedyGraphMem) updateAntiPastColoring() {
	for _, n := range g.uncoloredUnorderedAntiPast.Order() {
		g.colorNode(g.blueAntiPastOrder, g.redAntiPastOrder, g.mainKChain, n)
	}

	g.uncoloredUnorderedAntiPast.Clear()
}

func (g *GreedyGraphMem) updateMaxColoring(id string) {
	if !g.isMaxColoringTip(id) {
		return
	}

	g.updatePastColoringAccordingTo(id)
	g.mainKChain = g.getKChain(id)
}

func (g *GreedyGraphMem) updateColoringIncrementally(id string, parents []string) {

	if len(parents) > 0 {
		bluest := g.getBluest(parents...)
		g.coloringParents[id] = bluest
	}

	g.blueDiffPastOrder[id] = make(OrderMap)
	g.redDiffPastOrder[id] = make(OrderMap)

	coloringParent, hasColoringParent := g.coloringParents[id]
	var blueCount = 0
	if hasColoringParent {
		currentBlueCount, hasBlueCount := g.blueCount[id]
		parentBlueCount, parentHasBlueCount := g.blueCount[coloringParent]

		if hasBlueCount && parentHasBlueCount {
			blueCount = currentBlueCount + parentBlueCount
		} else if parentHasBlueCount {
			blueCount = parentBlueCount
		}
	}
	g.blueCount[id] = blueCount

	// Update virtual node's view of current node
	g.uncoloredUnorderedAntiPast.Add(id)

	g.updateDiffColoringOfNode(id, parents)
	g.updateMaxColoring(id)
}

func (g *GreedyGraphMem) calculateTopologicalOrder(coloringParent string, leaves []string, coloring, unordered *OrderedStringSet) *OrderedStringSet {
	var sortNodes = func(last string, later, toSort, unsorted *OrderedStringSet) []string {
		lastSet := NewOrderedStringSet()
		if last != "" {
			lastSet.Add(last)
		}

		var remaining = toSort.Difference(lastSet).Intersection(unsorted)

		// Sort the blue nodes from highest-globalId to lowest
		blueSet := remaining.Intersection(later)
		var blueList = blueSet.Elements()
		sort.Sort(sort.Reverse(sort.StringSlice(blueList)))

		// Sort the red nodes from highest-globalId to lowest
		redSet := remaining.Difference(blueSet)
		var redList = redSet.Elements()
		sort.Sort(sort.Reverse(sort.StringSlice(redList)))

		// last is coloring parent
		if last != "" {
			blueList = append(blueList, last)
		}

		order := make([]string, 0)
		order = append(order, redList...)
		order = append(order, blueList...)
		return order
	}

	var todo = newStringList()
	leavesSet := NewOrderedStringSet(leaves...)
	sortedNodes := sortNodes(coloringParent, coloring, leavesSet, unordered)
	for _, n := range sortedNodes {
		todo.push(n)
	}

	var ordered = NewOrderedStringSet()
	for todo.size() > 0 {
		node, ok := todo.pop()
		if !ok {
			log.Errorf("todo size > 0 but no element popped")
			break
		}

		if ordered.Contains(node) {
			continue
		}

		parents, _ := g.Parents(node)
		parentsSet := NewOrderedStringSet(parents...)
		parentsSet = parentsSet.Intersection(unordered)
		if parentsSet.Subset(ordered) {
			ordered.Add(node)
		} else {
			todo.push(node)

			nodeColoringParent, ok := g.coloringParents[node]
			if ok {
				sortedParentNodes := sortNodes(nodeColoringParent, coloring, parentsSet, unordered)
				for _, n := range sortedParentNodes {
					todo.push(n)
				}
			}
		}
	}

	return ordered
}

func (g *GreedyGraphMem) updateTopologicalOrderInAntiPast(leaves []string, coloringParent string) {
	var startingIndex = 0
	if coloringParent != "" {
		oIndex, ok := g.selfOrder[coloringParent]
		if ok {
			startingIndex = oIndex
		}
	}

	var blue = NewOrderedStringSet(g.blueAntiPastOrder.Order()...)
	var red = NewOrderedStringSet(g.redAntiPastOrder.Order()...)

	unordered := blue.Union(red)
	order := g.calculateTopologicalOrder(coloringParent, leaves, blue, unordered)
	for n, i := range order.members {
		newLocalId := i + startingIndex

		if g.blueAntiPastOrder.Contains(n) {
			g.blueAntiPastOrder.Set(n, newLocalId)
		} else {
			g.redAntiPastOrder.Set(n, newLocalId)
		}
	}
}

func (g *GreedyGraphMem) updateTopologicalOrderInDiffPast(id string) {
	var blue, red *OrderedStringSet

	blueDiffPastOrder, hasBlue := g.blueDiffPastOrder[id]
	if hasBlue {
		blue = NewOrderedStringSet(blueDiffPastOrder.Order()...)
	} else {
		blue = NewOrderedStringSet()
	}

	redDiffPastOrder, hasRed := g.redDiffPastOrder[id]
	if hasRed {
		red = NewOrderedStringSet(redDiffPastOrder.Order()...)
	} else {
		red = NewOrderedStringSet()
	}

	parents, _ := g.Parents(id)
	coloringParent, hasColoringParent := g.coloringParents[id]

	var startingIndex = 0
	if hasColoringParent {
		oIndex, ok := g.selfOrder[coloringParent]
		if ok {
			startingIndex = oIndex
		}
	}

	unordered := blue.Union(red)
	var order *OrderedStringSet
	if hasColoringParent {
		order = g.calculateTopologicalOrder(coloringParent, parents, blue, unordered)
	} else {
		order = g.calculateTopologicalOrder("", parents, blue, unordered)
	}
	for i, n := range order.Elements() {
		newLocalId := i + startingIndex
		if hasBlue && blueDiffPastOrder.Contains(n) {
			blueDiffPastOrder.Set(n, newLocalId)
		} else {
			redDiffPastOrder.Set(n, newLocalId)
		}
	}
}

func (g *GreedyGraphMem) updateSelfOrderIndex(id string) {
	var oIndex = 0

	var blueCount, redCount int
	blueDiffPastOrder, ok := g.blueDiffPastOrder[id]
	if ok {
		blueCount = len(blueDiffPastOrder)
	}
	oIndex += blueCount

	redDiffPastOrder, ok := g.redDiffPastOrder[id]
	if ok {
		redCount = len(redDiffPastOrder)
	}
	oIndex += redCount

	g.selfOrder[id] = oIndex

	coloringParent, hasParent := g.coloringParents[id]
	if hasParent {
		parentOrderIndex, hasOrder := g.selfOrder[coloringParent]
		if hasOrder {
			g.selfOrder[id] = oIndex + parentOrderIndex
		}
	}
}

func (g *GreedyGraphMem) updateTopologicalOrderIncrementally(id string) {
	g.updateTopologicalOrderInDiffPast(id)
	g.updateSelfOrderIndex(id)
}

// NewGreedyGraphMem returns a GreedyGraphMem
func NewGreedyGraphMem(k int) (*GreedyGraphMem, error) {
	var bluePastOrder = NewChainMap()
	var redPastOrder = NewChainMap()

	var blueAntiPastOrder = make(OrderMap)
	var redAntiPastOrder = make(OrderMap)
	var uncoloredUnorderedAntiPast = make(OrderMap)

	// When making ChainMaps of ChainMaps, you need to dereference the member ChainMaps,
	// so that we can tell what type the element in the ChainMap is later on.
	var pastOrder = NewChainMap(*bluePastOrder, *redPastOrder)
	var antiPastOrder = NewChainMap(blueAntiPastOrder, redAntiPastOrder)
	var antiPast = NewChainMap(*antiPastOrder, uncoloredUnorderedAntiPast)
	var coloringOrder = NewChainMap(*bluePastOrder, blueAntiPastOrder)
	var order = NewChainMap(*pastOrder, *antiPastOrder)

	g := GreedyGraphMem{
		k: k,

		children: make(map[string]*OrderedStringSet),
		parents: make(map[string]*OrderedStringSet),
		height: make(map[string]int),

		coloringChain: NewOrderedStringSet(),

		bluePastOrder: bluePastOrder,
		redPastOrder: redPastOrder,
		blueAntiPastOrder: &blueAntiPastOrder,
		redAntiPastOrder: &redAntiPastOrder,
		uncoloredUnorderedAntiPast: &uncoloredUnorderedAntiPast,

		pastOrder:     pastOrder,
		antiPastOrder: antiPastOrder,
		antiPast:      antiPast,
		coloringOrder: coloringOrder,
		order:         order,

		blueCount: make(map[string]int),
		coloringParents: make(map[string]string),
		blueDiffPastOrder: make(map[string]OrderMap),
		redDiffPastOrder: make(map[string]OrderMap),
		selfOrder: make(map[string]int),
	}

	return &g, nil
}