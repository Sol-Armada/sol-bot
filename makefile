.PHONY: clean build

version := $(shell git describe --tags --abbrev=0)
hash := $(shell git rev-parse --short HEAD)

clean:
	@rm -f bin/solbot

build: clean
	@go build -ldflags "-X main.version=${version} -X main.hash=${hash}" -o ./bin/solbot ./cmd/

build-arm: clean
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version=${version} -X main.hash=${hash}" -o ./bin/solbot-arm ./cmd/

build-staging: clean
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version=${version} -X main.hash=${hash} -X main.environment=staging" -o ./bin/solbot-arm ./cmd/
