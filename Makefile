.DEFAULT_GOAL := help
.PHONY: help check test gate deploy-us deploy-cn smoke e2e \
        lint-go lint-js typecheck test-go test-js test-python test-skills \
        check-html check-security \
        build build-skills build-sitemap build-web build-go build-docker \
        dev clean hooks version version-minor version-major

# ── 环境 ──
NODE_BIN := $(HOME)/.nvm/versions/node/v$(shell cat .nvmrc)/bin
SKILLS_DIR := $(if $(wildcard ../liki/scripts/build-archive.sh),../liki,skills-repo)
VERSION_FILE := cmd/liki/VERSION
SKILLS := liki
export PATH := $(HOME)/app/go/bin:$(HOME)/app/go-tools/bin:$(HOME)/go/bin:$(PATH)
export GOCACHE ?= /tmp/gocache
export GOLANGCI_LINT_CACHE ?= /tmp/golangci-lint-cache

# smoke URL（make smoke 或 make smoke to=us / to=cn）
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

check: lint-go lint-js typecheck check-html check-security check-caddy check-skills check-version ## 静态检查 + 安全扫描（~3min）

test: test-go test-js test-python e2e build check-packaging ## 全量测试（~15min，含 E2E）

gate: check test ## 部署门槛：check + test 全过才能部署

# ── 部署 ──

deploy-us: ## 部署 US（需先 make gate）
	scripts/deploy.sh us

deploy-cn: ## 部署 CN（需先 make gate）
	scripts/deploy.sh cn

smoke: ## 部署后冒烟验证（make smoke 或 make smoke to=us）
	scripts/smoke.sh $(URL)

# ── 调试（check / test 失败时隔离定位）──

lint-go: ## Go lint + vet
	golangci-lint run --timeout=3m
	go vet ./...

lint-js: ## JS lint
	cd web && PATH="$(NODE_BIN):$$PATH" npx eslint js/

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
	cd $(SKILLS_DIR) && python3 -m pytest tests/ -q --ignore=tests/test_integration.py

e2e: ## Docker 全栈 E2E（不跑 lint / unit）
	scripts/e2e.sh

build: build-skills build-sitemap build-web build-go ## 完整构建

# ── 工具 ──

dev: build ## 本地开发
	scripts/dev-start.sh

clean: ## 清理构建产物
	rm -rf bin/ web/dist/ web/app/dist/ coverage.out

hooks: ## 安装 git hooks
	git config core.hooksPath .githooks

version: ## bump patch（CalVer 日期）
	@V=$$(date +%Y.%m.%d); echo "$$V" > $(VERSION_FILE); echo "✅ → $$V"

version-minor: ## bump minor
	@V=$$(cat $(VERSION_FILE)); MAJOR=$${V%%.*}; REST=$${V#*.}; MINOR=$${REST%.*}; \
	echo "$$MAJOR.$$((MINOR + 1)).0" > $(VERSION_FILE); echo "✅ $$V → $$(cat $(VERSION_FILE))"

version-major: ## bump major
	@V=$$(cat $(VERSION_FILE)); MAJOR=$${V%%.*}; \
	echo "$$((MAJOR + 1)).0.0" > $(VERSION_FILE); echo "✅ $$V → $$(cat $(VERSION_FILE))"

# ════════════════════════════════════════════════════════════
#  内部依赖（help 不显示）
# ════════════════════════════════════════════════════════════

check-html:
	node web/scripts/check-html.cjs
	node web/scripts/check-i18n.cjs
	node web/scripts/check-classes.cjs
	node web/scripts/check-static-sync.cjs

check-security:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	- trivy fs --severity HIGH,CRITICAL --exit-code 1 . 2>/dev/null || echo "⚠ trivy not found, skipping"

check-skills:
	scripts/check-skills.sh

check-version:
	@test -f $(VERSION_FILE) || { echo "❌ Missing VERSION"; exit 1; }

check-caddy:
	scripts/check-caddy.sh

build-skills:
	test -f "$(SKILLS_DIR)/scripts/build-archive.sh" || { echo "❌ skills source not found: $(SKILLS_DIR)" >&2; exit 1; }
	cd $(SKILLS_DIR) && scripts/build-archive.sh
	SKILLS_REPO_ROOT=$(SKILLS_DIR) scripts/sync-skills.sh

build-sitemap:
	node web/scripts/build-sitemap.cjs

build-web:
	scripts/build-web.sh

build-go:
	scripts/build-go.sh

check-packaging:
	scripts/check-packaging.sh


build-docker:
	scripts/build-docker.sh --compose-file deploy/liki/docker-compose.yml \
		--export /tmp/liki_images.tar.gz --services web engine
