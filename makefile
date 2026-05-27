.PHONY: clean build build-arm

version := $(shell git describe --tags --abbrev=0)
hash := $(shell git rev-parse --short HEAD)

clean:
	@rm -f bin/solbot

build: clean
	@go build -ldflags "-X main.version=${version} -X main.hash=${hash}" -o ./bin/solbot ./cmd/

# Note: CONFIG is now runtime-selected via APP_ENV and SOLBOT_CONFIG_FILE.
# One binary works for all environments (local, staging, prod).
#
# For cross-compilation (e.g., to ARM64), use:
#   make build GOOS=linux GOARCH=arm64

build-arm: clean
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version=${version} -X main.hash=${hash}" -o ./bin/solbot-arm ./cmd/
