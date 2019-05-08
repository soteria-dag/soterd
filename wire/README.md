wire
====

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)
=======

Package wire implements the soter wire protocol. A comprehensive suite of
tests with 100% test coverage is provided to ensure proper functionality.

This package has intentionally been designed so it can be used as a standalone
package for any projects needing to interface with soter peers at the wire
protocol level.

## Installation and Updating

```bash
$ go get -u github.com/soteria-dag/soterd/wire
```

## Soter Message Overview

The soter protocol consists of exchanging messages between peers. Each message
is preceded by a header which identifies information about it such as which
soter network it is a part of, its type, how big it is, and a checksum to
verify validity. All encoding and decoding of message headers is handled by this
package.

To accomplish this, there is a generic interface for soter messages named
`Message` which allows messages of any type to be read, written, or passed
around through channels, functions, etc. In addition, concrete implementations
of most of the currently supported soter messages are provided. For these
supported messages, all of the details of marshalling and unmarshalling to and
from the wire using soter encoding are handled so the caller doesn't have to
concern themselves with the specifics.

## Reading Messages Example

In order to unmarshal soter messages from the wire, use the `ReadMessage`
function. It accepts any `io.Reader`, but typically this will be a `net.Conn`
to a remote node running a soter peer.  Example syntax is:

```Go
	// Use the most recent protocol version supported by the package and the
	// main soter network.
	pver := wire.ProtocolVersion
	soternet := wire.MainNet

	// Reads and validates the next soter message from conn using the
	// protocol version pver and the soter network soternet.  The returns
	// are a wire.Message, a []byte which contains the unmarshalled
	// raw payload, and a possible error.
	msg, rawPayload, err := wire.ReadMessage(conn, pver, soternet)
	if err != nil {
		// Log and handle the error
	}
```

See the package documentation for details on determining the message type.

## Writing Messages Example

In order to marshal soter messages to the wire, use the `WriteMessage`
function. It accepts any `io.Writer`, but typically this will be a `net.Conn`
to a remote node running a soter peer. Example syntax to request addresses
from a remote peer is:

```Go
	// Use the most recent protocol version supported by the package and the
	// main soter network.
	pver := wire.ProtocolVersion
	soternet := wire.MainNet

	// Create a new getaddr soter message.
	msg := wire.NewMsgGetAddr()

	// Writes a soter message msg to conn using the protocol version
	// pver, and the soter network soternet.  The return is a possible
	// error.
	err := wire.WriteMessage(conn, msg, pver, soternet)
	if err != nil {
		// Log and handle the error
	}
```

## License

Package wire is licensed under the [copyfree](http://copyfree.org) ISC
License.
