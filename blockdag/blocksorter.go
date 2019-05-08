// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdag

// blockSorter implements sort.Interface to allow a slice of blocks to
// be sorted by hash
type blockSorter []*blockNode

// Len returns the number of timestamps in the slice.  It is part of the
// sort.Interface implementation.
func (s blockSorter) Len() int {
	return len(s)
}

// Swap swaps the timestamps at the passed indices.  It is part of the
// sort.Interface implementation.
func (s blockSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Sorts blocks by hash string
func (s blockSorter) Less(i, j int) bool {
	return s[i].hash.String() < s[j].hash.String()
}