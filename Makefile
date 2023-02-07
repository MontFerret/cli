VERSION ?= $(shell sh versions.sh cli)
FERRET_VERSION = $(shell sh versions.sh ferret)
DIR_BIN = ./bin
NAME = ferret

default: build

build: vet lint test compile

install-tools:
	go install honnef.co/go/tools/cmd/staticcheck@latest && \
	go install golang.org/x/tools/cmd/goimports@latest && \
	go install github.com/mgechev/revive@latest

install:
	go mod download

compile:
	go build -v -o ${DIR_BIN}/${NAME} \
	-ldflags "-X main.version=${VERSION} -X github.com/MontFerret/cli/runtime.version=${FERRET_VERSION}" \
	./ferret/main.go

test:
	go test ./...

fmt:
	go fmt ./... && \
	goimports -w -local github.com/MontFerret ./browser ./cmd ./config ./ferret ./internal ./logger ./repl ./runtime

lint:
	staticcheck ./... && \
	revive -config revive.toml -formatter stylish -exclude ./pkg/parser/fql/... -exclude ./vendor/... ./...


vet:
	go vet ./...