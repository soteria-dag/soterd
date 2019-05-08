// Copyright (c) 2016-2017 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdag

import (
	"math"

	"github.com/soteria-dag/soterd/chaincfg"
)

const (
	// vbLegacyBlockVersion is the highest legacy block version before the
	// version bits scheme became active.
	vbLegacyBlockVersion = 4

	// vbTopBits defines the bits to set in the version to signal that the
	// version bits scheme is being used.
	vbTopBits = 0x20000000

	// vbTopMask is the bitmask to use to determine whether or not the
	// version bits scheme is in use.
	vbTopMask = 0xe0000000

	// vbNumBits is the total number of bits available for use with the
	// version bits scheme.
	vbNumBits = 29

	// unknownVerNumToCheck is the number of previous blocks to consider
	// when checking for a threshold of unknown block versions for the
	// purposes of warning the user.
	unknownVerNumToCheck = 100

	// unknownVerWarnNum is the threshold of previous blocks that have an
	// unknown version to use for the purposes of warning the user.
	unknownVerWarnNum = unknownVerNumToCheck / 2
)

// bitConditionChecker provides a thresholdConditionChecker which can be used to
// test whether or not a specific bit is set when it's not supposed to be
// according to the expected version based on the known deployments and the
// current state of the chain.  This is useful for detecting and warning about
// unknown rule activations.
type bitConditionChecker struct {
	bit   uint32
	chain *BlockDAG
}

// Ensure the bitConditionChecker type implements the thresholdConditionChecker
// interface.
var _ thresholdConditionChecker = bitConditionChecker{}

// BeginTime returns the unix timestamp for the median block time after which
// voting on a rule change starts (at the next window).
//
// Since this implementation checks for unknown rules, it returns 0 so the rule
// is always treated as active.
//
// This is part of the thresholdConditionChecker interface implementation.
func (c bitConditionChecker) BeginTime() uint64 {
	return 0
}

// EndTime returns the unix timestamp for the median block time after which an
// attempted rule change fails if it has not already been locked in or
// activated.
//
// Since this implementation checks for unknown rules, it returns the maximum
// possible timestamp so the rule is always treated as active.
//
// This is part of the thresholdConditionChecker interface implementation.
func (c bitConditionChecker) EndTime() uint64 {
	return math.MaxUint64
}

// RuleChangeActivationThreshold is the number of blocks for which the condition
// must be true in order to lock in a rule change.
//
// This implementation returns the value defined by the chain params the checker
// is associated with.
//
// This is part of the thresholdConditionChecker interface implementation.
func (c bitConditionChecker) RuleChangeActivationThreshold() uint32 {
	return c.chain.chainParams.RuleChangeActivationThreshold
}

// MinerConfirmationWindow is the number of blocks in each threshold state
// retarget window.
//
// This implementation returns the value defined by the chain params the checker
// is associated with.
//
// This is part of the thresholdConditionChecker interface implementation.
func (c bitConditionChecker) MinerConfirmationWindow() uint32 {
	return c.chain.chainParams.MinerConfirmationWindow
}

// Condition returns true when the specific bit associated with the checker is
// set and it's not supposed to be according to the expected version based on
// the known deployments and the current state of the chain.
//
// This function MUST be called with the chain state lock held (for writes).
//
// This is part of the thresholdConditionChecker interface implementation.
func (c bitConditionChecker) Condition(node *blockNode) (bool, error) {
	conditionMask := uint32(1) << c.bit
	version := uint32(node.version)
	if version&vbTopMask != vbTopBits {
		return false, nil
	}
	if version&conditionMask == 0 {
		return false, nil
	}

	expectedVersion, err := c.chain.calcNextBlockVersion(node.parents)
	if err != nil {
		return false, err
	}
	return uint32(expectedVersion)&conditionMask == 0, nil
}

// deploymentChecker provides a thresholdConditionChecker which can be used to
// test a specific deployment rule.  This is required for properly detecting
// and activating consensus rule changes.
type deploymentChecker struct {
	deployment *chaincfg.ConsensusDeployment
	chain      *BlockDAG
}

// Ensure the deploymentChecker type implements the thresholdConditionChecker
// interface.
var _ thresholdConditionChecker = deploymentChecker{}

// BeginTime returns the unix timestamp for the median block time after which
// voting on a rule change starts (at the next window).
//
// This implementation returns the value defined by the specific deployment the
// checker is associated with.
//
// This is part of the thresholdConditionChecker interface implementation.
func (c deploymentChecker) BeginTime() uint64 {
	return c.deployment.StartTime
}

// EndTime returns the unix timestamp for the median block time after which an
// attempted rule change fails if it has not already been locked in or
// activated.
//
// This implementation returns the value defined by the specific deployment the
// checker is associated with.
//
// This is part of the thresholdConditionChecker interface implementation.
func (c deploymentChecker) EndTime() uint64 {
	return c.deployment.ExpireTime
}

// RuleChangeActivationThreshold is the number of blocks for which the condition
// must be true in order to lock in a rule change.
//
// This implementation returns the value defined by the chain params the checker
// is associated with.
//
// This is part of the thresholdConditionChecker interface implementation.
func (c deploymentChecker) RuleChangeActivationThreshold() uint32 {
	return c.chain.chainParams.RuleChangeActivationThreshold
}

// MinerConfirmationWindow is the number of blocks in each threshold state
// retarget window.
//
// This implementation returns the value defined by the chain params the checker
// is associated with.
//
// This is part of the thresholdConditionChecker interface implementation.
func (c deploymentChecker) MinerConfirmationWindow() uint32 {
	return c.chain.chainParams.MinerConfirmationWindow
}

// Condition returns true when the specific bit defined by the deployment
// associated with the checker is set.
//
// This is part of the thresholdConditionChecker interface implementation.
func (c deploymentChecker) Condition(node *blockNode) (bool, error) {
	conditionMask := uint32(1) << c.deployment.BitNumber
	version := uint32(node.version)
	return (version&vbTopMask == vbTopBits) && (version&conditionMask != 0),
		nil
}

// calcNextBlockVersion calculates the expected version of the block after the
// passed previous block node based on the state of started and locked in
// rule change deployments.
//
// This function differs from the exported CalcNextBlockVersion in that the
// exported version uses the current best chain as the previous block node
// while this function accepts any block node.
//
// This function MUST be called with the chain state lock held (for writes).
func (b *BlockDAG) calcNextBlockVersion(prevNodes []*blockNode) (int32, error) {
	// Set the appropriate bits for each actively defined rule deployment
	// that is either in the process of being voted on, or locked in for the
	// activation at the next threshold window change.

	// This is a map of expectedVersion to number of parents that have it
	versions := make(map[int32]int)

	for id := 0; id < len(b.chainParams.Deployments); id++ {
		deployment := &b.chainParams.Deployments[id]
		cache := &b.deploymentCaches[id]
		checker := deploymentChecker{deployment: deployment, chain: b}
		states, err := b.thresholdStates(prevNodes, checker, cache)
		if err != nil {
			return 0, err
		}

		for _, state := range states {
			expectedVersion := uint32(vbTopBits)
			if state == ThresholdStarted || state == ThresholdLockedIn {
				expectedVersion |= uint32(1) << deployment.BitNumber
			}
			_, ok := versions[int32(expectedVersion)]
			if !ok {
				versions[int32(expectedVersion)] = 1
			} else {
				versions[int32(expectedVersion)]++
			}
		}
	}

	var mostPopularVersion int32
	var highestCount int
	for version, count := range versions {
		if count > highestCount {
			mostPopularVersion = version
			highestCount = count
		}
	}
	return mostPopularVersion, nil
}

// TODO(cedric): We may want to revisit how this is determined
// calcNextParentVersion calculates the expected version of the parent sub-header
// after the passed previous block nodes, based on what the majority of those nodes used.
func (b *BlockDAG) calcNextParentVersion(prevNodes []*blockNode) (int32, error) {
	// This is a map of expectedVersion to number of parents that have it
	versions := make(map[int32]int)

	for _, node := range prevNodes {
		_, ok := versions[node.parentVersion]
		if !ok {
			versions[node.parentVersion] = 1
		} else {
			versions[node.parentVersion]++
		}
	}

	// Set initial version, in case we are calculating next parent version for genesis block
	mostPopularVersion := int32(uint32(vbTopBits))
	var highestCount int
	for version, count := range versions {
		if count > highestCount {
			mostPopularVersion = version
			highestCount = count
		}
	}

	return mostPopularVersion, nil
}

// CalcNextBlockVersion calculates the expected version of the block after the
// end of the current best chain based on the state of started and locked in
// rule change deployments.
//
// This function is safe for concurrent access.
func (b *BlockDAG) CalcNextBlockVersion() (int32, error) {
	b.chainLock.Lock()
	version, err := b.calcNextBlockVersion(b.dView.Tips())
	b.chainLock.Unlock()
	return version, err
}

// CalcNextParentVersion calculates the expected version of the parent sub-header after the
// end of the current best chain based on what the most tips used in their parent sub-headers.
//
// This function is safe for concurrent access.
func (b *BlockDAG) CalcNextParentVersion() (int32, error) {
	b.chainLock.Lock()
	version, err := b.calcNextParentVersion(b.dView.Tips())
	b.chainLock.Unlock()
	return version, err
}

// warnUnknownRuleActivations displays a warning when any unknown new rules are
// either about to activate or have been activated.  This will only happen once
// when new rules have been activated and every block for those about to be
// activated.
//
// This function MUST be called with the chain state lock held (for writes)
func (b *BlockDAG) warnUnknownRuleActivations(nodes []*blockNode) error {
	// Warn if any unknown new rules are either about to activate or have
	// already been activated.
	for _, node := range nodes {
		for bit := uint32(0); bit < vbNumBits; bit++ {
			checker := bitConditionChecker{bit: bit, chain: b}
			cache := &b.warningCaches[bit]
			states, err := b.thresholdStates(node.parents, checker, cache)
			if err != nil {
				return err
			}

			for _, state := range states {
				switch state {
				case ThresholdActive:
					if !b.unknownRulesWarned {
						log.Warnf("Unknown new rules activated (bit %d)",
							bit)
						b.unknownRulesWarned = true
					}

				case ThresholdLockedIn:
					window := int32(checker.MinerConfirmationWindow())
					activationHeight := window - (node.height % window)
					log.Warnf("Unknown new rules are about to activate in "+
						"%d blocks (bit %d)", activationHeight, bit)
				}
			}
		}
	}

	return nil
}

// warnUnknownVersions logs a warning if a high enough percentage of the last
// blocks have unexpected versions.
//
// This function MUST be called with the chain state lock held (for writes)
func (b *BlockDAG) warnUnknownVersions(nodes []*blockNode) error {
	// Nothing to do if already warned.
	if b.unknownVersionsWarned {
		return nil
	}

	for _, node := range nodes {
		// Warn if enough previous blocks have unexpected versions.
		numUpgraded := uint32(0)
		parents := node.parents
		for i := uint32(0); i < unknownVerNumToCheck && parents != nil; i++ {
			expectedVersion, err := b.calcNextBlockVersion(parents)
			if err != nil {
				return err
			}
			if expectedVersion > vbLegacyBlockVersion &&
				(node.version & ^expectedVersion) != 0 {

				numUpgraded++
			}

			var newParents []*blockNode
			for _, parent := range parents {
				for _, grandParent := range parent.parents {
					newParents = append(newParents, grandParent)
				}
			}
			parents = newParents
		}
		if numUpgraded > unknownVerWarnNum {
			log.Warn("Unknown block versions are being mined, so new " +
				"rules might be in effect.  Are you running the " +
				"latest version of the software?")
			b.unknownVersionsWarned = true
		}
	}

	return nil
}
