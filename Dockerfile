FROM golang:1.17.3

RUN adduser --home /home/log-ingester --disabled-password --geco "" log-ingester
RUN adduser log-ingester sudo
RUN echo '%sudo ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers
RUN usermod -aG sudo log-ingester

WORKDIR /go/src/log-ingester
USER log-ingester

COPY bin/log-ingester ./

EXPOSE 9010

ENTRYPOINT ["/go/src/log-ingester/log-ingester"]