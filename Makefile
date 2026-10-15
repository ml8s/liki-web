RELEASE_VERSION ?=
export RELEASE_VERSION

.DEFAULT_GOAL := help
# 统一语义（四仓一致）：
#   make check  纯静态检查（lint+格式+类型+安全+skills；不跑测试、不构建）
#   make test   本仓全部测试（单元/集成/skill 验证；E2E 走 make e2e、构建走 make build）
#   make gate   check + test ＝ 源码门槛；镜像冒烟只在 candidate/release 流程执行
#   make build  本仓产物（前端+二进制）
#   make candidate RELEASE_VERSION=... 本地构建并冒烟 candidate，通过后推送并记录 exact digest
#   make e2e    浏览器 E2E（本仓视角）
#   make smoke  本仓镜像启动 + health/page 冒烟
.PHONY: help check test gate smoke e2e \
        lint-go fmt-check lint-js lint-docs typecheck test-go test-js test-python test-skills \
        check-html check-security \
        build build-skills build-sitemap build-web build-go image candidate test-go-coverage test-js-coverage \
        dev clean hooks version

# ── 环境 ──
NODE_BIN := $(HOME)/.nvm/versions/node/v$(shell cat .nvmrc)/bin
VERSION_FILE := cmd/liki-web/VERSION
PYTHON_BIN := $(HOME)/app/python3.12.14/bin
export PATH := $(PYTHON_BIN):$(NODE_BIN):$(HOME)/app/go/bin:$(HOME)/app/go-tools/bin:$(HOME)/go/bin:$(PATH)
export GOCACHE ?= /tmp/gocache
export GOLANGCI_LINT_CACHE ?= /tmp/golangci-lint-cache
export GOTOOLCHAIN ?= go1.26.9

DOMAIN_US = $(shell /bin/grep -oP '^LIKI_DOMAIN_US=\K.*' .env 2>/dev/null || echo "liki.hk")
DOMAIN_CN = $(shell /bin/grep -oP '^LIKI_DOMAIN_CN=\K.*' .env 2>/dev/null || echo "liki.hk")
ifeq ($(to),us)
  URL := https://$(DOMAIN_US)
else ifeq ($(to),cn)
  URL := https://$(DOMAIN_CN)
else
  URL ?= http://localhost:8080
endif

# ════════════════════════════════════════════════════════════
#  用户面对（help 显示）
# ════════════════════════════════════════════════════════════

help: ## 列出所有 target
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ── 质量门槛 ──

check: lint-go fmt-check lint-js lint-docs typecheck check-html check-security check-caddy check-skills check-version check-actions ## 静态检查（纯静态，不跑测试、不构建）

test: test-go test-js test-python test-skills ## 本仓测试（单元+集成+skill 验证）。E2E 走 make e2e；构建走 make build。
test-go-coverage: ## Run Go tests with the CI coverage policy
	go test -race -count=1 \
		-coverprofile=coverage.out \
		-coverpkg=./internal/... \
		./...
	@COV=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
		echo "Go coverage: $$COV%"; \
		if awk -v c="$$COV" -v m="65" 'BEGIN {exit !(c < m)}'; then \
			echo "❌ Go coverage $$COV% < 65%" >&2; exit 1; \
		fi

test-js-coverage: ## Run Vue/TS tests with V8 coverage
	cd web/app && npx vitest run --coverage


gate: check test ## 源码门槛：check + test；镜像冒烟只在 candidate/release 流程执行

check-actions: ## Reject mutable reusable workflow/action references
	python3 scripts/check_actions_pinned.py

smoke: image ## 本仓镜像冒烟：health + rendered homepage
	IMAGE_TAG=$(IMAGE_TAG) scripts/smoke-image.sh

candidate: ## 从干净 worktree 构建、本地冒烟，并记录 exact digest（不推 GHCR）
	scripts/build-candidate.sh

candidate-publish: ## 整机 candidate-test 通过后，将同一 digest 推送到 GHCR
	scripts/publish-candidate.sh

# ── 调试（check / test 失败时隔离定位）──

lint-go: ## Go lint + vet
	golangci-lint run --timeout=3m
	go vet ./...

fmt-check: ## Reject unformatted Go sources
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

lint-js: ## JS lint
	cd web/app && PATH="$(NODE_BIN):$$PATH" npx eslint src/

lint-docs: ## Markdown lint
	cd web && PATH="$(NODE_BIN):$$PATH" npm run lint:docs

typecheck: ## vue-tsc
	cd web/app && PATH="$(NODE_BIN):$$PATH" npx vue-tsc --noEmit

test-go: ## Go 测试
	go test -race -count=1 ./...

test-js: ## vitest（legacy + Vue SPA）
	cd web && PATH="$(NODE_BIN):$$PATH" npx vitest run test/
	cd web/app && PATH="$(NODE_BIN):$$PATH" npm run test

test-python: ## Python helper tests
	python3 -m unittest discover -s web/test -p '*_test.py' -q
	python3 -m unittest discover -s scripts/tests -p 'test_*.py' -q

test-skills: ## Python skill 测试
	scripts/verify-prepared-skills.sh

e2e: ## Docker 全栈 E2E（不跑 lint / unit）
	scripts/e2e.sh

build: build-skills build-sitemap build-web build-go ## 构建本仓产物（前端+二进制）

# ── 镜像（本地候选，非发布路径）──
# 本地开发镜像使用 local tag；发版 candidate 由 make candidate 构建并推送。
IMAGE_REGISTRY ?= ghcr.io/ml8s
IMAGE_TAG ?= local

image: build ## Build the local development image
	docker build -t $(IMAGE_REGISTRY)/liki-web:$(IMAGE_TAG) .

# ── 工具 ──

dev: build ## 本地开发
	scripts/dev-start.sh

clean: ## 清理构建产物
	rm -rf bin/ web/dist/ web/app/dist/ coverage.out

hooks: ## 安装 git hooks
	git config core.hooksPath .githooks

version: ## bump patch（CalVer date）
	@DATE="$$(date +%Y.%m.%d)"; CURRENT=$$(cat $(VERSION_FILE) 2>/dev/null || echo 0.0.0.0); \
	case "$$CURRENT" in $$DATE.*.*) BUILD=$${CURRENT\#\#*.} ;; *) BUILD=0 ;; esac; \
	case "$$BUILD" in ''|*[!0-9]*) BUILD=0 ;; esac; \
	V="$$DATE.$$((BUILD + 1))"; echo "$$V" > $(VERSION_FILE); echo "✅ → $$V"

# ════════════════════════════════════════════════════════════
#  内部依赖（help 不显示）
# ════════════════════════════════════════════════════════════

check-html:
	node web/scripts/check-inline-scripts.cjs
	node web/scripts/check-design-tokens.cjs
	node web/scripts/check-frontend-tokens.cjs
	node web/scripts/check-css-vars.cjs
	node web/scripts/check-css-contracts.cjs
	node web/scripts/check-performance.cjs
	node web/scripts/check-mobile-responsive.cjs
	node web/scripts/check-html.cjs
	node web/scripts/check-i18n.cjs
	node web/scripts/check-service-parity.cjs
	node web/scripts/check-classes.cjs
	node web/scripts/check-css-utility-freeze.cjs
	node web/scripts/check-static-sync.cjs

check-security:
	go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...

check-skills:
	scripts/check-skills.sh

check-version:
	@test -f $(VERSION_FILE) || { echo "❌ Missing VERSION"; exit 1; }

check-caddy:
	scripts/check-caddy.sh

build-skills:
	scripts/verify-prepared-skills.sh

build-sitemap:
	node web/scripts/build-sitemap.cjs

build-web:
	scripts/build-web.sh

build-go:
	scripts/build-go.sh

check-packaging:
	scripts/check-packaging.sh
