miningdag
======

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

## Overview

This is a work-in-progress CPU-based block miner.

`miningdag` is also in-transition. This means several things:
* Original btcd blockchain mining code remains in the `mining` package
* The `mining` package isn't referenced by the rest of the codebase
* Once original blockchain code is removed from this repository, the `miningdag` package will be renamed to `mining`
* The `github.com/soteria-dag/soterd/mining/cpuminer` package utilizes the `miningdag` package, even though it's under the `mining` namespace in the filesystem. 

The package provides a 'manager' (goroutine) that can optionally be started when soterd is run. Managers communicate with other managers via message passing over channels associated with them.

## Installation and Updating

```bash
$ go get -u github.com/soteria-dag/soterd/miningdag
```

## License

Package miningdag is licensed under the [copyfree](http://copyfree.org) ISC
License.
