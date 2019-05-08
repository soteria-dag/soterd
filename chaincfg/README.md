chaincfg
========

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

Package `chaincfg` defines chain configuration parameters for the three standard
Bitcoin networks and provides the ability for callers to define their own custom
Bitcoin networks.

Btcd authors designed this package so that it could be used as a standalone
package for any projects needing to use parameters for the standard Bitcoin
networks, or for projects needing to define their own network.

## Sample Use

```Go
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/soteria-dag/soterd/soterutil"
	"github.com/soteria-dag/soterd/chaincfg"
)

var testnet = flag.Bool("testnet", false, "operate on the testnet Bitcoin network")

// By default (without -testnet), use mainnet.
var chainParams = &chaincfg.MainNetParams

func main() {
	flag.Parse()

	// Modify active network parameters if operating on testnet.
	if *testnet {
		chainParams = &chaincfg.TestNet1Params
	}

	// later...

	// Create and print new payment address, specific to the active network.
	pubKeyHash := make([]byte, 20)
	addr, err := soterutil.NewAddressPubKeyHash(pubKeyHash, chainParams)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(addr)
}
```

## Installation and Updating

```bash
$ go get -u github.com/soteria-dag/soterd/chaincfg
```

## License

Package chaincfg is licensed under the [copyfree](http://copyfree.org) ISC
License.
