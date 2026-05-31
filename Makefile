APP := gladiator
PKG := ./cmd/gladiator
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
CGO_ENABLED ?= 0
LDFLAGS := -s -w -X github.com/nishchay-veer/gladiator/internal/build.Version=$(VERSION)

.PHONY: test build snapshot clean

test:
	go test ./...

build:
	mkdir -p bin
	CGO_ENABLED=$(CGO_ENABLED) go build -ldflags "$(LDFLAGS)" -o bin/$(APP) $(PKG)

snapshot:
	mkdir -p dist
	CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(APP)_darwin_amd64 $(PKG)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(APP)_darwin_arm64 $(PKG)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(APP)_linux_amd64 $(PKG)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(APP)_linux_arm64 $(PKG)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(APP)_windows_amd64.exe $(PKG)

clean:
	rm -rf bin dist
