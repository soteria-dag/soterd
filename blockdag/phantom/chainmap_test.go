// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package phantom

import (
	"reflect"
	"testing"
)

func TestOrderMap_Contains(t *testing.T) {
	om := make(OrderMap)
	var a = "A"
	var wantOrder = 1
	var nonExistent = "B"

	om[a] = wantOrder

	if !om.Contains(a) {
		t.Errorf("missing element %s", a)
	}

	if om.Contains(nonExistent) {
		t.Errorf("should not have element %s", nonExistent)
	}
}

func TestOrderMap_Get(t *testing.T) {
	om := make(OrderMap)
	var a = "A"
	var wantOrder = 1

	om[a] = wantOrder

	order, ok := om.Get(a)
	if !ok {
		t.Errorf("missing element %s", a)
	}

	if order != wantOrder {
		t.Errorf("wrong order for element %s; got %d, want %d", a, order, wantOrder)
	}

	var nonExistent = "B"
	order, ok = om.Get(nonExistent)
	if ok {
		t.Errorf("should not have element %s", nonExistent)
	}

	if order != 0 {
		t.Errorf("wrong order for non-existent element %s; got %d, want %d", nonExistent, order, 0)
	}
}

func TestOrderMap_Add(t *testing.T) {
	om := make(OrderMap)
	om["A"] = 4
	om.Add("B", "C")

	v, _ := om["B"]
	if v != 5 {
		t.Errorf("wrong order for B; got %d, want %d", v, 5)
	}
	v, _ = om["C"]
	if v != 6 {
		t.Errorf("wrong order for C; got %d, want %d", v, 6)
	}
}

func TestOrderMap_Set(t *testing.T) {
	om := make(OrderMap)

	var a = "A"
	var wantOrder = 7
	om.Set(a, wantOrder)

	got, ok := om[a]
	if !ok {
		t.Errorf("failed to set element %s", a)
	}

	if got != wantOrder {
		t.Errorf("wrong order set for %s; got %d, want %d", a, got, wantOrder)
	}
}

func TestOrderMap_Order(t *testing.T) {
	om := make(OrderMap)

	om.Set("C", 0)
	om.Set("A", 0)
	om.Set("B", 1)

	order := om.Order()
	expected := []string{"A", "C", "B"}
	if !reflect.DeepEqual(order, expected) {
		t.Errorf("wrong order; got %v, want %v", order, expected)
	}
}

func TestChainMap_Contains(t *testing.T) {
	om1 := make(OrderMap)
	om2 := make(OrderMap)

	cm := NewChainMap(om1, om2)

	var a = "A"
	var c = "C"

	om1["A"] = 0
	om2["B"] = 0

	if !cm.Contains(a) {
		t.Errorf("missing element %s", a)
	}

	if cm.Contains(c) {
		t.Errorf("should not have element %s", c)
	}

	// Add element c to an underlying OrderMap, and check that ChainMap can see it
	om2[c] = 2

	if !cm.Contains(c) {
		t.Errorf("missing element %s", c)
	}

	// Check that nested ChainMap lookups work
	cm2 := NewChainMap()
	cm3 := NewChainMap(*cm, *cm2)

	if !cm3.Contains(a) {
		t.Errorf("missing element %s", a)
	}

	var d = "D"
	cm2.members.PushBack(OrderMap{d: 7})

	if !cm3.Contains(d) {
		t.Errorf("missing element %s", d)
	}
}

func TestChainMap_Get(t *testing.T) {
	om1 := make(OrderMap)
	om2 := make(OrderMap)

	cm := NewChainMap(om1, om2)

	var a = "A"
	var b = "B"
	var nonExistent = "Z"

	om1["A"] = 0
	om2["A"] = 2
	om2["B"] = 3

	order, ok := cm.Get(a)
	if !ok {
		t.Errorf("missing element %s", a)
	}

	// om1 is first in the ChainMap, so it should match first, making the order 0, not 2
	if order != 0 {
		t.Errorf("wrong order for element %s; got %d, want %d", a, order, 0)
	}

	order, ok = cm.Get(b)
	if !ok {
		t.Errorf("missing element %s", b)
	}

	if order != 3 {
		t.Errorf("wrong order for element %s; got %d, want %d", b, order, 3)
	}

	order, ok = cm.Get(nonExistent)
	if ok {
		t.Errorf("should not have element %s", nonExistent)
	}

	if order != -1 {
		t.Errorf("wrong order for non-existent element %s; got %d, want %d", nonExistent, order, -1)
	}

	// Add an element to an underlying OrderMap, and check that we can retrieve its value
	var c = "C"
	om2[c] = 5
	order, ok = cm.Get(c)
	if !ok {
		t.Errorf("missing element %s", c)
	}

	if order != 5 {
		t.Errorf("wrong order for element %s; got %d, want %d", c, order, 5)
	}
}

func TestChainMap_Keys(t *testing.T) {
	om1 := make(OrderMap)
	om2 := make(OrderMap)
	om3 := make(OrderMap)

	cm := NewChainMap(om1, om2)
	cm2 := NewChainMap(om3, *cm)

	om1["A"] = 0
	om2["A"] = 2
	om2["B"] = 3
	om2["C"] = 5
	om3["C"] = 7

	keys := cm2.Keys()
	expected := []string{"C", "A", "B"}
	if !reflect.DeepEqual(keys.Elements(), expected) {
		t.Errorf("wrong keys for ChainMap; got %v, want %v", keys.Elements(), expected)
	}
}