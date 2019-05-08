// Copyright (c) 2013-2014 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package soterutil

const (
	// NanoSoter is a billionth of 1 Soter (SOTO)
	NanoSoter = 1e-9

	// NanoSoterPerSoter is the number of nanoSoter in one soter token (1 SOTO).
	NanoSoterPerSoter = 1e9

	// MaxNanoSoter is the maximum transaction amount allowed in nanoSoter.
	MaxNanoSoter = 21e5 * NanoSoterPerSoter
)
