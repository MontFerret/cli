VERSION ?= $(shell sh versions.sh cli)
FERRET_VERSION = $(shell sh versions.sh ferret)
DIR_BIN = ./bin
NAME = ferret

default: build

build: vet test compile

install:
	go get

compile:
	go build -v -o ${DIR_BIN}/${NAME} \
	-ldflags "-X main.version=${VERSION} -X github.com/MontFerret/cli/runtime.version=${FERRET_VERSION}" \
	./main.go

test:
	go test ./...

cover:
	go test -race -coverprofile=coverage.txt -covermode=atomic ... && \
	curl -s https://codecov.io/bash | bash

doc:
	godoc -http=:6060 -index

fmt:
	go fmt ./...

lint:
	revive -config revive.toml -formatter stylish ./...

vet:
	go vet ./...