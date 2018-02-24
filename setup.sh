#!/bin/sh

go get -u github.com/golang/dep/cmd/dep
cd server
dep ensure
cd ../nekonote
dep ensure

docker build -t koneko-online-judge-image-cpp ../server/container/cpp/
