DB_DSN ?= postgres://Kirill@localhost:5432/nutrition_dev?sslmode=disable
MIG_DIR := db/migrations

.PHONY: help db-up db-down db-reset db-status db-recreate test build

help:
	@echo "Targets:"
	@echo "  db-up         goose up"
	@echo "  db-down       goose down (one)"
	@echo "  db-reset      goose reset (down all)"
	@echo "  db-status     goose status"
	@echo "  db-recreate   drop + create $(DB_DSN) database, then up"
	@echo "  test          go test ./..."
	@echo "  build         go build ./..."

db-up:
	goose -dir $(MIG_DIR) postgres "$(DB_DSN)" up

db-down:
	goose -dir $(MIG_DIR) postgres "$(DB_DSN)" down

db-reset:
	goose -dir $(MIG_DIR) postgres "$(DB_DSN)" reset

db-status:
	goose -dir $(MIG_DIR) postgres "$(DB_DSN)" status

db-recreate:
	@dbname=$$(echo "$(DB_DSN)" | sed -E 's|.*/([^/?]+).*|\1|'); \
	host=$$(echo "$(DB_DSN)" | sed -E 's|.*@([^:/]+).*|\1|'); \
	port=$$(echo "$(DB_DSN)" | sed -E 's|.*:([0-9]+)/.*|\1|'); \
	dropdb --if-exists -h $$host -p $$port $$dbname && \
	createdb -h $$host -p $$port $$dbname && \
	$(MAKE) db-up

test:
	go test ./...

build:
	go build ./...
