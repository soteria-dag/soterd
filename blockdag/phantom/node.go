// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package phantom

import "sort"

// NODE
// in the BlockDAG case:
// parent of the node has an edge from the node to it: node -> parent
// child of the node has an edge from it to the node: child -> node
type Node struct {
	id string
	parents map[*Node]struct{}
	children map[*Node]struct{}
}

// nodeWithDistance represents a node and its distance from a reference node
type nodeWithDistance struct {
	node *Node
	distance int
}

func newNode(id string) *Node {
	return &Node{
		id: id,
		parents: make(map[*Node]struct{}),
		children: make(map[*Node]struct{}),
	}
}

func (n *Node) GetId() string {
	return n.id
}

func (n *Node) getChildren() *nodeSet {
	set := newNodeSet()

	for k := range n.children {
		set.add(k)
	}

	return set
}

// GetIds returns the ids of the given nodes
func GetIds(nodes []*Node) []string {
	var ids = make([]string, len(nodes))
	for i, node := range nodes {
		ids[i] = node.GetId()
	}
	return ids
}

// SortNodes returns a slice of the nodes that are alphabetically-sorted
// (useful for sorting orderedNodeSet.elements())
func SortNodes(nodes []*Node) []*Node {
	sorted := make([]*Node, len(nodes))
	copy(sorted, nodes)

	less := func(i, j int) bool {
		if nodes[i].GetId() < nodes[j].GetId() {
			return true
		} else {
			return false
		}
	}

	sort.Slice(sorted, less)

	return sorted
}

// sortNodeWithDistance returns a slice of alphabetically-sorted nodeWithDistance elements
func sortNodeWithDistance (nodes []nodeWithDistance) []nodeWithDistance {
	sorted := make([]nodeWithDistance, len(nodes))
	copy(sorted, nodes)

	less := func(i, j int) bool {
		if sorted[i].node.GetId() < sorted[j].node.GetId() {
			return true
		} else {
			return false
		}
	}

	sort.Slice(sorted, less)

	return sorted
}