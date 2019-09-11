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

func genPhantomFig3GreedyGraphMem(g *GreedyGraphMem) error {
	nodes := []string{"GENESIS", "B", "C", "D", "E", "F", "H", "I", "J", "K", "L", "M"}
	for _, id := range nodes {
		ok, err := g.AddNode(id)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("node %s not added to graph", id)
		}
	}

	edges := []struct{
		id string
		parents []string
	}{
		{"GENESIS", []string{}},
		{"B", []string{"GENESIS"}},
		{"C", []string{"GENESIS"}},
		{"D", []string{"GENESIS"}},
		{"E",  []string{"GENESIS"}},
		{"F", []string{"B", "C"}},
		{"H", []string{"C", "D", "E"}},
		{"I", []string{"E"}},
		{"J", []string{"F", "H"}},
		{"K", []string{"B", "H", "I"}},
		{"L", []string{"D", "I"}},
		{"M", []string{"F", "K"}},
	}

	for _, e := range edges {
		_, err := g.Add(e.id, e.parents)
		if err != nil {
			return fmt.Errorf("edges %s -> %s not added to graph: %s", e.id, e.parents, err)
		}
	}

	return nil
}

func genPhantomFig4GreedyGraphMem(g *GreedyGraphMem) error {
	nodes := []string{"GENESIS", "B", "C", "D", "E", "F", "H", "I", "J", "K",
		"L", "M", "N", "O", "P", "Q", "R", "S", "T", "U"}
	for _, id := range nodes {
		ok, err := g.AddNode(id)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("node %s not added to graph", id)
		}
	}

	edges := []struct{
		id string
		parents []string
	}{
		{"GENESIS", []string{}},
		{"B", []string{"GENESIS"}},
		{"C", []string{"GENESIS"}},
		{"D", []string{"GENESIS"}},
		{"E",  []string{"GENESIS"}},
		{"F", []string{"B", "C"}},
		{"H", []string{"E"}},
		{"I", []string{"C", "D"}},
		{"J", []string{"D", "F"}},
		{"K", []string{"E", "I", "J"}},
		{"L", []string{"F"}},
		{"M", []string{"K", "L"}},
		{"N", []string{"D", "H"}},
		{"O", []string{"K"}},
		{"P", []string{"K"}},
		{"Q", []string{"N"}},
		{"R", []string{"N", "O", "P"}},
		{"S", []string{"Q"}},
		{"T", []string{"S"}},
		{"U", []string{"T"}},
	}

	for _, e := range edges {
		_, err := g.Add(e.id, e.parents)
		if err != nil {
			return fmt.Errorf("edges %s -> %s not added to graph: %s", e.id, e.parents, err)
		}
	}

	return nil
}

func TestGreedyGraphMem_AddNode(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	var a = "A"
	ok, err := g.AddNode(a)
	if err != nil {
		t.Errorf("failed to add node %s: %s", a, err)
	}

	if !ok {
		t.Errorf("node %s not added to graph", a)
	}

	// Confirm that node is in db
	nodes := g.Nodes()

	expected := []string{a}
	if !reflect.DeepEqual(nodes, expected) {
		t.Errorf("node %s not in graph; got %v, want %v", a, nodes, expected)
	}

	// A second attempt to add the node should return ok = false
	ok, err = g.AddNode(a)
	if err != nil {
		t.Errorf("failed to add node %s: %s", a, err)
	}

	if ok {
		t.Errorf("should not have added node %s twice", a)
	}
}

func TestGreedyGraphMem_AddEdge(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	var a = "A"
	var b = "B"
	ok, err := g.AddNode(a)
	if err != nil {
		t.Errorf("failed to add node %s: %s", a, err)
	}

	ok, err = g.AddNode(b)
	if err != nil {
		t.Errorf("failed to add node %s: %s", b, err)
	}

	ok, err = g.AddEdge(a, b)
	if err != nil {
		t.Errorf("failed to add edge %s -> %s: %s", a, b, err)
	}

	if !ok {
		t.Errorf("edge %s -> %s not added to graph", a, b)
	}

	// Confirm edges
	parents, err := g.Parents(a)
	if err != nil {
		t.Errorf("failed to get parents of %s: %s", a, err)
	}

	expected := []string{b}
	if !reflect.DeepEqual(parents, expected) {
		t.Errorf("wrong parents for %s; got %v, want %v", a, parents, expected)
	}

	// A second attempt to add the edge should return ok = false
	ok, err = g.AddEdge(a, b)
	if err != nil {
		t.Errorf("failed to add edge %s -> %s: %s", a, b, err)
	}

	if ok {
		t.Errorf("should not have succeeded in adding edge %s -> %s twice", a, b)
	}
}

func TestGreedyGraphMem_AddMultiEdge(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	var a = "A"
	var b = "B"
	var c = "C"
	parents := []string{b, c}

	_, _ = g.AddNode(b)
	_, _ = g.AddNode(c)
	ok, err := g.AddMultiEdge(a, parents)
	if err != nil {
		t.Errorf("failed to add edges %s -> %s", a, parents)
	}

	if !ok {
		t.Errorf("edges %s -> %s not added to graph", a, parents)
	}

	// Confirm edges
	got, err := g.Parents(a)
	if err != nil {
		t.Errorf("failed to get parents of %s: %s", a, err)
	}

	if !reflect.DeepEqual(got, parents) {
		t.Errorf("wrong parents for %s; got %v, want %v", a, got, parents)
	}

	// A second attempt to add the edges should return ok = false
	ok, err = g.AddMultiEdge(a, parents)
	if err != nil {
		t.Errorf("failed to add edges %s -> %s", a, parents)
	}

	if ok {
		t.Errorf("should not have succeeded in adding edges %s -> %s twice", a, parents)
	}
}

func TestGreedyGraphMem_Add(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	var b = "B"
	var c = "C"
	_, _ = g.AddNode(b)
	_, _ = g.AddNode(c)

	var a = "A"
	var parents = []string{b, c}
	ok, err := g.Add(a, parents)
	if err != nil {
		t.Errorf("failed to add node %s: %s", a, err)
	}

	if !ok {
		t.Errorf("node %s not added to graph", a)
	}

	// Confirm that node is in db
	nodes := g.Nodes()
	sort.Strings(nodes)

	expected := []string{a, b, c}
	if !reflect.DeepEqual(nodes, expected) {
		t.Errorf("node %s not in graph; got %v, want %v", a, nodes, expected)
	}

	// Confirm node's parents in db
	got, err := g.Parents(a)
	if err != nil {
		t.Errorf("failed to get parents of %s: %s", a, err)
	}

	if !reflect.DeepEqual(got, parents) {
		t.Errorf("wrong parents for node %s; got %v, want %v", a, got, parents)
	}

	// A second attempt to add the node should return ok = false,
	ok, err = g.Add(a, parents)
	if err != nil {
		t.Errorf("failed to add node %s: %s", a, err)
	}

	if ok {
		t.Errorf("should not have added node %s twice", a)
	}

	// TODO(cedric): And the node's blueCount should remain the same
}

func TestGreedyGraphMem_RemoveEdge(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	var a = "A"
	var b = "B"
	_, _ = g.AddNode(a)
	_, _ = g.AddNode(b)
	_, _ = g.AddEdge(a, b)

	ok := g.RemoveEdge(a, b)

	if !ok {
		t.Errorf("edge %s -> %s not removed from graph", a, b)
	}

	// Confirm edges removed
	parents, err := g.Parents(a)
	if err != nil {
		t.Errorf("failed to get parents of %s: %s", a, err)
	}

	if len(parents) > 0 {
		t.Errorf("should not have any parents for %s; got %v, want %v", a, parents, []string{})
	}

	// A second attempt to remove the edge should return ok = false
	ok = g.RemoveEdge(a, b)

	if ok {
		t.Errorf("should not have succeeded in removing edge %s -> %s twice", a, b)
	}

	// TODO(cedric): confirm blue counts and related removed or updated
}

func TestGreedyGraphMem_RemoveTip(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	err = genPhantomFig3GreedyGraphMem(g)
	if err != nil {
		t.Errorf("failed to generate phantom paper figure 3: %s", err)
	}

	var c = "C"
	var m = "M"

	// Check that removal of non-tip is a noop
	err = g.RemoveTip(c)
	if err != nil {
		t.Errorf("failed RemoveTip for %s: %s", c, err)
	}

	nodes := g.Nodes()
	allNodes := NewOrderedStringSet(nodes...)
	if !allNodes.Contains(c) {
		t.Errorf("should not have removed node %s", c)
	}

	// Attempt to remove a tip
	err = g.RemoveTip(m)
	if err != nil {
		t.Errorf("failed RemoveTip for %s: %s", m, err)
	}

	// Define a function that will check the db for any references to the node
	nodeExists := func(id string) bool {
		_, inParents := g.parents[id]
		_, inChildren := g.children[id]
		_, inHeight := g.height[id]

		return inParents || inChildren || inHeight
	}

	stillExists := nodeExists(m)

	if stillExists {
		t.Errorf("failed to remove all references to node %s", m)
	}

	// Confirm what the new tip is
	if g.coloringTip == "" {
		t.Errorf("graph should have a new coloring tip")
	}

	var j = "J"
	expectedTip := j
	if g.coloringTip != expectedTip {
		t.Errorf("wrong new coloring tip; got %s, want %s", g.coloringTip, expectedTip)
	}

	expectedChain := g.getKChain(j)
	got := g.mainKChain

	if !reflect.DeepEqual(got.chain.Elements(), expectedChain.chain.Elements()) {
		t.Errorf("wrong main coloring chain; got %v, want %v", got.chain.Elements(), expectedChain.chain.Elements())
	}
}

func TestGreedyGraphMem_NodeExists(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	err = genPhantomFig3GreedyGraphMem(g)
	if err != nil {
		t.Errorf("failed to generate phantom paper figure 3: %s", err)
	}

	var h = "H"
	ok, err := g.NodeExists(h)
	if err != nil {
		t.Errorf("failed to check if node %s exists: %s", h, err)
	}

	if !ok {
		t.Errorf("missing node %s", h)
	}

	var nonExistent = "BANANA"
	ok, err = g.NodeExists(nonExistent)
	if err != nil {
		t.Errorf("failed to check if node %s exists: %s", nonExistent, err)
	}

	if ok {
		t.Errorf("node %s shouldn't exist in graph", nonExistent)
	}
}

func TestGreedyGraphMem_Nodes(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	err = genPhantomFig3GreedyGraphMem(g)
	if err != nil {
		t.Errorf("failed to generate phantom paper figure 3: %s", err)
	}

	nodes := g.Nodes()

	sort.Strings(nodes)
	expected := []string{"B", "C", "D", "E", "F", "GENESIS", "H", "I", "J", "K", "L", "M"}
	if !reflect.DeepEqual(nodes, expected) {
		t.Errorf("wrong nodes found; got %s, want %s", nodes, expected)
	}
}

func TestGreedyGraphMem_Tips(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	err = genPhantomFig3GreedyGraphMem(g)
	if err != nil {
		t.Errorf("failed to generate phantom paper figure 3: %s", err)
	}

	tips, err := g.Tips()
	if err != nil {
		t.Fatalf("failed to get tips: %s", err)
	}

	expected := []string{"J", "L", "M"}
	if !reflect.DeepEqual(tips, expected) {
		t.Errorf("wrong tips found; got %s, want %s", tips, expected)
	}
}

func TestGreedyGraphMem_TipDiff(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new Graph2: %s", err)
	}

	err = genPhantomFig3GreedyGraphMem(g)
	if err != nil {
		t.Errorf("failed to generate phantom fig3: %s", err)
	}

	subTips := []string{"H", "I"}
	diff, err := g.TipDiff(subTips)
	if err != nil {
		t.Errorf("failed to get tip diff for %s: %s", subTips, err)
	}

	expected := []string{"B", "F", "J", "K", "L", "M"}
	sort.Strings(diff)

	if !reflect.DeepEqual(diff, expected) {
		t.Errorf("wrong tip diff for %s; got %s, want %s", subTips, diff, expected)
	}
}

func TestGreedyGraphMem_Parents(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	err = genPhantomFig3GreedyGraphMem(g)
	if err != nil {
		t.Errorf("failed to generate phantom paper figure 3: %s", err)
	}

	h := "H"

	parents, err := g.Parents(h)
	if err != nil {
		t.Fatalf("failed to get parents of %s: %s", h, err)
	}

	expected := []string{"C", "D", "E"}
	if !reflect.DeepEqual(parents, expected) {
		t.Errorf("wrong parents of %s found; got %s, want %s", h, parents, expected)
	}
}

func TestGreedyGraphMem_Past(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	err = genPhantomFig3GreedyGraphMem(g)
	if err != nil {
		t.Errorf("failed to generate phantom paper figure 3: %s", err)
	}

	j := "J"

	past, err := g.Past(j)
	if err != nil {
		t.Fatalf("failed to get past of %s: %s", j, err)
	}

	expected := []string{"F", "H", "B", "C", "D", "E", "GENESIS"}
	if !reflect.DeepEqual(past, expected) {
		t.Errorf("wrong past of %s found; got %s, want %s", j, past, expected)
	}
}

func TestGreedyGraphMem_Height(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	err = genPhantomFig3GreedyGraphMem(g)
	if err != nil {
		t.Errorf("failed to generate phantom paper figure 3: %s", err)
	}

	var node = "K"

	// Node K has multiple paths from genesis, one with height 2, and one with height 3.
	// g.Height() should return the maximum height.
	height, err := g.Height(node)
	if err != nil {
		t.Errorf("failed to get height for node %s: %s", node, err)
	}

	expected := 3
	if height != expected {
		t.Errorf("wrong height for node %s; got %d, want %d", node, height, expected)
	}
}

func TestGreedyGraphMem_Order_Fig3(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	err = genPhantomFig3GreedyGraphMem(g)
	if err != nil {
		t.Errorf("failed to generate phantom fig3: %s", err)
	}

	order, err := g.Order()
	if err != nil {
		t.Errorf("failed to get order: %s", err)
	}

	// This is the order we expect from a greedy PHANTOM sort
	expected := []string{"GENESIS", "C", "D", "E", "H", "B", "I", "K", "F", "M", "J", "L"}
	if !reflect.DeepEqual(order, expected) {
		t.Errorf("wrong order for k=%d; got %s, want %s", k, order, expected)
	}
}

func TestGreedyGraphMem_Order_Fig4(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	err = genPhantomFig4GreedyGraphMem(g)
	if err != nil {
		t.Errorf("failed to generate phantom fig3: %s", err)
	}

	order, err := g.Order()
	if err != nil {
		t.Errorf("failed to get order: %s", err)
	}

	// This is the order we expect from a greedy PHANTOM sort
	expected := []string{"GENESIS", "B", "C", "F", "D", "J", "E", "I", "K", "O", "P",
		"H", "N", "R", "L", "M", "Q", "S", "T", "U"}
	if !reflect.DeepEqual(order, expected) {
		t.Errorf("wrong order for k=%d; got %s, want %s", k, order, expected)
	}
}

func TestGreedyGraphMem_ToString(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new Graph2: %s", err)
	}

	err = genPhantomFig3GreedyGraphMem(g)
	if err != nil {
		t.Errorf("failed to generate phantom fig3: %s", err)
	}

	s, err := g.ToString()
	if err != nil {
		t.Errorf("failed to convert graph to string: %s", err)
	}

	expected := `J->F
J->H
L->D
L->I
M->F
M->K
F->B
F->C
H->C
H->D
H->E
D->GENESIS
I->E
K->B
K->H
K->I
B->GENESIS
C->GENESIS
E->GENESIS
GENESIS
`

	if s != expected {
		t.Errorf("graph string doesn't match")
	}
}

func TestGreedyGraphMem_clearAntiPastOrder(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	var a = "A"
	var b = "B"

	g.blueAntiPastOrder.Add(a)
	g.redAntiPastOrder.Add(b)

	g.clearAntiPastOrder()

	if g.blueAntiPastOrder.Contains(a) {
		t.Errorf("node %s should not be in blueAntiPastOrder", a)
	}

	if g.redAntiPastOrder.Contains(b) {
		t.Errorf("node %s should not be in redAntiPastOrder", b)
	}
}

func TestGreedyGraphMem_isBlue(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	err = genPhantomFig3GreedyGraphMem(g)
	if err != nil {
		t.Errorf("failed to generate phantom fig3: %s", err)
	}

	m := "M"
	ok := g.isBlue(m)
	if !ok {
		t.Errorf("node %s should be blue", m)
	}

	l := "L"
	ok = g.isBlue(l)
	if ok {
		t.Errorf("node %s should be red", l)
	}
}

func TestGreedyGraphMem_getColoring(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	err = genPhantomFig3GreedyGraphMem(g)
	if err != nil {
		t.Errorf("failed to generate phantom fig3: %s", err)
	}

	coloring := g.getColoring()
	expected := []string{"C", "D", "E", "GENESIS", "K", "H", "J", "M"}
	if !reflect.DeepEqual(coloring.Elements(), expected) {
		t.Errorf("wrong coloring; got %v, want %v", coloring.Elements(), expected)
	}
}

func TestGreedyGraphMem_sortByBluest(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	err = genPhantomFig3GreedyGraphMem(g)
	if err != nil {
		t.Errorf("failed to generate phantom fig3: %s", err)
	}

	var nodes = []string{"K", "L", "J"}
	sorted := g.sortByBluest(nodes...)
	expected := []string{"L", "K", "J"}
	if !reflect.DeepEqual(sorted, expected) {
		t.Errorf("wrong sortByBluest order; got %v, want %v", sorted, expected)
	}
}

func TestGreedyGraphMem_getBluest(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	err = genPhantomFig3GreedyGraphMem(g)
	if err != nil {
		t.Errorf("failed to generate phantom fig3: %s", err)
	}

	var nodes = []string{"K", "L", "J"}
	bluest := g.getBluest(nodes...)
	expected := "J"
	if bluest != expected {
		t.Errorf("wrong bluest; got %s, want %s", bluest, expected)
	}
}

func TestGreedyGraphMem_coloringRule2(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	err = genPhantomFig4GreedyGraphMem(g)
	if err != nil {
		t.Errorf("failed to generate phantom fig3: %s", err)
	}

	mainChain := g.mainKChain

	var m = "M"
	ok := g.coloringRule2(mainChain, m)
	if !ok {
		t.Errorf("%s should be blue", m)
	}

	var s = "S"
	ok = g.coloringRule2(mainChain, s)
	if ok {
		t.Errorf("%s should be red", s)
	}
}

func TestGreedyGraphMem_colorNode(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	err = genPhantomFig4GreedyGraphMem(g)
	if err != nil {
		t.Errorf("failed to generate phantom fig3: %s", err)
	}

	var blue = make(OrderMap)
	var red = make(OrderMap)
	var m = "M"
	var s = "S"
	mainChain := g.mainKChain

	g.colorNode(&blue, &red, mainChain, m)
	g.colorNode(&blue, &red, mainChain, s)

	if !blue.Contains(m) {
		t.Errorf("%s should be in blueOrder", m)
	}

	if red.Contains(m) {
		t.Errorf("%s should not be in redOrder", m)
	}

	if blue.Contains(s) {
		t.Errorf("%s should not be in blueOrder", s)
	}

	if !red.Contains(s) {
		t.Errorf("%s should be in redOrder", s)
	}
}

func TestGreedyGraphMem_getKChain(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	err = genPhantomFig3GreedyGraphMem(g)
	if err != nil {
		t.Errorf("failed to generate phantom fig3: %s", err)
	}

	var h = "H"
	var nonExistent = "BANANA"

	noChain := g.getKChain(nonExistent)
	// Aside from the non-existent node, there shouldn't be any other members
	if noChain.chain.Difference(NewOrderedStringSet(nonExistent)).Size() > 0 {
		t.Errorf("chain for non existent node should be empty")
	}

	if noChain.minHeight != -1 {
		t.Errorf("wrong minHeight for chain of non existent node; got %d, want %d", noChain.minHeight, -1)
	}

	hChain := g.getKChain(h)
	expected := []string{"H", "C"}
	if !reflect.DeepEqual(hChain.chain.Elements(), expected) {
		t.Errorf("wrong kChain for %s; got %v, want %v", h, hChain.chain.Elements(), expected)
	}

	if hChain.minHeight != 1 {
		t.Errorf("wrong minHeight for chain of %s; got %d, want %d", h, hChain.minHeight, 1)
	}

	g2, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	err = genPhantomFig4GreedyGraphMem(g2)
	if err != nil {
		t.Errorf("failed to generate phantom fig4: %s", err)
	}

	var r = "R"

	rChain := g2.getKChain(r)
	expected = []string{"R", "O", "K"}
	if !reflect.DeepEqual(rChain.chain.Elements(), expected) {
		t.Errorf("wrong kChain for %s; got %v, want %v", r, rChain.chain.Elements(), expected)
	}

	if rChain.minHeight != 4 {
		t.Errorf("wrong minHeight for chain of %s; got %d, want %d", r, rChain.minHeight, 4)
	}
}

func TestGreedyGraphMem_getAntiPast(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	err = genPhantomFig3GreedyGraphMem(g)
	if err != nil {
		t.Errorf("failed to generate phantom fig3: %s", err)
	}

	// Antipast for virtual node is the antiPast of the coloring tip of the graph (M)
	// So only J and L should not be present.
	antiPast := g.getAntiPast("")
	expected := []string{"C", "D", "E", "GENESIS", "K", "H", "F", "B", "I"}
	if !reflect.DeepEqual(antiPast.Elements(), expected) {
		t.Errorf("wrong antiPast for virtual node; got %v, want %v", antiPast.Elements(), expected)
	}

	h := "H"
	antiPast = g.getAntiPast(h)
	expected = []string{"J", "L", "M", "K", "F", "H", "B", "I"}
	if !reflect.DeepEqual(antiPast.Elements(), expected) {
		t.Errorf("wrong antiPast for %s; got %v, want %v", h, antiPast.Elements(), expected)
	}
}

func TestGreedyGraphMem_isABluerThanB(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	err = genPhantomFig3GreedyGraphMem(g)
	if err != nil {
		t.Errorf("failed to generate phantom fig3: %s", err)
	}

	ok := g.isABluerThanB("M", "H")
	if !ok {
		t.Errorf("wrong bluer-than answer")
	}

	ok = g.isABluerThanB("H", "J")
	if ok {
		t.Errorf("wrong bluer-than answer")
	}
}

func TestGreedyGraphMem_isMaxColoringTip(t *testing.T) {
	var k = 3
	g, err := NewGreedyGraphMem(k)
	if err != nil {
		t.Errorf("failed to create new GreedyGraphMem: %s", err)
	}

	err = genPhantomFig3GreedyGraphMem(g)
	if err != nil {
		t.Errorf("failed to generate phantom fig3: %s", err)
	}

	var h = "H"
	ok := g.isMaxColoringTip("H")
	if ok {
		t.Errorf("wrong isMaxColoringTip answer for %s; got %v, want %v", h, ok, false)
	}

	// M should no longer be max coloring tip
	var m = "M"
	_, _ = g.Add("Z", []string{m})
	ok = g.isMaxColoringTip(m)
	if ok {
		t.Errorf("wrong isMaxColoringTip answer for %s; got %v, want %v", m, ok, false)
	}
}