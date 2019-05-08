## Soterd startup

##### soterd.go
* Flow control starts in `soterd.go -> main()`
* `soterdMain()` is called
* A new `server` type is created and `Start()` method is called on it. This passes flow to `server.go`

##### server.go
`server.Start()`
* Starts [peerHandler](managers_handlers.md)
    * Starts [address manager](managers_handlers.md)
    * Starts [connection manager](managers_handlers.md)
    * Starts [sync manager](managers_handlers.md)
* Starts [miner](managers_handlers.md) (when the `--generate` parameter specified with the `--miningaddr=` parameter)

From this point, if the node's dag is in need of updating, the sync manager will initiate block transfer from a peer with a higher height than ours.

If started, the miner will start generating blocks after it's connected to at least one peer on the network (so that it can advertise blocks).