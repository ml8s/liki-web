RELEASE_VERSION ?=
export RELEASE_VERSION

.DEFAULT_GOAL := help
# 统一语义（四仓一致）：
#   make check  纯静态检查（lint+格式+类型+安全；不跑测试、不构建）
#   make test   本仓全部测试（单元/集成；E2E 走 make e2e、构建走 make build）
#   make gate   check + test ＝ 源码门槛；镜像在本地构建
#   make build  本仓产物（前端+二进制）
#   make e2e    浏览器 E2E（本仓视角）
#   make smoke  本仓镜像启动 + health/page 冒烟
.PHONY: help check test gate smoke e2e e2e-a11y \
        lint-go fmt-check lint-js lint-docs typecheck test-go test-js test-python \
        check-web-static check-security check-go-version runtime-version-check python-env \
        check-actions check-caddy \
        build build-sitemap build-web build-go image sync-city-data test-go test-go-race test-go-coverage test-js-coverage \
        dev clean version

# ── 环境 ──
NODE_BIN := $(HOME)/.nvm/versions/node/v$(shell cat .nvmrc)/bin
VERSION_FILE := cmd/liki-web/VERSION
PYTHON_HOST := $(shell UV_CACHE_DIR=/tmp/uv-cache uv python find 3.12.14 2>/dev/null || command -v python3)
PYTHON_BIN := $(CURDIR)/.venv/bin/python
export PYTHON_BIN
GOPATH := $(HOME)/app/go-tools
GOBIN := $(GOPATH)/bin
GO_BIN := $(HOME)/app/go/bin/go
GO_CACHE_FALLBACK_DEFAULT := $(if $(TMPDIR),$(patsubst %/,%,$(TMPDIR)),/tmp)/liki-web-go-cache
GO_CACHE_FALLBACK := $(or $(GO_CACHE_FALLBACK),$(GO_CACHE_FALLBACK_DEFAULT))

# Never bind the build to a possibly read-only host cache. Prefer Go's normal
# cache when it is writable, including an explicit environment override; when
# it is not writable, deterministically fall back to a writable cache.
go_writable_cache = $(shell \
	preferred='$(1)'; fallback='$(2)'; \
	if [ -n "$$preferred" ]; then mkdir -p "$$preferred" 2>/dev/null || true; fi; \
	if [ -n "$$preferred" ] && [ -d "$$preferred" ] && [ -w "$$preferred" ]; then \
		printf '%s' "$$preferred"; \
	else \
		mkdir -p "$$fallback" && printf '%s' "$$fallback"; \
	fi)
GO_INPUT_MODCACHE := $(if $(filter undefined,$(origin GOMODCACHE)),$(shell $(GO_BIN) env GOMODCACHE 2>/dev/null),$(GOMODCACHE))
GO_INPUT_BUILDCACHE := $(if $(filter undefined,$(origin GOCACHE)),$(shell $(GO_BIN) env GOCACHE 2>/dev/null),$(GOCACHE))
GO_INPUT_LINTCACHE := $(if $(filter undefined,$(origin GOLANGCI_LINT_CACHE)),,$(GOLANGCI_LINT_CACHE))
GOMODCACHE := $(call go_writable_cache,$(GO_INPUT_MODCACHE),$(GO_CACHE_FALLBACK)/mod)
GOCACHE := $(call go_writable_cache,$(GO_INPUT_BUILDCACHE),$(GO_CACHE_FALLBACK)/build)
GOLANGCI_LINT_CACHE := $(call go_writable_cache,$(GO_INPUT_LINTCACHE),$(GO_CACHE_FALLBACK)/golangci-lint)
GATE_JOBS ?= $(shell nproc 2>/dev/null || echo 4)
export PATH := $(PYTHON_BIN):$(NODE_BIN):$(HOME)/app/go/bin:$(GOBIN):$(PATH)
export GOPATH GOBIN GOMODCACHE GOCACHE GOLANGCI_LINT_CACHE GO_CACHE_FALLBACK GATE_JOBS
export GOTOOLCHAIN ?= go1.26.10

# ════════════════════════════════════════════════════════════
#  用户面对（help 显示）
# ════════════════════════════════════════════════════════════

help: ## 列出所有 target
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ── 质量门槛 ──

check: lint-go fmt-check lint-js lint-docs typecheck check-web-static check-security check-caddy check-go-version runtime-version-check check-actions ## 静态检查（纯静态，不跑测试、不构建）

test: test-go test-js test-python ## 本仓测试（单元+集成）。E2E 走 make e2e；构建走 make build。
test-go-race: ## Go 竞态检测（独立于覆盖率闸门）
	go test -race -count=1 ./...

test-go-coverage: ## Run Go tests with the CI coverage policy（race 由 test-go-race 独立并行）
	go test -count=1 \
		-coverprofile=coverage.out \
		-coverpkg=./internal/... \
		./...
	@COV=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
		echo "Go coverage: $$COV%"; \
		if awk -v c="$$COV" -v m="40" 'BEGIN {exit !(c < m)}'; then \
			echo "❌ Go coverage $$COV% < 40%" >&2; exit 1; \
		fi

test-js-coverage: ## Run Vue/TS tests with V8 coverage
	cd web/app && npx vitest run --coverage


gate: ## 源码门槛：check + test
	$(MAKE) --jobs=$(GATE_JOBS) --output-sync=target check test
	$(MAKE) e2e-a11y

check-go-version: ## Require the security-supported Go toolchain
	scripts/check-go-version.sh

runtime-version-check: python-env ## 本地 node/python 必须等于 .nvmrc/.python-version（与 CI 同源）
	scripts/check-runtime-versions.sh

python-env: ## Provision the repository-local Python test environment
	@expected="$$(cat .python-version)"; \
		if [ ! -x "$(PYTHON_BIN)" ] || [ "$$($(PYTHON_BIN) -V 2>&1 | sed 's/^Python //')" != "$$expected" ]; then \
			rm -rf .venv; \
			"$(PYTHON_HOST)" -m venv .venv; \
		fi; \
		"$(PYTHON_BIN)" -m pip install --disable-pip-version-check --requirement scripts/requirements-dev.txt

check-actions: python-env ## Reject mutable reusable workflow/action references
	"$(PYTHON_BIN)" scripts/check_actions_pinned.py

smoke: image ## 本仓镜像冒烟：health + rendered homepage
	IMAGE_TAG=$(IMAGE_TAG) scripts/smoke-image.sh

# ── 调试（check / test 失败时隔离定位）──

lint-go: ## Go lint + vet（走 deps/golangci-lint.version pin，与 CI 同版本）
	scripts/lint-go.sh

fmt-check: ## Reject unformatted Go sources
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

lint-js: ## JS/TS lint（静态站点 + Vue SPA）
	cd web && PATH="$(NODE_BIN):$$PATH" npm run lint
	cd web/app && PATH="$(NODE_BIN):$$PATH" npm run lint

lint-docs: ## Markdown lint
	cd web && PATH="$(NODE_BIN):$$PATH" npm run lint:docs

typecheck: ## astro check + e2e tsc + vue-tsc
	cd web && PATH="$(NODE_BIN):$$PATH" npm run typecheck
	cd web/app && PATH="$(NODE_BIN):$$PATH" npm run typecheck

test-go: ## Go 测试
	go test -race -count=1 ./...

test-js-static: ## vitest（静态站点 web/）
	cd web && PATH="$(NODE_BIN):$$PATH" npm run test

test-js-app: ## vitest（Vue SPA web/app）
	cd web/app && PATH="$(NODE_BIN):$$PATH" npm run test

test-js: test-js-static test-js-app ## vitest（静态站点 + Vue SPA）

test-python: python-env ## Python helper tests
	"$(PYTHON_BIN)" -m unittest discover -s scripts/tests -p 'test_*.py' -q

e2e: ## Docker 全栈 E2E（不跑 lint / unit）
	scripts/e2e.sh

e2e-a11y: build ## Run the accessibility E2E contract in make gate
	scripts/e2e-a11y.sh

build: build-sitemap build-web build-go ## 构建本仓产物（前端+二进制）

# ── 镜像（本地开发，非发布路径）──
# 本地镜像 tag 由 IMAGE_TAG 指定；部署由 liki-deploy 按 sha 组装。
IMAGE_REGISTRY ?= liki-local
IMAGE_TAG ?= local

image: build ## Build the local development image
	docker build -t $(IMAGE_REGISTRY)/liki-web:$(IMAGE_TAG) .

# ── 工具 ──

dev: build ## 本地开发
	scripts/dev-start.sh

clean: ## 清理构建产物
	rm -rf bin/ web/dist/ web/app/dist/ coverage.out web/sitemap.xml web/.astro web/app/coverage test-results

version: ## bump patch（CalVer date）
	@DATE="$$(date +%Y.%m.%d)"; CURRENT=$$(cat $(VERSION_FILE) 2>/dev/null || echo 0.0.0.0); \
	case "$$CURRENT" in $$DATE.*.*) BUILD=$${CURRENT\#\#*.} ;; *) BUILD=0 ;; esac; \
	case "$$BUILD" in ''|*[!0-9]*) BUILD=0 ;; esac; \
	V="$$DATE.$$((BUILD + 1))"; echo "$$V" > $(VERSION_FILE); echo "✅ → $$V"

# ════════════════════════════════════════════════════════════
#  内部依赖（help 不显示）
# ════════════════════════════════════════════════════════════

check-web-static:
	node web/scripts/check-route-contracts.mjs
	node web/scripts/generate-route-artifacts.mjs --check
	node web/scripts/check-city-data-sync.js
	node web/scripts/generate-city-data-version.mjs --check
	node web/scripts/check-inline-scripts.js
	node web/scripts/check-design-tokens.js
	node web/scripts/check-frontend-tokens.js
	node web/scripts/check-css-vars.js
	node web/scripts/check-form-kit.js
	node web/scripts/check-performance.js
	node web/scripts/check-mobile-responsive.js
	node web/scripts/check-i18n.js
	node web/scripts/check-chart-messages.js
	node web/scripts/check-error-codes.js
	node web/scripts/check-retired-chat-transports.mjs
	node web/scripts/check-service-parity.js
	node web/scripts/check-classes.js
	node web/scripts/check-css-utility-freeze.js
	node web/scripts/check-static-sync.js

check-security:
	go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...

check-caddy:
	scripts/check-caddy.sh

sync-city-data:
	cp ../liki/engine/internal/engine/city/cities_data.json web/public/data/cities_data.json
	node web/scripts/generate-city-data-version.mjs

build-sitemap:
	node web/scripts/build-sitemap.js

build-web:
	node web/scripts/generate-city-data-version.mjs
	scripts/build-web.sh

build-go:
	scripts/build-go.sh
