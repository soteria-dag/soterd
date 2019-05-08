## Adding RPC Calls

Commit [e30eb31](https://github.com/soteria-dag/soterd/commit/e30eb319feb1d92bc286190633c47d288c9d7e9c) implemented the `getdagtips` RPC call, which you could refer to when making your own changes.

Commit [4e0d0b4](https://github.com/soteria-dag/soterd/commit/4e0d0b4c0c69ed3049d3cc66ef42068b3a9a7b71) shows how to add new entries to `rpcadapter.go`, to allow an RPC call to access information from other managers (like connection manager).

These are the places that need updating when working with RPC calls:

|Package|File|Why?|
|---|----|----|
|`main`|`rpcserver.go`|This is where you implement the server-side handler for the call (ex: `handleGetDAGTips`)|
|`main`|`rpcserverhelp.go`|Add help strings and result types for the RPC call|
|`rpcclient`|(depends on category of rpc call)|Add functions for sending RPC call, and unmarshalling the response into a result type.|
|`soterjson`|(depends on category of rpc call)|Add function for returning a new RPC call command type, and register it with the name of the RPC call (`MustRegisterCmd()`) in the `init()` function.|
|`soterjson`|(depends on category of rpc call) `*results.go`|Add a struct to represent the payload of the RPC call's response. Don't forget the json struct tags. This will be used to unmarshal the response into a result type that the RPC client can work with.|
|`integration`|`integration/rpcserver_test.go`|Add a test for your new RPC call. If your changes are modifying existing functionality or are meant to replace an existing call, you may also need to update `integration/rpctest/rpc_harness.go` and/or `integration/rpctest/rpc_harness_test.go`|
||`docs/json_rpc_api.md`|Documentation for the RPC call|

#### Notes for specific files

##### rpcserver.go

* You'll need to implement a handler for the RPC call, and ensure that there's a reference for it in the `rpcHandlersBeforeInit` map.

##### rpcserverhelp.go

* Add help strings to the `helpDescsEnUS` map
* Add an entry to the `rpcResultTypes` map

The integration tests are not normally run when `go test` is called, because they use [build tags (constraints)](https://golang.org/pkg/go/build/#hdr-Build_Constraints). The top of the file contains the current build tags, but at the time of writing the build tags for `integration/rpcserver_test.go` is `rpctest`. To run the integration tests for the rpc server, you can issue a `go test` command like this:
```shell
go test -v -tags rpctest "github.com/soteria-dag/soterd/integration"
```