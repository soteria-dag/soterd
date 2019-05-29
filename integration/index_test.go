// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
//
// This file is ignored during the regular tests due to the following build tag.
// +build rpctest indexer
// You can run tests from this file in isolation by using the build tags, like so:
// go test -v -count=1 -tags "indexer" github.com/soteria-dag/soterd/integration

package integration

import (
	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/chaincfg/chainhash"
	"github.com/soteria-dag/soterd/integration/rpctest"
	"github.com/soteria-dag/soterd/soterjson"
	"github.com/soteria-dag/soterd/soterutil"
	"github.com/soteria-dag/soterd/wire"
	"testing"
)

func TestTxIndexer(t *testing.T) {

	keepLogs := true

	// Set to debug or trace to produce more logging output from miners.
	extraArgs := []string{
		"--txindex",
		//"--addrindex",
		"--nocfilters",
		"--debuglevel=debug",
	}

	miner, err := rpctest.New(&chaincfg.SimNetParams, nil, extraArgs, keepLogs)
	if err != nil {
		t.Fatalf("unable to create primary mining node: %v", err)
	}
	if err := miner.SetUp(false, 0); err != nil {
		t.Fatalf("unable to complete mining node setup: %v", err)
	}

	// mine some blocks
	blockHashes, err := miner.Node.Generate(5)
	if err != nil {
		t.Fatalf("error generating blocks: %v", err)
	}

	blockHash := blockHashes[0]
	block, err := miner.Node.GetBlock(blockHash)
	if err != nil {
		t.Fatalf("unable to get block: %v", err)
	}
	txHashes, err := block.TxHashes()

	// get raw transaction
	tx, err := miner.Node.GetRawTransactionVerbose(&txHashes[0])
	if err != nil {
		t.Fatalf("error calling GetRawTransactionVerbose: %v", err)
	}
	if tx == nil {
		t.Fatalf("expected tx with hash: %v", txHashes[0])
	}

	defer miner.TearDown()
}

func TestAddrIndexer(t *testing.T) {
	keepLogs := true

	// Set to debug or trace to produce more logging output from miners.
	extraArgs := []string{
		"--addrindex",
		"--nocfilters",
		"--debuglevel=debug",
	}

	miner, err := rpctest.New(&chaincfg.SimNetParams, nil, extraArgs, keepLogs)
	if err != nil {
		t.Fatalf("unable to create primary mining node: %v", err)
	}
	if err := miner.SetUp(false, 0); err != nil {
		t.Fatalf("unable to complete mining node setup: %v", err)
	}

	// mine some blocks
	blockHashes, err := miner.Node.Generate(5)
	if err != nil {
		t.Fatalf("error generating blocks: %v", err)
	}

	blockHash := blockHashes[0]
	block, err := miner.Node.GetBlock(blockHash)
	if err != nil {
		t.Fatalf("unable to get block: %v", err)
	}
	txHashes, err := block.TxHashes()

	// get raw transaction
	tx, err := miner.Node.GetRawTransactionVerbose(&txHashes[0])
	if err != nil {
		t.Fatalf("error calling GetRawTransactionVerbose: %v", err)
	}
	if tx == nil {
		t.Fatalf("expected tx with hash: %v", txHashes[0])
	}

	minerAddress := tx.Vout[0].ScriptPubKey.Addresses[0]

	address, err := soterutil.DecodeAddress(minerAddress, &chaincfg.SimNetParams)
	if err != nil {
		t.Fatalf("unable to convert string to address: %v", err)
	}

	// search raw transaction
	res, err := miner.Node.SearchRawTransactionsVerbose(address,0, 10,
		true, false, []string{})
	if err != nil {
		t.Fatalf("error calling SearchRawTransactionsVerbose: %v", err)
	}
	if len(res) != 5 {
		t.Fatalf("expecting %d transactions, got %d", 5, len(res))
	}

}

func TestCFilterIndexer(t *testing.T) {
	keepLogs := true

	// Set to debug or trace to produce more logging output from miners.
	extraArgs := []string{
		//"--nocfilters",
		"--debuglevel=debug",
	}

	miner, err := rpctest.New(&chaincfg.SimNetParams, nil, extraArgs, keepLogs)
	if err != nil {
		t.Fatalf("unable to create primary mining node: %v", err)
	}
	if err := miner.SetUp(false, 0); err != nil {
		t.Fatalf("unable to complete mining node setup: %v", err)
	}

	// mine some blocks
	blockHashes, err := miner.Node.Generate(5)
	if err != nil {
		t.Fatalf("error generating blocks: %v", err)
	}

	blockHash := blockHashes[0]

	// get cfilter
	cfilter, err := miner.Node.GetCFilter(blockHash, wire.GCSFilterRegular)
	if err != nil {
		t.Fatalf("error calling GetCFilter: %v", err)
	}
	if cfilter == nil {
		t.Fatalf("expecting cfilter returned for block")
	}
	cfilter, err = miner.Node.GetCFilter(&chainhash.Hash{}, wire.GCSFilterRegular)
	if err != nil {
		t.Fatalf("error calling GetCFilter: %v", err)
	}
	if cfilter != nil && len(cfilter.Data) > 0 {
		t.Fatalf("no cfilter expected for zerohash: %v", cfilter)
	}
}

// Start with indexes disabled, mine some blocks, then restart with indexer on,
// they should catch up
func TestIndexerCatchUp(t *testing.T) {
	keepLogs := true

	// Set to debug or trace to produce more logging output from miners.
	extraArgs := []string{
		//"--txindex",
		"--nocfilters",
		"--debuglevel=debug",
	}

	miner, err := rpctest.New(&chaincfg.SimNetParams, nil, extraArgs, keepLogs)
	if err != nil {
		t.Fatalf("unable to create primary mining node: %v", err)
	}
	if err := miner.SetUp(false, 0); err != nil {
		t.Fatalf("unable to complete mining node setup: %v", err)
	}

	// mine some blocks
	blockHashes, err := miner.Node.Generate(5)
	if err != nil {
		t.Fatalf("error generating blocks: %v", err)
	}

	blockHash := blockHashes[0]
	block, err := miner.Node.GetBlock(blockHash)
	if err != nil {
		t.Fatalf("unable to get block: %v", err)
	}
	txHashes, err := block.TxHashes()

	// get raw transaction
	_, err = miner.Node.GetRawTransactionVerbose(&txHashes[0])
	if err == nil {
		t.Fatalf("Expecting error calling GetRawTransactionVerbose, txindex disabled")
	}

	err = miner.Restart([]string{"--txindex", "--addrindex"}, []string{"--nocfilters"})
	if err != nil {
		t.Fatalf("unable to restart node: %v", err)
	}

	tx, err := miner.Node.GetRawTransactionVerbose(&txHashes[0])
	if err != nil {
		t.Fatalf("Error calling GetRawTransactionVerbose: %v", err)
	}

	var jsontx *soterjson.TxRawResult = tx
	if  tx != nil && jsontx.Hash != txHashes[0].String() {
		t.Fatalf("expected tx with hash: %v", txHashes[0])
	}

	// get cfilter
	/*
	cfilter, err := miner.Node.GetCFilter(blockHash, wire.GCSFilterRegular)
	if err != nil {
		t.Fatalf("error calling GetCFilter: %v", err)
	}
	if cfilter == nil {
		t.Fatalf("expecting cfilter returned for block")
	}*/
}