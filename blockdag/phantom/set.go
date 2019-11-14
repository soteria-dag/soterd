// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package phantom

import (
	"sort"
)

// NODESET
type nodeSet struct {
	nodes map[*Node]struct{}
}

func newNodeSet() *nodeSet {
	return &nodeSet{
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

	nset.nodes[node] = keyExists
	//fmt.Printf("Added node %s\n", node.id)
}

func (nset *nodeSet) remove(node *Node) {
	// Remove this node
	delete(nset.nodes, node)
}

// returns elements of set, sorted by id
func (nset *nodeSet) elements() []*Node {
	nodes := make([]*Node, len(nset.nodes))
	index := 0
	for k := range nset.nodes {
		nodes[index] = k
		index += 1
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
	_, ok := nset.nodes[node]
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
	nodes map[*Node]int
	order map[int]*Node
	counter int
}

func newOrderedNodeSet() *orderedNodeSet {
	return &orderedNodeSet {
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

	// Remove this node
	i, ok := ons.nodes[node]
	if ok {
		delete(ons.order, i)
	}
	delete(ons.nodes, node)
}

// size returns the size of the ordered node set
func (ons *orderedNodeSet) size() int {
	return len(ons.nodes)
}

// An alternative to using a container/list List type, for managing a list of nodes
type nodeList struct {
	elements []*Node
}

func newNodeList(cap int) *nodeList {
	return &nodeList{
		elements: make([]*Node, 0, cap),
	}
}

// at returns the node at the given index, or nil if there's no Node there.
func (nl *nodeList) at(i int) *Node {
	if i > len(nl.elements) - 1 || i < 0 {
		return nil
	}

	return nl.elements[i]
}

// delete removes a node at the given index, and returns it
func (nl *nodeList) delete(i int) *Node {
	end := len(nl.elements) - 1
	n := nl.elements[i]
	// Copy values from the deletion point to the left by one
	copy(nl.elements[i:], nl.elements[i+1:])
	// Dereference the last value
	nl.elements[end] = nil
	// Truncate the slice
	nl.elements = nl.elements[:end]

	return n
}

// insert adds a Node at the given index.
// This method of insertion avoids creating a new slice, so garbage collection isn't involved.
func (nl *nodeList) insert(i int, n *Node) {
	// Add a nil value to the end of the slice, to make room for the new Node.
	nl.elements = append(nl.elements, nil)
	// Copy values from the insertion point to the right by one
	copy(nl.elements[i+1:], nl.elements[i:])
	// Set the value at the insertion point
	nl.elements[i] = n
}

// push adds a Node to the end of the list
func (nl *nodeList) push(n *Node) {
	nl.elements = append(nl.elements, n)
}

// pop removes a Node from the end of the list, and returns it. nil is returned if there's no Node to pop.
func (nl *nodeList) pop() *Node {
	size := len(nl.elements)
	if size == 0 {
		return nil
	}

	// This method of deletion is used instead of calling nl.Delete(), because it's faster.
	end := size - 1
	n := nl.elements[end]
	nl.elements[end] = nil
	nl.elements = nl.elements[0:end]

	return n
}

// shift removes a Node from the front of the list, and returns it
func (nl *nodeList) shift() *Node {
	if len(nl.elements) == 0 {
		return nil
	}

	// This method of deletion is used instead of calling nl.Delete(), because it's faster.
	n := nl.elements[0]
	nl.elements[0] = nil
	nl.elements = nl.elements[1:]
	return n
}

// size returns the size of the nodeList
func (nl *nodeList) size() int {
	return len(nl.elements)
}

// unshift adds a Node to the front of the list
func (nl *nodeList) unshift(n *Node) {
	nl.insert(0, n)
}