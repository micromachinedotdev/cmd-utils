# Variables
APP_PREFIX := @micromachine.dev
BINARY := micromachine
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//')
COMMIT := $(shell git rev-parse --short HEAD)
LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT)
.PHONY: lint test deps build-all release npm-publish

# Run linter
lint:
	golangci-lint run

test:
	go test ./...

# Install dependencies
deps:
	go mod download
	go mod tidy

# Build all platforms
build-all: build-darwin-arm64 build-darwin-x64 build-linux-arm64 build-linux-arm build-linux-x64 build-win32-x64 build-win32-arm64

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

build-win32-arm64:
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o npm/$(APP_PREFIX)/win32-arm64/bin/$(BINARY).exe ./main.go

build-win32-x64:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o npm/$(APP_PREFIX)/win32-x64/bin/$(BINARY).exe ./main.go

release:
	npx --package semantic-release \
	--package @semantic-release/exec \
	--package @semantic-release/git \
	--package conventional-changelog-conventionalcommits \
	semantic-release


npm-publish:
	@echo "Publishing version $(VERSION)"
	node -e " \
		const fs = require('fs'); \
		const pkg = JSON.parse(fs.readFileSync('npm/$(BINARY)/package.json')); \
		for (const dep in pkg.optionalDependencies) { \
			pkg.optionalDependencies[dep] = '$(VERSION)'; \
		} \
		fs.writeFileSync('npm/$(BINARY)/package.json', JSON.stringify(pkg, null, 2) + '\n'); \
	"
	@find npm/$(APP_PREFIX) -name "package.json" -execdir npm version $(VERSION) --no-git-tag-version \; -execdir npm publish --access public \;
	cd npm/$(BINARY) && npm version $(VERSION) --no-git-tag-version && npm publish --access public