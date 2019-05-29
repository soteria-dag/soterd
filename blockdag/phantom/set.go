// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package phantom

import (
	"sort"
)

// NODESET
type nodeSet struct {
	ids map[string]*Node
	nodes map[*Node]struct{}
}

func newNodeSet() *nodeSet {
	return &nodeSet{
		ids: make(map[string]*Node),
		nodes: make(map[*Node]struct{}),
	}
}

func (nset *nodeSet) size() int {
	return len(nset.nodes)
}

func (nset *nodeSet) add(node *Node) {
	if node == nil {
		return
	}

	if nset.contains(node) {
		return
	}

	nset.ids[node.GetId()] = node
	nset.nodes[node] = keyExists
	//fmt.Printf("Added node %s\n", node.id)
}

func (nset *nodeSet) remove(node *Node) {

	n, same := nset.ids[node.GetId()]
	if same && n != node {
		// Attempt to remove other node with the same id as this one
		delete(nset.nodes, n)
		delete(nset.ids, n.GetId())
	}

	// Remove this node
	delete(nset.nodes, node)
	delete(nset.ids, node.GetId())
}

// returns elements of set, sorted by id
func (nset *nodeSet) elements() []*Node {
	nodes := make([]*Node, 0, len(nset.nodes))

	for k := range nset.nodes {
		nodes = append(nodes, k)
	}

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].GetId() < nodes[j].GetId() {
			return true
		} else {
			return false
		}
	})

	return nodes
}

func (nset *nodeSet) contains(node *Node) bool {
	if _, ok := nset.nodes[node]; ok {
		return true
	}

	_, ok := nset.ids[node.GetId()]
	return ok
}

// returns nset - nset2
func (nset *nodeSet) difference(nset2 *nodeSet) *nodeSet {
	diff := newNodeSet()
	if nset2 == nil {
		return diff
	}

	for k := range nset.nodes {
		if !nset2.contains(k) {
			diff.add(k)
		}
	}
	return diff
}

// returns nset intersection nset2
func (nset *nodeSet) intersection(nset2 *nodeSet) *nodeSet {
	intersection := newNodeSet()
	if nset2 == nil {
		return intersection
	}

	for k := range nset.nodes {
		if nset2.contains(k) {
			intersection.add(k)
		}
	}
	return intersection

}

func (nset *nodeSet) clone() *nodeSet {
	clone := newNodeSet()
	for k := range nset.nodes {
		clone.add(k)
	}

	return clone
}

// ORDEREDNODESET
type orderedNodeSet struct {
	ids map[string]*Node
	nodes map[*Node]int
	order map[int]*Node
	counter int
}

func newOrderedNodeSet() *orderedNodeSet {
	return &orderedNodeSet {
		ids: make(map[string]*Node),
		nodes: make(map[*Node]int),
		order: make(map[int]*Node),
		counter: 0,
	}
}

// add the node to the ordered node set, if it's not null and doesn't already exist
func (ons *orderedNodeSet) add(node *Node) {
	if node == nil {
		return
	}

	if ons.contains(node) {
		return
	}

	ons.counter += 1
	ons.ids[node.GetId()] = node
	ons.nodes[node] = ons.counter
	ons.order[ons.counter] = node
}

// clone returns a copy of the ordered node set
func (ons *orderedNodeSet) clone() *orderedNodeSet {
	clone := newOrderedNodeSet()
	for _, k := range ons.elements() {
		clone.add(k)
	}

	return clone
}

// contains returns true if the node is in the ordered node set
func (ons *orderedNodeSet) contains(node *Node) bool {
	_, ok := ons.nodes[node]
	if ok {
		return ok
	}

	_, ok = ons.ids[node.GetId()]
	return ok
}

// difference returns ons - ons2
func (ons *orderedNodeSet) difference(ons2 *orderedNodeSet) *orderedNodeSet {
	diff := newOrderedNodeSet()
	if ons2 == nil {
		return diff
	}

	for _, k := range ons.elements() {
		if !ons2.contains(k){
			diff.add(k)
		}
	}

	return diff
}

// elements returns a list of nodes in the order they were added to the ordered node set
func (ons *orderedNodeSet) elements() []*Node {
	var indexes = make([]int, 0, len(ons.order))
	var nodes = make([]*Node, 0, len(ons.order))
	for k := range ons.order {
		indexes = append(indexes, k)
	}

	sort.Ints(indexes)

	for _, v := range indexes {
		nodes = append(nodes, ons.order[v])
	}
	return nodes
}

// intersection returns ons intersection ons2
func (ons *orderedNodeSet) intersection(ons2 *orderedNodeSet) *orderedNodeSet {
	intersection := newOrderedNodeSet()
	if ons2 == nil {
		return intersection
	}

	for _, k := range ons.elements() {
		if ons2.contains(k) {
			intersection.add(k)
		}
	}

	return intersection
}

// remove a node from the ordered node set
func (ons *orderedNodeSet) remove(node *Node) {
	if node == nil {
		return
	}

	n, same := ons.ids[node.GetId()]
	if same && n != node {
		// Attempt to remove other node with the same id as this one
		i, ok := ons.nodes[n]
		if ok {
			delete(ons.order, i)
		}
		delete(ons.nodes, n)
		delete(ons.ids, n.GetId())
	}

	// Remove this node
	i, ok := ons.nodes[node]
	if ok {
		delete(ons.order, i)
	}
	delete(ons.nodes, node)
	delete(ons.ids, node.GetId())
}

// size returns the size of the ordered node set
func (ons *orderedNodeSet) size() int {
	return len(ons.nodes)
}