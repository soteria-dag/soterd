rpcclient
=========

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

rpcclient implements a Websocket-enabled soter JSON-RPC client package written
in [Go](http://golang.org/). It provides a client for interfacing with soterd's JSON-RPC API.


## Status

This package is currently under active development. It is already stable and the infrastructure is complete.

As development of soterd continues, you can expect that the API calls will drift further away from Bitcoin Core's JSON-RPC API. The soterd JSON-RPC API is not stable yet. 


## Documentation

* [API Reference](doc.go)
    * TODO: This will be updated to godoc.org link when soteria-dag is made public 
* [soterd Websockets Example](examples/soterdwebsockets/README.md)
  Connects to a soterd RPC server using TLS-secured websockets, registers for
  block connected and block disconnected notifications, and gets the current
  block count
* [soterwallet Websockets Example](examples/soterwalletwebsockets/README.md)
  Connects to a soterwallet RPC server using TLS-secured websockets, registers for
  notifications about changes to account balances, and gets a list of unspent
  transaction outputs (utxos) the wallet can sign
* [Bitcoin Core HTTP POST Example](examples/bitcoincorehttp/README.md)
  Connects to a bitcoin core RPC server using HTTP POST mode with TLS disabled
  and gets the current block count
* [Synchronous vs Asyncronous RPC calls](../docs/async_rpc_calls.md)

## Major Features

* Supports Websockets (soterd/soterwallet) and HTTP POST mode (bitcoin core)
* Provides callback and registration functions for soterd/soterwallet notifications
* Supports soterd extensions
* Translates to and from higher-level and easier to use Go types
* Offers a synchronous (blocking) and asynchronous API
* When running in Websockets mode (the default):
  * Automatic reconnect handling (can be disabled)
  * Outstanding commands are automatically reissued
  * Registered notifications are automatically reregistered
  * Back-off support on reconnect attempts

## Installation

```bash
$ go get -u github.com/soteria-dag/soterd/rpcclient
```

## License

Package rpcclient is licensed under the [copyfree](http://copyfree.org) ISC
License.
