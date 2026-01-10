BUILD_DIR ?= ./dist

GOLANGCI_LINT_CONFIG_URL ?= https://raw.githubusercontent.com/maratori/golangci-lint-config/refs/heads/main/.golangci.yml

MAIN_BRANCH ?= main

.PHONY: ensure-build-dir
ensure-build-dir:
	mkdir -p $(BUILD_DIR)

.PHONY: pre-commit
pre-commit: lint verify-test-coverage build

.PHONY: check
check: fmt lint verify-test-coverage build

.PHONY: install-deps
install-deps:
	go mod download

.PHONY: update-lint-config
update-lint-config:
	@tmp=$$(mktemp); \
	if curl -fsSL $(GOLANGCI_LINT_CONFIG_URL) -o "$$tmp"; then \
		mv "$$tmp" .golangci.yaml && \
		sed -i '' "s|github.com/my/project|github.com/hu553in/telekilogram|g" .golangci.yaml; \
	else \
		rm -f "$$tmp"; \
		exit 1; \
	fi

.PHONY: fmt
fmt:
	golangci-lint fmt

.PHONY: lint
lint:
	golangci-lint run

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
