## Updating P2P Wire Protocol

Soterd defines its own wire protocol for p2p communication.

In soterd the wire protocol is used for _both_ on-disk storage **and** network transmission of blocks.

This wire protocol only serializes the expected data, so if you want to make a change to the structure of a message, there are a number of places that need updating so that expectations aren't broken. In general these are:

* (`wire` package) Serialization/deserialization functions
* (`wire` package) New cases in `readElement`, `writeElement`, to support serialization/deserialization of types not currently matched.
* Tests for the above

If modifying the structure of an existing message, this will mean updating the expected bytes in related test suites. This could mean updating:
* Type initialization 
* Sequence of hard-coded byte arrays as [golang integer literals](https://golang.org/ref/spec#Integer_literals)
* Hex-encoded strings representing raw bytes
* Size or offset values
* [golang test example code verification](https://golang.org/pkg/testing/#hdr-Examples) (comment containing the expected number of bytes)
* Files containing pre-generated data that's used as test data in multiple tests
 
Updating pre-generated test data is the most labour-intensive, because it requires you to have both an old and new implementation of the affected code available.
* You may need to write a golang program that can use the old code and the new code to decode the existing data, update the affected components, and write the new test data back out.
* Alternatively you could generate a completely new set of test data. It depends on what the test data is for and what expectations the test suites that use it have. 

### Examples of p2p protocol changes

Commit [2e67fd3](https://github.com/soteria-dag/soterd/commit/2e67fd353a400785ebdf4cc1f428e130f050e9a6) implemented a change to the block structure (adding parent sub-header to the block). It covers all of the above points, so it could be used as a reference when considering making updates to the p2p wire protocol. 