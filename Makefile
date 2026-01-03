# Variables
APP_PREFIX := @micromachine.dev
VERSION := 1.0.0
BINARY := micromachine
VERSION := $(shell git describe --tags --abbrev=0 | sed 's/^v//')
COMMIT := $(shell git rev-parse --short HEAD)
LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT)

# Run linter
lint:
	golangci-lint run

# Install dependencies
deps:
	go mod download
	go mod tidy

# Build all platforms
build-all: build-darwin-arm64 build-darwin-x64 build-linux-arm64 build-linux-arm build-linux-x64 build-windows-x64 build-windows-arm64

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o npm/$(APP_PREFIX)/darwin-arm64/bin/$(BINARY) ./main.go

build-darwin-x64:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o npm/$(APP_PREFIX)/darwin-x64/bin/$(BINARY) ./main.go

build-linux-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o npm/$(APP_PREFIX)/linux-arm64/bin/$(BINARY) ./main.go

build-linux-arm:
	GOOS=linux GOARCH=arm CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o npm/$(APP_PREFIX)/linux-arm/bin/$(BINARY) ./main.go

build-linux-x64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o npm/$(APP_PREFIX)/linux-x64/bin/$(BINARY) ./main.go

build-windows-arm64:
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o npm/$(APP_PREFIX)/win32-arm64/$(BINARY).exe ./main.go

build-windows-x64:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o npm/$(APP_PREFIX)/win32-x64/$(BINARY).exe ./main.go

npm-publish:
	@echo "Publishing version $(VERSION)"
	cd npm/$(APP_PREFIX)/darwin-arm64 && npm version $(VERSION) --no-git-tag-version && npm publish --access public
	cd npm/$(APP_PREFIX)/darwin-x64 && npm version $(VERSION) --no-git-tag-version && npm publish --access public
	cd npm/$(APP_PREFIX)/linux-arm64 && npm version $(VERSION) --no-git-tag-version && npm publish --access public
	cd npm/$(APP_PREFIX)/linux-arm && npm version $(VERSION) --no-git-tag-version && npm publish --access public
	cd npm/$(APP_PREFIX)/linux-x64 && npm version $(VERSION) --no-git-tag-version && npm publish --access public
	cd npm/$(APP_PREFIX)/win32-arm64 && npm version $(VERSION) --no-git-tag-version && npm publish --access public
	cd npm/$(APP_PREFIX)/win32-x64 && npm version $(VERSION) --no-git-tag-version && npm publish --access public
	cd npm/$(BINARY) && npm version $(VERSION) --no-git-tag-version && npm publish --access public