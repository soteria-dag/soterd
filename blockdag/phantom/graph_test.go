// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package phantom

import (
	"reflect"
	"testing"
)

func TestNodeId(t *testing.T) {
	var node = newNode("A")

	if node.GetId() != "A" {
		t.Errorf("GetId returned wrong node id. Expecting %s, got %s", "A", node.GetId())
	}
}

func TestNodeGetChildren(t *testing.T) {
	var parent = newNode("Parent")
	var c1 = newNode("c1")
	var c2 = newNode("c2")

	parent.children[c1] = keyExists
	parent.children[c2] = keyExists

	var children = parent.getChildren().elements()
	var expected = []*node{c1, c2}
	if !reflect.DeepEqual(children, expected) {
		t.Errorf("Returned wrong child nodes. Expecting %v, got %v",
			getIds(expected), getIds(children))
	}
}

func TestNodeSetAdd(t *testing.T) {
	var nodeSet= newNodeSet()
	var node = newNode("A")
	nodeSet.add(node)

	if _, ok := nodeSet.nodes[node]; !ok {
		t.Errorf("Error adding node to node set")
	}
}

func TestNodeSetRemove(t *testing.T) {
	var nodeSet= newNodeSet()
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
	var expected = []*node{}
	if !reflect.DeepEqual(nodeSet.elements(), expected) {
		t.Errorf("Wrong set of elements, expecting %v, got %v",
			getIds(expected), getIds(nodeSet.elements()))
	}

	var nodeA = newNode("A")
	var nodeB = newNode("B")
	nodeSet.add(nodeA)
	nodeSet.add(nodeB)

	expected = []*node{nodeA, nodeB}

	if !reflect.DeepEqual(nodeSet.elements(), expected) {
		t.Errorf("Wrong set of elements, expecting %v, got %v",
			getIds(expected), getIds(nodeSet.elements()))
	}
}

func TestNodeSetDifference(t *testing.T) {
	var nodeSet1 = newNodeSet()
	var nodeSet2 = newNodeSet()

	var nodeA = newNode("A")

	nodeSet1.add(nodeA)
	nodeSet2.add(nodeA)

	var expected = []*node{}
	var diffSet = nodeSet1.difference(nodeSet2)

	if !reflect.DeepEqual(expected, diffSet.elements()) {
		t.Errorf("Wrong difference of sets, expecting %v, got %v",
			getIds(expected), getIds(diffSet.elements()))
	}

	var nodeB = newNode("B")
	nodeSet2.add(nodeB)

	diffSet = nodeSet1.difference(nodeSet2)
	if !reflect.DeepEqual(expected, diffSet.elements()) {
		t.Errorf("Wrong difference of sets, expecting %v, got %v",
			getIds(expected), getIds(diffSet.elements()))
	}

	diffSet = nodeSet2.difference(nodeSet1)
	expected = []*node{nodeB}
	if !reflect.DeepEqual(expected, diffSet.elements()) {
		t.Errorf("Wrong difference of sets, expecting %v, got %v",
			getIds(expected), getIds(diffSet.elements()))
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
	var expected = []*node{}

	if !reflect.DeepEqual(expected, interSet.elements()) {
		t.Errorf("Wrong intersection of sets, expecting %v, got %v",
			getIds(expected), getIds(interSet.elements()))
	}

	nodeSet2.add(nodeA)
	interSet = nodeSet1.intersection(nodeSet2)
	expected = []*node{nodeA}

	if !reflect.DeepEqual(expected, interSet.elements()) {
		t.Errorf("Wrong intersection of sets, expecting %v, got %v",
			getIds(expected), getIds(interSet.elements()))
	}

	interSet = nodeSet2.intersection(nodeSet1)
	if !reflect.DeepEqual(expected, interSet.elements()) {
		t.Errorf("Wrong intersection of sets, expecting %v, got %v",
			getIds(expected), getIds(interSet.elements()))
	}
}

func TestGraphAddNode(t *testing.T) {
	var g = NewGraph()
	var nodeA = newNode("A")
	g.AddNode(nodeA)

	if _, ok := g.nodes["A"]; !ok {
		t.Errorf("node A not in graph.")
	}

	if _, ok := g.nodes["B"]; ok {
		t.Errorf("node B in graph.")
	}

	g.AddNodeById("B")
	if _, ok := g.nodes["B"]; !ok {
		t.Errorf("node B not in graph.")
	}
}

func TestGraphAddEdge(t *testing.T) {
	var g = NewGraph()
	g.AddNodeById("A")
	g.AddNodeById("B")

	g.AddEdgeById("A", "B")

	var nodeA = g.GetNodeById("A")
	var nodeB = g.GetNodeById("B")
	if _, ok := nodeA.parents[nodeB]; !ok {
		t.Errorf("Edge from A -> B not added to parents correctly.")
	}

	if _, ok := nodeB.children[nodeA]; !ok {
		t.Errorf("Edge from A -> B not added to children correctly.")
	}
}

func TestGraphGetTips(t *testing.T) {
	var g = NewGraph()
	var nodeA = newNode("A")
	var nodeB = newNode("B")
	g.AddNode(nodeA)
	g.AddNode(nodeB)

	var tips = g.GetTips()
	var expected = []*node{nodeA, nodeB}

	if !reflect.DeepEqual(expected, tips) {
		t.Errorf("Incorrect set of tips, expecting %v, got %v",
			getIds(expected), getIds(tips))
	}

	g.AddEdge(nodeA, nodeB)
	tips = g.GetTips()
	expected = []*node{nodeA}

	if !reflect.DeepEqual(expected, tips) {
		t.Errorf("Incorrect set of tips, expecting %v, got %v",
			getIds(expected), getIds(tips))
	}
}

func TestGraphGetPast(t *testing.T) {
	var g = NewGraph()
	g.AddNodeById("GENESIS")
	g.AddNodeById("A")
	g.AddNodeById("B")
	g.AddNodeById("C")
	g.AddEdgeById("A", "GENESIS")
	g.AddEdgeById("C", "GENESIS")
	g.AddEdgeById("B", "A")
	g.AddEdgeById("B", "C")

	//var str = g.PrintGraph()
	var pastB = g.getPast(g.GetNodeById("B")).PrintGraph()
	var expected = "A->GENESIS\nC->GENESIS\nGENESIS\n"
	if pastB != expected {
		t.Errorf("Incorrect graph past for B, expecting %s, got %s", expected, pastB)
	}

	var pastA = g.getPast(g.GetNodeById("A")).PrintGraph()
	expected = "GENESIS\n"
	if pastA != expected {
		t.Errorf("Incorrect graph past for A, expecting %s, got %s", expected, pastA)
	}

	var pastGenesis = g.getPast(g.GetNodeById("GENESIS")).PrintGraph()
	expected = ""
	if pastGenesis != expected {
		t.Errorf("Incorrect graph past for Genesis, expecting %s, got %s", expected, pastGenesis)
	}
}

func TestGraphGetFuture(t *testing.T) {
	var g = NewGraph()
	g.AddNodeById("GENESIS")
	g.AddNodeById("A")
	g.AddNodeById("B")
	g.AddNodeById("C")
	g.AddEdgeById("A", "GENESIS")
	g.AddEdgeById("C", "GENESIS")
	g.AddEdgeById("B", "A")
	g.AddEdgeById("B", "C")

	var futureA = g.getFuture(g.GetNodeById("A"))
	var expected = []*node{g.GetNodeById("B")}
	if !reflect.DeepEqual(expected, futureA.elements()) {
		t.Errorf("Incorrect graph future for A, expecting %v, got %v",
			getIds(expected), getIds(futureA.elements()))
	}

	var futureB = g.getFuture(g.GetNodeById("B"))
	expected = []*node{}
	if !reflect.DeepEqual(expected, futureB.elements()) {
		t.Errorf("Incorrect graph future for B, expecting %v, got %v",
			getIds(expected), getIds(futureB.elements()))
	}

	var futureGenesis = g.getFuture(g.GetNodeById("GENESIS"))
	expected = []*node{g.GetNodeById("A"), g.GetNodeById("B"), g.GetNodeById("C")}
	if !reflect.DeepEqual(expected, futureGenesis.elements()) {
		t.Errorf("Incorrect graph future for Genesis, expecting %v, got %v",
			getIds(expected), getIds(futureGenesis.elements()))
	}
}

func TestGraphGetAnticone(t *testing.T) {
	var g = NewGraph()
	g.AddNodeById("GENESIS")
	g.AddNodeById("A")
	g.AddNodeById("B")
	g.AddNodeById("C")
	g.AddNodeById("D")

	g.AddEdgeById("A", "GENESIS")
	g.AddEdgeById("B", "GENESIS")
	g.AddEdgeById("C", "A")
	g.AddEdgeById("D", "B")
	g.AddEdgeById("D", "C")

	var anticoneA = g.getAnticone(g.GetNodeById("A"))
	var expected = []*node{g.GetNodeById("B")}

	if !reflect.DeepEqual(expected, anticoneA.elements()) {
		t.Errorf("Incorrect anticone of A, expecting %v, got %v",
			getIds(expected), getIds(anticoneA.elements()))
	}

	var anticoneB = g.getAnticone(g.GetNodeById("B"))
	expected = []*node{g.GetNodeById("A"), g.GetNodeById("C")}

	if !reflect.DeepEqual(expected, anticoneB.elements()) {
		t.Errorf("Incorrect anticone of B, expecting %v, got %v",
			getIds(expected), getIds(anticoneB.elements()))
	}

	var anticoneD = g.getAnticone(g.GetNodeById("D"))
	expected = []*node{}

	if !reflect.DeepEqual(expected, anticoneD.elements()) {
		t.Errorf("Incorrect anticone of D, expecting %v, got %v",
			getIds(expected), getIds(anticoneD.elements()))
	}

}
