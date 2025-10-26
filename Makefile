APP_NAME := funwithflags
GO_FILES := $(shell find . -name '*.go' -not -path "./vendor/*")

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: lint
lint:
	go vet ./...

.PHONY: test
test:
	go test ./...

.PHONY: build
build:
	go build -o bin/$(APP_NAME) ./cmd/server

.PHONY: run
run:
	go run ./cmd/server

.PHONY: tidy
tidy:
	go mod tidy
