.PHONY: help build test lint vet clean check check-html check-caddy check-skills check-version check-packaging
.PHONY: dev hooks
.PHONY: test-js test-js-watch test-db test-api test-ui test-remote test-all
.PHONY: deploy deploy-all deploy-us deploy-cn
.PHONY: test-setup
.PHONY: sync-skills build-archive build-web build-go build-docker build-sitemap
.PHONY: version

# Node.js binary path — driven by .nvmrc (single source of truth)
NODE_BIN := $(HOME)/.nvm/versions/node/v$(shell cat .nvmrc)/bin

# skills 仓根（ml8s/liki 的 clone 目录，与仓库名一致）
SKILLS_DIR := ../liki

# Go / lint tools may live outside the default login PATH.
export PATH := $(HOME)/app/go/bin:$(HOME)/app/go-tools/bin:$(HOME)/go/bin:$(PATH)
export GOCACHE ?= /tmp/gocache
export GOLANGCI_LINT_CACHE ?= /tmp/golangci-lint-cache

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ── Build (leaf targets) ──

sync-skills: ## Sync 4 sub-skills from ../liki/skills/ to web/skills/
	scripts/sync-skills.sh

build-archive: ## Build liki skill archive + sync to web/skills/
	cd $(SKILLS_DIR) && scripts/build-archive.sh
	make sync-skills

build-sitemap: ## Generate sitemap.xml from Astro pages
	node web/scripts/build-sitemap.cjs

build-web: ## Build Astro static pages
	scripts/build-web.sh

build-go: ## Build Go binary
	scripts/build-go.sh

build-docker: ## Build Docker images (requires Docker)
	scripts/build-docker.sh --compose-file deploy/liki/docker-compose.yml --export /tmp/liki_images.tar.gz --services web engine

build: build-archive build-sitemap build-web build-go ## Full build (skills + sitemap + web + Go)

# ── Dev ──

dev: build-go build-web ## Start local dev environment (Go + engine + Caddy)
	scripts/dev-start.sh

hooks: ## Set git hooks path
	git config core.hooksPath .githooks

clean: ## Remove build artifacts
	rm -rf bin/

# ── Test ──
# to=us|cn 指定远程目标，缺省为本地

DOMAIN_US = $(shell /bin/grep -oP '^DOMAIN_US=\K.*' .env 2>/dev/null || echo "liki.hk")
DOMAIN_CN = $(shell /bin/grep -oP '^DOMAIN_CN=\K.*' .env 2>/dev/null || echo "liki.hk")

ifeq ($(to),us)
  URL := https://$(DOMAIN_US)
else ifeq ($(to),cn)
  URL := https://$(DOMAIN_CN)
else
  URL ?= http://localhost:8080
endif

# ── Layer 0: 静态检查 ──

# ── Layer 1: 单元测试（无依赖）──

test: ## Run Go unit tests (all packages)
	go test -race -count=1 ./...

test-js: ## Run vitest JS tests
	cd web && PATH="$(NODE_BIN):$$PATH" npx vitest run test/
	cd web/app && PATH="$(NODE_BIN):$$PATH" npm run test

test-js-watch: ## Run vitest in watch mode
	cd web && PATH="$(NODE_BIN):$$PATH" npx vitest test/

test-skills: ## Run skill python tests (rules engine: factors/duanyu/agent_cli)——前置无服务，排除 integration（Docker 段全跑）
	@cd $(SKILLS_DIR) && python3 -m pytest tests/ -q --ignore=tests/test_integration.py

test-setup: ## Install web dependencies (npm install)
	cd web && npm install

# ── Layer 2: 集成测试（文件 DB，不启服务）──

test-db: ## Run DB integration tests
	go test -tags integration -count=1 -timeout 30s ./internal/http/
	go test -tags integration -count=1 -timeout 30s -run "^(TestSharedPayment|TestRuntime)" ./internal/payment/

# ── Layer 3+4: 服务测试（需 $URL）──

test-api: ## Run API smoke tests against $URL
	scripts/test-api.sh $(URL) $(or $(API_URL),$(URL))

test-ui: ## Run Playwright E2E tests against $URL
	scripts/test-ui.sh $(URL)

# ── 远程验证 ──

test-remote: ## Run test-api against remote $URL (E2E 需在本地/CI 运行)
	@[ -n "$(URL)" ] || { echo "Usage: make test-remote URL=https://liki.hk"; exit 1; }
	$(MAKE) test-api URL=$(URL)

# ── 全量流水线 ──

test-all: lint vet check-html check-caddy check-skills check-packaging check-version build test-js test-db ## Full CI pipeline
	@echo "=== coverage ==="
	@go test -race -count=1 -coverprofile=coverage.out ./... -coverpkg=./internal/...
	@COV=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	echo "Coverage: $${COV}%"; \
	if [ "$$(echo "$${COV} < 50" | bc)" -eq 1 ]; then echo "❌ Coverage < 50%"; exit 1; fi
	@echo ""
	@echo "=== govulncheck ==="
	@go run golang.org/x/vuln/cmd/govulncheck@latest ./... 2>&1 | tail -5 || true
	@echo ""
	@echo "=== Docker E2E ==="
	@scripts/docker-test.sh up || { scripts/docker-test.sh down 2>/dev/null; echo "❌ Docker 环境启动失败"; exit 1; }; \
	$(MAKE) test-api URL=http://localhost:8080 API_URL=http://localhost:8080; RC=$$?; \
	$(MAKE) test-ui URL=http://localhost:8080; RC2=$$?; \
	cd $(SKILLS_DIR) && LIKI_RPC_URL=http://localhost:8080/jsonrpc python3 -m pytest tests/ -q; RC3=$$?; \
	cd ../liki-web && scripts/docker-test.sh down; \
	[ $$RC -eq 0 ] && [ $$RC2 -eq 0 ] && [ $$RC3 -eq 0 ] || { echo "❌ E2E 失败（test-api:$$RC test-ui:$$RC2 skill-pytest:$$RC3）"; exit 1; }
	@echo "✅ test-all 全部通过"

# ── Lint / Check ──

lint:
	golangci-lint run --timeout=3m
	cd web && npx eslint js/

vet:
	go vet ./...

check-html:
	node web/scripts/build-sitemap.cjs
	node web/scripts/check-html.cjs
	node web/scripts/check-i18n.cjs
	node web/scripts/check-classes.cjs
	node web/scripts/check-static-sync.cjs

SKILLS := liki-bazi liki-divination liki-fengshui liki-naming

check-skills: build-archive sync-skills
	@for skill in $(SKILLS); do \
	  test -f "web/skills/$$skill/SKILL.md" || { echo "❌ Missing web/skills/$$skill/SKILL.md"; exit 1; }; \
	done
	@echo "✓ All $(words $(SKILLS)) skills present"
	@test -f "web/.well-known/agent-skills/index.json" || { echo "❌ Missing .well-known/agent-skills/index.json"; exit 1; }
	@echo "✓ V2 index.json exists"
	@python3 scripts/check-index-json.py
	@test -f "web/skills/liki-bazi.tar.gz" || { echo "❌ Missing skills/liki-bazi.tar.gz (run: make build-archive)"; exit 1; }
	@echo "✓ V2 archive exists"

check-version:
	@test -f cmd/liki/VERSION || { echo "❌ Missing cmd/liki/VERSION"; exit 1; }
	@echo "✓ VERSION file: $$(cat cmd/liki/VERSION)"

VERSION_FILE := cmd/liki/VERSION

version: ## 写入今日日期（CalVer；里程碑时 git tag -a <日期>）
	@V=$$(date +%Y.%m.%d); echo "$$V" > $(VERSION_FILE); echo "✅ → $$V"

version-minor: ## Bump MINOR (0.14.0 → 0.15.0)
	@V=$$(cat $(VERSION_FILE)); \
	MAJOR=$${V%%.*}; \
	REST=$${V#*.}; \
	MINOR=$${REST%.*}; \
	echo "$$MAJOR.$$((MINOR + 1)).0" > $(VERSION_FILE); \
	echo "✅ $$V → $$(cat $(VERSION_FILE))"

version-major: ## Bump MAJOR (0.14.0 → 1.0.0)
	@V=$$(cat $(VERSION_FILE)); \
	MAJOR=$${V%%.*}; \
	echo "$$((MAJOR + 1)).0.0" > $(VERSION_FILE); \
	echo "✅ $$V → $$(cat $(VERSION_FILE))"

check-caddy:
	@if command -v caddy >/dev/null 2>&1; then \
	  caddy fmt deploy/liki/Caddyfile --diff 2>&1 || true; \
	  caddy fmt deploy/liki/Caddyfile.local --diff 2>&1 || true; \
	  echo "✓ Caddyfiles parsed"; \
	fi

check-packaging:
	@echo "=== 模拟部署打包 ==="
	@TMP=$$(mktemp -d); \
	cp -rL "web/." "$$TMP/"; \
	echo -n "  文件数: "; find "$$TMP" -type f | wc -l; \
	test -f "$$TMP/.well-known/agent-skills/index.json" || { echo "❌ 打包后缺失 .well-known/agent-skills/index.json"; rm -rf "$$TMP"; exit 1; }; \
	test -f "$$TMP/skills/liki-bazi.tar.gz" || { echo "❌ 打包后缺失 skills/liki-bazi.tar.gz"; rm -rf "$$TMP"; exit 1; }; \
	for skill in $(SKILLS); do \
	  test -f "$$TMP/skills/$$skill/SKILL.md" || { echo "❌ 打包后缺失 skills/$$skill/SKILL.md"; rm -rf "$$TMP"; exit 1; }; \
	done; \
	echo "✓ .well-known + archive + skills 全部存在"; \
	rm -rf "$$TMP"

check: lint vet check-html check-caddy check-skills check-packaging check-version test

# ── Deploy ──

SERVER_US ?= 43.130.2.209
SERVER_CN ?= 120.79.194.247
SERVER_USER_US := ubuntu
SERVER_USER_CN := root

update-engine: ## 更新引擎源码（liki 拉最新 master——engine 现随 ml8s/liki 仓库分发，部署/测试前必做）
	cd $(SKILLS_DIR) && git pull --ff-only origin master

deploy: deploy-us ## Alias: 默认部署到 US（同 deploy-us）

deploy-us: update-engine build build-docker
	@test -f .env || { echo "❌ .env 不存在，请先创建 .env 文件"; exit 1; }
	scripts/deploy-remote.sh --server $(SERVER_US) --user $(SERVER_USER_US) --domain $(DOMAIN_US) --image /tmp/liki_images.tar.gz --env .env
	rm -f /tmp/liki_images.tar.gz

deploy-cn: update-engine build build-docker
	@test -f .env || { echo "❌ .env 不存在，请先创建 .env 文件"; exit 1; }
	scripts/deploy-remote.sh --server $(SERVER_CN) --user $(SERVER_USER_CN) --domain $(DOMAIN_CN) --image /tmp/liki_images.tar.gz --env .env
	rm -f /tmp/liki_images.tar.gz

deploy-all: build build-docker
	@test -f .env || { echo "❌ .env 不存在，请先创建 .env 文件"; exit 1; }
	scripts/deploy-remote.sh --server $(SERVER_US) --user $(SERVER_USER_US) --domain $(DOMAIN_US) --image /tmp/liki_images.tar.gz --env .env
	scripts/deploy-remote.sh --server $(SERVER_CN) --user $(SERVER_USER_CN) --domain $(DOMAIN_CN) --image /tmp/liki_images.tar.gz --env .env
	rm -f /tmp/liki_images.tar.gz
