#!/bin/bash

protoc --go_out=./model/proto/log --go_opt=paths=source_relative --go-grpc_out=./model/proto/log \
 model/proto/log/log.proto

