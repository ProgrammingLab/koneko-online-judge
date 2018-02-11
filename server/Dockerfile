FROM docker:stable-dind
LABEL maintainer="gedorinku <gedorinku@yahoo.co.jp>"

# Install Go
RUN apk --no-cache add musl-dev go
ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

# Setup Docker
ENV DOCKER_VERSION 1.35
ENV DOCKER_HOST tcp://docker:2375

# Setup Koneko
RUN apk --no-cache add git mercurial \
    && go get github.com/golang/dep/cmd/dep \
    && go get github.com/gedorinku/koneko-online-judge/... \
    && cd /go/src/github.com/gedorinku/koneko-online-judge/server/ \
    && dep ensure \
    && go build main.go \
    && apk del git mercurial

RUN apk --no-cache add bash

WORKDIR /go/src/github.com/gedorinku/koneko-online-judge/server/
# dockerdを起動させないようにする
ENTRYPOINT []
