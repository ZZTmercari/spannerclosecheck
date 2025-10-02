.PHONY: build test install clean

build:
	go build -o spannerclosecheck main.go

test:
	go test ./...

install:
	go install

clean:
	rm -f spannerclosecheck

lint:
	golangci-lint run

.DEFAULT_GOAL := build
