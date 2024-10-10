# 'builder' includes all build & testing dependencies.
FROM debian:12 AS builder

RUN apt-get update &&\
    apt-get install --no-install-recommends -y \
      build-essential=12.9 \
      ca-certificates=20230311 \
      git=1:2.39.5-0+deb12u1 \
      curl=7.88.1-10+deb12u7 &&\
    apt-get clean &&\
    rm -rf /var/lib/apt/lists/*

# The FDB installer is not deleted so it can be used to
# install the library in the final stage at the end of
# this file.
ARG FDB_LIB_URL
RUN curl -Lo /fdb.deb $FDB_LIB_URL &&\
    dpkg -i /fdb.deb

ARG GO_URL
RUN curl -Lo /go.tar.gz $GO_URL &&\
    tar -C /usr/local -xzf /go.tar.gz &&\
    rm /go.tar.gz
ENV PATH="/root/go/bin:/usr/local/go/bin:${PATH}"
ENV GOCACHE="/cache/gocache"
ENV GOMODCACHE="/cache/gomod"

ARG GOLANGCI_LINT_URL="https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh"
ARG GOLANGCI_LINT_VER
RUN curl -Lo /golint.sh $GOLANGCI_LINT_URL &&\
    sh /golint.sh -b "$(go env GOPATH)/bin" $GOLANGCI_LINT_VER &&\
    rm /golint.sh
ENV GOLANGCI_LINT_CACHE="/cache/golangci-lint"

ARG SHELLCHECK_URL
RUN curl -Lo /shellcheck.tar.xz $SHELLCHECK_URL &&\
    tar -xf /shellcheck.tar.xz &&\
    mv /shellcheck-*/shellcheck /usr/local/bin &&\
    rm -r /shellcheck.tar.xz /shellcheck-*

ARG HADOLINT_URL
RUN curl -Lo /usr/local/bin/hadolint $HADOLINT_URL &&\
    chmod +x /usr/local/bin/hadolint

ARG JQ_URL
RUN curl -Lo /usr/local/bin/jq $JQ_URL &&\
    chmod +x /usr/local/bin/jq

ARG PANDOC_URL
RUN curl -Lo /pandoc.deb $PANDOC_URL &&\
    dpkg -i /pandoc.deb &&\
    rm /pandoc.deb

# Configure git so it allows any user to run git commands
# on the /fql directory. This allows the user which runs
# CI to be different from the user which built the Docker
# image.
RUN git config --global --add safe.directory /fql


# 'gobuild' executes 'go build'.
FROM builder AS gobuild

COPY . /src
WORKDIR /src

ARG FQL_VER
RUN go build -o /fql -ldflags="-X 'github.com/janderland/fql/internal/app.Version=${FQL_VER}'"


# The final stage builds the 'fql' image.
FROM debian:12

COPY --from=gobuild /fdb.deb /fdb.deb
RUN dpkg -i ./fdb.deb &&\
    rm /fdb.deb

ENV TERM="xterm-256color"

COPY --from=gobuild /fql /fql
COPY ./docker/shim.sh /shim.sh
ENTRYPOINT ["/shim.sh"]
