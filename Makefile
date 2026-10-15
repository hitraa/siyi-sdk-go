.PHONY: all build test test-race lint clean run-web run-cli coverage

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
GOVET=$(GOCMD) vet

# Binary names
CLI_BIN=bin/siyi-cli
WEB_BIN=bin/siyi-web

all: test build

build:
	@mkdir -p bin
	$(GOBUILD) -v -o $(CLI_BIN) ./cmd/siyi-cli
	$(GOBUILD) -v -o $(WEB_BIN) ./cmd/siyi-web

test:
	$(GOTEST) -v ./...

test-race:
	$(GOTEST) -race -count=1 -v ./...

coverage:
	$(GOTEST) -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report written to coverage.html"

lint:
	$(GOVET) ./...
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping"; \
	fi

clean:
	$(GOCLEAN)
	rm -rf bin/ coverage.out coverage.html

run-web: build
	./$(WEB_BIN) -port 8080 -ip 192.168.144.25

run-cli: build
	./$(CLI_BIN) -h
