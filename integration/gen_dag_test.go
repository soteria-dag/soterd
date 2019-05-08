// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
//
// This file is ignored during the regular tests due to the following build tag.
// +build rpctest dag daggen
// You can run tests from this file in isolation by using the build tags, like so:
// go test -v -count=1 -tags "daggen" github.com/soteria-dag/soterd/integration

package integration

import (
	"testing"
	"time"

	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/integration/rpctest"
	"github.com/soteria-dag/soterd/rpcclient"
)

// countDAGBlocks returns the number of blocks with multiple parents (indicating DAG structure) found on a node
func countDAGBlocks(node *rpctest.Harness) (int, error) {
	count := 0

	tips, err := node.Node.GetDAGTips()
	if err != nil {
		return count, err
	}

	for height := int32(0); height <= tips.MaxHeight; height++ {
		hashes, err := node.Node.GetBlockHash(int64(height))
		if err != nil {
			return count, err
		}

		for _, hash := range hashes {
			block, err := node.Node.GetBlock(hash)
			if err != nil {
				return count, err
			}

			if (*block).Parents.Size > 1 {
				count++
			}
		}
	}

	return count, nil
}

// TestDAGGen attempts to generate a DAG by running a large number of miners simultaneously
func TestDAGGen(t *testing.T) {
	var miners []*rpctest.Harness

	// NOTE(cedric): The minerCount := 4 and blockCount := 100 settings below result in reliable DAG generation on an
	// 8-core Intel i7-4790 CPU @ 3.60GHz

	// Number of miners to spawn
	minerCount := 4
	// Number of blocks to generate on each miner
	blockCount := 15
	// Max time to wait for blocks to sync between nodes, once all are generated
	wait := time.Minute

	// Set to true, to retain logs from miners
	keepLogs := false
	// Set to debug or trace to produce more logging output from miners.
	extraArgs := []string{
		//"--debuglevel=debug",
	}

	for i := 0; i < minerCount; i++ {
		miner, err := rpctest.New(&chaincfg.SimNetParams, nil, extraArgs, keepLogs)
		if err != nil {
			t.Fatalf("unable to create mining node %v: %v", i, err)
		}

		if err := miner.SetUp(false, 0); err != nil {
			t.Fatalf("unable to complete mining node %v setup: %v", i, err)
		}

		if keepLogs {
			t.Logf("miner %d log dir: %s", i, miner.LogDir())
		}

		miners = append(miners, miner)
	}
	// NOTE(cedric): We'll call defer on a single anonymous function instead of minerCount times in the above loop
	defer func() {
		for _, miner := range miners {
			_ = (*miner).TearDown()
		}
	}()

	// Connect the nodes to one another
	err := rpctest.ConnectNodes(miners)
	if err != nil {
		t.Fatalf("unable to connect nodes: %v", err)
	}

	// Generate blocks on each miner.
	// We'll know DAG was created when blocks with multiple parents were generated
	var futures []*rpcclient.FutureGenerateResult
	for _, miner := range miners {
		future := miner.Node.GenerateAsync(uint32(blockCount))
		futures = append(futures, &future)
	}

	// Wait for block generation to finish
	for i, future := range futures {
		_, err := (*future).Receive()
		if err != nil {
			t.Fatalf("failed to wait for blocks to generate on node %v: %v", i, err)
		}
	}

	t.Log("waiting for blocks to sync between nodes")
	_ = rpctest.WaitForDAG(miners, wait)

	// Check nodes for blocks with multiple parents. This means we accomplished generating a DAG
	dagBlockCount := 0
	for i, miner := range miners {
		count, err := countDAGBlocks(miner)
		if err != nil {
			t.Fatalf("failed to count DAG blocks on miner %v: %v", i, err)
		}

		t.Logf("node %v\tfound %v blocks with multiple parents", i, count)
		dagBlockCount += count
	}

	if dagBlockCount == 0 {
		t.Fatalf("failed to generate DAG blocks!")
	}

	// Confirm that the DAG on all miners is identical
	err = rpctest.CompareDAG(miners)
	if err != nil {
		for i, miner := range miners {
			dErr := rpctest.DumpDAG(i, miner)
			if dErr != nil {
				t.Logf("failed to dump dag for miner %v: %v", i, dErr)
			}
		}
		t.Fatalf(err.Error())
	}
}