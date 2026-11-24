.DEFAULT_GOAL := check

BUILD_DIR ?= ./dist

ACTIONLINT := bunx --minimum-release-age 259200 github-actionlint

.PHONY: install-deps
install-deps:
	go mod download

.PHONY: lint
lint:
	bun fmt --check
	golangci-lint fmt --diff
	golangci-lint run

.PHONY: lint-fix
lint-fix:
	bun fmt
	golangci-lint fmt
	golangci-lint run --fix

.PHONY: check-hooks
check-hooks:
	bunx --minimum-release-age 259200 lefthook validate

.PHONY: build
build:
	CGO_ENABLED=1 GOFLAGS="-buildvcs=false" \
	go build -trimpath -ldflags="-s -w" -o "$(BUILD_DIR)/telekilogram" ./cmd

.PHONY: check-generated
check-generated: install-deps
	go tool sqlc diff

.PHONY: check-deps
check-deps: install-deps
	go mod tidy -diff
	go mod verify

.PHONY: check-vulns
check-vulns: install-deps
	go tool govulncheck ./...

.PHONY: test
test:
	mkdir -p -- "$(BUILD_DIR)"
	go test \
		-race \
		-coverprofile="$(BUILD_DIR)/coverage.out" \
		-covermode=atomic \
		-coverpkg=./... \
		./...

.PHONY: check-coverage
check-coverage: test
	go tool go-test-coverage --config=./.testcoverage.yml --profile="$(BUILD_DIR)/coverage.out"

.PHONY: check-workflows
check-workflows:
	$(ACTIONLINT)

.PHONY: check
check: lint check-hooks build check-generated check-deps check-vulns check-coverage check-workflows

.PHONY: check-fix
check-fix: lint-fix
	$(MAKE) check

.PHONY: clean
clean:
	test -z "$(BUILD_DIR)" || rm -f -- "$(BUILD_DIR)/telekilogram" "$(BUILD_DIR)/coverage.out"

.PHONY: sqlc
sqlc:
	go tool sqlc generate
