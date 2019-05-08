blockdag package
===

Package `blockdag` implements soter block handling and chain selection rules.

The rules that soterd follows for validating and accepting blocks into the dag are
based on btcd and bitcoind. As development continues the these rules may deviate
further from btcd and bitcoind, since maintaining compatibility with bitcoin network
is not a design goal.

This package provides support for inserting new blocks into the block dag according
those rules. It includes functionality such as rejecting duplicate blocks, ensuring
blocks and transactions follow all rules, and orphan handling.

Since this package does not deal with other soter specifics such as network
communication or wallets, it provides a notification system which gives the
caller a high level of flexibility in how they want to react to certain events
such as orphan blocks which need their parents requested and newly connected
dag blocks which might result in wallet updates.

## Soter DAG Processing Overview

The general structure of the block DAG will be:
* A genesis block at height 0 (it has no parents, there are no other blocks at its same height/generation)
* One or more blocks connected to the genesis block (height 1)
* One or more blocks connected to those blocks

This results in a structure of blocks spanning away from the genesis block, with a set of blocks at each 
generation/height away from the genesis block. Connectivity between the blocks isn't as strict as with a blockchain; you
could have multiple parents, and have parents at different heights (within an upper limit).

The set of blocks without any blocks connected to them are called the DAG's "tips". These blocks are used as parents when
new blocks are mined, and are also referenced when nodes are syncing with one another.

Before a block is allowed into the block dag, it must go through a series
of validation rules. The following list serves as a general outline of
those rules to provide some intuition into what is going on under the hood, but
is by no means exhaustive:

 - Reject duplicate blocks
 - Perform a series of sanity checks on the block and its transactions such as
   verifying proof of work, timestamps, number and character of transactions,
   transaction amounts, script complexity, and merkle root calculations
 - Save the most recent orphan blocks for a limited time in case their parent
   blocks become available
 - Stop processing if the block is an orphan as the rest of the processing
   depends on the block's position within the block dag
 - Perform a series of more thorough checks that depend on the block's position
   within the block dag such as verifying block difficulties adhere to
   difficulty retarget rules, timestamps are after the median of the last
   several blocks, all transactions are finalized, checkpoint blocks match, and
   block versions are in line with the previous blocks
 - Determine how the block fits into the dag and perform different actions
   accordingly
 - When a block is being connected to the dag, perform further checks on the
   block's transactions such as verifying transaction duplicates, script
   complexity for the combination of connected scripts, coinbase maturity,
   double spends, and connected transaction values
 - Run the transaction scripts to verify the spender is allowed to spend the
   coins
 - Insert the block into the block database

## Errors

Errors returned by this package are either the raw errors provided by underlying
calls or of type blockdag.RuleError.  This allows the caller to differentiate
between unexpected errors, such as database errors, versus errors due to rule
violations through type assertions.  In addition, callers can programmatically
determine the specific rule violation by examining the ErrorCode field of the
type asserted blockdag.RuleError.

## Bitcoin Improvement Proposals

This package still defines BIPs like those listed below, but their behaviour
may be different under the DAG implementation. We may disable and remove BIPs
that aren't compatible or don't make sense to use with DAG.

		BIP0016 (https://en.bitcoin.it/wiki/BIP_0016)
		BIP0030 (https://en.bitcoin.it/wiki/BIP_0030)
		BIP0034 (https://en.bitcoin.it/wiki/BIP_0034)
