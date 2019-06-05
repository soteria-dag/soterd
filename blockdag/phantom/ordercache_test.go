// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package phantom

import (
	"math"
	"reflect"
	"testing"
)

func modInt32(x, y int32) int32 {
	return int32(math.Mod(float64(x), float64(y)))
}

func TestOrderCache(t *testing.T) {
	var oc = NewOrderCache()

	a := newNode("a")
	var order = []*Node{a}
	var tips = []*Node{a}

	// Check that we can't add to cache for low heights
	err := oc.Add(0, tips, order)
	if err == nil {
		t.Fatalf("Should have failed to add to orderCache when height is too low")
	}

	_, exists := oc.MaxHeight()
	if exists {
		t.Fatalf("orderCache should not have a cache entry yet")
	}

	// Check that it's ok to add to cache at minOrderCacheDistance or higher
	height := minOrderCacheDistance
	err = oc.Add(height, tips, order)
	if err != nil {
		t.Error(err.Error())
	}

	if oc.Size() != 1 {
		t.Fatalf("wrong orderCache size; got %d, want %d", oc.Size(), 1)
	}

	max, exists := oc.MaxHeight()
	if !exists {
		t.Fatalf("orderCache should have a cache entry")
	}

	if max != height {
		t.Fatalf("orderCache MaxHeight incorrect; got %d, want %d", max, height)
	}

	// Check cache retrieval
	//
	// oc.Get returns the first cache match ~below~ the given height,
	// so it should miss when there's only one entry and its at the same height
	// as our request.
	entry, hit := oc.Get(height)
	if hit {
		t.Fatalf("orderCache should miss for height %d", height)
	}

	newHeight := minOrderCacheDistance * int32(2)
	err = oc.Add(newHeight, tips, order)
	entry, hit = oc.Get(newHeight)
	if !hit {
		t.Fatalf("orderCache miss for height %d", newHeight)
	}

	if !reflect.DeepEqual(entry.Order, order) {
		t.Fatalf("orderCache hit entry order incorrect; got %v, want %v", GetIds(entry.Order), GetIds(order))
	}

	if !reflect.DeepEqual(entry.Tips, tips) {
		t.Fatalf("orderCache hit entry tips incorrect; got %v, want %v", GetIds(entry.Tips), GetIds(tips))
	}

	oc.Expire(oc.Size())
	if oc.Size() != 0 {
		t.Fatalf("orderCache expiry failed; got %d, want %d", oc.Size(), 0)
	}

	// Check that entries are only added at correct intervals
	offset := int32(1) // We start at height 1, since height 0 is genesis block
	count := minOrderCacheDistance * int32(3)
	desiredEntries := int((count + offset) - minOrderCacheDistance)
	for h := offset; h < count + offset; h++ {
		_ = oc.Add(h, tips, order)

		if modInt32(h, minOrderCacheDistance) == 0 {
			if _, ok := oc.CanAdd(h); ok {
				t.Fatalf("orderCache shouldn't say it's ok to add duplicate cache entries at the same height")
			}

			size := oc.Size()
			_ = oc.Add(h, tips, order)
			if oc.Size() > size {
				t.Fatalf("orderCache shouldn't allow duplicate cache entries at the same height; got %d, want %d", oc.Size(), size)
			}
		}
	}

	if oc.Size() != desiredEntries {
		t.Fatalf("wrong orderCache size for multiple additions; got %d, want %d", oc.Size(), desiredEntries)
	}

	max, exists = oc.MaxHeight()
	if max != count {
		t.Fatalf("orderCache MaxHeight incorrect for multiple additions; got %d, want %d", max, count)
	}

	expectedHeights := make([]int32, 0, desiredEntries)
	for i := minOrderCacheDistance; i < minOrderCacheDistance + int32(desiredEntries); i++ {
		expectedHeights = append(expectedHeights, i)
	}

	if !reflect.DeepEqual(oc.Heights(), expectedHeights) {
		t.Fatalf("orderCache Heights incorrect; got %v, want %v", oc.Heights(), expectedHeights)
	}

	// Check expiry for an amount larger than the cache
	oc.Expire(oc.Size() + 1)
	if oc.Size() != 0 {
		t.Fatalf("orderCache expiry failed when desired expiry count higher than orderCache size; got %d, want %d", oc.Size(), 0)
	}

	// Check maximum size of cache
	count = int32(maxOrderCacheSize) + (minOrderCacheDistance * 2) + int32(cacheFullExpireInterval + 1)
	for h := int32(0); h < count; h++ {
		_ = oc.Add(int32(h), tips, order)
	}

	if oc.Size() > maxOrderCacheSize {
		t.Fatalf("orderCache max size greater than limit; got %d, want %d", oc.Size(), maxOrderCacheSize)
	}
}