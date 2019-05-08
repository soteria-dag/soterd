// Copyright (c) 2017 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdag

import (
	"github.com/soteria-dag/soterd/chaincfg/chainhash"
	"sort"
	"sync"
)

// approxNodesPerWeek is an approximation of the number of new blocks there are
// in a week on average.
const approxNodesPerWeek = 6 * 24 * 7 // TODO: update based on k

// log2FloorMasks defines the masks to use when quickly calculating
// floor(log2(x)) in a constant log2(32) = 5 steps, where x is a uint32, using
// shifts.  They are derived from (2^(2^x) - 1) * (2^(2^x)), for x in 4..0.
var log2FloorMasks = []uint32{0xffff0000, 0xff00, 0xf0, 0xc, 0x2}

// fastLog2Floor calculates and returns floor(log2(x)) in a constant 5 steps.
func fastLog2Floor(n uint32) uint8 {
	rv := uint8(0)
	exponent := uint8(16)
	for i := 0; i < 5; i++ {
		if n&log2FloorMasks[i] != 0 {
			rv += exponent
			n >>= exponent
		}
		exponent >>= 1
	}
	return rv
}

// chainView provides a flat view of a specific branch of the block chain from
// its tip back to the genesis block and provides various convenience functions
// for comparing chains.
//
// For example, assume a block chain with a side chain as depicted below:
//   genesis -> 1 -> 2 -> 3 -> 4  -> 5 ->  6  -> 7  -> 8
//                         \-> 4a -> 5a -> 6a
//
// The chain view for the branch ending in 6a consists of:
//   genesis -> 1 -> 2 -> 3 -> 4a -> 5a -> 6a
type dagView struct {
	mtx      sync.Mutex
	dagTips  map[*blockNode]struct{}
	nodes    [][]*blockNode
}

// newChainView returns a new chain view for the given tip block node.  Passing
// nil as the tip will result in a chain view that is not initialized.  The tip
// can be updated at any time via the setTip function.
func newDAGView(tips []*blockNode) *dagView {
	// The mutex is intentionally not held since this is a constructor.
	var c dagView
	c.nodes = make([][]*blockNode, 0)
	c.dagTips = make(map[*blockNode]struct{})
	if tips != nil && len(tips) > 0 {
		for _, tip := range tips {
			c.addTip(tip)
		}
	}
	return &c
}

// genesis returns the genesis block for the chain view.  This only differs from
// the exported version in that it is up to the caller to ensure the lock is
// held.
//
// This function MUST be called with the view mutex locked (for reads).
func (c *dagView) genesis() *blockNode {
	if len(c.nodes) == 0 {
		return nil
	}

	return c.nodes[0][0]
}

// Genesis returns the genesis block for the chain view.
//
// This function is safe for concurrent access.
func (c *dagView) Genesis() *blockNode {
	c.mtx.Lock()
	genesis := c.genesis()
	c.mtx.Unlock()
	return genesis
}

// tip returns the current tip block node for the chain view.  It will return
// nil if there is no tip.  This only differs from the exported version in that
// it is up to the caller to ensure the lock is held.
//
// This function MUST be called with the view mutex locked (for reads).
func (c *dagView) tips() []*blockNode {
	if len(c.nodes) == 0 {
		return nil
	}

	tips := make([]*blockNode, 0, len(c.dagTips))
	for tip := range c.dagTips {
		tips = append(tips, tip)
	}

	sort.Sort(blockSorter(tips))

	return tips
}

// Tip returns the current tip block node for the chain view.  It will return
// nil if there is no tip.
//
// This function is safe for concurrent access.
func (c *dagView) Tips() []*blockNode {
	c.mtx.Lock()
	tip := c.tips()
	c.mtx.Unlock()
	return tip
}

// setTip sets the chain view to use the provided block node as the current tip
// and ensures the view is consistent by populating it with the nodes obtained
// by walking backwards all the way to genesis block as necessary.  Further
// calls will only perform the minimum work needed, so switching between chain
// tips is efficient.  This only differs from the exported version in that it is
// up to the caller to ensure the lock is held.
//
// This function MUST be called with the view mutex locked (for writes).
/*func (c *chainView) setTip(node *blockNode) {
	if node == nil {
		// Keep the backing array around for potential future use.
		c.nodes = c.nodes[:0]
		return
	}

	// Create or resize the slice that will hold the block nodes to the
	// provided tip height.  When creating the slice, it is created with
	// some additional capacity for the underlying array as append would do
	// in order to reduce overhead when extending the chain later.  As long
	// as the underlying array already has enough capacity, simply expand or
	// contract the slice accordingly.  The additional capacity is chosen
	// such that the array should only have to be extended about once a
	// week.
	needed := node.height + 1
	if int32(cap(c.nodes)) < needed {
		nodes := make([]*blockNode, needed, needed+approxNodesPerWeek)
		copy(nodes, c.nodes)
		c.nodes = nodes
	} else {
		prevLen := int32(len(c.nodes))
		c.nodes = c.nodes[0:needed]
		for i := prevLen; i < needed; i++ {
			c.nodes[i] = nil
		}
	}

	for node != nil && c.nodes[node.height] != node {
		c.nodes[node.height] = node
		node = node.parent
	}
}

// SetTip sets the chain view to use the provided block node as the current tip
// and ensures the view is consistent by populating it with the nodes obtained
// by walking backwards all the way to genesis block as necessary.  Further
// calls will only perform the minimum work needed, so switching between chain
// tips is efficient.
//
// This function is safe for concurrent access.
func (c *chainView) SetTip(node *blockNode) {
	c.mtx.Lock()
	c.setTip(node)
	c.mtx.Unlock()
}*/

func (c *dagView) addTip(node *blockNode) {
	// Create or resize the slice that will hold the block nodes to the
	// provided tip height.  When creating the slice, it is created with
	// some additional capacity for the underlying array as append would do
	// in order to reduce overhead when extending the chain later.  As long
	// as the underlying array already has enough capacity, simply expand or
	// contract the slice accordingly.  The additional capacity is chosen
	// such that the array should only have to be extended about once a
	// week.
	needed := node.height + 1
	//fmt.Printf("array size needed: %d\n", needed)
	if int32(cap(c.nodes)) < needed {
		nodes := make([][]*blockNode, needed, needed+approxNodesPerWeek)
		copy(nodes, c.nodes)
		c.nodes = nodes
	} else {
		prevLen := int32(len(c.nodes))
		if prevLen < needed {
			c.nodes = c.nodes[0:needed]
		}
	}

	if node != nil {
		// populate nodes
		c.checkAndAddNode(node.height, node)

		// add to tip set
		c.dagTips[node] = struct{}{}
	}
	//fmt.Printf("len %d\n", len(c.nodes[0]))
}

func (c *dagView) AddTip(node *blockNode) {
	c.mtx.Lock()
	c.addTip(node)
	c.mtx.Unlock()
}

func (c *dagView) checkAndAddNode(height int32, node *blockNode) bool {
	if node != nil && !c.containsAtHeight(node.height, node) {
		// add in sorted order of hashes
		nodes := c.nodes[node.height]
		nodes = append(nodes, node)
		for i, nodeH := range nodes {
			if node.hash.String() < nodeH.hash.String() { //TODO: better way to compare hashes?
				copy(nodes[i+1:], nodes[i:])
				nodes[i] = node
				break
			}
		}

		c.nodes[node.height] = nodes
		// add parents if they are missing as well
		parents := node.parents
		for _, parent := range parents {
			c.checkAndAddNode(parent.height, parent)
		}
		return true
	}

	return false
}

func (c *dagView) removeTip(node *blockNode) {
	// remove from tip set
	for tip := range c.dagTips {
		if node == tip {
			delete(c.dagTips, node)
		}
	}
}

func (c *dagView) RemoveTip(node *blockNode) {
	c.mtx.Lock()
	c.removeTip(node)
	c.mtx.Unlock()

}

func (c *dagView) containsAtHeight(height int32, node *blockNode) bool {
	if height > int32(len(c.nodes) - 1) {
		return false
	}

	nodes := c.nodes[height]
	for _, nodeH := range nodes {
		if nodeH == node {
			return true
		}
	}
	return false
}

// height returns the height of the tip of the chain view.  It will return -1 if
// there is no tip (which only happens if the chain view has not been
// initialized).  This only differs from the exported version in that it is up
// to the caller to ensure the lock is held.
//
// This function MUST be called with the view mutex locked (for reads).
func (c *dagView) height() int32 {
	return int32(len(c.nodes) - 1)
}

// Height returns the height of the tip of the chain view.  It will return -1 if
// there is no tip (which only happens if the chain view has not been
// initialized).
//
// This function is safe for concurrent access.
func (c *dagView) Height() int32 {
	c.mtx.Lock()
	height := c.height()
	c.mtx.Unlock()
	return height
}

// nodeByHeight returns the block node at the specified height.  Nil will be
// returned if the height does not exist.  This only differs from the exported
// version in that it is up to the caller to ensure the lock is held.
//
// This function MUST be called with the view mutex locked (for reads).
func (c *dagView) nodesByHeight(height int32) []*blockNode {
	if height < 0 || height >= int32(len(c.nodes)) {
		return nil
	}

	nodes := c.nodes[height]
	if nodes != nil && len(nodes) > 1 {
		sort.Sort(blockSorter(nodes))
	}

	return nodes
}

// NodeByHeight returns the block node at the specified height.  Nil will be
// returned if the height does not exist.
//
// This function is safe for concurrent access.
func (c *dagView) NodesByHeight(height int32) []*blockNode {
	c.mtx.Lock()
	nodes := c.nodesByHeight(height)
	c.mtx.Unlock()
	return nodes
}


func (c *dagView) virtualHash() *chainhash.Hash {
	return generateTipsHash(c.tips())
}

func (c *dagView) count() int {
	var nodeCount = 0
	for _, height := range c.nodes {
		nodeCount += len(height)
	}
	return nodeCount
}
// Equals returns whether or not two chain views are the same.  Uninitialized
// views (tip set to nil) are considered equal.
//
// This function is safe for concurrent access.
func (c *dagView) Equals(other *dagView) bool {
	c.mtx.Lock()
	other.mtx.Lock()
	//fmt.Printf("len1: %d, len2: %d\n", len(c.nodes), len(other.nodes))
	//fmt.Printf("hash equals : %v\n", c.virtualHash().IsEqual(other.virtualHash()))
	equals := len(c.nodes) == len(other.nodes) && c.virtualHash().IsEqual(other.virtualHash())
	other.mtx.Unlock()
	c.mtx.Unlock()
	return equals
}

// contains returns whether or not the chain view contains the passed block
// node.  This only differs from the exported version in that it is up to the
// caller to ensure the lock is held.
//
// This function MUST be called with the view mutex locked (for reads).
func (c *dagView) contains(node *blockNode) bool {
	return c.containsAtHeight(node.height, node)
}

// Contains returns whether or not the chain view contains the passed block
// node.
//
// This function is safe for concurrent access.
func (c *dagView) Contains(node *blockNode) bool {
	c.mtx.Lock()
	contains := c.contains(node)
	c.mtx.Unlock()
	return contains
}

// next returns the successor to the provided node for the chain view.  It will
// return nil if there is no successor or the provided node is not part of the
// view.  This only differs from the exported version in that it is up to the
// caller to ensure the lock is held.
//
// See the comment on the exported function for more details.
//
// This function MUST be called with the view mutex locked (for reads).
func (c dagView) next(node *blockNode) *blockNode {
	if node == nil || !c.contains(node) {
		return nil
	}

	nodes := c.nodesByHeight(node.height)
	for i, nodeH := range nodes {
		if nodeH == node && i < (len(nodes) - 1) {
			return nodes[i+1]
		}
	}

	nodes = c.nodesByHeight(node.height + 1)
	if nodes == nil {
		return nil
	} else {
		return nodes[0]
	}
}

// Next returns the successor to the provided node for the chain view.  It will
// return nil if there is no successor or the provided node is not part of the
// view.
//
// For example, assume a block chain with a side chain as depicted below:
//   genesis -> 1 -> 2 -> 3 -> 4  -> 5 ->  6  -> 7  -> 8
//                         \-> 4a -> 5a -> 6a
//
// Further, assume the view is for the longer chain depicted above.  That is to
// say it consists of:
//   genesis -> 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8
//
// Invoking this function with block node 5 would return block node 6 while
// invoking it with block node 5a would return nil since that node is not part
// of the view.
//
// This function is safe for concurrent access.
func (c *dagView) Next(node *blockNode) *blockNode {
	c.mtx.Lock()
	next := c.next(node)
	c.mtx.Unlock()
	return next
}

// findFork returns the final common block between the provided node and the
// the chain view.  It will return nil if there is no common block.  This only
// differs from the exported version in that it is up to the caller to ensure
// the lock is held.
//
// See the exported FindFork comments for more details.
//
// This function MUST be called with the view mutex locked (for reads).
/*
func (c *chainView) findFork(node *blockNode) *blockNode {
	// No fork point for node that doesn't exist.
	if node == nil {
		return nil
	}

	// When the height of the passed node is higher than the height of the
	// tip of the current chain view, walk backwards through the nodes of
	// the other chain until the heights match (or there or no more nodes in
	// which case there is no common node between the two).
	//
	// NOTE: This isn't strictly necessary as the following section will
	// find the node as well, however, it is more efficient to avoid the
	// contains check since it is already known that the common node can't
	// possibly be past the end of the current chain view.  It also allows
	// this code to take advantage of any potential future optimizations to
	// the Ancestor function such as using an O(log n) skip list.
	chainHeight := c.height()
	if node.height > chainHeight {
		node = node.Ancestor(chainHeight)
	}

	// Walk the other chain backwards as long as the current one does not
	// contain the node or there are no more nodes in which case there is no
	// common node between the two.
	for node != nil && !c.contains(node) {
		node = node.parent
	}

	return node
}
*/
// FindFork returns the final common block between the provided node and the
// the chain view.  It will return nil if there is no common block.
//
// For example, assume a block chain with a side chain as depicted below:
//   genesis -> 1 -> 2 -> ... -> 5 -> 6  -> 7  -> 8
//                                \-> 6a -> 7a
//
// Further, assume the view is for the longer chain depicted above.  That is to
// say it consists of:
//   genesis -> 1 -> 2 -> ... -> 5 -> 6 -> 7 -> 8.
//
// Invoking this function with block node 7a would return block node 5 while
// invoking it with block node 7 would return itself since it is already part of
// the branch formed by the view.
//
// This function is safe for concurrent access.
/*
func (c *chainView) FindFork(node *blockNode) *blockNode {
	c.mtx.Lock()
	fork := c.findFork(node)
	c.mtx.Unlock()
	return fork
}
*/

// blockLocator returns a block locator for the passed block node.  The passed
// node can be nil in which case the block locator for the current tip
// associated with the view will be returned.  This only differs from the
// exported version in that it is up to the caller to ensure the lock is held.
//
// See the exported BlockLocator function comments for more details.
//
// This function MUST be called with the view mutex locked (for reads).
func (c *dagView) blockLocator(node *blockNode) BlockLocator {
	// Use the current tip if requested.
	var maxHeight int32
	if node == nil {
		tips := c.tips()
		maxHeight = tips[0].height
		node = tips[0]
		for _, tip := range tips {
			if tip.height > maxHeight {
				maxHeight = tip.height
				node = tip
			}
		}
	}
	if node == nil {
		// We weren't able to locate any block
		return nil
	}

	if node.height > maxHeight {
		maxHeight = node.height
	}
	locator := make(BlockLocator, 0, 1)
	locator = append(locator, &maxHeight)

	return locator
}

// BlockLocator returns a block locator for the passed block node.  The passed
// node can be nil in which case the block locator for the current tip
// associated with the view will be returned.
//
// See the BlockLocator type for details on the algorithm used to create a block
// locator.
//
// This function is safe for concurrent access.
func (c *dagView) BlockLocator(node *blockNode) BlockLocator {
	c.mtx.Lock()
	locator := c.blockLocator(node)
	c.mtx.Unlock()
	return locator
}

