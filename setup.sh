#!/bin/sh

go get -u github.com/golang/dep/cmd/dep
cd server
dep ensure -v -vendor-only
cd ../nekonote
dep ensure -v -vendor-only
cd ../runner
dep ensure -v -vendor-only
GOOS=linux GOARCH=amd64 go build -ldflags '-extldflags "-static"' .

docker build -t koneko-online-judge-image-cpp ../server/container/cpp/
docker build -t koneko-online-judge-image-python3 ../server/container/python3/
docker build -t koneko-online-judge-image-openjdk ../container/openjdk/
