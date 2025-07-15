build_dir := "./build"
bin_name := "app"

set dotenv-load := true

all: check run

[private]
ensure-build-dir:
    mkdir -p {{build_dir}}

check: install-deps lint fmt test build

install-deps:
    go mod download

lint:
    golangci-lint run

fmt:
    golangci-lint fmt

test: ensure-build-dir
    go test -v ./... \
        -coverprofile="{{build_dir}}/coverage.out" \
        -covermode=atomic \
        -coverpkg=./...

build: ensure-build-dir
    go build -o {{build_dir}}/{{bin_name}}

run: ensure-build-dir
    {{build_dir}}/{{bin_name}}
