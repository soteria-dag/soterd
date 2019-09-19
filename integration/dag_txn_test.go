// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
//
// This file is ignored during the regular tests due to the following build tag.
// +build rpctest dag dagtxn
// You can run tests from this file in isolation by using the build tags, like so:
// go test -v -count=1 -tags "dagtxn" github.com/soteria-dag/soterd/integration

package integration


import (
	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/chaincfg/chainhash"
	"github.com/soteria-dag/soterd/integration/rpctest"
	"github.com/soteria-dag/soterd/rpcclient"
	"github.com/soteria-dag/soterd/soterutil"
	"github.com/soteria-dag/soterd/txscript"
	"github.com/soteria-dag/soterd/wire"
	"testing"
	"time"
)

func spendCoinbase(r *rpctest.Harness, t *testing.T, amt soterutil.Amount) (*chainhash.Hash, error) {
	// Grab a fresh address from the wallet.
	addr, err := r.NewAddress()
	if err != nil {
		t.Fatalf("unable to get new address: %v", err)
	}

	// Next, send amt SOTER to this address, spending from one of our mature
	// coinbase outputs.
	addrScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		t.Fatalf("unable to generate pkscript to addr: %v", err)
	}
	output := wire.NewTxOut(int64(amt), addrScript)
	txid, err := r.SendOutputs([]*wire.TxOut{output}, 10)

	return txid, err
}

func TestDAGTxnGen(t *testing.T) {
	// Set to true, to retain logs from miners
	keepLogs := false
	// Set to debug or trace to produce more logging output from miners.
	extraArgs := []string{
		//"--debuglevel=debug",
	}

	matureOutputs := 10
	txAmt := 1000
	numTxs := matureOutputs
	numBlocks := 100
	var miner, err = rpctest.New(&chaincfg.SimNetParams, nil, extraArgs, keepLogs)
	if err != nil {
		t.Fatalf("unable to create mining node: %v", err)
	}

	if err := miner.SetUp(true, uint32(matureOutputs)); err != nil {
		t.Fatalf("unable to complete mining node setup: %v", err)
	}

	if keepLogs {
		t.Logf("mining node log dir: %s", miner.LogDir())
	}

	defer miner.TearDown()

	//fmt.Printf("current balance: %d\n", miner.ConfirmedBalance())

	for x := 0; x < numBlocks; x++ {
		txnIds := make([]*chainhash.Hash, numTxs)
		for i := 0; i < numTxs; i++ {
			txnIds[i], err = spendCoinbase(miner, t, soterutil.Amount(txAmt))
			//fmt.Println(i)
			//fmt.Printf("current balance: %d\n", miner.ConfirmedBalance())
		}

		blockHashes, err := miner.Node.Generate(1)
		if err != nil {
			t.Fatalf("unable to generate single block: %v", err)
		}

		_, err = miner.Node.GetBlock(blockHashes[0])
		//fmt.Println(blockHashes)
		//fmt.Printf("mined block %d\n", x)
		//fmt.Printf("current balance after block mined: %d\n", miner.ConfirmedBalance())

		if x > 10 {
			numTxs++
		}
	}

}

func TestDAGMultiNodeTxnGen(t *testing.T) {
	// Set to true, to retain logs from miners
	keepLogs := false
	// Set to debug or trace to produce more logging output from miners.
	extraArgs := []string{
		//"--debuglevel=debug",
	}

	matureOutputs := 10
	txAmt := 1000
	numTxs := matureOutputs
	numBlocks := 100

	var miners = make([]*rpctest.Harness, 0)
	minerCount := 2
	// Max time to wait for blocks to sync between nodes, once all are generated
	//wait := time.Minute

	for i := 0; i < minerCount; i++ {
		var miner, err = rpctest.New(&chaincfg.SimNetParams, nil, extraArgs, keepLogs)
		if err != nil {
			t.Fatalf("unable to create mining node: %v", err)
		}

		if err := miner.SetUp(false, 0); err != nil {
			t.Fatalf("unable to complete mining node setup: %v", err)
		}

		if keepLogs {
			t.Logf("miner %d log dir: %s", i, miner.LogDir())
		}

		miners = append(miners, miner)
	}

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
	var futures []*rpcclient.FutureGenerateResult
	var expectedBlockCount = 0
	for m := 0; m < minerCount; m++ {
		numBlocksPerNode := uint32(50) //uint32(miners[m].ActiveNet.CoinbaseMaturity) + uint32(matureOutputs)
		expectedBlockCount += int(numBlocksPerNode)
		future := miners[m].Node.GenerateAsync(numBlocksPerNode)
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
	var blkCount int64
	maxTries := 1
	tries := 0
	for blkCount < int64(expectedBlockCount) {
		//_ = rpctest.WaitForDAG(miners, wait)
		time.Sleep(time.Second)
		blkCount, err = miners[0].Node.GetBlockCount()
		if err != nil {
			t.Fatalf("Error getting block count: %v", err)
		}
		tries++
		t.Logf("waiting for sync try %d", tries)
		if tries >= maxTries {
			//t.Log("tried all times")
			break
		}
	}


	for m := 0; m < minerCount; m++ {
		blkCount, err := miners[m].Node.GetBlockCount()
		if err != nil {
			t.Fatalf("Error getting block count: %v", err)
		}
		res, err := miners[m].Node.GetDAGTips()
		if err != nil {
			t.Fatalf("Error getting DAG tips: %v", err)
		}
		balance := miners[m].ConfirmedBalance()
		t.Logf("miner %d: block count %d\n", m, blkCount)
		t.Logf("miner %d: max height %d\n", m, res.MaxHeight)
		t.Logf("miner %d balance %d\n", m, balance)
	}


	for m := 0; m < minerCount; m++ {
		miner := miners[m]
		numTxs = matureOutputs
		for x := 0; x < numBlocks; x++ {
			//txnIds := make([]*chainhash.Hash, numTxs)
			minedTxns := 0
			for i := 0; i < numTxs; i++ {
				if miner.ConfirmedBalance() > soterutil.Amount(txAmt) {
					_, err = spendCoinbase(miner, t, soterutil.Amount(txAmt))
					if err == nil {
						minedTxns++
					} else {
						t.Logf("Error spending: %v", err)
					}
				}
			}

			_, err = miner.Node.Generate(1)

			if err == nil {
				//fmt.Printf("miner %d mined block %d with %d txns\n", m, x, minedTxns)
				//fmt.Printf("current balance after block mined: %d\n", miner.ConfirmedBalance())
			} else {
				t.Logf("Error generating block: %v", err)
			}
		}
	}


	for  m := 0; m < minerCount; m++ {
		t.Log(miners[m].Node.GetBlockCount())
		t.Log(miners[m].ConfirmedBalance())
	}
}

