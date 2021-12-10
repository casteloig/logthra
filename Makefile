# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=log-ingester


GO_OPT= -mod vendor -race

.PHONY: build
build:
	go build $(GO_OPT) -o ./bin/$(BINARY_NAME) ./

.PHONY: install
install:
	go install $(GO_OPT) ./

.PHONY: dbuild
dbuild:
	@docker build -t casteloig/log-ingester:0.1 ./

.PHONY: drun
drun:
	@docker run -d --name log-ingester -p 9010:9010 log-ingester:latest
