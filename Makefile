BINARY_NAME=redir
GO=go
IMAGE_NAME?=ghcr.io/alexandreroman/redir
GIT_COMMIT=$(shell git rev-parse --short HEAD)
LDFLAGS=-ldflags "-X main.gitCommit=$(GIT_COMMIT)"

.PHONY: build run clean test lint docker-build docker-run

build:
	$(GO) build $(LDFLAGS) -o $(BINARY_NAME) .

run: build
	./$(BINARY_NAME)

clean:
	$(GO) clean

test:
	$(GO) test ./...

lint:
	golangci-lint run

docker-build:
	docker build --build-arg GIT_COMMIT=$(GIT_COMMIT) -t $(IMAGE_NAME) .

docker-run:
	docker compose up --build
