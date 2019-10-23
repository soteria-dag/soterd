// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdag

import (
	"encoding/binary"
	"fmt"
	"github.com/Qitmeer/qitmeer-lib/crypto/cuckoo"
	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/chaincfg/chainhash"
	"github.com/soteria-dag/soterd/soterutil"
	"github.com/soteria-dag/soterd/txscript"
	"github.com/soteria-dag/soterd/wire"
	"math"
	"reflect"
	"testing"
	"time"
)
func calcMerkleRoot(txns []*wire.MsgTx) chainhash.Hash {
	if len(txns) == 0 {
		return chainhash.Hash{}
	}

	utilTxns := make([]*soterutil.Tx, 0, len(txns))
	for _, tx := range txns {
		utilTxns = append(utilTxns, soterutil.NewTx(tx))
	}
	merkles := BuildMerkleTreeStore(utilTxns, false)
	return *merkles[len(merkles)-1]
}

func createMsgBlockForTest(height uint32, ts int64, parents []*wire.MsgBlock, transactions []*wire.MsgTx) *wire.MsgBlock {
	var coinbaseTx = createCoinbaseTxForTest(height)

	parentHashes := make([]*chainhash.Hash, len(parents))
	for i, parent := range parents {
		parentHash := parent.BlockHash()
		parentHashes[i] = &parentHash
	}

	parentData := make([]*wire.Parent, len(parents))
	for i, parent := range parents {
		parentData[i] = &wire.Parent{Hash: parent.BlockHash()}
	}

	var txs = []*wire.MsgTx{coinbaseTx}
	if transactions != nil {
		for _, tx := range transactions {
			txs = append(txs, tx)
		}
	}

	var blockPrevHash = GenerateTipsHash(parentHashes)
	var Block = wire.MsgBlock{
		Header: wire.BlockHeader{
			Version: 1,
			PrevBlock: *blockPrevHash,
			MerkleRoot: calcMerkleRoot(txs),
			Timestamp: time.Unix(ts, 0),
			Bits:      0x1010000,               // 1
			Nonce:     0x00000000,
		},
		Parents: wire.ParentSubHeader{
			Version: 1,
			Size: int32(len(parentData)),
			Parents: parentData,
		},
		Verification: wire.VerificationSubHeader{
			Version: 1,
			Size: 0,
			CycleNonces: []uint32{},
		},
		Transactions: txs,
	}

	// solve for cuckoo cycle nonces
	solveMsgBlockForTest(&Block)

	return &Block
}

// solveMsgBlockForTest solves a block with cuckoo cycle
func solveMsgBlockForTest(block *wire.MsgBlock) {
	var targetDifficulty = CompactToBig(block.Header.Bits)

	c := cuckoo.NewCuckoo()

	for i := block.Header.Nonce; i <= uint32(math.MaxUint32); i++ {
		block.Header.Nonce = i
		hash := block.Header.BlockHash()

		cycleNonces, isFound := c.PoW(hash.CloneBytes())
		if !isFound {
			continue
		}

		if err := cuckoo.Verify(hash.CloneBytes(), cycleNonces); err != nil {
			continue
		}

		// The block is solved when:
		// a) The cuckoo cycle is valid
		// b) The cuckoo cycle proof difficulty is greater than or equal to the target difficulty
		proofDifficulty := ProofDifficulty(cycleNonces)
		if proofDifficulty.Cmp(targetDifficulty) >= 0 {
			block.Verification.CycleNonces = cycleNonces
			block.Verification.Size = int32(len(cycleNonces))
			break
		}
	}
}

var opTrueScript = []byte{txscript.OP_TRUE}

func standardCoinbaseScript(blockHeight int32, extraNonce uint64) ([]byte, error) {
	return txscript.NewScriptBuilder().AddInt64(int64(blockHeight)).
		AddInt64(int64(extraNonce)).Script()
}

// opReturnScript returns a provably-pruneable OP_RETURN script with the
// provided data.
func opReturnScript(data []byte) []byte {
	builder := txscript.NewScriptBuilder()
	script, err := builder.AddOp(txscript.OP_RETURN).AddData(data).Script()
	if err != nil {
		panic(err)
	}
	return script
}

// uniqueOpReturnScript returns a standard provably-pruneable OP_RETURN script
// with a random uint64 encoded as the data.
func uniqueOpReturnScript() []byte {
	rand, err := wire.RandomUint64()
	if err != nil {
		panic(err)
	}

	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data[0:8], rand)
	return opReturnScript(data)
}

func createCoinbaseTxForTest(height uint32) *wire.MsgTx {
	extraNonce := uint64(0)
	coinbaseScript, err := standardCoinbaseScript(int32(height), extraNonce)
	if err != nil {
		panic(err)
	}

	tx := wire.NewMsgTx(1)
	tx.AddTxIn(&wire.TxIn{
		// Coinbase transactions have no inputs, so previous outpoint is
		// zero hash and max index.
		PreviousOutPoint: *wire.NewOutPoint(&chainhash.Hash{},
			wire.MaxPrevOutIndex),
		Sequence:        wire.MaxTxInSequenceNum,
		SignatureScript: coinbaseScript,
	})
	tx.AddTxOut(&wire.TxOut{
		Value:    0x12a05f200, // 5000000000
		PkScript: opTrueScript,
	})
	return tx

	/*	v := uint32(height)
	buf := make([]byte, 5)
	buf[0] = 0x04
	binary.BigEndian.PutUint32(buf[1:], v)

	var CoinbaseTx = wire.MsgTx{
		Version: 1,
		TxIn: []*wire.TxIn{
			{
				PreviousOutPoint: wire.OutPoint{
					Hash:  chainhash.Hash{},
					Index: 0xffffffff,
				},
				SignatureScript: buf,
				Sequence:        0xffffffff,
			},
		},
		TxOut: []*wire.TxOut{
			{
				Value: 0x12a05f200, // 5000000000
				PkScript: []byte{
					0x41, // OP_DATA_65
					0x04, 0x1b, 0x0e, 0x8c, 0x25, 0x67, 0xc1, 0x25,
					0x36, 0xaa, 0x13, 0x35, 0x7b, 0x79, 0xa0, 0x73,
					0xdc, 0x44, 0x44, 0xac, 0xb8, 0x3c, 0x4e, 0xc7,
					0xa0, 0xe2, 0xf9, 0x9d, 0xd7, 0x45, 0x75, 0x16,
					0xc5, 0x81, 0x72, 0x42, 0xda, 0x79, 0x69, 0x24,
					0xca, 0x4e, 0x99, 0x94, 0x7d, 0x08, 0x7f, 0xed,
					0xf9, 0xce, 0x46, 0x7c, 0xb9, 0xf7, 0xc6, 0x28,
					0x70, 0x78, 0xf8, 0x01, 0xdf, 0x27, 0x6f, 0xdf,
					0x84, // 65-byte signature
					0xac, // OP_CHECKSIG
				},
			},
		},
		LockTime: 0,
	}

	return &CoinbaseTx
*/
}

func createSpendTxForTest(outpoints []*wire.OutPoint, amount soterutil.Amount, fee soterutil.Amount) *wire.MsgTx {
	spendTx := wire.NewMsgTx(1)

	for _, outpoint := range outpoints {
		spendTx.AddTxIn(&wire.TxIn{
			PreviousOutPoint: *outpoint,
			Sequence:         wire.MaxTxInSequenceNum,
			SignatureScript:  nil,
		})
	}
	spendTx.AddTxOut(wire.NewTxOut(int64(amount-fee),
		opTrueScript))
	spendTx.AddTxOut(wire.NewTxOut(0, uniqueOpReturnScript()))

	return spendTx
}

func addBlockForTest(dag *BlockDAG, msgBlock *wire.MsgBlock) (bool, error) {
	block := soterutil.NewBlock(msgBlock)
	_, isOrphan, err := dag.ProcessBlock(block, BFNone)
	if err != nil {
		return false, err
	}
	return isOrphan, nil
}

var CoinbaseTx = wire.MsgTx{
	Version: 1,
	TxIn: []*wire.TxIn{
		{
			PreviousOutPoint: wire.OutPoint{
				Hash:  chainhash.Hash{},
				Index: 0xffffffff,
			},
			SignatureScript: []byte{
				0x03, 0x01, 0x00, 0x00,
			},
			Sequence: 0xffffffff,
		},
	},
	TxOut: []*wire.TxOut{
		{
			Value: 0x12a05f200, // 5000000000
			PkScript: []byte{
				0x41, // OP_DATA_65
				0x04, 0x1b, 0x0e, 0x8c, 0x25, 0x67, 0xc1, 0x25,
				0x36, 0xaa, 0x13, 0x35, 0x7b, 0x79, 0xa0, 0x73,
				0xdc, 0x44, 0x44, 0xac, 0xb8, 0x3c, 0x4e, 0xc7,
				0xa0, 0xe2, 0xf9, 0x9d, 0xd7, 0x45, 0x75, 0x16,
				0xc5, 0x81, 0x72, 0x42, 0xda, 0x79, 0x69, 0x24,
				0xca, 0x4e, 0x99, 0x94, 0x7d, 0x08, 0x7f, 0xed,
				0xf9, 0xce, 0x46, 0x7c, 0xb9, 0xf7, 0xc6, 0x28,
				0x70, 0x78, 0xf8, 0x01, 0xdf, 0x27, 0x6f, 0xdf,
				0x84, // 65-byte signature
				0xac, // OP_CHECKSIG
			},
		},
	},
	LockTime: 0,
}

var orphanParentHash = chainhash.Hash([32]byte{ // Make go vet happy.
	0x50, 0x12, 0x01, 0x19, 0x17, 0x2a, 0x61, 0x04,
	0x21, 0xa6, 0xc3, 0x01, 0x1d, 0xd3, 0x30, 0xd9,
	0xdf, 0x07, 0xb6, 0x36, 0x16, 0xc2, 0xcc, 0x1f,
	0x1c, 0xd0, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00,
}) // 000000000002d01c1fccc21636b607dfd930d31d01c3a62104612a1719011250

//orphan block hash: 7be5a3b0798c3a55dc02ae3714acd9439c719ba436804189d679d20bd072498a
var orphanPrevHash = GenerateTipsHash([]*chainhash.Hash{&orphanParentHash})
var BlockOrphan = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version: 1,
		PrevBlock: *orphanPrevHash,
		MerkleRoot: CoinbaseTx.TxHash(),
		Timestamp: time.Unix(1543949845, 0), // 2018-12-04 18:57:25
		Bits:      0x1010000,               // 1
		Nonce:     0x00000001,
	},
	Parents: wire.ParentSubHeader{
		Version: 1,
		Parents: []*wire.Parent{ {Hash: orphanParentHash} },
		Size:    1,
	},
	Verification: wire.VerificationSubHeader{
		Version: 1,
		Size: 0,
		CycleNonces: []uint32{},
	},
	Transactions: []*wire.MsgTx{&CoinbaseTx},
}

// TestHaveBlock tests the HaveBlock API to ensure proper functionality.
func TestHaveBlock(t *testing.T) {
	t.Parallel()
	// Create a new database and dag instance to run tests against.
	dag, teardownFunc, err := chainSetup("haveblock",
		&chaincfg.SimNetParams)
	if err != nil {
		t.Errorf("Failed to setup dag instance: %v", err)
		return
	}
	defer teardownFunc()

	// Since we're not dealing with the real block dag, set the coinbase
	// maturity to 1.
	dag.TstSetCoinbaseMaturity(1)

	// create blocks
	msgblock1 := createMsgBlockForTest(1,
		time.Now().Unix(),
		[]*wire.MsgBlock{chaincfg.SimNetParams.GenesisBlock},
		nil)
	isOrphan, err := addBlockForTest(dag, msgblock1)
	if err != nil {
		t.Errorf("Error adding block %v\n", err)
	}
	if isOrphan {
		t.Errorf("ProcessBlock incorrectly returned block1"+
			"is an orphan\n")
		return
	}

	// The orphan block needs to be solved, before we'll be able to add it to the dag
	solveMsgBlockForTest(&BlockOrphan)

	isOrphan, err = addBlockForTest(dag, &BlockOrphan)
	if err != nil {
		t.Errorf("Error adding block %v\n", err)
	}
	if !isOrphan {
		t.Errorf("ProcessBlock block should be an orphan\n")
		return
	}

	tests := []struct {
		hash string
		want bool
	}{
		// Genesis block should be present.
		{hash: chaincfg.SimNetParams.GenesisHash.String(), want: true},

		// Block 1 should be present.
		{hash: msgblock1.BlockHash().String(), want: true},

		// Block 100000 should be present (as an orphan).
		{hash: BlockOrphan.BlockHash().String(), want: true},

		// Random hashes should not be available.
		{hash: "123", want: false},
	}

	for i, test := range tests {
		hash, err := chainhash.NewHashFromStr(test.hash)
		if err != nil {
			t.Errorf("NewHashFromStr: %v", err)
			continue
		}

		result, err := dag.HaveBlock(hash)
		if err != nil {
			t.Errorf("HaveBlock #%d unexpected error: %v", i, err)
			return
		}
		if result != test.want {
			t.Errorf("HaveBlock #%d got %v want %v", i, result,
				test.want)
			continue
		}
	}
}

// test block connected correctly, tip set updated accordingly
func TestDAGSnapshot(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping due to -test.short flag")
	}
	// Create a new database and dag instance to run tests against.
	dag, teardownFunc, err := chainSetup("dagsnapshot",
		&chaincfg.SimNetParams)
	if err != nil {
		t.Errorf("Failed to setup dag instance: %v", err)
		return
	}
	defer teardownFunc()

	// Since we're not dealing with the real block dag, set the coinbase
	// maturity to 1.
	dag.TstSetCoinbaseMaturity(1)

	// new dag with only genesis block
	snapshot := dag.DAGSnapshot()
	tipHashes := snapshot.Tips

	if tipHashes == nil || len(tipHashes) != 1 {
		t.Errorf("DAGSnapshot expecting one tip\n")
	}
	if !tipHashes[0].IsEqual(chaincfg.SimNetParams.GenesisHash) {
		t.Errorf("DAGSnapshot expecting one tip %s: got %s\n", chaincfg.SimNetParams.GenesisHash, tipHashes[0])
	}

	//create blocks
	now := time.Now().Unix()
	var blocks = make([]*wire.MsgBlock, 3)
	blocks[0] = createMsgBlockForTest(1, now - 1000, []*wire.MsgBlock{chaincfg.SimNetParams.GenesisBlock}, nil)
	blocks[1] = createMsgBlockForTest(1, now - 800, []*wire.MsgBlock{chaincfg.SimNetParams.GenesisBlock}, nil)
	blocks[2] = createMsgBlockForTest(2, now - 600, []*wire.MsgBlock{blocks[0], blocks[1]}, nil)

	// block1 w/ parent block0
	block1Hash := blocks[0].BlockHash()
	_, err = addBlockForTest(dag, blocks[0])
	if err != nil {
		t.Fatalf("failed to add block %s: %s", blocks[0].BlockHash(), err)
	}

	snapshot = dag.DAGSnapshot()
	tipHashes = snapshot.Tips

	if tipHashes == nil || len(tipHashes) != 1 {
		t.Errorf("DAGSnapshot expecting one tip\n")
	}
	if !tipHashes[0].IsEqual(&block1Hash) {
		t.Errorf("DAGSnapshot expecting one tip %s: got %s\n", block1Hash, tipHashes[0])
	}

	// add block2 w/ parent block0
	block2Hash := blocks[1].BlockHash()
	addBlockForTest(dag, blocks[1])

	snapshot = dag.DAGSnapshot()
	tipHashes = snapshot.Tips

	if tipHashes == nil || len(tipHashes) != 2 {
		t.Errorf("DAGSnapshot expecting two tips\n")
	}

	tipSet := make(map[chainhash.Hash]struct{})
	for _, hash := range tipHashes {
		tipSet[hash] = struct{}{}
	}

	if _, ok := tipSet[block1Hash]; !ok {
		t.Errorf("DAGSnapshot tips should include block1: %v\n", block1Hash)
	}
	if _, ok := tipSet[block2Hash]; !ok {
		t.Errorf("DAGSnapshot tips should include block2: %v\n", block2Hash)
	}

	// add block3 with parents block1 and block2
	block3Hash := blocks[2].BlockHash()
	addBlockForTest(dag, blocks[2])

	snapshot = dag.DAGSnapshot()
	tipHashes = snapshot.Tips

	if tipHashes == nil || len(tipHashes) != 1 {
		t.Errorf("DAGSnapshot expecting one tip\n")
	}

	if !tipHashes[0].IsEqual(&block3Hash) {
		t.Errorf("DAGSnapshot expecting one tip %s: got %s\n", block3Hash, tipHashes[0])
	}

}

func TestGetOrphanRoot(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping due to -test.short flag")
	}
	dag, teardownFunc, err := chainSetup("getorphanroot",
		&chaincfg.SimNetParams)
	if err != nil {
		t.Errorf("Failed to setup dag instance: %v", err)
		return
	}
	defer teardownFunc()

	// Since we're not dealing with the real block dag, set the coinbase
	// maturity to 1.
	dag.TstSetCoinbaseMaturity(1)

	//create blocks
	now := time.Now().Unix()
	var blocks= make([]*wire.MsgBlock, 6)
	blocks[0] = createMsgBlockForTest(1, now-1000, []*wire.MsgBlock{chaincfg.SimNetParams.GenesisBlock}, nil)
	blocks[1] = createMsgBlockForTest(1, now-900, []*wire.MsgBlock{chaincfg.SimNetParams.GenesisBlock}, nil)
	blocks[2] = createMsgBlockForTest(2, now-800, []*wire.MsgBlock{blocks[0], blocks[1]}, nil)
	blocks[3] = createMsgBlockForTest(3, now-700, []*wire.MsgBlock{blocks[2]}, nil)
	blocks[4] = createMsgBlockForTest(3, now-600, []*wire.MsgBlock{blocks[2]}, nil)
	blocks[5] = createMsgBlockForTest(4, now-500, []*wire.MsgBlock{blocks[3], blocks[4]}, nil)

	isOrphan, _ := addBlockForTest(dag, blocks[5])
	block6Hash := blocks[5].BlockHash()
	fmt.Printf("Block 6 hash: %v\n", block6Hash)
	if !isOrphan {
		t.Errorf("ProcessBlock block 6 should be an orphan\n")
	}

	orphanRoots := dag.GetOrphanRoot(&block6Hash)
	if len(orphanRoots) != 1 {
		t.Errorf("GetOrphanRoot expected length 1 got %d\n", len(orphanRoots))
	}

	if len(orphanRoots) != 1 || !orphanRoots[0].IsEqual(&block6Hash) {
		t.Errorf("GetOrphanRoot block 6 expected to be orphan root\n")
	}

	// add parents block 4 and 5, still orphans
	isOrphan,_ = addBlockForTest(dag, blocks[4])
	block5Hash := blocks[4].BlockHash()
	fmt.Printf("Block 5 hash: %v\n", block5Hash)
	if !isOrphan {
		t.Errorf("ProcessBlock block 5 should be an orphan\n")
	}

	isOrphan,_ = addBlockForTest(dag, blocks[3])
	block4Hash := blocks[3].BlockHash()
	fmt.Printf("Block 4 hash: %v\n", block4Hash)
	if !isOrphan {
		t.Errorf("ProcessBlock block 4 should be an orphan\n")
	}

	orphanRoots = dag.GetOrphanRoot(&block6Hash)
	if len(orphanRoots) != 2 {
		fmt.Printf("orphanRoots: %v\n", orphanRoots)
		t.Errorf("GetOrphanRoot expected length 2 got %d\n", len(orphanRoots))
		return
	}

	hashSet := make(map[chainhash.Hash]struct{})
	for _, hash := range orphanRoots {
		hashSet[hash] = struct{}{}
	}

	if _, exists := hashSet[block4Hash]; !exists {
		t.Errorf("GetOrphanRoot expected block 4 in list\n")
	}

	if _, exists := hashSet[block5Hash]; !exists {
		t.Errorf("GetOrphanRoot expected block 5 in list\n")
	}

	// add rest of blocks
	isOrphan,_ = addBlockForTest(dag, blocks[2])
	block3Hash := blocks[2].BlockHash()
	fmt.Printf("Block 3 hash: %v\n", block3Hash)
	if !isOrphan {
		t.Errorf("ProcessBlock block 3 should be an orphan\n")
	}

	addBlockForTest(dag, blocks[1])
	block2Hash := blocks[1].BlockHash()
	fmt.Printf("Block 2 hash: %v\n", block2Hash)

	addBlockForTest(dag, blocks[0])
	block1Hash := blocks[0].BlockHash()
	fmt.Printf("Block 1 hash: %v\n", block1Hash)

	//block 6 no longer orphan
	orphanRoots = dag.GetOrphanRoot(&block6Hash)
	if len(orphanRoots) != 0 {
		t.Errorf("GetOrphanRoot expected length 0 got %d\n", len(orphanRoots))
	}
}

func TestHeaderByHash(t *testing.T) {
	t.Parallel()
	dag := newFakeChain(&chaincfg.SimNetParams)
	now := time.Now().Unix()
	msgblock := createMsgBlockForTest(1, now-1000, []*wire.MsgBlock{chaincfg.SimNetParams.GenesisBlock}, nil)
	node := newBlockNode(&msgblock.Header, &msgblock.Parents, []*blockNode{dag.dView.Genesis()})
	dag.index.AddNode(node)

	header, err := dag.HeaderByHash(&node.hash)
	if err != nil {
		t.Errorf("HeaderByHash encountered an error: %v\n", err)
		return
	}
	if header != node.Header() {
		t.Errorf("HeaderByHash got wrong header, expected %v got %v", node.Header(), header)
	}
}

func TestMainChainHasBlock(t *testing.T) {
	t.Parallel()
	dag := newFakeChain(&chaincfg.SimNetParams)
	now := time.Now().Unix()
	msgblock := createMsgBlockForTest(1, now-1000, []*wire.MsgBlock{chaincfg.SimNetParams.GenesisBlock}, nil)
	node := newBlockNode(&msgblock.Header, &msgblock.Parents, []*blockNode{dag.dView.Genesis()})

	hasBlock := dag.MainChainHasBlock(&node.hash)
	if hasBlock {
		t.Errorf("Main chain should not have block %v", node.hash)
	}

	dag.index.AddNode(node)
	dag.dView.AddTip(node)

	hasBlock = dag.MainChainHasBlock(&node.hash)

	if !hasBlock {
		t.Errorf("Main chain should have block %v", node.hash)
	}
}

func TestBlockHeightByHash(t *testing.T) {
	t.Parallel()
	dag := newFakeChain(&chaincfg.SimNetParams)
	now := time.Now().Unix()
	msgblock := createMsgBlockForTest(1, now-1000, []*wire.MsgBlock{chaincfg.SimNetParams.GenesisBlock}, nil)
	node := newBlockNode(&msgblock.Header, &msgblock.Parents, []*blockNode{dag.dView.Genesis()})

	dag.index.AddNode(node)
	dag.dView.AddTip(node)

	height, err := dag.BlockHeightByHash(&node.hash)
	if err != nil {
		t.Errorf("BlockHeightByHash encountered an error: %v\n", err)
		return
	}

	if height != 1 {
		t.Errorf("BlockHeightByHash expecting height 1 for: %v\n", node.hash)
	}

}

func TestBlockHashesByHeight(t *testing.T) {
	t.Parallel()
	dag := newFakeChain(&chaincfg.SimNetParams)
	now := time.Now().Unix()

	msgblock1 := createMsgBlockForTest(1, now-1000, []*wire.MsgBlock{chaincfg.SimNetParams.GenesisBlock}, nil)
	node1 := newBlockNode(&msgblock1.Header, &msgblock1.Parents, []*blockNode{dag.dView.Genesis()})
	msgblock2 := createMsgBlockForTest(1, now-800, []*wire.MsgBlock{chaincfg.SimNetParams.GenesisBlock}, nil)
	node2 := newBlockNode(&msgblock2.Header, &msgblock2.Parents, []*blockNode{dag.dView.Genesis()})
	msgblock3 := createMsgBlockForTest(2, now-600, []*wire.MsgBlock{ msgblock1 }, nil)
	node3 := newBlockNode(&msgblock3.Header, &msgblock3.Parents, []*blockNode{ node1 })

	dag.index.AddNode(node1)
	dag.dView.AddTip(node1)
	dag.index.AddNode(node2)
	dag.dView.AddTip(node2)
	dag.index.AddNode(node3)
	dag.dView.AddTip(node3)

	blocks, err := dag.BlockHashesByHeight(1)
	if err != nil {
		t.Errorf("BlockHHashesByHeight encountered an error: %v\n", err)
		return
	}
	if len(blocks) != 2 {
		fmt.Printf("BlockHashesByHeight expecting 2 blocks at height 1\n")
	}

}

func TestHeightRange(t *testing.T) {
	t.Parallel()
	dag := newFakeChain(&chaincfg.SimNetParams)
	now := time.Now().Unix()

	msgblock1 := createMsgBlockForTest(1, now-1000, []*wire.MsgBlock{chaincfg.SimNetParams.GenesisBlock}, nil)
	node1 := newBlockNode(&msgblock1.Header, &msgblock1.Parents, []*blockNode{dag.dView.Genesis()})
	msgblock2 := createMsgBlockForTest(1, now-800, []*wire.MsgBlock{chaincfg.SimNetParams.GenesisBlock}, nil)
	node2 := newBlockNode(&msgblock2.Header, &msgblock2.Parents, []*blockNode{dag.dView.Genesis()})
	msgblock3 := createMsgBlockForTest(2, now-600, []*wire.MsgBlock{ msgblock1 }, nil)
	node3 := newBlockNode(&msgblock3.Header, &msgblock3.Parents, []*blockNode{ node1 })

	dag.index.AddNode(node1)
	dag.dView.AddTip(node1)
	dag.index.AddNode(node2)
	dag.dView.AddTip(node2)
	dag.index.AddNode(node3)
	dag.dView.AddTip(node3)

	hashes := make([]chainhash.Hash, 4)
	hashes[0] = *chaincfg.SimNetParams.GenesisHash
	if msgblock1.BlockHash().String() < msgblock2.BlockHash().String() {
		hashes[1] = node1.hash
		hashes[2] = node2.hash
	} else {
		hashes[2] = node1.hash
		hashes[1] = node2.hash
	}

	hashes[3] = node3.hash

	tests := []struct {
		name        string
		startHeight int32            // starting height inclusive
		endHeight   int32            // end height exclusive
		hashes      []chainhash.Hash // expected located hashes
		expectError bool
	}{
		{
			name:        "blocks",
			startHeight: 1,
			endHeight:   3,
			hashes:      hashes[1:4],
		},
		{
			name:        "blocks past tip",
			startHeight: 1,
			endHeight:   10,
			hashes:      hashes[1:4],
		},
		{
			name:        "start height equals end height",
			startHeight: 1,
			endHeight:   1,
			hashes:      nil,
		},
		{
			name:        "invalid start height",
			startHeight: -1,
			endHeight:   10,
			expectError: true,
		},
		{
			name:        "end height less than start",
			startHeight: 2,
			endHeight:   0,
			expectError: true,
		},
	}
	for _, test := range tests {
		hashes, err := dag.HeightRange(test.startHeight, test.endHeight)
		if err != nil {
			if !test.expectError {
				t.Errorf("%s: unexpected error: %v", test.name, err)
			}
			continue
		}

		if !reflect.DeepEqual(hashes, test.hashes) {
			t.Errorf("%s: unexpected hashes -- got %v, want %v",
				test.name, hashes, test.hashes)
		}
	}
}

func TestHeightToHashRange(t *testing.T) {
	t.Parallel()
	dag := newFakeChain(&chaincfg.SimNetParams)
	now := time.Now().Unix()

	msgblock1 := createMsgBlockForTest(1, now-1000, []*wire.MsgBlock{chaincfg.SimNetParams.GenesisBlock}, nil)
	node1 := newBlockNode(&msgblock1.Header, &msgblock1.Parents, []*blockNode{dag.dView.Genesis()})
	msgblock2 := createMsgBlockForTest(1, now-800, []*wire.MsgBlock{chaincfg.SimNetParams.GenesisBlock}, nil)
	node2 := newBlockNode(&msgblock2.Header, &msgblock2.Parents, []*blockNode{dag.dView.Genesis()})
	msgblock3 := createMsgBlockForTest(2, now-600, []*wire.MsgBlock{msgblock1}, nil)
	node3 := newBlockNode(&msgblock3.Header, &msgblock3.Parents, []*blockNode{node1})
	msgblock4 := createMsgBlockForTest(3, now-400, []*wire.MsgBlock{msgblock2, msgblock3}, nil)
	node4 := newBlockNode(&msgblock4.Header, &msgblock4.Parents, []*blockNode{node2, node3})

	dag.index.AddNode(node1)
	dag.index.SetStatusFlags(node1, statusValid)
	dag.dView.AddTip(node1)
	dag.index.AddNode(node2)
	dag.index.SetStatusFlags(node2, statusValid)
	dag.dView.AddTip(node2)
	dag.index.AddNode(node3)
	dag.index.SetStatusFlags(node3, statusValid)
	dag.dView.AddTip(node3)
	dag.index.AddNode(node4)
	dag.dView.AddTip(node4)

	hashes := make([]chainhash.Hash, 5)
	hashes[0] = *chaincfg.SimNetParams.GenesisHash
	if msgblock1.BlockHash().String() < msgblock2.BlockHash().String() {
		hashes[1] = node1.hash
		hashes[2] = node2.hash
	} else {
		hashes[2] = node1.hash
		hashes[1] = node2.hash
	}

	hashes[3] = node3.hash
	hashes[4] = node4.hash

	tests := []struct {
		name        string
		startHeight int32            // locator for requested inventory
		endHash     chainhash.Hash   // stop hash for locator
		maxResults  int              // max to locate, 0 = wire const
		hashes      []chainhash.Hash // expected located hashes
		expectError bool
	}{
		{
			name:        "blocks below tip",
			startHeight: 1,
			endHash:     node3.hash,
			maxResults:  10,
			hashes:      hashes[1:4],
		},
		{
			name:        "invalid start height",
			startHeight: 19,
			endHash:     node4.hash,
			maxResults:  10,
			expectError: true,
		},
		{
			name:        "too many results",
			startHeight: 1,
			endHash:     node3.hash,
			maxResults:  2,
			expectError: true,
		},
		{
			name:        "unvalidated block",
			startHeight: 1,
			endHash:     node4.hash,
			maxResults:  10,
			expectError: true,
		},
	}
	for _, test := range tests {
		hashes, err := dag.HeightToHashRange(test.startHeight, &test.endHash,
			test.maxResults)
		if err != nil {
			if !test.expectError {
				t.Errorf("%s: unexpected error: %v", test.name, err)
			}
			continue
		}

		if !reflect.DeepEqual(hashes, test.hashes) {
			t.Errorf("%s: unxpected hashes -- got %v, want %v",
				test.name, hashes, test.hashes)
		}
	}
}

// jenlouie: commented out throwing this error for DAG-66
/*
func TestDAGParentHeightLimit(t *testing.T) {
	dag, teardownFunc, err := chainSetup("parentheightlimit",
		&chaincfg.SimNetParams)
	if err != nil {
		t.Errorf("Failed to setup dag instance: %v", err)
		return
	}
	defer teardownFunc()

	// Since we're not dealing with the real block dag, set the coinbase
	// maturity to 1.
	dag.TstSetCoinbaseMaturity(1)

	//create blocks
	now := time.Now().Unix()
	var blocks = make([]*wire.MsgBlock, maxGenerationDifference + 1)
	numBlocks := len(blocks)
	blocks[0] = chaincfg.SimNetParams.GenesisBlock
	for i := 1; i <= maxGenerationDifference; i++ {
		blocks[i] = createMsgBlockForTest(uint32(i), now-int64((numBlocks-i)*10), []*wire.MsgBlock{blocks[i-1]}, nil)
		addBlockForTest(dag, blocks[i], t)
	}

	badBlock := createMsgBlockForTest(maxGenerationDifference + 1,
		now - 10,
		[]*wire.MsgBlock{blocks[maxGenerationDifference], chaincfg.SimNetParams.GenesisBlock}, nil)
	block := soterutil.NewBlock(badBlock)
	_, _, err = dag.ProcessBlock(block, BFNone)

	if err == nil {
		t.Errorf("Failed to reject block with parents past the generational limit: %v", err)
	}
}
*/

func TestGraphUpdate(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping due to -test.short flag")
	}
	dag, teardownFunc, err := chainSetup("graphupdate",
		&chaincfg.SimNetParams)
	if err != nil {
		t.Errorf("Failed to setup dag instance: %v", err)
		return
	}
	defer teardownFunc()

	// Since we're not dealing with the real block dag, set the coinbase
	// maturity to 1.
	dag.TstSetCoinbaseMaturity(1)

	//create blocks
	now := time.Now().Unix()
	var blocks = make([]*wire.MsgBlock, 3)
	numBlocks := len(blocks)
	blocks[0] = chaincfg.SimNetParams.GenesisBlock
	for i := 1; i <= numBlocks - 1; i++ {
		blocks[i] = createMsgBlockForTest(uint32(i), now-int64((numBlocks-i)*10), []*wire.MsgBlock{blocks[i-1]}, nil)
		_, err := addBlockForTest(dag, blocks[i])
		if err != nil {
			t.Fatalf("failed to add block to dag: %s", err)
		}
	}

	expected := ""
	for i := numBlocks - 1; i >= 0; i-- {
		block := blocks[i]
		expected += block.BlockHash().String()
		if i == 0 {
			expected += "\n"
		} else {
			expected += "->" + blocks[i-1].BlockHash().String() + "\n"
		}
	}

	graphStr := dag.graph.PrintGraph()
	if expected != graphStr{
		t.Errorf("Expected graph to be %s, got %s", expected, graphStr)
	}
}