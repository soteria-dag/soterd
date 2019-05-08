metrics
=======

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

Package metrics implements a metrics collection manager.

## Overview

Metrics manager handles the receiving metrics from other managers, and providing metrics to other parts of code through its methods.

The intent of the metrics manager was to expose run-time stats from various other managers to external systems via RPC calls. 

Currently the metrics manager is used to expose block-mining metrics to external systems via the `getblockmetrics` RPC call.

## Installation and Updating

```bash
$ go get -u github.com/soteria-dag/soterd/metrics
```

## License

Package metrics is licensed under the [copyfree](http://copyfree.org) ISC License.