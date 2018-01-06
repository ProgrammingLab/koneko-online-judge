# Koneko Online Judge (=^ - ^=)

### Setup:
```
   go get github.com/revel/revel
   go get github.com/revel/cmd/revel
   go get github.com/gedorinku/koneko-online-judge
   docker build -t koneko-online-judge-image-cpp $GOPATH/src/github.com/gedorinku/koneko-online-judge/container/cpp/
```

### Set environment variables:
```
    export KOJ_SECRET="pJLzyoiDe17L36mytqC912j81PfTiolHm1veQK6Grn1En3YFdB5lvEHVTwFEaWvj"
    export KOJ_DB_DRIVER="mysql"
    export KOJ_DB_SPEC="user:password@/dbName?charset=utf8&parseTime=True&loc=Local"
    # DOCKER_API_VERSIONにはインストールされているバージョンを指定
    # 指定しないと'client version 1.36 is too new.'とか怒られる
    export DOCKER_API_VERSION="1.35"
```

### Start the web server:
```
   revel run github.com/gedorinku/koneko-online-judge
   # Go to http://localhost:9000/
```

## Code Layout

The directory structure of a generated Revel application:

    conf/             Configuration directory
        app.conf      Main app configuration file
        routes        Routes definition file

    app/              App sources
        init.go       Interceptor registration
        controllers/  App controllers go here
        views/        Templates directory
        models/
        deamon/

    messages/         Message files

    public/           Public static assets
        css/          CSS files
        js/           Javascript files
        images/       Image files

    tests/            Test suites
