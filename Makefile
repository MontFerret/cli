VERSION ?= $(shell sh versions.sh cli)
FERRET_VERSION = $(shell sh versions.sh ferret)
DIR_BIN = ./bin
NAME = ferret

default: build

build: vet lint test compile

install:
	go mod download

compile:
	go build -v -o ${DIR_BIN}/${NAME} \
	-ldflags "-X main.version=${VERSION} -X github.com/MontFerret/cli/runtime.version=${FERRET_VERSION}" \
	./ferret/main.go

test:
	go test ./...

fmt:
	go fmt ./...

lint:
	revive -config revive.toml -formatter stylish ./...

vet:
	go vet ./...