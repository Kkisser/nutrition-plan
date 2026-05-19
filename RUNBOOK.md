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
  CORE_JWT_SECRET="dev-stable-secret-do-not-use-in-prod" \
  CORE_HTTP_ADDR=":8086" \
  go run ./cmd/server
```

Проверка: `curl http://localhost:8086/health` →
`{"db":"ok","status":"ok"}` (200) или `{"db":"unreachable","status":"degraded"}` (503).

#### Переменные окружения core

| Переменная | Дефолт | Назначение |
|------------|--------|-----------|
| `DATABASE_DSN` | — | DSN PostgreSQL, обязательна. |
| `CORE_JWT_SECRET` | — | Секрет для подписи JWT (HS256). В dev стабильный, чтобы рестарт не валил токены пользователей. |
| `CORE_HTTP_ADDR` | `:8080` | Адрес HTTP-сервера. |
| `PRICE_SERVICE_URL` | — | URL price-service. Не задан → `/pricing` отдаёт 503. |
| `CORE_MAIL_PROVIDER` | `log` | `log` — письма пишутся в slog; `smtp` (TODO) — реальный SMTP. |
| `CORE_EXPOSE_AUTH_TOKEN` | `true` | `true` (dev) — `/auth/register` возвращает `confirm_token` в JSON; `false` (prod) — только через mailer. |
| `CORE_CORS_ORIGINS` | — | CSV разрешённых origin'ов для CORS. Пусто → CORS выключен. `*` — любой origin. |
| `CORE_AUTH_RATE_RPM` | `10` | Лимит запросов на `/auth/{register,login}` в минуту на IP. `0` — выкл. |
| `CORE_LOG_FORMAT` | `text` | `text` (dev) или `json` (prod). |
| `CORE_LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error`. |

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
   увидеть автоматически рассчитанную группу КФА. Опциональный блок
   «Ручные целевые КБЖУ» с зажимом ±15% от нормы.
2. **Сформировать план** — 7 дней × 4 приёма пищи. Доступен экспорт в
   PDF («Скачать PDF» — A4 через системный print).
3. **📌 закрепить** блюдо — при следующей перегенерации останется на месте.
4. **🔄 заменить** — перейти в каталог и выбрать другое блюдо.
5. **🚫 не показывать** — автоматическая перегенерация без этого блюда.
6. **История** — список ранее сформированных недель из IndexedDB.
7. **Покупки** — агрегированный список, сгруппирован по категориям
   (Молочные / Мясо / Овощи / …).
8. **Семья** — экспорт своего плана JSON, импорт партнёрского, общий
   список покупок (одноимённые продукты суммируются).
9. **Цена** — оценка стоимости (если подключён price-service); при
   недоступности — fallback на список без цен.
10. **Offline-режим** — выключить сеть в DevTools, открыть план — он
    доступен из IndexedDB.
11. **PWA install** — на десктоп-Chrome баннер «Установить»; на iOS
    Safari — инструкция «Поделиться → На экран Домой».

---

## 5. Тесты

```sh
make test                            # Go + tsc (всё локально)

# Go: unit + integration
cd core && go test ./...

# Go: бенчмарк планировщика
cd core && go test ./internal/planner -bench=. -benchmem -run='^$'

# Web: tsc проверка
cd web && npx tsc --noEmit

# Web: unit-тесты (vitest + happy-dom)
cd web && npm test

# Web: e2e (Playwright, поднимет vite dev сам; core должен быть запущен)
cd web && npm run e2e
```

В Go проекте unit/integration тесты по всем 12 пакетам: planner,
repository (на реальной БД), targets, catalog, activity, micronutrients,
compliance, shopping, pricing, mailer, api (включая CORS и rate
limiting), config.

Бенчмарк планировщика на M-серии Apple: 13–32 µs/op для 20–200 рецептов.

Frontend unit-тесты: 23 теста на критический путь (kfa, strings,
combineShopping).

E2E (Playwright): 4 сценария — регистрация → анкета → план, валидация
формы, ошибки логина. Требует запущенный core на :8086 и доступ к dev
порту 5173 (Playwright сам поднимает vite dev).

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
