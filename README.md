# Koneko Online Judge (=^ - ^=)
[![Build Status](https://travis-ci.org/gedorinku/koneko-online-judge.svg?branch=master)](https://travis-ci.org/gedorinku/koneko-online-judge)

[gedorinku/koneko-client](https://github.com/gedorinku/koneko-client)

## Requirements
- Docker

## Documents
[GitHub Wiki](https://github.com/gedorinku/koneko-online-judge/wiki)

## Config
こねこの設定ファイルは、`./server/koneko.toml`です。
`./server/koneko.sample.toml`を`./server/koneko.toml`にリネームしていい感じに設定してください。

## Usage(with Docker Compose)

### Set environment variable
```
export KOJ_DB_PASSWORD="password"
```

### Start the backend server
```
go get github.com/gedorinku/koneko-online-judge/...
cd $GOPATH/src/github.com/gedorinku/koneko-online-judge/
docker-compose up
```

## Debug

### Setup
```
go get github.com/gedorinku/koneko-online-judge/...
cd $GOPATH/src/github.com/gedorinku/koneko-online-judge/server/
docker build -t koneko-online-judge-image-cpp container/cpp/
dep ensure
```

### Set environment variables
```
# DOCKER_API_VERSIONにはインストールされているバージョンを指定
# 指定しないと'client version 1.36 is too new.'とか怒られる
export DOCKER_API_VERSION="1.35"
```

### Start the backend server
```
cd server
go build main.go
./main
# Go to http://localhost:9000/
```
