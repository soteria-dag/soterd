# You can build the soterd container by issuing this command from
# src/github.com/soteria-dag directory (not soterd directory).
# This is because docker needs access to both the soterd and soterwallet repositories (which are currently private).
# Once both soterd and soterwallet are public repos, this won't be necessary.
#
# cd ~/src/github.com/soteria-dag
# docker build --tag soteria-dag/soterd:latest -f soterd/Dockerfile .
FROM golang:1.13

LABEL description="Soteria DAG soterd image"

# Update apt & packages, including graphviz for rendering dags
ENV DEBIAN_FRONTEND noninteractive
RUN apt-get update && apt-get install -y apt-utils && apt-get -y -q dist-upgrade && apt-get install -y jq graphviz

ENV GO111MODULE on

# TODO(cedric): Switch to git clone with https url once soterd is publicly available
# If you are getting the source code from an invitation only testnet (for example, proviate github repo), please manually clone the repo.
COPY ./soterd /go/src/github.com/soteria-dag/soterd
COPY ./soterwallet /go/src/github.com/soteria-dag/soterwallet

# TODO(cedric): Once we migrate from glide to go.mod we won't need this step
# Create a fresh go.mod file for soterd
WORKDIR /go/src/github.com/soteria-dag/soterd
RUN rm --force go.mod go.sum; go mod init github.com/soteria-dag/soterd

# TODO(cedric): Don't replace github dependencies with local filesystem, once both repositories are publicly available
# Update go.mod to use the filesystem to resolve the dependencies, instead of github.
RUN go mod edit -replace=github.com/soteria-dag/soterd=/go/src/github.com/soteria-dag/soterd /go/src/github.com/soteria-dag/soterd/go.mod

RUN go mod edit -replace=github.com/soteria-dag/soterd=/go/src/github.com/soteria-dag/soterd /go/src/github.com/soteria-dag/soterwallet/go.mod
RUN go mod edit -replace=github.com/soteria-dag/soterwallet=/go/src/github.com/soteria-dag/soterwallet /go/src/github.com/soteria-dag/soterwallet/go.mod

# Update dependencies to whatever the 'latest' filessytem version is, without using `go get` because it
# doesn't seem to respect the `replace` statement.
#
# Update soterwallet for latest soterd version
WORKDIR /go/src/github.com/soteria-dag/soterd
RUN /bin/bash -c 'hash=$(git log --max-count=1 --format=format:"%H" | cut -c 1-12); \
                  ts=$(git log --max-count=1 --format=format:"%ct"); \
                  dt=$(TZ=UTC date -d @$ts +"%Y%m%d%H%M%S"); \
                  ver="v0.0.0"; \
                  modVer="${ver}-${dt}-${hash}"; \
                  go mod edit -droprequire=github.com/soteria-dag/soterd /go/src/github.com/soteria-dag/soterwallet/go.mod; \
                  go mod edit -require=github.com/soteria-dag/soterd@$modVer /go/src/github.com/soteria-dag/soterwallet/go.mod'

WORKDIR /go/src/github.com/soteria-dag/soterd
RUN go build && go install . ./cmd/... && mkdir -p /root/.soterd && touch /root/.soterd/soterd.conf

WORKDIR /go/src/github.com/soteria-dag/soterwallet
RUN go build && go install . ./cmd/... && mkdir -p /root/.soterwallet && touch /root/.soterwallet/soterwallet.conf

WORKDIR /go/src/github.com/soteria-dag/soterd
