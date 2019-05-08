// Copyright (c) 2014-2016 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	// maxCachedMetrics is the maximum number of entries each *Cache field of MetricsManager will hold.
	// Once the maximum is reached, older entries are dropped from the cache.
	maxCachedMetrics = 5000
)

// Config is a configuration struct used to initialize a new MetricsManager.
type Config struct {
	// Provide the metrics manager with the channels it will listen to for new metrics
	MinerSolveCount *chan struct{}
	MinerSolveHashes *chan string
	MinerSolveTimes *chan time.Duration
}

// MetricsManager is used to collect metrics from other managers in code, and make the data
// available to other systems.
type MetricsManager struct {
	// Help Start and Stop methods to determine if manager has already been started/stopped
	started        int32
	shutdown       int32

	// Miner block solve count metric sent here
	minerSolveCount		*chan struct{}
	solveCount int64
	solveCountLock sync.RWMutex

	// Miner solved-block hashes are sent here
	minerSolveHashes	*chan string
	minerSolveHashesCache []string
	minerSolveHashesLock sync.RWMutex

	// Miner block solve time metrics are sent here
	minerSolveTimes		*chan time.Duration
	// Block solve times are cached here, so that other systems can query the data (RPC Calls)
	minerSolveTimesCache []*time.Duration
	// A lock to prevent multiple updates to cache at the same time
	minerSolveTimesLock sync.RWMutex

	// Helps wait for goroutines to finish before continuing
	wg sync.WaitGroup

	// Listens on quit for message to stop processing metrics
	quit	chan struct{}
}

// cacheSolveHash caches the given hash string in the in-memory cache. It doesn't have a maximum size, so this metric is
// only sent to the metrics manager when the soterd network this node is running under is wire.SimNet.
func (mm *MetricsManager) cacheSolveHash(h string) {
	mm.minerSolveHashesLock.Lock()
	defer mm.minerSolveHashesLock.Unlock()

	mm.minerSolveHashesCache = append(mm.minerSolveHashesCache, h)
}

// cacheSolveTime caches the given solve time in the in-memory cache. If we've reached the maximum number of cached
// times, we'll remove the oldest values.
func (mm *MetricsManager) cacheSolveTime(s time.Duration) {
	mm.minerSolveTimesLock.Lock()
	defer mm.minerSolveTimesLock.Unlock()

	mm.minerSolveTimesCache = append(mm.minerSolveTimesCache, &s)

	// Truncate cache to the limit
	for (len(mm.minerSolveTimesCache)) > maxCachedMetrics {
		mm.minerSolveTimesCache = mm.minerSolveTimesCache[1:]
	}
}

// incSolveCount increments the counter for solving blocks each time it's called
func (mm *MetricsManager) incSolveCount() {
	mm.solveCountLock.Lock()
	defer mm.solveCountLock.Unlock()

	mm.solveCount++
}

// Reads from metrics message channels, and handles metrics accordingly
func (mm *MetricsManager) metricsHandler() {
	defer mm.wg.Done()

	for {
		select {
			case <-*mm.minerSolveCount:
				mm.incSolveCount()
			case h := <-*mm.minerSolveHashes:
				mm.cacheSolveHash(h)
			case m := <-*mm.minerSolveTimes:
				mm.cacheSolveTime(m)
			case <-mm.quit:
				return
		}
	}
}

// New returns a new MetricsManager based on the given config.
// Use Start() to start processing metrics.
func New(config *Config) (*MetricsManager, error) {
	mm := MetricsManager{
		minerSolveCount: config.MinerSolveCount,
		minerSolveHashes: config.MinerSolveHashes,
		minerSolveTimes: config.MinerSolveTimes,
		quit: make(chan struct{}),
	}

	return &mm, nil
}

// Starts processing metrics in a new goroutine
func (mm *MetricsManager) Start() {
	// Already started?
	if atomic.AddInt32(&mm.started, 1) != 1 {
		return
	}

	log.Info("Starting metrics manager")
	mm.wg.Add(1)
	go mm.metricsHandler()
}

// Stop ends processing of metrics
func (mm *MetricsManager) Stop() {
	if atomic.AddInt32(&mm.shutdown, 1) != 1 {
		log.Warn("Metrics manager is already in the process of stopping")
		return
	}

	log.Info("Metrics manager stopping")
	close(mm.quit)
	mm.wg.Wait()
	log.Info("Metrics manager stopped")
}

// MinerSolveCount returns the number of solved blocks by the miner(s)
func (mm *MetricsManager) MinerSolveCount() int64 {
	mm.solveCountLock.RLock()
	defer mm.solveCountLock.RUnlock()

	return mm.solveCount
}

// MinerSolveHashes returns the solved-block hash cache
func (mm *MetricsManager) MinerSolveHashes() []string {
	mm.minerSolveHashesLock.RLock()
	defer mm.minerSolveHashesLock.RUnlock()

	return mm.minerSolveHashesCache
}

// MinerSolveTimes returns the solve times cache
func (mm *MetricsManager) MinerSolveTimes() []time.Duration {
	mm.minerSolveTimesLock.RLock()
	defer mm.minerSolveTimesLock.RUnlock()

	var solveTimes []time.Duration
	for _, s := range mm.minerSolveTimesCache {
		solveTimes = append(solveTimes, *s)
	}

	return solveTimes
}