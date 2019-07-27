FROM openjdk:11-jdk-slim-buster
LABEL maintainer="Ryota Egusa <egusa.ryota@gmail.com>"

ENV JAVA_HOME /usr/local/openjdk-11

RUN apt-get update \
    && apt-get install time sudo \
    && apt-get clean

ENV KOTLIN_COMPILER_VERSION 1.3.41

RUN apt-get update \
    && apt-get install -y wget unzip \
    && wget https://github.com/JetBrains/kotlin/releases/download/v${KOTLIN_COMPILER_VERSION}/kotlin-compiler-${KOTLIN_COMPILER_VERSION}.zip -O /tmp/kotlin-compiler.zip \
    && unzip /tmp/kotlin-compiler.zip -d /usr/lib \
    && rm -f /tmp/kotlin-compiler.zip \
    && apt-get remove -y wget unzip \
    && apt-get autoremove -y \
    && apt-get clean

ENV PATH $PATH:/usr/lib/kotlinc/bin

RUN sh -c 'echo 127.0.1.1 $(hostname) >> /etc/hosts'

RUN mkdir /tmp/koj-workspace && chmod 777 /tmp/koj-workspace
