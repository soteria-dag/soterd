soterd mining methods
===

There are several different ways of triggering mining on a soterd node:

1. Using the `--generate` cli option when running soterd
2. Issuing a `generate` RPC call to a running soterd node
3. Issuing a `setgenerate` RPC call to a running soterd node


## 1. `--generate` cli option
When using `--generate`, soterd will [start the CPU miner](https://github.com/soteria-dag/soterd/blob/bb8a8e4211b4cd51d48b2c0c1ee3b130666151c6/server.go#L2290) with [as many workers as there are CPU cores](https://github.com/soteria-dag/soterd/blob/bb8a8e4211b4cd51d48b2c0c1ee3b130666151c6/mining/cpuminer/cpuminer.go#L48) on the system.

* Mining is asynchronous
* The `--generate` cli option and the setgenerate RPC call use the same code.


## 2. `generate` RPC call

The generate RPC call takes a _blocks_ parameter, indicating the number of blocks you want to generate.

* Mining is [synchronous](https://github.com/soteria-dag/soterd/blob/bb8a8e4211b4cd51d48b2c0c1ee3b130666151c6/mining/cpuminer/cpuminer.go#L575)
* [RPC call doesn't return until](https://github.com/soteria-dag/soterd/blob/bb8a8e4211b4cd51d48b2c0c1ee3b130666151c6/rpcserver.go#L918) _all blocks have been generated_


**NOTE**: At time of writing, the `CPUMiner.GenerateNBlocks` method used by the RPC call doesn't consider if a block has been accepted when building its list of blocks that were generated, so it's possible for this RPC call to generate blocks that won't show up in the dag if they were rejected due to not satisfying Proof-Of-Work limits, etc.


## 3. `setgenerate` RPC call
The [setgenerate RPC call](json_rpc_api.md#setgenerate) takes a _generate_ boolean parameter, and a _genproclimit_ numeric parameter with a default of `-1`.

* The _generate_ boolean parameter [starts block mining](https://github.com/soteria-dag/soterd/blob/bb8a8e4211b4cd51d48b2c0c1ee3b130666151c6/rpcserver.go#L3473) (if it hasn't been started already) when set to `true`, and [stops block mining](https://github.com/soteria-dag/soterd/blob/bb8a8e4211b4cd51d48b2c0c1ee3b130666151c6/rpcserver.go#L3460) if set to `false`.
* The _genproclimit_ numeric parameter sets the number of mining workers.
A value of `-1` (default) will [set the number of mining workers](https://github.com/soteria-dag/soterd/blob/bb8a8e4211b4cd51d48b2c0c1ee3b130666151c6/mining/cpuminer/cpuminer.go#L518) to the number of CPU cores on the system.
A value of `0` will [stop mining](https://github.com/soteria-dag/soterd/blob/bb8a8e4211b4cd51d48b2c0c1ee3b130666151c6/rpcserver.go#L3456).


* Mining is asynchronous
* `setgenerate` uses the same code as the `--generate` cli option.


**NOTE**: The `setgenerate` RPC call has been removed from Bitcoin Core 0.13.0