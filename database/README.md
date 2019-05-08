database
========

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

Package database provides a block and metadata storage database.

Please note that this package is intended to enable soterd to support different
database backends and is not something that a client can directly access as only
one entity can have the database open at a time (for most database backends),
and that entity will be soterd.

When a client wants programmatic access to the data provided by soterd, they'll
likely want to use the [rpcclient](https://github.com/soteria-dag/soterd/tree/master/rpcclient)
package which makes use of the [JSON-RPC API](https://github.com/soteria-dag/soterd/tree/master/docs/json_rpc_api.md).

However, this package could be extremely useful for any applications requiring
Bitcoin block storage capabilities.

The default backend, ffldb, has a strong focus on speed, efficiency, and
robustness.  It makes use of leveldb for the metadata, flat files for block
storage, and strict checksums in key areas to ensure data integrity.

## Feature Overview

- Key/value metadata store
- Bitcoin block storage
- Efficient retrieval of block headers and regions (transactions, scripts, etc)
- Read-only and read-write transactions with both manual and managed modes
- Nested buckets
- Iteration support including cursors with seek capability
- Supports registration of backend databases
- Comprehensive test coverage

## Database

The main entry point is the DB interface.  It exposes functionality for
transactional-based access and storage of metadata and block data.  It is
obtained via the Create and Open functions which take a database type string
that identifies the specific database driver (backend) to use as well as
arguments specific to the specified driver.

The interface provides facilities for obtaining transactions (the Tx interface)
that are the basis of all database reads and writes.  Unlike some database
interfaces that support reading and writing without transactions, this interface
requires transactions even when only reading or writing a single key.

The Begin function provides an unmanaged transaction while the View and Update
functions provide a managed transaction.  These are described in more detail
below.

## Transactions

The Tx interface provides facilities for rolling back or committing changes that
took place while the transaction was active.  It also provides the root metadata
bucket under which all keys, values, and nested buckets are stored.  A
transaction can either be read-only or read-write and managed or unmanaged.

### Managed versus Unmanaged Transactions

A managed transaction is one where the caller provides a function to execute
within the context of the transaction and the commit or rollback is handled
automatically depending on whether or not the provided function returns an
error.  Attempting to manually call Rollback or Commit on the managed
transaction will result in a panic.

An unmanaged transaction, on the other hand, requires the caller to manually
call Commit or Rollback when they are finished with it.  Leaving transactions
open for long periods of time can have several adverse effects, so it is
recommended that managed transactions are used instead.

## Buckets

The Bucket interface provides the ability to manipulate key/value pairs and
nested buckets as well as iterate through them.

The Get, Put, and Delete functions work with key/value pairs, while the Bucket,
CreateBucket, CreateBucketIfNotExists, and DeleteBucket functions work with
buckets.  The ForEach function allows the caller to provide a function to be
called with each key/value pair and nested bucket in the current bucket.

### Metadata Bucket

As discussed above, all of the functions which are used to manipulate key/value
pairs and nested buckets exist on the Bucket interface.  The root metadata
bucket is the upper-most bucket in which data is stored and is created at the
same time as the database.  Use the Metadata function on the Tx interface
to retrieve it.

### Nested Buckets

The CreateBucket and CreateBucketIfNotExists functions on the Bucket interface
provide the ability to create an arbitrary number of nested buckets.  It is
a good idea to avoid a lot of buckets with little data in them as it could lead
to poor page utilization depending on the specific driver in use.

## Test coverage

The `testdata` directory contains the `dagblocks1-256.bz2` bzipped file of blocks, which is used in ffldb tests. This file's contents were generated from btcd's original `blocks1-256.bz2` file of blocks, converted from blockchain to blockdag.    


## Installation and Updating

```bash
$ go get -u github.com/soteria-dag/soterd/database
```

## Examples

These examples will be referred to via godoc.org, once the soterd package is publicly published. The code in the examples in this document comes from [example](https://golang.org/pkg/testing/#hdr-Examples) functions in `example_test.go` (which is what godoc tooling would pull the examples from) 

### Basic Usage Example

Demonstrates creating a new database and using a managed read-write transaction to store and retrieve metadata.

```go
// This example assumes the ffldb driver is imported.
//
// import (
// 	"github.com/soteria-dag/soterd/database"
// 	_ "github.com/soteria-dag/soterd/database/ffldb"
// )

// Create a database and schedule it to be closed and removed on exit.
// Typically you wouldn't want to remove the database right away like
// this, nor put it in the temp directory, but it's done here to ensure
// the example cleans up after itself.
dbPath := filepath.Join(os.TempDir(), "exampleusage")
db, err := database.Create("ffldb", dbPath, wire.MainNet)
if err != nil {
    fmt.Println(err)
    return
}
defer os.RemoveAll(dbPath)
defer db.Close()

// Use the Update function of the database to perform a managed
// read-write transaction.  The transaction will automatically be rolled
// back if the supplied inner function returns a non-nil error.
err = db.Update(func(tx database.Tx) error {
    // Store a key/value pair directly in the metadata bucket.
    // Typically a nested bucket would be used for a given feature,
    // but this example is using the metadata bucket directly for
    // simplicity.
    key := []byte("mykey")
    value := []byte("myvalue")
    if err := tx.Metadata().Put(key, value); err != nil {
        return err
    }

    // Read the key back and ensure it matches.
    if !bytes.Equal(tx.Metadata().Get(key), value) {
        return fmt.Errorf("unexpected value for key '%s'", key)
    }

    // Create a new nested bucket under the metadata bucket.
    nestedBucketKey := []byte("mybucket")
    nestedBucket, err := tx.Metadata().CreateBucket(nestedBucketKey)
    if err != nil {
        return err
    }

    // The key from above that was set in the metadata bucket does
    // not exist in this new nested bucket.
    if nestedBucket.Get(key) != nil {
        return fmt.Errorf("key '%s' is not expected nil", key)
    }

    return nil
})
if err != nil {
    fmt.Println(err)
    return
}
```

### Block Storage and Retrieval Example
  
Demonstrates creating a new database, using a managed read-write transaction to store a block, and then using a managed read-only transaction to fetch the block.

```go
// This example assumes the ffldb driver is imported.
//
// import (
// 	"github.com/btcsuite/btcd/database"
// 	_ "github.com/btcsuite/btcd/database/ffldb"
// )

// Create a database and schedule it to be closed and removed on exit.
// Typically you wouldn't want to remove the database right away like
// this, nor put it in the temp directory, but it's done here to ensure
// the example cleans up after itself.
dbPath := filepath.Join(os.TempDir(), "exampleblkstorage")
db, err := database.Create("ffldb", dbPath, wire.MainNet)
if err != nil {
    fmt.Println(err)
    return
}
defer os.RemoveAll(dbPath)
defer db.Close()

// Use the Update function of the database to perform a managed
// read-write transaction and store a genesis block in the database as
// and example.
err = db.Update(func(tx database.Tx) error {
    genesisBlock := chaincfg.MainNetParams.GenesisBlock
    return tx.StoreBlock(btcutil.NewBlock(genesisBlock))
})
if err != nil {
    fmt.Println(err)
    return
}

// Use the View function of the database to perform a managed read-only
// transaction and fetch the block stored above.
var loadedBlockBytes []byte
err = db.Update(func(tx database.Tx) error {
    genesisHash := chaincfg.MainNetParams.GenesisHash
    blockBytes, err := tx.FetchBlock(genesisHash)
    if err != nil {
        return err
    }

    // As documented, all data fetched from the database is only
    // valid during a database transaction in order to support
    // zero-copy backends.  Thus, make a copy of the data so it
    // can be used outside of the transaction.
    loadedBlockBytes = make([]byte, len(blockBytes))
    copy(loadedBlockBytes, blockBytes)
    return nil
})
if err != nil {
    fmt.Println(err)
    return
}

// Typically at this point, the block could be deserialized via the
// wire.MsgBlock.Deserialize function or used in its serialized form
// depending on need.  However, for this example, just display the
// number of serialized bytes to show it was loaded as expected.
fmt.Printf("Serialized block size: %d bytes\n", len(loadedBlockBytes))
```

## License

Package database is licensed under the [copyfree](http://copyfree.org) ISC
License.
