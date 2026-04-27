BINARY_NAME ?= lim
DIST_DIR ?= dist

.PHONY: build test clean dist-linux

build:
	go build -o $(BINARY_NAME) .

test:
	go test ./...

clean:
	rm -rf $(DIST_DIR)

# Produces Linux binaries suitable for redistribution.
dist-linux: clean
	mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 .
