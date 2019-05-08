peer
====

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

Package peer provides a common base for creating and managing network peers.

This package has intentionally been designed so it can be used as a standalone
package for any projects needing a full featured soter peer base to build on.

## Overview

This package builds upon the `wire` package, which provides the fundamental
primitives necessary to speak the soter wire protocol, in order to simplify
the process of creating fully functional peers.  In essence, it provides a
common base for creating concurrent safe fully validating nodes, Simplified
Payment Verification (SPV) nodes, proxies, etc.

A quick overview of the major features peer provides are as follows:

 - Provides a basic concurrent safe soter peer for handling soter
   communications via the peer-to-peer protocol
 - Full duplex reading and writing of soter protocol messages
 - Automatic handling of the initial handshake process including protocol
   version negotiation
 - Asynchronous message queueing of outbound messages with optional channel for
   notification when the message is actually sent
 - Flexible peer configuration
   - Caller is responsible for creating outgoing connections and listening for
     incoming connections so they have flexibility to establish connections as
     they see fit (proxies, etc)
   - User agent name and version
   - soter network
   - Service support signalling (full nodes, bloom filters, etc)
   - Maximum supported protocol version
   - Ability to register callbacks for handling soter protocol messages
 - Inventory message batching and send trickling with known inventory detection
   and avoidance
 - Automatic periodic keep-alive pinging and pong responses
 - Random nonce generation and self connection detection
 - Proper handling of bloom filter related commands when the caller does not
   specify the related flag to signal support
   - Disconnects the peer when the protocol version is high enough
   - Does not invoke the related callbacks for older protocol versions
 - Snapshottable peer statistics such as the total number of bytes read and
   written, the remote address, user agent, and negotiated protocol version
 - Helper functions pushing addresses, getblocks, getheaders, and reject
   messages
   - These could all be sent manually via the standard message output function,
     but the helpers provide additional nice functionality such as duplicate
     filtering and address randomization
 - Ability to wait for shutdown/disconnect
 - Comprehensive test coverage

## Peer vs RPC

While soterd implements soter p2p protocol and an RPC protocol with overlapping functionality, soterd **only** exchanges blocks with peers via the p2p protocol.

## Installation and Updating

```bash
$ go get -u github.com/soteria-dag/soterd/peer
```

## Examples

### New Outbound Peer Example  

This example demonstrates the basic process for initializing and creating an outbound peer. Peers negotiate by exchanging version and verack messages. For demonstration, a simple handler for the version message is attached to the peer.

```go
package main

import (
    "fmt"
    "net"
    "time"

    "github.com/soteria-dag/soterd/chaincfg"
    "github.com/soteria-dag/soterd/peer"
    "github.com/soteria-dag/soterd/wire"
)

// mockRemotePeer creates a basic inbound peer listening on the simnet port for
// use with Example_peerConnection.  It does not return until the listner is
// active.
func mockRemotePeer() error {
    // Configure peer to act as a simnet node that offers no services.
    peerCfg := &peer.Config{
        UserAgentName:    "peer",  // User agent name to advertise.
        UserAgentVersion: "1.0.0", // User agent version to advertise.
        ChainParams:      &chaincfg.SimNetParams,
        TrickleInterval:  time.Second * 10,
    }

    // Accept connections on the simnet port.
    listener, err := net.Listen("tcp", "127.0.0.1:18555")
    if err != nil {
        return err
    }
    go func() {
        conn, err := listener.Accept()
        if err != nil {
            fmt.Printf("Accept: error %v\n", err)
            return
        }

        // Create and start the inbound peer.
        p := peer.NewInboundPeer(peerCfg)
        p.AssociateConnection(conn)
    }()

    return nil
}

// This example demonstrates the basic process for initializing and creating an
// outbound peer.  Peers negotiate by exchanging version and verack messages.
// For demonstration, a simple handler for version message is attached to the
// peer.
func main() {
    // Ordinarily this will not be needed since the outbound peer will be
    // connecting to a remote peer, however, since this example is executed
    // and tested, a mock remote peer is needed to listen for the outbound
    // peer.
    if err := mockRemotePeer(); err != nil {
        fmt.Printf("mockRemotePeer: unexpected error %v\n", err)
        return
    }

    // Create an outbound peer that is configured to act as a simnet node
    // that offers no services and has listeners for the version and verack
    // messages.  The verack listener is used here to signal the code below
    // when the handshake has been finished by signalling a channel.
    verack := make(chan struct{})
    peerCfg := &peer.Config{
        UserAgentName:    "peer",  // User agent name to advertise.
        UserAgentVersion: "1.0.0", // User agent version to advertise.
        ChainParams:      &chaincfg.SimNetParams,
        Services:         0,
        TrickleInterval:  time.Second * 10,
        Listeners: peer.MessageListeners{
            OnVersion: func(p *peer.Peer, msg *wire.MsgVersion) {
                fmt.Println("outbound: received version")
            },
            OnVerAck: func(p *peer.Peer, msg *wire.MsgVerAck) {
                verack <- struct{}{}
            },
        },
    }
    p, err := peer.NewOutboundPeer(peerCfg, "127.0.0.1:18555")
    if err != nil {
        fmt.Printf("NewOutboundPeer: error %v\n", err)
        return
    }

    // Establish the connection to the peer address and mark it connected.
    conn, err := net.Dial("tcp", p.Addr())
    if err != nil {
        fmt.Printf("net.Dial: error %v\n", err)
        return
    }
    p.AssociateConnection(conn)

    // Wait for the verack message or timeout in case of failure.
    select {
    case <-verack:
    case <-time.After(time.Second * 1):
        fmt.Printf("Example_peerConnection: verack timeout")
    }

    // Disconnect the peer.
    p.Disconnect()
    p.WaitForDisconnect()

}
```

## License

Package peer is licensed under the [copyfree](http://copyfree.org) ISC License.
