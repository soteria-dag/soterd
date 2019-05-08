txscript
========

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

Package txscript implements the soter transaction script language.  There is
a comprehensive test suite.

This package has intentionally been designed so it can be used as a standalone
package for any projects needing to use or validate soter transaction scripts.

## Soter Scripts

Soter provides a stack-based, FORTH-like language for the scripts in
the soter transactions It is currently identical to the Bitcoin Script language. This language is not turing complete
although it is still fairly powerful.  A description of the language
can be found at https://en.bitcoin.it/wiki/Script

## Installation and Updating

```bash
$ go get -u github.com/soteria-dag/soterd/txscript
```

## Examples

* [Standard Pay-to-pubkey-hash Script](http://godoc.org/github.com/btcsuite/btcd/txscript#example-PayToAddrScript)  
  Demonstrates creating a script which pays to a soter address.  It also
  prints the created script hex and uses the DisasmString function to display
  the disassembled script.

* [Extracting Details from Standard Scripts](http://godoc.org/github.com/btcsuite/btcd/txscript#example-ExtractPkScriptAddrs)  
  Demonstrates extracting information from a standard public key script.

* [Manually Signing a Transaction Output](http://godoc.org/github.com/btcsuite/btcd/txscript#example-SignTxOutput)  
  Demonstrates manually creating and signing a redeem transaction.


## License

Package txscript is licensed under the [copyfree](http://copyfree.org) ISC
License.
