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

func TestNodeListAt(t *testing.T) {
	nl := newNodeList(0)
	a := newNode("A")
	b := newNode("B")

	nl.push(a)
	nl.push(b)

	if nl.at(-1) != nil {
		t.Errorf("nodeList should return nil for a negative index")
	}

	if nl.at(nl.size() + 1) != nil {
		t.Errorf("nodeList should return nil for an out-of-bounds index")
	}

	if nl.at(0) != a {
		t.Errorf("wrong value at index %d; got %s, want %s", 0, nl.at(0).GetId(), a.GetId())
	}

	if nl.at(1) != b {
		t.Errorf("wrong value at index %d; got %s, want %s", 1, nl.at(1).GetId(), b.GetId())
	}
}

func TestNodeListDelete(t *testing.T) {
	nl := newNodeList(0)
	a := newNode("A")
	b := newNode("B")
	c := newNode("C")

	nl.push(a)
	nl.push(b)
	nl.push(c)

	n := nl.delete(1)

	if n != b {
		t.Errorf("wrong value deleted; got %s, want %s", n.GetId(), b.GetId())
	}

	if nl.size() != 2 {
		t.Errorf("size is wrong; got %d, want %d", nl.size(), 2)
	}

	expected := []*Node{a, c}
	if !reflect.DeepEqual(nl.elements, expected) {
		t.Errorf("contents wrong; got %v, want %v", GetIds(nl.elements), GetIds(expected))
	}
}

func TestNodeListInsert(t *testing.T) {
	nl := newNodeList(0)
	a := newNode("A")
	b := newNode("B")
	c := newNode("C")

	nl.push(a)
	nl.push(c)

	nl.insert(1, b)

	expected := []*Node{a, b, c}

	if !reflect.DeepEqual(nl.elements, expected) {
		t.Errorf("contents wrong; got %v, want %v", GetIds(nl.elements), GetIds(expected))
	}

	if nl.size() != 3 {
		t.Errorf("size is wrong; got %d, want %d", nl.size(), 3)
	}
}

func TestNodeListPush(t *testing.T) {
	nl := newNodeList(0)
	a := newNode("A")
	b := newNode("B")

	if nl.size() != 0 {
		t.Errorf("size is wrong; got %d, want %d", nl.size(), 0)
	}

	nl.push(a)

	if nl.size() != 1 {
		t.Errorf("size is wrong; got %d, want %d", nl.size(), 1)
	}

	nl.push(b)

	if nl.size() != 2 {
		t.Errorf("size is wrong; got %d, want %d", nl.size(), 2)
	}

	nl.push(nil)

	if nl.size() != 3 {
		t.Errorf("size is wrong; got %d, want %d", nl.size(), 3)
	}

	expected := []*Node{a, b, nil}
	if !reflect.DeepEqual(nl.elements, expected) {
		t.Errorf("contents wrong; got %v, want %v", GetIds(nl.elements), GetIds(expected))
	}
}

func TestNodeListPop(t *testing.T) {
	nl := newNodeList(0)
	a := newNode("A")
	b := newNode("B")

	nl.push(a)
	nl.push(b)

	n := nl.pop()

	if nl.size() != 1 {
		t.Errorf("size is wrong; got %d, want %d", nl.size(), 1)
	}

	if n != b {
		t.Errorf("wrong value popped; got %s, want %s", n.GetId(), b.GetId())
	}

	n = nl.pop()

	if nl.size() != 0 {
		t.Errorf("size is wrong; got %d, want %d", nl.size(), 0)
	}

	if n != a {
		t.Errorf("wrong value popped; got %s, want %s", n.GetId(), a.GetId())
	}

	n = nl.pop()

	if nl.size() != 0 {
		t.Errorf("size is wrong; got %d, want %d", nl.size(), 0)
	}

	if n != nil {
		t.Errorf("wrong value popped; got %s, want %v", n.GetId(), nil)
	}
}

func TestNodeListShift(t *testing.T) {
	nl := newNodeList(0)
	a := newNode("A")
	b := newNode("B")

	nl.push(a)
	nl.push(b)

	n := nl.shift()

	if nl.size() != 1 {
		t.Errorf("size is wrong; got %d, want %d", nl.size(), 1)
	}

	if n != a {
		t.Errorf("wrong value shifted; got %s, want %s", n.GetId(), a.GetId())
	}

	n = nl.shift()

	if nl.size() != 0 {
		t.Errorf("size is wrong; got %d, want %d", nl.size(), 0)
	}

	if n != b {
		t.Errorf("wrong value shifted; got %s, want %s", n.GetId(), b.GetId())
	}

	n = nl.shift()

	if nl.size() != 0 {
		t.Errorf("size is wrong; got %d, want %d", nl.size(), 0)
	}

	if n != nil {
		t.Errorf("wrong value shifted; got %s, want %v", n.GetId(), nil)
	}
}

func TestNodeListSize(t *testing.T) {
	nl := newNodeList(0)
	a := newNode("A")
	b := newNode("B")

	if nl.size() != 0 {
		t.Errorf("size is wrong; got %d, want %d", nl.size(), 0)
	}

	nl.push(a)

	if nl.size() != 1 {
		t.Errorf("size is wrong; got %d, want %d", nl.size(), 1)
	}

	nl.push(b)

	if nl.size() != 2 {
		t.Errorf("size is wrong; got %d, want %d", nl.size(), 2)
	}
}

func TestNodeListUnshift(t *testing.T) {
	nl := newNodeList(0)
	a := newNode("A")
	b := newNode("B")

	nl.push(b)
	nl.unshift(a)

	if nl.size() != 2 {
		t.Errorf("size is wrong; got %d, want %d", nl.size(), 2)
	}

	expected := []*Node{a, b}

	if !reflect.DeepEqual(nl.elements, expected) {
		t.Errorf("contents wrong; got %v, want %v", GetIds(nl.elements), GetIds(expected))
	}
}

func TestStringSet_Add(t *testing.T) {
	ss := NewStringSet()
	a := "A"
	ss.Add(a)

	if _, ok := ss.members[a]; !ok {
		t.Errorf("failed to add %s to StringSet", a)
	}
}

func TestStringSet_Remove(t *testing.T) {
	ss := NewStringSet()
	a := "A"
	ss.Add(a)
	ss.Remove(a)

	if _, ok := ss.members[a]; ok {
		t.Errorf("failed to remove %s to StringSet", a)
	}
}

func TestStringSet_Size(t *testing.T) {
	ss := NewStringSet()

	if ss.Size() != 0 {
		t.Errorf("wrong initial StringSet size; got %d, want %d", ss.Size(), 0)
	}

	a := "A"
	b := "B"

	ss.Add(a)

	if ss.Size() != 1 {
		t.Errorf("wrong StringSet size; got %d, want %d", ss.Size(), 1)
	}

	ss.Add(b)

	if ss.Size() != 2 {
		t.Errorf("wrong StringSet size; got %d, want %d", ss.Size(), 2)
	}

	ss.Remove(a)

	if ss.Size() != 1 {
		t.Errorf("wrong StringSet size; got %d, want %d", ss.Size(), 1)
	}
}

func TestStringSet_Contains(t *testing.T) {
	ss := NewStringSet()
	a := "A"

	if ss.Contains(a) {
		t.Errorf("StringSet shouldn't contain %s", a)
	}

	ss.Add(a)

	if !ss.Contains(a) {
		t.Errorf("StringSet should contain %s", a)
	}

	ss.Remove(a)

	if ss.Contains(a) {
		t.Errorf("StringSet shouldn't contain %s", a)
	}
}

func TestStringSet_Elements(t *testing.T) {
	ss := NewStringSet()
	a := "A"
	b := "B"
	c := "C"

	expected := []string{}
	if !reflect.DeepEqual(ss.Elements(), expected) {
		t.Errorf("wrong elements; got %v, expected %v", ss.Elements(), expected)
	}

	ss.Add(c)
	ss.Add(a)
	ss.Add(b)

	// We expect that elements are ordered alphabetically
	expected = []string{"A", "B", "C"}
	if !reflect.DeepEqual(ss.Elements(), expected) {
		t.Errorf("wrong elements; got %v, expected %v", ss.Elements(), expected)
	}
}

func TestStringSet_Difference(t *testing.T) {
	var ss1 = NewStringSet()
	var ss2 = NewStringSet()

	var a = "A"
	var b = "B"

	ss1.Add(a)
	ss2.Add(a)

	var expected = []string{}
	var diffSet = ss1.Difference(ss2)

	if !reflect.DeepEqual(diffSet.Elements(), expected) {
		t.Errorf("wrong difference of sets; got %v, want %v", diffSet.Elements(), expected)
	}

	ss2.Add(b)

	diffSet = ss1.Difference(ss2)
	if !reflect.DeepEqual(diffSet.Elements(), expected) {
		t.Errorf("wrong difference of sets; got %v, want %v", diffSet.Elements(), expected)
	}

	expected = []string{b}
	diffSet = ss2.Difference(ss1)
	if !reflect.DeepEqual(diffSet.Elements(), expected) {
		t.Errorf("wrong difference of sets; got %v, want %v", diffSet.Elements(), expected)
	}
}

func TestStringSet_Intersection(t *testing.T) {
	var ss1 = NewStringSet()
	var ss2 = NewStringSet()

	var a = "A"
	var b = "B"

	ss1.Add(a)
	ss2.Add(b)

	var expected = []string{}
	var interSet = ss1.Intersection(ss2)

	if !reflect.DeepEqual(interSet.Elements(), expected) {
		t.Errorf("wrong intersection of sets; got %v, want %v", interSet.Elements(), expected)
	}

	ss2.Add(a)

	expected = []string{a}
	interSet = ss1.Intersection(ss2)

	if !reflect.DeepEqual(interSet.Elements(), expected) {
		t.Errorf("wrong intersection of sets; got %v, want %v", interSet.Elements(), expected)
	}

	interSet = ss2.Intersection(ss1)

	if !reflect.DeepEqual(interSet.Elements(), expected) {
		t.Errorf("wrong intersection of sets; got %v, want %v", interSet.Elements(), expected)
	}
}

func TestOrderedStringSet_Add(t *testing.T) {
	ss := NewOrderedStringSet()
	a := "A"
	ss.Add(a)

	if ss.counter != 1 {
		t.Errorf("wrong counter value; got %d, want %d", ss.counter, 1)
	}

	order, ok := ss.members[a]
	if !ok {
		t.Errorf("failed to add %s to set", a)
	}
	if order != 1 {
		t.Errorf("wrong set order for %s; got %d, want %d", a, order, 1)
	}

	value, ok := ss.order[1]
	if !ok {
		t.Errorf("missing set order for %s in set", a)
	}

	if value != a {
		t.Errorf("wrong set order value for member %s; got %s, want %s", a, value, a)
	}

	// A second add shouldn't affect the set
	ss.Add(a)

	if ss.counter != 1 {
		t.Errorf("wrong counter value; got %d, want %d", ss.counter, 1)
	}
}

func TestOrderedStringSet_Clone(t *testing.T) {
	ss := NewOrderedStringSet()
	a := "A"
	b := "B"
	c := "C"
	d := "D"
	e := "E"
	ss.Add(c)
	ss.Add(a)
	ss.Add(b)

	ss2 := ss.Clone()

	if !reflect.DeepEqual(ss.Elements(), ss2.Elements()) {
		t.Errorf("wrong set members for clone; got %v, want %v", ss2.Elements(), ss.Elements())
	}

	// Modifying the original set shouldn't affect the clone
	ss.Add(d)

	if ss2.Contains(d) {
		t.Errorf("set clone should not be affected by changes to the original set")
	}

	// Modifying the clone shouldn't affect the original
	ss2.Add(e)

	if ss.Contains(e) {
		t.Errorf("original should not be affected by changes to clone")
	}
}

func TestOrderedStringSet_Contains(t *testing.T) {
	ss := NewOrderedStringSet()
	a := "A"

	if ss.Contains(a) {
		t.Errorf("set should not contain %s", a)
	}

	ss.Add(a)

	if !ss.Contains(a) {
		t.Errorf("set should contain %s", a)
	}
}

func TestOrderedStringSet_Difference(t *testing.T) {
	ss := NewOrderedStringSet()
	ss2 := NewOrderedStringSet()

	a := "A"

	var expected = []string{}
	var diff = ss.Difference(ss2)

	if !reflect.DeepEqual(diff.Elements(), expected) {
		t.Errorf("wrong difference of sets; got %v, want %v", diff.Elements(), expected)
	}

	ss.Add(a)
	expected = []string{a}
	diff = ss.Difference(ss2)
	if !reflect.DeepEqual(diff.Elements(), expected) {
		t.Errorf("wrong difference of sets; got %v, want %v", diff.Elements(), expected)
	}

	expected = []string{}
	diff = ss2.Difference(ss)
	if !reflect.DeepEqual(diff.Elements(), expected) {
		t.Errorf("wrong difference of sets; got %v, want %v", diff.Elements(), expected)
	}

	ss2.Add(a)
	diff = ss2.Difference(ss)
	if !reflect.DeepEqual(diff.Elements(), expected) {
		t.Errorf("wrong difference of sets; got %v, want %v", diff.Elements(), expected)
	}
}

func TestOrderedStringSet_Elements(t *testing.T) {
	ss := NewOrderedStringSet()
	a := "A"
	b := "B"
	c := "C"

	expected := []string{}
	if !reflect.DeepEqual(ss.Elements(), expected) {
		t.Errorf("wrong set members; got %v, want %v", ss.Elements(), expected)
	}

	ss.Add(c)
	ss.Add(a)
	ss.Add(b)

	expected = []string{c, a, b}
	if !reflect.DeepEqual(ss.Elements(), expected) {
		t.Errorf("wrong set members; got %v, want %v", ss.Elements(), expected)
	}
}

func TestOrderedStringSet_Intersection(t *testing.T) {
	ss := NewOrderedStringSet()
	ss2 := NewOrderedStringSet()

	a := "A"
	b := "B"

	var expected = []string{}
	var intersect = ss.Intersection(ss2)

	if !reflect.DeepEqual(intersect.Elements(), expected) {
		t.Errorf("wrong intersection of sets; got %v, want %v", intersect.Elements(), expected)
	}

	ss.Add(a)
	ss2.Add(b)

	intersect = ss.Intersection(ss2)
	if !reflect.DeepEqual(intersect.Elements(), expected) {
		t.Errorf("wrong intersection of sets; got %v, want %v", intersect.Elements(), expected)
	}

	ss2.Add(a)
	expected = []string{a}
	intersect = ss.Intersection(ss2)
	if !reflect.DeepEqual(intersect.Elements(), expected) {
		t.Errorf("wrong intersection of sets; got %v, want %v", intersect.Elements(), expected)
	}

	// Both sets should have the same intersection
	intersect = ss2.Intersection(ss)
	if !reflect.DeepEqual(intersect.Elements(), expected) {
		t.Errorf("wrong intersection of sets; got %v, want %v", intersect.Elements(), expected)
	}
}

func TestOrderedStringSet_Remove( t *testing.T) {
	ss := NewOrderedStringSet()
	a := "A"
	b := "B"
	c := "C"

	ss.Add(c)
	ss.Add(a)
	ss.Add(b)
	ss.Remove(a)

	if _, ok := ss.members[a]; ok {
		t.Errorf("failed to remove %s from set", a)
	}

	var expected = []string{c, b}
	if !reflect.DeepEqual(ss.Elements(), expected) {
		t.Errorf("wrong set members after removal of %s; got %v, want %v", a, ss.Elements(), expected)
	}

	ss.Add(a)
	expected = []string{c, b, a}
	if !reflect.DeepEqual(ss.Elements(), expected) {
		t.Errorf("wrong set order; got %v, want %v", ss.Elements(), expected)
	}
}

func TestOrderedStringSet_Size(t *testing.T) {
	ss := NewOrderedStringSet()
	a := "A"
	b := "B"
	c := "C"

	ss.Add(c)

	if ss.Size() != 1 {
		t.Errorf("wrong set size; got %d, want %d", ss.Size(), 1)
	}

	ss.Add(a)
	ss.Add(b)

	if ss.Size() != 3 {
		t.Errorf("wrong set size; got %d, want %d", ss.Size(), 3)
	}

	ss.Remove(a)

	if ss.Size() != 2 {
		t.Errorf("wrong set size; got %d, want %d", ss.Size(), 2)
	}

	ss.Add(a)

	if ss.Size() != 3 {
		t.Errorf("wrong set size; got %d, want %d", ss.Size(), 3)
	}
}

func TestOrderedStringSet_Subset(t *testing.T) {
	ss := NewOrderedStringSet()
	ss2 := NewOrderedStringSet()

	a := "A"
	b := "B"

	ss.Add(a)
	ss2.Add(b)

	if ss.Subset(ss2) {
		t.Errorf("set should not not a subset")
	}

	ss2.Add(a)

	if !ss.Subset(ss2) {
		t.Errorf("set should be a subset")
	}

	ss.Add(b)

	if !ss.Subset(ss2) {
		t.Errorf("set should be a subset")
	}
}

func TestOrderedStringSet_Union(t *testing.T) {
	ss := NewOrderedStringSet()
	ss2 := NewOrderedStringSet()

	a := "A"
	b := "B"

	ss.Add(a)
	ss2.Add(a)
	ss2.Add(b)

	union := ss.Union(ss2)

	expected := []string{a, b}
	if !reflect.DeepEqual(union.Elements(), expected) {
		t.Errorf("union set has wrong members; got %v, want %v", union.Elements(), expected)
	}
}

func TestStringList_At(t *testing.T) {
	sl := newStringList()
	a := "A"
	b := "B"

	sl.push(a)
	sl.push(b)

	_, ok := sl.at(-1)
	if ok {
		t.Errorf("at() should return nil for a negative index")
	}

	_, ok = sl.at(sl.size() + 1)
	if ok {
		t.Errorf("at() should return nil for an out-of-bounds index")
	}

	got, ok := sl.at(0)
	if !ok {
		t.Errorf("missing value at index %d", 0)
	}

	if got != a {
		t.Errorf("wrong value at index %d; got %s, want %s", 0, got, a)
	}

	got, ok = sl.at(1)
	if got != b {
		t.Errorf("wrong value at index %d; got %s, want %s", 1, got, b)
	}
}

func TestStringList_Delete(t *testing.T) {
	sl := newStringList()
	a := "A"
	b := "B"
	c := "C"

	sl.push(a)
	sl.push(b)
	sl.push(c)

	got, ok := sl.delete(1)

	if !ok {
		t.Errorf("delete() should return true index %d", 1)
	}

	if got != b {
		t.Errorf("wrong value deleted; got %s, want %s", got, b)
	}

	if sl.size() != 2 {
		t.Errorf("size is wrong; got %d, want %d", sl.size(), 2)
	}

	expected := []string{a, c}
	if !reflect.DeepEqual(sl.elements, expected) {
		t.Errorf("elements wrong; got %v, want %v", sl.elements, expected)
	}
}

func TestStringList_Insert(t *testing.T) {
	sl := newStringList()
	a := "A"
	b := "B"
	c := "C"

	sl.push(a)
	sl.push(c)

	sl.insert(1, b)

	expected := []string{a, b, c}

	if !reflect.DeepEqual(sl.elements, expected) {
		t.Errorf("elements wrong; got %v, want %v", sl.elements, expected)
	}

	if sl.size() != 3 {
		t.Errorf("size is wrong; got %d, want %d", sl.size(), 3)
	}
}

func TestStringList_Push(t *testing.T) {
	sl := newStringList()
	a := "A"
	b := "B"

	if sl.size() != 0 {
		t.Errorf("size is wrong; got %d, want %d", sl.size(), 0)
	}

	sl.push(a)

	if sl.size() != 1 {
		t.Errorf("size is wrong; got %d, want %d", sl.size(), 1)
	}

	sl.push(b)

	if sl.size() != 2 {
		t.Errorf("size is wrong; got %d, want %d", sl.size(), 2)
	}

	sl.push("")

	if sl.size() != 3 {
		t.Errorf("size is wrong; got %d, want %d", sl.size(), 3)
	}

	expected := []string{a, b, ""}
	if !reflect.DeepEqual(sl.elements, expected) {
		t.Errorf("elements wrong; got %v, want %v", sl.elements, expected)
	}
}

func TestStringList_Pop(t *testing.T) {
	sl := newStringList()
	a := "A"
	b := "B"

	sl.push(a)
	sl.push(b)

	got, ok := sl.pop()

	if !ok {
		t.Errorf("pop() should return true")
	}

	if sl.size() != 1 {
		t.Errorf("size is wrong; got %d, want %d", sl.size(), 1)
	}

	if got != b {
		t.Errorf("wrong value popped; got %s, want %s", got, b)
	}

	got, ok = sl.pop()

	if !ok {
		t.Errorf("pop() should return true")
	}

	if sl.size() != 0 {
		t.Errorf("size is wrong; got %d, want %d", sl.size(), 0)
	}

	if got != a {
		t.Errorf("wrong value popped; got %s, want %s", got, a)
	}

	got, ok = sl.pop()

	if ok {
		t.Errorf("pop() should return false")
	}

	if sl.size() != 0 {
		t.Errorf("size is wrong; got %d, want %d", sl.size(), 0)
	}

	if got != "" {
		t.Errorf("wrong value popped; got %s, want %v", got, "")
	}
}

func TestStringList_Shift(t *testing.T) {
	sl := newStringList()
	a := "A"
	b := "B"

	sl.push(a)
	sl.push(b)

	got, ok := sl.shift()

	if !ok {
		t.Errorf("shift() should return true")
	}

	if sl.size() != 1 {
		t.Errorf("size is wrong; got %d, want %d", sl.size(), 1)
	}

	if got != a {
		t.Errorf("wrong value shifted; got %s, want %s", got, a)
	}

	got, ok = sl.shift()

	if !ok {
		t.Errorf("shift() should return true")
	}

	if sl.size() != 0 {
		t.Errorf("size is wrong; got %d, want %d", sl.size(), 0)
	}

	if got != b {
		t.Errorf("wrong value shifted; got %s, want %s", got, b)
	}

	got, ok = sl.shift()

	if ok {
		t.Errorf("shift() should return false")
	}

	if sl.size() != 0 {
		t.Errorf("size is wrong; got %d, want %d", sl.size(), 0)
	}

	if got != "" {
		t.Errorf("wrong value shifted; got %s, want %v", got, "")
	}
}

func TestStringList_Size(t *testing.T) {
	sl := newStringList()
	a := "A"
	b := "B"

	if sl.size() != 0 {
		t.Errorf("size is wrong; got %d, want %d", sl.size(), 0)
	}

	sl.push(a)

	if sl.size() != 1 {
		t.Errorf("size is wrong; got %d, want %d", sl.size(), 1)
	}

	sl.push(b)

	if sl.size() != 2 {
		t.Errorf("size is wrong; got %d, want %d", sl.size(), 2)
	}
}

func TestStringList_Unshift(t *testing.T) {
	sl := newStringList()
	a := "A"
	b := "B"

	sl.push(b)
	sl.unshift(a)

	if sl.size() != 2 {
		t.Errorf("size is wrong; got %d, want %d", sl.size(), 2)
	}

	expected := []string{a, b}
	if !reflect.DeepEqual(sl.elements, expected) {
		t.Errorf("elements wrong; got %v, want %v", sl.elements, expected)
	}
}