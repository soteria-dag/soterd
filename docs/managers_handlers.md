## Managers and Handlers

Some functionality in soterd is handled by "managers". These are goroutines that communicate with the main process and other managers through channels. Each with a specific purpose.

### Address manager

Maintains a pool of peer addresses that nodes could connect to. It is started by the peer handler.

For more info, [see the addrmgr README](../addrmgr/README.md) or godoc documentation.


### Connection manager

Handles creating and accepting connection requests, for nodes participating in P2P network. It is started by the peer handler.

For more info, [see the connmgr README](../connmgr/README.md).


### Metrics manager

Metrics manager handles the receiving metrics from other managers, and providing metrics to other parts of code through its methods.

The intent of the metrics manager was to expose run-time stats from various other managers to external systems via RPC calls. 

Currently the metrics manager is used to expose block-mining metrics to external systems via the `getblockmetrics` RPC call.  

For more info, [see the metrics README](../metrics/README.md).

### Miner

The miner handles spawning goroutines to generate, solve, and submit blocks. It's unique in that it can run as a manager or be used on an as-needed basis.

It's started [under a few conditions](mining_methods.md):
1. When soterd's configuration indicates that the miner should be started. (the `--generate` parameter specified with the `--miningaddr=` parameter)
2. When soterd is asked to generate blocks, via the `setgenerate` RPC command (which was removed in [Bitcoin core 0.13.0](https://bitcoin.org/en/release/v0.13.0))

Block mining can also be triggered with the `generate` RPC command, but this is different in that it doesn't start a mining worker controller to oversee multiple mining workers (goroutines); Blocks are mined sequentially with 1 mining-worker, and hashes are returned after mining has completed for all blocks.


### Peer handler

The peer handler takes care of operations like:
* Adding and removing peers
* Relaying and broadcasting messages to peers

The peer handler is also responsible for starting and stopping the:
* Address manager
* Connection manager
* Sync manager

By starting the connection manager, `inHandler` goroutines are started for each new peer, which dispatches peer messages to the appropriate function call or method to handle the message. 

#### Where functionality is implemented

The peer-related types and much of the related methods are defined in the `github.com/soteria-dag/soterd/peer` package, but the implementation of the listeners for supported messages are in `server.go`. 

For more information on the `peer` package, see the [peer package README](../peer/README.md) or godoc file.

### Sync manager

Handles block synchronization between this node and the peers it is connected to.

It is started by the peer manager, and also receives notifications from the miner (when new blocks are generated).

When started, the `blockHandler` goroutine is started, which dispatches messages to the appropriate function call or method to handle the message.

For block synchronization, the two most important message handlers are `handleInvMsg` and `handleBlockMsg`. The block synchronization process will involve a sequence of messages sent to and received by peers which will alternate flow between these two handlers. When altering the behaviour of one of them it's important to consider the effect it would have on the other.

For more information on the block synchronization flow, see the [netsync package README](../netsync/README.md) or godoc file. 