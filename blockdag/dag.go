// Copyright (c) 2013-2018 The btcsuite developers
// Copyright (c) 2015-2018 The Decred developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdag

import (
	"container/list"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/soteria-dag/soterd/blockdag/phantom"
	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/chaincfg/chainhash"
	"github.com/soteria-dag/soterd/database"
	"github.com/soteria-dag/soterd/soterutil"
	"github.com/soteria-dag/soterd/txscript"
	"github.com/soteria-dag/soterd/wire"
)

const (
	// maxOrphanBlocks is the maximum number of orphan blocks that can be
	// queued.
	maxOrphanBlocks = 700
	// maxGenerationDifference is how many generations ago we allow parents of a DAG block to be
	// expect a parent to be within max generations
	maxGenerationDifference = 70

	// coloring and sorting k form phantom paper
	coloringK = 3
)

// BlockLocator is used to help locate specific blocks. The locator
// is the height at which we expect to find the block(s) (the first element in the array)
//
// It's represented as an array of *int32, so that functions that attempt to create
// a locator can return nil (which is valid for an uninitialized array) to represent
// that we weren't able to locate the block.
type BlockLocator []*int32

// orphanBlock represents a block that we don't yet have the parent for.  It
// is a normal block plus an expiration time to prevent caching the orphan
// forever.
type orphanBlock struct {
	block      *soterutil.Block
	expiration time.Time
}

// BestState houses information about the current best block and other info
// related to the state of the main chain as it exists from the point of view of
// the current best block.
//
// The BestSnapshot method can be used to obtain access to this information
// in a concurrent safe manner and the data will not be changed out from under
// the caller when chain state changes occur as the function name implies.
// However, the returned snapshot must be treated as immutable since it is
// shared by all callers.
type BestState struct {
	Hash        chainhash.Hash // The hash of the block.
	Height      int32          // The height of the block.
	Bits        uint32         // The difficulty bits of the block.
	BlockSize   uint64         // The size of the block.
	BlockWeight uint64         // The weight of the block.
	NumTxns     uint64         // The number of txns in the block.
	TotalTxns   uint64         // The total number of txns in the chain.
	MedianTime  time.Time      // Median time as per CalcPastMedianTime.
}

// newBestState returns a new best stats instance for the given parameters.
func newBestState(node *blockNode, blockSize, blockWeight, numTxns,
	totalTxns uint64, medianTime time.Time) *BestState {

	return &BestState{
		Hash:        node.hash,
		Height:      node.height,
		Bits:        node.bits,
		BlockSize:   blockSize,
		BlockWeight: blockWeight,
		NumTxns:     numTxns,
		TotalTxns:   totalTxns,
		MedianTime:  medianTime,
	}
}

type DAGState struct {
	Tips      []chainhash.Hash // Hash of the tip blocks
	Hash      chainhash.Hash //Hash of the tip hashes
	MinHeight int32
	MaxHeight int32
	BlkCount  uint32
}

func newDAGState(tips []*blockNode, blkCount uint32) *DAGState {
	tipHashes := make([]chainhash.Hash, len(tips))
	i := 0
	var maxHeight int32 = math.MinInt32
	var minHeight int32 = math.MaxInt32
	for _, tip := range tips {
		tipHashes[i] = tip.hash
		i++
		if tip.height > maxHeight {
			maxHeight = tip.height
		}
		if tip.height < minHeight {
			minHeight = tip.height
		}
	}

	hash := generateTipsHash(tips)

	return &DAGState{
		Tips: tipHashes,
		Hash: *hash,
		MinHeight: minHeight,
		MaxHeight: maxHeight,
		BlkCount: blkCount,
	}
}

// BlockDAG provides functions for working with the soter block directed acyclic graph.
// It includes functionality such as rejecting duplicate blocks, ensuring blocks
// follow all rules, orphan handling, checkpoint handling, and best chain
// selection with reorganization.
type BlockDAG struct {
	// The following fields are set when the instance is created and can't
	// be changed afterwards, so there is no need to protect them with a
	// separate mutex.
	//checkpoints         []chaincfg.Checkpoint
	//checkpointsByHeight map[int32]*chaincfg.Checkpoint
	db           database.DB
	chainParams  *chaincfg.Params
	timeSource   MedianTimeSource
	sigCache     *txscript.SigCache
	indexManager IndexManager
	hashCache    *txscript.HashCache

	// The following fields are calculated based upon the provided chain
	// parameters.  They are also set when the instance is created and
	// can't be changed afterwards, so there is no need to protect them with
	// a separate mutex.
	minRetargetTimespan int64 // target timespan / adjustment factor
	maxRetargetTimespan int64 // target timespan * adjustment factor
	blocksPerRetarget   int64 // target timespan / target time per block

	// chainLock protects concurrent access to the vast majority of the
	// fields in this struct below this point.
	chainLock sync.RWMutex

	// These fields are related to the memory block index.  They both have
	// their own locks, however they are often also protected by the chain
	// lock to help prevent logic races when blocks are being processed.
	//
	// index houses the entire block index in memory.
	index             *blockIndex
	dView             *dagView
	graph             *phantom.Graph
	blueSet           *phantom.BlueSetCache
	nodeOrder         []*chainhash.Hash
	orderCache        *phantom.OrderCache

	// These fields are related to handling of orphan blocks.  They are
	// protected by a combination of the chain lock and the orphan lock.
	orphanLock   sync.RWMutex
	orphans      map[chainhash.Hash]*orphanBlock
	prevOrphans  map[chainhash.Hash][]*orphanBlock
	oldestOrphan *orphanBlock

	// These fields are related to checkpoint handling.  They are protected
	// by the chain lock.
	//nextCheckpoint *chaincfg.Checkpoint
	//checkpointNode *blockNode

	// The state is used as a fairly efficient way to cache information
	// about the current best chain state that is returned to callers when
	// requested.  It operates on the principle of MVCC such that any time a
	// new block becomes the best block, the state pointer is replaced with
	// a new struct and the old state is left untouched.  In this way,
	// multiple callers can be pointing to different best chain states.
	// This is acceptable for most callers because the state is only being
	// queried at a specific point in time.
	//
	// In addition, some of the fields are stored in the database so the
	// chain state can be quickly reconstructed on load.
	stateLock     sync.RWMutex
	stateSnapshot *BestState
	dagSnapshot   *DAGState

	// The following caches are used to efficiently keep track of the
	// current deployment threshold state of each rule change deployment.
	//
	// This information is stored in the database so it can be quickly
	// reconstructed on load.
	//
	// warningCaches caches the current deployment threshold state for blocks
	// in each of the **possible** deployments.  This is used in order to
	// detect when new unrecognized rule changes are being voted on and/or
	// have been activated such as will be the case when older versions of
	// the software are being used
	//
	// deploymentCaches caches the current deployment threshold state for
	// blocks in each of the actively defined deployments.
	warningCaches    []thresholdStateCache
	deploymentCaches []thresholdStateCache

	// The following fields are used to determine if certain warnings have
	// already been shown.
	//
	// unknownRulesWarned refers to warnings due to unknown rules being
	// activated.
	//
	// unknownVersionsWarned refers to warnings due to unknown versions
	// being mined.
	unknownRulesWarned    bool
	unknownVersionsWarned bool

	// The notifications field stores a slice of callbacks to be executed on
	// certain blockchain events.
	notificationsLock sync.RWMutex
	notifications     []NotificationCallback
}

// visitOrphan helps to add orphans to the order by traversing down its parents until it looks like the dependencies
// should be added to the order.
func (b *BlockDAG) visitOrphan(hash *chainhash.Hash, order *[]chainhash.Hash, tempMark, done *map[chainhash.Hash]int) error {
	_, isDone := (*done)[*hash]
	if isDone {
		return nil
	}

	// Detect non-DAG graph (circular dependencies)
	_, inProgress := (*tempMark)[*hash]
	if inProgress {
		return fmt.Errorf("Non-DAG block connectivity detected!")
	}

	// Mark this hash
	(*tempMark)[*hash] = 0

	// Parents are ordered before this hash
	parents, hasParents := b.prevOrphans[*hash]
	if hasParents {
		for _, parent := range parents {
			err := b.visitOrphan(parent.block.Hash(), order, tempMark, done)
			if err != nil {
				return err
			}
		}
	}

	// We're done visiting this orphan
	(*done)[*hash] = 0
	*order = append(*order, *hash)

	return nil
}

// HaveBlock returns whether or not the chain instance has the block represented
// by the passed hash.  This includes checking the various places a block can
// be like part of the main chain, on a side chain, or in the orphan pool.
//
// This function is safe for concurrent access.
func (b *BlockDAG) HaveBlock(hash *chainhash.Hash) (bool, error) {
	exists, err := b.blockExists(hash)
	if err != nil {
		return false, err
	}
	return exists || b.IsKnownOrphan(hash), nil
}

// IsKnownOrphan returns whether the passed hash is currently a known orphan.
// Keep in mind that only a limited number of orphans are held onto for a
// limited amount of time, so this function must not be used as an absolute
// way to test if a block is an orphan block.  A full block (as opposed to just
// its hash) must be passed to ProcessBlock for that purpose.  However, calling
// ProcessBlock with an orphan that already exists results in an error, so this
// function provides a mechanism for a caller to intelligently detect *recent*
// duplicate orphans and react accordingly.
//
// This function is safe for concurrent access.
func (b *BlockDAG) IsKnownOrphan(hash *chainhash.Hash) bool {
	// Protect concurrent access.  Using a read lock only so multiple
	// readers can query without blocking each other.
	b.orphanLock.RLock()
	_, exists := b.orphans[*hash]
	b.orphanLock.RUnlock()

	return exists
}

// GetOrphanBlocks returns a list of the orphan blocks
func (b *BlockDAG) GetOrphanBlocks() []*soterutil.Block {
	// Protect concurrent access. Using a read lock only so multiple
	// readers can query without blocking each other.
	b.orphanLock.RLock()
	defer b.orphanLock.RUnlock()

	var blocks []*soterutil.Block
	for _, orphan := range b.orphans {
		blocks = append(blocks, orphan.block)
	}

	return blocks
}

// GetOrphanChildren returns a list of orphans that are children of the given hash
func (b *BlockDAG) GetOrphanChildren(parent *chainhash.Hash) []*soterutil.Block {
	// Protect concurrent access. Using a read lock only so multiple
	// readers can query without blocking each other.
	b.orphanLock.RLock()
	defer b.orphanLock.RUnlock()

	var children []*soterutil.Block
	for _, orphan := range b.orphans {
		if orphan.block.MsgBlock().Parents.IsParent(parent) {
			children = append(children, orphan.block)
		}
	}

	return children
}

// GetOrphanLocator returns a BlockLocator with a height of (lowest orphan height - 2).
//
// -2 is used because locator height means "start at blocks _after_ this height". This makes sense when the desired
// height is the maxHeight of your DAG tip. Since we want to start with the lowest orphan's parent, we need -1 for the
// parent's height and another -1 to say "start at the parent" (instead of the block after the parent).
//
// The minimum height returned is zero.
func (b *BlockDAG) GetOrphanLocator(hashes []chainhash.Hash) BlockLocator {
	// Protect concurrent access. Using a read lock only so multiple
	// readers can query without blocking each other.
	b.orphanLock.RLock()
	defer b.orphanLock.RUnlock()

	// How far from the lowest orphan height we want to set the locator height at
	locatorDistance := int32(2)

	var minHeight int32
	heightSet := false
	for _, hash := range hashes {
		orphan, exists := b.orphans[hash]
		if !exists {
			continue
		}

		height := orphan.block.Height()
		header := &orphan.block.MsgBlock().Header
		if height == soterutil.BlockHeightUnknown {
			coinbaseTxs := orphan.block.Transactions()
			if !ShouldHaveSerializedBlockHeight(header) || len(coinbaseTxs) == 0 {
				// Orphan height isn't already known, and we can't determine it based on the contents
				// of the block (because the version is too low or there's no coinbase transactions to examine)
				continue
			} else {
				// If the block header version is 2+, we can try extracting the block height
				// from the scriptSig of the first coinbase transaction.
				cbHeight, err := ExtractCoinbaseHeight(coinbaseTxs[0])
				if err != nil {
					log.Warnf("Unable to extract height from coinbase tx: %v", err)
					continue
				}
				height = cbHeight
			}
		}

		if !heightSet {
			minHeight = height
			heightSet = true
		} else if height < minHeight {
			minHeight = height
		}
	}

	locatorHeight := minHeight - locatorDistance
	if locatorHeight < int32(0) {
		locatorHeight = int32(0)
	}

	locator := b.BlockLocatorFromHeight(locatorHeight)
	return locator
}

// GetOrphanOrder returns a list of orphans in the order they should be processed, based on topological sort using
// Depth-first search
func (b *BlockDAG) GetOrphanOrder() ([]chainhash.Hash, error) {
	// Protect concurrent access.  Using a read lock only so multiple
	// readers can query without blocking each other.
	b.orphanLock.RLock()
	defer b.orphanLock.RUnlock()

	order := make([]chainhash.Hash, 0)
	// temp is used to track non-dag connections between blocks (circular dependencies)
	temp := make(map[chainhash.Hash]int)
	// done is used to track which hashes have been ordered already
	done := make(map[chainhash.Hash]int)

	for hash := range b.orphans {
		err := b.visitOrphan(&hash, &order, &temp, &done)
		if err != nil {
			return order, err
		}
	}

	return order, nil
}

// GetOrphanRoot returns the head of the chain for the provided hash from the
// map of orphan blocks.
//
// This function is safe for concurrent access. hashes returned
func (b *BlockDAG) GetOrphanRoot(hash *chainhash.Hash) []chainhash.Hash {
	// Protect concurrent access.  Using a read lock only so multiple
	// readers can query without blocking each other.
	b.orphanLock.RLock()
	defer b.orphanLock.RUnlock()

	// Keep looping while the parent of each orphaned block is
	// known and is an orphan itself.

	children := make(map[chainhash.Hash][]chainhash.Hash)
	orphanRootMap := make(map[chainhash.Hash]struct{})
	prevHashes := list.New()
	prevHashes.PushBack(*hash)

	for prevHashes.Len() > 0 {
		prevHash := prevHashes.Front().Value.(chainhash.Hash)
		prevHashes.Remove(prevHashes.Front())
		//fmt.Printf("removed from queue: %v\n", prevHash)
		orphan, exists := b.orphans[prevHash]
		if !exists {
			continue
		}

		// add to set of orphan roots
		orphanRootMap[prevHash] = struct{}{}
		//fmt.Printf("Added to root map: %v\n", prevHash)
		childNodes, childExists := children[prevHash]
		// remove children from set of orphan roots
		if childExists {
			for _, child := range childNodes {
				delete(orphanRootMap, child)
				//fmt.Printf("Removed from root map: %v\n", child)
			}
		}

		// add parents to list of hashes to check if orphan
		for _, parentHash := range orphan.block.MsgBlock().Parents.ParentHashes() {
			if _, exists := orphanRootMap[parentHash]; !exists {
				prevHashes.PushBack(parentHash)
			}
			if children[parentHash] == nil {
				children[parentHash] = make([]chainhash.Hash, 0)
			}
			children[parentHash] = append(children[parentHash], prevHash)
		}
	}

	// return set of orphan blocks
	orphanRoot := make([]chainhash.Hash, 0)
	for k := range orphanRootMap {
		//fmt.Printf("Adding to list of roots: %v\n", k)
		orphanRoot = append(orphanRoot, k)
	}

	return orphanRoot
}

// GetOrphans returns a list of orphanBlocks (slightly different from GetOrphanBlocks)
func (b *BlockDAG) GetOrphans() []*orphanBlock {
	// Protect concurrent access. Using a read lock only so multiple
	// readers can query without blocking each other.
	b.orphanLock.RLock()
	defer b.orphanLock.RUnlock()

	var orphans []*orphanBlock
	for _, orphan := range b.orphans {
		orphans = append(orphans, orphan)
	}

	return orphans
}

// removeOrphanBlock removes the passed orphan block from the orphan pool and
// previous orphan index.
func (b *BlockDAG) removeOrphanBlock(orphan *orphanBlock) {
	// Protect concurrent access.
	b.orphanLock.Lock()
	defer b.orphanLock.Unlock()

	// Remove the orphan block from the orphan pool.
	orphanHash := orphan.block.Hash()
	delete(b.orphans, *orphanHash)

	// Remove the reference from the previous orphan index too.  An indexing
	// for loop is intentionally used over a range here as range does not
	// reevaluate the slice on each iteration nor does it adjust the index
	// for the modified slice.
	for _, parentHash := range orphan.block.MsgBlock().Parents.ParentHashes() {
		orphans := b.prevOrphans[parentHash]
		for i := 0; i < len(orphans); i++ {
			hash := orphans[i].block.Hash()
			if hash.IsEqual(orphanHash) {
				copy(orphans[i:], orphans[i+1:])
				orphans[len(orphans)-1] = nil
				orphans = orphans[:len(orphans)-1]
				i--
			}
		}
		b.prevOrphans[parentHash] = orphans

		// Remove the map entry altogether if there are no longer any orphans
		// which depend on the parent hash.
		if len(b.prevOrphans[parentHash]) == 0 {
			delete(b.prevOrphans, parentHash)
		}
	}
}

// addOrphanBlock adds the passed block (which is already determined to be
// an orphan prior calling this function) to the orphan pool.  It lazily cleans
// up any expired blocks so a separate cleanup poller doesn't need to be run.
// It also imposes a maximum limit on the number of outstanding orphan
// blocks and will remove the oldest received orphan block if the limit is
// exceeded.
func (b *BlockDAG) addOrphanBlock(block *soterutil.Block) {
	// Remove expired orphan blocks.
	for _, oBlock := range b.orphans {
		if time.Now().After(oBlock.expiration) {
			b.removeOrphanBlock(oBlock)
			continue
		}

		// Update the oldest orphan block pointer so it can be discarded
		// in case the orphan pool fills up.
		if b.oldestOrphan == nil || oBlock.expiration.Before(b.oldestOrphan.expiration) {
			b.oldestOrphan = oBlock
		}
	}

	// Limit orphan blocks to prevent memory exhaustion.
	if len(b.orphans)+1 > maxOrphanBlocks {
		// Remove the oldest orphan to make room for the new one.
		b.removeOrphanBlock(b.oldestOrphan)
		b.oldestOrphan = nil
	}

	// Protect concurrent access.  This is intentionally done here instead
	// of near the top since removeOrphanBlock does its own locking and
	// the range iterator is not invalidated by removing map entries.
	b.orphanLock.Lock()
	defer b.orphanLock.Unlock()

	// Insert the block into the orphan map with an expiration time
	// 1 hour from now.
	expiration := time.Now().Add(time.Hour)
	oBlock := &orphanBlock{
		block:      block,
		expiration: expiration,
	}
	b.orphans[*block.Hash()] = oBlock

	// Add to previous hash lookup index for faster dependency lookups.
	for _, parentHash := range block.MsgBlock().Parents.ParentHashes() {
		//prevHash := &block.MsgBlock().Header.PrevBlock
		b.prevOrphans[parentHash] = append(b.prevOrphans[parentHash], oBlock)
	}
}

// SequenceLock represents the converted relative lock-time in seconds, and
// absolute block-height for a transaction input's relative lock-times.
// According to SequenceLock, after the referenced input has been confirmed
// within a block, a transaction spending that input can be included into a
// block either after 'seconds' (according to past median time), or once the
// 'BlockHeight' has been reached.
type SequenceLock struct {
	Seconds     int64
	BlockHeight int32
}

// CalcSequenceLock computes a relative lock-time SequenceLock for the passed
// transaction using the passed UtxoViewpoint to obtain the past median time
// for blocks in which the referenced inputs of the transactions were included
// within. The generated SequenceLock lock can be used in conjunction with a
// block height, and adjusted median block time to determine if all the inputs
// referenced within a transaction have reached sufficient maturity allowing
// the candidate transaction to be included in a block.
//
// This function is safe for concurrent access.
func (b *BlockDAG) CalcSequenceLock(tx *soterutil.Tx, utxoView *UtxoViewpoint, mempool bool) (*SequenceLock, error) {
	b.chainLock.Lock()
	defer b.chainLock.Unlock()

	return b.calcSequenceLock(b.dView.Tips(), tx, utxoView, mempool)
}

// calcSequenceLock computes the relative lock-times for the passed
// transaction. See the exported version, CalcSequenceLock for further details.
//
// This function MUST be called with the chain state lock held (for writes).
func (b *BlockDAG) calcSequenceLock(nodes []*blockNode, tx *soterutil.Tx, utxoView *UtxoViewpoint, mempool bool) (*SequenceLock, error) {

	// A value of -1 for each relative lock type represents a relative time
	// lock value that will allow a transaction to be included in a block
	// at any given height or time. This value is returned as the relative
	// lock time in the case that BIP 68 is disabled, or has not yet been
	// activated.
	sequenceLock := &SequenceLock{Seconds: -1, BlockHeight: -1}

	// The sequence locks semantics are always active for transactions
	// within the mempool.
	csvSoftforkActive := mempool

	// If we're performing block validation, then we need to query the BIP9
	// state.
	if !csvSoftforkActive {
		// Obtain the latest BIP9 version bits state for the
		// CSV-package soft-fork deployment. The adherence of sequence
		// locks depends on the current soft-fork state.

		stateCount := 0
		activeCount := 0
		for _, node := range nodes {
			csvStates, err := b.deploymentStates(node.parents, chaincfg.DeploymentCSV)
			if err != nil {
				return nil, err
			}

			for _, state := range csvStates {
				stateCount++
				if state == ThresholdActive {
					activeCount++
				}
			}
		}
		csvSoftforkActive = stateCount == activeCount
	}

	// If the transaction's version is less than 2, and BIP 68 has not yet
	// been activated then sequence locks are disabled. Additionally,
	// sequence locks don't apply to coinbase transactions Therefore, we
	// return sequence lock values of -1 indicating that this transaction
	// can be included within a block at any given height or time.
	mTx := tx.MsgTx()
	sequenceLockActive := mTx.Version >= 2 && csvSoftforkActive
	if !sequenceLockActive || IsCoinBase(tx) {
		return sequenceLock, nil
	}

	// Grab the next height from the PoV of the passed blockNodes to use for
	// inputs present in the mempool.
	var highestParent int32
	for _, node := range nodes {
		maxHeight := node.parentsMaxHeight()
		if maxHeight > highestParent {
			highestParent = maxHeight
		}
	}
	nextHeight := highestParent + 1

	for txInIndex, txIn := range mTx.TxIn {
		utxo := utxoView.LookupEntry(txIn.PreviousOutPoint)
		if utxo == nil {
			str := fmt.Sprintf("output %v referenced from "+
				"transaction %s:%d either does not exist or "+
				"has already been spent", txIn.PreviousOutPoint,
				tx.Hash(), txInIndex)
			return sequenceLock, ruleError(ErrMissingTxOut, str)
		}

		// If the input height is set to the mempool height, then we
		// assume the transaction makes it into the next block when
		// evaluating its sequence blocks.
		inputHeight := utxo.BlockHeight()
		if inputHeight == 0x7fffffff {
			inputHeight = nextHeight
		}

		// Given a sequence number, we apply the relative time lock
		// mask in order to obtain the time lock delta required before
		// this input can be spent.
		sequenceNum := txIn.Sequence
		relativeLock := int64(sequenceNum & wire.SequenceLockTimeMask)

		switch {
		// Relative time locks are disabled for this input, so we can
		// skip any further calculation.
		case sequenceNum&wire.SequenceLockTimeDisabled == wire.SequenceLockTimeDisabled:
			continue
		case sequenceNum&wire.SequenceLockTimeIsSeconds == wire.SequenceLockTimeIsSeconds:
			// This input requires a relative time lock expressed
			// in seconds before it can be spent.  Therefore, we
			// need to query for the block prior to the one in
			// which this input was included within so we can
			// compute the past median time for the block prior to
			// the one which included this referenced output.
			prevInputHeight := inputHeight - 1
			if prevInputHeight < 0 {
				prevInputHeight = 0
			}

			var times []time.Time
			for _, node := range nodes {
				for _, blockNode := range node.Ancestors(prevInputHeight) {
					medianTime := blockNode.CalcPastMedianTime()
					times = append(times, medianTime)
				}
			}
			sort.Sort(ByTime(times))
			medianTime := times[len(times)/2]

			// Time based relative time-locks as defined by BIP 68
			// have a time granularity of RelativeLockSeconds, so
			// we shift left by this amount to convert to the
			// proper relative time-lock. We also subtract one from
			// the relative lock to maintain the original lockTime
			// semantics.
			timeLockSeconds := (relativeLock << wire.SequenceLockTimeGranularity) - 1
			timeLock := medianTime.Unix() + timeLockSeconds
			if timeLock > sequenceLock.Seconds {
				sequenceLock.Seconds = timeLock
			}
		default:
			// The relative lock-time for this input is expressed
			// in blocks so we calculate the relative offset from
			// the input's height as its converted absolute
			// lock-time. We subtract one from the relative lock in
			// order to maintain the original lockTime semantics.
			blockHeight := inputHeight + int32(relativeLock-1)
			if blockHeight > sequenceLock.BlockHeight {
				sequenceLock.BlockHeight = blockHeight
			}
		}
	}

	return sequenceLock, nil
}

// NOTE(cedric in DAG-30): Uncommented LockTimeToSequence due to dependency
// in integration/csv_fork_test.go
// (get rpc-related tests working after dag netsync changes)
// At time of comment this function isn't currently referenced anywhere other than tests.
//
// LockTimeToSequence converts the passed relative locktime to a sequence
// number in accordance to BIP-68.
// See: https://github.com/bitcoin/bips/blob/master/bip-0068.mediawiki
//  * (Compatibility)
func LockTimeToSequence(isSeconds bool, locktime uint32) uint32 {

	// If we're expressing the relative lock time in blocks, then the
	// corresponding sequence number is simply the desired input age.
	if !isSeconds {
		return locktime
	}

	// Set the 22nd bit which indicates the lock time is in seconds, then
	// shift the locktime over by 9 since the time granularity is in
	// 512-second intervals (2^9). This results in a max lock-time of
	// 33,553,920 seconds, or 1.1 years.
	return wire.SequenceLockTimeIsSeconds |
		locktime>>wire.SequenceLockTimeGranularity
}

// getReorganizeNodes finds the fork point between the main chain and the passed
// node and returns a list of block nodes that would need to be detached from
// the main chain and a list of block nodes that would need to be attached to
// the fork point (which will be the end of the main chain after detaching the
// returned list of block nodes) in order to reorganize the chain such that the
// passed node is the new end of the main chain.  The lists will be empty if the
// passed node is not on a side chain.
//
// This function may modify node statuses in the block index without flushing.
//
// This function MUST be called with the chain state lock held (for reads).
/*func (b *BlockChain) getReorganizeNodes(node *blockNode) (*list.List, *list.List) {
	attachNodes := list.New()
	detachNodes := list.New()

	// Do not reorganize to a known invalid chain. Ancestors deeper than the
	// direct parent are checked below but this is a quick check before doing
	// more unnecessary work.
	if b.index.NodeStatus(node.parent).KnownInvalid() {
		b.index.SetStatusFlags(node, statusInvalidAncestor)
		return detachNodes, attachNodes
	}

	// Find the fork point (if any) adding each block to the list of nodes
	// to attach to the main tree.  Push them onto the list in reverse order
	// so they are attached in the appropriate order when iterating the list
	// later.
	forkNode := b.bestChain.FindFork(node)
	invalidChain := false
	for n := node; n != nil && n != forkNode; n = n.parent {
		if b.index.NodeStatus(n).KnownInvalid() {
			invalidChain = true
			break
		}
		attachNodes.PushFront(n)
	}

	// If any of the node's ancestors are invalid, unwind attachNodes, marking
	// each one as invalid for future reference.
	if invalidChain {
		var next *list.Element
		for e := attachNodes.Front(); e != nil; e = next {
			next = e.Next()
			n := attachNodes.Remove(e).(*blockNode)
			b.index.SetStatusFlags(n, statusInvalidAncestor)
		}
		return detachNodes, attachNodes
	}

	// Start from the end of the main chain and work backwards until the
	// common ancestor adding each block to the list of nodes to detach from
	// the main chain.
	for n := b.bestChain.Tip(); n != nil && n != forkNode; n = n.parent {
		detachNodes.PushBack(n)
	}

	return detachNodes, attachNodes
}
*/

// Returns all input txns, even ones not spend b/c they have already been spent or are invalid.
// Needed for indexes.
func (b *BlockDAG) getAllInputTxos(block *soterutil.Block, view *UtxoViewpoint) ([]SpentTxOut, error) {
	stxos := make([]SpentTxOut, countSpentOutputs(block))
	index := 0
	for _, tx:= range block.Transactions() {
		if IsCoinBase(tx) {
			continue
		}
		for _, txIn := range tx.MsgTx().TxIn {
			entry := view.entries[txIn.PreviousOutPoint]

			if entry == nil {
				return nil, AssertError(fmt.Sprintf("view missing input %v",
					txIn.PreviousOutPoint))
			}

			var stxo = SpentTxOut{
				Amount:     entry.Amount(),
				PkScript:   entry.PkScript(),
				Height:     entry.BlockHeight(),
				IsCoinBase: entry.IsCoinBase(),
			}
			stxos[index] = stxo
			index++
		}
	}

	return stxos, nil
}

// connectBlock handles connecting the passed node/block to the end of the main
// (best) chain.
//
// This passed utxo view must have all referenced txos the block spends marked
// as spent and all of the new txos the block creates added to it.  In addition,
// the passed stxos slice must be populated with all of the information for the
// spent txos.  This approach is used because the connection validation that
// must happen prior to calling this function requires the same details, so
// it would be inefficient to repeat it.
//
// This function MUST be called with the chain state lock held (for writes).
func (b *BlockDAG) connectBlock(node *blockNode, block *soterutil.Block,
	view *UtxoViewpoint, stxos []SpentTxOut) error {

	// For DAG, the latest block received might not have tips as parents, but inner blocks in the graph

	// Make sure it's extending the end of the best chain.
	/*prevHash := &block.MsgBlock().Header.PrevBlock
	if !prevHash.IsEqual(&b.bestChain.Tip().hash) {
		return blockchain.AssertError("connectBlock must be called with a block " +
			"that extends the main chain")
	}
	*/

	// Sanity check the correct number of stxos are provided.
	//if len(stxos) != countSpentOutputs(block) {
	//	return AssertError("connectBlock called with inconsistent " +
//			"spent transaction out information")
//	}

	// No warnings about unknown rules or versions until the chain is
	// current.
	/*
		if b.isCurrent() {
			// Warn if any unknown new rules are either about to activate or
			// have already been activated.
			if err := b.warnUnknownRuleActivations(node); err != nil {
				return err
			}

			// Warn if a high enough percentage of the last blocks have
			// unexpected versions.
			if err := b.warnUnknownVersions(node); err != nil {
				return err
			}
		}
	*/

	// Write any block status changes to DB before updating best state.
	err := b.index.flushToDB()
	if err != nil {
		return err
	}

	// Generate a new best state snapshot that will be used to update the
	// database and later memory if all database updates are successful.
	b.stateLock.RLock()
	curTotalTxns := b.stateSnapshot.TotalTxns
	b.stateLock.RUnlock()
	numTxns := uint64(len(block.MsgBlock().Transactions))
	blockSize := uint64(block.MsgBlock().SerializeSize())
	blockWeight := uint64(GetBlockWeight(block))
	state := newBestState(node, blockSize, blockWeight, numTxns,
		curTotalTxns+numTxns, node.CalcPastMedianTime())

	curTotalBlks := b.dagSnapshot.BlkCount
	tips := b.dView.tips()
	// add new block
	dagTips := append(tips, node)
	// remove block's parents
	for _, parent := range node.parents {
		for j, node := range dagTips {
			if node == parent {
				copy(dagTips[j:], dagTips[j+1:])
				dagTips[len(dagTips)-1] = nil
				dagTips = dagTips[:len(dagTips)-1]
				break
			}
		}
	}

	dagState := newDAGState(dagTips, curTotalBlks + 1)
	newView := NewUtxoViewpoint()

	// Atomically insert info into the database.
	err = b.db.Update(func(dbTx database.Tx) error {
		// Update best block state.
		err := dbPutBestState(dbTx, state, node.workSum)
		if err != nil {
			return err
		}

		err = dbPutDAGState(dbTx, dagState)
		if err != nil {
			return err
		}

		// Add the block hash and height to the block index which tracks
		// the main chain.
		err = dbPutBlockIndex(dbTx, block.Hash(), node.height)
		if err != nil {
			return err
		}

		// add new node to graph
		var strHash = block.Hash().String()
		var nodeAdded = b.graph.AddNodeById(strHash)
		if !nodeAdded {
			//TODO: throw error
			log.Infof("Node not added to graph: %s", strHash)
		 }
		for _, parent := range block.MsgBlock().Parents.ParentHashes() {
			var edgeAdded = b.graph.AddEdgeById(strHash, parent.String())
			if !edgeAdded {
				//TODO: throw error
				log.Infof("Edge not added to graph, from %s to %s", strHash, parent.String())
			}
		}

		// sort blocks
		genesisHash := b.dView.Genesis().hash.String()
		_, sortOrder, err := phantom.OrderDAG(b.graph, b.graph.GetNodeById(genesisHash), coloringK, b.blueSet, dagState.MinHeight, b.orderCache)
		if err != nil {
			return err
		}

		// array to save sort order
		sortedHashes := make([]*chainhash.Hash, len(sortOrder))

		// generate new utxo set (from genesis to tips)
		// jenlouie: view will contain all tx, this might take too much space
		// might have to save utxo set to db, then load it back out every so often

		for i, node := range sortOrder {
			blockHash, err := chainhash.NewHashFromStr(node.GetId())
			if err != nil {
				return err
			}
			sortedHashes[i] = blockHash

			var soterBlock *soterutil.Block
			if block.Hash().IsEqual(blockHash) {
				soterBlock = block
			} else {
				soterBlock, err = b.BlockByHash(blockHash)
				if err != nil {
					return err
				}
			}

			err = newView.connectTransactionsForSorting(soterBlock, nil, b.chainParams)
			if err != nil {
				return err
			}
		}

		b.nodeOrder = sortedHashes

		//err = dbPutUtxoView(dbTx, view)
		err = dbPutUtxoView(dbTx, newView)
		if err != nil {
			return err
		}

		blockStxos, err := b.getAllInputTxos(block, newView)
		if err != nil {
			return err
		}

		// Update the transaction spend journal by adding a record for
		// the block that contains all txos spent by it.
		err = dbPutSpendJournalEntry(dbTx, block.Hash(), blockStxos)
		if err != nil {
			return err
		}

		// Allow the index manager to call each of the currently active
		// optional indexes with the block being connected so they can
		// update themselves accordingly.
		if b.indexManager != nil {
			err := b.indexManager.ConnectBlock(dbTx, block, blockStxos)
			if err != nil {
				return err
			}
		}

		return nil
	})
	
	if err != nil {
		// Remove the block from the graph
		b.graph.RemoveTipById(block.Hash().String())
		return err
	}

	// Prune fully spent entries and mark all entries in the view unmodified
	// now that the modifications have been committed to the database.
	//view.commit()
	newView.commit()

	// This node is now the end of the best chain.
	b.dView.AddTip(node)

	// Remove node's parents from tip set
	for _, parent := range node.parents {
		b.dView.RemoveTip(parent)
	}

	// Update the state for the best block.  Notice how this replaces the
	// entire struct instead of updating the existing one.  This effectively
	// allows the old version to act as a snapshot which callers can use
	// freely without needing to hold a lock for the duration.  See the
	// comments on the state variable for more details.
	b.stateLock.Lock()
	b.stateSnapshot = state
	b.dagSnapshot = dagState
	b.stateLock.Unlock()

	// Notify the caller that the block was connected to the main chain.
	// The caller would typically want to react with actions such as
	// updating wallets.
	b.chainLock.Unlock()
	b.sendNotification(NTBlockConnected, block)
	b.chainLock.Lock()

	return nil
}

// disconnectBlock handles disconnecting the passed node/block from the end of
// the main (best) chain.
//
// This function MUST be called with the chain state lock held (for writes).
/*func (b *BlockChain) disconnectBlock(node *blockNode, block *soterutil.Block, view *UtxoViewpoint) error {
	// Make sure the node being disconnected is the end of the best chain.
	if !node.hash.IsEqual(&b.bestChain.Tip().hash) {
		return AssertError("disconnectBlock must be called with the " +
			"block at the end of the main chain")
	}

	// Load the previous block since some details for it are needed below.
	prevNode := node.parent
	var prevBlock *soterutil.Block
	err := b.db.View(func(dbTx database.Tx) error {
		var err error
		prevBlock, err = dbFetchBlockByNode(dbTx, prevNode)
		return err
	})
	if err != nil {
		return err
	}

	// Write any block status changes to DB before updating best state.
	err = b.index.flushToDB()
	if err != nil {
		return err
	}

	// Generate a new best state snapshot that will be used to update the
	// database and later memory if all database updates are successful.
	b.stateLock.RLock()
	curTotalTxns := b.stateSnapshot.TotalTxns
	b.stateLock.RUnlock()
	numTxns := uint64(len(prevBlock.MsgBlock().Transactions))
	blockSize := uint64(prevBlock.MsgBlock().SerializeSize())
	blockWeight := uint64(GetBlockWeight(prevBlock))
	newTotalTxns := curTotalTxns - uint64(len(block.MsgBlock().Transactions))
	state := newBestState(prevNode, blockSize, blockWeight, numTxns,
		newTotalTxns, prevNode.CalcPastMedianTime())

	err = b.db.Update(func(dbTx database.Tx) error {
		// Update best block state.
		err := dbPutBestState(dbTx, state, node.workSum)
		if err != nil {
			return err
		}

		// Remove the block hash and height from the block index which
		// tracks the main chain.
		err = dbRemoveBlockIndex(dbTx, block.Hash(), node.height)
		if err != nil {
			return err
		}

		// Update the utxo set using the state of the utxo view.  This
		// entails restoring all of the utxos spent and removing the new
		// ones created by the block.
		err = dbPutUtxoView(dbTx, view)
		if err != nil {
			return err
		}

		// Before we delete the spend journal entry for this back,
		// we'll fetch it as is so the indexers can utilize if needed.
		stxos, err := dbFetchSpendJournalEntry(dbTx, block)
		if err != nil {
			return err
		}

		// Update the transaction spend journal by removing the record
		// that contains all txos spent by the block.
		err = dbRemoveSpendJournalEntry(dbTx, block.Hash())
		if err != nil {
			return err
		}

		// Allow the index manager to call each of the currently active
		// optional indexes with the block being disconnected so they
		// can update themselves accordingly.
		if b.indexManager != nil {
			err := b.indexManager.DisconnectBlock(dbTx, block, stxos)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Prune fully spent entries and mark all entries in the view unmodified
	// now that the modifications have been committed to the database.
	view.commit()

	// This node's parent is now the end of the best chain.
	b.bestChain.SetTip(node.parent)

	// Update the state for the best block.  Notice how this replaces the
	// entire struct instead of updating the existing one.  This effectively
	// allows the old version to act as a snapshot which callers can use
	// freely without needing to hold a lock for the duration.  See the
	// comments on the state variable for more details.
	b.stateLock.Lock()
	b.stateSnapshot = state
	b.stateLock.Unlock()

	// Notify the caller that the block was disconnected from the main
	// chain.  The caller would typically want to react with actions such as
	// updating wallets.
	b.chainLock.Unlock()
	b.sendNotification(NTBlockDisconnected, block)
	b.chainLock.Lock()

	return nil
}
*/

// countSpentOutputs returns the number of utxos the passed block spends.
func countSpentOutputs(block *soterutil.Block) int {
	// Exclude the coinbase transaction since it can't spend anything.
	var numSpent int
	for _, tx := range block.Transactions()[1:] {
		numSpent += len(tx.MsgTx().TxIn)
	}
	return numSpent
}

// reorganizeChain reorganizes the block chain by disconnecting the nodes in the
// detachNodes list and connecting the nodes in the attach list.  It expects
// that the lists are already in the correct order and are in sync with the
// end of the current best chain.  Specifically, nodes that are being
// disconnected must be in reverse order (think of popping them off the end of
// the chain) and nodes the are being attached must be in forwards order
// (think pushing them onto the end of the chain).
//
// This function may modify node statuses in the block index without flushing.
//
// This function MUST be called with the chain state lock held (for writes).
/*func (b *BlockChain) reorganizeChain(detachNodes, attachNodes *list.List) error {
	// Nothing to do if no reorganize nodes were provided.
	if detachNodes.Len() == 0 && attachNodes.Len() == 0 {
		return nil
	}

	// Ensure the provided nodes match the current best chain.
	tip := b.bestChain.Tip()
	if detachNodes.Len() != 0 {
		firstDetachNode := detachNodes.Front().Value.(*blockNode)
		if firstDetachNode.hash != tip.hash {
			return AssertError(fmt.Sprintf("reorganize nodes to detach are "+
				"not for the current best chain -- first detach node %v, "+
				"current chain %v", &firstDetachNode.hash, &tip.hash))
		}
	}

	// Ensure the provided nodes are for the same fork point.
	if attachNodes.Len() != 0 && detachNodes.Len() != 0 {
		firstAttachNode := attachNodes.Front().Value.(*blockNode)
		lastDetachNode := detachNodes.Back().Value.(*blockNode)
		if firstAttachNode.parent.hash != lastDetachNode.parent.hash {
			return AssertError(fmt.Sprintf("reorganize nodes do not have the "+
				"same fork point -- first attach parent %v, last detach "+
				"parent %v", &firstAttachNode.parent.hash,
				&lastDetachNode.parent.hash))
		}
	}

	// Track the old and new best chains heads.
	oldBest := tip
	newBest := tip

	// All of the blocks to detach and related spend journal entries needed
	// to unspend transaction outputs in the blocks being disconnected must
	// be loaded from the database during the reorg check phase below and
	// then they are needed again when doing the actual database updates.
	// Rather than doing two loads, cache the loaded data into these slices.
	detachBlocks := make([]*soterutil.Block, 0, detachNodes.Len())
	detachSpentTxOuts := make([][]SpentTxOut, 0, detachNodes.Len())
	attachBlocks := make([]*soterutil.Block, 0, attachNodes.Len())

	// Disconnect all of the blocks back to the point of the fork.  This
	// entails loading the blocks and their associated spent txos from the
	// database and using that information to unspend all of the spent txos
	// and remove the utxos created by the blocks.
	view := NewUtxoViewpoint()
	view.SetBestHash(&oldBest.hash)
	for e := detachNodes.Front(); e != nil; e = e.Next() {
		n := e.Value.(*blockNode)
		var block *soterutil.Block
		err := b.db.View(func(dbTx database.Tx) error {
			var err error
			block, err = dbFetchBlockByNode(dbTx, n)
			return err
		})
		if err != nil {
			return err
		}
		if n.hash != *block.Hash() {
			return AssertError(fmt.Sprintf("detach block node hash %v (height "+
				"%v) does not match previous parent block hash %v", &n.hash,
				n.height, block.Hash()))
		}

		// Load all of the utxos referenced by the block that aren't
		// already in the view.
		err = view.fetchInputUtxos(b.db, block)
		if err != nil {
			return err
		}

		// Load all of the spent txos for the block from the spend
		// journal.
		var stxos []SpentTxOut
		err = b.db.View(func(dbTx database.Tx) error {
			stxos, err = dbFetchSpendJournalEntry(dbTx, block)
			return err
		})
		if err != nil {
			return err
		}

		// Store the loaded block and spend journal entry for later.
		detachBlocks = append(detachBlocks, block)
		detachSpentTxOuts = append(detachSpentTxOuts, stxos)

		err = view.disconnectTransactions(b.db, block, stxos)
		if err != nil {
			return err
		}

		newBest = n.parent
	}

	// Set the fork point only if there are nodes to attach since otherwise
	// blocks are only being disconnected and thus there is no fork point.
	var forkNode *blockNode
	if attachNodes.Len() > 0 {
		forkNode = newBest
	}

	// Perform several checks to verify each block that needs to be attached
	// to the main chain can be connected without violating any rules and
	// without actually connecting the block.
	//
	// NOTE: These checks could be done directly when connecting a block,
	// however the downside to that approach is that if any of these checks
	// fail after disconnecting some blocks or attaching others, all of the
	// operations have to be rolled back to get the chain back into the
	// state it was before the rule violation (or other failure).  There are
	// at least a couple of ways accomplish that rollback, but both involve
	// tweaking the chain and/or database.  This approach catches these
	// issues before ever modifying the chain.
	for e := attachNodes.Front(); e != nil; e = e.Next() {
		n := e.Value.(*blockNode)

		var block *soterutil.Block
		err := b.db.View(func(dbTx database.Tx) error {
			var err error
			block, err = dbFetchBlockByNode(dbTx, n)
			return err
		})
		if err != nil {
			return err
		}

		// Store the loaded block for later.
		attachBlocks = append(attachBlocks, block)

		// Skip checks if node has already been fully validated. Although
		// checkConnectBlock gets skipped, we still need to update the UTXO
		// view.
		if b.index.NodeStatus(n).KnownValid() {
			err = view.fetchInputUtxos(b.db, block)
			if err != nil {
				return err
			}
			err = view.connectTransactions(block, nil)
			if err != nil {
				return err
			}

			newBest = n
			continue
		}

		// Notice the spent txout details are not requested here and
		// thus will not be generated.  This is done because the state
		// is not being immediately written to the database, so it is
		// not needed.
		//
		// In the case the block is determined to be invalid due to a
		// rule violation, mark it as invalid and mark all of its
		// descendants as having an invalid ancestor.
		err = b.checkConnectBlock(n, block, view, nil)
		if err != nil {
			if _, ok := err.(RuleError); ok {
				b.index.SetStatusFlags(n, statusValidateFailed)
				for de := e.Next(); de != nil; de = de.Next() {
					dn := de.Value.(*blockNode)
					b.index.SetStatusFlags(dn, statusInvalidAncestor)
				}
			}
			return err
		}
		b.index.SetStatusFlags(n, statusValid)

		newBest = n
	}

	// Reset the view for the actual connection code below.  This is
	// required because the view was previously modified when checking if
	// the reorg would be successful and the connection code requires the
	// view to be valid from the viewpoint of each block being connected or
	// disconnected.
	view = NewUtxoViewpoint()
	view.SetBestHash(&b.bestChain.Tip().hash)

	// Disconnect blocks from the main chain.
	for i, e := 0, detachNodes.Front(); e != nil; i, e = i+1, e.Next() {
		n := e.Value.(*blockNode)
		block := detachBlocks[i]

		// Load all of the utxos referenced by the block that aren't
		// already in the view.
		err := view.fetchInputUtxos(b.db, block)
		if err != nil {
			return err
		}

		// Update the view to unspend all of the spent txos and remove
		// the utxos created by the block.
		err = view.disconnectTransactions(b.db, block,
			detachSpentTxOuts[i])
		if err != nil {
			return err
		}

		// Update the database and chain state.
		err = b.disconnectBlock(n, block, view)
		if err != nil {
			return err
		}
	}

	// Connect the new best chain blocks.
	for i, e := 0, attachNodes.Front(); e != nil; i, e = i+1, e.Next() {
		n := e.Value.(*blockNode)
		block := attachBlocks[i]

		// Load all of the utxos referenced by the block that aren't
		// already in the view.
		err := view.fetchInputUtxos(b.db, block)
		if err != nil {
			return err
		}

		// Update the view to mark all utxos referenced by the block
		// as spent and add all transactions being created by this block
		// to it.  Also, provide an stxo slice so the spent txout
		// details are generated.
		stxos := make([]SpentTxOut, 0, countSpentOutputs(block))
		err = view.connectTransactions(block, &stxos)
		if err != nil {
			return err
		}

		// Update the database and chain state.
		err = b.connectBlock(n, block, view, stxos)
		if err != nil {
			return err
		}
	}

	// Log the point where the chain forked and old and new best chain
	// heads.
	if forkNode != nil {
		log.Infof("REORGANIZE: Chain forks at %v (height %v)", forkNode.hash,
			forkNode.height)
	}
	log.Infof("REORGANIZE: Old best chain head was %v (height %v)",
		&oldBest.hash, oldBest.height)
	log.Infof("REORGANIZE: New best chain head is %v (height %v)",
		newBest.hash, newBest.height)

	return nil
}
*/

// connectBestChain handles connecting the passed block to the chain while
// respecting proper chain selection according to the chain with the most
// proof of work.  In the typical case, the new block simply extends the main
// chain.  However, it may also be extending (or creating) a side chain (fork)
// which may or may not end up becoming the main chain depending on which fork
// cumulatively has the most proof of work.  It returns whether or not the block
// ended up on the main chain (either due to extending the main chain or causing
// a reorganization to become the main chain).
//
// The flags modify the behavior of this function as follows:
//  - BFFastAdd: Avoids several expensive transaction validation operations.
//    This is useful when using checkpoints.
//
// This function MUST be called with the chain state lock held (for writes).
func (b *BlockDAG) connectBestChain(node *blockNode, block *soterutil.Block, flags BehaviorFlags) (bool, error) {
	fastAdd := flags&BFFastAdd == BFFastAdd

	flushIndexState := func() {
		// Intentionally ignore errors writing updated node status to DB. If
		// it fails to write, it's not the end of the world. If the block is
		// valid, we flush in connectBlock and if the block is invalid, the
		// worst that can happen is we revalidate the block after a restart.
		if writeErr := b.index.flushToDB(); writeErr != nil {
			log.Warnf("Error flushing block index changes to disk: %v",
				writeErr)
		}
	}

	//jenlouie: removed check that this is extending tip of chain since in DAG a new block does not have to

	// Skip checks if node has already been fully validated.
	fastAdd = fastAdd || b.index.NodeStatus(node).KnownValid()

	// Perform several checks to verify the block can be connected
	// to the main chain without violating any rules and without
	// actually connecting the block.
	view := NewUtxoViewpoint()
	view.SetBestHash(generateTipsHash(node.parents)) //TODO: ??
	stxos := make([]SpentTxOut, 0, countSpentOutputs(block))
	if !fastAdd {
		err := b.checkConnectBlock(node, block, view, &stxos)
		if err == nil {
			b.index.SetStatusFlags(node, statusValid)
		} else if _, ok := err.(RuleError); ok {
			b.index.SetStatusFlags(node, statusValidateFailed)
		} else {
			return false, err
		}

		flushIndexState()

		if err != nil {
			return false, err

		}
	}

	// In the fast add case the code to check the block connection
	// was skipped, so the utxo view needs to load the referenced
	// utxos, spend them, and add the new utxos being created by
	// this block.
	if fastAdd {
		/*
		err := view.fetchInputUtxos(b.db, block)
		if err != nil {
			return false, err
		}
		err = view.connectTransactions(block, &stxos)
		if err != nil {
			return false, err
		}
		*/
	}

	// Connect the block to the main chain.
	err := b.connectBlock(node, block, view, stxos)
	if err != nil {
		// If we got hit with a rule error, then we'll mark
		// that status of the block as invalid and flush the
		// index state to disk before returning with the error.
		if _, ok := err.(RuleError); ok {
			b.index.SetStatusFlags(
				node, statusValidateFailed,
			)
		}

		flushIndexState()

		return false, err
	}

	// If this is fast add, or this block node isn't yet marked as
	// valid, then we'll update its status and flush the state to
	// disk again.
	if fastAdd || !b.index.NodeStatus(node).KnownValid() {
		b.index.SetStatusFlags(node, statusValid)
		flushIndexState()
	}

	return true, nil

	/*
		if fastAdd {
			log.Warnf("fastAdd set in the side chain case? %v\n",
				block.Hash())
		}

		// We're extending (or creating) a side chain, but the cumulative
		// work for this new side chain is not enough to make it the new chain.
		if node.workSum.Cmp(b.bestChain.Tip().workSum) <= 0 {
			// Log information about how the block is forking the chain.
			fork := b.bestChain.FindFork(node)
			if fork.hash.IsEqual(parentHash) {
				log.Infof("FORK: Block %v forks the chain at height %d"+
					"/block %v, but does not cause a reorganize",
					node.hash, fork.height, fork.hash)
			} else {
				log.Infof("EXTEND FORK: Block %v extends a side chain "+
					"which forks the chain at height %d/block %v",
					node.hash, fork.height, fork.hash)
			}

			return false, nil
		}

		// We're extending (or creating) a side chain and the cumulative work
		// for this new side chain is more than the old best chain, so this side
		// chain needs to become the main chain.  In order to accomplish that,
		// find the common ancestor of both sides of the fork, disconnect the
		// blocks that form the (now) old fork from the main chain, and attach
		// the blocks that form the new chain to the main chain starting at the
		// common ancenstor (the point where the chain forked).
		detachNodes, attachNodes := b.getReorganizeNodes(node)

		// Reorganize the chain.
		log.Infof("REORGANIZE: Block %v is causing a reorganize.", node.hash)
		err := b.reorganizeChain(detachNodes, attachNodes)

		// Either getReorganizeNodes or reorganizeChain could have made unsaved
		// changes to the block index, so flush regardless of whether there was an
		// error. The index would only be dirty if the block failed to connect, so
		// we can ignore any errors writing.
		if writeErr := b.index.flushToDB(); writeErr != nil {
			log.Warnf("Error flushing block index changes to disk: %v", writeErr)
		}
	*/
	//return err == nil, err
}

// isCurrent returns whether or not the chain believes it is current.  Several
// factors are used to guess, but the key factors that allow the chain to
// believe it is current are:
//  - Latest block height is after the latest checkpoint (if enabled)
//  - Latest block has a timestamp newer than 24 hours ago
//
// This function MUST be called with the chain state lock held (for reads).
func (b *BlockDAG) isCurrent() bool {
	// Not current if the latest main (best) chain height is before the
	// latest known good checkpoint (when checkpoints are enabled).
	/*checkpoint := b.LatestCheckpoint()
	if checkpoint != nil && b.bestChain.Tip().height < checkpoint.Height {
		return false
	}*/

	// Not current if the latest best block has a timestamp before 24 hours
	// ago.
	//
	// The chain appears to be current if none of the checks reported
	// otherwise.
	minus24Hours := b.timeSource.AdjustedTime().Add(-24 * time.Hour).Unix()

	for _, tip := range b.dView.Tips() {
		if tip.timestamp >= minus24Hours {
			return true
		}
	}
	return false
	//return b.bestChain.Tip().timestamp >= minus24Hours
}

// IsCurrent returns whether or not the chain believes it is current.  Several
// factors are used to guess, but the key factors that allow the chain to
// believe it is current are:
//  - Latest block height is after the latest checkpoint (if enabled)
//  - Latest block has a timestamp newer than 24 hours ago
//
// This function is safe for concurrent access.
func (b *BlockDAG) IsCurrent() bool {
	b.chainLock.RLock()
	defer b.chainLock.RUnlock()

	return b.isCurrent()
}

// BestSnapshot returns information about the current best chain block and
// related state as of the current point in time.  The returned instance must be
// treated as immutable since it is shared by all callers.
//
// This function is safe for concurrent access.
func (b *BlockDAG) BestSnapshot() *BestState {
	b.stateLock.RLock()
	snapshot := b.stateSnapshot
	b.stateLock.RUnlock()
	return snapshot
}

func (b *BlockDAG) DAGSnapshot() *DAGState {
	b.stateLock.RLock()
	snapshot := b.dagSnapshot
	b.stateLock.RUnlock()
	return snapshot
}

// DAGColoring returns the blue set of blocks after coloring is run on the DAG
// Based on the last block added
func (b *BlockDAG) DAGColoring() []*chainhash.Hash {
	b.chainLock.RLock()
	defer b.chainLock.RUnlock()

	latestBlock := b.BestSnapshot().Hash
	latestNode := b.graph.GetNodeById(latestBlock.String())
	blueNodes := b.blueSet.GetBlueNodes(latestNode)
	if blueNodes != nil {
		blueHashes := make([]*chainhash.Hash, len(blueNodes))
		for i, node := range blueNodes {
			hash, _ := chainhash.NewHashFromStr(node.GetId())
			blueHashes[i] = hash
		}
		return blueHashes
	}

	return nil
}

// DAGOrdering returns the ordering of the blocks after the DAG is sorted
func (b *BlockDAG) DAGOrdering() []*chainhash.Hash {
	b.chainLock.RLock()
	defer b.chainLock.RUnlock()

	return b.nodeOrder
}

// HeaderByHash returns the block header identified by the given hash or an
// error if it doesn't exist. Note that this will return headers from both the
// main and side chains.
func (b *BlockDAG) HeaderByHash(hash *chainhash.Hash) (wire.BlockHeader, error) {
	node := b.index.LookupNode(hash)
	if node == nil {
		err := fmt.Errorf("block %s is not known", hash)
		return wire.BlockHeader{}, err
	}

	return node.Header(), nil
}

// MainChainHasBlock returns whether or not the block with the given hash is in
// the main chain.
//
// This function is safe for concurrent access.
func (b *BlockDAG) MainChainHasBlock(hash *chainhash.Hash) bool {
	node := b.index.LookupNode(hash)
	return node != nil && b.dView.Contains(node)
}

// BlockLocatorFromHash returns a block locator for the passed block hash.
// See BlockLocator for details on the algorithm used to create a block locator.
//
// In addition to the general algorithm referenced above, this function will
// return the block locator for the latest known tip of the main (best) chain if
// the passed hash is not currently known.
//
// This function is safe for concurrent access.
func (b *BlockDAG) BlockLocatorFromHash(hash *chainhash.Hash) BlockLocator {
	b.chainLock.RLock()
	node := b.index.LookupNode(hash)
	locator := b.dView.blockLocator(node)
	b.chainLock.RUnlock()
	return locator
}

// BLockLocatorFromHeight returns a block locator for the passed height.
// If the height is less than 0, the locator's height will be 0.
func (b *BlockDAG) BlockLocatorFromHeight(height int32) BlockLocator {
	locator := make(BlockLocator, 0, 1)
	if height < 0 {
		minHeight := int32(0)
		locator = append(locator, &minHeight)
	} else {
		locator = append(locator, &height)
	}

	return locator
}

// LatestBlockLocator returns a block locator for the latest known tip of the
// main (best) chain.
//
// This function is safe for concurrent access.
func (b *BlockDAG) LatestBlockLocator() (BlockLocator, error) {
	b.chainLock.RLock()
	locator := b.dView.BlockLocator(nil)
	b.chainLock.RUnlock()
	return locator, nil
}

// BlockHeightByHash returns the height of the block with the given hash in the
// main chain.
//
// This function is safe for concurrent access.
func (b *BlockDAG) BlockHeightByHash(hash *chainhash.Hash) (int32, error) {
	node := b.index.LookupNode(hash)
	if node == nil || !b.dView.Contains(node) {
		str := fmt.Sprintf("block %s is not in the main chain", hash)
		return 0, errNotInMainChain(str)
	}

	return node.height, nil
}

// BlockHashesByHeight returns the hashes of the blocks at the given height in the
// main chain.
//
// This function is safe for concurrent access.
func (b *BlockDAG) BlockHashesByHeight(blockHeight int32) ([]chainhash.Hash, error) {
	nodes := b.dView.NodesByHeight(blockHeight)
	if nodes == nil || len(nodes) == 0 {
		str := fmt.Sprintf("no block at height %d exists", blockHeight)
		return nil, errNotInMainChain(str)

	}

	hashes := make([]chainhash.Hash, len(nodes))
	i := 0
	for _, node := range nodes {
		hashes[i] = node.hash
		i++
	}

	return hashes, nil
}

// HeightRange returns a range of block hashes for the given start and end
// heights.  It is inclusive of the start height and exclusive of the end
// height.  The end height will be limited to the current main chain height.
//
// This function is safe for concurrent access.
func (b *BlockDAG) HeightRange(startHeight, endHeight int32) ([]chainhash.Hash, error) {
	// Ensure requested heights are sane.
	if startHeight < 0 {
		return nil, fmt.Errorf("start height of fetch range must not "+
			"be less than zero - got %d", startHeight)
	}
	if endHeight < startHeight {
		return nil, fmt.Errorf("end height of fetch range must not "+
			"be less than the start height - got start %d, end %d",
			startHeight, endHeight)
	}

	// There is nothing to do when the start and end heights are the same,
	// so return now to avoid the chain view lock.
	if startHeight == endHeight {
		return nil, nil
	}

	// Grab a lock on the chain view to prevent it from changing due to a
	// reorg while building the hashes.
	b.dView.mtx.Lock()
	defer b.dView.mtx.Unlock()

	// When the requested start height is after the most recent best chain
	// height, there is nothing to do.
	latestHeight := b.dView.height()
	if startHeight > latestHeight {
		return nil, nil
	}

	// Limit the ending height to the latest height of the chain.
	if endHeight > latestHeight+1 {
		endHeight = latestHeight + 1
	}

	// Fetch as many as are available within the specified range.
	hashes := make([]chainhash.Hash, 0)
	for i := startHeight; i < endHeight; i++ {
		nodes := b.dView.nodesByHeight(i)
		for _, node := range nodes {
			hashes = append(hashes, node.hash)
		}
	}
	return hashes, nil
}

// HeightToHashRange returns a range of block hashes for the given start height
// and end hash, inclusive on both ends.  The hashes are for all blocks that are
// ancestors of endHash with height greater than or equal to startHeight.  The
// end hash must belong to a block that is known to be valid.
//
// This function is safe for concurrent access.
func (b *BlockDAG) HeightToHashRange(startHeight int32,
	endHash *chainhash.Hash, maxResults int) ([]chainhash.Hash, error) {

	endNode := b.index.LookupNode(endHash)
	if endNode == nil {
		return nil, fmt.Errorf("no known block header with hash %v", endHash)
	}
	if !b.index.NodeStatus(endNode).KnownValid() {
		return nil, fmt.Errorf("block %v is not yet validated", endHash)
	}
	endHeight := endNode.height

	if startHeight < 0 {
		return nil, fmt.Errorf("start height (%d) is below 0", startHeight)
	}
	if startHeight > endHeight {
		return nil, fmt.Errorf("start height (%d) is past end height (%d)",
			startHeight, endHeight)
	}

	resultsLength := 0
	for i := startHeight; i <= endHeight; i++ {
		resultsLength += len(b.dView.NodesByHeight(i))
	}
	//resultsLength := int(endHeight - startHeight + 1)
	if resultsLength > maxResults {
		return nil, fmt.Errorf("number of results (%d) would exceed max (%d)",
			resultsLength, maxResults)
	}

	// Walk backwards from endHeight to startHeight, collecting block hashes.
	//node := endNode
	j := resultsLength - 1
	hashes := make([]chainhash.Hash, resultsLength)
	for i := endHeight; i >= startHeight; i-- {
		nodes := b.dView.NodesByHeight(i)
		//fmt.Printf("number of nodes at height %d: %d\n", i, len(nodes))
		for k := len(nodes) - 1; k >= 0; k-- {
			hashes[j] = nodes[k].hash
			j--
		}
	}
	return hashes, nil
}

// IntervalBlockHashes returns hashes for all blocks that are ancestors of
// endHash where the block height is a positive multiple of interval.
//
// This function is safe for concurrent access.
func (b *BlockDAG) IntervalBlockHashes(endHash *chainhash.Hash, interval int,
) ([]chainhash.Hash, error) {

	endNode := b.index.LookupNode(endHash)
	if endNode == nil {
		return nil, fmt.Errorf("no known block header with hash %v", endHash)
	}
	if !b.index.NodeStatus(endNode).KnownValid() {
		return nil, fmt.Errorf("block %v is not yet validated", endHash)
	}
	endHeight := endNode.height

	resultsLength := int(endHeight) / interval
	hashes := make([]chainhash.Hash, resultsLength)

	b.dView.mtx.Lock()
	defer b.dView.mtx.Unlock()

	blockNode := endNode
	for index := int(endHeight) / interval; index > 0; index-- {
		// Use the bestChain chainView for faster lookups once lookup intersects
		// the best chain.
		blockHeight := int32(index * interval)
		if b.dView.contains(blockNode) {
			blockNodes := b.dView.nodesByHeight(blockHeight)
			blockNode = blockNodes[0] //TODO: only returns one node at that height
		} else {
			// shouldn't happen
			//blockNode = blockNode.Ancestor(blockHeight)
		}

		hashes[index-1] = blockNode.hash
	}

	return hashes, nil
}

// locateInventory returns the nodes of the blocks after the first known block in
// the locator up to the stop hash or the provided max number of entries.
//
// In addition, there are two special cases:
//
// - When no locators are provided, the stop hash is treated as a request for
//   that block, so it will either return the node associated with the stop hash
//   if it is known, or nil if it is unknown
// - When locators are provided, but none of them are known, nodes starting
//   after the genesis block will be returned
//
// This is primarily a helper function for the locateBlocks and locateHeaders
// functions.
//
// This function MUST be called with the chain state lock held (for reads).
func (b *BlockDAG) locateInventory(locator BlockLocator, hashStop *chainhash.Hash, maxEntries uint32) []*blockNode {
	var stopNode *blockNode
	if hashStop.IsEqual(&zeroHash) {
		// A hashStop equal to zeroHash means we should return blocks from locator to tips
		dagState := b.DAGSnapshot()
		nodes := b.dView.NodesByHeight(dagState.MaxHeight)
		latest := nodes[len(nodes) - 1]
		stopNode = b.index.LookupNode(&latest.hash)
	} else {
		stopNode = b.index.LookupNode(hashStop)
		if stopNode == nil {
			// If we don't have the node for hashStop, don't return anything
			return nil
		}
	}

	if len(locator) == 0 {
		// There are no block locators so a specific block is being requested
		// as identified by the stop hash.
		if stopNode == nil {
			// No blocks with the stop hash were found so there is
			// nothing to do.
			return nil
		}
		if maxEntries >= uint32(1) {
			return []*blockNode{stopNode}
		} else {
			return nil
		}
	}

	genesisHeight := b.dView.Genesis().height
	var startHeight int32
	if *locator[0] == genesisHeight {
		// We don't count the genesis block as inventory; It's already hard-coded into the client.
		startHeight = genesisHeight + 1
	} else {
		startHeight = *locator[0] + 1
	}

	inventory := make([]*blockNode, 0)
	MAXREACHED:
	for h := startHeight; h <= stopNode.height; h++ {
		nodes := b.dView.NodesByHeight(h)
		for _, n := range nodes {
			if uint32(len(inventory)) < maxEntries {
				inventory = append(inventory, n)
			} else {
				break MAXREACHED
			}
		}
	}

	return inventory
}

// locateBlocks returns the hashes of the blocks after the first known block in
// the locator until the provided stop hash is reached, or up to the provided
// max number of block hashes.
//
// See the comment on the exported function for more details on special cases.
//
// This function MUST be called with the chain state lock held (for reads).
func (b *BlockDAG) locateBlocks(locator BlockLocator, hashStop *chainhash.Hash, maxHashes uint32) []chainhash.Hash {
	// Find the node after the first known block in the locator while respecting the stop hash and max entries.
	nodes := b.locateInventory(locator, hashStop, maxHashes)

	// Populate and return the found hashes.
	hashes := make([]chainhash.Hash, 0, len(nodes))

	for _, node := range nodes {
		hashes = append(hashes, node.hash)
	}

	return hashes
}

// LocateBlocks returns the hashes of the blocks after the first known block in
// the locator until the provided stop hash is reached, or up to the provided
// max number of block hashes.
//
// In addition, there are two special cases:
//
// - When no locators are provided, the stop hash is treated as a request for
//   that block, so it will either return the stop hash itself if it is known,
//   or nil if it is unknown
// - When locators are provided, but none of them are known, hashes starting
//   after the genesis block will be returned
//
// This function is safe for concurrent access.
func (b *BlockDAG) LocateBlocks(locator BlockLocator, hashStop *chainhash.Hash, maxHashes uint32) []chainhash.Hash {
	b.chainLock.RLock()
	hashes := b.locateBlocks(locator, hashStop, maxHashes)
	b.chainLock.RUnlock()
	return hashes
}

// locateHeaders returns the headers of the blocks after the first known block
// in the locator until the provided stop hash is reached, or up to the provided
// max number of block headers.
//
// See the comment on the exported function for more details on special cases.
//
// This function MUST be called with the chain state lock held (for reads).
func (b *BlockDAG) locateHeaders(locator BlockLocator, hashStop *chainhash.Hash, maxHeaders uint32) []wire.BlockHeader {
	// Find the node after the first known block in the locator while respecting the stop hash and max entries.
	nodes := b.locateInventory(locator, hashStop, maxHeaders)

	// Populate and return the found headers.
	headers := make([]wire.BlockHeader, 0, len(nodes))

	for _, node := range nodes {
		headers = append(headers, node.Header())
	}

	return headers
}

// LocateHeaders returns the headers of the blocks after the first known block
// in the locator until the provided stop hash is reached, or up to a max of
// wire.MaxBlockHeadersPerMsg headers.
//
// In addition, there are two special cases:
//
// - When no locators are provided, the stop hash is treated as a request for
//   that header, so it will either return the header for the stop hash itself
//   if it is known, or nil if it is unknown
// - When locators are provided, but none of them are known, headers starting
//   after the genesis block will be returned
//
// This function is safe for concurrent access.
func (b *BlockDAG) LocateHeaders(locator BlockLocator, hashStop *chainhash.Hash) []wire.BlockHeader {
	b.chainLock.RLock()
	headers := b.locateHeaders(locator, hashStop, wire.MaxBlockHeadersPerMsg)
	b.chainLock.RUnlock()
	return headers
}

// Given nodes that are the tips of a subgraph, return list of blocks that
// are missing from the subgraph tips to the tips of the graph
func (b *BlockDAG) GetBlockDiff(subtips []*chainhash.Hash) []*chainhash.Hash {
	ids := make([]string, len(subtips))
	for i, hash := range subtips {
		ids[i] = hash.String()
	}

	missingIds := b.graph.GetMissingNodes(ids)

	missingHashes := make([]*chainhash.Hash, len(missingIds))
	for i, id := range missingIds {
		missingHashes[i], _ = chainhash.NewHashFromStr(id)
	}

	return missingHashes
}

// IndexManager provides a generic interface that the is called when blocks are
// connected and disconnected to and from the tip of the main chain for the
// purpose of supporting optional indexes.
type IndexManager interface {
	// Init is invoked during chain initialize in order to allow the index
	// manager to initialize itself and any indexes it is managing.  The
	// channel parameter specifies a channel the caller can close to signal
	// that the process should be interrupted.  It can be nil if that
	// behavior is not desired.
	Init(*BlockDAG, <-chan struct{}) error

	// ConnectBlock is invoked when a new block has been connected to the
	// main chain. The set of output spent within a block is also passed in
	// so indexers can access the previous output scripts input spent if
	// required.
	ConnectBlock(database.Tx, *soterutil.Block, []SpentTxOut) error

	// DisconnectBlock is invoked when a block has been disconnected from
	// the main chain. The set of outputs scripts that were spent within
	// this block is also returned so indexers can clean up the prior index
	// state for this block.
	//DisconnectBlock(database.Tx, *soterutil.Block, []SpentTxOut) error
}

// Config is a descriptor which specifies the blockchain instance configuration.
type Config struct {
	// DB defines the database which houses the blocks and will be used to
	// store all metadata created by this package such as the utxo set.
	//
	// This field is required.
	DB database.DB

	// Interrupt specifies a channel the caller can close to signal that
	// long running operations, such as catching up indexes or performing
	// database migrations, should be interrupted.
	//
	// This field can be nil if the caller does not desire the behavior.
	Interrupt <-chan struct{}

	// ChainParams identifies which chain parameters the chain is associated
	// with.
	//
	// This field is required.
	ChainParams *chaincfg.Params

	// Checkpoints hold caller-defined checkpoints that should be added to
	// the default checkpoints in ChainParams.  Checkpoints must be sorted
	// by height.
	//
	// This field can be nil if the caller does not wish to specify any
	// checkpoints.
	//Checkpoints []chaincfg.Checkpoint

	// TimeSource defines the median time source to use for things such as
	// block processing and determining whether or not the chain is current.
	//
	// The caller is expected to keep a reference to the time source as well
	// and add time samples from other peers on the network so the local
	// time is adjusted to be in agreement with other peers.
	TimeSource MedianTimeSource

	// SigCache defines a signature cache to use when when validating
	// signatures.  This is typically most useful when individual
	// transactions are already being validated prior to their inclusion in
	// a block such as what is usually done via a transaction memory pool.
	//
	// This field can be nil if the caller is not interested in using a
	// signature cache.
	SigCache *txscript.SigCache

	// IndexManager defines an index manager to use when initializing the
	// chain and connecting and disconnecting blocks.
	//
	// This field can be nil if the caller does not wish to make use of an
	// index manager.
	IndexManager IndexManager

	// HashCache defines a transaction hash mid-state cache to use when
	// validating transactions. This cache has the potential to greatly
	// speed up transaction validation as re-using the pre-calculated
	// mid-state eliminates the O(N^2) validation complexity due to the
	// SigHashAll flag.
	//
	// This field can be nil if the caller is not interested in using a
	// signature cache.
	HashCache *txscript.HashCache
}

// New returns a BlockChain instance using the provided configuration details.
func New(config *Config) (*BlockDAG, error) {
	// Enforce required config fields.
	if config.DB == nil {
		return nil, AssertError("blockchain.New database is nil")
	}
	if config.ChainParams == nil {
		return nil, AssertError("blockchain.New chain parameters nil")
	}
	if config.TimeSource == nil {
		return nil, AssertError("blockchain.New timesource is nil")
	}

	// Generate a checkpoint by height map from the provided checkpoints
	// and assert the provided checkpoints are sorted by height as required.
	/*var checkpointsByHeight map[int32]*chaincfg.Checkpoint
	var prevCheckpointHeight int32
	if len(config.Checkpoints) > 0 {
		checkpointsByHeight = make(map[int32]*chaincfg.Checkpoint)
		for i := range config.Checkpoints {
			checkpoint := &config.Checkpoints[i]
			if checkpoint.Height <= prevCheckpointHeight {
				return nil, AssertError("blockchain.New " +
					"checkpoints are not sorted by height")
			}

			checkpointsByHeight[checkpoint.Height] = checkpoint
			prevCheckpointHeight = checkpoint.Height
		}
	}*/

	params := config.ChainParams
	targetTimespan := int64(params.TargetTimespan / time.Millisecond)
	targetTimePerBlock := int64(params.TargetTimePerBlock / time.Millisecond)
	adjustmentFactor := params.RetargetAdjustmentFactor
	b := BlockDAG{
		//checkpoints:         config.Checkpoints,
		//checkpointsByHeight: checkpointsByHeight,
		db:                  config.DB,
		chainParams:         params,
		timeSource:          config.TimeSource,
		sigCache:            config.SigCache,
		indexManager:        config.IndexManager,
		minRetargetTimespan: targetTimespan / adjustmentFactor,
		maxRetargetTimespan: targetTimespan * adjustmentFactor,
		blocksPerRetarget:   int64(targetTimespan / targetTimePerBlock),
		index:               newBlockIndex(config.DB, params),
		hashCache:           config.HashCache,
		dView:               newDAGView(nil),
		graph:               phantom.NewGraph(),
		nodeOrder:           make([]*chainhash.Hash, 0),
		blueSet:             phantom.NewBlueSetCache(),
		orderCache:          phantom.NewOrderCache(),
		orphans:             make(map[chainhash.Hash]*orphanBlock),
		prevOrphans:         make(map[chainhash.Hash][]*orphanBlock),
		warningCaches:       newThresholdCaches(vbNumBits),
		deploymentCaches:    newThresholdCaches(chaincfg.DefinedDeployments),
	}

	// Initialize the chain state from the passed database.  When the db
	// does not yet contain any chain state, both it and the chain state
	// will be initialized to contain only the genesis block.
	if err := b.initChainState(); err != nil {
		return nil, err
	}

	// Perform any upgrades to the various chain-specific buckets as needed.
	//if err := b.maybeUpgradeDbBuckets(config.Interrupt); err != nil {
	//	return nil, err
	//}

	// Initialize and catch up all of the currently active optional indexes
	// as needed.
	if config.IndexManager != nil {
		err := config.IndexManager.Init(&b, config.Interrupt)
		if err != nil {
			return nil, err
		}
	}

	// Initialize rule change threshold state caches.
	//if err := b.initThresholdCaches(); err != nil {
	//	return nil, err
	//}

	tips := b.dView.Tips()
	log.Infof("Chain state (height %d, number of tips %d)",
		b.dView.Height(), len(tips))

	return &b, nil
}
