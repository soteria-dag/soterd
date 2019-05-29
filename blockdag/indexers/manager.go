// Copyright (c) 2016 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package dagindexers

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/soteria-dag/soterd/blockdag"
	"github.com/soteria-dag/soterd/chaincfg/chainhash"
	"github.com/soteria-dag/soterd/database"
	"github.com/soteria-dag/soterd/soterutil"
	"github.com/soteria-dag/soterd/wire"
)

var (
	// indexTipsBucketName is the name of the db bucket used to house the
	// current tips of each index.
	indexTipsBucketName = []byte("idxtips")

	// indexProcessedBucketName is the name of the db bucket used to
	// keep track of the blocks already in the index
	indexProcessedBucketName = []byte("idxblkhash")
)

func getProcessedBucketName(idxKey []byte) []byte {
	var bucketName []byte
	bucketName = append(bucketName, indexProcessedBucketName[:]...)
	bucketName = append(bucketName, idxKey[:]...)
	return bucketName
}

func dbPutBlockHash(dbTx database.Tx, idxKey []byte, hash *chainhash.Hash) error {
	meta := dbTx.Metadata()
	index := meta.Bucket(getProcessedBucketName(idxKey))
	return index.Put(hash[:], []byte{0x1})
}

func dbRemoveBlockHash(dbTx database.Tx, idxKey []byte, hash *chainhash.Hash) error {
	meta := dbTx.Metadata()
	index := meta.Bucket(getProcessedBucketName(idxKey))
	return index.Delete(hash[:])
}

func dbFetchBlockHash(dbTx database.Tx, idxKey  []byte, hash *chainhash.Hash) bool {
	meta := dbTx.Metadata()
	index := meta.Bucket(getProcessedBucketName(idxKey))
	val := index.Get(hash[:])

	if val == nil {
		return false
	} else {
		return true
	}
}

// -----------------------------------------------------------------------------
// The index manager tracks the current tips of each index by using a parent
// bucket that contains an entry for index.
//
// The serialized format for an index tip is:
//
//   [<number tips><block hash1><block hash2>],...
//
//   Field           Type             Size
//   block height    uint32           4 bytes
//   block hash      chainhash.Hash   chainhash.HashSize
// -----------------------------------------------------------------------------
func dbPutIndexerTips(dbTx database.Tx, idxKey []byte, tipHashes []*chainhash.Hash) error {
	serialized := make([]byte, chainhash.HashSize * len(tipHashes) + 4)
	byteOrder.PutUint32(serialized[:], uint32(len(tipHashes)))
	for i, hash := range tipHashes {
		start := (chainhash.HashSize * i) + 4
		copy(serialized[start:], hash[:])
	}

	indexesBucket := dbTx.Metadata().Bucket(indexTipsBucketName)
	return indexesBucket.Put(idxKey, serialized)
}

func dbFetchIndexerTips(dbTx database.Tx, idxKey []byte) ([]*chainhash.Hash, error) {
	indexesBucket := dbTx.Metadata().Bucket(indexTipsBucketName)
	serialized := indexesBucket.Get(idxKey)

	if (len(serialized) - 4) % chainhash.HashSize != 0 {
		return nil, database.Error{
			ErrorCode: database.ErrCorruption,
			Description: fmt.Sprintf("unexpected end of data for "+
				"index %q tips", string(idxKey)),
		}
	}

	numTips := int(byteOrder.Uint32(serialized[:]))
	tips := make([]*chainhash.Hash, numTips)
	for i := 0; i < numTips; i++ {
		var hash chainhash.Hash
		start := (chainhash.HashSize * i) + 4
		end := start + chainhash.HashSize
		copy(hash[:], serialized[start:end])
		tips[i] = &hash
	}

	return tips, nil
}

// dbIndexConnectBlock adds all of the index entries associated with the
// given block using the provided indexer and updates the tip of the indexer
// accordingly.  An error will be returned if the current tip for the indexer is
// not the previous block for the passed block.
func dbIndexConnectBlock(dbTx database.Tx, indexer Indexer, block *soterutil.Block,
	stxo []blockdag.SpentTxOut) error {

	idxKey := indexer.Key()

	// if already processed block, skip
	exists := dbFetchBlockHash(dbTx, idxKey, block.Hash())
	if exists {
		return nil
	}

	curTips, err := dbFetchIndexerTips(dbTx, idxKey)
	if err != nil {
		return err
	}

	parents := make([]*chainhash.Hash, 0)
	nonParents := make([]*chainhash.Hash, 0)
	for _, curTip := range curTips {
		if block.MsgBlock().Parents.IsParent(curTip) {
			parents = append(parents, curTip)
		} else {
			nonParents = append(nonParents, curTip)
		}
	}

	// TODO(jenlouie): can we check if new block extend tips? Would need to make sure
	// blocks are added in topographic order

	// Notify the indexer with the connected block so it can index it.
	if err := indexer.ConnectBlock(dbTx, block, stxo); err != nil {
		return err
	}

	// Mark block as seen
	if err := dbPutBlockHash(dbTx, idxKey, block.Hash()); err != nil {
		return err
	}

	// Update the current index tip.
	newTips := append(nonParents, block.Hash())
	return dbPutIndexerTips(dbTx, idxKey, newTips)
}

// Manager defines an index manager that manages multiple optional indexes and
// implements the blockchain.IndexManager interface so it can be seamlessly
// plugged into normal chain processing.
type Manager struct {
	db             database.DB
	enabledIndexes []Indexer
}

// Ensure the Manager type implements the blockchain.IndexManager interface.
var _ blockdag.IndexManager = (*Manager)(nil)

// indexDropKey returns the key for an index which indicates it is in the
// process of being dropped.
func indexDropKey(idxKey []byte) []byte {
	dropKey := make([]byte, len(idxKey)+1)
	dropKey[0] = 'd'
	copy(dropKey[1:], idxKey)
	return dropKey
}

// maybeFinishDrops determines if each of the enabled indexes are in the middle
// of being dropped and finishes dropping them when the are.  This is necessary
// because dropping and index has to be done in several atomic steps rather than
// one big atomic step due to the massive number of entries.
func (m *Manager) maybeFinishDrops(interrupt <-chan struct{}) error {
	indexNeedsDrop := make([]bool, len(m.enabledIndexes))
	err := m.db.View(func(dbTx database.Tx) error {
		// None of the indexes needs to be dropped if the index tips
		// bucket hasn't been created yet.
		indexesBucket := dbTx.Metadata().Bucket(indexTipsBucketName)
		if indexesBucket == nil {
			return nil
		}

		// Mark the indexer as requiring a drop if one is already in
		// progress.
		for i, indexer := range m.enabledIndexes {
			dropKey := indexDropKey(indexer.Key())
			if indexesBucket.Get(dropKey) != nil {
				indexNeedsDrop[i] = true
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	if interruptRequested(interrupt) {
		return errInterruptRequested
	}

	// Finish dropping any of the enabled indexes that are already in the
	// middle of being dropped.
	for i, indexer := range m.enabledIndexes {
		if !indexNeedsDrop[i] {
			continue
		}

		log.Infof("Resuming %s drop", indexer.Name())
		err := dropIndex(m.db, indexer.Key(), indexer.Name(), interrupt)
		if err != nil {
			return err
		}
	}

	return nil
}

// maybeCreateIndexes determines if each of the enabled indexes have already
// been created and creates them if not.
func (m *Manager) maybeCreateIndexes(dbTx database.Tx) error {
	indexesBucket := dbTx.Metadata().Bucket(indexTipsBucketName)
	for _, indexer := range m.enabledIndexes {
		// Nothing to do if the index tip already exists.
		idxKey := indexer.Key()
		if indexesBucket.Get(idxKey) != nil {
			continue
		}

		// The tip for the index does not exist, so create it and
		// invoke the create callback for the index so it can perform
		// any one-time initialization it requires.
		if err := indexer.Create(dbTx); err != nil {
			return err
		}

		// Set the tip for the index to values which represent an
		// uninitialized index.
		err := dbPutIndexerTips(dbTx, idxKey, make([]*chainhash.Hash,0))
		if err != nil {
			return err
		}
	}

	return nil
}

// Init initializes the enabled indexes.  This is called during chain
// initialization and primarily consists of catching up all indexes to the
// current best chain tip.  This is necessary since each index can be disabled
// and re-enabled at any time and attempting to catch-up indexes at the same
// time new blocks are being downloaded would lead to an overall longer time to
// catch up due to the I/O contention.
//
// This is part of the blockchain.IndexManager interface.
func (m *Manager) Init(chain *blockdag.BlockDAG, interrupt <-chan struct{}) error {
	// Nothing to do when no indexes are enabled.
	if len(m.enabledIndexes) == 0 {
		return nil
	}

	if interruptRequested(interrupt) {
		return errInterruptRequested
	}

	// Finish and drops that were previously interrupted.
	if err := m.maybeFinishDrops(interrupt); err != nil {
		return err
	}

	// Create the initial state for the indexes as needed.
	err := m.db.Update(func(dbTx database.Tx) error {
		// Create the bucket for the current tips as needed.
		meta := dbTx.Metadata()
		_, err := meta.CreateBucketIfNotExists(indexTipsBucketName)
		if err != nil {
			return err
		}

		return m.maybeCreateIndexes(dbTx)
	})
	if err != nil {
		return err
	}

	// Initialize each of the enabled indexes.
	for _, indexer := range m.enabledIndexes {
		if err := indexer.Init(); err != nil {
			return err
		}
	}
	// jenlouie: All blocks should be in main chain

	// Fetch current DAG tips
	// If index has different tips, need to catch up
	tipHash := chain.DAGSnapshot().Hash
	needUpdate := false
	tips := make([][]*chainhash.Hash, len(m.enabledIndexes))
	err = m.db.View(func(dbTx database.Tx) error {
		for i, indexer := range m.enabledIndexes {
			idxKey := indexer.Key()
			indexerTips, err := dbFetchIndexerTips(dbTx, idxKey)
			if err != nil {
				return err
			}
			tips[i] = indexerTips
			indexerTipsHash := blockdag.GenerateTipsHash(indexerTips)
			if !indexerTipsHash.IsEqual(&tipHash) {
				needUpdate = true
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Nothing to index if all of the indexes are caught up.
	if !needUpdate {
		return nil
	}

	// Create a progress logger for the indexing process below.
	progressLogger := newBlockProgressLogger("Indexed", log)

	// At this point, one or more indexes are behind the current best chain
	// tip and need to be caught up, so log the details and loop through
	// each block that needs to be indexed.
	log.Infof("Catching up indexes to tips %v", tipHash)

	// Getting missing blocks from each index
	// Keep track of which index is missing which block
	missingHashesIndex := make(map[chainhash.Hash][]int)
	hashHeight := make(map[chainhash.Hash]int32)
	for i := range m.enabledIndexes {
		indexerTips := tips[i]
		indexerTipsHash := blockdag.GenerateTipsHash(indexerTips)
		if indexerTipsHash.IsEqual(&tipHash) {
			continue
		}

		missingHashes := chain.GetBlockDiff(indexerTips)
		for _, hash := range missingHashes {
			if interruptRequested(interrupt) {
				return errInterruptRequested
			}
			missingHashesIndex[*hash] = append(missingHashesIndex[*hash], i)
			height, err := chain.BlockHeightByHash(hash)
				if err != nil {
					return err
				}
			hashHeight[*hash] = height
		}
	}
	log.Debugf("Missing hash index %v", missingHashesIndex)

	// sort by height
	type kv struct {
		Key   chainhash.Hash
		Value int32
	}

	var ss []kv
	for k, v := range hashHeight {
		ss = append(ss, kv{k, v})
	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value < ss[j].Value
	})


	// loop through missing blocks by height
	for _, entry := range ss {

		hash := entry.Key
		idxarr := missingHashesIndex[hash]

		if interruptRequested(interrupt) {
			return errInterruptRequested
		}

		// Fetch block
		block, err := chain.BlockByHash(&hash)
		if err != nil {
			return err
		}

		// Get spend journal if needed
		// and add block to index
		var spentTxos []blockdag.SpentTxOut
		for _, idxidx := range idxarr {
			indexer := m.enabledIndexes[idxidx]
			if spentTxos == nil && indexNeedsInputs(indexer) {
				spentTxos, err = chain.FetchSpendJournal(block)
				if err != nil {
					return err
				}
			}

			err := m.db.Update(func(dbTx database.Tx) error {
				return dbIndexConnectBlock(
					dbTx, indexer, block, spentTxos,
				)
			})
			if err != nil {
				return err
			}
			log.Debugf("Added block %v to indexer %v", block.Hash(), indexer.Name())
			// Log indexing progress.
			progressLogger.LogBlockHeight(block)
		}

		if interruptRequested(interrupt) {
			return errInterruptRequested
		}
	}

	log.Infof("Indexes caught up to tips %v", chain.DAGSnapshot().Tips)
	return nil
}

// indexNeedsInputs returns whether or not the index needs access to the txouts
// referenced by the transaction inputs being indexed.
func indexNeedsInputs(index Indexer) bool {
	if idx, ok := index.(NeedsInputser); ok {
		return idx.NeedsInputs()
	}

	return false
}

// dbFetchTx looks up the passed transaction hash in the transaction index and
// loads it from the database.
func dbFetchTx(dbTx database.Tx, hash *chainhash.Hash) (*wire.MsgTx, error) {
	// Look up the location of the transaction.
	blockRegion, err := dbFetchTxIndexEntry(dbTx, hash)
	if err != nil {
		return nil, err
	}
	if blockRegion == nil {
		return nil, fmt.Errorf("transaction %v not found", hash)
	}

	// Load the raw transaction bytes from the database.
	txBytes, err := dbTx.FetchBlockRegion(blockRegion)
	if err != nil {
		return nil, err
	}

	// Deserialize the transaction.
	var msgTx wire.MsgTx
	err = msgTx.Deserialize(bytes.NewReader(txBytes))
	if err != nil {
		return nil, err
	}

	return &msgTx, nil
}

// ConnectBlock must be invoked when a block is extending the main chain.  It
// keeps track of the state of each index it is managing, performs some sanity
// checks, and invokes each indexer.
//
// This is part of the blockchain.IndexManager interface.
func (m *Manager) ConnectBlock(dbTx database.Tx, block *soterutil.Block,
	stxos []blockdag.SpentTxOut) error {

	// Call each of the currently active optional indexes with the block
	// being connected so they can update accordingly.
	for _, index := range m.enabledIndexes {
		err := dbIndexConnectBlock(dbTx, index, block, stxos)
		if err != nil {
			return err
		}
	}
	return nil
}

// NewManager returns a new index manager with the provided indexes enabled.
//
// The manager returned satisfies the blockchain.IndexManager interface and thus
// cleanly plugs into the normal blockchain processing path.
func NewManager(db database.DB, enabledIndexes []Indexer) *Manager {
	return &Manager{
		db:             db,
		enabledIndexes: enabledIndexes,
	}
}

// dropIndex drops the passed index from the database.  Since indexes can be
// massive, it deletes the index in multiple database transactions in order to
// keep memory usage to reasonable levels.  It also marks the drop in progress
// so the drop can be resumed if it is stopped before it is done before the
// index can be used again.
func dropIndex(db database.DB, idxKey []byte, idxName string, interrupt <-chan struct{}) error {
	// Nothing to do if the index doesn't already exist.
	var needsDelete bool
	err := db.View(func(dbTx database.Tx) error {
		indexesBucket := dbTx.Metadata().Bucket(indexTipsBucketName)
		if indexesBucket != nil && indexesBucket.Get(idxKey) != nil {
			needsDelete = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	if !needsDelete {
		log.Infof("Not dropping %s because it does not exist", idxName)
		return nil
	}

	// Mark that the index is in the process of being dropped so that it
	// can be resumed on the next start if interrupted before the process is
	// complete.
	log.Infof("Dropping all %s entries.  This might take a while...",
		idxName)
	err = db.Update(func(dbTx database.Tx) error {
		indexesBucket := dbTx.Metadata().Bucket(indexTipsBucketName)
		return indexesBucket.Put(indexDropKey(idxKey), idxKey)
	})
	if err != nil {
		return err
	}

	// Since the indexes can be so large, attempting to simply delete
	// the bucket in a single database transaction would result in massive
	// memory usage and likely crash many systems due to ulimits.  In order
	// to avoid this, use a cursor to delete a maximum number of entries out
	// of the bucket at a time. Recurse buckets depth-first to delete any
	// sub-buckets.
	const maxDeletions = 2000000
	var totalDeleted uint64

	// Recurse through all buckets in the index, cataloging each for
	// later deletion.
	var subBuckets [][][]byte
	var subBucketClosure func(database.Tx, []byte, [][]byte) error
	subBucketClosure = func(dbTx database.Tx,
		subBucket []byte, tlBucket [][]byte) error {
		// Get full bucket name and append to subBuckets for later
		// deletion.
		var bucketName [][]byte
		if (tlBucket == nil) || (len(tlBucket) == 0) {
			bucketName = append(bucketName, subBucket)
		} else {
			bucketName = append(tlBucket, subBucket)
		}
		subBuckets = append(subBuckets, bucketName)
		// Recurse sub-buckets to append to subBuckets slice.
		bucket := dbTx.Metadata()
		for _, subBucketName := range bucketName {
			bucket = bucket.Bucket(subBucketName)
		}
		return bucket.ForEachBucket(func(k []byte) error {
			return subBucketClosure(dbTx, k, bucketName)
		})
	}

	// Call subBucketClosure with top-level bucket.
	err = db.View(func(dbTx database.Tx) error {
		return subBucketClosure(dbTx, idxKey, nil)
	})
	if err != nil {
		return nil
	}

	// Iterate through each sub-bucket in reverse, deepest-first, deleting
	// all keys inside them and then dropping the buckets themselves.
	for i := range subBuckets {
		bucketName := subBuckets[len(subBuckets)-1-i]
		// Delete maxDeletions key/value pairs at a time.
		for numDeleted := maxDeletions; numDeleted == maxDeletions; {
			numDeleted = 0
			err := db.Update(func(dbTx database.Tx) error {
				subBucket := dbTx.Metadata()
				for _, subBucketName := range bucketName {
					subBucket = subBucket.Bucket(subBucketName)
				}
				cursor := subBucket.Cursor()
				for ok := cursor.First(); ok; ok = cursor.Next() &&
					numDeleted < maxDeletions {

					if err := cursor.Delete(); err != nil {
						return err
					}
					numDeleted++
				}
				return nil
			})
			if err != nil {
				return err
			}

			if numDeleted > 0 {
				totalDeleted += uint64(numDeleted)
				log.Infof("Deleted %d keys (%d total) from %s",
					numDeleted, totalDeleted, idxName)
			}
		}

		if interruptRequested(interrupt) {
			return errInterruptRequested
		}

		// Drop the bucket itself.
		err = db.Update(func(dbTx database.Tx) error {
			bucket := dbTx.Metadata()
			for j := 0; j < len(bucketName)-1; j++ {
				bucket = bucket.Bucket(bucketName[j])
			}
			return bucket.DeleteBucket(bucketName[len(bucketName)-1])
		})
	}

	// Call extra index specific deinitialization for the transaction index.
	if idxName == txIndexName {
		if err := dropBlockIDIndex(db); err != nil {
			return err
		}
	}

	// Remove the index tip, index bucket, and in-progress drop flag now
	// that all index entries have been removed.
	err = db.Update(func(dbTx database.Tx) error {
		meta := dbTx.Metadata()
		indexesBucket := meta.Bucket(indexTipsBucketName)
		if err := indexesBucket.Delete(idxKey); err != nil {
			return err
		}

		return indexesBucket.Delete(indexDropKey(idxKey))
	})
	if err != nil {
		return err
	}

	log.Infof("Dropped %s", idxName)
	return nil
}
