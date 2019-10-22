// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package phantom

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

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
	var expected = []*Node{nodeA, nodeB}

	if !reflect.DeepEqual(expected, tips) {
		t.Errorf("Incorrect set of tips, expecting %v, got %v",
			GetIds(expected), GetIds(tips))
	}

	g.AddEdge(nodeA, nodeB)
	tips = g.GetTips()
	expected = []*Node{nodeA}

	if !reflect.DeepEqual(expected, tips) {
		t.Errorf("Incorrect set of tips, expecting %v, got %v",
			GetIds(expected), GetIds(tips))
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

func TestGraphGetPastWithHorizon(t *testing.T) {
	var g = NewGraph()
	prevNode := "GENESIS"
	g.AddNodeById(prevNode)
	for i := 0; i < edgeHorizon* 3; i++ {
		node := fmt.Sprintf("A%d", i)
		g.AddNodeById(node)
		g.AddEdgeById(node, prevNode)
		prevNode = node
	}

	extra := 5
	pastNode := fmt.Sprintf("A%d", edgeHorizon+ extra)
	var past = g.getPastWithHorizon(g.getNodeById(pastNode), edgeHorizon)

	beyondHorizonNode := fmt.Sprintf("A%d", extra - 1)
	beyond := past.getNodeById(beyondHorizonNode)
	if beyond != nil {
		t.Errorf("Incorrect horizon for graph past of %s; got %v, want %v",
			pastNode, beyond, nil)
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
	var expected = []*Node{g.GetNodeById("B")}
	if !reflect.DeepEqual(expected, futureA.elements()) {
		t.Errorf("Incorrect graph future for A, expecting %v, got %v",
			GetIds(expected), GetIds(futureA.elements()))
	}

	var futureB = g.getFuture(g.GetNodeById("B"))
	expected = []*Node{}
	if !reflect.DeepEqual(expected, futureB.elements()) {
		t.Errorf("Incorrect graph future for B, expecting %v, got %v",
			GetIds(expected), GetIds(futureB.elements()))
	}

	var futureGenesis = g.getFuture(g.GetNodeById("GENESIS"))
	expected = []*Node{g.GetNodeById("A"), g.GetNodeById("B"), g.GetNodeById("C")}
	if !reflect.DeepEqual(expected, futureGenesis.elements()) {
		t.Errorf("Incorrect graph future for Genesis, expecting %v, got %v",
			GetIds(expected), GetIds(futureGenesis.elements()))
	}
}

func TestGraphGetFutureWithHorizon(t *testing.T) {
	var g = NewGraph()
	prevNode := "GENESIS"
	g.AddNodeById(prevNode)
	for i := 0; i < edgeHorizon* 3; i++ {
		node := fmt.Sprintf("A%d", i)
		g.AddNodeById(node)
		g.AddEdgeById(node, prevNode)
		prevNode = node
	}

	extra := 5
	futureNode := fmt.Sprintf("A%d", edgeHorizon+ extra)
	var future = g.getFutureWithHorizon(g.getNodeById(futureNode), edgeHorizon)

	beyondHorizonNode := fmt.Sprintf("A%d", (edgeHorizon* 2) + extra + 1)
	beyond := g.getNodeById(beyondHorizonNode)
	if future.contains(beyond) {
		t.Errorf("Incorrect horizon for graph future of %s; got true for %v, want false",
			futureNode, beyondHorizonNode)
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
	var expected = []*Node{g.GetNodeById("B")}

	if !reflect.DeepEqual(expected, anticoneA.elements()) {
		t.Errorf("Incorrect anticone of A, expecting %v, got %v",
			GetIds(expected), GetIds(anticoneA.elements()))
	}

	var anticoneB = g.getAnticone(g.GetNodeById("B"))
	expected = []*Node{g.GetNodeById("A"), g.GetNodeById("C")}

	if !reflect.DeepEqual(expected, anticoneB.elements()) {
		t.Errorf("Incorrect anticone of B, expecting %v, got %v",
			GetIds(expected), GetIds(anticoneB.elements()))
	}

	var anticoneD = g.getAnticone(g.GetNodeById("D"))
	expected = []*Node{}

	if !reflect.DeepEqual(expected, anticoneD.elements()) {
		t.Errorf("Incorrect anticone of D, expecting %v, got %v",
			GetIds(expected), GetIds(anticoneD.elements()))
	}

}

func TestGraphGetMissingNodes(t *testing.T) {
	var g = NewGraph()
	g.AddNodeById("GENESIS")
	g.AddNodeById("A")
	g.AddNodeById("B")
	g.AddNodeById("C")

	g.AddEdgeById("A", "GENESIS")
	g.AddEdgeById("B", "GENESIS")
	g.AddEdgeById("C", "GENESIS")

	g.AddNodeById("D")
	g.AddNodeById("E")
	g.AddNodeById("F")
	g.AddNodeById("G")

	g.AddEdgeById("D", "A")
	g.AddEdgeById("E", "A")
	g.AddEdgeById("F", "B")
	g.AddEdgeById("G", "C")

	g.AddNodeById("H")
	g.AddNodeById("I")
	g.AddNodeById("J")
	g.AddNodeById("K")

	g.AddEdgesById("H", []string{"F", "G"})
	g.AddEdgeById("J", "D")
	g.AddEdgesById("I", []string{"E", "F"})
	g.AddEdgesById("K", []string{"I", "J"})

	// no missing nodes with current tips
	missingNodes := g.GetMissingNodes([]string{"H", "K"})
	if len(missingNodes) != 0 {
		t.Errorf("Expecting no missing nodes got %v", missingNodes)
	}

	missingNodes = g.GetMissingNodes([]string{"X", "D", "E", "H"})
	sort.Strings(missingNodes)
	expected := []string{"I", "J", "K"}

	if !reflect.DeepEqual(expected, missingNodes) {
		t.Errorf("Incorrect missing Nodes, expecting %v, got %v",
			expected, missingNodes)
	}

	missingNodes = g.GetMissingNodes([]string{"D", "E"})
	sort.Strings(missingNodes)
	expected = []string{"B", "C", "F", "G", "H", "I", "J", "K"}

	if !reflect.DeepEqual(expected, missingNodes) {
		t.Errorf("Incorrect missing Nodes, expecting %v, got %v",
			expected, missingNodes)
	}
}

func TestGraphRemoveTip(t *testing.T) {

	var g = NewGraph()
	g.AddNodeById("GENESIS")
	g.AddNodeById("A")
	g.AddNodeById("B")
	g.AddNodeById("C")

	g.AddEdgeById("A", "GENESIS")
	g.AddEdgeById("B", "GENESIS")
	g.AddEdgeById("C", "GENESIS")

	g.AddNodeById("D")
	g.AddNodeById("E")
	g.AddNodeById("F")
	g.AddNodeById("G")

	g.AddEdgeById("D", "A")
	g.AddEdgeById("E", "A")
	g.AddEdgeById("F", "B")
	g.AddEdgeById("G", "C")

	tips := g.getTips()
	expected := []string{"D", "E", "F", "G"}

	if !reflect.DeepEqual(expected, GetIds(tips)) {
		t.Errorf("Incorrect tips, expecting %v, got %v",
			expected, GetIds(tips))
	}

	// fail to remove non-tip
	g.RemoveTipById("B")
	if g.getNodeById("B" ) == nil {
		t.Errorf("Non-tip should not be removed")
	}

	// remove tip
	g.RemoveTipById("D")
	if g.getNodeById("D") != nil {
		t.Errorf("Tip still in graph, should be removed")
	}

	tips = g.getTips()
	expected = []string{"E", "F", "G"}
	if !reflect.DeepEqual(expected, GetIds(tips)) {
		t.Errorf("Incorrect tips, expecting %v, got %v",
			expected, GetIds(tips))
	}

	// remove tip, parent should be added to tip set
	g.RemoveTipById("E")
	if g.getNodeById("E") != nil {
		t.Errorf("Tip still in graph, should be removed")
	}

	tips = g.getTips()
	// Tips are ordered by addition. When E was removed, its parent A was added to tips.
	// Tips were F, G, so tips order is now F, G, A.
	expected = []string{"F", "G", "A"}
	if !reflect.DeepEqual(expected, GetIds(tips)) {
		t.Errorf("Incorrect tips, expecting %v, got %v",
			expected, GetIds(tips))
	}
}