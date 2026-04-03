VERSION ?= 0.1.0
LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION)"
BINARY := monocular

.PHONY: build build-all test vet lint clean

build:
	go build $(LDFLAGS) -o $(BINARY) .

build-all:
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY)-darwin-arm64 .
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-darwin-amd64 .
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-linux-amd64 .
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY)-linux-arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-windows-amd64.exe .

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

lint: vet
	@echo "Lint passed (go vet)"

clean:
	rm -f $(BINARY) $(BINARY)-*
