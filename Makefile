# Single source of build, test, and check commands.
# Web-App Development Stack v2.0, Sections 13 and 14.
# Changes to the `check` targets are Critical changes (Change Management Standard, Section 44).

APP      ?= webapp
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS  := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)

# Local PostgreSQL administrator connection, used to create development and
# test databases. The developer's PostgreSQL role must be a superuser locally.
ADMIN_DATABASE_URL ?= postgres:///postgres?host=/var/run/postgresql
TEST_DATABASE_URL  ?= $(ADMIN_DATABASE_URL)
DEV_DB             ?= $(APP)_dev

# Tool versions, pinned at the template's first build.
SQLC_VERSION          ?= v1.31.1
GOLANGCI_LINT_VERSION ?= v2.15.0
GOVULNCHECK_VERSION   ?= v1.9.0
GITLEAKS_VERSION      ?= v8.30.1

# Development-mode environment for `make run` and `make synth`.
DEV_ENV := APP_ENV=development \
	APP_ADDR=127.0.0.1:8080 \
	BASE_URL=http://localhost:8080 \
	AUTH_MODE=dev \
	DATABASE_URL="postgres:///$(DEV_DB)?host=/var/run/postgresql" \
	DATABASE_ROLE=$(APP)_app

.PHONY: help
help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-14s %s\n",$$1,$$2}'

## ---------------------------------------------------------------- quality gates

.PHONY: check
check: fmt-check vet lint vuln secrets sqlc-check test ## Run every quality gate (required before merge)
	@echo "All quality gates passed."

.PHONY: fmt-check
fmt-check: ## Fail if any Go file is not gofmt-formatted
	@out=$$(gofmt -l cmd internal db web); if [ -n "$$out" ]; then echo "Not formatted:"; echo "$$out"; exit 1; fi

.PHONY: vet
vet: ## go vet
	go vet ./...

.PHONY: lint
lint: ## golangci-lint (staticcheck, gosec, errcheck, ...)
	golangci-lint run ./...

.PHONY: vuln
vuln: ## Known-vulnerability check
	govulncheck ./...

.PHONY: secrets
secrets: ## Secret detection over the Git history
	gitleaks detect --redact --no-banner

.PHONY: sqlc-check
sqlc-check: ## Fail if internal/dbgen is out of date with db/
	sqlc diff

.PHONY: test
test: ## All tests, including integration tests against PostgreSQL
	@if [ -z "$(TEST_DATABASE_URL)" ]; then echo "TEST_DATABASE_URL is required"; exit 1; fi
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" REQUIRE_TEST_DATABASE=1 go test -race -count=1 ./...

## ---------------------------------------------------------------- development

.PHONY: fmt
fmt: ## Format Go code
	gofmt -w cmd internal db web

.PHONY: generate
generate: ## Regenerate internal/dbgen from db/queries and db/migrations
	sqlc generate

.PHONY: tools
tools: ## Install the quality-gate tools
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
	go install github.com/zricethezav/gitleaks/v8@$(GITLEAKS_VERSION)

.PHONY: db-dev
db-dev: ## Create (or update) the local development database
	psql "$(ADMIN_DATABASE_URL)" -v ON_ERROR_STOP=1 -f db/bootstrap/001_roles.sql
	@psql "$(ADMIN_DATABASE_URL)" -tAc "SELECT 1 FROM pg_database WHERE datname = '$(DEV_DB)'" | grep -q 1 || \
		psql "$(ADMIN_DATABASE_URL)" -v ON_ERROR_STOP=1 -c 'CREATE DATABASE "$(DEV_DB)" OWNER $(APP)_owner'
	psql "postgres:///$(DEV_DB)?host=/var/run/postgresql" -v ON_ERROR_STOP=1 -f db/bootstrap/002_schema.sql
	APP_ENV=development DATABASE_URL="postgres:///$(DEV_DB)?host=/var/run/postgresql" DATABASE_ROLE=$(APP)_owner \
		go run ./cmd/$(APP) migrate up

.PHONY: synth
synth: ## Load synthetic data into the development database
	$(DEV_ENV) go run ./cmd/synthdata

.PHONY: run
run: ## Run the application locally in Development Mode
	$(DEV_ENV) go run -ldflags "$(LDFLAGS)" ./cmd/$(APP) serve

## ---------------------------------------------------------------- build

.PHONY: build
build: ## Build the release binary into bin/
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(APP) ./cmd/$(APP)
	cd bin && sha256sum $(APP) > $(APP).sha256

.PHONY: clean
clean: ## Remove build output
	rm -rf bin

## ---------------------------------------------------------------- governing documents

# docs/standards/ is a Git submodule of deltra-standards, pinned to the versions
# in force (Web-App Development Stack Section 6). Moving the pin is a Critical
# change, made only by the Approver. Agents are denied `git submodule`.
STANDARDS_URL ?=
STANDARDS_REF ?=

.PHONY: standards
standards: ## Attach docs/standards to deltra-standards (Approver; once per application)
	@test -n "$(STANDARDS_URL)" || { echo "Usage: make standards STANDARDS_URL=<deltra-standards repository URL> [STANDARDS_REF=<tag or commit>]"; exit 1; }
	@test ! -e docs/standards || { echo "docs/standards already exists"; exit 1; }
	git submodule add $(STANDARDS_URL) docs/standards
	@if [ -n "$(STANDARDS_REF)" ]; then git -C docs/standards checkout -q $(STANDARDS_REF) && git add docs/standards; fi
	@echo "Pinned docs/standards at $$(git -C docs/standards rev-parse --short HEAD). Commit it, and record the version in the decision log."

.PHONY: standards-pin
standards-pin: ## Move docs/standards to another deltra-standards version (Approver; Critical change)
	@test -n "$(STANDARDS_REF)" || { echo "Usage: make standards-pin STANDARDS_REF=<tag or commit>"; exit 1; }
	git -C docs/standards fetch -q origin
	git -C docs/standards checkout -q $(STANDARDS_REF)
	git add docs/standards
	@echo "docs/standards now at $(STANDARDS_REF). Run the doc-consistency-reviewer, then commit."

## ---------------------------------------------------------------- new application

# Usage: make init NAME=procurement [MODULE=github.com/deltra/procurement]
# Run once, immediately after copying the template. NAME: lowercase letters only.
.PHONY: init
init: ## Rename the template for a new application
	@echo "$(NAME)" | grep -Eq '^[a-z]+$$' || { echo "NAME must be lowercase letters only"; exit 1; }
	@test -d cmd/webapp || { echo "Template already initialized"; exit 1; }
	@test -f go.sum -a -d internal/dbgen || { echo "The template's first build is not done. Run it in deltra-webapp-template first (README Section 2, \"First build of the template itself\")."; exit 1; }
	git mv cmd/webapp cmd/$(NAME)
	git mv deploy/webapp.service deploy/$(NAME).service
	git mv deploy/nginx-webapp.conf deploy/nginx-$(NAME).conf
	git mv deploy/webapp.env.example deploy/$(NAME).env.example
	grep -rlI --exclude-dir=.git --exclude=Makefile 'webapp' . | xargs sed -i \
		-e 's/deltra-webapp-template/@@TEMPLATE@@/g' -e 's/webapp/$(NAME)/g' -e 's/@@TEMPLATE@@/deltra-webapp-template/g'
	grep -rlI --exclude-dir=.git --exclude=Makefile 'Webapp' . | xargs -r sed -i "s/Webapp/$$(echo $(NAME) | sed 's/./\U&/')/g"
	sed -i 's/^APP      ?= webapp/APP      ?= $(NAME)/' Makefile
	@if [ -n "$(MODULE)" ]; then \
		go mod edit -module $(MODULE); \
		grep -rlI --exclude-dir=.git '"deltra/$(NAME)' . | xargs -r sed -i 's#"deltra/$(NAME)#"$(MODULE)#g'; \
	fi
	@echo "Initialized $(NAME). Next: make standards STANDARDS_URL=... STANDARDS_REF=<tag>; then make check; then add the SRS and fill in CLAUDE.md (README Section 3)"
