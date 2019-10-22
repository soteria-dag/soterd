Developers Intro
===

This document is meant for people familiar with writing software, and/or blockdag|blockchain technology.

For a description of terms you'll find in this project, please refer to the [Glossary of Terms](glossary_of_terms.md) document.

## Soterd current functionality 

Since most of the code is based off of earlier [btcd](https://github.com/btcsuite/btcd) efforts, soterd's peer communication and RPC protocol are very similar to those used in bitcoin nodes; Knowledge you have of bitcoin node implementations like btcd or [bitcoind](https://github.com/bitcoin/bitcoin) is transferrable to this project. README files are placed throughout this repository to help you become familiar with the layout and functionality of soterd code.

### Blocks
 
Soterd can generate, validate, download and advertise blockdag using rules similar to those in Bitcoin Core. Some of Bitcoin Core's BIPs don't apply to  blockdag, so features that don't make sense for use with  blockdag may be disabled and removed as development continues.

Soterd inherits and extends btcd's block validation testing framework, which contains all of the 'official' bitcoin block acceptance tests (and some additional ones). 

### Transactions

Soterd maintains a transaction pool, and relays individual transactions that have not yet made it into a block. It ensures all individual transactions admitted to the pool follow the rules required by the block chain and also includes more strict checks which filter transactions based on miner requirements ("standard" transactions).

## Excluded functionality

Like btcd, soterd does not include wallet functionality. [soterwallet](https://github.com/soteria-dag/soterwallet) is intended for making or receive payments with soterd, however it is still being updated from the [btcwallet](https://github.com/btcsuite/btcwallet) fork to be compatible with soterd. In the meantime, transactions can be demonstrated with the [gentx](https://github.com/soteria-dag/soter-tools/cmd/gentx/README.md) command. 


## Installation

### Docker container

Refer to the [Getting started with Docker](getting_started_docker.md) document for instructions on running soterd without the need for golang and git tooling.

### Linux/macOS - Build from Source

#### Requirements

[Go](http://golang.org) 1.13 or newer. (It may compile on earlier versions, but is tested on Go 1.13 and later)

[Git](https://git-scm.com/)

##### Optional

[graphviz](https://graphviz.org/), for DAG rendering functionality

#### Build and install steps

1. Install Go according to the [installation instructions](http://golang.org/doc/install)

2. Ensure Go was installed properly and is a supported version:

    ```bash
    $ go version
    $ go env GOROOT GOPATH GO111MODULE
    ```

    NOTE: The `GOROOT` and `GOPATH` above _must not_ be the same path.  It is
    recommended that `GOPATH` is set to a directory in your home directory such as
    `~/goprojects` to avoid write permission issues. It is also recommended to add
    `$GOPATH/bin` to your `PATH` at this point.

3. If soteria-dag projects _aren't publicly available yet_, you may need to redirect git requests using https to use ssh instead. This allows `go mod` and related package-management tools to work without prompting you for github credentials.

    ```bash
    # Add this section to your ~/.gitconfig file

    [url "ssh://git@github.com/soteria-dag/"]
        insteadOf = https://github.com/soteria-dag/
    ```

4. Run the following commands to obtain soterd, all dependencies, and install it:

    ```bash
    $ git clone https://github.com/soteria-dag/soterd $GOPATH/src/github.com/soteria-dag/soterd
    $ cd $GOPATH/src/github.com/soteria-dag/soterd
    $ go get -u github.com/Masterminds/glide
    $ glide install
    $ GO111MODULE=on go install . ./cmd/...
    ```

soterd (and utilities) will now be installed in `$GOPATH/bin`.  If you did not already add the _bin_ directory to your system path during Go installation, we recommend you do so now.


## Updating

### Linux/macOS - Build from Source

- Run the following commands to update soterd, all dependencies, and install it:

```bash
$ cd $GOPATH/src/github.com/soteria-dag/soterd
$ git pull && glide install
$ GO111MODULE=on go install . ./cmd/...
```

## Running soterd

### Docker container

Refer to the [Getting started with Docker](getting_started_docker.md) document for instructions on running soterd without the need for golang tooling.

### Linux/macOS

The following command will run a soterd node on testnet

```bash
$ soterd --testnet
```

### Configuring soterd

Refer to the [Configuration](README.md#Configuration) section of the main README file in the _docs_ folder for information on setting soterd configuration.

### Demo
If you are interested in seeing a quick demonstration of blockdag in action, [dagviz](../cmd/dagviz/README.md) is a good starting point.


## Developer resources

See the [Developer Resources](README.md#DeveloperResources) section of the main _docs/README.md_ file.
