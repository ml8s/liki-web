.DEFAULT_GOAL := help
# 统一语义（四仓一致）：
#   make check  纯静态契约检查（lint+格式+类型+安全+skills；不跑测试、不构建）
#   make test   本仓全部测试（单元/集成/skill 验证；E2E 走 make e2e、构建走 make build）
#   make gate   check + test + image smoke ＝ 推送门槛
#   make build  本仓产物（前端+二进制；镜像由 CI release 构建）
#   make image  本地镜像（调试用，非发布路径）
#   make e2e    浏览器 E2E（本仓视角）
#   make smoke  本仓镜像启动 + health/page 冒烟
.PHONY: help check test gate smoke smoke-deploy e2e \
        lint-go fmt-check lint-js lint-docs typecheck test-go test-js test-python test-skills \
        check-html check-security \
        build build-skills build-sitemap build-web build-go image \
        dev clean hooks version version-minor version-major

# ── 环境 ──
NODE_BIN := $(HOME)/.nvm/versions/node/v$(shell cat .nvmrc)/bin
VERSION_FILE := cmd/liki-web/VERSION
PYTHON_BIN := $(HOME)/app/python3.12.14/bin
export PATH := $(PYTHON_BIN):$(NODE_BIN):$(HOME)/app/go/bin:$(HOME)/app/go-tools/bin:$(HOME)/go/bin:$(PATH)
export GOCACHE ?= /tmp/gocache
export GOLANGCI_LINT_CACHE ?= /tmp/golangci-lint-cache
export GOTOOLCHAIN ?= go1.26.9

# deployed smoke URL（make smoke-deploy URL=...）
DOMAIN_US = $(shell /bin/grep -oP '^DOMAIN_US=\K.*' .env 2>/dev/null || echo "liki.hk")
DOMAIN_CN = $(shell /bin/grep -oP '^DOMAIN_CN=\K.*' .env 2>/dev/null || echo "liki.hk")
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

gate: check test smoke ## 部署门槛：check + test + image smoke

check-actions: ## Reject mutable reusable workflow/action references
	python3 scripts/check_actions_pinned.py

smoke: image ## 本仓镜像冒烟：health + rendered homepage
	IMAGE_TAG=$(IMAGE_TAG) scripts/smoke-image.sh

smoke-deploy: ## 部署后冒烟验证（make smoke-deploy URL=...）
	scripts/smoke.sh $(URL)

# ── 调试（check / test 失败时隔离定位）──

lint-go: ## Go lint + vet
	golangci-lint run --timeout=3m
	go vet ./...

fmt-check: ## Reject unformatted Go sources
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

lint-js: ## JS lint
	cd web && PATH="$(NODE_BIN):$$PATH" npx eslint js/
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

build: build-skills build-sitemap build-web build-go ## 构建本仓产物（前端+二进制）。镜像由 CI release 构建，非本地。

# ── 镜像（本地调试用，非发布路径）──
# 生产镜像由 CI release 构建并推送 GHCR；此目标仅用于本地构建/调试。
IMAGE_REGISTRY ?= ghcr.io/ml8s
IMAGE_TAG ?= dev

image: build ## Build liki-web image locally (debug; production uses CI release)
	docker build -t $(IMAGE_REGISTRY)/liki-web:$(IMAGE_TAG) .

# ── 工具 ──

dev: build ## 本地开发
	scripts/dev-start.sh

clean: ## 清理构建产物
	rm -rf bin/ web/dist/ web/app/dist/ coverage.out

hooks: ## 安装 git hooks
	git config core.hooksPath .githooks

version: ## bump patch（CalVer date）
	@V="$$(date +%Y.%m.%d).0"; echo "$$V" > $(VERSION_FILE); echo "✅ → $$V"

version-minor: ## bump minor
	@V=$$(cat $(VERSION_FILE)); NEXT=$$(printf '%s' "$$V" | awk -F. 'NF == 4 && $$1 ~ /^[0-9]+$$/ && $$2 ~ /^[0-9]+$$/ {print $$1 "." $$2 + 1 ".0.0"; ok=1} END {exit !ok}'); \
	[ -n "$$NEXT" ] || { echo "❌ Expected CalVer version: YYYY.MM.PP.BUILD" >&2; exit 1; }; \
	echo "$$NEXT" > $(VERSION_FILE); echo "✅ $$V → $$NEXT"

version-major: ## bump major
	@V=$$(cat $(VERSION_FILE)); NEXT=$$(printf '%s' "$$V" | awk -F. 'NF == 4 && $$1 ~ /^[0-9]+$$/ {print $$1 + 1 ".0.0.0"; ok=1} END {exit !ok}'); \
	[ -n "$$NEXT" ] || { echo "❌ Expected CalVer version: YYYY.MM.PP.BUILD" >&2; exit 1; }; \
	echo "$$NEXT" > $(VERSION_FILE); echo "✅ $$V → $$NEXT"

# ════════════════════════════════════════════════════════════
#  内部依赖（help 不显示）
# ════════════════════════════════════════════════════════════

check-html:
	node web/scripts/check-inline-scripts.cjs
	node web/scripts/check-design-tokens.cjs
	node web/scripts/check-performance.cjs
	node web/scripts/check-mobile-responsive.cjs
	node web/scripts/check-html.cjs
	node web/scripts/check-i18n.cjs
	node web/scripts/check-classes.cjs
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
