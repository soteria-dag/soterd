// This file is ignored during the regular tests due to the following build tag.
// +build rpctest dag dagsync
// You can run tests from this file in isolation by using the build tags, like so:
// go test -v -count=1 -tags "dagsync" github.com/soteria-dag/soterd/integration

package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/chaincfg/chainhash"
	"github.com/soteria-dag/soterd/integration/rpctest"
	"github.com/soteria-dag/soterd/rpcclient"
	"github.com/soteria-dag/soterd/soterutil"
)

// findMissingBlocks returns the number of blocks missing in the dag from expectedBlocks
func findMissingBlocks(nodeNum int, aNode *rpctest.Harness, expectedBlocks map[string]int) (map[string]int, error) {
	missing := make(map[string]int)

	tips, err := aNode.Node.GetDAGTips()
	if err != nil {
		return missing, err
	}

	foundBlocks := make(map[string]int)
	for height := int32(0); height <= tips.MaxHeight; height++ {
		hashes, err := aNode.Node.GetBlockHash(int64(height))
		if err != nil {
			return missing, err
		}
		var hashStrings []string
		for _, h := range hashes {
			hashStrings = append(hashStrings, (*h).String())
		}

		for _, hash := range hashes {
			foundBlocks[hash.String()] = 0
		}
	}

	for hash, _ := range expectedBlocks {
		_, ok := foundBlocks[hash]
		if !ok {
			missing[hash] = 0
		}
	}

	return missing, nil
}

// save bytes to a file descriptor
func save(bytes []byte, fh *os.File) error {
	_, err := fh.Write(bytes)
	if err != nil {
		return err
	}

	err = fh.Close()
	if err != nil {
		return err
	}

	return nil
}

// renderDagHTML renders the dag on the node to an HTML file, and returns the path to the file
func renderDagHTML(h *rpctest.Harness) (string, error) {
	result, err := h.Node.RenderDag()
	if err != nil {
		return "", fmt.Errorf("failed to render dag: %s", err)
	}

	svg, err := soterutil.DotToSvg([]byte(result.Dot))
	if err != nil {
		return "", fmt.Errorf("failed to convert dot to svg: %s", err)
	}

	svgEmbed, err := soterutil.StripSvgXmlDecl(svg)
	if err != nil {
		return "", fmt.Errorf("failed to strip svg xml declaration tag: %s", err)
	}

	html, err := soterutil.RenderSvgHTML(svgEmbed, "dag")

	fh, err := ioutil.TempFile("", "dag_*.html")
	if err != nil {
		return "", err
	}

	err = save(html, fh)
	if err != nil {
		return fh.Name(), err
	}

	return fh.Name(), err
}

// TestBlockSync generates and tests block transfer between a miner and sync peers
func TestBlockSync(t *testing.T) {
	// Max time to wait for blocks to sync between nodes
	wait := time.Second * 30
	// Number of blocks to generate in initial sync test
	blockCount := 250
	// Set to true, to retain logs from miners
	keepLogs := false
	// Set to debug or trace to produce more logging output from miners.
	extraArgs := []string{
		//"--debuglevel=debug",
	}

	// Initialize primary mining node
	miner, err := rpctest.New(&chaincfg.SimNetParams, nil, extraArgs, keepLogs)
	if err != nil {
		t.Fatalf("unable to create primary mining node: %v", err)
	}
	if err := miner.SetUp(false, 0); err != nil {
		t.Fatalf("unable to complete mining node setup: %v", err)
	}
	defer miner.TearDown()

	// Initialize a sync node
	sync, err := rpctest.New(&chaincfg.SimNetParams, nil, extraArgs, keepLogs)
	if err != nil {
		t.Fatalf("unable to create sync node: %v", err)
	}
	if err := sync.SetUp(false, 0); err != nil {
		t.Fatalf("unable to complete sync node setup: %v", err)
	}
	defer sync.TearDown()

	// Initialize a second sync node, which will test
	// - syncing from height 0 between the miner and this node
	// - syncing from its current height after disconnecting and reconnecting to miner node
	sync2, err := rpctest.New(&chaincfg.SimNetParams, nil, extraArgs, keepLogs)
	if err != nil {
		t.Fatalf("unable to create sync 2 node: %v", err)
	}
	if err := sync2.SetUp(false, 0); err != nil {
		t.Fatalf("unable to complete sync node setup: %v", err)
	}
	defer sync2.TearDown()

	if keepLogs {
		t.Logf("Log dir for miner: %s", miner.LogDir())
		t.Logf("Log dir for sync: %s", sync.LogDir())
		t.Logf("Log dir for sync 2: %s", sync2.LogDir())
	}

	// Establish a p2p connection from the sync node to the minter
	if err := rpctest.ConnectNode(sync, miner); err != nil {
		t.Fatalf("unable to connect sync node to miner: %v", err)
	}

	// Confirm that the peers are connected
	connected, err := rpctest.IsConnected(sync, miner)
	if err != nil {
		t.Fatalf("sync node not connected to miner: %v", err)
	} else if !connected {
		t.Fatal("sync node not connected to miner")
	}

	// Ask the miner to mine some blocks
	blockHashes, err := miner.Node.Generate(uint32(blockCount))
	if err != nil {
		t.Fatalf("unable to generate blocks: %v", err)
	}

	for _, hash := range blockHashes {
		// Miner should already have the block available, since it generated it >_>
		minerBlock, err := rpctest.WaitForBlock(miner, hash, wait)
		if err != nil {
			t.Fatalf("unable to retrieve block from miner %v: %v", hash, err)
		}

		syncBlock, err := rpctest.WaitForBlock(sync, hash, wait)
		if err != nil {
			t.Fatalf("unable to retrieve block from sync node %v: %v", hash, err)
		}

		// Compare blocks
		err = rpctest.CheckBlocksEqual(minerBlock, syncBlock)
		if err != nil {
			t.Fatalf("block %v not identical between nodes!: %v", hash, err)
		}
	}

	// Connect the sync2 node to the miner
	err = sync2.Node.AddNode(miner.P2PAddress(), rpcclient.ANAdd)
	if err != nil {
		t.Fatalf("unable to connect sync2 node to miner: %v", err)
	}

	bootstrapNodes := []*rpctest.Harness{miner, sync2}
	// Wait for blocks to sync between nodes
	_ = rpctest.WaitForDAG(bootstrapNodes, wait)

	// Confirm that the DAG between miner and sync2 is identical
	err = rpctest.CompareDAG(bootstrapNodes)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// Disconnect the sync node from the miner
	err = sync.Node.AddNode(miner.P2PAddress(), rpcclient.ANRemove)
	if err != nil {
		t.Fatalf("unable to disconnect sync node from miner: %s", err)
	}
	// Confirm that sync node is not connected to miner
	connected, err = rpctest.IsConnected(sync, miner)
	if err != nil {
		t.Fatalf("couldn't tell if sync node connected to miner: %s", err)
	} else if connected {
		t.Fatal("sync node still connected to miner")
	}

	// Mine some more blocks
	_, err = miner.Node.Generate(5)
	if err != nil {
		t.Fatalf("unable to generate blocks: %s", err)
	}

	// Re-connect sync node to miner
	if err := rpctest.ConnectNode(sync, miner); err != nil {
		t.Fatalf("unable to connect sync node to miner: %v", err)
	}
	// Confirm that the peers are connected
	connected, err = rpctest.IsConnected(sync, miner)
	if err != nil {
		t.Fatalf("couldn't tell if sync node connected to miner: %s", err)
	} else if !connected {
		t.Fatal("sync node not connected to miner")
	}

	reconnectedNodes := []*rpctest.Harness{miner, sync}
	// Wait for blocks to sync between nodes
	_ = rpctest.WaitForDAG(reconnectedNodes, wait)

	// Confirm that the DAG between miner and sync is identical
	err = rpctest.CompareDAG(reconnectedNodes)
	if err != nil {
		t.Fatalf(err.Error())
	}
}

// TestDAGBlock attempts to generate and sync a DAG between multiple nodes, with one multi-parent block
func TestDAGBlock(t *testing.T) {
	var miners []*rpctest.Harness

	// Number of miners to spawn
	minerCount := 4
	// Number of blocks to generate on each miner
	blockCount := 3
	// Max time to wait for blocks to sync between nodes, once all are generated
	wait := time.Second * 10

	// Set to true, to retain logs from miners
	keepLogs := false
	// Set to debug or trace to produce more logging output from miners
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

	// Track how many times each block has been seen
	var minerBlocks []map[string]int
	for i := 0; i < minerCount; i++ {
		minerBlocks = append(minerBlocks, make(map[string]int))
	}

	// Generate blocks on each miner.
	// We mine blocks before connecting nodes, to guarantee multiple tips
	// We'll later do another round of generation to sync the nodes to the same tips
	for i, miner := range miners {
		hashes, err := miner.Node.Generate(uint32(blockCount))
		if err != nil {
			t.Fatalf("miner %v failed to generate blocks: %v", i, err)
		}

		for _, hash := range hashes {
			hashStr := (*hash).String()

			if _, exists := minerBlocks[i][hashStr]; exists {
				minerBlocks[i][hashStr]++
			} else {
				minerBlocks[i][hashStr] = 1
			}
		}

		//// Save the dag generated by each miner
		//dagFile, err := renderDagHTML(miner)
		//if err != nil {
		//	t.Fatalf("miner %d failed to render dag: %s", i, err)
		//}
		//t.Logf("miner %d rendered dag ~before~ sync: %s", i, dagFile)
	}

	// Connect the nodes to one another
	err := rpctest.ConnectNodes(miners)
	if err != nil {
		t.Fatalf("unable to connect nodes: %v", err)
	}

	// Wait for the blocks to sync between miners
	_ = rpctest.WaitForDAG(miners, wait)

	// Generate another set of blocks, to trigger sync (since they are all at the same height)
	// This time we generate 1 block on miners in sequence, so that blocks from all nodes are propagated
	// In the end, should all be at the same height.
	//
	// The first block generated on the second miner will have multiple parents, because the first miner
	// triggered sync with its new block, so the second miner's tips will include the last generated blocks from
	// both first and second miner.
	//t.Log("Second generation of blocks to trigger sync")
	var multiParentBlockHash *chainhash.Hash
	for i, miner := range miners {
		if err != nil {
			t.Fatalf("miner %v failed to get tips: %v", i, err)
		}

		hashes, err := miner.Node.Generate(uint32(1))
		if err != nil {
			t.Fatalf("miner %v failed to generate blocks: %v", i, err)
		}

		for _, hash := range hashes {
			hashStr := (*hash).String()

			if _, exists := minerBlocks[i][hashStr]; exists {
				minerBlocks[i][hashStr]++
			} else {
				minerBlocks[i][hashStr] = 1
			}
		}

		firstHash := hashes[0]
		if i == 1 {
			// Second miner's first block should have multiple parents
			multiParentBlockHash = firstHash
		}

		err = rpctest.WaitForBlocks(miners, hashes, wait)

		if err != nil {
			t.Fatalf("miner %v %v", i, err)
		}
	}

	// Check for duplicate blocks
	for i, blocks := range minerBlocks {
		// Check for duplicate blocks within miner
		for hashStr, count := range blocks {
			if count > 1 {
				t.Fatalf("miner %v generated block %v %v times", i, hashStr, count)
			}
		}

		// Check for duplicate blocks between miners
		for j, otherBlocks := range minerBlocks {
			if i == j {
				continue
			}

			for hashStr, _ := range blocks {
				if _, exists := otherBlocks[hashStr]; exists {
					t.Fatalf("miner %v and %v generated the same block %v", i, j, hashStr)
				}
			}
		}
	}

	// Confirm that a multi-parent (DAG) block was generated
	for i, miner := range miners {
		block, err := miner.Node.GetBlock(multiParentBlockHash)
		if err != nil {
			t.Fatalf("miner %v failed to get multi-parent block %v", i, *multiParentBlockHash)
		}

		//// Save the dag generated by each miner
		//dagFile, err := renderDagHTML(miner)
		//if err != nil {
		//	t.Fatalf("miner %d failed to render dag: %s", i, err)
		//}
		//t.Logf("miner %d rendered dag ~after~ sync: %s", i, dagFile)

		if block.Parents.Size < 2 {
			t.Fatalf("miner %v block %v not multi-parent; parents %v", i, *multiParentBlockHash, block.Parents.ParentHashes())
		}
	}

	// Check nodes for missing blocks from their original generated set
	// If they're missing then there's an issue with processing synced/advertised blocks between nodes
	for i, miner := range miners {
		_, err := findMissingBlocks(i, miner, minerBlocks[i])
		if err != nil {
			t.Fatalf("miner %v\tcouldn't find missing blocks: %v", i, err)
		}
	}

	// Confirm that the DAG on all miners is identical
	err = rpctest.CompareDAG(miners)
	if err != nil {
		t.Fatalf(err.Error())
	}
}