# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=log-ingester
BINARY_NAME_API=log-ingester-api


GO_OPT= -mod vendor -race

.PHONY: build
build:
	go build $(GO_OPT) -o ./ingester/bin/$(BINARY_NAME) ./ingester
	go build $(GO_OPT) -o ./api/bin/$(BINARY_NAME_API) ./api

.PHONY: install
install:
	go install $(GO_OPT) ./

.PHONY: dbuild
dbuild:
	@docker build -t casteloig/log-ingester:0.2 ./ingester
	@docker build -t casteloig/log-ingester-api:0.2 ./api

.PHONY: drun
drun:
	@docker run -d --name log-ingester -p 9010:9010 log-ingester:latest
