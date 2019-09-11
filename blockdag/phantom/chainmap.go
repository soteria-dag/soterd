// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package phantom

import (
	"container/list"
	"sort"
)

// OrderMap maps a string to an int
type OrderMap map[string]int

// Return true if the key exists in the OrderMap
func (om OrderMap) Contains(key string) bool {
	_, ok := om[key]
	return ok
}

// Return the value of the key in the OrderMap
func (om OrderMap) Get(key string) (int, bool) {
	order, ok := om[key]
	return order, ok
}

// max returns the highest order value in the OrderMap
func (om OrderMap) max() int {
	var highest int
	for _, v := range om {
		if v > highest {
			highest = v
		}
	}

	return highest
}

// Add the keys to the OrderMap, starting from the maximum order value + 1
func (om OrderMap) Add(keys ...string) {
	var offset = om.max() + 1
	for i, k := range keys {
		if !om.Contains(k) {
			order := offset + i
			om[k] = order
		}
	}
}

// Set the value of the key in the OrderMap
func (om OrderMap) Set(key string, value int) {
	om[key] = value
}

// Remove a key from the OrderMap
func (om OrderMap) Remove(key string) {
	delete(om, key)
}

// Clear removes all keys from the OrderMap
func (om OrderMap) Clear() {
	for k := range om {
		delete(om, k)
	}
}

// Order returns the keys of the OrderMap from lowest-value to highest
func (om OrderMap) Order() []string {
	var sorted = make([]string, len(om))
	index := 0
	for k := range om {
		sorted[index] = k
		index += 1
	}

	var less = func(i, j int) bool {
		var iNode = sorted[i]
		iOrder, _ := om[iNode]

		var jNode = sorted[j]
		jOrder, _ := om[jNode]

		if iOrder < jOrder {
			return true
		}

		if iOrder == jOrder && iNode < jNode {
			return true
		}

		return false
	}

	sort.Slice(sorted, less)

	return sorted
}

type ChainMap struct {
	// ChainMaps can contain map or ChainMap types, so we'll use List since it is type-agnostic;
	// slices can only contain values of one type, and you can't use interface{} as the type because your elements
	// will have to be interface{} types, and not _any_ type.
	members *list.List
}

// NewChainMap returns a new ChainMap
func NewChainMap(elements ...interface{}) *ChainMap {
	chainMap := ChainMap{
		members: list.New(),
	}

	for _, e := range elements {
		chainMap.members.PushBack(e)
	}

	return &chainMap
}

// Contains returns true when it finds that the key exists in the first matching map in the ChainMap.
// If the element is itself a chainmap, its members are parsed first.
func (cm *ChainMap) Contains(key string) bool {
	for e := cm.members.Front(); e != nil; e = e.Next() {
		switch e.Value.(type) {
		case ChainMap:
			chainMap := e.Value.(ChainMap)
			if chainMap.Contains(key) {
				return true
			}
		case OrderMap:
			orderMap := e.Value.(OrderMap)
			if orderMap.Contains(key) {
				return true
			}
		}
	}

	return false
}

// Get returns the first match from the sequence of members in the ChainMap.
// Returns true if a match was found.
func (cm *ChainMap) Get(key string) (int, bool) {
	for e := cm.members.Front(); e != nil; e = e.Next() {
		switch e.Value.(type) {
		case ChainMap:
			chainMap := e.Value.(ChainMap)
			order, ok := chainMap.Get(key)
			if ok {
				return order, ok
			}
		case OrderMap:
			orderMap := e.Value.(OrderMap)
			order, ok := orderMap.Get(key)
			if ok {
				return order, ok
			}
		}
	}

	return -1, false
}

// Keys returns an OrderedStringSet of keys from the ChainMap
func (cm *ChainMap) Keys() *OrderedStringSet {
	var keys = NewOrderedStringSet()

	for e := cm.members.Front(); e != nil; e = e.Next() {
		switch e.Value.(type) {
		case ChainMap:
			chainMap := e.Value.(ChainMap)

			for _, k := range chainMap.Keys().Elements() {
				keys.Add(k)
			}
		case OrderMap:
			orderMap := e.Value.(OrderMap)
			for _, k := range orderMap.Order() {
				keys.Add(k)
			}
		}
	}

	return keys
}

// Pop removes and returns the elements of the last entry of the ChainMap
func (cm *ChainMap) Pop() []string {
	e := cm.members.Back()
	if e == nil {
		// ChainMap is empty
		return []string{}
	}

	member := cm.members.Remove(e)
	switch member.(type) {
	case ChainMap:
		chainMap := e.Value.(ChainMap)
		return chainMap.Keys().Elements()
	case OrderMap:
		orderMap := e.Value.(OrderMap)
		return orderMap.Order()
	}

	return []string{}
}

// Remove removes an item from all OrderMaps in the ChainMap
func (cm *ChainMap) Remove(key string) {
	for e := cm.members.Front(); e != nil; e = e.Next() {
		switch e.Value.(type) {
		case ChainMap:
			chainMap := e.Value.(ChainMap)
			chainMap.Remove(key)
		case OrderMap:
			orderMap := e.Value.(OrderMap)
			orderMap.Remove(key)
		}
	}
}