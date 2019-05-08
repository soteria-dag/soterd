package blockdag

import (
	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/chaincfg/chainhash"
	"github.com/soteria-dag/soterd/soterutil"

	//"github.com/soteria-dag/soterd/soterutil"
	"github.com/soteria-dag/soterd/wire"
	"testing"
	"time"
)

// same txIn used by different txs, one in block A, one in block B with two different txOuts
func TestUTXOViewpointSimpleDoubleSpend(t *testing.T) {

	dag, teardownFunc, err := chainSetup("utxoviewpoint_simpledoublespend",
		&chaincfg.SimNetParams)
	if err != nil {
		t.Errorf("Failed to setup dag instance: %v", err)
		return
	}
	defer teardownFunc()

	// Since we're not dealing with the real block dag, set the coinbase
	// maturity to 1.
	dag.TstSetCoinbaseMaturity(1)

	block1 := createMsgBlockForTest(1,
		time.Now().Unix() - 1000,
		[]*wire.MsgBlock{chaincfg.SimNetParams.GenesisBlock},
		nil)
	addBlockForTest(dag, block1, t)

	block2 := createMsgBlockForTest(2,
		time.Now().Unix() - 900,
		[]*wire.MsgBlock{block1},
		nil)
	addBlockForTest(dag, block2, t)

	cbTx := block1.Transactions[0]
	cbTxHash := cbTx.TxHash()
	outpoint := wire.NewOutPoint(&cbTxHash, uint32(0))
	outpoints := []*wire.OutPoint{outpoint}

	tx := createSpendTxForTest(outpoints, soterutil.Amount(1000), soterutil.Amount(10))

	// block A using cb tx
	blockA := createMsgBlockForTest(3,
		time.Now().Unix(),
		[]*wire.MsgBlock{block2},
		[]*wire.MsgTx{tx})
	addBlockForTest(dag, blockA, t)

	tx2 := createSpendTxForTest(outpoints, soterutil.Amount(5000), soterutil.Amount(10))

	// block B using cb tx
	blockB := createMsgBlockForTest(3,
		time.Now().Unix(),
		[]*wire.MsgBlock{block2},
		[]*wire.MsgTx{tx2})

	_, err = addBlockForTest(dag, blockB, t)
	if err != nil {
		t.Errorf("Error adding block B: %v\n", err)
		return
	}

	// A before B
	order := []chainhash.Hash{
			chaincfg.SimNetParams.GenesisBlock.BlockHash(),
			block1.BlockHash(),
			block2.BlockHash(),
			blockA.BlockHash(),
			blockB.BlockHash()}

	newView := NewUtxoViewpoint()

	for _, hash := range order {
		soterBlock, err := dag.BlockByHash(&hash)
		if err != nil {
			t.Errorf("BlockByHash error: %v", err)
			return
		}
		err = newView.connectTransactionsForSorting(soterBlock, nil, dag.chainParams)
		if err != nil {
			t.Errorf("Connect transactions for sorting error: %v", err)
			return
		}
	}

	// output tx from blockA should be in view as spendable
	txHash := tx.TxHash()
	blockAOutpoint := wire.NewOutPoint(&txHash, 0)
	entryA := newView.LookupEntry(*blockAOutpoint)
	if entryA == nil {
		t.Errorf("Block A tx output should exist in view")
	} else if entryA.IsSpent() || entryA.IsIgnored() {
		t.Errorf("Block A tx output should be spendable in view")
	}

	// output tx from blockB should be in view as ignored
	tx2Hash := tx2.TxHash()
	blockBOutpoint := wire.NewOutPoint(&tx2Hash, 0)
	entryB := newView.LookupEntry(*blockBOutpoint)
	if entryB == nil {
		t.Errorf("Block B tx output should exist in view")
	} else if !entryB.IsIgnored() {
		t.Errorf("Block B tx output should be ignored")
	}

	// B before A
	order = []chainhash.Hash{
		chaincfg.SimNetParams.GenesisBlock.BlockHash(),
		block1.BlockHash(),
		block2.BlockHash(),
		blockB.BlockHash(),
		blockA.BlockHash()}

	newView = NewUtxoViewpoint()

	for _, hash := range order {
		soterBlock, err := dag.BlockByHash(&hash)
		if err != nil {
			t.Errorf("BlockByHash error: %v", err)
			return
		}
		err = newView.connectTransactionsForSorting(soterBlock, nil, dag.chainParams)
		if err != nil {
			t.Errorf("Connect transactions for sorting error: %v", err)
			return
		}
	}

	// output tx from blockB should be in view as spendable
	entryB = newView.LookupEntry(*blockBOutpoint)
	if entryB == nil {
		t.Errorf("Block B tx output should exist in view")
	} else if entryB.IsSpent() || entryB.IsIgnored() {
		t.Errorf("Block B tx output should be spendable in view")
	}

	// output tx from blockA should be in view as ignored
	entryA = newView.LookupEntry(*blockAOutpoint)
	if entryA == nil {
		t.Errorf("Block A tx output should exist in view")
	} else if !entryA.IsIgnored() {
		t.Errorf("Block A tx output should be ignored")
	}
}

// add same tx to 2 blocks of same height
func TestUTXOViewpointDuplicateTx(t *testing.T) {
	dag, teardownFunc, err := chainSetup("utxoviewpoint_duplicatetx",
		&chaincfg.SimNetParams)
	if err != nil {
		t.Errorf("Failed to setup dag instance: %v", err)
		return
	}
	defer teardownFunc()

	// Since we're not dealing with the real block dag, set the coinbase
	// maturity to 1.
	dag.TstSetCoinbaseMaturity(1)

	block1 := createMsgBlockForTest(1,
		time.Now().Unix() - 1000,
		[]*wire.MsgBlock{chaincfg.SimNetParams.GenesisBlock},
		nil)
	addBlockForTest(dag, block1, t)

	block2 := createMsgBlockForTest(2,
		time.Now().Unix() - 900,
		[]*wire.MsgBlock{block1},
		nil)
	addBlockForTest(dag, block2, t)

	cbTx := block1.Transactions[0]
	cbTxHash := cbTx.TxHash()
	outpoint := wire.NewOutPoint(&cbTxHash, uint32(0))
	outpoints := []*wire.OutPoint{outpoint}

	tx := createSpendTxForTest(outpoints, soterutil.Amount(1000), soterutil.Amount(10))

	// block A using cb tx
	blockA := createMsgBlockForTest(3,
		time.Now().Unix() - 10,
		[]*wire.MsgBlock{block2},
		[]*wire.MsgTx{tx})
	addBlockForTest(dag, blockA, t)

	// block B using cb tx
	blockB := createMsgBlockForTest(3,
		time.Now().Unix(),
		[]*wire.MsgBlock{block2},
		[]*wire.MsgTx{tx})

	_, err = addBlockForTest(dag, blockB, t)
	if err != nil {
		t.Errorf("Both block A and block B should be included in chain, error adding block B: %v\n", err)
		return
	}

	order := []chainhash.Hash{
		chaincfg.SimNetParams.GenesisBlock.BlockHash(),
		block1.BlockHash(),
		block2.BlockHash(),
		blockA.BlockHash(),
		blockB.BlockHash()}

	newView := NewUtxoViewpoint()

	for _, hash := range order {
		soterBlock, err := dag.BlockByHash(&hash)
		if err != nil {
			t.Errorf("BlockByHash error: %v", err)
			return
		}
		err = newView.connectTransactionsForSorting(soterBlock, nil, dag.chainParams)
		if err != nil {
			t.Errorf("Connect transactions for sorting error: %v", err)
			return
		}
	}

	// output tx from blockA should be in view as spendable
	txHash := tx.TxHash()
	blockAOutpoint := wire.NewOutPoint(&txHash, 0)
	entryA := newView.LookupEntry(*blockAOutpoint)
	if entryA == nil {
		t.Errorf("Duplicate tx output should exist in view")
	} else if entryA.IsSpent() || entryA.IsIgnored() {
		t.Errorf("Duplicate tx output should be spendable in view")
	}
}

// txs that dependent on rejects double spend tx should also be rejected
func TestUTXOViewpointChildOfDoubleSpend(t *testing.T) {
	dag, teardownFunc, err := chainSetup("utxoviewpoint_childofdoublespend",
		&chaincfg.SimNetParams)
	if err != nil {
		t.Errorf("Failed to setup dag instance: %v", err)
		return
	}
	defer teardownFunc()

	// Since we're not dealing with the real block dag, set the coinbase
	// maturity to 1.
	dag.TstSetCoinbaseMaturity(1)

	block1 := createMsgBlockForTest(1,
		time.Now().Unix() - 1000,
		[]*wire.MsgBlock{chaincfg.SimNetParams.GenesisBlock},
		nil)
	addBlockForTest(dag, block1, t)

	block2 := createMsgBlockForTest(2,
		time.Now().Unix() - 900,
		[]*wire.MsgBlock{block1},
		nil)
	addBlockForTest(dag, block2, t)

	cbTx := block1.Transactions[0]
	cbTxHash := cbTx.TxHash()
	outpoint := wire.NewOutPoint(&cbTxHash, uint32(0))
	outpoints := []*wire.OutPoint{outpoint}

	tx := createSpendTxForTest(outpoints, soterutil.Amount(1000), soterutil.Amount(10))

	// block A using cb tx
	blockA := createMsgBlockForTest(3,
		time.Now().Unix() - 800,
		[]*wire.MsgBlock{block2},
		[]*wire.MsgTx{tx})
	addBlockForTest(dag, blockA, t)

	tx2 := createSpendTxForTest(outpoints, soterutil.Amount(5000), soterutil.Amount(10))
	tx2Hash := tx2.TxHash()
	// block B using cb tx
	blockB := createMsgBlockForTest(3,
		time.Now().Unix() - 700,
		[]*wire.MsgBlock{block2},
		[]*wire.MsgTx{tx2})

	_, err = addBlockForTest(dag, blockB, t)
	if err != nil {
		t.Errorf("Error adding block B: %v\n", err)
		return
	}

	// block C using double spend from B
	blockBOutpoint := wire.NewOutPoint(&tx2Hash, uint32(0))
	tx3 := createSpendTxForTest([]*wire.OutPoint{blockBOutpoint}, soterutil.Amount(1000), soterutil.Amount(10))
	blockC := createMsgBlockForTest(4,
		time.Now().Unix() - 600,
		[]*wire.MsgBlock{blockB},
		[]*wire.MsgTx{tx3})

	_, err = addBlockForTest(dag, blockC, t)
	if err != nil {
		t.Errorf("Error adding block C: %v\n", err)
		return
	}

	// A before B
	order := []chainhash.Hash{
		chaincfg.SimNetParams.GenesisBlock.BlockHash(),
		block1.BlockHash(),
		block2.BlockHash(),
		blockA.BlockHash(),
		blockB.BlockHash(),
		blockC.BlockHash()}

	newView := NewUtxoViewpoint()

	for _, hash := range order {
		soterBlock, err := dag.BlockByHash(&hash)
		if err != nil {
			t.Errorf("BlockByHash error: %v", err)
			return
		}
		err = newView.connectTransactionsForSorting(soterBlock, nil, dag.chainParams)
		if err != nil {
			t.Errorf("Connect transactions for sorting error: %v", err)
			return
		}
	}

	entryB := newView.LookupEntry(*blockBOutpoint)
	if entryB == nil {
		t.Errorf("Block B tx output should exist in view")
	} else if !entryB.IsIgnored() {
		t.Errorf("Block B tx output should be ignored")
	}

	tx3Hash := tx3.TxHash()
	blockCOutpoint := wire.NewOutPoint(&tx3Hash, 0)
	entryC := newView.LookupEntry(*blockCOutpoint)
	if entryC == nil {
		t.Errorf("Block C tx output should exist in view")
	} else if !entryC.IsIgnored() {
		t.Errorf("Block C tx output should be ignored")
	}

	//TODO: B before A
}

// txIns that are inputs along with a double spend input should
// remain valid and spendable when double spend input is rejected
func TestUtxoViewpointCoInputOfDoubleSpend(t *testing.T) {
	dag, teardownFunc, err := chainSetup("utxoviewpoint_coinputofdoublespend",
		&chaincfg.SimNetParams)
	if err != nil {
		t.Errorf("Failed to setup dag instance: %v", err)
		return
	}
	defer teardownFunc()

	// Since we're not dealing with the real block dag, set the coinbase
	// maturity to 1.
	dag.TstSetCoinbaseMaturity(1)

	block1 := createMsgBlockForTest(1,
		time.Now().Unix() - 1000,
		[]*wire.MsgBlock{chaincfg.SimNetParams.GenesisBlock},
		nil)
	addBlockForTest(dag, block1, t)

	block2 := createMsgBlockForTest(2,
		time.Now().Unix() - 900,
		[]*wire.MsgBlock{block1},
		nil)
	addBlockForTest(dag, block2, t)

	block3 := createMsgBlockForTest(3,
		time.Now().Unix() - 800,
		[]*wire.MsgBlock{block2},
		nil)
	addBlockForTest(dag, block3, t)


	cbTx := block1.Transactions[0]
	cbTxHash := cbTx.TxHash()
	cbOutpoint := wire.NewOutPoint(&cbTxHash, uint32(0))

	cbTx2 := block2.Transactions[0]
	cbTx2Hash := cbTx2.TxHash()
	cbOutpoint2 := wire.NewOutPoint(&cbTx2Hash, uint32(0))

	tx := createSpendTxForTest([]*wire.OutPoint{cbOutpoint}, soterutil.Amount(1000), soterutil.Amount(10))

	// block A using cb tx
	blockA := createMsgBlockForTest(4,
		time.Now().Unix() - 700,
		[]*wire.MsgBlock{block3},
		[]*wire.MsgTx{tx})
	addBlockForTest(dag, blockA, t)

	tx2 := createSpendTxForTest([]*wire.OutPoint{cbOutpoint, cbOutpoint2}, soterutil.Amount(5000), soterutil.Amount(10))
	//tx2Hash := tx2.TxHash()
	// block B using cb tx
	blockB := createMsgBlockForTest(4,
		time.Now().Unix() - 600,
		[]*wire.MsgBlock{block3},
		[]*wire.MsgTx{tx2})

	_, err = addBlockForTest(dag, blockB, t)
	if err != nil {
		t.Errorf("Error adding block B: %v\n", err)
		return
	}

	order := []chainhash.Hash{
		chaincfg.SimNetParams.GenesisBlock.BlockHash(),
		block1.BlockHash(),
		block2.BlockHash(),
		blockA.BlockHash(),
		blockB.BlockHash()}

	newView := NewUtxoViewpoint()

	for _, hash := range order {
		soterBlock, err := dag.BlockByHash(&hash)
		if err != nil {
			t.Errorf("BlockByHash error: %v", err)
			return
		}
		err = newView.connectTransactionsForSorting(soterBlock, nil, dag.chainParams)
		if err != nil {
			t.Errorf("Connect transactions for sorting error: %v", err)
			return
		}
	}

	// double spend tx should not go through,
	// input should be unspent
	entryCB2 := newView.LookupEntry(*cbOutpoint2)
	if entryCB2 == nil {
		t.Errorf("CB from block 2 output should exist in view")
	} else if entryCB2.IsSpent() || entryCB2.IsIgnored() {
		t.Errorf("CB from block 2 not spent and should be spendable in view")
	}
}