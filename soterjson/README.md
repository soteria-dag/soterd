soterjson
=======

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

Package soterjson implements concrete types for marshalling to and from the JSON-RPC API. A comprehensive suite of tests is provided to ensure proper functionality.

Although this package was primarily written for the soteria-dag, it has
intentionally been designed so it can be used as a standalone package for any
projects needing to marshal to and from bitcoin-like JSON-RPC requests and responses.

Note that although it's possible to use this package directly to implement an
RPC client, it is not recommended since it is only intended as an infrastructure
package.  Instead, RPC clients should use the
[rpcclient](https://github.com/soteria-dag/rpcclient) package which provides
a full blown RPC client with many features such as automatic connection
management, websocket support, automatic notification re-registration on
reconnect, and conversion from the raw underlying RPC types (strings, floats,
ints, etc) to higher-level types with many nice and useful properties.

## Installation and Updating

```bash
$ go get -u github.com/soteria-dag/soterd/soterjson
```

## Examples

### Marshal Command  

Demonstrates how to create and marshal a command into a JSON-RPC request.
```go
// Create a new getblock command.  Notice the nil parameter indicates
// to use the default parameter for that fields.  This is a common
// pattern used in all of the New<Foo>Cmd functions in this package for
// optional fields.  Also, notice the call to soterjson.Bool which is a
// convenience function for creating a pointer out of a primitive for
// optional parameters.
blockHash := "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f"
gbCmd := soterjson.NewGetBlockCmd(blockHash, soterjson.Bool(false), nil)

// Marshal the command to the format suitable for sending to the RPC
// server.  Typically the client would increment the id here which is
// request so the response can be identified.
id := 1
marshalledBytes, err := soterjson.MarshalCmd(id, gbCmd)
if err != nil {
    fmt.Println(err)
    return
}

// Display the marshalled command.  Ordinarily this would be sent across
// the wire to the RPC server, but for this example, just display it.
fmt.Printf("%s\n", marshalledBytes)
```

Output
```
{"jsonrpc":"1.0","method":"getblock","params":["000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f",false],"id":1}
```

### Unmarshal Command  

Demonstrates how to unmarshal a JSON-RPC request and then unmarshal the concrete request into a concrete command.
```go
// Ordinarily this would be read from the wire, but for this example,
// it is hard coded here for clarity.
data := []byte(`{"jsonrpc":"1.0","method":"getblock","params":["000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f",false],"id":1}`)

// Unmarshal the raw bytes from the wire into a JSON-RPC request.
var request soterjson.Request
if err := json.Unmarshal(data, &request); err != nil {
    fmt.Println(err)
    return
}

// Typically there isn't any need to examine the request fields directly
// like this as the caller already knows what response to expect based
// on the command it sent.  However, this is done here to demonstrate
// why the unmarshal process is two steps.
if request.ID == nil {
    fmt.Println("Unexpected notification")
    return
}
if request.Method != "getblock" {
    fmt.Println("Unexpected method")
    return
}

// Unmarshal the request into a concrete command.
cmd, err := soterjson.UnmarshalCmd(&request)
if err != nil {
    fmt.Println(err)
    return
}

// Type assert the command to the appropriate type.
gbCmd, ok := cmd.(*soterjson.GetBlockCmd)
if !ok {
    fmt.Printf("Incorrect command type: %T\n", cmd)
    return
}

// Display the fields in the concrete command.
fmt.Println("Hash:", gbCmd.Hash)
fmt.Println("Verbose:", *gbCmd.Verbose)
fmt.Println("VerboseTx:", *gbCmd.VerboseTx)
```

Output
```
Hash: 000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f
Verbose: false
VerboseTx: false
```

### Marshal Response  

Demonstrates how to marshal a JSON-RPC response.
```go
// Marshal a new JSON-RPC response.  For example, this is a response
// to a getblockheight request.
marshalledBytes, err := soterjson.MarshalResponse(1, 350001, nil)
if err != nil {
    fmt.Println(err)
    return
}

// Display the marshalled response.  Ordinarily this would be sent
// across the wire to the RPC client, but for this example, just display
// it.
fmt.Printf("%s\n", marshalledBytes)
```

Output
```
{"result":350001,"error":null,"id":1}

```

### Unmarshal Response  

Demonstrates how to unmarshal a JSON-RPC response and then unmarshal the result field in the response to a concrete type.

```go
// Ordinarily this would be read from the wire, but for this example,
// it is hard coded here for clarity.  This is an example response to a
// getblockheight request.
data := []byte(`{"result":350001,"error":null,"id":1}`)

// Unmarshal the raw bytes from the wire into a JSON-RPC response.
var response soterjson.Response
if err := json.Unmarshal(data, &response); err != nil {
    fmt.Println("Malformed JSON-RPC response:", err)
    return
}

// Check the response for an error from the server.  For example, the
// server might return an error if an invalid/unknown block hash is
// requested.
if response.Error != nil {
    fmt.Println(response.Error)
    return
}

// Unmarshal the result into the expected type for the response.
var blockHeight int32
if err := json.Unmarshal(response.Result, &blockHeight); err != nil {
    fmt.Printf("Unexpected result type: %T\n", response.Result)
    return
}
fmt.Println("Block height:", blockHeight)
```

Output
```
Block height: 350001
```


## License

Package soterjson is licensed under the [copyfree](http://copyfree.org) ISC
License.
