// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package phantom

import (
	"reflect"
	"testing"
)

func TestNodeSetAdd(t *testing.T) {
	var nodeSet = newNodeSet()
	var node = newNode("A")
	nodeSet.add(node)

	if _, ok := nodeSet.nodes[node]; !ok {
		t.Errorf("Error adding node to node set")
	}
}

func TestNodeSetRemove(t *testing.T) {
	var nodeSet = newNodeSet()
	var node = newNode("A")
	nodeSet.add(node)
	nodeSet.remove(node)

	if _, ok := nodeSet.nodes[node]; ok {
		t.Errorf("Error removing node from node set")
	}
}

func TestNodeSetContains(t *testing.T) {
	var nodeSet = newNodeSet()
	var nodeA = newNode("A")
	var nodeB = newNode("B")

	nodeSet.add(nodeA)

	if !nodeSet.contains(nodeA) {
		t.Errorf("node set should contain node A.")
	}

	if nodeSet.contains(nodeB) {
		t.Errorf("node set should not contain node B.")
	}
}

func TestNodeSetSize(t *testing.T) {
	var nodeSet = newNodeSet()

	if nodeSet.size() != 0 {
		t.Errorf("Wrong node set size, expecting %d, got %d", 0, nodeSet.size())
	}

	var nodeA = newNode("A")
	var nodeB = newNode("B")

	nodeSet.add(nodeA)

	if nodeSet.size() != 1 {
		t.Errorf("Wrong node set size, expecting %d, got %d", 1, nodeSet.size())
	}

	nodeSet.add(nodeB)

	if nodeSet.size() != 2 {
		t.Errorf("Wrong node set size, expecting %d, got %d", 2, nodeSet.size())
	}
}

func TestNodeSetElements(t *testing.T) {
	var nodeSet = newNodeSet()
	var expected = []*Node{}
	if !reflect.DeepEqual(nodeSet.elements(), expected) {
		t.Errorf("Wrong set of elements, expecting %v, got %v",
			GetIds(expected), GetIds(nodeSet.elements()))
	}

	var nodeA = newNode("A")
	var nodeB = newNode("B")
	nodeSet.add(nodeA)
	nodeSet.add(nodeB)

	expected = []*Node{nodeA, nodeB}

	if !reflect.DeepEqual(nodeSet.elements(), expected) {
		t.Errorf("Wrong set of elements, expecting %v, got %v",
			GetIds(expected), GetIds(nodeSet.elements()))
	}
}

func TestNodeSetDifference(t *testing.T) {
	var nodeSet1 = newNodeSet()
	var nodeSet2 = newNodeSet()

	var nodeA = newNode("A")

	nodeSet1.add(nodeA)
	nodeSet2.add(nodeA)

	var expected = []*Node{}
	var diffSet = nodeSet1.difference(nodeSet2)

	if !reflect.DeepEqual(expected, diffSet.elements()) {
		t.Errorf("Wrong difference of sets, expecting %v, got %v",
			GetIds(expected), GetIds(diffSet.elements()))
	}

	var nodeB = newNode("B")
	nodeSet2.add(nodeB)

	diffSet = nodeSet1.difference(nodeSet2)
	if !reflect.DeepEqual(expected, diffSet.elements()) {
		t.Errorf("Wrong difference of sets, expecting %v, got %v",
			GetIds(expected), GetIds(diffSet.elements()))
	}

	diffSet = nodeSet2.difference(nodeSet1)
	expected = []*Node{nodeB}
	if !reflect.DeepEqual(expected, diffSet.elements()) {
		t.Errorf("Wrong difference of sets, expecting %v, got %v",
			GetIds(expected), GetIds(diffSet.elements()))
	}
}

func TestNodeSetIntersection(t *testing.T) {
	var nodeSet1 = newNodeSet()
	var nodeSet2 = newNodeSet()

	var nodeA = newNode("A")
	var nodeB = newNode("B")
	nodeSet1.add(nodeA)
	nodeSet2.add(nodeB)

	var interSet = nodeSet1.intersection(nodeSet2)
	var expected = []*Node{}

	if !reflect.DeepEqual(expected, interSet.elements()) {
		t.Errorf("Wrong intersection of sets, expecting %v, got %v",
			GetIds(expected), GetIds(interSet.elements()))
	}

	nodeSet2.add(nodeA)
	interSet = nodeSet1.intersection(nodeSet2)
	expected = []*Node{nodeA}

	if !reflect.DeepEqual(expected, interSet.elements()) {
		t.Errorf("Wrong intersection of sets, expecting %v, got %v",
			GetIds(expected), GetIds(interSet.elements()))
	}

	interSet = nodeSet2.intersection(nodeSet1)
	if !reflect.DeepEqual(expected, interSet.elements()) {
		t.Errorf("Wrong intersection of sets, expecting %v, got %v",
			GetIds(expected), GetIds(interSet.elements()))
	}
}

func TestOrderedNodeSetAdd(t *testing.T) {
	ons := newOrderedNodeSet()
	node := newNode("A")
	ons.add(node)

	if _, ok := ons.nodes[node]; !ok {
		t.Errorf("Error adding node %s to ordered node set", node.GetId())
	}

	if _, ok := ons.order[1]; !ok {
		t.Errorf("Error adding node %s to ordered node set, no order entry for key %d", node.GetId(), 1)
	}
}

func TestOrderedNodeSetRemove(t *testing.T) {
	ons := newOrderedNodeSet()
	nodeA := newNode("A")
	nodeB := newNode("B")
	nodeC := newNode("C")
	ons.add(nodeA)
	ons.add(nodeB)
	ons.add(nodeC)
	ons.remove(nodeB)

	if _, ok := ons.nodes[nodeB]; ok {
		t.Errorf("Error removing node %s from ordered node set", nodeB.GetId())
	}

	expected := []*Node{nodeA, nodeC}
	if !reflect.DeepEqual(ons.elements(), expected) {
		t.Errorf("Order is incorrect after node %s removed; got %v, want %v",
			nodeB.id, GetIds(ons.elements()), GetIds(expected))
	}

	ons.add(nodeB)
	expected = []*Node{nodeA, nodeC, nodeB}
	if !reflect.DeepEqual(ons.elements(), expected) {
		t.Errorf("Order is incorrect after node %s removed; got %v, want %v",
			nodeB.id, GetIds(ons.elements()), GetIds(expected))
	}
}

func TestOrderedNodeSetContains(t *testing.T) {
	ons := newOrderedNodeSet()
	nodeA := newNode("A")
	nodeB := newNode("B")
	ons.add(nodeA)

	if !ons.contains(nodeA) {
		t.Errorf("Ordered node set should contain node %s.", nodeA.GetId())
	}

	if ons.contains(nodeB) {
		t.Errorf("Ordered node set should not contain node %s.", nodeB.GetId())
	}
}

func TestOrderedNodeSetSize(t *testing.T) {
	var ons = newOrderedNodeSet()

	if ons.size() != 0 {
		t.Errorf("Wrong ordered node set size; got %d, want %d", ons.size(), 0)
	}

	var nodeA = newNode("A")
	var nodeB = newNode("B")
	var nodeC = newNode("C")

	ons.add(nodeA)

	if ons.size() != 1 {
		t.Errorf("Wrong ordered node set size; got %d, want %d", ons.size(), 1)
	}

	ons.add(nodeB)
	ons.add(nodeC)

	if ons.size() != 3 {
		t.Errorf("Wrong ordered node set size; got %d, want %d", ons.size(), 3)
	}

	ons.remove(nodeB)

	if ons.size() != 2 {
		t.Errorf("Wrong ordered node set size; got %d, want %d", ons.size(), 2)
	}

	ons.add(nodeB)
	if ons.size() != 3 {
		t.Errorf("Wrong ordered node set size; got %d, want %d", ons.size(), 3)
	}
}

func TestOrderedNodeSetElements(t *testing.T) {
	var ons = newOrderedNodeSet()
	var expected = []*Node{}

	if !reflect.DeepEqual(ons.elements(), expected) {
		t.Errorf("Wrong set of elements; got %v, want %v", GetIds(ons.elements()), GetIds(expected))
	}

	var nodeA = newNode("A")
	var nodeB = newNode("B")
	var nodeC = newNode("C")
	ons.add(nodeA)
	ons.add(nodeB)
	ons.add(nodeC)

	expected = []*Node{nodeA, nodeB, nodeC}
	if !reflect.DeepEqual(ons.elements(), expected) {
		t.Errorf("Wrong set of elements; got %v, want %v", GetIds(ons.elements()), GetIds(expected))
	}

	ons.remove(nodeB)
	expected = []*Node{nodeA, nodeC}
	if !reflect.DeepEqual(ons.elements(), expected) {
		t.Errorf("Wrong set of elements; got %v, want %v", GetIds(ons.elements()), GetIds(expected))
	}

	ons.add(nodeB)
	expected = []*Node{nodeA, nodeC, nodeB}
	if !reflect.DeepEqual(ons.elements(), expected) {
		t.Errorf("Wrong set of elements; got %v, want %v", GetIds(ons.elements()), GetIds(expected))
	}
}

func TestOrderedNodeSetDifference(t *testing.T) {
	var ons1 = newOrderedNodeSet()
	var ons2 = newOrderedNodeSet()

	var nodeA = newNode("A")
	var nodeB = newNode("B")

	var expected = []*Node{}
	var diff = ons1.difference(ons2)

	if !reflect.DeepEqual(diff.elements(), expected) {
		t.Errorf("Wrong difference of sets; got %v, want %v",
			GetIds(diff.elements()), GetIds(expected))
	}

	ons1.add(nodeA)
	ons2.add(nodeA)
	diff = ons1.difference(ons2)
	if !reflect.DeepEqual(diff.elements(), expected) {
		t.Errorf("Wrong difference of sets; got %v, want %v",
			GetIds(diff.elements()), GetIds(expected))
	}

	ons2.add(nodeB)
	diff = ons1.difference(ons2)
	if !reflect.DeepEqual(diff.elements(), expected) {
		t.Errorf("Wrong difference of sets; got %v, want %v",
			GetIds(diff.elements()), GetIds(expected))
	}

	diff = ons2.difference(ons1)
	expected = []*Node{nodeB}
	if !reflect.DeepEqual(diff.elements(), expected) {
		t.Errorf("Wrong difference of sets; got %v, want %v",
			GetIds(diff.elements()), GetIds(expected))
	}
}

func TestOrderedNodeSetIntersection(t *testing.T) {
	var ons1 = newOrderedNodeSet()
	var ons2 = newOrderedNodeSet()

	var nodeA = newNode("A")
	var nodeB = newNode("B")

	var expected = []*Node{}
	var inter = ons1.intersection(ons2)

	if !reflect.DeepEqual(inter.elements(), expected) {
		t.Errorf("Wrong intersection of sets; got %v, want %v",
			GetIds(inter.elements()), GetIds(expected))
	}

	ons1.add(nodeA)
	ons2.add(nodeB)
	inter = ons1.intersection(ons2)
	if !reflect.DeepEqual(inter.elements(), expected) {
		t.Errorf("Wrong intersection of sets; got %v, want %v",
			GetIds(inter.elements()), GetIds(expected))
	}

	ons2.add(nodeA)
	inter = ons1.intersection(ons2)
	expected = []*Node{nodeA}
	if !reflect.DeepEqual(inter.elements(), expected) {
		t.Errorf("Wrong intersection of sets; got %v, want %v",
			GetIds(inter.elements()), GetIds(expected))
	}

	inter = ons2.intersection(ons1)
	if !reflect.DeepEqual(inter.elements(), expected) {
		t.Errorf("Wrong intersection of sets; got %v, want %v",
			GetIds(inter.elements()), GetIds(expected))
	}
}