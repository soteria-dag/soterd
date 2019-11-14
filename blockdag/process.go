// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdag

import (
	"fmt"

	"github.com/soteria-dag/soterd/chaincfg/chainhash"
	"github.com/soteria-dag/soterd/database"
	"github.com/soteria-dag/soterd/soterutil"
)

// BehaviorFlags is a bitmask defining tweaks to the normal behavior when
// performing chain processing and consensus rules checks.
type BehaviorFlags uint32

const (
	// BFFastAdd may be set to indicate that several checks can be avoided
	// for the block since it is already known to fit into the chain due to
	// already proving it correct links into the chain up to a known
	// checkpoint.  This is primarily used for headers-first mode.
	BFFastAdd BehaviorFlags = 1 << iota

	// BFNoPoWCheck may be set to indicate the proof of work check which
	// ensures a block hashes to a value less than the required target will
	// not be performed.
	BFNoPoWCheck

	// BFNone is a convenience value to specifically indicate no flags.
	BFNone BehaviorFlags = 0
)

// blockExists determines whether a block with the given hash exists either in
// the main chain or any side chains.
//
// This function is safe for concurrent access.
func (b *BlockDAG) blockExists(hash *chainhash.Hash) (bool, error) {
	// Check block index first (could be main chain or side chain blocks).
	if b.index.HaveBlock(hash) {
		return true, nil
	}

	// Check in the database.
	var exists bool
	err := b.db.View(func(dbTx database.Tx) error {
		var err error
		exists, err = dbTx.HasBlock(hash)
		if err != nil || !exists {
			return err
		}

		// Ignore side chain blocks in the database.  This is necessary
		// because there is not currently any record of the associated
		// block index data such as its block height, so it's not yet
		// possible to efficiently load the block and do anything useful
		// with it.
		//
		// Ultimately the entire block index should be serialized
		// instead of only the current main chain so it can be consulted
		// directly.
		_, err = dbFetchHeightByHash(dbTx, hash)
		if isNotInMainChainErr(err) {
			exists = false
			return nil
		}
		return err
	})
	return exists, err
}

// processOrphans determines if there are any orphans whose dependencies are now resolved (presumably by accepting
// the most recently-processed block into the dag).
// It repeats this process until no more orphans with resolved dependencies could be found.
// Orphans are processed from least-dependent to most, to maximize possibility of resolving dag.
//
// The flags do not modify the behavior of this function directly, however they
// are needed to pass along to maybeAcceptBlock.
//
// This function MUST be called with the chain state lock held (for writes).
func (b *BlockDAG) processOrphans(flags BehaviorFlags) (bool, error) {
	accepted := false

	// Determine orphan processing order
	// (We can't go by height because the blocks haven't been accepted into the dag yet)
	orphanHashes, err := b.GetOrphanOrder()
	if err != nil {
		return false, err
	}

	orphans := make([]*orphanBlock, 0)
	b.orphanLock.RLock()
	for _, hash := range orphanHashes {
		orphan, exists := b.orphans[hash]
		if exists {
			orphans = append(orphans, orphan)
		}
	}
	b.orphanLock.RUnlock()

	checked := make(map[chainhash.Hash]int)
	for len(orphans) > 0 {
		orphan := orphans[0]
		orphans[0] = nil
		orphans = orphans[1:]

		hash := orphan.block.Hash()

		_, exists := checked[*hash]
		if exists {
			// Multiple blocks may have the same parent, but we only need to check them once
			continue
		} else {
			checked[*hash] = 1
		}

		if !b.IsKnownOrphan(hash) {
			// Skip an orphan if it was already resolved, but listed more than once in orphan order
			continue
		}

		parents := orphan.block.MsgBlock().Parents.ParentHashes()
		missingParent := false
		for _, parent := range parents {
			exists, err := b.blockExists(&parent)
			if err != nil {
				return accepted, err
			}
			if !exists {
				missingParent = true

				if b.IsKnownOrphan(&parent) {
					// If the parent is an orphan, examine it next
					b.orphanLock.RLock()
					parentOrphan := b.orphans[parent]
					b.orphanLock.RUnlock()

					next := []*orphanBlock{parentOrphan}
					orphans = append(next, orphans...)
				}
			}
		}

		if missingParent {
			continue
		}

		// Remove the orphan from the orphan pool.
		b.removeOrphanBlock(orphan)

		// Orphan has no missing parents, so we'll attempt to accept it into the DAG
		_, err := b.maybeAcceptBlock(orphan.block, flags)
		if err != nil {
			log.Warnf("Couldn't process orphan %v with parents %v: %v", hash, parents, err)
			continue
		} else {
			accepted = true
			log.Infof("Accepted resolved orphan block %v", hash)
		}
	}

	return accepted, nil
}

// ProcessBlock is the main workhorse for handling insertion of new blocks into
// the block chain.  It includes functionality such as rejecting duplicate
// blocks, ensuring blocks follow all rules, orphan handling, and insertion into
// the block chain along with best chain selection and reorganization.
//
// When no errors occurred during processing, the first return value indicates
// whether or not the block is on the main chain and the second indicates
// whether or not the block is an orphan.
//
// This function is safe for concurrent access.
func (b *BlockDAG) ProcessBlock(block *soterutil.Block, flags BehaviorFlags) (bool, bool, error) {
	b.chainLock.Lock()
	defer b.chainLock.Unlock()

	//fastAdd := flags&BFFastAdd == BFFastAdd

	blockHash := block.Hash()
	log.Tracef("Processing block %v", blockHash)

	// The block must not already exist in the main chain or side chains.
	exists, err := b.blockExists(blockHash)
	if err != nil {
		return false, false, err
	}
	if exists {
		str := fmt.Sprintf("already have block %v", blockHash)
		return false, false, ruleError(ErrDuplicateBlock, str)
	}

	// The block must not already exist as an orphan.
	if _, exists := b.orphans[*blockHash]; exists {
		str := fmt.Sprintf("already have block (orphan) %v", blockHash)
		return false, false, ruleError(ErrDuplicateBlock, str)
	}

	// Perform preliminary sanity checks on the block and its transactions.
	err = checkBlockSanity(block, b.chainParams.PowLimit, b.timeSource, flags)
	if err != nil {
		return false, false, err
	}

	// Find the previous checkpoint and perform some additional checks based
	// on the checkpoint.  This provides a few nice properties such as
	// preventing old side chain blocks before the last checkpoint,
	// rejecting easy to mine, but otherwise bogus, blocks that could be
	// used to eat memory, and ensuring expected (versus claimed) proof of
	// work requirements since the previous checkpoint are met.
	/*
	blockHeader := &block.MsgBlock().Header
	checkpointNode, err := b.findPreviousCheckpoint()
	if err != nil {
		return false, false, err
	}
	if checkpointNode != nil {
		// Ensure the block timestamp is after the checkpoint timestamp.
		checkpointTime := time.Unix(checkpointNode.timestamp, 0)
		if blockHeader.Timestamp.Before(checkpointTime) {
			str := fmt.Sprintf("block %v has timestamp %v before "+
				"last checkpoint timestamp %v", blockHash,
				blockHeader.Timestamp, checkpointTime)
			return false, false, ruleError(ErrCheckpointTimeTooOld, str)
		}
		if !fastAdd {
			// Even though the checks prior to now have already ensured the
			// proof of work exceeds the claimed amount, the claimed amount
			// is a field in the block header which could be forged.  This
			// check ensures the proof of work is at least the minimum
			// expected based on elapsed time since the last checkpoint and
			// maximum adjustment allowed by the retarget rules.
			duration := blockHeader.Timestamp.Sub(checkpointTime)
			requiredTarget := CompactToBig(b.calcEasiestDifficulty(
				checkpointNode.bits, duration))
			currentTarget := CompactToBig(blockHeader.Bits)
			if currentTarget.Cmp(requiredTarget) > 0 {
				str := fmt.Sprintf("block target difficulty of %064x "+
					"is too low when compared to the previous "+
					"checkpoint", currentTarget)
				return false, false, ruleError(ErrDifficultyTooLow, str)
			}
		}
	}
   */

	// Handle orphan blocks.
	// If any parent block does not exist, add block as orphan
	parentHashes := block.MsgBlock().Parents.ParentHashes()
	for _, parentHash := range parentHashes {
		exists, err := b.blockExists(&parentHash)
		if err != nil {
			return false, false, err
		}

		if !exists {
			log.Infof("Adding orphan block %v with parents %v", blockHash, parentHashes)
			b.addOrphanBlock(block)

			return false, true, nil
		}
	}

	// The block has passed all context independent checks and appears sane
	// enough to potentially accept it into the block chain.
	isMainChain, err := b.maybeAcceptBlock(block, flags)
	if err != nil {
		return false, false, err
	}

	// Accept any orphan blocks that depend on this block (they are
	// no longer orphans) and repeat for those accepted blocks until
	// there are no more.
	//
	// NOTE(cedric): We don't fail accepting our original block when processing other orphans,
	// because this would mean that if the propagation of the entire DAG wasn't sequential
	// (all parents received before all children), we could end up in a situation where some blocks are
	// never added to the DAG because there's an orphan with two parents and one of them isn't resolved yet.
	accepted, err := b.processOrphans(flags)
	for accepted && err == nil {
		// Keep accepting orphan blocks until no more have been accepted
		accepted, err = b.processOrphans(flags)
	}

	log.Debugf("Accepted block %v", blockHash)

	return isMainChain, false, nil
}
