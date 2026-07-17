.DEFAULT_GOAL := check

BUILD_DIR ?= ./dist

PRETTIER := bunx prettier -u
ACTIONLINT := bunx github-actionlint
TAPLO := bunx @taplo/cli

.PHONY: ensure-build-dir
ensure-build-dir:
	mkdir -p $(BUILD_DIR)

.PHONY: check-workflows
check-workflows:
	$(ACTIONLINT)

.PHONY: check
check: lint build check-generated check-deps check-vulns verify-test-coverage check-workflows

.PHONY: check-fix
check-fix: lint-fix
	$(MAKE) check

.PHONY: install-deps
install-deps:
	go mod download

.PHONY: lint
lint:
	$(PRETTIER) -c .
	$(TAPLO) fmt --check
	golangci-lint fmt --diff
	golangci-lint run

.PHONY: lint-fix
lint-fix:
	$(PRETTIER) -w .
	$(TAPLO) fmt
	golangci-lint fmt
	golangci-lint run --fix

.PHONY: check-deps
check-deps: install-deps
	go mod tidy -diff
	go mod verify

.PHONY: check-generated
check-generated: install-deps
	go tool sqlc diff

.PHONY: check-vulns
check-vulns: install-deps
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
	go tool go-test-coverage --config=./.testcoverage.yml

.PHONY: build
build: ensure-build-dir install-deps
	CGO_ENABLED=1 GOFLAGS="-buildvcs=false" \
	go build -trimpath -ldflags="-s -w" -o $(BUILD_DIR)/telekilogram ./cmd

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

.PHONY: sqlc
sqlc:
	go tool sqlc generate
