// Copyright (c) 2013-2018 The btcsuite developers
// Copyright (c) 2015-2018 The Decred developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdag

import (
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/chaincfg/chainhash"
	"github.com/soteria-dag/soterd/database"
	"github.com/soteria-dag/soterd/wire"
)

// blockStatus is a bit field representing the validation state of the block.
type blockStatus byte

const (
	// statusDataStored indicates that the block's payload is stored on disk.
	statusDataStored blockStatus = 1 << iota

	// statusValid indicates that the block has been fully validated.
	statusValid

	// statusValidateFailed indicates that the block has failed validation.
	statusValidateFailed

	// statusInvalidAncestor indicates that one of the block's ancestors has
	// has failed validation, thus the block is also invalid.
	statusInvalidAncestor

	// statusNone indicates that the block has no validation state flags set.
	//
	// NOTE: This must be defined last in order to avoid influencing iota.
	statusNone blockStatus = 0
)

// HaveData returns whether the full block data is stored in the database. This
// will return false for a block node where only the header is downloaded or
// kept.
func (status blockStatus) HaveData() bool {
	return status&statusDataStored != 0
}

// KnownValid returns whether the block is known to be valid. This will return
// false for a valid block that has not been fully validated yet.
func (status blockStatus) KnownValid() bool {
	return status&statusValid != 0
}

// KnownInvalid returns whether the block is known to be invalid. This may be
// because the block itself failed validation or any of its ancestors is
// invalid. This will return false for invalid blocks that have not been proven
// invalid yet.
func (status blockStatus) KnownInvalid() bool {
	return status&(statusValidateFailed|statusInvalidAncestor) != 0
}

// blockNode represents a block within the DAG
type blockNode struct {
	// NOTE: Additions, deletions, or modifications to the order of the
	// definitions in this struct should not be changed without considering
	// how it affects alignment on 64-bit platforms.  The current order is
	// specifically crafted to result in minimal padding.  There will be
	// hundreds of thousands of these in memory, so a few extra bytes of
	// padding adds up.

	// parents are the parent block for this node.
	parents []*blockNode

	// parent metadata like version and extra data
	parentMetadata []*parentInfo

	parentVersion int32

	// hash is the double sha 256 of the block.
	hash chainhash.Hash

	// workSum is the total amount of work in the chain up to and including
	// this node.
	workSum *big.Int

	// height is parentsMaxHeight + 1
	height int32

	// Some fields from block headers to
	// reconstruct headers from memory.  These must be treated as
	// immutable and are intentionally ordered to avoid padding on 64-bit
	// platforms.
	version    int32
	bits       uint32
	nonce      uint32
	timestamp  int64
	merkleRoot chainhash.Hash

	// status is a bitfield representing the validation state of the block. The
	// status field, unlike the other fields, may be written to and so should
	// only be accessed using the concurrent-safe NodeStatus method on
	// blockIndex once the node has been added to the global index.
	status blockStatus
}

type parentInfo struct {
	hash chainhash.Hash
	data [32]byte
}

// initBlockNode initializes a block node from the given header and parent node,
// calculating the height and workSum from the respective fields on the parent.
// This function is NOT safe for concurrent access.  It must only be called when
// initially creating a node.
func initBlockNode(node *blockNode, blockHeader *wire.BlockHeader,
	parentSubHeader *wire.ParentSubHeader, parents []*blockNode) {
	*node = blockNode{
		hash:       blockHeader.BlockHash(),
		workSum:    CalcWork(blockHeader.Bits),
		version:    blockHeader.Version,
		bits:       blockHeader.Bits,
		nonce:      blockHeader.Nonce,
		timestamp:  blockHeader.Timestamp.Unix(),
		merkleRoot: blockHeader.MerkleRoot,
	}

	if parentSubHeader != nil {
		node.parentVersion = parentSubHeader.Version
		parentInfos := make([]*parentInfo, 0, len(parentSubHeader.Parents))
		for _, p := range parentSubHeader.Parents {
			info := &parentInfo{
				hash: p.Hash,
				data: p.Data,
			}
			parentInfos = append(parentInfos, info)
		}
		node.parentMetadata = parentInfos
	}

	if parents != nil {
		node.parents = parents
		// height = max height of parents + 1
		node.height = node.parentsMaxHeight() + 1
		//node.workSum = node.workSum.Add(parent.workSum, node.workSum)
	}
}

// newBlockNode returns a new block node for the given block header and parent
// node, calculating the height and workSum from the respective fields on the
// parent. This function is NOT safe for concurrent access.
func newBlockNode(blockHeader *wire.BlockHeader, parentSubHeader *wire.ParentSubHeader,
	parents []*blockNode) *blockNode {
	var node blockNode
	initBlockNode(&node, blockHeader, parentSubHeader, parents)
	return &node
}

// parentsMaxHeight returns the maximum height of the parent blocks of a block node.
func (node *blockNode) parentsMaxHeight() int32 {
	if node.parents == nil {
		return int32(0)
	}

	var maxHeight int32
	for _, parent := range node.parents {
		if parent.height > maxHeight {
			maxHeight = parent.height
		}
	}
	return maxHeight
}

// TODO: move to util??
func generateTipsHash(tips []*blockNode) *chainhash.Hash {
	// sort hashes
	tipHashes := make([]*chainhash.Hash, len(tips))
	//fmt.Printf("number of tips: %d\n", len(tips))
	//fmt.Printf("number of tip hashes: %d\n", len(tipHashes))
	for i, tip := range tips {
		tipHashes[i] = &tip.hash
	}

	sort.Slice(tipHashes, func(i, j int) bool {
		return tipHashes[i].String() < tipHashes[j].String()
	})

	//concatentate
	size := chainhash.HashSize * len(tipHashes)
	var concatBytes = make([]byte, size)
	for i, tipHash := range tipHashes {
		//fmt.Printf("tip hash %s\n", tipHash.String())
		start := i * chainhash.HashSize
		end := (i + 1) * chainhash.HashSize
		//fmt.Printf("start %d, end %d\n", start, end)
		copy(concatBytes[start:end], tipHash[:])
	}
	//fmt.Printf("concatbytes: %v\n", concatBytes)
	// double sha
	newHash := chainhash.DoubleHashH(concatBytes[:])
	//fmt.Printf("hash str %s\n", newHash.String())
	return &newHash
}

func GenerateTipsHash(hashes []*chainhash.Hash) *chainhash.Hash {
	// sort hashes
	sort.Slice(hashes, func(i, j int) bool {
		return hashes[i].String() < hashes[j].String()
	})

	//concatentate
	size := chainhash.HashSize * len(hashes)
	var concatBytes = make([]byte, size)
	for i, tipHash := range hashes {
		//fmt.Printf("tip hash %s\n", tipHash.String())
		start := i * chainhash.HashSize
		end := (i + 1) * chainhash.HashSize
		//fmt.Printf("start %d, end %d\n", start, end)
		copy(concatBytes[start:end], tipHash[:])
	}
	//fmt.Printf("concatbytes: %v\n", concatBytes)
	// double sha
	newHash := chainhash.DoubleHashH(concatBytes[:])
	//fmt.Printf("hash str %s\n", newHash.String())	r
	return &newHash
}

// Header constructs a block header from the node and returns it.
//
// This function is safe for concurrent access.
func (node *blockNode) Header() wire.BlockHeader {
	// No lock is needed because all accessed fields are immutable.
	prevHash := &zeroHash
	if node.parents != nil {
		prevHash = generateTipsHash(node.parents)
	}
	return wire.BlockHeader{
		Version:    node.version,
		PrevBlock:  *prevHash,
		MerkleRoot: node.merkleRoot,
		Timestamp:  time.Unix(node.timestamp, 0),
		Bits:       node.bits,
		Nonce:      node.nonce,
	}
}

func (node *blockNode) ParentSubHeader() wire.ParentSubHeader {
	parents := make([]*wire.Parent, len(node.parentMetadata))
	for i, pm := range node.parentMetadata {
		parents[i] = &wire.Parent{Hash: pm.hash, Data: pm.data}
	}

	return wire.ParentSubHeader{
		Version: node.parentVersion,
		Size:    int32(len(parents)),
		Parents: parents,
	}
}

// Ancestors returns the ancestor block nodes at the provided height by following
// the chain backwards from this node.  The returned block will be nil when a
// height is requested that is after the height of the passed node or is less
// than zero.
//
// This function is safe for concurrent access.
func (node *blockNode) Ancestors(height int32) []*blockNode {
	if height < 0 || height > node.height {
		return nil
	}

	parents := node.parents
	checked := make(map[chainhash.Hash]struct{})
	for _, parent := range parents {
		checked[parent.hash] = struct{}{}
	}
	parentsHeight := node.parentsMaxHeight()
	for {
		if parents == nil || len(parents) == 0 || parentsHeight <= height {
			break
		}

		var grandParentMaxHeight int32
		var grandParents []*blockNode
		for _, parent := range parents {
			grandParentsHeight := parent.parentsMaxHeight()
			if grandParentsHeight > grandParentMaxHeight {
				grandParentMaxHeight = grandParentsHeight
			}

			for _, grandParent := range parent.parents {
				_, exists := checked[grandParent.hash]
				if !exists {
					grandParents = append(grandParents, grandParent)
					checked[grandParent.hash] = struct{}{}
				}
			}
		}

		parents = grandParents
		parentsHeight = grandParentMaxHeight
	}

	return parents
}

// RelativeAncestors returns the ancestor block nodes at a relative 'distance' blocks
// before this node. This is equivalent to calling Ancestors with the nodes'
// height minus provided distance.
//
// This function is safe for concurrent access.
func (node *blockNode) RelativeAncestors(distance int32) []*blockNode {
	return node.Ancestors(node.height - distance)
}

// CalcPastMedianTime calculates the median time of the previous few blocks
// prior to, and including, the block node.
//
// This function is safe for concurrent access.
func (node *blockNode) CalcPastMedianTime() time.Time {
	// Create a slice of the previous few block timestamps used to calculate
	// the median per the number defined by the constant medianTimeBlocks.
	timestamps := make([]int64, medianTimeBlocks)
	numNodes := 0
	iterNode := node
	for i := 0; i < medianTimeBlocks && iterNode != nil; i++ {
		timestamps[i] = iterNode.timestamp
		numNodes++
		parents := iterNode.parents
		if parents != nil {
			iterNode = parents[0] // pick first parent
		} else {
			iterNode = nil
		}
	}

	// Prune the slice to the actual number of available timestamps which
	// will be fewer than desired near the beginning of the block chain
	// and sort them.
	timestamps = timestamps[:numNodes]
	sort.Sort(timeSorter(timestamps))

	// NOTE: The consensus rules incorrectly calculate the median for even
	// numbers of blocks.  A true median averages the middle two elements
	// for a set with an even number of elements in it.   Since the constant
	// for the previous number of blocks to be used is odd, this is only an
	// issue for a few blocks near the beginning of the chain.  I suspect
	// this is an optimization even though the result is slightly wrong for
	// a few of the first blocks since after the first few blocks, there
	// will always be an odd number of blocks in the set per the constant.
	//
	// This code follows suit to ensure the same rules are used, however, be
	// aware that should the medianTimeBlocks constant ever be changed to an
	// even number, this code will be wrong.
	medianTimestamp := timestamps[numNodes/2]
	return time.Unix(medianTimestamp, 0)
}

// blockIndex provides facilities for keeping track of an in-memory index of the
// block chain.  Although the name block chain suggests a single chain of
// blocks, it is actually a tree-shaped structure where any node can have
// multiple children.  However, there can only be one active branch which does
// indeed form a chain from the tip all the way back to the genesis block.
type blockIndex struct {
	// The following fields are set when the instance is created and can't
	// be changed afterwards, so there is no need to protect them with a
	// separate mutex.
	db          database.DB
	chainParams *chaincfg.Params

	sync.RWMutex
	index map[chainhash.Hash]*blockNode
	dirty map[*blockNode]struct{}
}

// newBlockIndex returns a new empty instance of a block index.  The index will
// be dynamically populated as block nodes are loaded from the database and
// manually added.
func newBlockIndex(db database.DB, chainParams *chaincfg.Params) *blockIndex {
	return &blockIndex{
		db:          db,
		chainParams: chainParams,
		index:       make(map[chainhash.Hash]*blockNode),
		dirty:       make(map[*blockNode]struct{}),
	}
}

// HaveBlock returns whether or not the block index contains the provided hash.
//
// This function is safe for concurrent access.
func (bi *blockIndex) HaveBlock(hash *chainhash.Hash) bool {
	bi.RLock()
	_, hasBlock := bi.index[*hash]
	bi.RUnlock()
	return hasBlock
}

// LookupNode returns the block node identified by the provided hash.  It will
// return nil if there is no entry for the hash.
//
// This function is safe for concurrent access.
func (bi *blockIndex) LookupNode(hash *chainhash.Hash) *blockNode {
	bi.RLock()
	node := bi.index[*hash]
	bi.RUnlock()
	return node
}

// AddNode adds the provided node to the block index and marks it as dirty.
// Duplicate entries are not checked so it is up to caller to avoid adding them.
//
// This function is safe for concurrent access.
func (bi *blockIndex) AddNode(node *blockNode) {
	bi.Lock()
	bi.addNode(node)
	bi.dirty[node] = struct{}{}
	bi.Unlock()
}

// addNode adds the provided node to the block index, but does not mark it as
// dirty. This can be used while initializing the block index.
//
// This function is NOT safe for concurrent access.
func (bi *blockIndex) addNode(node *blockNode) {
	bi.index[node.hash] = node
}

// NodeStatus provides concurrent-safe access to the status field of a node.
//
// This function is safe for concurrent access.
func (bi *blockIndex) NodeStatus(node *blockNode) blockStatus {
	bi.RLock()
	status := node.status
	bi.RUnlock()
	return status
}

// SetStatusFlags flips the provided status flags on the block node to on,
// regardless of whether they were on or off previously. This does not unset any
// flags currently on.
//
// This function is safe for concurrent access.
func (bi *blockIndex) SetStatusFlags(node *blockNode, flags blockStatus) {
	bi.Lock()
	node.status |= flags
	bi.dirty[node] = struct{}{}
	bi.Unlock()
}

// UnsetStatusFlags flips the provided status flags on the block node to off,
// regardless of whether they were on or off previously.
//
// This function is safe for concurrent access.
func (bi *blockIndex) UnsetStatusFlags(node *blockNode, flags blockStatus) {
	bi.Lock()
	node.status &^= flags
	bi.dirty[node] = struct{}{}
	bi.Unlock()
}

// flushToDB writes all dirty block nodes to the database. If all writes
// succeed, this clears the dirty set.
func (bi *blockIndex) flushToDB() error {
	bi.Lock()
	if len(bi.dirty) == 0 {
		bi.Unlock()
		return nil
	}

	err := bi.db.Update(func(dbTx database.Tx) error {
		for node := range bi.dirty {
			err := dbStoreBlockNode(dbTx, node)
			if err != nil {
				return err
			}
		}
		return nil
	})

	// If write was successful, clear the dirty set.
	if err == nil {
		bi.dirty = make(map[*blockNode]struct{})
	}

	bi.Unlock()
	return err
}
