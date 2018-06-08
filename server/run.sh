#!/bin/sh

docker build -t koneko-online-judge-image-cpp ./container/cpp/
docker build -t koneko-online-judge-image-python3 ./container/python3

./main
