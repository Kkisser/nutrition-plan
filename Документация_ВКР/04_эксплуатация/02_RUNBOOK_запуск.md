# RUNBOOK — как запустить проект с нуля

Пошаговая инструкция для научного руководителя, рецензента или нового
члена команды. Цель: за 10 минут получить рабочее приложение в браузере.

---

## 0. Что нужно установить

| Инструмент | Версия | macOS (Homebrew) |
|------------|--------|-------------------|
| Go         | 1.23+  | `brew install go` |
| Node.js    | 20+    | `brew install node` |
| PostgreSQL | 14+    | Postgres.app или `brew install postgresql@16` |
| goose      | v3     | `brew install goose` |
| Python     | 3.11+  | `brew install python@3.12` |

Опционально:
- `make` — для упрощённого запуска (есть на macOS из коробки).

---

## 1. Подготовка БД

Запустите PostgreSQL (Postgres.app или `brew services start postgresql@16`).
Убедитесь, что сервер слушает на 5432:

```sh
pg_isready -h localhost -p 5432
```

---

## 2. Запуск проекта одной командой

Из корня проекта:

```sh
make smoke    # создаст БД, накатит миграции, загрузит тестовый набор
make dev      # запустит ядро + фронт в фоне
```

Откройте http://localhost:5173 в браузере.

`make stop` — остановить все фоновые процессы.

---

## 3. Запуск по шагам (если нужно отладить)

### 3.1. БД и тестовые данные

```sh
createdb -h localhost -p 5432 nutrition_dev
cd core && make db-up         # 5 миграций
cd ../loader
python3 -m venv .venv && . .venv/bin/activate && pip install -e .
DATABASE_DSN="postgres://$USER@localhost:5432/nutrition_dev" \
  loader load-all --data-dir data/smoke
```

Smoke-набор: 20 продуктов, 6 микронутриентов, 7 блюд, 16 норм энергии.

### 3.2. Ядро

```sh
cd core
DATABASE_DSN="postgres://$USER@localhost:5432/nutrition_dev?sslmode=disable" \
  CORE_HTTP_ADDR=":8086" \
  go run ./cmd/server
```

Проверка: `curl http://localhost:8086/health` → `{"status":"ok"}`.

### 3.3. Фронт

```sh
cd web
npm install
npm run dev
```

Откройте http://localhost:5173. Vite-прокси перенаправляет /api → :8086.

### 3.4. price-service (опционально)

Сервис оценки стоимости. Требует HTTPS-доступ к search.edadeal.io.

```sh
cd price-service
cp .env.example .env  # уже есть конфиг по умолчанию
set -a && . ./.env && set +a
go run ./cmd/price-service serve
```

Затем перезапустите core с переменной:

```sh
PRICE_SERVICE_URL=http://localhost:8085 go run ./cmd/server
```

Если Edadil недоступен (нет сети, TLS-таймаут) — фронт покажет понятное
сообщение «сервис цен сейчас недоступен», основной функционал не страдает.

---

## 4. Что попробовать в браузере

1. **Анкета** — заполнить, ответить на 3 вопроса мини-анкеты активности,
   увидеть автоматически рассчитанную группу КФА.
2. **Сформировать план** — 7 дней × 4 приёма пищи.
3. **📌 закрепить** блюдо — при следующей перегенерации останется на месте.
4. **🔄 заменить** — перейти в каталог и выбрать другое блюдо.
5. **🚫 не показывать** — автоматическая перегенерация без этого блюда.
6. **Покупки** — агрегированный список.
7. **Цена** — оценка стоимости (если подключён price-service).
8. **Offline-режим** — выключить сеть в DevTools, открыть план — он
   доступен из IndexedDB.

---

## 5. Тесты

```sh
make test            # Go + tsc
cd core && go test ./...
cd web && npx tsc -b --noEmit
```

В Go проекте ~50 unit/integration тестов: planner, repository (на реальной
БД), targets, catalog, activity, micronutrients, compliance, shopping,
pricing, config.

Интеграционные тесты в `core/internal/repository/pg/` запускаются только
при `DATABASE_DSN` (`make test` пропустит их, если переменная не задана).

---

## 6. Технологический стек (зафиксирован)

| Слой | Технология |
|------|-----------|
| Ядро | Go 1.23 + pgx/v5 + net/http |
| БД   | PostgreSQL 14+ |
| Загрузчик справочников | Python 3.11 + psycopg3 + Click |
| Фронт | React 18 + TypeScript + Vite |
| PWA  | vite-plugin-pwa + Workbox |
| Локальное хранилище фронта | IndexedDB (idb-keyval) |
| price-service | Go 1.23 (отдельный stateless сервис) |

---

## 7. Структура репозитория

```
102839/
├── README.md, ФУНКЦИОНАЛ.md, МАТМОДЕЛЬ.txt, СХЕМА_БД.md,
│   КОНТРАКТ_API.md, АНКЕТА_активности.md, ФРОНТЕНД.md,
│   ИСТОЧНИКИ_ОБОСНОВАНИЯ.md, ТРЕКЕР_задач.md       — спецификации
├── core/                                            — Go-ядро
│   ├── cmd/server/                                  — HTTP-сервер
│   ├── internal/{domain,activity,config,db,repository/pg,
│   │             targets,catalog,planner,micronutrients,
│   │             compliance,shopping,pricing,api}/  — модули
│   ├── db/migrations/                               — goose
│   ├── Makefile
│   └── go.mod
├── loader/                                          — Python загрузчик
│   ├── loader/{cli,db,sources/}
│   ├── data/{templates,smoke}/
│   └── pyproject.toml
├── web/                                             — PWA-фронт
│   ├── src/{api,pages,components,lib}/
│   ├── vite.config.ts, package.json, tsconfig.json
│   └── public/{icons,manifest}
├── price-service/                                   — отдельный сервис
└── Makefile                                         — единые команды
```

---

## 8. Частые проблемы

**`createdb: database "nutrition_dev" already exists`** — БД уже создана,
пропустите этот шаг.

**`goose: command not found`** — `brew install goose`.

**`could not connect to server`** — Postgres не запущен. Откройте
Postgres.app или `brew services start postgresql@16`.

**`pricing disabled (PRICE_SERVICE_URL not set)`** в UI — это норма,
если не запускали price-service. Основной функционал (план, покупки) работает.

**`TLS handshake timeout`** в логах price-service — нет доступа к
Edadil. Скорее всего нет интернета или сервис заблокирован сетью.

**`Node version mismatch`** при `npm install` — обновите Node до 20+.
