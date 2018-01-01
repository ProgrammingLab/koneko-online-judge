FROM gcc:7.2.0
MAINTAINER gedorinku <gedorinku@yahoo.co.jp>

RUN useradd koneko

RUN apt-get update && apt-get install time

RUN mkdir /tmp/koj-workspace && chmod 777 /tmp/koj-workspace

CMD ["/bin/bash"]
