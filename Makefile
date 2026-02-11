BINARY := utsuro
BIN_DIR := bin
CMD := ./cmd/utsuro

GO ?= go
GOCACHE ?= /tmp/go-build
GOMODCACHE ?= /tmp/go-mod
GOENV := GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE)

.PHONY: all
all: build

.PHONY: build
build: $(BIN_DIR)/$(BINARY)

$(BIN_DIR)/$(BINARY): go.mod $(shell find cmd internal -name '*.go' -type f)
	$(GOENV) $(GO) build -o $@ $(CMD)

.PHONY: run
run:
	$(GOENV) $(GO) run $(CMD)

.PHONY: test
test:
	$(GOENV) $(GO) test ./... -count=1 -timeout=120s

.PHONY: cover
cover:
	$(GOENV) $(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -func=coverage.out

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: vet
vet:
	$(GOENV) $(GO) vet ./...

.PHONY: tidy
tidy:
	$(GOENV) $(GO) mod tidy

.PHONY: clean
clean:
	rm -f $(BIN_DIR)/$(BINARY)
	rm -f coverage.out
