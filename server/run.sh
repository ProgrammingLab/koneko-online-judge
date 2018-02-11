#!/bin/sh

docker build -t koneko-online-judge-image-cpp ./container/cpp/

./main
