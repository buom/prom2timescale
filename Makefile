export GO111MODULE=on

.PHONY: build
build:
	go build -o bin/prom2timescale .
