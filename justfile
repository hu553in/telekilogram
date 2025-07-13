build_dir := "build"
bin_name := "app"

set dotenv-load := true

all: build run

build:
    go build -o {{build_dir}}/{{bin_name}}

run:
    {{build_dir}}/{{bin_name}}
