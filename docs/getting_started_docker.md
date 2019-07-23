### Getting started with Docker

These instructions describe getting started with soterd using the [docker](https://docs.docker.com/install/) and/or [docker-compose](https://docs.docker.com/compose/install/) tools, as an alternative to compiling/installing with the golang tooling.

#### Building the docker image

1. If soteria-dag projects _aren't publicly available yet_, you need to clone the private dependencies of soteria-dag before building the container image.

    ```bash
    mkdir soteria-dag
    git clone https://github.com:soteria-dag/soterwallet.git soteria-dag/soterwallet
    ```


1. Clone the soteria-dag soterd git repository

    ```bash
    git clone https://github.com/soteria-dag/soterd.git soteria-dag/soterd
    ```

2. Build the docker image for soteria-dag/soterd

    Run this command the `soteria-dag` directory, **not** the `soterd` directory. When building a container image for a private project, we need to run the `docker build` command from a directory that has access to the private project _and_ all of its private dependencies.
    ```
    cd soteria-dag
    docker build --tag soteria-dag/soterd:latest -f soterd/Dockerfile .
    ```

#### Using the docker image to demonstrate block-dag generation

1. Run [dagviz](../cmd/dagviz/README.md) to generate a sample dag and step through its creation

    ```bash
    docker run --rm --volume=`pwd`/demo:/srv/demo soteria-dag/soterd:latest dagviz -duration 10 -output /srv/demo
    ```

2. Open the `./demo/dag_0.html` file in your browser, and navigate through the dag (if `-stepping` was specified for the `dagviz` run).

* Blocks are colored based on the node that generated them.
* You can hover your mouse over a block to see more information about it.
* You can step forward/backward through snapshots of dag-generation with _prev_ and _next_ hyperlinks (if `-stepping` was specified for the `dagviz` run).

    ```bash
    /opt/google/chrome/chrome --new-window file://`pwd`/demo/dag_0.html
    ```

#### Running a soterd node using the `docker` command

```
docker run --rm --publish "5070-5071/tcp" soteria-dag/soterd:latest soterd --testnet --datadir=/srv/soterd/data --logdir=/srv/soterd/logs --listen=0.0.0.0:5070 --rpclisten=0.0.0.0:5071 --rpcuser=USER --rpcpass=PASS --miningaddr=mrqvnRT17uBvozaaZXZJb2LSeWiWadx96N
```
After the docker instance is started, you can use `docker ps` to see which ports on the host are mapped to `listen` and `rpclisten`, then publish the host ports to other nodes so they can connect to your node.

#### Running several soterd nodes using the `docker-compose` command

These steps show how we can use `docker-compose` to run and control 2 simnet nodes. Simnet is used in these steps instead of testnet, to demonstrate block generation without having to wait very long for results.

1. Spin up simnet and testnet soterd nodes with `docker-compose`

    ```bash
    cd soterd
    docker-compose up -d
    ```

2. Check the dag tips on simnet1

    ```bash
    docker-compose exec simnet1 soterctl --simnet --rpcuser=USER --rpcpass=PASS --rpcserver=127.0.0.1:18556 --skipverify getdagtips
    ```

3. Create a connection between simnet1 and simnet2

    ```bash
    docker-compose exec simnet1 soterctl --simnet --rpcuser=USER --rpcpass=PASS --rpcserver=127.0.0.1:18556 --skipverify addnode simnet2:18565 add
    ```

4. Show that simnet1 is now connected to simnet2

    ```bash
    docker-compose exec simnet1 soterctl --simnet --rpcuser=USER --rpcpass=PASS --rpcserver=127.0.0.1:18556 --skipverify getpeerinfo
    ```

5. Start mining on simnet
    ```bash
    docker-compose exec simnet1 soterctl --simnet --rpcuser=USER --rpcpass=PASS --rpcserver=127.0.0.1:18556 --skipverify setgenerate true 1
    docker-compose exec simnet2 soterctl --simnet --rpcuser=USER --rpcpass=PASS --rpcserver=127.0.0.1:18566 --skipverify setgenerate true 1
    ```

6. You can see if it's generated blocks by repeating the `getdagtips` rpc call or by tailing soterd logs with `docker-compose`

    ```bash
    docker-compose logs --follow --tail=10 simnet1 simnet2
    ```

7. Once it has, stop mining on simnet

    ```bash
    docker-compose exec simnet1 soterctl --simnet --rpcuser=USER --rpcpass=PASS --rpcserver=127.0.0.1:18556 --skipverify setgenerate false 0
    docker-compose exec simnet2 soterctl --simnet --rpcuser=USER --rpcpass=PASS --rpcserver=127.0.0.1:18566 --skipverify setgenerate false 0
    ```

8. Check dag state on each node

    ```bash
    # Check simnet1 tips
    docker-compose exec simnet1 soterctl --simnet --rpcuser=USER --rpcpass=PASS --rpcserver=127.0.0.1:18556 --skipverify getdagtips

    # Check simnet2 tips
    docker-compose exec simnet2 soterctl --simnet --rpcuser=USER --rpcpass=PASS --rpcserver=127.0.0.1:18566 --skipverify getdagtips
    ```

9. Render the dag as an SVG file

    ```bash
    docker-compose exec simnet1 /bin/bash -c "soterctl --simnet --rpcuser=USER --rpcpass=PASS --rpcserver=127.0.0.1:18556 --skipverify renderdag | jq --raw-output .dot | dot -Tsvg" > dag.svg
    ```

10. We can open the SVG file in our browser

    ```bash
    /opt/google/chrome/chrome --new-window file://`pwd`/dag.svg
    ```

11. Stop the nodes

    ```bash
    docker-compose down
    ```

#### Running a testnet node using the `docker-compose` command

The `docker-compose.yml` file contains a `testnet` node definition, which you can target with:

```bash
docker-compose up -d testnet
```

Control of this node works the same as with simnet, except that you're likely to want to connect the node to Soteria's testnet:

```bash
docker-compose exec testnet soterctl --testnet --rpcuser=USER --rpcpass=PASS --rpcserver=127.0.0.1:5071 --skipverify addnode 134.209.59.43:5070 add
``` 
