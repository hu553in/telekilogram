BUILD_DIR ?= ./dist

.PHONY: ensure-build-dir
ensure-build-dir:
	mkdir -p $(BUILD_DIR)

.PHONY: pre-commit
pre-commit: build lint check-deps verify-test-coverage

.PHONY: check
check: build fmt lint check-deps verify-test-coverage

.PHONY: install-deps
install-deps:
	go mod download

.PHONY: fmt
fmt:
	golangci-lint fmt

.PHONY: lint
lint:
	golangci-lint run

.PHONY: check-deps
check-deps: install-deps
	go tool govulncheck ./...

.PHONY: test
test: ensure-build-dir install-deps
	go test \
		-race \
		-coverprofile="$(BUILD_DIR)/coverage.out" \
		-covermode=atomic \
		-coverpkg=./... \
		./...

.PHONY: verify-test-coverage
verify-test-coverage: test
	go run github.com/vladopajic/go-test-coverage/v2@latest --config=./.testcoverage.yml

.PHONY: build
build: install-deps
	CGO_ENABLED=1 GOFLAGS="-buildvcs=false" \
    go build -trimpath -ldflags="-s -w" -o $(BUILD_DIR)/telekilogram ./cmd

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

.PHONY: sqlc
sqlc:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate
