#!/bin/bash

function gobin () {
    if [ ! -z "$GOBIN" ]; then
        gobin_dir=$GOBIN
    else
        gobin_dir="${HOME}/go/bin"
    fi
}

gobin
mkdir -p $gobin_dir
CURDIR=`pwd`

cd ./mining/cuckoo/solver/cuckoo && \
make lean19x1 && \
echo "lean19x1 build ok" && \
mv lean19x1 $gobin_dir/ && \
make verify19 && \
echo "verify19 build ok" && \
mv verify19 $gobin_dir/

cd $CURDIR
