// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdag

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"sort"
	"time"

	qithash "github.com/Qitmeer/qitmeer-lib/common/hash"
	"github.com/soteria-dag/soterd/chaincfg/chainhash"
	"github.com/soteria-dag/soterd/wire"
)

var (
	// bigOne is 1 represented as a big.Int.  It is defined here to avoid
	// the overhead of creating it multiple times.
	bigOne = big.NewInt(1)

	// oneLsh256 is 1 shifted left 256 bits.
	// This is equivalent to 2^256, like:
	// oneLsh256 := big.NewInt(0).Exp(big.NewInt(2), big.NewInt(256), nil)
	//
	// It is defined here to avoid the overhead of creating it multiple times.
	oneLsh256 = new(big.Int).Lsh(bigOne, 256)

	// How many generations of blocks we will look into the past of from a block,
	// when calculating the target difficulty for a new block.
	difficultyGenerations int32 = 23

	// How frequently we want blocks to be generated on average
	targetBlockGenRate = time.Minute

	// How much to adjust the difficulty, to attempt to match the targetBlockGenRate in future blocks
	difficultyChangePct float64 = 10
)

// HashToBig converts a chainhash.Hash into a big.Int that can be used to
// perform math comparisons.
func HashToBig(hash *chainhash.Hash) *big.Int {
	// A Hash is in little-endian, but the big package wants the bytes in
	// big-endian, so reverse them.
	buf := *hash
	blen := len(buf)
	for i := 0; i < blen/2; i++ {
		buf[i], buf[blen-1-i] = buf[blen-1-i], buf[i]
	}

	return new(big.Int).SetBytes(buf[:])
}

// CompactToBig converts a compact representation of a whole number N to an
// unsigned 32-bit number.  The representation is similar to IEEE754 floating
// point numbers.
//
// Like IEEE754 floating point, there are three basic components: the sign,
// the exponent, and the mantissa.  They are broken out as follows:
//
//	* the most significant 8 bits represent the unsigned base 256 exponent
// 	* bit 23 (the 24th bit) represents the sign bit
//	* the least significant 23 bits represent the mantissa
//
//	-------------------------------------------------
//	|   Exponent     |    Sign    |    Mantissa     |
//	-------------------------------------------------
//	| 8 bits [31-24] | 1 bit [23] | 23 bits [22-00] |
//	-------------------------------------------------
//
// The formula to calculate N is:
// 	N = (-1^sign) * mantissa * 256^(exponent-3)
//
// This compact form is only used in soter to encode unsigned 256-bit numbers
// which represent difficulty targets, thus there really is not a need for a
// sign bit, but it is implemented here to stay consistent with bitcoind.
func CompactToBig(compact uint32) *big.Int {
	// Extract the mantissa, sign bit, and exponent.
	mantissa := compact & 0x007fffff
	isNegative := compact&0x00800000 != 0
	exponent := uint(compact >> 24)

	// Since the base for the exponent is 256, the exponent can be treated
	// as the number of bytes to represent the full 256-bit number.  So,
	// treat the exponent as the number of bytes and shift the mantissa
	// right or left accordingly.  This is equivalent to:
	// N = mantissa * 256^(exponent-3)
	var bn *big.Int
	if exponent <= 3 {
		mantissa >>= 8 * (3 - exponent)
		bn = big.NewInt(int64(mantissa))
	} else {
		bn = big.NewInt(int64(mantissa))
		bn.Lsh(bn, 8*(exponent-3))
	}

	// Make it negative if the sign bit is set.
	if isNegative {
		bn = bn.Neg(bn)
	}

	return bn
}

// BigToCompact converts a whole number N to a compact representation using
// an unsigned 32-bit number.  The compact representation only provides 23 bits
// of precision, so values larger than (2^23 - 1) only encode the most
// significant digits of the number.  See CompactToBig for details.
func BigToCompact(n *big.Int) uint32 {
	// No need to do any work if it's zero.
	if n.Sign() == 0 {
		return 0
	}

	// Since the base for the exponent is 256, the exponent can be treated
	// as the number of bytes.  So, shift the number right or left
	// accordingly.  This is equivalent to:
	// mantissa = mantissa / 256^(exponent-3)
	var mantissa uint32
	exponent := uint(len(n.Bytes()))
	if exponent <= 3 {
		mantissa = uint32(n.Bits()[0])
		mantissa <<= 8 * (3 - exponent)
	} else {
		// Use a copy to avoid modifying the caller's original number.
		tn := new(big.Int).Set(n)
		mantissa = uint32(tn.Rsh(tn, 8*(exponent-3)).Bits()[0])
	}

	// When the mantissa already has the sign bit set, the number is too
	// large to fit into the available 23-bits, so divide the number by 256
	// and increment the exponent accordingly.
	if mantissa&0x00800000 != 0 {
		mantissa >>= 8
		exponent++
	}

	// Pack the exponent, sign bit, and mantissa into an unsigned 32-bit
	// int and return it.
	compact := uint32(exponent<<24) | mantissa
	if n.Sign() < 0 {
		compact |= 0x00800000
	}
	return compact
}

// CalcWork calculates a work value from difficulty bits.  Bitcoin increases
// the difficulty for generating a block by decreasing the value which the
// generated hash must be less than.  This difficulty target is stored in each
// block header using a compact representation as described in the documentation
// for CompactToBig.  The main chain is selected by choosing the chain that has
// the most proof of work (highest difficulty).  Since a lower target difficulty
// value equates to higher actual difficulty, the work value which will be
// accumulated must be the inverse of the difficulty.  Also, in order to avoid
// potential division by zero and really small floating point numbers, the
// result adds 1 to the denominator and multiplies the numerator by 2^256.
func CalcWork(bits uint32) *big.Int {
	// Return a work value of zero if the passed difficulty bits represent
	// a negative number. Note this should not happen in practice with valid
	// blocks, but an invalid block could trigger it.
	difficultyNum := CompactToBig(bits)
	if difficultyNum.Sign() <= 0 {
		return big.NewInt(0)
	}

	// (1 << 256) / (difficultyNum + 1)
	denominator := new(big.Int).Add(difficultyNum, bigOne)
	return new(big.Int).Div(oneLsh256, denominator)
}

// calcEasiestDifficulty calculates the easiest possible difficulty that a block
// can have given starting difficulty bits and a duration.  It is mainly used to
// verify that claimed proof of work by a block is sane as compared to a
// known good checkpoint.
func (b *BlockDAG) calcEasiestDifficulty(bits uint32, duration time.Duration) uint32 {
	// Convert types used in the calculations below.
	durationVal := int64(duration / time.Second)
	adjustmentFactor := big.NewInt(b.chainParams.RetargetAdjustmentFactor)

	// The test network rules allow minimum difficulty blocks after more
	// than twice the desired amount of time needed to generate a block has
	// elapsed.
	if b.chainParams.ReduceMinDifficulty {
		reductionTime := int64(b.chainParams.MinDiffReductionTime /
			time.Second)
		if durationVal > reductionTime {
			return b.chainParams.PowLimitBits
		}
	}

	// Since easier difficulty equates to higher numbers, the easiest
	// difficulty for a given duration is the largest value possible given
	// the number of retargets for the duration and starting difficulty
	// multiplied by the max adjustment factor.
	newTarget := CompactToBig(bits)
	for durationVal > 0 && newTarget.Cmp(b.chainParams.PowLimit) < 0 {
		newTarget.Mul(newTarget, adjustmentFactor)
		durationVal -= b.maxRetargetTimespan
	}

	// Limit new value to the proof of work limit.
	if newTarget.Cmp(b.chainParams.PowLimit) > 0 {
		newTarget.Set(b.chainParams.PowLimit)
	}

	return BigToCompact(newTarget)
}

// findPrevTestNetDifficulty returns the difficulty of the previous block which
// did not have the special testnet minimum difficulty rule applied.
//
// This function MUST be called with the chain state lock held (for writes).
func (b *BlockDAG) findPrevTestNetDifficulty(startNode *blockNode) uint32 {
	// Search backwards through the chain for the last block without
	// the special rule applied.
	iterNode := startNode
	for iterNode != nil && int64(iterNode.height) % b.blocksPerRetarget != 0 &&
		iterNode.bits == b.chainParams.PowLimitBits {

		parents := iterNode.parents
		iterNode = parents[0] //TODO: recheck logic
	}

	// Return the found difficulty or the minimum difficulty if no
	// appropriate block was found.
	lastBits := b.chainParams.PowLimitBits
	if iterNode != nil {
		lastBits = iterNode.bits
	}
	return lastBits
}

// calcNextRequiredDifficulty calculates the required difficulty for the block
// after the passed previous block node based on the difficulty retarget rules.
// This function differs from the exported CalcNextRequiredDifficulty in that
// the exported version uses the current best chain as the previous block node
// while this function accepts any block node.
func (b *BlockDAG) calcNextRequiredDifficulty(lastNode *blockNode, newBlockTime time.Time) (uint32, error) {
	// Genesis block.
	if lastNode == nil {
		return b.chainParams.PowLimitBits, nil
	}

	// Return the previous block's difficulty requirements if this block
	// is not at a difficulty retarget interval.
	if int64(lastNode.height+1) % b.blocksPerRetarget != 0 {
		// For networks that support it, allow special reduction of the
		// required difficulty once too much time has elapsed without
		// mining a block.
		if b.chainParams.ReduceMinDifficulty {
			// Return minimum difficulty when more than the desired
			// amount of time has elapsed without mining a block.
			reductionTime := int64(b.chainParams.MinDiffReductionTime /
				time.Second)
			allowMinTime := lastNode.timestamp + reductionTime
			if newBlockTime.Unix() > allowMinTime {
				return b.chainParams.PowLimitBits, nil
			}

			// The block was mined within the desired timeframe, so
			// return the difficulty for the last block which did
			// not have the special minimum difficulty rule applied.
			return b.findPrevTestNetDifficulty(lastNode), nil
		}

		// For the main network (or any unrecognized networks), simply
		// return the previous block's difficulty requirements.
		return lastNode.bits, nil
	}

	// Get the block node at the previous retarget (targetTimespan days
	// worth of blocks).
	prevRetarget := int64(lastNode.height) - b.blocksPerRetarget + int64(1)
	ancestorNodes := b.dView.NodesByHeight(int32(prevRetarget))
	//firstNode := lastNode.RelativeAncestor(b.blocksPerRetarget - 1)
	if ancestorNodes == nil {
		return 0, AssertError("unable to obtain previous retarget block")
	}
	firstNode := ancestorNodes[0] //TODO: recheck logic

	// Limit the amount of adjustment that can occur to the previous
	// difficulty.
	actualTimespan := lastNode.timestamp - firstNode.timestamp
	adjustedTimespan := actualTimespan
	if actualTimespan < b.minRetargetTimespan {
		adjustedTimespan = b.minRetargetTimespan
	} else if actualTimespan > b.maxRetargetTimespan {
		adjustedTimespan = b.maxRetargetTimespan
	}

	// Calculate new target difficulty as:
	//  currentDifficulty * (adjustedTimespan / targetTimespan)
	// The result uses integer division which means it will be slightly
	// rounded down.  Bitcoind also uses integer division to calculate this
	// result.
	oldTarget := CompactToBig(lastNode.bits)
	newTarget := new(big.Int).Mul(oldTarget, big.NewInt(adjustedTimespan))
	targetTimeSpan := int64(b.chainParams.TargetTimespan / time.Millisecond)
	newTarget.Div(newTarget, big.NewInt(targetTimeSpan))

	// Limit new value to the proof of work limit.
	if newTarget.Cmp(b.chainParams.PowLimit) > 0 {
		newTarget.Set(b.chainParams.PowLimit)
	}

	// Log new target difficulty and return it.  The new target logging is
	// intentionally converting the bits back to a number instead of using
	// newTarget since conversion to the compact representation loses
	// precision.
	newTargetBits := BigToCompact(newTarget)
	log.Debugf("Difficulty retarget at block height %d", lastNode.height+1)
	log.Debugf("Old target %08x (%064x)", lastNode.bits, oldTarget)
	log.Debugf("New target %08x (%064x)", newTargetBits, CompactToBig(newTargetBits))
	log.Debugf("Actual timespan %v, adjusted timespan %v, target timespan %v",
		time.Duration(actualTimespan)*time.Second,
		time.Duration(adjustedTimespan)*time.Second,
		b.chainParams.TargetTimespan)

	return newTargetBits, nil
}

// CalcNextRequiredDifficulty calculates the required difficulty for the block
// after the end of the current best chain based on the difficulty retarget
// rules.
//
// This function is safe for concurrent access.
func (b *BlockDAG) CalcNextRequiredDifficulty(timestamp time.Time) (uint32, error) {
	b.chainLock.Lock()
	tips := b.dView.Tips()

	// find most recent tip
	var recentTip *blockNode = nil
	var recentTipTime = time.Time{}.Unix()
	for _, prevNode := range tips {
		if prevNode.timestamp > recentTipTime {
			recentTip = prevNode
			recentTipTime = prevNode.timestamp
		}
	}

	difficulty, err := b.calcNextRequiredDifficulty(recentTip, timestamp)
	b.chainLock.Unlock()
	return difficulty, err
}

// scaleDifficulty scales the difficulty value
func scaleDifficulty(d *big.Int) *big.Int {
	scaled := big.NewInt(0).Div(oneLsh256, d)
	if scaled.Cmp(bigOne) < 0 {
		// Scaled difficulty can't be less than 1
		return bigOne
	}
	return scaled
}

// ProofDifficulty returns the difficulty value of the cuckoo cycle proof.
// The proof difficulty is defined as the maximum difficulty of 2^256 divided by
// the double-sha256 digest of the cycle nonces.
func ProofDifficulty(cycleNonces []uint32) *big.Int {
	var hashInt big.Int
	cycleNoncesHash := qithash.DoubleHashB(Uint32ToBytes(cycleNonces))
	hashInt.SetBytes(cycleNoncesHash[:])

	return scaleDifficulty(&hashInt)
}

// TargetDifficulty returns
// the target difficulty for the block's cuckoo cycle, and
// if there was an error while determining target difficulty.
//
// A solution is valid if its proof difficulty is higher than the returned target difficulty.
func (b *BlockDAG) TargetDifficulty(blockHeight int32) (*big.Int, error) {
	if blockHeight == 0 {
		// There are no other blocks before this, so provide an initial difficulty
		return bigOne, nil
	}

	var lowHeight int32
	if lowHeight - difficultyGenerations < 0 {
		// Can't look before genesis block
		lowHeight = 0
	}

	// Retrieve difficulty and timestamps from blocks at heights from blockHeight to blockHeight - difficultyGenerations.
	// These values are used to determine target difficulty for a block at the given blockHeight.
	var byHeight = make([][]*wire.MsgBlock, 0)
	for i := blockHeight; i >= lowHeight; i-- {
		hashes, err := b.BlockHashesByHeight(blockHeight)
		if err != nil {
			return nil, err
		}

		var blocks = make([]*wire.MsgBlock, 0)
		for _, hash := range hashes {
			block, err := b.BlockByHash(&hash)
			if err != nil {
				return nil, err
			}

			blocks = append(blocks, block.MsgBlock())
		}

		byHeight = append(byHeight, blocks)
	}

	if len(byHeight) == 0 {
		err := fmt.Errorf("No blocks found between %d and %d", blockHeight, lowHeight)
		return nil, err
	}

	// Define a function that converts the block's difficulty from compact form
	var blockDifficulty = func(block *wire.MsgBlock) *big.Int {
		return CompactToBig(block.Header.Bits)
	}

	// The initial target difficulty will be the median difficulty of the blocks at the median of the heights.
	var targetDifficulty *big.Int
	blocks := byHeight[len(byHeight) / 2]
	var sorted = make([]*wire.MsgBlock, len(blocks))
	for i, bd := range blocks {
		sorted[i] = bd
	}
	sort.Slice(sorted, func(i, j int) bool {
		iDiff := blockDifficulty(sorted[i])
		jDiff := blockDifficulty(sorted[j])
		return iDiff.Cmp(jDiff) < 0
	})
	targetDifficulty = blockDifficulty(sorted[len(sorted) / 2])

	// The target difficulty will be increased if the time between new generations of blocks is lower than
	// our target block-generation rate, for the majority of generations in our sample.
	// It'll be decreased if the time between new generations of blocks is greater than our target block-generation rate.
	if len(byHeight) == 1 {
		// No reason to adjust target difficulty if there's only one generation of blocks.
		return BigIntMax(targetDifficulty, bigOne), nil
	}

	// Define a function that can return a sorted slice of block timestamps.
	var blockTimes = func(blocks []*wire.MsgBlock) []time.Time {
		times := make([]time.Time, len(blocks))
		for i, b := range blocks {
			times[i] = b.Header.Timestamp
		}
		sort.Sort(ByTime(times))
		return times
	}

	// Track how many generations of blocks were solved either too slow or too fast, for our target block-generation rate.
	var tooFast, tooSlow int32
	for heightOffset, blocks := range byHeight {
		if heightOffset == len(byHeight) - 1 {
			continue
		}

		times := blockTimes(blocks)
		median := times[len(times) / 2]

		nextTimes := blockTimes(byHeight[heightOffset + 1])
		nextMedian := nextTimes[len(nextTimes) / 2]

		delta := median.Sub(nextMedian)
		if delta > targetBlockGenRate {
			tooFast += 1
		} else if delta < targetBlockGenRate {
			tooSlow += 1
		}
	}

	// Determine by how much we'd adjust the target difficulty
	pct := big.NewFloat(0).Quo(
		big.NewFloat(difficultyChangePct),
		big.NewFloat(100))

	var changeAmount = big.NewInt(0)
	big.NewFloat(0).Mul(
		big.NewFloat(0).SetInt(targetDifficulty),
		pct).Int(changeAmount)
	if changeAmount.Cmp(bigOne) < 0 {
		// Minimum difficulty change of +/- 1
		changeAmount = big.NewInt(0).Set(bigOne)
	}

	// Adjust target difficulty if necessary
	if tooFast > difficultyGenerations / 2 {
		// Majority of block generations were solved too quickly. We need to increase the target difficulty
		targetDifficulty = big.NewInt(0).Add(targetDifficulty, changeAmount)
	} else if tooSlow > difficultyGenerations / 2 {
		// Majority of block generations solved too slowly. We need to decrease the target difficulty
		targetDifficulty = big.NewInt(0).Sub(targetDifficulty, changeAmount)
	}

	// Return a minimum target difficulty of 1
	return BigIntMax(targetDifficulty, bigOne), nil
}

// Uint32ToBytes converts a slice of uint32 into a big.Int
func Uint32ToBytes(v []uint32) []byte {
	var buf = make([]byte, 4*len(v))
	for i, x := range v {
		binary.LittleEndian.PutUint32(buf[4*i:], x)
	}
	return buf
}

// BigIntMax returns the larger of the two big.Int types
func BigIntMax(a, b *big.Int) *big.Int {
	if a.Cmp(b) > 0 {
		return a
	}

	return b
}