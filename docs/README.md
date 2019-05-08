### Table of Contents
1. [About](#About)
2. [Getting Started](#GettingStarted)
    1. [Docker](#Docker)
    2. [Install from source](#Source)
    3. [Configuration](#Configuration)
    4. [Controlling and Querying soterd via soterctl](#SoterctlConfig)
    5. [Mining](#Mining)
3. [Help](#Help)
    1. [Network Configuration](#NetworkConfig)
    2. [Wallet](#Wallet)
4. [Contact](#Contact)
5. [Developer Resources](#DeveloperResources)
    1. [Code Contribution Guidelines](#ContributionGuidelines)
    2. [JSON-RPC Reference](#JSONRPCReference)
    3. [Utility commands](#SoterdUtilities)
    4. [The soter-related Go Packages](#GoPackages)
    5. [Managers and handlers (goroutines)](#ManagerHandler)
    6. [Updating RPC Calls](#UpdateRPCCall)
    7. [Updating P2P Wire Protocol](#UpdateP2PWire)
    8. [Soterd profiling](#SoterdProfiling)
    9. [Soterd startup](#SoterdStartup)
6. [Other Soteria projects](#Soteria)

<a name="About" />

### 1. About

soterd is a full soter node implementation written in Go (golang). It started as a fork of the [btcd](https://github.com/btcsuite/btcd) project, where development focus has changed from providing a bitcoin node to providing an [implementation of blockdag]((docs/intro_to_blockdag.md)) for soter.

The intent of this project is for it to act as a component in Soteria's Trade Anything Platform (TAP); a pillar of Soteria's [Self Sustainable Decentralized Economy](https://www.ssde.io/) vision (SSDE).

For a description of terms you'll find in this project, please refer to the [Glossary of Terms](glossary_of_terms.md) document.

### Current functionality 

#### Blocks
 
Soterd can generate, validate, download and advertise blockdag using rules similar to those in Bitcoin Core. Some of Bitcoin Core's BIPs don't apply to  blockdag, so features that don't make sense for use with  blockdag may be disabled and removed as development continues.

Soterd inherits and extends btcd's block validation testing framework, which contains all of the 'official' bitcoin block acceptance tests (and some additional ones). 

#### Transactions

Soterd maintains a transaction pool, and relays individual transactions that have not yet made it into a block. It ensures all individual transactions admitted to the pool follow the rules required by the block chain and also includes more strict checks which filter transactions based on miner requirements ("standard" transactions).

#### Excluded functionality

Like btcd, soterd does not include wallet functionality. [soterwallet](https://github.com/soteria-dag/soterwallet) is intended for making or receive payments with soterd, however it is still being updated from the [btcwallet](https://github.com/btcsuite/btcwallet) fork to be compatible with soterd. In the meantime, transactions can be demonstrated with the [gentx](https://github.com/soteria-dag/soter-tools/cmd/gentx/README.md) command.

<a name="GettingStarted" />

### 2. Getting Started

<a name="Docker" />

#### 2.1 Docker container

Refer to the [Getting started with Docker](getting_started_docker.md) document for instructions on running soterd without the need for golang and git tooling.

<a name="Source" />

#### 2.2 Install from source

Refer to the [Setting up soterd](install_run_update.md) document for instructions on building soterd from source, installing, and running it.

<a name="Configuration" />

#### 2.3 Configuration

soterd has several configuration options, which you can see with `soterd --help`.

<a name="SoterctlConfig" />

#### 2.4 Controlling and Querying soterd via soterctl

soterctl is a command line utility that can be used to both control and query soterd via [RPC](http://www.wikipedia.org/wiki/Remote_procedure_call). soterd does **not** enable its RPC server by default; You must configure at minimum both an RPC username and password or both an RPC limited username and password:

* soterd.conf configuration file
```
[Application Options]
rpcuser=myuser
rpcpass=SomeDecentp4ssw0rd
rpclimituser=mylimituser
rpclimitpass=Limitedp4ssw0rd
```
* soterctl.conf configuration file
```
[Application Options]
rpcuser=myuser
rpcpass=SomeDecentp4ssw0rd
```
OR
```
[Application Options]
rpclimituser=mylimituser
rpclimitpass=Limitedp4ssw0rd
```
For a list of available options, run: `$ soterctl --help`

<a name="Mining" />

#### 2.5 Mining

soterd supports the `getblocktemplate` RPC.
The limited user cannot access this RPC.


1. Add the payment addresses with the `miningaddr` option.

    ```
    [Application Options]
    rpcuser=myuser
    rpcpass=SomeDecentp4ssw0rd
    miningaddr=12c6DSiU4Rq3P4ZxziKxzrL5LmMBrzjrJX
    miningaddr=1M83ju3EChKYyysmM2FXtLNftbacagd8FR
    ```

2. Add soterd's RPC TLS certificate to system Certificate Authority list.

    `cgminer` uses [curl](http://curl.haxx.se/) to fetch data from the RPC server.
    Since curl validates the certificate by default, we must install the `soterd` RPC
    certificate into the default system Certificate Authority list.

    **Ubuntu**

    1. Copy rpc.cert to /usr/share/ca-certificates: `# cp /home/user/.soterd/rpc.cert /usr/share/ca-certificates/soterd.crt`
    2. Add soterd.crt to /etc/ca-certificates.conf: `# echo soterd.crt >> /etc/ca-certificates.conf`
    3. Update the CA certificate list: `# update-ca-certificates`

3. Set your mining software url to use https.

    `$ cgminer -o https://127.0.0.1:8334 -u rpcuser -p rpcpassword`

<a name="Help" />

### 3. Help

<a name="NetworkConfig" />

#### 3.1 Network Configuration

* [What Ports Are Used by Default?](default_ports.md)
* [How To Listen on Specific Interfaces](configure_peer_server_listen_interfaces.md)
* [How To Configure RPC Server to Listen on Specific Interfaces](configure_rpc_server_listen_interfaces.md)
* [Configuring soterd with Tor](configuring_tor.md)

<a name="Wallet" />

#### 3.2 Wallet

Soterd doesn't include a wallet. [soterwallet](https://github.com/soteria-dag/soterwallet) is intended for making or receive payments with soterd, however it is still being updated from the [btcwallet](https://github.com/btcsuite/btcwallet) fork to be compatible with soterd. In the meantime, transactions can be demonstrated with the [gentx](https://github.com/soteria-dag/soter-tools/cmd/gentx/README.md) command.


<a name="Contact" />

### 4. Contact

Please refer to the (Soteria website)

<a name="DeveloperResources" />

### 5. Developer Resources

<a name="ContributionGuidelines" />

* [Code Contribution Guidelines](code_contribution_guidelines.md)

<a name="JSONRPCReference" />

* [JSON-RPC Reference](json_rpc_api.md)
    * [RPC Examples](json_rpc_api.md#ExampleCode)

<a name="SoterdUtilities">

* The [cmd](../cmd/README.md) directory contains stand-alone utilities that assist in the setup and management of soterd nodes

<a name="GoPackages" />

* The soter-related Go Packages:
    * [addrmgr](../addrmgr/README.md) - A p2p address manager, to help decide which nodes to connect to
    * [blockdag](../blockdag/README.md) - Provides Soter block handling and dag selection rules
        * [blockdag/phantom](../blockdag/phantom/README.md)
        * [blockdag/fullblocktests](../blockdag/fullblocktests) - Provides a set of block tests for testing the consensus validation rules
    * [chaincfg](../chaincfg/README.md) - Defines dag configuration parameters for soter networks
            * [chainhash](../chaincfg/chainhash/README.md) - Provides a generic hash type and related functions
    * [connmgr](../connmgr/README.md) - A p2p connection manager
    * [database](../database/README.md) - Provides block and metadata storage
        * [database/ffldb](../database/ffldb//README.md) - A driver for database package that uses leveldb for backing storage.
        * [database/internal/treap](../database/internal/treap//README.md) - An implementation of treap data structure for use in database code
    * [integration](../integration/README.md) - Integration test suites (inter-node communication, etc)
        * [integration/rpctest](../integration/rpctest/README.md) - Test harness for integration tests
    * [mempool](../mempool/README.md) - Provides a policy-enforced pool of unmined soter transactions
    * [metrics](../metrics/README.md) - Handles receiving runtime metrics from other managers, and makes data available via RPC calls.
    * [miningdag](../miningdag/README.md) - A work-in-progress CPU-based block miner
        [mining/cpuminer](../mining/cpuminer/README.md)
    * [netsync](../netsync/README.md) - Provides block synchronization between peers
    * [peer](../peer/README.md) - Provides p2p communication protocol and functionality for creating and managing network peers    
    * [rpcclient](../rpcclient/README.md) - A Websocket-enabled Soter JSON-RPC client
    * [soterec](../soterec/README.md) - Provides support for the elliptic curve cryptographic functions needed for Soter scripts
    * [soterjson](../soterjson/README.md) - Provides an extensive API
      for the underlying JSON-RPC command and return values
    * [soterlog](../soterlog/README.md) - Provides a logging interface for other packages
    * [soterutil](../soterutil/README.md) - Provides Soter convenience functions and types
    * [txscript](../txscript/README.md) -
          Implements the Soter transaction scripting language
    * [wire](../wire/README.md) - Implements and extends the Soter wire protocol

<a name="ManagerHandler" />

#### Managers and handlers

Some functionality in soterd is separated out into goroutines that communicate with each other and soterd via channels. There are managers/handlers for:
* P2P node address tracking 
* P2P network connections
* Peer state and communication
* Metrics collection
* Block mining
* Block synchronization between this node and the P2P network

For more information on managers and handlers, refer to the [Managers and Handlers](managers_handlers.md) documentation.

<a name="UpdateRPCCall" />

#### Updating RPC Calls

If you're interested in making changes to or adding new RPC calls, see the [Adding RPC Calls](adding_rpc_calls.md) documentation for notes on what parts of codebase you may need to touch during your efforts.

<a name="UpdateP2PWire" />

#### Updating P2P Wire Protocol

If you're interested in making changes to the P2P wire protocol for things like blocks or transactions, see the [Updating P2P Wire Protocol](updating_p2p_wire_protocol.md) documentation for notes on what parts of the codebase you may need to touch during your efforts.

<a name="SoterdProfiling" />

#### Soterd Profiling

When working on soterd code you may want to measure potential performance changes related to your changes. The [Soterd profiling](soterd_profiling.md) document describes how you can do this.

<a name="SoterdStartup" />

#### Soterd startup

The [Soterd startup](soterd_startup.md) document has a brief description of the soterd startup process.


<a name="Soteria" />

### 6. Other Soteria projects

These projects are related to soterd, and provide additional functionality:

* [soterwallet](https://github.com/soteria-dag/soterwallet) - for making or receiving payments with soterd and the soter network. See the [Wallet](#Wallet) section for a note about its current functionality.
* [soterdash](https://github.com/soteria-dag/soterdash) - a web ui that provides information about the soter network.
    * Soterd node info
    * BlockDAG info
        * Blocks
        * Transactions
    * P2P network connectivity