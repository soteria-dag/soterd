// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
//
// This file is ignored during the regular tests due to the following build tag.
// +build rpctest dag dagcoloring
// You can run tests from this file in isolation by using the build tags, like so:
// go test -v -count=1 -tags "dagcoloring" github.com/soteria-dag/soterd/integration

package integration

import (
	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/integration/rpctest"
	"testing"
)

// TestGetDAGColoring tests getcoloring RPC call
func TestGetDAGColoring(t *testing.T) {

	keepLogs := false

	// Set to debug or trace to produce more logging output from miners.
	extraArgs := []string{
		//"--debuglevel=debug",
	}

	miner, err := rpctest.New(&chaincfg.SimNetParams, nil, extraArgs, keepLogs)
	if err != nil {
		t.Fatalf("unable to create primary mining node: %v", err)
	}
	if err := miner.SetUp(false, 0); err != nil {
		t.Fatalf("unable to complete mining node setup: %v", err)
	}
	defer miner.TearDown()

	blockHashes, err := miner.Node.Generate(3)
	dagColoring, err := miner.Node.GetDAGColoring()

	for i, dagNode := range dagColoring {
		// skip genesis block
		if i == 0 {
			continue
		}
		hash := blockHashes[i-1]
		if dagNode.Hash != hash.String() {
			t.Fatalf("Order expecting hash %v at %d, got %v", hash, i, dagNode.Hash)
		}

		if !dagNode.IsBlue {
			t.Fatalf("Expecting block %v to be blue", dagNode.Hash)
		}

	}
}
