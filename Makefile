DB_NAME      ?= nutrition_dev
DB_HOST      ?= localhost
DB_PORT      ?= 5432
DB_USER      ?= $(USER)
DB_DSN       ?= postgres://$(DB_USER)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable
CORE_ADDR    ?= :8086
# Стабильный dev-секрет: чтобы JWT-токены пережили рестарт core.
# В проде ОБЯЗАТЕЛЬНО переопределить (см. DEPLOY.md).
CORE_JWT_SECRET ?= dev-stable-secret-do-not-use-in-prod
PRICE_URL    ?=

PIDS_DIR     := .pids

.PHONY: help db-create db-drop db-up db-status loader-install smoke \
        core core-bg web web-bg dev stop test test-core test-web \
        test-web-unit test-e2e bench verify \
        price-service price-service-bg

help:
	@echo "Targets:"
	@echo "  db-create        createdb $(DB_NAME)"
	@echo "  db-drop          dropdb $(DB_NAME) (drops smoke data!)"
	@echo "  db-up            goose migrate up"
	@echo "  db-status        goose status"
	@echo "  loader-install   set up Python venv for loader"
	@echo "  smoke            create DB + migrate + load smoke CSVs"
	@echo "  core             run core in foreground"
	@echo "  core-bg          run core in background → $(PIDS_DIR)/core.pid"
	@echo "  web              run vite dev (foreground)"
	@echo "  web-bg           run vite dev in background"
	@echo "  dev              core-bg + web-bg + tail logs"
	@echo "  stop             stop all background processes"
	@echo "  test             go test + tsc"
	@echo "  test-core        go test ./... (с DATABASE_DSN — integration)"
	@echo "  test-web         tsc check"
	@echo "  test-web-unit    vitest (web unit tests)"
	@echo "  test-e2e         playwright (требует core на :8086)"
	@echo "  bench            go bench планировщика"
	@echo "  verify           полный приёмочный прогон (core+web+e2e+bench)"
	@echo "  price-service-bg run price-service on :8085 (requires Edadil access)"
	@echo
	@echo "  DB_DSN=$(DB_DSN)"

db-create:
	createdb -h $(DB_HOST) -p $(DB_PORT) $(DB_NAME) || true

db-drop:
	dropdb --if-exists -h $(DB_HOST) -p $(DB_PORT) $(DB_NAME)

db-up:
	cd core && DB_DSN="$(DB_DSN)" $(MAKE) db-up

db-status:
	cd core && DB_DSN="$(DB_DSN)" $(MAKE) db-status

loader-install:
	cd loader && python3 -m venv .venv && \
		. .venv/bin/activate && pip install -q -e .

smoke: db-create db-up
	cd loader && [ -d .venv ] || ($(MAKE) -C .. loader-install)
	cd loader && . .venv/bin/activate && \
		DATABASE_DSN="$(DB_DSN)" loader load-all --data-dir data/smoke

core:
	cd core && DATABASE_DSN="$(DB_DSN)" CORE_HTTP_ADDR=$(CORE_ADDR) \
		CORE_JWT_SECRET="$(CORE_JWT_SECRET)" \
		$(if $(PRICE_URL),PRICE_SERVICE_URL=$(PRICE_URL),) \
		go run ./cmd/server

core-bg:
	@mkdir -p $(PIDS_DIR)
	@(cd core && DATABASE_DSN="$(DB_DSN)" CORE_HTTP_ADDR=$(CORE_ADDR) \
		CORE_JWT_SECRET="$(CORE_JWT_SECRET)" \
		$(if $(PRICE_URL),PRICE_SERVICE_URL=$(PRICE_URL),) \
		go run ./cmd/server) > $(PIDS_DIR)/core.log 2>&1 & \
		echo $$! > $(PIDS_DIR)/core.pid
	@echo "core → http://localhost$(CORE_ADDR) (log: $(PIDS_DIR)/core.log)"

web:
	cd web && npm run dev

web-bg:
	@mkdir -p $(PIDS_DIR)
	@(cd web && npm run dev) > $(PIDS_DIR)/web.log 2>&1 & \
		echo $$! > $(PIDS_DIR)/web.pid
	@echo "web  → http://localhost:5173    (log: $(PIDS_DIR)/web.log)"

dev: core-bg web-bg
	@echo
	@echo "→ open http://localhost:5173 in browser"
	@echo "→ make stop  — stop background processes"
	@echo "→ tail -f $(PIDS_DIR)/{core,web}.log to follow logs"

stop:
	@for f in $(PIDS_DIR)/*.pid; do \
		[ -f "$$f" ] && kill $$(cat $$f) 2>/dev/null; \
	done
	@pkill -f 'go run ./cmd/server' 2>/dev/null || true
	@pkill -f 'vite' 2>/dev/null || true
	@pkill -f 'price-service' 2>/dev/null || true
	@rm -rf $(PIDS_DIR)
	@echo "stopped"

test: test-core test-web

test-core:
	cd core && DATABASE_DSN="$(DB_DSN)" go test ./...

test-web:
	cd web && npx tsc -b --noEmit

test-web-unit:
	cd web && npm test

test-e2e:
	cd web && npm run e2e

bench:
	cd core && go test ./internal/planner -bench=. -benchmem -run='^$$' -benchtime=2s

# verify — единая команда для приёмочного тестирования всего стека.
# Ожидает запущенный core на :8086 (для e2e). Если core не запущен —
# поднимет временный (см. dev), либо запустите вручную: `make core-bg`.
verify: test-core test-web test-web-unit bench
	@echo "→ запускаю e2e (требует core на :8086)..."
	@curl -sf http://localhost:8086/health >/dev/null 2>&1 || ( \
		echo "core не отвечает на :8086. Запустите 'make core-bg' и повторите." && exit 1)
	$(MAKE) test-e2e
	@echo
	@echo "✓ verify OK — все слои зелёные"

price-service-bg:
	@mkdir -p $(PIDS_DIR)
	@cd price-service && [ -f .env ] || cp .env.example .env
	@(cd price-service && set -a && . ./.env && set +a && \
		go run ./cmd/price-service serve) > $(PIDS_DIR)/price.log 2>&1 & \
		echo $$! > $(PIDS_DIR)/price.pid
	@echo "price-service → http://localhost:8085 (log: $(PIDS_DIR)/price.log)"
	@echo "Note: requires HTTPS access to search.edadeal.io. If TLS fails,"
	@echo "      the service will exit; pricing UI shows a friendly error."
