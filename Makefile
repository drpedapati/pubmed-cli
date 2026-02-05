.PHONY: build test test-integration install lint clean

BINARY := pubmed
PKG := ./cmd/pubmed

build:
	go build -o $(BINARY) $(PKG)

test:
	go test -short -count=1 ./...

test-integration:
	go test -tags integration -count=1 -v ./...

install:
	go install $(PKG)

lint:
	@which golangci-lint > /dev/null 2>&1 || echo "Install golangci-lint: https://golangci-lint.run/welcome/install/"
	golangci-lint run ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)
	go clean

coverage:
	go test -short -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
