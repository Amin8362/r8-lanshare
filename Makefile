BINARY ?= r8-lanshare

.PHONY: all clean vet

all:
	go mod tidy
	CGO_ENABLED=0 GOAMD64=v1 go build -trimpath -ldflags="-s -w" -o $(BINARY) .

vet:
	go mod tidy
	go vet ./...

clean:
	rm -f $(BINARY) *.exe
