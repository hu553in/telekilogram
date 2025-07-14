build_dir := "./build"
bin_name := "app"

set dotenv-load := true

all: check run

check: install-deps lint fmt test build

install-deps:
    go mod download

lint:
    golangci-lint run

fmt:
    golangci-lint fmt

test:
    go test -v -coverprofile="{{build_dir}}/coverage.out" -cover ./...

build:
    go build -o {{build_dir}}/{{bin_name}}

run:
    {{build_dir}}/{{bin_name}}
