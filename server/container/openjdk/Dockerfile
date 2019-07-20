FROM openjdk:11-jdk-slim-buster
LABEL maintainer="Ryota Egusa <egusa.ryota@gmail.com>"

RUN apt-get update \
    && apt-get install time sudo \
    && apt-get clean

RUN sh -c 'echo 127.0.1.1 $(hostname) >> /etc/hosts'

RUN mkdir /tmp/koj-workspace && chmod 777 /tmp/koj-workspace
