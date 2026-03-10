BINARY_NAME=redir
GO=go
IMAGE_NAME?=ghcr.io/alexandreroman/redir

.PHONY: build run clean test lint docker-build docker-run

build:
	$(GO) build -o $(BINARY_NAME) .

run: build
	./$(BINARY_NAME)

clean:
	rm -f $(BINARY_NAME)

test:
	$(GO) test ./...

lint:
	golangci-lint run

docker-build:
	docker build -t $(IMAGE_NAME) .

docker-run:
	docker compose up --build
