// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package phantom

import (
	"fmt"
	"sort"
	"sync"
)

const (
	// The minimum dag height distance from the last orderCache entry that we'll create new entries at
	minOrderCacheDistance = int32(200)

	// The maximum size of the orderCache
	maxOrderCacheSize = int(minOrderCacheDistance) * 20
)

type OrderCacheEntry struct {
	Tips []*Node
	Order []*Node
}

type OrderCache struct {
	// Mapping of height of tips of cache entry, to the dag order for that height
	cache map[int32]*OrderCacheEntry

	// This is used to determine if we should trigger an expiry when the cache is full
	addsSinceLastExpire int

	// Methods use locks to ensure stable access to the cache between goroutines
	sync.RWMutex
}

// Define an interface we can use to sort the order cache height entries
type byHeight []int32
func (h byHeight) Len() int { return len(h) }
func (h byHeight) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h byHeight) Less(i, j int) bool { return h[i] < h[j] }

// NewOrderCache returns an orderCache, which can be used to help reduce repetitive computation as dag size increases
func NewOrderCache() *OrderCache {
	oc := OrderCache{
		cache: make(map[int32]*OrderCacheEntry),
	}

	return &oc
}

// add adds the entry to the cache
func (oc *OrderCache) add(height int32, tips, order []*Node) error {
	reason, ok := oc.canAdd(height)
	if !ok {
		return fmt.Errorf("failed to add to OrderCache, %s", reason)
	}

	size := oc.size()
	if size >= maxOrderCacheSize && oc.addsSinceLastExpire >= cacheFullExpireInterval {
		n := (size - maxOrderCacheSize) + 1
		log.Debugf("Expiring %d lowest orderCache items", n)
		oc.expire(n)

		oc.addsSinceLastExpire = 0
	}

	oc.cache[height] = &OrderCacheEntry{
		Tips: tips,
		Order: order,
	}

	oc.addsSinceLastExpire += 1

	return nil
}

// Add adds the entry to the cache
func (oc *OrderCache) Add(height int32, tips, order []*Node) error {
	oc.Lock()
	defer oc.Unlock()

	return oc.add(height, tips, order)
}

// canAdd returns true if it looks like it's ok to add an entry to the cache at the given height
func (oc *OrderCache) canAdd(height int32) (string, bool) {
	if height < minOrderCacheDistance {
		reason := fmt.Sprintf("height is too low; got %d, want >= %d", height, minOrderCacheDistance)
		return reason, false
	}

	if _, ok := oc.cache[height]; ok {
		// We don't allow duplicate cache entries
		reason := fmt.Sprintf("existing cache entry at height %d", height)
		return reason, false
	}

	return "", true
}

// canAdd returns true if it looks like it's ok to add an entry to the cache at the given height
func (oc *OrderCache) CanAdd(height int32) (string, bool) {
	oc.Lock()
	defer oc.Unlock()

	return oc.canAdd(height)
}

// expire removes the lowest n entries from the cache
func (oc *OrderCache) expire(amt int) {
	if amt <= 0 {
		return
	}

	heights := oc.heights()

	size := len(heights)
	if size < amt {
		amt = size
	}

	for i := 0; i < amt; i++ {
		delete(oc.cache, heights[i])
	}
}

// Expire removes the lowest n entries from the cache
func (oc *OrderCache) Expire(amt int) {
	oc.Lock()
	defer oc.Unlock()

	oc.expire(amt)
}

// get returns a cache entry appropriate for the height, and whether a matching cache entry was found
func (oc *OrderCache) get(height int32) (*OrderCacheEntry, bool) {
	if height < minOrderCacheDistance {
		return nil, false
	}

	target := height - minOrderCacheDistance

	// Check for a cache entry at the target height
	entry, exists := oc.cache[target]
	if exists {
		return entry, true
	}

	cacheHeights := oc.heights()
	// Try to find cache matches from highest to lowest-height
	sort.Sort(sort.Reverse(byHeight(cacheHeights)))

	for _, h := range cacheHeights {
		if h > target {
			continue
		}

		entry, exists := oc.cache[h]
		if !exists {
			continue
		}

		return entry, true
	}

	return nil, false
}

// Get returns a cache entry appropriate for the height, and whether a matching cache entry was found
func (oc *OrderCache) Get(height int32) (*OrderCacheEntry, bool) {
	oc.RLock()
	defer oc.RUnlock()

	return oc.get(height)
}

// heights returns a slice of cache heights, sorted from lowest to highest
func (oc *OrderCache) heights() []int32 {
	heights := make([]int32, oc.size())
	index := 0
	for h := range oc.cache {
		heights[index] = h
		index += 1
	}

	sort.Sort(byHeight(heights))

	return heights
}

// Heights returns a slice of cache heights, sorted from lowest to highest
func (oc *OrderCache) Heights() []int32 {
	oc.RLock()
	defer oc.RUnlock()

	return oc.heights()
}

// maxHeight returns the height of the highest cache entry, and if there are entries
func (oc *OrderCache) maxHeight() (int32, bool) {
	max := int32(-1)
	if oc.size() == 0 {
		return max, false
	}

	for h := range oc.cache {
		if h > max {
			max = h
		}
	}

	return max, true
}

// MaxHeight returns the height of the highest cache entry, and if there are entries
func (oc *OrderCache) MaxHeight() (int32, bool) {
	oc.RLock()
	defer oc.RUnlock()

	return oc.maxHeight()
}

// size returns the size of the orderCache
func (oc *OrderCache) size() int {
	return len(oc.cache)
}

// Size returns the size of the orderCache
func (oc *OrderCache) Size() int {
	oc.RLock()
	defer oc.RUnlock()

	return oc.size()
}