FROM golang:1.17.3

RUN adduser --home /home/api --disabled-password --geco "" api
RUN adduser api sudo
RUN echo '%sudo ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers
RUN usermod -aG sudo api

WORKDIR /go/src/api
USER api

COPY ./bin/log-ingester-api ./

EXPOSE 9010

ENTRYPOINT ["/go/src/api/log-ingester-api"]