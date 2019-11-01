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
make cuda29 && \
echo "cuda29 build ok" && \
mv cuda29 $gobin_dir/ && \
make verify29 && \
echo "verify29 build ok" && \
mv verify29 $gobin_dir/

cd $CURDIR
