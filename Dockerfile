FROM docker:stable-dind
LABEL maintainer="Ryota Egusa <egusa.ryota@gmail.com>"

# Install Go
RUN apk --no-cache add musl-dev go
ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

# Setup Docker
ENV DOCKER_VERSION 1.35

# Setup Koneko
RUN apk --no-cache add bash git mercurial
RUN go get github.com/golang/dep/cmd/dep
COPY . /go/src/github.com/ProgrammingLab/koneko-online-judge
RUN cd /go/src/github.com/ProgrammingLab/koneko-online-judge/server/ \
    && dep ensure -vendor-only \
    && go build main.go
RUN cd /go/src/github.com/ProgrammingLab/koneko-online-judge/runner/ \
    && dep ensure -vendor-only \
    && go build -ldflags '-extldflags "-static"' . \
    && chmod 700 runner

WORKDIR /go/src/github.com/ProgrammingLab/koneko-online-judge/server/
# dockerdを起動させないようにする
ENTRYPOINT []
