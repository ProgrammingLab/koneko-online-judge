FROM alpine:edge
LABEL maintainer="kp54 <kangpang65@gmail.com>"

RUN apk install sudo python3 

RUN sh -c 'echo 127.0.1.1 $(hostname) >> /etc/hosts'

RUN mkdir /tmp/koj-workspace && chmod 777 /tmp/koj-workspace
