#!/bin/sh

go get -u github.com/golang/dep/cmd/dep
cd server
dep ensure
cd ../nekonote
dep ensure
