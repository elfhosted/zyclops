# Variables
APP_NAME := zyclops
DOCKER_IMAGE := elfhosted/$(APP_NAME)
DOCKER_TAG := latest

# Go related variables
GOPATH := $(shell go env GOPATH)
GOBIN := $(GOPATH)/bin
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOCLEAN := $(GOCMD) clean

# Build flags
LDFLAGS := -w -s

# Environment variables with defaults
SERVER_PORT ?= 8080

.PHONY: all build clean test run docker-build docker-run

all: clean build test

build:
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o bin/$(APP_NAME) ./cmd/zyclops

clean:
	$(GOCLEAN)
	rm -f bin/$(APP_NAME)
	rm -rf torrents.bleve

test:
	$(GOTEST) -v ./...

run: build
	./bin/$(APP_NAME)

docker-build:
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run:
	docker run -p $(SERVER_PORT):$(SERVER_PORT) \
		-v $(HOME)/.kube:/root/.kube \
		-e SERVER_PORT=$(SERVER_PORT) \
		$(DOCKER_IMAGE):$(DOCKER_TAG)
