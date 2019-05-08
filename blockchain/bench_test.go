// Copyright (c) 2015 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"testing"

	"github.com/soteria-dag/soterd/soterutil"
)

// BenchmarkIsCoinBase performs a simple benchmark against the IsCoinBase
// function.
func BenchmarkIsCoinBase(b *testing.B) {
	tx, _ := soterutil.NewBlock(&Block100000).Tx(1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsCoinBase(tx)
	}
}

// BenchmarkIsCoinBaseTx performs a simple benchmark against the IsCoinBaseTx
// function.
func BenchmarkIsCoinBaseTx(b *testing.B) {
	tx := Block100000.Transactions[1]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsCoinBaseTx(tx)
	}
}
