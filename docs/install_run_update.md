Setting up soterd
===

This document describes installation, updating, and running soterd.


## Installation

### Docker container

Refer to the [Getting started with Docker](getting_started_docker.md) document for instructions on running soterd without the need for golang and git tooling.

### Linux/macOS - Build from Source

#### Requirements

* [Go](http://golang.org) 1.13 or newer. (It may compile on earlier versions, but is tested on Go 1.13 and later)
* [Git](https://git-scm.com/)

##### Optional

* [graphviz](https://graphviz.org/), for DAG rendering functionality

If you'd like to use an external cuckoo solver (the `--lean` or `--gpu` cli flags), the build requirements are listed in the [External cuckoo solver document](external_cuckoo_solver.md).

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
    $ git checkout exp0 # change to other branch if you like, or omit this step if on master
    $ go get -u github.com/Masterminds/glide
    $ glide install
    $ GO111MODULE=on go install . ./cmd/...
    ```

soterd (and utilities) will now be installed in `$GOPATH/bin`.  If you did not already add the _bin_ directory to your system path during Go installation, we recommend you do so now.


## Updating

### Docker container

Re-run the `docker build` command from the [Getting started with Docker](docs/getting_started_docker.md) document, and restart any soterd containers.

### Linux/macOS - Build from Source

- Run the following commands to update soterd, all dependencies, and install it:

```bash
$ cd $GOPATH/src/github.com/soteria-dag/soterd
$ git pull && glide install
$ GO111MODULE=on go install . ./cmd/...
```

## Running soterd

### Docker container

Refer to the [Getting started with Docker](docs/getting_started_docker.md) document for instructions on running soterd without the need for golang tooling.

### Linux/macOS

The following command will run a soterd node on testnet

```bash
$ ./soterd --testnet
```

### External cuckoo solvers

If you'd like to use an external cuckoo solver (the `--lean` or `--gpu` cli flags), the build requirements are listed in the [External cuckoo solver document](external_cuckoo_solver.md).

### Configuring soterd

Refer to the [Configuration](README.md#Configuration) section of the main README file in the _docs_ folder for information on setting soterd configuration.

### Demo
If you are interested in seeing a quick demonstration of blockdag in action, [dagviz](../cmd/dagviz/README.md) is a good starting point.


## Additional documentation

Refer to the [main README file](README.md) and the _docs_ folder for more information.
