// Copyright (c) 2016-2017 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdag

import (
	"fmt"

	"github.com/soteria-dag/soterd/chaincfg/chainhash"
)

// ThresholdState define the various threshold states used when voting on
// consensus changes.
type ThresholdState byte

// These constants are used to identify specific threshold states.
const (
	// ThresholdDefined is the first state for each deployment and is the
	// state for the genesis block has by definition for all deployments.
	ThresholdDefined ThresholdState = iota

	// ThresholdStarted is the state for a deployment once its start time
	// has been reached.
	ThresholdStarted

	// ThresholdLockedIn is the state for a deployment during the retarget
	// period which is after the ThresholdStarted state period and the
	// number of blocks that have voted for the deployment equal or exceed
	// the required number of votes for the deployment.
	ThresholdLockedIn

	// ThresholdActive is the state for a deployment for all blocks after a
	// retarget period in which the deployment was in the ThresholdLockedIn
	// state.
	ThresholdActive

	// ThresholdFailed is the state for a deployment once its expiration
	// time has been reached and it did not reach the ThresholdLockedIn
	// state.
	ThresholdFailed

	// numThresholdsStates is the maximum number of threshold states used in
	// tests.
	numThresholdsStates
)

// thresholdStateStrings is a map of ThresholdState values back to their
// constant names for pretty printing.
var thresholdStateStrings = map[ThresholdState]string{
	ThresholdDefined:  "ThresholdDefined",
	ThresholdStarted:  "ThresholdStarted",
	ThresholdLockedIn: "ThresholdLockedIn",
	ThresholdActive:   "ThresholdActive",
	ThresholdFailed:   "ThresholdFailed",
}

// String returns the ThresholdState as a human-readable name.
func (t ThresholdState) String() string {
	if s := thresholdStateStrings[t]; s != "" {
		return s
	}
	return fmt.Sprintf("Unknown ThresholdState (%d)", int(t))
}

// thresholdConditionChecker provides a generic interface that is invoked to
// determine when a consensus rule change threshold should be changed.
type thresholdConditionChecker interface {
	// BeginTime returns the unix timestamp for the median block time after
	// which voting on a rule change starts (at the next window).
	BeginTime() uint64

	// EndTime returns the unix timestamp for the median block time after
	// which an attempted rule change fails if it has not already been
	// locked in or activated.
	EndTime() uint64

	// RuleChangeActivationThreshold is the number of blocks for which the
	// condition must be true in order to lock in a rule change.
	RuleChangeActivationThreshold() uint32

	// MinerConfirmationWindow is the number of blocks in each threshold
	// state retarget window.
	MinerConfirmationWindow() uint32

	// Condition returns whether or not the rule change activation condition
	// has been met.  This typically involves checking whether or not the
	// bit associated with the condition is set, but can be more complex as
	// needed.
	Condition(*blockNode) (bool, error)
}

// thresholdStateCache provides a type to cache the threshold states of each
// threshold window for a set of IDs.
type thresholdStateCache struct {
	entries map[chainhash.Hash]ThresholdState
}

// Lookup returns the threshold state associated with the given hash along with
// a boolean that indicates whether or not it is valid.
func (c *thresholdStateCache) Lookup(hash *chainhash.Hash) (ThresholdState, bool) {
	state, ok := c.entries[*hash]
	return state, ok
}

// Update updates the cache to contain the provided hash to threshold state
// mapping.
func (c *thresholdStateCache) Update(hash *chainhash.Hash, state ThresholdState) {
	c.entries[*hash] = state
}

// newThresholdCaches returns a new array of caches to be used when calculating
// threshold states.
func newThresholdCaches(numCaches uint32) []thresholdStateCache {
	caches := make([]thresholdStateCache, numCaches)
	for i := 0; i < len(caches); i++ {
		caches[i] = thresholdStateCache{
			entries: make(map[chainhash.Hash]ThresholdState),
		}
	}
	return caches
}

// thresholdState returns the current rule change threshold state for the block
// AFTER the given nodes and deployment ID.  The cache is used to ensure the
// threshold states for previous windows are only calculated once.
//
// This function MUST be called with the chain state lock held (for writes).
func (b *BlockDAG) thresholdStates(nodes []*blockNode, checker thresholdConditionChecker, cache *thresholdStateCache) ([]ThresholdState, error) {
	// The threshold state for the window that contains the genesis block is
	// defined by definition.
	confirmationWindow := int32(checker.MinerConfirmationWindow())

	if nodes == nil {
		return []ThresholdState{ThresholdDefined}, nil
	}

	for _, node := range nodes {
		if node.height+1 < confirmationWindow {
			return []ThresholdState{ThresholdDefined}, nil
		}
	}

	// Get the ancestors that are the last blocks of the previous confirmation
	// window in order to get their threshold states. This can be done because
	// the state is the same for all blocks within a given window.
	var prevNodes []*blockNode
	for _, node := range nodes {
		ancestors := node.Ancestors(node.height - (node.height+1)%confirmationWindow)
		for _, ancestorNode := range ancestors {
			prevNodes = append(prevNodes, ancestorNode)
		}
	}

	// Iterate backwards through each of the previous confirmation windows
	// to find the most recently cached threshold state.
	var neededStates []*blockNode
	for prevNodes != nil {
		// Nothing more to do if the state of the blocks are already
		// cached.
		cachedCount := 0
		for _, node := range prevNodes {
			_, ok := cache.Lookup(&node.hash)
			if ok {
				cachedCount++
			}
		}
		if cachedCount == len(prevNodes) {
			break
		}

		// The start and expiration times are based on the median block
		// time, so calculate it now.
		prevNodesBeforeStartTime := 0
		for _, node := range prevNodes {
			medianTime := node.CalcPastMedianTime()

			// The state is simply defined if the start time hasn't been
			// been reached yet.
			if uint64(medianTime.Unix()) < checker.BeginTime() {
				cache.Update(&node.hash, ThresholdDefined)
				prevNodesBeforeStartTime++
			} else {
				// Add this node to the list of nodes that need the state
				// calculated and cached.
				neededStates = append(neededStates, node)
			}
		}
		if prevNodesBeforeStartTime == len(prevNodes) {
			break
		}

		// Get the ancestors that are the last blocks of the previous
		// confirmation window.
		var newPrevNodes []*blockNode
		for _, node := range prevNodes {
			ancestors := node.RelativeAncestors(confirmationWindow)
			for _, nodeParent := range ancestors {
				newPrevNodes = append(newPrevNodes, nodeParent)
			}
		}
		prevNodes = newPrevNodes
	}

	// Start with the threshold state for the most recent confirmation
	// window that has a cached state.
	if prevNodes != nil {
		for _, node := range prevNodes {
			_, ok := cache.Lookup(&node.hash)
			if !ok {
				return []ThresholdState{ThresholdFailed}, AssertError(fmt.Sprintf(
					"thresholdState: cache lookup failed for %v",
					node.hash))
			}
		}
	}

	var states []ThresholdState
	for j := 0; j < len(prevNodes); j++ {
		state := ThresholdDefined

		// Since each threshold state depends on the state of the previous
		// window, iterate starting from the oldest unknown window.
		for neededNum := len(neededStates) - 1; neededNum >= 0; neededNum-- {
			prevNode := neededStates[neededNum]

			switch state {
			case ThresholdDefined:
				// The deployment of the rule change fails if it expires
				// before it is accepted and locked in.
				medianTime := prevNode.CalcPastMedianTime()
				medianTimeUnix := uint64(medianTime.Unix())
				if medianTimeUnix >= checker.EndTime() {
					state = ThresholdFailed
					break
				}

				// The state for the rule moves to the started state
				// once its start time has been reached (and it hasn't
				// already expired per the above).
				if medianTimeUnix >= checker.BeginTime() {
					state = ThresholdStarted
				}

			case ThresholdStarted:
				// The deployment of the rule change fails if it expires
				// before it is accepted and locked in.
				medianTime := prevNode.CalcPastMedianTime()
				if uint64(medianTime.Unix()) >= checker.EndTime() {
					state = ThresholdFailed
					break
				}

				// At this point, the rule change is still being voted
				// on by the miners, so iterate backwards through the
				// confirmation window to count all of the votes in it.
				//
				// NOTE(cedric): Now that there are more potential prevNodes,
				// this could result in increased votes in BlockDAG compared to BlockChain.
				var count uint32
				countNodes := []*blockNode{prevNode}
				for i := int32(0); i < confirmationWindow; i++ {
					var newCountNodes []*blockNode
					for _, countNode := range countNodes {
						condition, err := checker.Condition(countNode)
						if err != nil {
							states = append(states, ThresholdFailed)
							return states, err
						}
						if condition {
							count++
						}

						// Get the previous block nodes.
						for _, parent := range countNode.parents {
							newCountNodes = append(newCountNodes, parent)
						}
					}
					countNodes = newCountNodes
				}

				// The state is locked in if the number of blocks in the
				// period that voted for the rule change meets the
				// activation threshold.
				if count >= checker.RuleChangeActivationThreshold() {
					state = ThresholdLockedIn
				}

			case ThresholdLockedIn:
				// The new rule becomes active when its previous state
				// was locked in.
				state = ThresholdActive

			// Nothing to do if the previous state is active or failed since
			// they are both terminal states.
			case ThresholdActive:
			case ThresholdFailed:
			}

			// Update the cache to avoid recalculating the state in the
			// future.
			cache.Update(&prevNode.hash, state)
		}
		states = append(states, state)
	}

	return states, nil
}

// ThresholdStates returns the current rule change threshold state of the given
// deployment ID for the block AFTER the end of the current best chain.
//
// This function is safe for concurrent access.
func (b *BlockDAG) ThresholdStates(deploymentID uint32) ([]ThresholdState, error) {
	b.chainLock.Lock()
	states, err := b.deploymentStates(b.dView.Tips(), deploymentID)
	b.chainLock.Unlock()

	return states, err
}

// IsDeploymentActive returns true if the target deploymentID is active, and
// false otherwise.
//
// This function is safe for concurrent access.
func (b *BlockDAG) IsDeploymentActive(deploymentID uint32) (bool, error) {
	b.chainLock.Lock()
	states, err := b.deploymentStates(b.dView.Tips(), deploymentID)
	b.chainLock.Unlock()
	if err != nil {
		return false, err
	}
	activeCount := 0
	for _, state := range states {
		if state == ThresholdActive {
			activeCount++
		}
	}

	return activeCount == len(states), nil
}

// deploymentStates returns the current rule change thresholds for a given
// deploymentID. The thresholds are evaluated from the point of view of the block
// nodes passed in as the first argument to this method.
//
// It is important to note that, as the variable name indicates, this function
// expects the block nodes prior to the block for which the deployment state is
// desired.  In other words, the returned deployment state is for the block
// AFTER the passed nodes.
//
// This function MUST be called with the chain state lock held (for writes).
func (b *BlockDAG) deploymentStates(prevNodes []*blockNode, deploymentID uint32) ([]ThresholdState, error) {
	if deploymentID > uint32(len(b.chainParams.Deployments)) {
		return []ThresholdState{ThresholdFailed}, DeploymentError(deploymentID)
	}

	deployment := &b.chainParams.Deployments[deploymentID]
	checker := deploymentChecker{deployment: deployment, chain: b}
	cache := &b.deploymentCaches[deploymentID]

	return b.thresholdStates(prevNodes, checker, cache)
}

// initThresholdCaches initializes the threshold state caches for each warning
// bit and defined deployment and provides warnings if the chain is current per
// the warnUnknownVersions and warnUnknownRuleActivations functions.
func (b *BlockDAG) initThresholdCaches() error {
	// Initialize the warning and deployment caches by calculating the
	// threshold state for each of them.  This will ensure the caches are
	// populated and any states that needed to be recalculated due to
	// definition changes is done now.
	for _, prevNode := range b.dView.Tips() {
		for bit := uint32(0); bit < vbNumBits; bit++ {
			checker := bitConditionChecker{bit: bit, chain: b}
			cache := &b.warningCaches[bit]
			_, err := b.thresholdStates(prevNode.parents, checker, cache)
			if err != nil {
				return err
			}
		}

		for id := 0; id < len(b.chainParams.Deployments); id++ {
			deployment := &b.chainParams.Deployments[id]
			cache := &b.deploymentCaches[id]
			checker := deploymentChecker{deployment: deployment, chain: b}
			_, err := b.thresholdStates(prevNode.parents, checker, cache)
			if err != nil {
				return err
			}
		}
	}

	// No warnings about unknown rules or versions until the chain is
	// current.
	if b.isCurrent() {
		// Warn if a high enough percentage of the last blocks have
		// unexpected versions.
		tips := b.dView.Tips()
		if err := b.warnUnknownVersions(tips); err != nil {
			return err
		}

		// Warn if any unknown new rules are either about to activate or
		// have already been activated.
		if err := b.warnUnknownRuleActivations(tips); err != nil {
			return err
		}
	}

	return nil
}
