// Copyright (c) 2016 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpctest

import (
	"bytes"
	"fmt"
	"github.com/soteria-dag/soterd/soterutil"
	"github.com/wcharczuk/go-chart"
	"io/ioutil"
	"reflect"
	"time"

	"github.com/soteria-dag/soterd/chaincfg/chainhash"
	"github.com/soteria-dag/soterd/rpcclient"
	"github.com/soteria-dag/soterd/soterjson"
	"github.com/soteria-dag/soterd/wire"
)

// JoinType is an enum representing a particular type of "node join". A node
// join is a synchronization tool used to wait until a subset of nodes have a
// consistent state with respect to an attribute.
type JoinType uint8

const (
	// Blocks is a JoinType which waits until all nodes share the same
	// block height.
	Blocks JoinType = iota

	// Mempools is a JoinType which blocks until all nodes have identical
	// mempool.
	Mempools
)

// JoinNodes is a synchronization tool used to block until all passed nodes are
// fully synced with respect to an attribute. This function will block for a
// period of time, finally returning once all nodes are synced according to the
// passed JoinType. This function be used to to ensure all active test
// harnesses are at a consistent state before proceeding to an assertion or
// check within rpc tests.
func JoinNodes(nodes []*Harness, joinType JoinType) error {
	switch joinType {
	case Blocks:
		return syncBlocks(nodes)
	case Mempools:
		return syncMempools(nodes)
	}
	return nil
}

// absInt32 returns the absolute value of x
func absInt32(x int32) int32 {
	if x < 0 {
		return -x
	} else {
		return x
	}
}

// We'll provide renderDagsDot with a function for picking colors of blocks.
//
// The colorPicker should return a string in the graphviz format:
// #rrggbb, where rr is 2 hex characters for red, gg is 2 hex characters for green, bb is 2 hex characters for blue.
func colorPicker (v int) string {
	color := chart.GetAlternateColor(v)
	// Slice of bytes is used here instead of int value of color, so that Sprintf
	// uses 2 characters per byte instead of 1, which is what the graphviz format wants.
	return fmt.Sprintf("#%x%x%x", []byte{color.R}, []byte{color.G}, []byte{color.B})
}


// We'll provide renderDagsDot with a function for picking style of the blocks.
//
// The stylePicker should return a string in the graphviz format:
// "filled" if input is true, or 
// "filled, dashed" if otherwise
func stylePicker (v bool) string {

	if (v) {
		return "filled"
	} 	
	return "filled, dashed"
}

// keys returns the keys for the map of integers
func keys(m map[int32]int) []int32 {
	k := make([]int32, 0)
	for v := range m {
		k = append(k, v)
	}

	return k
}


// meanInterval returns the mean difference between values of numbers from an array of numbers
func meanInterval(values []int32) float64 {
	if len(values) < 2 {
		return float64(0)
	}

	diffs := make([]int32, 0)
	for i := 0; i < (len(values) - 1); i++ {
		d := absInt32(values[i + 1] - values[i])
		diffs = append(diffs, d)
	}

	mean := float64(sumInt32(diffs)) / float64(len(diffs))

	return mean
}

// save bytes to a dynamically-named file based on the provided pattern
func save(bytes []byte, pattern string) (string, error) {
	t, err := ioutil.TempFile("", pattern)
	if err != nil {
		return "", err
	}

	_, err = t.Write(bytes)
	if err != nil {
		return "", err
	}

	err = t.Close()
	if err != nil {
		return "", err
	}

	return t.Name(), nil
}

// sumInt32 returns the sumInt32 of values in x
func sumInt32(x []int32) int32 {
	total := int32(0)
	for _, v := range x {
		total += v
	}

	return total
}

// CountDAGBlocks returns
// the number of blocks with multiple parents (indicating DAG structure) found on a node,
// the mean distance in generations between multi-parent blocks
// the error, if one occurred.
func CountDAGBlocks(node *Harness) (int, float64, error) {
	count := 0
	mpbHeights := make(map[int32]int)

	tips, err := node.Node.GetDAGTips()
	if err != nil {
		return count, meanInterval(keys(mpbHeights)), err
	}

	for height := int32(0); height <= tips.MaxHeight; height++ {
		hashes, err := node.Node.GetBlockHash(int64(height))
		if err != nil {
			return count, meanInterval(keys(mpbHeights)), err
		}

		for _, hash := range hashes {
			block, err := node.Node.GetBlock(hash)
			if err != nil {
				return count, meanInterval(keys(mpbHeights)), err
			}

			if (*block).Parents.Size > 1 {
				count++
				mpbHeights[height] = 1
			}
		}
	}

	return count, meanInterval(keys(mpbHeights)), nil
}

// DumpDAG prints out the dag on a node
func DumpDAG(i int, node *Harness) error {
	tips, err := node.Node.GetDAGTips()
	if err != nil {
		return err
	}

	for height := int32(0); height <= tips.MaxHeight; height++ {
		hashes, err := node.Node.GetBlockHash(int64(height))
		if err != nil {
			return err
		}

		for _, hash := range hashes {
			block, err := node.Node.GetBlock(hash)
			if err != nil {
				return err
			}

			fmt.Printf("miner %v\theight %v\tblock %v\tparents %v\n", i, height, hash, block.Parents.ParentHashes())
		}
	}

	return nil
}

// RenderDagsDot returns a representation of the dag in graphviz DOT file format.
//
// RenderDagsDot makes use of the "dot" command, which is a part of the "graphviz" suite of software.
// http://graphviz.org/
func RenderDagsDot(nodes []*Harness, rankdir string) ([]byte, error) {
	var dot bytes.Buffer
	// How many characters of a hash string to use for the 'label' of a block in the graph
	smallHashLen := 7

	// Map blocks to the nodes that created them. This will be used to color blocks in dag
	blockCreator := make(map[string]int)
	for i, n := range nodes {
		resp, err := n.Node.GetBlockMetrics()
		if err != nil {
			continue
		}

		for _, hash := range resp.BlkHashes {
			blockCreator[hash] = i
		}
	}

	// We'll use the first node for the dag, and metrics from all nodes for block coloring
	node := nodes[0]
	tips, err := node.Node.GetDAGTips()
	if err != nil {
		return dot.Bytes(), err
	}

	dag := make([][]*wire.MsgBlock, 0)
	blockIndex := make(map[string]*wire.MsgBlock)

	// Index all the blocks
	for height := int32(0); height <= tips.MaxHeight; height++ {
		blocks := make([]*wire.MsgBlock, 0)

		hashes, err := node.Node.GetBlockHash(int64(height))
		if err != nil {
			return dot.Bytes(), err
		}

		for _, hash := range hashes {
			block, err := node.Node.GetBlock(hash)
			if err != nil {
				return dot.Bytes(), err
			}

			blockIndex[block.BlockHash().String()] = block
			blocks = append(blocks, block)
		}

		dag = append(dag, blocks)
	}

	// Build a map of Block coloring Results 
	dagcoloring, err := node.Node.GetDAGColoring()
	if err != nil {
		return dot.Bytes(), err
	}
	blockcoloring := make(map[string]bool)
	for _, dagNode := range dagcoloring {
		hash := dagNode.Hash
		coloring := dagNode.IsBlue
		blockcoloring[hash] = coloring
	}

	// Express dag in DOT file format

	// graphIndex tracks block hash -> graph node number, which is used to connect parent-child blocks together.
	graphIndex := make(map[string]int)
	// n keeps track of the 'node' number in graph file language
	n := 0

	// Specify that this graph is directed, and set the ID to 'dag'
	_, err = fmt.Fprintln(&dot, "digraph dag {")
	if err != nil {
		return dot.Bytes(), err
	}

	// Set graph-level attribute to help keep a tighter left-aligned layout of graph in large renderings.
	_, err = fmt.Fprintln(&dot, "ordering=out;")
	if err != nil {
		return dot.Bytes(), err
	}

	// Set graph orientation 
	_, err = fmt.Fprintf(&dot, "rankdir=\"%s\";", rankdir)
	if err != nil {
		return dot.Bytes(), err
	}

	// Create a node in the graph for each block
	for height, blocks := range dag {
		for _, block := range blocks {
			hash := block.BlockHash().String()
			smallHashIndex := len(hash) - smallHashLen
			graphIndex[hash] = n

			// determine the coloring of the block and fetch the style string: default, "filled" or "filled,dashed"
			dagcoloring := blockcoloring[hash]
			style := stylePicker(dagcoloring)

			creator, exists := blockCreator[hash]

			var err error
			if exists {
				// color this block based on which miner created it

				color := colorPicker(creator)
				_, err = fmt.Fprintf(&dot, "n%d [label=\"%s\", tooltip=\"node %d height %d hash %s\", fillcolor=\"%s\", style=\"%s\"];\n",
					n, hash[smallHashIndex:], creator, height, hash, color, style)
			} else {
				// No color for this block
				_, err = fmt.Fprintf(&dot, "n%d [label=\"%s\", tooltip=\"height %d hash %s\", style=\"%s\"];\n",
					n, hash[smallHashIndex:], height, hash, style)
			}
			if err != nil {
				return dot.Bytes(), err
			}

			n++
		}
	}

	// Connect the nodes in the graph together
	for _, blocks := range dag {
		for _, block := range blocks {
			blockN := graphIndex[block.BlockHash().String()]

			for _, parent := range block.Parents.Parents {
				parentN := graphIndex[parent.Hash.String()]

				_, err := fmt.Fprintf(&dot, "n%d -> n%d;\n", blockN, parentN)
				if err != nil {
					return dot.Bytes(), err
				}
			}
		}
	}

	// Close the graph statement list
	dot.WriteString("}")

	return dot.Bytes(), nil
}

// SaveDagHTML save an HTML document containing an svg image of the node's dag
func SaveDagHTML(r *Harness) (string, error) {
	dot, err := r.Node.RenderDag()
	if err != nil {
		return "", err
	}

	svg, err := soterutil.DotToSvg([]byte(dot.Dot))
	if err != nil {
		return "", err
	}

	svgEmbed, err := soterutil.StripSvgXmlDecl(svg)
	if err != nil {
		return "", err
	}

	h, err := soterutil.RenderSvgHTML(svgEmbed, "dag")
	if err != nil {
		return "", err
	}

	return save(h, "dag_*.html")
}

// checkHeaderEqual returns an error if headers aren't identical
func checkHeaderEqual(headerA, headerB wire.BlockHeader) error {
	if headerA.Version != headerB.Version {
		return fmt.Errorf("block header Version mismatch: %v != %v", headerA.Version, headerB.Version)
	}

	if !headerA.PrevBlock.IsEqual(&headerB.PrevBlock) {
		return fmt.Errorf("block header PrevBlock mismatch: %v != %v", headerA.PrevBlock, headerB.PrevBlock)
	}

	if !headerA.MerkleRoot.IsEqual(&headerB.MerkleRoot) {
		return fmt.Errorf("block header MerkleRoot mismatch: %v != %v", headerA.MerkleRoot, headerB.MerkleRoot)
	}

	if !headerA.Timestamp.Equal(headerB.Timestamp) {
		return fmt.Errorf("block header Timestamp mismatch: %v != %v", headerA.Timestamp, headerB.Timestamp)
	}

	if headerA.Bits != headerB.Bits {
		return fmt.Errorf("block header Bits mismatch: %v != %v", headerA.Bits, headerB.Bits)
	}

	if headerA.Nonce != headerB.Nonce {
		return fmt.Errorf("block header Nonce mismatch: %v != %v", headerA.Bits, headerB.Bits)
	}

	return nil
}

// assertParentIdentical returns an error if given parents aren't identical
func assertParentIdentical(parentA, parentB *wire.Parent) error {
	if !parentA.Hash.IsEqual(&parentB.Hash) {
		return fmt.Errorf("block parent Hash mismatch: %v != %v", parentA.Hash, parentB.Hash)
	}

	if parentA.Data != parentB.Data {
		return fmt.Errorf("block parent Data mismatch: %v != %v", parentA.Data, parentB.Data)
	}

	return nil
}

// checkParentSubHeaderEqual returns an error if parent sub-headers aren't identical
func checkParentSubHeaderEqual(parentsA, parentsB wire.ParentSubHeader) error {
	if parentsA.Version != parentsB.Version {
		return fmt.Errorf("block parent sub-header Version mismatch: %v != %v", parentsA.Version, parentsB.Version)
	}

	if parentsA.Size != parentsB.Size {
		return fmt.Errorf("block parent sub-header Size (claimed number of parents) mismatch: %v != %v", parentsA.Size, parentsB.Size)
	}

	if len(parentsA.Parents) != len(parentsB.Parents) {
		return fmt.Errorf("block parent sub-header actual number of parents mismatch: %v != %v", len(parentsA.Parents), len(parentsB.Parents))
	}

	// Index parents, to assist with finding missing parents and in comparing them
	parentIndexA := make(map[string]int)
	parentIndexB := make(map[string]int)

	for i, hash := range parentsA.ParentHashes() {
		parentIndexA[hash.String()] = i
	}

	for i, hash := range parentsB.ParentHashes() {
		parentIndexB[hash.String()] = i
	}

	// Examine parents
	for hash, indexA := range parentIndexA {
		// Check for missing parent
		indexB, ok := parentIndexB[hash]
		if !ok {
			return fmt.Errorf("block parent sub-header B missing A parent: %v", hash)
		}

		// Confirm parents are identical
		err := assertParentIdentical(parentsA.Parents[indexA], parentsB.Parents[indexB])
		if err != nil {
			return err
		}

	}

	for hash, _ := range parentIndexB {
		// We've already checked if parents are identical, so we just need to see if parentsB
		// has parents that parentsA does not.
		//
		// (I think we would only hit this case if the Size field of the parent sub-header didn't match
		// the length of the Parents array)
		_, ok := parentIndexA[hash]
		if !ok {
			return fmt.Errorf("block parent sub-header A missing B parent: %v", hash)
		}
	}

	return nil
}

// checkTransactionsEqual returns an error if the transactions aren't identical.
// Since the wire.MsgTx.TxHash() method serializes the object into a hash, we implicitly check for
// equivalence by testing for missing transactions in transA and transB
func checkTransactionsEqual(transA, transB []*wire.MsgTx) error {
	if len(transA) != len(transB) {
		return fmt.Errorf("block transactions count mismatch: %v != %v", len(transA), len(transB))
	}

	// Index transactions, to assist with finding missing transactions and in comparing them
	txIndexA := make(map[string]int)
	txIndexB := make(map[string]int)

	for i, tx := range transA {
		txHash := tx.TxHash().String()
		txIndexA[txHash] = i
	}

	for i, tx := range transB {
		txHash := tx.TxHash().String()
		txIndexB[txHash] = i
	}

	// Examine transactions
	for hash, _ := range txIndexA {
		_, ok := txIndexB[hash]
		if !ok {
			return fmt.Errorf("block transaction from A missing from or different in B: %v", hash)
		}
	}

	for hash, _ := range txIndexB {
		_, ok := txIndexA[hash]
		if !ok {
			return fmt.Errorf("block transaction from B missing from or different in A: %v", hash)
		}
	}

	return nil
}

// syncMempools blocks until all nodes have identical mempools.
func syncMempools(nodes []*Harness) error {
	poolsMatch := false

retry:
	for !poolsMatch {
		firstPool, err := nodes[0].Node.GetRawMempool()
		if err != nil {
			return err
		}

		// If all nodes have an identical mempool with respect to the
		// first node, then we're done. Otherwise, drop back to the top
		// of the loop and retry after a short wait period.
		for _, node := range nodes[1:] {
			nodePool, err := node.Node.GetRawMempool()
			if err != nil {
				return err
			}

			if !reflect.DeepEqual(firstPool, nodePool) {
				time.Sleep(time.Millisecond * 100)
				continue retry
			}
		}

		poolsMatch = true
	}

	return nil
}

// syncBlocks blocks until all nodes report the same best chain.
func syncBlocks(nodes []*Harness) error {
	blocksMatch := false

retry:
	for !blocksMatch {
		var prevHash *chainhash.Hash
		var prevHeight int32
		for _, node := range nodes {
			blockHash, blockHeight, err := node.Node.GetBestBlock()
			if err != nil {
				return err
			}
			if prevHash != nil && (*blockHash != *prevHash ||
				blockHeight != prevHeight) {

				time.Sleep(time.Millisecond * 100)
				continue retry
			}
			prevHash, prevHeight = blockHash, blockHeight
		}

		blocksMatch = true
	}

	return nil
}

// CheckBlocksEqual returns an error if the blocks aren't identical
func CheckBlocksEqual(blockA, blockB *wire.MsgBlock) error {
	// Inspect header
	err := checkHeaderEqual(blockA.Header, blockB.Header)
	if err != nil {
		return err
	}

	// Inspect parent sub-header
	err = checkParentSubHeaderEqual(blockA.Parents, blockB.Parents)
	if err != nil {
		return err
	}

	// Inspect transactions
	err = checkTransactionsEqual(blockA.Transactions, blockB.Transactions)
	if err != nil {
		return err
	}

	return nil
}

// CompareDAG returns an error if the DAG on any of the nodes is different
func CompareDAG(nodes []*Harness) error {
	// All node DAGs will be compared to the first node's
	targetNodeIndex := 0
	targetNode := nodes[targetNodeIndex]
	tips, err := targetNode.Node.GetDAGTips()
	if err != nil {
		return err
	}

	dag := [][]*chainhash.Hash{}
	for height := int32(0); height <= tips.MaxHeight; height++ {
		hashes, err := targetNode.Node.GetBlockHash(int64(height))
		if err != nil {
			return err
		}

		dag = append(dag, hashes)
	}

	// Compare other nodes to the targetNode
	for i, node := range nodes {
		if i == targetNodeIndex {
			continue
		}

		for height, hashes := range dag {
			otherHashes, err := node.Node.GetBlockHash(int64(height))
			if err != nil {
				return err
			}

			// Check if the number of blocks at this height are the same (should catch duplicate block issues)
			if len(otherHashes) != len(hashes) {
				err := fmt.Errorf("node %v has different number of hashes at height %v than node %v: %v != %v",
					i, height, targetNodeIndex, len(otherHashes), len(hashes))
				return err
			}

			// Check if there's missing hashes in otherHashes
			for _, hash := range hashes {
				match := false
				for _, otherHash := range otherHashes {
					if hash.IsEqual(otherHash) {
						match = true
						break
					}
				}
				if !match {
					err := fmt.Errorf("node %v has missing hash %v at height %v", i, hash, height)
					return err
				}
			}

			// Check if there's extra hashes in otherHashes
			for _, otherHash := range otherHashes {
				match := false
				for _, hash := range hashes {
					if otherHash.IsEqual(hash) {
						match = true
						break
					}
				}
				if !match {
					err := fmt.Errorf("node %v has extra hash %v at height %v", i, otherHash, height)
					return err
				}
			}

			// Check that blocks at this height are identical
			for _, hash := range hashes {
				block, err := targetNode.Node.GetBlock(hash)
				if err != nil {
					err := fmt.Errorf("failed to get block %v from target node %v", hash, targetNodeIndex)
					return err
				}

				otherBlock, err := nodes[i].Node.GetBlock(hash)
				if err != nil {
					err := fmt.Errorf("failed to get block %v from node %v", hash, i)
					return err
				}

				err = CheckBlocksEqual(block, otherBlock)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// IsConnected returns true if 'from' node is connected to 'to' node
func IsConnected(from *Harness, to *Harness) (bool, error) {
	toAddr := to.P2PAddress()

	fromPeers, err := from.Node.GetPeerInfo()
	if err != nil {
		return false, err
	}

	for _, peerInfo := range fromPeers {
		if peerInfo.Addr == toAddr {
			return true, nil
		}
	}

	return false, nil
}

// ConnectNode establishes a new peer-to-peer connection between the "from"
// harness and the "to" harness.  The connection made is flagged as persistent,
// therefore in the case of disconnects, "from" will attempt to reestablish a
// connection to the "to" harness.
func ConnectNode(from *Harness, to *Harness) error {
	peerInfo, err := from.Node.GetPeerInfo()
	if err != nil {
		return err
	}
	numPeers := len(peerInfo)

	targetAddr := to.P2PAddress()
	if err := from.Node.AddNode(targetAddr, rpcclient.ANAdd); err != nil {
		return err
	}

	// Block until a new connection has been established.
	peerInfo, err = from.Node.GetPeerInfo()
	if err != nil {
		return err
	}
	for len(peerInfo) <= numPeers {
		peerInfo, err = from.Node.GetPeerInfo()
		if err != nil {
			return err
		}
	}

	return nil
}

// ConnectNodes connects all the nodes to one another
func ConnectNodes(nodes []*Harness) error {
	for i, node := range nodes {
		for j, peer := range nodes {
			if i == j {
				continue
			}

			// Skip nodes that already have a connection established from either end
			connected, err := IsConnected(peer, node)
			if err != nil {
				return err
			}
			if connected {
				continue
			}

			connected, err = IsConnected(node, peer)
			if err != nil {
				return err
			}
			if connected {
				continue
			}

			err = ConnectNode(peer, node)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// TearDownAll tears down all active test harnesses.
func TearDownAll() error {
	harnessStateMtx.Lock()
	defer harnessStateMtx.Unlock()

	for _, harness := range testInstances {
		if err := harness.tearDown(); err != nil {
			return err
		}
	}

	return nil
}

// WaitForBlock waits up to 'wait' duration from now for a block to be found on the harness' node.
func WaitForBlock(h *Harness, hash *chainhash.Hash, wait time.Duration) (*wire.MsgBlock, error) {
	pollInterval := time.Duration(time.Millisecond * 500)
	waitThreshold := time.Now().Add(wait)

	for {
		block, err := h.Node.GetBlock(hash)
		switch errType := err.(type) {
		case nil:
			// No error: Block exists and was returned
			return block, err
		case *soterjson.RPCError:
			// RPC error
			if errType.Code == soterjson.ErrRPCBlockNotFound {
				// Block wasn't found
				if time.Now().Before(waitThreshold) {
					// We'll wait a bit for the block to appear
					time.Sleep(pollInterval)
				} else {
					// Timed out waiting for block to appear
					return block, err
				}
			} else {
				// Other RPC error
				return block, err
			}
		default:
			// A non-RPC error
			return block, err
		}
	}
}

// WaitForBlocks waits for all of the given nodes to have the given blocks
func WaitForBlocks(nodes []*Harness, hashes []*chainhash.Hash, wait time.Duration) error {
	pollInterval := time.Duration(time.Millisecond * 500)
	waitThreshold := time.Now().Add(wait)

	for {
		matchingNodes := 0
		for _, node := range nodes {
			matching := 0
			for _, hash := range hashes {
				_, err := WaitForBlock(node, hash, wait)
				if err != nil {
					continue
				}
				matching++
			}

			if matching == len(hashes) {
				matchingNodes++
			}
		}

		if matchingNodes == len(nodes) {
			return nil
		} else if time.Now().Before(waitThreshold){
			time.Sleep(pollInterval)
		} else {
			var hashStrings []string
			for _, hash := range hashes {
				hashStr := (*hash).String()
				hashStrings = append(hashStrings, hashStr)
			}
			err := fmt.Errorf("Timeout while waiting for nodes to sync blocks %v", hashStrings)
			return err
		}
	}
}

// WaitForDAG waits for all the given nodes to have the same dag
func WaitForDAG(nodes []*Harness, wait time.Duration) error {
	pollInterval := time.Duration(time.Second)
	waitThreshold := time.Now().Add(wait)

	if len(nodes) < 2 {
		return nil
	}

	for {
		err := CompareDAG(nodes)
		if err != nil {
			if time.Now().Before(waitThreshold) {
				time.Sleep(pollInterval)
			} else {
				timeout := fmt.Errorf("Timeout while waiting for nodes to sync dag")
				return timeout
			}
		} else {
			return nil
		}
	}
}

// ActiveHarnesses returns a slice of all currently active test harnesses. A
// test harness if considered "active" if it has been created, but not yet torn
// down.
func ActiveHarnesses() []*Harness {
	harnessStateMtx.RLock()
	defer harnessStateMtx.RUnlock()

	activeNodes := make([]*Harness, 0, len(testInstances))
	for _, harness := range testInstances {
		activeNodes = append(activeNodes, harness)
	}

	return activeNodes
}
