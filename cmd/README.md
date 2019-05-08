cmd
===

This directory contains code for stand-alone utility programs that can help set-up and manage soterd. The utilities in `cmd` are why we include `./cmd/...` in the build/install instructions.
```
go install . ./cmd/...
``` 

Below is a description of each program.

## addblock

The `addblock` command provides functionality for loading blocks from a file into the block dag DB. The intended use of this is to help bootstrap a larger number of soterd nodes without those nodes having to spend as much time syncing blocks from peers.

For the regular bitcoin blockchain, bootstrapping was done by downloading a `bootstrap.dat` file with bittorrent, and using the `addblock` command to import the blocks from `bootstrap.dat` into the btcd node.

While it's possible to use `addblock` with soterd, there isn't yet a large-enough dag for users to consider out-of-band bootstrapping.


## soterctl

The `soterctl` command provides a CLI interface to the soterd RPC listener. It does this by using RPC command definitions from the [soterjson](../soterjson/README.md) package.

It helps users to examine the state of their soterd nodes, and to control soterd from outside of soterd itself.


## dagdemo

The [dagdemo](dagdemo/README.md) command spins up 4 nodes in simnet, has them mine 50 blocks each, graphs the resulting dag and renders it as an html file. This command has been _depreciated_ in favour of `dagviz`. Please use `dagviz` instead.


## dagviz

The [dagviz](dagviz/README.md) command performs similar tasks as `dagdemo` with more parameters and instead of creating a single resulting dag, it generates a series snapshots of dag at a configurable interval and renders them as a group html file.


## dagparam

The [dagparam](dagparam/README.md) command can be used to help explore the effects of different net params on block creation/propagation and dag structure. It renders test run results as an html file.


## findcheckpoint

The `findcheckpoint` command is intended for finding and displaying checkpoints in the chain. We've disabled checkpoint-related code, so this command currently has no use. 


## gencerts

The `gencerts` command can be used to generate self-signed SSL certificates, which could be used with `soterd` `--rpccert` and `--rpckey` options for securing access to the RPC server.

gencerts calls `soterutil.NewTLSCertPair`, which is also used by
* integration test suites, and
* the RPC listener setup when soterd starts (if no cert exists already)

So there won't normally be a need to run this command on its own.
