#!/bin/sh

go get -u github.com/golang/dep/cmd/dep
cd server
dep ensure -vendor-only
cd ../nekonote
dep ensure -vendor-only
cd ../runner
dep ensure -vendor-only

docker build -t koneko-online-judge-image-cpp ../server/container/cpp/
