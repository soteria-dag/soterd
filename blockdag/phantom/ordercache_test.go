// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package phantom

import (
	"fmt"
	"reflect"
	"testing"
)

func TestOrderCache(t *testing.T) {
	oc := newOrderCache()

	genesis := newNode("genesis")
	a1 := newNode("a1")
	a1.parents[genesis] = keyExists
	a2 := newNode("a2")
	a2.parents[a1] = keyExists

	tips := []*node{a2}
	blueSet := newNodeSet()
	blueSet.add(a1)
	order := []*node{genesis, a1, a2}

	// Add the node info to the orderCache
	oc.add(tips, blueSet, order)

	// Check the orderCache contents
	if oc.size() != 1 {
		t.Errorf("orderCache size want %d, got %d", 1, oc.size())
	}

	if len(oc.tips) != 1 {
		t.Errorf("orderCache tips len want %d, got %d", 1, len(oc.tips))
	}

	if len(oc.blueSet) != 1 {
		t.Errorf("orderCache blueSet len want %d, got %d", 1, len(oc.blueSet))
	}

	if !reflect.DeepEqual(oc.blueSet[0], blueSet) {
		t.Errorf("orderCache blueSet want %v, got %v",
			getIds(blueSet.elements()), getIds(oc.blueSet[0].elements()))
	}

	if !reflect.DeepEqual(oc.order[0], order) {
		t.Errorf("orderCache order want %v, got %v", getIds(order), getIds(oc.order[0]))
	}

	// Check return values of other orderCache methods
	wantVirtual := "VIRTUAL->a2\na2->a1\na1->genesis\ngenesis\n"
	if oc.getOldestVirtual().PrintGraph() != wantVirtual {
		t.Errorf("orderCache getOldestVirtual want %s, got %s",
			wantVirtual, oc.getOldestVirtual().PrintGraph())
	}

	if !reflect.DeepEqual(oc.oldestBlueSet(), blueSet) {
		t.Errorf("orderCache oldestBlueSet want %v, got %v",
			getIds(blueSet.elements()), getIds(oc.oldestBlueSet().elements()))
	}

	a3 := newNode("a3")
	a3.parents[a2] = keyExists
	newTips := []*node{a3}
	newOrder := []*node{genesis, a1, a2, a3}

	// After adding the new tips and order, the oldestOrder and Tips should still be equal to the first addition
	oc.add(newTips, blueSet, newOrder)

	if !reflect.DeepEqual(oc.oldestOrder(), order) {
		t.Errorf("orderCache oldestOrder want %v, got %v", getIds(order), getIds(oc.oldestOrder()))
	}

	if !reflect.DeepEqual(oc.oldestTips(), tips) {
		t.Errorf("orderCache oldestTips want %v, got %v", getIds(tips), getIds(oc.oldestTips()))
	}

	// Check result of canUseCache as a sub-test
	t.Run("canUseCache", func(t *testing.T) {
		oc := newOrderCache()
		blueSet := newNodeSet()

		var nodes []*node
		var prevNode *node
		for i := 0; i < maxOrderCacheSize - 1; i++ {
			var name string
			if i == 0 {
				name = "genesis"
			} else {
				name = "n" + fmt.Sprintf("%d", i)
			}

			n := newNode(name)

			if prevNode != nil {
				n.parents[prevNode] = keyExists
			}

			nodes = append(nodes, n)
			prevNode = n

			oc.add([]*node{n}, blueSet, nodes)
		}

		if oc.canUseCache() {
			t.Error("orderCache canUseCache should return false when orderCache size is < maxOrderCacheSize")
		}

		n := newNode("final")
		n.parents[prevNode] = keyExists
		nodes = append(nodes, n)
		oc.add([]*node{n}, blueSet, nodes)

		if !oc.canUseCache() {
			t.Errorf("orderCache canUseCache should return true when orderCache size is >= maxOrderCacheSize")
		}

	})

	// Check pruning of orderCache as a sub-test
	t.Run("pruning", func(t *testing.T) {
		oc := newOrderCache()
		blueSet := newNodeSet()

		var nodes []*node
		var prevNode *node
		for i := 0; i < maxOrderCacheSize * 2; i++ {
			var name string
			if i == 0 {
				name = "genesis"
			} else {
				name = "n" + fmt.Sprintf("%d", i)
			}

			n := newNode(name)

			if prevNode != nil {
				n.parents[prevNode] = keyExists
			}

			nodes = append(nodes, n)
			prevNode = n

			oc.add([]*node{n}, blueSet, nodes)
		}

		if oc.size() > maxOrderCacheSize {
			t.Errorf("orderCache size should have been pruned to %d; got %d", maxOrderCacheSize, oc.size())
		}
	})
}