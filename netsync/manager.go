// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package netsync

import (
	"container/list"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"

	// "time"

	"github.com/soteria-dag/soterd/blockdag"
	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/chaincfg/chainhash"
	"github.com/soteria-dag/soterd/database"
	"github.com/soteria-dag/soterd/mempool"
	peerpkg "github.com/soteria-dag/soterd/peer"
	"github.com/soteria-dag/soterd/soterutil"
	"github.com/soteria-dag/soterd/wire"
)

const (
	// minInFlightBlocks is the minimum number of blocks that should be
	// in the request queue for headers-first mode before requesting
	// more.
	minInFlightBlocks = 10

	// maxRejectedTxns is the maximum number of rejected transactions
	// hashes to store in memory.
	maxRejectedTxns = 1000

	// maxRequestedBlocks is the maximum number of requested block
	// hashes to store in memory.
	maxRequestedBlocks = wire.MaxInvPerMsg

	// maxRequestedTxns is the maximum number of requested transactions
	// hashes to store in memory.
	maxRequestedTxns = wire.MaxInvPerMsg

	// maxStallDuration is the time after which we will disconnect our
	// current sync peer if we haven't made progress.
	maxStallDuration = 3 * time.Minute

	// stallSampleInterval the interval at which we will check to see if our sync has stalled.
	// It's also used to determine if a request is eligible to retry
	stallSampleInterval = 30 * time.Second
)

// zeroHash is the zero value hash (all zeros).  It is defined as a convenience.
var zeroHash chainhash.Hash

// newPeerMsg signifies a newly connected peer to the block handler.
type newPeerMsg struct {
	peer *peerpkg.Peer
}

// blockMsg packages a soter block message and the peer it came from together
// so the block handler has access to that information.
type blockMsg struct {
	block *soterutil.Block
	peer  *peerpkg.Peer
	reply chan struct{}
}

// invMsg packages a soter inv message and the peer it came from together
// so the block handler has access to that information.
type invMsg struct {
	inv  *wire.MsgInv
	peer *peerpkg.Peer
}

// headersMsg packages a soter headers message and the peer it came from
// together so the block handler has access to that information.
type headersMsg struct {
	headers *wire.MsgHeaders
	peer    *peerpkg.Peer
}

// donePeerMsg signifies a newly disconnected peer to the block handler.
type donePeerMsg struct {
	peer *peerpkg.Peer
}

// txMsg packages a soter tx message and the peer it came from together
// so the block handler has access to that information.
type txMsg struct {
	tx    *soterutil.Tx
	peer  *peerpkg.Peer
	reply chan struct{}
}

// getSyncPeerMsg is a message type to be sent across the message channel for
// retrieving the current sync peer.
type getSyncPeerMsg struct {
	reply chan int32
}

// processBlockResponse is a response sent to the reply channel of a
// processBlockMsg.
type processBlockResponse struct {
	isOrphan bool
	err      error
}

// processBlockMsg is a message type to be sent across the message channel
// for requested a block is processed.  Note this call differs from blockMsg
// above in that blockMsg is intended for blocks that came from peers and have
// extra handling whereas this message essentially is just a concurrent safe
// way to call ProcessBlock on the internal block chain instance.
type processBlockMsg struct {
	block *soterutil.Block
	flags blockdag.BehaviorFlags
	reply chan processBlockResponse
}

// isCurrentMsg is a message type to be sent across the message channel for
// requesting whether or not the sync manager believes it is synced with the
// currently connected peers.
type isCurrentMsg struct {
	reply chan bool
}

// pauseMsg is a message type to be sent across the message channel for
// pausing the sync manager.  This effectively provides the caller with
// exclusive access over the manager until a receive is performed on the
// unpause channel.
type pauseMsg struct {
	unpause <-chan struct{}
}

// headerNode is used as a node in a list of headers that are linked together
// between checkpoints.
type headerNode struct {
	height int32
	hash   *chainhash.Hash
}

// peerSyncState stores additional information that the SyncManager tracks
// about a peer.
type peerSyncState struct {
	syncCandidate   bool
	requestQueue    []*wire.InvVect
	requestedTxns   map[chainhash.Hash]*requestExpiry
	requestedBlocks map[chainhash.Hash]*requestExpiry
}

// requestExpiry holds information about whether or not a request has expired.
// An expired request is eligible for retry
type requestExpiry struct {
	reqTime time.Time
	attempts int
}

// SyncManager is used to communicate block related messages with peers. The
// SyncManager is started as by executing Start() in a goroutine. Once started,
// it selects peers to sync from and starts the initial block download. Once the
// chain is in sync, the SyncManager handles incoming block and header
// notifications and relays announcements of new blocks to peers.
type SyncManager struct {
	peerNotifier   PeerNotifier
	started        int32
	shutdown       int32
	chain          *blockdag.BlockDAG
	txMemPool      *mempool.TxPool
	chainParams    *chaincfg.Params
	progressLogger *blockProgressLogger
	msgChan        chan interface{}
	wg             sync.WaitGroup
	quit           chan struct{}

	// These fields should only be accessed from the blockHandler thread
	rejectedTxns    map[chainhash.Hash]struct{}
	requestedTxns   map[chainhash.Hash]*requestExpiry
	requestedBlocks map[chainhash.Hash]*requestExpiry
	syncPeer        *peerpkg.Peer
	peerStates      map[*peerpkg.Peer]*peerSyncState
	lastProgressTime time.Time

	// The following fields are used for headers-first mode.
	headersFirstMode bool
	headerList       *list.List
	startHeader      *list.Element
	// NOTE(cedric): Commented out to disable checkpoint-related code (JIRA DAG-3)
	// 
	//
	// nextCheckpoint   *chaincfg.Checkpoint

	// An optional fee estimator.
	feeEstimator *mempool.FeeEstimator
}

// expired returns whether or not the request has expired
func (e *requestExpiry) expired() bool {
	return time.Now().After(e.reqTime.Add(stallSampleInterval))
}

// reset re-sets the reqTime fields
func (e *requestExpiry) reset() {
	e.reqTime = time.Now()
	e.attempts = 0
}

// NewRequestExpiry returns a new requestExpiry type
func NewRequestExpiry() *requestExpiry {
	return &requestExpiry{
		reqTime: time.Now(),
	}
}

// canReqBlock returns true if it's ok to request the block, from the perspective of
// sync manager's active request tracking.
func (sm *SyncManager) canReqBlock(state *peerSyncState, hash chainhash.Hash) bool {
	peerReqExp, peerExists := state.requestedBlocks[hash]

	if peerExists && !peerReqExp.expired() {
		return false
	}

	// Another peer may have recently asked for this block
	reqExp, exists := sm.requestedBlocks[hash]

	if exists && !reqExp.expired() {
		return false
	}

	return true
}

// canReqTxn returns true if it's ok to request the transaction, from the perspective of
// sync manager's active request tracking.
func (sm *SyncManager) canReqTxn(state *peerSyncState, hash chainhash.Hash) bool {
	peerReqExp, peerExists := state.requestedTxns[hash]

	if peerExists && !peerReqExp.expired() {
		return false
	}

	// Another peer may have recently asked for this transaction
	reqExp, exists := sm.requestedTxns[hash]

	if exists && !reqExp.expired() {
		return false
	}

	return true
}

// hasPendingBlocks returns true if there are non-expired requests for blocks with the peer
func (sm *SyncManager) hasPendingBlocks(peer *peerpkg.Peer) bool {
	state, exists := sm.peerStates[peer]
	if !exists {
		return false
	}

	for _, req := range state.requestedBlocks {
		if !req.expired() {
			return true
		}
	}

	return false
}

// resetHeaderState sets the headers-first mode state to values appropriate for
// syncing from a new peer.
func (sm *SyncManager) resetHeaderState(newestHash *chainhash.Hash, newestHeight int32) {
	sm.headersFirstMode = false
	sm.headerList.Init()
	sm.startHeader = nil

	// NOTE(cedric): Commented out to disable checkpoint-related code (JIRA DAG-3)
	// 
	//
	// When there is a next checkpoint, add an entry for the latest known
	// block into the header pool.  This allows the next downloaded header
	// to prove it links to the chain properly.
	// if sm.nextCheckpoint != nil {
	// 	node := headerNode{height: newestHeight, hash: newestHash}
	// 	sm.headerList.PushBack(&node)
	// }
}

// NOTE(cedric): Commented out to disable checkpoint-related code (JIRA DAG-3)
// 
//
// findNextHeaderCheckpoint returns the next checkpoint after the passed height.
// It returns nil when there is not one either because the height is already
// later than the final checkpoint or some other reason such as disabled
// checkpoints.
// func (sm *SyncManager) findNextHeaderCheckpoint(height int32) *chaincfg.Checkpoint {
// 	checkpoints := sm.chain.Checkpoints()
// 	if len(checkpoints) == 0 {
// 		return nil
// 	}

// 	// There is no next checkpoint if the height is already after the final
// 	// checkpoint.
// 	finalCheckpoint := &checkpoints[len(checkpoints)-1]
// 	if height >= finalCheckpoint.Height {
// 		return nil
// 	}

// 	// Find the next checkpoint.
// 	nextCheckpoint := finalCheckpoint
// 	for i := len(checkpoints) - 2; i >= 0; i-- {
// 		if height >= checkpoints[i].Height {
// 			break
// 		}
// 		nextCheckpoint = &checkpoints[i]
// 	}
// 	return nextCheckpoint
// }

// startSync will choose the best peer among the available candidate peers to
// download/sync the blockchain from.  When syncing is already running, it
// simply returns.  It also examines the candidates for any which are no longer
// candidates and removes them as needed.
func (sm *SyncManager) startSync() {
	// Return now if we're already syncing.
	if sm.syncPeer != nil {
		return
	}

	// Once the segwit soft-fork package has activated, we only
	// want to sync from peers which are witness enabled to ensure
	// that we fully validate all blockchain data.
	segwitActive, err := sm.chain.IsDeploymentActive(chaincfg.DeploymentSegwit)
	if err != nil {
		log.Errorf("Unable to query for segwit soft-fork state: %v", err)
		return
	}

	dagState := sm.chain.DAGSnapshot()
	var equalPeers, higherPeers []*peerpkg.Peer
	for peer, state := range sm.peerStates {
		if !state.syncCandidate {
			continue
		}

		if segwitActive && !peer.IsWitnessEnabled() {
			log.Debugf("peer %v not witness enabled, skipping", peer)
			continue
		}

		// Remove sync candidate peers that are no longer candidates due
		// to passing their latest known block.  NOTE: The < is
		// intentional as opposed to <=.  While technically the peer
		// doesn't have a later block when it's equal, it will likely
		// have one soon so it is a reasonable choice.  It also allows
		// the case where both are at 0 such as during regression test.
		if peer.MaxBlockHeight() < dagState.MaxHeight {
			state.syncCandidate = false
			continue
		}

		// Pick the peer with the highest block height
		if peer.MaxBlockHeight() > dagState.MaxHeight {
			higherPeers = append(higherPeers, peer)
		} else if peer.MaxBlockHeight() == dagState.MaxHeight {
			equalPeers = append(equalPeers, peer)
		}
	}

	// Sync with a randomly-chosen peer with a dag height higher than ours, falling back to a random peer at our
	// same height if there weren't any greater.
	var bestPeer *peerpkg.Peer
	switch {
	case len(higherPeers) > 0:
		bestPeer = higherPeers[rand.Intn(len(higherPeers))]
	case len(equalPeers) > 0:
		bestPeer = equalPeers[rand.Intn(len(equalPeers))]
	}

	// Start syncing from the best peer if one was selected.
	if bestPeer != nil {
		// Clear the requestedBlocks if the sync peer changes, otherwise
		// we may ignore blocks we need that the last sync peer failed
		// to send.
		sm.requestedBlocks = make(map[chainhash.Hash]*requestExpiry)

		locator, err := sm.chain.LatestBlockLocator()
		if err != nil {
			log.Errorf("Failed to get block locator for the "+
				"latest block: %v", err)
			return
		}

		log.Infof("Syncing to block height %d from peer %v",
			bestPeer.MaxBlockHeight(), bestPeer.Addr())

		// NOTE(cedric): Commented out to disable checkpoint-related code (JIRA DAG-3)
		// 
		//
		// When the current height is less than a known checkpoint we
		// can use block headers to learn about which blocks comprise
		// the chain up to the checkpoint and perform less validation
		// for them.  This is possible since each header contains the
		// hash of the previous header and a merkle root.  Therefore if
		// we validate all of the received headers link together
		// properly and the checkpoint hashes match, we can be sure the
		// hashes for the blocks in between are accurate.  Further, once
		// the full blocks are downloaded, the merkle root is computed
		// and compared against the value in the header which proves the
		// full block hasn't been tampered with.
		//
		// Once we have passed the final checkpoint, or checkpoints are
		// disabled, use standard inv messages learn about the blocks
		// and fully validate them.  Finally, regression test mode does
		// not support the headers-first approach so do normal block
		// downloads when in regression test mode.
		// if sm.nextCheckpoint != nil &&
		// 	best.Height < sm.nextCheckpoint.Height &&
		// 	sm.chainParams != &chaincfg.RegressionNetParams {

		// 	bestPeer.PushGetHeadersMsg(locator, sm.nextCheckpoint.Hash)
		// 	sm.headersFirstMode = true
		// 	log.Infof("Downloading headers for blocks %d to "+
		// 		"%d from peer %s", best.Height+1,
		// 		sm.nextCheckpoint.Height, bestPeer.Addr())
		// } else {
		// 	bestPeer.PushGetBlocksMsg(locator, &zeroHash)
		// }
		_ = bestPeer.PushGetBlocksMsg(locator, &zeroHash)
		sm.syncPeer = bestPeer

		// Reset the last progress time now that we have a non-nil
		// syncPeer to avoid instantly detecting it as stalled in the
		// event the progress time hasn't been updated recently.
		sm.lastProgressTime = time.Now()
	} else {
		log.Warnf("No sync peer candidates available")
	}
}

// stopSync unsets the 'sync' state of the sync manager
func (sm *SyncManager) stopSync() {
	log.Debugf("Stopping sync with peer %v", sm.syncPeer.Addr())
	sm.syncPeer = nil
	if sm.headersFirstMode {
		best := sm.chain.BestSnapshot()
		sm.resetHeaderState(&best.Hash, best.Height)
	}
}

// isSyncCandidate returns whether or not the peer is a candidate to consider
// syncing from.
func (sm *SyncManager) isSyncCandidate(peer *peerpkg.Peer) bool {
	// Typically a peer is not a candidate for sync if it's not a full node,
	// however regression test is special in that the regression tool is
	// not a full node and still needs to be considered a sync candidate.
	if sm.chainParams == &chaincfg.RegressionNetParams {
		// The peer is not a candidate if it's not coming from localhost
		// or the hostname can't be determined for some reason.
		host, _, err := net.SplitHostPort(peer.Addr())
		if err != nil {
			return false
		}

		if host != "127.0.0.1" && host != "localhost" {
			return false
		}
	} else {
		// The peer is not a candidate for sync if it's not a full
		// node. Additionally, if the segwit soft-fork package has
		// activated, then the peer must also be upgraded.
		segwitActive, err := sm.chain.IsDeploymentActive(chaincfg.DeploymentSegwit)
		if err != nil {
			log.Errorf("Unable to query for segwit "+
				"soft-fork state: %v", err)
		}
		nodeServices := peer.Services()
		if nodeServices&wire.SFNodeNetwork != wire.SFNodeNetwork ||
			(segwitActive && !peer.IsWitnessEnabled()) {
			return false
		}
	}

	// Candidate if all checks passed.
	return true
}

// handleNewPeerMsg deals with new peers that have signalled they may
// be considered as a sync peer (they have already successfully negotiated).  It
// also starts syncing if needed.  It is invoked from the syncHandler goroutine.
func (sm *SyncManager) handleNewPeerMsg(peer *peerpkg.Peer) {
	// Ignore if in the process of shutting down.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		return
	}

	log.Infof("New valid peer %s (%s)", peer, peer.UserAgent())

	// Initialize the peer state
	isSyncCandidate := sm.isSyncCandidate(peer)
	sm.peerStates[peer] = &peerSyncState{
		syncCandidate:   isSyncCandidate,
		requestedTxns:   make(map[chainhash.Hash]*requestExpiry),
		requestedBlocks: make(map[chainhash.Hash]*requestExpiry),
	}

	// Start syncing by choosing the best candidate if needed.
	if isSyncCandidate && sm.syncPeer == nil {
		sm.startSync()
	}
}

// handleStallSample will switch to a new sync peer if the current one has
// stalled. This is detected when by comparing the last progress timestamp with
// the current time, and disconnecting the peer if we stalled before reaching
// their highest advertised block.
func (sm *SyncManager) handleStallSample() {
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		return
	}

	// If we don't have an active sync peer, exit early.
	if sm.syncPeer == nil {
		return
	}

	// If the stall timeout has not elapsed, exit early.
	if time.Since(sm.lastProgressTime) <= maxStallDuration {
		return
	}

	// Check to see that the peer's sync state exists.
	state, exists := sm.peerStates[sm.syncPeer]
	if !exists {
		return
	}

	sm.clearRequestedState(state)

	disconnectSyncPeer := sm.shouldDCStalledSyncPeer()
	sm.updateSyncPeer(disconnectSyncPeer)
}

// shouldDCStalledSyncPeer determines whether or not we should disconnect a
// stalled sync peer. If the peer has stalled and its reported height is greater
// than our own best height, we will disconnect it. Otherwise, we will keep the
// peer connected in case we are already at tip.
func (sm *SyncManager) shouldDCStalledSyncPeer() bool {
	maxHeight := sm.syncPeer.MaxBlockHeight()
	startHeight := sm.syncPeer.StartingHeight()

	var peerHeight int32
	if maxHeight > startHeight {
		peerHeight = maxHeight
	} else {
		peerHeight = startHeight
	}

	// If we've stalled out yet the sync peer reports having more blocks for
	// us we will disconnect them. This allows us at tip to not disconnect
	// peers when we are equal or they temporarily lag behind us.
	best := sm.chain.BestSnapshot()
	return peerHeight > best.Height
}

// handleDonePeerMsg deals with peers that have signalled they are done.  It
// removes the peer as a candidate for syncing and in the case where it was
// the current sync peer, attempts to select a new best peer to sync from.  It
// is invoked from the syncHandler goroutine.
func (sm *SyncManager) handleDonePeerMsg(peer *peerpkg.Peer) {
	state, exists := sm.peerStates[peer]
	if !exists {
		log.Warnf("Received done peer message for unknown peer %s", peer)
		return
	}

	// Remove the peer from the list of candidate peers.
	delete(sm.peerStates, peer)

	log.Infof("Lost peer %s", peer)

	sm.clearRequestedState(state)

	if peer == sm.syncPeer {
		// Update the sync peer. The server has already disconnected the
		// peer before signaling to the sync manager.
		sm.updateSyncPeer(false)
	}
}

// clearRequestedState wipes all expected transactions and blocks from the sync
// manager's requested maps that were requested under a peer's sync state, This
// allows them to be re-requested by a subsequent sync peer.
func (sm *SyncManager) clearRequestedState(state *peerSyncState) {
	// Remove requested transactions from the global map so that they will
	// be fetched from elsewhere next time we get an inv.
	for txHash := range state.requestedTxns {
		delete(sm.requestedTxns, txHash)
	}

	// Remove requested blocks from the global map so that they will be
	// fetched from elsewhere next time we get an inv.
	// TODO: we could possibly here check which peers have these blocks
	// and request them now to speed things up a little.
	for blockHash := range state.requestedBlocks {
		delete(sm.requestedBlocks, blockHash)
	}
}

// updateSyncPeer choose a new sync peer to replace the current one. If
// dcSyncPeer is true, this method will also disconnect the current sync peer.
// If we are in header first mode, any header state related to prefetching is
// also reset in preparation for the next sync peer.
func (sm *SyncManager) updateSyncPeer(dcSyncPeer bool) {
	log.Infof("Updating sync peer, no progress for: %v",
		time.Since(sm.lastProgressTime))

	// First, disconnect the current sync peer if requested.
	if dcSyncPeer {
		sm.syncPeer.Disconnect()
	}

	// Reset any header state before we choose our next active sync peer.
	if sm.headersFirstMode {
		best := sm.chain.BestSnapshot()
		sm.resetHeaderState(&best.Hash, best.Height)
	}

	sm.syncPeer = nil
	sm.startSync()
}

// handleTxMsg handles transaction messages from all peers.
func (sm *SyncManager) handleTxMsg(tmsg *txMsg) {
	peer := tmsg.peer
	state, exists := sm.peerStates[peer]
	if !exists {
		log.Warnf("Received tx message from unknown peer %s", peer)
		return
	}

	// NOTE:  BitcoinJ, and possibly other wallets, don't follow the spec of
	// sending an inventory message and allowing the remote peer to decide
	// whether or not they want to request the transaction via a getdata
	// message.  Unfortunately, the reference implementation permits
	// unrequested data, so it has allowed wallets that don't follow the
	// spec to proliferate.  While this is not ideal, there is no check here
	// to disconnect peers for sending unsolicited transactions to provide
	// interoperability.
	txHash := tmsg.tx.Hash()

	// Ignore transactions that we have already rejected.  Do not
	// send a reject message here because if the transaction was already
	// rejected, the transaction was unsolicited.
	if _, exists = sm.rejectedTxns[*txHash]; exists {
		log.Debugf("Ignoring unsolicited previously rejected "+
			"transaction %v from %s", txHash, peer)
		return
	}

	// Process the transaction to include validation, insertion in the
	// memory pool, orphan handling, etc.
	acceptedTxs, err := sm.txMemPool.ProcessTransaction(tmsg.tx,
		true, true, mempool.Tag(peer.ID()))

	// Remove transaction from request maps. Either the mempool/chain
	// already knows about it and as such we shouldn't have any more
	// instances of trying to fetch it, or we failed to insert and thus
	// we'll retry next time we get an inv.
	delete(state.requestedTxns, *txHash)
	delete(sm.requestedTxns, *txHash)

	if err != nil {
		// Do not request this transaction again until a new block
		// has been processed.
		sm.rejectedTxns[*txHash] = struct{}{}
		sm.limitMap(sm.rejectedTxns, maxRejectedTxns)

		// When the error is a rule error, it means the transaction was
		// simply rejected as opposed to something actually going wrong,
		// so log it as such.  Otherwise, something really did go wrong,
		// so log it as an actual error.
		if _, ok := err.(mempool.RuleError); ok {
			log.Debugf("Rejected transaction %v from %s: %v",
				txHash, peer, err)
		} else {
			log.Errorf("Failed to process transaction %v: %v",
				txHash, err)
		}

		// Convert the error into an appropriate reject message and
		// send it.
		code, reason := mempool.ErrToRejectErr(err)
		peer.PushRejectMsg(wire.CmdTx, code, reason, txHash, false)
		return
	}

	sm.peerNotifier.AnnounceNewTransactions(acceptedTxs)
}

// current returns true if we believe we are synced with our peers, false if we
// still have blocks to check
func (sm *SyncManager) current() bool {
	if !sm.chain.IsCurrent() {
		return false
	}

	dagState := sm.chain.DAGSnapshot()
	// If we are below the height of our peers, we are not current
	for peer, _ := range sm.peerStates {
		if dagState.MaxHeight < peer.MaxBlockHeight() {
			return false
		}
	}

	// if blockChain thinks we are current and we have no syncPeer it
	// is probably right.
	if sm.syncPeer == nil {
		return true
	}

	// No matter what chain thinks, if we are below the block we are syncing
	// to we are not current.
	if dagState.MaxHeight < sm.syncPeer.MaxBlockHeight() {
		return false
	}
	return true
}

// handleBlockMsg handles block messages from all peers.
func (sm *SyncManager) handleBlockMsg(bmsg *blockMsg) {
	peer := bmsg.peer
	state, exists := sm.peerStates[peer]
	if !exists {
		log.Warnf("Received block message from unknown peer %s", peer)
		return
	}

	// If we didn't ask for this block then the peer is misbehaving.
	blockHash := bmsg.block.Hash()
	if _, exists = state.requestedBlocks[*blockHash]; !exists {
		// The regression test intentionally sends some blocks twice
		// to test duplicate block insertion fails.  Don't disconnect
		// the peer or ignore the block when we're in regression test
		// mode in this case so the chain code is actually fed the
		// duplicate blocks.
		if sm.chainParams != &chaincfg.RegressionNetParams {
			log.Warnf("Got unrequested block %v from %s -- "+
				"disconnecting", blockHash, peer.Addr())
			peer.Disconnect()
			return
		}
	}

	// When in headers-first mode, if the block matches the hash of the
	// first header in the list of headers that are being fetched, it's
	// eligible for less validation since the headers have already been
	// verified to link together and are valid up to the next checkpoint.
	// Also, remove the list entry for all blocks except the checkpoint
	// since it is needed to verify the next round of headers links
	// properly.
	isCheckpointBlock := false
	behaviorFlags := blockdag.BFNone
	if sm.headersFirstMode {
		firstNodeEl := sm.headerList.Front()
		if firstNodeEl != nil {
			firstNode := firstNodeEl.Value.(*headerNode)
			if blockHash.IsEqual(firstNode.hash) {
				behaviorFlags |= blockdag.BFFastAdd
				// NOTE(cedric): Commented out to disable checkpoint-related code (JIRA DAG-3)
				// 
				//
				// if firstNode.hash.IsEqual(sm.nextCheckpoint.Hash) {
				// 	isCheckpointBlock = true
				// } else {
				// 	sm.headerList.Remove(firstNodeEl)
				// }
				sm.headerList.Remove(firstNodeEl)
			}
		}
	}

	// Remove block from request maps. Either chain will know about it and
	// so we shouldn't have any more instances of trying to fetch it, or we
	// will fail the insert and thus we'll retry next time we get an inv.
	delete(state.requestedBlocks, *blockHash)
	delete(sm.requestedBlocks, *blockHash)

	// Process the block to include validation, best chain selection, orphan
	// handling, etc.
	_, isOrphan, err := sm.chain.ProcessBlock(bmsg.block, behaviorFlags)
	if err != nil {
		// When the error is a rule error, it means the block was simply
		// rejected as opposed to something actually going wrong, so log
		// it as such.  Otherwise, something really did go wrong, so log
		// it as an actual error.
		if _, ok := err.(blockdag.RuleError); ok {
			log.Infof("Rejected block %v from %s: %v", blockHash,
				peer, err)
		} else {
			log.Errorf("Failed to process block %v: %v",
				blockHash, err)
		}
		if dbErr, ok := err.(database.Error); ok && dbErr.ErrorCode ==
			database.ErrCorruption {
			panic(dbErr)
		}

		// Convert the error into an appropriate reject message and
		// send it.
		code, reason := mempool.ErrToRejectErr(err)
		peer.PushRejectMsg(wire.CmdBlock, code, reason, blockHash, false)
		return
	}

	// Meta-data about the new block this peer is reporting. We use this
	// below to update this peer's latest block height and the heights of
	// other peers based on their last announced block hash. This allows us
	// to dynamically update the block heights of peers, avoiding stale
	// heights when looking for a new sync peer. Upon acceptance of a block
	// or recognition of an orphan, we also use this information to update
	// the block heights over other peers who's invs may have been ignored
	// if we are actively syncing while the chain is not yet current or
	// who may have lost the lock announcement race.
	var heightUpdate int32
	var blkHashUpdate *chainhash.Hash

	// Request the parents for the orphan block from the peer that sent it.
	if isOrphan {
		var cbHeight int32
		// We've just received an orphan block from a peer. In order
		// to update the height of the peer, we try to extract the
		// block height from the scriptSig of the coinbase transaction.
		// Extraction is only attempted if the block's version is
		// high enough (ver 2+).
		header := &bmsg.block.MsgBlock().Header
		if blockdag.ShouldHaveSerializedBlockHeight(header) {
			coinbaseTx := bmsg.block.Transactions()[0]
			cbHeight, err = blockdag.ExtractCoinbaseHeight(coinbaseTx)
			if err != nil {
				log.Warnf("Unable to extract height from "+
					"coinbase tx: %v", err)
			} else {
				log.Debugf("Extracted height of %v from "+
					"orphan block", cbHeight)
				heightUpdate = cbHeight
				blkHashUpdate = blockHash
			}
		}

		sm.reqOrphanParents(peer)
	} else {
		if peer == sm.syncPeer {
			sm.lastProgressTime = time.Now()
		}

		// When the block is not an orphan, log information about it and
		// update the chain state.
		sm.progressLogger.LogBlockHeight(bmsg.block)

		// Update this peer's latest block height, for future
		// potential sync node candidacy.
		best := sm.chain.BestSnapshot()
		dagState := sm.chain.DAGSnapshot()
		heightUpdate = dagState.MaxHeight
		blkHashUpdate = &best.Hash

		// Clear the rejected transactions.
		sm.rejectedTxns = make(map[chainhash.Hash]struct{})
	}

	// Update the block height for this peer, if higher than current.
	// But only send a message to the server for updating peer heights
	// if this is an orphan or our chain is "current". This avoids sending
	// a spammy amount of messages if we're syncing the chain from scratch.
	if blkHashUpdate != nil && heightUpdate > 0 {
		if heightUpdate > peer.MaxBlockHeight() {
			peer.UpdateMaxBlockHeight(heightUpdate)
		}
		if isOrphan || sm.current() {
			go sm.peerNotifier.UpdatePeerHeights(blkHashUpdate, heightUpdate,
				peer)
		}
	}

	// Check if we should stop syncing, now that peer heights have been updated
	if !sm.headersFirstMode {
		if sm.syncPeer != nil && sm.current() {
			sm.stopSync()
		}
	}

	var blockHeight = bmsg.block.Height()
	// If this block is the parent of an orphan, request block inventory between this parent's height and the highest
	// orphan block.
	// jenlouie: Only check if the parent is not an orphan, otherwise, the block's height is not assigned yet and
	// the locator will ask for blocks starting from genesis.
	if !isOrphan {
		sm.reqOrphanChildren(peer, bmsg.block)

		if blockHeight == peer.MaxBlockHeight() {
			// Once we've reached peer's last-known max height, ask the peer if there's any more blocks it has for us.
			locator := blockdag.BlockLocator([]*int32{&blockHeight})
			err = peer.PushGetBlocksMsg(locator, &zeroHash)
			if err != nil {
				log.Warnf("Failed to send getblocks message to peer %s: %s", peer, err)
				return
			}
		}
	}

	// Nothing more to do if we aren't in headers-first mode.
	if !sm.headersFirstMode {
		return
	}

	// This is headers-first mode, so if the block is not a checkpoint
	// request more blocks using the header list when the request queue is
	// getting short.
	if !isCheckpointBlock {
		if sm.startHeader != nil &&
			len(state.requestedBlocks) < minInFlightBlocks {
			sm.fetchHeaderBlocks()
		}
		return
	}

	// NOTE(cedric): Commented out to disable checkpoint-related code (JIRA DAG-3)
	// 
	//
	// This is headers-first mode and the block is a checkpoint.  When
	// there is a next checkpoint, get the next round of headers by asking
	// for headers starting from the block after this one up to the next
	// checkpoint.
	// prevHeight := sm.nextCheckpoint.Height
	// prevHash := sm.nextCheckpoint.Hash
	// sm.nextCheckpoint = sm.findNextHeaderCheckpoint(prevHeight)
	// if sm.nextCheckpoint != nil {
	// 	locator := blockchain.BlockLocator([]*chainhash.Hash{prevHash})
	// 	err := peer.PushGetHeadersMsg(locator, sm.nextCheckpoint.Hash)
	// 	if err != nil {
	// 		log.Warnf("Failed to send getheaders message to "+
	// 			"peer %s: %v", peer.Addr(), err)
	// 		return
	// 	}
	// 	log.Infof("Downloading headers for blocks %d to %d from "+
	// 		"peer %s", prevHeight+1, sm.nextCheckpoint.Height,
	// 		sm.syncPeer.Addr())
	// 	return
	// }

	// This is headers-first mode, the block is a checkpoint, and there are
	// no more checkpoints, so switch to normal mode by requesting blocks
	// from the block after this one up to the end of the chain (zero hash).
	sm.headersFirstMode = false
	sm.headerList.Init()
	log.Infof("Reached the final checkpoint -- switching to normal mode")
	locator := blockdag.BlockLocator([]*int32{&blockHeight})
	err = peer.PushGetBlocksMsg(locator, &zeroHash)
	if err != nil {
		log.Warnf("Failed to send getblocks message to peer %s: %v",
			peer.Addr(), err)
		return
	}
}

// fetchHeaderBlocks creates and sends a request to the syncPeer for the next
// list of blocks to be downloaded based on the current list of headers.
func (sm *SyncManager) fetchHeaderBlocks() {
	// Nothing to do if there is no start header.
	if sm.startHeader == nil {
		log.Warnf("fetchHeaderBlocks called with no start header")
		return
	}

	// Build up a getdata request for the list of blocks the headers
	// describe.  The size hint will be limited to wire.MaxInvPerMsg by
	// the function, so no need to double check it here.
	gdmsg := wire.NewMsgGetDataSizeHint(uint(sm.headerList.Len()))
	numRequested := 0
	for e := sm.startHeader; e != nil; e = e.Next() {
		node, ok := e.Value.(*headerNode)
		if !ok {
			log.Warn("Header list node type is not a headerNode")
			continue
		}

		iv := wire.NewInvVect(wire.InvTypeBlock, node.hash, node.height)
		haveInv, err := sm.haveInventory(iv)
		if err != nil {
			log.Warnf("Unexpected failure when checking for "+
				"existing inventory during header block "+
				"fetch: %v", err)
		}
		if !haveInv {
			syncPeerState := sm.peerStates[sm.syncPeer]

			reqExp := NewRequestExpiry()
			sm.requestedBlocks[*node.hash] = reqExp
			syncPeerState.requestedBlocks[*node.hash] = reqExp

			// If we're fetching from a witness enabled peer
			// post-fork, then ensure that we receive all the
			// witness data in the blocks.
			if sm.syncPeer.IsWitnessEnabled() {
				iv.Type = wire.InvTypeWitnessBlock
			}

			gdmsg.AddInvVect(iv)
			numRequested++
		}
		sm.startHeader = e.Next()
		if numRequested >= wire.MaxInvPerMsg {
			break
		}
	}
	if len(gdmsg.InvList) > 0 {
		sm.syncPeer.QueueMessage(gdmsg, nil)
	}
}

// filterSupportedInv returns a list of inventory we support processing
func filterSupportedInv(inventory []*wire.InvVect) []*wire.InvVect {
	var supported []*wire.InvVect
	for _, iv := range inventory {
		switch iv.Type {
		case wire.InvTypeBlock:
		case wire.InvTypeTx:
		case wire.InvTypeWitnessBlock:
		case wire.InvTypeWitnessTx:
		default:
			continue
		}

		supported = append(supported, iv)
	}

	return supported
}

// filterNeededInv returns a list of inventory we need to request from a peer, in outbound getdata requests
func (sm *SyncManager) filterNeededInv(peer *peerpkg.Peer, inventory []*wire.InvVect) []*wire.InvVect {
	var needed []*wire.InvVect

	for _, iv := range inventory {
		haveInv, err := sm.haveInventory(iv)
		if err != nil {
			log.Warnf("Unexpected failure when checking for "+
				"existing inventory during inv message "+
				"processing: %v", err)
			continue
		}

		if haveInv {
			continue
		}

		if iv.Type == wire.InvTypeTx {
			// Skip the transaction if it has already been rejected.
			if _, exists := sm.rejectedTxns[iv.Hash]; exists {
				continue
			}
		}

		// Ignore block inventory from non-witness enabled peers, as after segwit activation we only want to download
		// from peers that can provide us full witness data for blocks.
		if iv.Type == wire.InvTypeBlock && !peer.IsWitnessEnabled() {
			continue
		}

		needed = append(needed, iv)
	}

	return needed
}

// handleHeadersMsg handles block header messages from all peers.  Headers are
// requested when performing a headers-first sync.
func (sm *SyncManager) handleHeadersMsg(hmsg *headersMsg) {
	peer := hmsg.peer
	_, exists := sm.peerStates[peer]
	if !exists {
		log.Warnf("Received headers message from unknown peer %s", peer)
		return
	}

	// The remote peer is misbehaving if we didn't request headers.
	msg := hmsg.headers
	numHeaders := len(msg.Headers)
	if !sm.headersFirstMode {
		log.Warnf("Got %d unrequested headers from %s -- "+
			"disconnecting", numHeaders, peer.Addr())
		peer.Disconnect()
		return
	}

	// Nothing to do for an empty headers message.
	if numHeaders == 0 {
		return
	}

	// Process all of the received headers ensuring each one connects to the
	// previous and that checkpoints match.
	//
	// NOTE(cedric): Commented out to disable checkpoint-related code (JIRA DAG-3)
	// 
	//
	// receivedCheckpoint := false
	var finalHash *chainhash.Hash
	for _, blockHeader := range msg.Headers {
		blockHash := blockHeader.BlockHash()
		finalHash = &blockHash

		// Ensure there is a previous header to compare against.
		prevNodeEl := sm.headerList.Back()
		if prevNodeEl == nil {
			log.Warnf("Header list does not contain a previous" +
				"element as expected -- disconnecting peer")
			peer.Disconnect()
			return
		}

		// NOTE(cedric in DAG-32): jenlouie noted that checking header connectivity doesn't
		// make sense in DAG; A block can connect to any block, not just the previous one.
		// For now I'll comment-out the check.
		//
		// Ensure the header properly connects to the previous one and
		// add it to the list of headers.
		// TODO(jenlouie): revisit in DAG-17 bootstrapping DAG
		//node := headerNode{hash: &blockHash}
		//prevNode := prevNodeEl.Value.(*headerNode)
		//if prevNode.hash.IsEqual(&blockHeader.PrevBlock) {
		//	node.height = prevNode.height + 1
		//	e := sm.headerList.PushBack(&node)
		//	if sm.startHeader == nil {
		//		sm.startHeader = e
		//	}
		//} else {
		//	log.Warnf("Received block header that does not "+
		//		"properly connect to the chain from peer %s "+
		//		"-- disconnecting", peer.Addr())
		//	peer.Disconnect()
		//	return
		//}

		// NOTE(cedric): Commented out to disable checkpoint-related code (JIRA DAG-3)
		// 
		//
		// // Verify the header at the next checkpoint height matches.
		// if node.height == sm.nextCheckpoint.Height {
		// 	if node.hash.IsEqual(sm.nextCheckpoint.Hash) {
		// 		receivedCheckpoint = true
		// 		log.Infof("Verified downloaded block "+
		// 			"header against checkpoint at height "+
		// 			"%d/hash %s", node.height, node.hash)
		// 	} else {
		// 		log.Warnf("Block header at height %d/hash "+
		// 			"%s from peer %s does NOT match "+
		// 			"expected checkpoint hash of %s -- "+
		// 			"disconnecting", node.height,
		// 			node.hash, peer.Addr(),
		// 			sm.nextCheckpoint.Hash)
		// 		peer.Disconnect()
		// 		return
		// 	}
		// 	break
		// }
	}

	// NOTE(cedric): Commented out to disable checkpoint-related code (JIRA DAG-3)
	// 
	//
	// // When this header is a checkpoint, switch to fetching the blocks for
	// // all of the headers since the last checkpoint.
	// if receivedCheckpoint {
	// 	// Since the first entry of the list is always the final block
	// 	// that is already in the database and is only used to ensure
	// 	// the next header links properly, it must be removed before
	// 	// fetching the blocks.
	// 	sm.headerList.Remove(sm.headerList.Front())
	// 	log.Infof("Received %v block headers: Fetching blocks",
	// 		sm.headerList.Len())
	// 	sm.progressLogger.SetLastLogTime(time.Now())
	// 	sm.fetchHeaderBlocks()
	// 	return
	// }

	// This header is not a checkpoint, so request the next batch of
	// headers starting from the latest known header.
	locator := sm.chain.BlockLocatorFromHash(finalHash)
	// NOTE(cedric): Commented out to disable checkpoint-related code (JIRA DAG-3)
	// 
	//
	// When uncommented, we would request from the latest known header to
	// the next checkpoint.
	// err := peer.PushGetHeadersMsg(locator, sm.nextCheckpoint.Hash)
	err := peer.PushGetHeadersMsg(locator, nil)
	if err != nil {
		log.Warnf("Failed to send getheaders message to "+
			"peer %s: %v", peer.Addr(), err)
		return
	}
}

// haveInventory returns whether or not the inventory represented by the passed
// inventory vector is known.  This includes checking all of the various places
// inventory can be when it is in different states such as blocks that are part
// of the main chain, on a side chain, in the orphan pool, and transactions that
// are in the memory pool (either the main pool or orphan pool).
func (sm *SyncManager) haveInventory(invVect *wire.InvVect) (bool, error) {
	switch invVect.Type {
	case wire.InvTypeWitnessBlock:
		fallthrough
	case wire.InvTypeBlock:
		// Ask chain if the block is known to it in any form (main
		// chain, side chain, or orphan).
		return sm.chain.HaveBlock(&invVect.Hash)

	case wire.InvTypeWitnessTx:
		fallthrough
	case wire.InvTypeTx:
		// Ask the transaction memory pool if the transaction is known
		// to it in any form (main pool or orphan).
		if sm.txMemPool.HaveTransaction(&invVect.Hash) {
			return true, nil
		}

		// Check if the transaction exists from the point of view of the
		// end of the main chain.  Note that this is only a best effort
		// since it is expensive to check existence of every output and
		// the only purpose of this check is to avoid downloading
		// already known transactions.  Only the first two outputs are
		// checked because the vast majority of transactions consist of
		// two outputs where one is some form of "pay-to-somebody-else"
		// and the other is a change output.
		prevOut := wire.OutPoint{Hash: invVect.Hash}
		for i := uint32(0); i < 2; i++ {
			prevOut.Index = i
			entry, err := sm.chain.FetchUtxoEntry(prevOut)
			if err != nil {
				return false, err
			}
			if entry != nil && !entry.IsSpent() {
				return true, nil
			}
		}

		return false, nil
	}

	// The requested inventory is is an unsupported type, so just claim
	// it is known to avoid requesting it.
	return true, nil
}

// handleInvMsg handles inv messages from peers
// We examine inventory advertised and request info that we don't have
func (sm *SyncManager) handleInvMsg(imsg *invMsg) {
	// Ignore message if it's from a node we're not connected to
	peer := imsg.peer
	_, exists := sm.peerStates[peer]
	if !exists {
		log.Warnf("Received inv message from unknown peer %s", peer)
		return
	}

	// Only process supported inventory types
	inventory := filterSupportedInv(imsg.inv.InvList)

	if len(inventory) == 1 && inventory[0].Hash.IsEqual(&zeroHash) {
		// Peer is indicating that it has more blocks to send us, so ask for blocks
		// from our current tips to theirs.
		maxHeight := sm.chain.DAGSnapshot().MaxHeight
		locator := sm.chain.BlockLocatorFromHeight(maxHeight)
		err := peer.PushGetBlocksMsg(locator, &zeroHash)
		if err != nil {
			log.Warnf("Failed to send getblocks message to peer %s: %s", peer, err)
		}
		return
	}

	// Update the peer's last announced block
	for i := len(inventory) - 1; i >= 0; i-- {
		if inventory[i].Type == wire.InvTypeBlock {
			peer.UpdateLastAnnouncedBlock(&inventory[i].Hash)
			break
		}
	}

	// Update the peer's height
	maxHeight := invMaxBlockHeight(inventory)
	if maxHeight > peer.MaxBlockHeight() {
		peer.UpdateMaxBlockHeight(maxHeight)
	}

	// If we're in sync mode but have caught up to our sync-peer, exit sync mode
	if sm.syncPeer != nil && sm.chain.DAGSnapshot().MaxHeight >= sm.syncPeer.MaxBlockHeight() {
		sm.stopSync()
	}

	// If we're in sync mode, don't process inventory messages from non-sync peers
	if sm.syncPeer != nil && peer != sm.syncPeer {
		log.Warnf("Ignoring inv message from peer %v MaxBlockHeight %v (syncPeer %v, MaxBlockHeight %v), because we are in sync mode",
			peer, peer.MaxBlockHeight(), sm.syncPeer, sm.syncPeer.MaxBlockHeight())
		return
	}

	// Cache inventory that the peer advertised to us.
	// It looks like this is used to help determine what should be advertised back to this peer
	// (skipping blocks it already knows of)
	for _, iv := range inventory {
		peer.AddKnownInventory(iv)
	}

	if !sm.headersFirstMode {
		// Request blocks we don't have
		sm.reqBlocks(peer, inventory)
	}

	// Request blocks for our orphans' parents.
	// This will end up triggering inv messages containing blocks between the orphan's parent's height and the orphan.
	sm.reqOrphanParents(peer)
}

// reqOrphanChildren sends a getblocks message to the peer, for blocks between the height of the orphan parent to
// each child orphan block.
func (sm *SyncManager) reqOrphanChildren(peer *peerpkg.Peer, parent *soterutil.Block) {
	children := sm.chain.GetOrphanChildren(parent.Hash())
	for _, child := range children {
		locator := sm.chain.BlockLocatorFromHeight(parent.Height())
		peer.PushGetBlocksMsg(locator, child.Hash())
	}
}

// reqOrphanParents sends getdata messages to the peer for missing parent blocks.
// How this works:
// 1. We request the orphan's parent blocks directly
// 2. In handleBlock() where the response is received, we check if the block is a parent of any orphans
// 3. If we find orphan children, we call reqOrphanChildren to issue a getblocks message for blocks between the height
//    of the orphan parent to the hash of the highest orphan block.
//func (sm *SyncManager) reqOrphanParents(peer *peerpkg.Peer) {
func (sm *SyncManager) reqOrphanParents(peer *peerpkg.Peer) {
	state, exists := sm.peerStates[peer]
	if !exists {
		log.Errorf("Can't request orphan parent blocks from unknown peer %s", peer)
		return
	}

	// needed map contains what we'll request from peer
	needed := make(map[*wire.InvVect]int)

	// Look at all orphans (not just those in our current inventory)
	for _, orphan := range sm.chain.GetOrphanBlocks() {
		for _, parent := range orphan.MsgBlock().Parents.Parents {
			if !sm.canReqBlock(state, parent.Hash) {
				// A request for this block is already in progress
				continue
			}

			iv := wire.NewInvVect(wire.InvTypeBlock, &parent.Hash, -1)

			if peer.IsWitnessEnabled() {
				iv.Type = wire.InvTypeWitnessBlock
			}

			have, err := sm.haveInventory(iv)
			if err != nil {
				log.Warnf("Unexpected failure when checking inventory for orphan block %v parent %v: %v",
					orphan.Hash(), parent.Hash, err)
				continue
			}

			if have {
				// We don't need to request a block we already have
				continue
			}

			needed[iv] = 0
		}
	}

	// Create getdata messages to accommodate all of the needed blocks
	gdMsgs := make([]*wire.MsgGetData, 0)
	msg := wire.NewMsgGetData()
	for len(needed) > 0 {
		var iv *wire.InvVect
		for pIv := range needed {
			iv = pIv
			break
		}
		delete(needed, iv)

		if len(msg.InvList) >= wire.MaxInvPerMsg {
			gdMsgs = append(gdMsgs, msg)
			msg = wire.NewMsgGetData()
		}

		// Track that we're making a request for this block, so that the response isn't dropped when we receive it,
		// and so that we don't make duplicate requests.
		// (we drop unsolicited blocks)
		sm.trackReqBlock(state, iv.Hash)

		msg.AddInvVect(iv)
	}

	if len(msg.InvList) > 0 {
		gdMsgs = append(gdMsgs, msg)
	}

	// Send getdata message(s)
	for _, msg := range gdMsgs {
		peer.QueueMessage(msg, nil)
	}
}

// lastInvBlock returns the last block in the inventory, or nil if one wasn't found.
func lastInvBlock(inventory []*wire.InvVect) *wire.InvVect {
	var iv *wire.InvVect
	for i := len(inventory) - 1; i >= 0; i-- {
		if inventory[i].Type == wire.InvTypeBlock {
			iv = inventory[i]
			break
		}
	}
	return iv
}

// limitMap is a helper function for maps that require a maximum limit by
// evicting a random transaction if adding a new value would cause it to
// overflow the maximum allowed.
func (sm *SyncManager) limitMap(m map[chainhash.Hash]struct{}, limit int) {
	if len(m)+1 > limit {
		// Remove a random entry from the map.  For most compilers, Go's
		// range statement iterates starting at a random item although
		// that is not 100% guaranteed by the spec.  The iteration order
		// is not important here because an adversary would have to be
		// able to pull off preimage attacks on the hashing function in
		// order to target eviction of specific entries anyways.
		for txHash := range m {
			delete(m, txHash)
			return
		}
	}
}

// limitRequests deletes requests from the map of requestExpiry types, until
// the number remaining + 1 are below the limit.
// The +1 allows for one more request to be tracked without exceeding the limit.
func (sm *SyncManager) limitTrackedReqs(m map[chainhash.Hash]*requestExpiry, limit int) {
	for hash, reqExp := range m {
		if reqExp.expired() {
			delete(m, hash)
		}
	}

	if limit < 0 {
		return
	}

	for (len(m) + 1) > limit {
		for hash := range m {
			delete(m, hash)
		}
	}
}

// blockHandler is the main handler for the sync manager.  It must be run as a
// goroutine.  It processes block and inv messages in a separate goroutine
// from the peer handlers so the block (MsgBlock) messages are handled by a
// single thread without needing to lock memory data structures.  This is
// important because the sync manager controls which blocks are needed and how
// the fetching should proceed.
func (sm *SyncManager) blockHandler() {
	var stallTicker = time.NewTicker(stallSampleInterval)
	defer stallTicker.Stop()

	// Dispatch incoming messages to fast & slow channels, based on the message type.
	dispatch := func(fast, slow chan interface{}) {
		for m := range sm.msgChan {
			switch m.(type) {
			case *blockMsg:
				slow <- m
			case *processBlockMsg:
				slow <- m
			default:
				fast <- m
			}
		}
	}

	var fast = make(chan interface{})
	var slow = make(chan interface{})

	// This goroutine will sort messages from msgChan into different queues. It's meant to help the sync manager
	// remain responsive when receiving a large number of messages that take longer to process.
	go dispatch(fast, slow)

out:
	for {
		select {
		case m := <-fast:
			switch msg := m.(type) {
			case *newPeerMsg:
				sm.handleNewPeerMsg(msg.peer)

			case *txMsg:
				sm.handleTxMsg(msg)
				msg.reply <- struct{}{}

			case *blockMsg:
				sm.handleBlockMsg(msg)
				msg.reply <- struct{}{}

			case *invMsg:
				sm.handleInvMsg(msg)

			case *headersMsg:
				sm.handleHeadersMsg(msg)

			case *donePeerMsg:
				sm.handleDonePeerMsg(msg.peer)

			case getSyncPeerMsg:
				var peerID int32
				if sm.syncPeer != nil {
					peerID = sm.syncPeer.ID()
				}
				msg.reply <- peerID

			case processBlockMsg:
				_, isOrphan, err := sm.chain.ProcessBlock(
					msg.block, msg.flags)
				if err != nil {
					msg.reply <- processBlockResponse{
						isOrphan: false,
						err:      err,
					}
				}

				msg.reply <- processBlockResponse{
					isOrphan: isOrphan,
					err:      nil,
				}

			case isCurrentMsg:
				msg.reply <- sm.current()

			case pauseMsg:
				// Wait until the sender unpauses the manager.
				<-msg.unpause

			default:
				log.Warnf("Invalid message type in block "+
					"handler: %T", msg)
			}

		case m := <-slow:
			switch msg := m.(type) {
			case *newPeerMsg:
				sm.handleNewPeerMsg(msg.peer)

			case *txMsg:
				sm.handleTxMsg(msg)
				msg.reply <- struct{}{}

			case *blockMsg:
				sm.handleBlockMsg(msg)
				msg.reply <- struct{}{}

			case *invMsg:
				sm.handleInvMsg(msg)

			case *headersMsg:
				sm.handleHeadersMsg(msg)

			case *donePeerMsg:
				sm.handleDonePeerMsg(msg.peer)

			case getSyncPeerMsg:
				var peerID int32
				if sm.syncPeer != nil {
					peerID = sm.syncPeer.ID()
				}
				msg.reply <- peerID

			case processBlockMsg:
				_, isOrphan, err := sm.chain.ProcessBlock(
					msg.block, msg.flags)
				if err != nil {
					msg.reply <- processBlockResponse{
						isOrphan: false,
						err:      err,
					}
				}

				msg.reply <- processBlockResponse{
					isOrphan: isOrphan,
					err:      nil,
				}

			case isCurrentMsg:
				msg.reply <- sm.current()

			case pauseMsg:
				// Wait until the sender unpauses the manager.
				<-msg.unpause

			default:
				log.Warnf("Invalid message type in block "+
					"handler: %T", msg)
			}

		case <-stallTicker.C:
			sm.handleStallSample()

		case <-sm.quit:
			break out
		}
	}

	sm.wg.Done()
	log.Trace("Block handler done")
}

// handleBlockchainNotification handles notifications from blockchain.  It does
// things such as request orphan block parents and relay accepted blocks to
// connected peers.
func (sm *SyncManager) handleBlockchainNotification(notification *blockdag.Notification) {
	switch notification.Type {
	// A block has been accepted into the block chain.  Relay it to other
	// peers.
	case blockdag.NTBlockAccepted:
		// TODO(cedric): We may need to redefine what current() means in DAG. With the current definition of being below
		// the max height of peers, this would mean that if we received a block at around the same time we generated one,
		// we wouldn't relay our new block to our peers. (the notification code doesn't distinguish between relayed blocks
		// and blocks we generated).
		// For now we'll just not consider 'current' when propagating block info to peers.
		//
		// Don't relay if we are not current. Other peers that are
		// current should already know about it.
		//if !sm.current() {
		//	return
		//}

		block, ok := notification.Data.(*soterutil.Block)
		if !ok {
			log.Warnf("Chain accepted notification is not a block.")
			break
		}

		// Generate the inventory vector and relay it.
		iv := wire.NewInvVect(wire.InvTypeBlock, block.Hash(), block.Height())
		sm.peerNotifier.RelayInventory(iv, block.MsgBlock().Header)

	// A block has been connected to the main block chain.
	case blockdag.NTBlockConnected:
		block, ok := notification.Data.(*soterutil.Block)
		if !ok {
			log.Warnf("Chain connected notification is not a block.")
			break
		}

		// Remove all of the transactions (except the coinbase) in the
		// connected block from the transaction pool.  Secondly, remove any
		// transactions which are now double spends as a result of these
		// new transactions.  Finally, remove any transaction that is
		// no longer an orphan. Transactions which depend on a confirmed
		// transaction are NOT removed recursively because they are still
		// valid.
		for _, tx := range block.Transactions()[1:] {
			sm.txMemPool.RemoveTransaction(tx, false)
			sm.txMemPool.RemoveDoubleSpends(tx)
			sm.txMemPool.RemoveOrphan(tx)
			sm.peerNotifier.TransactionConfirmed(tx)
			acceptedTxs := sm.txMemPool.ProcessOrphans(tx)
			sm.peerNotifier.AnnounceNewTransactions(acceptedTxs)
		}

		// Register block with the fee estimator, if it exists.
		if sm.feeEstimator != nil {
			err := sm.feeEstimator.RegisterBlock(block)

			// If an error is somehow generated then the fee estimator
			// has entered an invalid state. Since it doesn't know how
			// to recover, create a new one.
			if err != nil {
				sm.feeEstimator = mempool.NewFeeEstimator(
					mempool.DefaultEstimateFeeMaxRollback,
					mempool.DefaultEstimateFeeMinRegisteredBlocks)
			}
		}

	// A block has been disconnected from the main block chain.
	case blockdag.NTBlockDisconnected:
		block, ok := notification.Data.(*soterutil.Block)
		if !ok {
			log.Warnf("Chain disconnected notification is not a block.")
			break
		}

		// Reinsert all of the transactions (except the coinbase) into
		// the transaction pool.
		for _, tx := range block.Transactions()[1:] {
			_, _, err := sm.txMemPool.MaybeAcceptTransaction(tx,
				false, false)
			if err != nil {
				// Remove the transaction and all transactions
				// that depend on it if it wasn't accepted into
				// the transaction pool.
				sm.txMemPool.RemoveTransaction(tx, true)
			}
		}

		// Rollback previous block recorded by the fee estimator.
		if sm.feeEstimator != nil {
			sm.feeEstimator.Rollback(block.Hash())
		}
	}
}

// invMaxBlockHeight returns the max height of blocks in inventory
func invMaxBlockHeight(inventory []*wire.InvVect) int32 {
	var maxHeight int32
	for _, iv := range inventory {
		switch iv.Type {
			case wire.InvTypeBlock:
			case wire.InvTypeWitnessBlock:
			default:
				continue
		}

		if iv.Height > maxHeight {
			maxHeight = iv.Height
		}
	}

	return maxHeight
}

// reqBlocks sends a getdata message to the peer for inventory we need
func (sm *SyncManager) reqBlocks(peer *peerpkg.Peer, inventory []*wire.InvVect) {
	state, exists := sm.peerStates[peer]
	if !exists {
		log.Errorf("Can't request blocks from unknown peer %s", peer)
		return
	}

	// Filter inventory to items we need from peer
	needed := sm.filterNeededInv(peer, inventory)

	if len(inventory) > 0 && len(needed) == 0 && !sm.hasPendingBlocks(peer) && sm.syncPeer != nil {
		// The peer has sent us inventory of blocks we already have, and there's no more pending blocks for this peer.
		// This situation has likely occurred due to out-of-order processing of blocks.
		//
		// We will respond by asking the peer if they have any more blocks for us.
		log.Debugf("Possible out-of-order sync flow detected with peer %s. Asking peer if it has more blocks for us.", peer)
		maxHeight := sm.chain.DAGSnapshot().MaxHeight
		locator := sm.chain.BlockLocatorFromHeight(maxHeight)
		err := peer.PushGetBlocksMsg(locator, &zeroHash)
		if err != nil {
			log.Warnf("Failed to send getblocks message to peer %s: %s", peer, err)
		}
		return
	}

	// Add needed inventory to the request queue for the peer
	for _, iv := range needed {
		state.requestQueue = append(state.requestQueue, iv)
	}

	// Generate getdata message, for needed inventory
	numRequested := 0
	gdmsg := wire.NewMsgGetData()
	// Operate on a new slice, which we'll shrink as we add inventory to the getdata message.
	// We'll then re-assign the shrunken slice to the peer's requestQueue
	requestQueue := state.requestQueue

	for len(requestQueue) > 0 {
		iv := requestQueue[0]
		requestQueue[0] = nil
		requestQueue = requestQueue[1:]

		switch iv.Type {
		case wire.InvTypeWitnessBlock:
			fallthrough
		case wire.InvTypeBlock:
			// Request the block if there is not already a pending request that hasn't expired
			if sm.canReqBlock(state, iv.Hash) {
				// Track that we're making a request for this block
				sm.trackReqBlock(state, iv.Hash)

				if peer.IsWitnessEnabled() {
					iv.Type = wire.InvTypeWitnessBlock
				}

				gdmsg.AddInvVect(iv)
				numRequested++
			}

		case wire.InvTypeWitnessTx:
			fallthrough
		case wire.InvTypeTx:
			// Request the transaction if there is not already a pending request that hasn't expired
			if sm.canReqTxn(state, iv.Hash) {
				sm.trackReqTxn(state, iv.Hash)

				// If the peer is capable, request the txn
				// including all witness data.
				if peer.IsWitnessEnabled() {
					iv.Type = wire.InvTypeWitnessTx
				}

				gdmsg.AddInvVect(iv)
				numRequested++
			}
		}

		if numRequested >= wire.MaxInvPerMsg {
			break
		}
	}

	state.requestQueue = requestQueue
	if len(gdmsg.InvList) > 0 {
		peer.QueueMessage(gdmsg, nil)
	}
}

// trackReqBlock tracks the request for the block in sync manager
// Request tracking is used for two purposes:
// 1. To reduce duplicate messages from being sent to peers
// 2. To prevent handleBlockMsg from dropping the block (thinking that it wasn't requested)
func (sm *SyncManager) trackReqBlock(state *peerSyncState, hash chainhash.Hash) {
	reqExp := NewRequestExpiry()

	// Global block request tracking
	_, exists := sm.requestedBlocks[hash]
	if !exists {
		sm.requestedBlocks[hash] = reqExp
		sm.limitTrackedReqs(sm.requestedBlocks, maxRequestedBlocks)
	} else {
		sm.requestedBlocks[hash].attempts++
	}

	// Peer block request tracking
	_, exists = state.requestedBlocks[hash]
	if !exists {
		state.requestedBlocks[hash] = reqExp
	} else {
		state.requestedBlocks[hash].attempts++
	}
}

// trackReqTxn tracks the request for the transaction in sync manager
func (sm *SyncManager) trackReqTxn(state *peerSyncState, hash chainhash.Hash) {
	reqExp := NewRequestExpiry()

	// Global block request tracking
	_, exists := sm.requestedTxns[hash]
	if !exists {
		sm.requestedTxns[hash] = reqExp
		sm.limitTrackedReqs(sm.requestedTxns, maxRequestedTxns)
	} else {
		sm.requestedTxns[hash].attempts++
	}

	// Peer transaction request tracking
	_, exists = state.requestedTxns[hash]
	if !exists {
		state.requestedTxns[hash] = reqExp
	} else {
		state.requestedTxns[hash].attempts++
	}
}

// updatePeerHeight update's the peer's height to the height of the block for the hash, if it's known to us.
// Return true if we updated the height, false if not.
func (sm *SyncManager) updatePeerHeight(peer *peerpkg.Peer, hash *chainhash.Hash) (bool, error) {
	have, err := sm.chain.HaveBlock(hash)
	if err != nil {
		return false, err
	}

	if have {
		blkHeight, err := sm.chain.BlockHeightByHash(hash)
		if err != nil {
			return false, err
		} else if blkHeight > peer.MaxBlockHeight() {
			peer.UpdateMaxBlockHeight(blkHeight)
			return true, err
		}
	}

	return false, err
}

// NewPeer informs the sync manager of a newly active peer.
func (sm *SyncManager) NewPeer(peer *peerpkg.Peer) {
	// Ignore if we are shutting down.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		return
	}
	sm.msgChan <- &newPeerMsg{peer: peer}
}

// QueueTx adds the passed transaction message and peer to the block handling
// queue. Responds to the done channel argument after the tx message is
// processed.
func (sm *SyncManager) QueueTx(tx *soterutil.Tx, peer *peerpkg.Peer, done chan struct{}) {
	// Don't accept more transactions if we're shutting down.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		done <- struct{}{}
		return
	}

	sm.msgChan <- &txMsg{tx: tx, peer: peer, reply: done}
}

// QueueBlock adds the passed block message and peer to the block handling
// queue. Responds to the done channel argument after the block message is
// processed.
func (sm *SyncManager) QueueBlock(block *soterutil.Block, peer *peerpkg.Peer, done chan struct{}) {
	// Don't accept more blocks if we're shutting down.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		done <- struct{}{}
		return
	}

	sm.msgChan <- &blockMsg{block: block, peer: peer, reply: done}
}

// QueueInv adds the passed inv message and peer to the block handling queue.
func (sm *SyncManager) QueueInv(inv *wire.MsgInv, peer *peerpkg.Peer) {
	// No channel handling here because peers do not need to block on inv
	// messages.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		return
	}

	sm.msgChan <- &invMsg{inv: inv, peer: peer}
}

// QueueHeaders adds the passed headers message and peer to the block handling
// queue.
func (sm *SyncManager) QueueHeaders(headers *wire.MsgHeaders, peer *peerpkg.Peer) {
	// No channel handling here because peers do not need to block on
	// headers messages.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		return
	}

	sm.msgChan <- &headersMsg{headers: headers, peer: peer}
}

// DonePeer informs the blockmanager that a peer has disconnected.
func (sm *SyncManager) DonePeer(peer *peerpkg.Peer) {
	// Ignore if we are shutting down.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		return
	}

	sm.msgChan <- &donePeerMsg{peer: peer}
}

// Start begins the core block handler which processes block and inv messages.
func (sm *SyncManager) Start() {
	// Already started?
	if atomic.AddInt32(&sm.started, 1) != 1 {
		return
	}

	log.Trace("Starting sync manager")
	sm.wg.Add(1)
	go sm.blockHandler()
}

// Stop gracefully shuts down the sync manager by stopping all asynchronous
// handlers and waiting for them to finish.
func (sm *SyncManager) Stop() error {
	if atomic.AddInt32(&sm.shutdown, 1) != 1 {
		log.Warn("Sync manager is already in the process of stopping")
		return nil
	}

	log.Info("Sync manager stopping")
	close(sm.quit)
	sm.wg.Wait()
	log.Info("Sync manager stopped")
	return nil
}

// SyncPeerID returns the ID of the current sync peer, or 0 if there is none.
func (sm *SyncManager) SyncPeerID() int32 {
	reply := make(chan int32)
	sm.msgChan <- getSyncPeerMsg{reply: reply}
	return <-reply
}

// ProcessBlock makes use of ProcessBlock on an internal instance of a block
// chain.
func (sm *SyncManager) ProcessBlock(block *soterutil.Block, flags blockdag.BehaviorFlags) (bool, error) {
	reply := make(chan processBlockResponse, 1)
	sm.msgChan <- processBlockMsg{block: block, flags: flags, reply: reply}
	response := <-reply
	return response.isOrphan, response.err
}

// IsCurrent returns whether or not the sync manager believes it is synced with
// the connected peers.
func (sm *SyncManager) IsCurrent() bool {
	reply := make(chan bool)
	sm.msgChan <- isCurrentMsg{reply: reply}
	return <-reply
}

// Pause pauses the sync manager until the returned channel is closed.
//
// Note that while paused, all peer and block processing is halted.  The
// message sender should avoid pausing the sync manager for long durations.
func (sm *SyncManager) Pause() chan<- struct{} {
	c := make(chan struct{})
	sm.msgChan <- pauseMsg{c}
	return c
}

// New constructs a new SyncManager. Use Start to begin processing asynchronous
// block, tx, and inv updates.
func New(config *Config) (*SyncManager, error) {
	sm := SyncManager{
		peerNotifier:    config.PeerNotifier,
		chain:           config.Chain,
		txMemPool:       config.TxMemPool,
		chainParams:     config.ChainParams,
		rejectedTxns:    make(map[chainhash.Hash]struct{}),
		requestedTxns:   make(map[chainhash.Hash]*requestExpiry),
		requestedBlocks: make(map[chainhash.Hash]*requestExpiry),
		peerStates:      make(map[*peerpkg.Peer]*peerSyncState),
		progressLogger:  newBlockProgressLogger("Processed", log),
		msgChan:         make(chan interface{}, config.MaxPeers*3),
		headerList:      list.New(),
		quit:            make(chan struct{}),
		feeEstimator:    config.FeeEstimator,
	}

	// NOTE(cedric): Commented out to disable checkpoint-related code (JIRA DAG-3)
	// 
	//
	// best := sm.chain.BestSnapshot()
	// if !config.DisableCheckpoints {
	// 	// Initialize the next checkpoint based on the current height.
	// 	sm.nextCheckpoint = sm.findNextHeaderCheckpoint(best.Height)
	// 	if sm.nextCheckpoint != nil {
	// 		sm.resetHeaderState(&best.Hash, best.Height)
	// 	}
	// } else {
	// 	log.Info("Checkpoints are disabled")
	// }
	log.Info("Checkpoints are disabled")

	sm.chain.Subscribe(sm.handleBlockchainNotification)

	return &sm, nil
}
