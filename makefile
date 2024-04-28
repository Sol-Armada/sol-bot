.PHONY: clean build

version := $(shell git describe --tags --abbrev=0)
hash := $(shell git rev-parse --short HEAD)

clean:
	@rm -f bin/solbot

build: clean
	@go build -ldflags "-X main.version=${version} -X main.hash=${hash}" -o ./bin/solbot ./cmd/
