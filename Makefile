MODULE := ns-apns
BIN := bin
VERSION ?= 0.1.0
LDFLAGS := -s -w -X ns-apns/internal/version.Version=$(VERSION)

.PHONY: build dist test tidy clean

build:
	@mkdir -p $(BIN)
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BIN)/ns-apns ./cmd/ns-apns

dist:
	@mkdir -p $(BIN)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN)/ns-apns-linux-amd64 ./cmd/ns-apns
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BIN)/ns-apns-linux-arm64 ./cmd/ns-apns
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BIN)/ns-apns ./cmd/ns-apns

test:
	go test ./...

tidy:
	go mod tidy

clean:
	rm -rf $(BIN)
