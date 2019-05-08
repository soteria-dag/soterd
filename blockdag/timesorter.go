// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdag

import "time"

// timeSorter implements sort.Interface to allow a slice of timestamps to
// be sorted.
type timeSorter []int64

// Len returns the number of timestamps in the slice.  It is part of the
// sort.Interface implementation.
func (s timeSorter) Len() int {
	return len(s)
}

// Swap swaps the timestamps at the passed indices.  It is part of the
// sort.Interface implementation.
func (s timeSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less returns whether the timstamp with index i should sort before the
// timestamp with index j.  It is part of the sort.Interface implementation.
func (s timeSorter) Less(i, j int) bool {
	return s[i] < s[j]
}

// ByTime implements sort.Interface to allow a slice of time.Time types
// to be sorted.
type ByTime []time.Time

// Len returns the number of time.Time types in the slice. It's a part of
// the ByTime sort.Interface implementation.
func (t ByTime) Len() int { return len(t) }

// Swap swaps the time.Time types at the given slice indices. It's a part
// of the ByTime sort.Interface implementation.
func (t ByTime) Swap(i, j int) { t[i], t[j] = t[j], t[i] }

// Less returns true if the time.Time at index i should appear before the
// time.Time at index j in the sorted slice. It's a part of the ByTime
// sort.Interface implementation.
func (t ByTime) Less(i, j int) bool { return t[i].Before(t[j]) }
