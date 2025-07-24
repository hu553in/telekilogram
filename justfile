build_dir := "./build"
bin_name := "app"

set dotenv-load := true

all: check run

[private]
ensure-build-dir:
    mkdir -p {{build_dir}}

pre-commit: install-deps lint test build

check: install-deps fmt lint test build

install-deps:
    go mod download

fmt:
    golangci-lint fmt

lint:
    golangci-lint run

test: ensure-build-dir
    go test -v ./... \
        -coverprofile="{{build_dir}}/coverage.out" \
        -covermode=atomic \
        -coverpkg=./...

build: ensure-build-dir
    go build -o {{build_dir}}/{{bin_name}}

run: ensure-build-dir
    {{build_dir}}/{{bin_name}}
