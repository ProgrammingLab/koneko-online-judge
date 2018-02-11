# Koneko Online Judge (=^ - ^=)
[![Build Status](https://travis-ci.org/gedorinku/koneko-online-judge.svg?branch=test)](https://travis-ci.org/gedorinku/koneko-online-judge)

## Requirements
- Docker

## Usage(with Docker Compose)

### Set environment variable
```
    export KOJ_DB_PASSWORD="password"
```

### Start the backend server
```
    go get github.com/gedorinku/koneko-online-judge/...
    cd $GOPATH/src/github.com/gedorinku/koneko-online-judge/server/
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
    export KOJ_DB_DRIVER="mysql"
    export KOJ_DB_SPEC="user:password@/dbName?charset=utf8&parseTime=True&loc=Local"
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
