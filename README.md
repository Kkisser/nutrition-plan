# Персональный план питания

Модуль формирования персонализированного недельного плана питания.
Выпускная квалификационная работа МИСиС, 2026.

**Стек:** Go 1.23 · React 18 + TypeScript · PostgreSQL 16 · Python 3.14 (loader) · PWA

---

## Что умеет

- **Формирование плана на 7 дней × 4 приёма пищи** двухфазным алгоритмом
  (greedy + balance hill-climbing), целевые КБЖУ — по нормам МР 2.3.1.0253-21.
- **6 диет**: classic, keto, vegetarian, vegan, paleo, fasting.
  Каталог 83 рецепта, ≥3 рецепта на любую пару (диета × приём).
- **Замена блюд, закрепление, исключение, ручные целевые КБЖУ** (±15% от нормы).
- **Список покупок** агрегируется из плана, группируется по категориям
  (Молочные / Мясо / Овощи / …).
- **История недель** в IndexedDB, экспорт плана в PDF.
- **Семейное меню** (§15 функционала): объединение планов двух пользователей
  в общий список покупок.
- **Опциональная оценка стоимости** через price-service (внешний микросервис
  на Edadil API).
- **Auth**: JWT + bcrypt + email-верификация (mock-mailer в dev, готово к SMTP).
- **PWA**: офлайн-доступ к последнему плану, install prompt для Chrome/iOS.
- **Production-grade**: CORS, rate limiting, structured logging (slog),
  healthcheck с пингом БД.

## Скриншоты

UI работает в браузере на http://localhost:5173 после `make dev`.
Экраны: Login → Survey (анкета) → Plan → Replace → Shopping → Pricing →
History → Family.

---

## Быстрый старт

Требуется: Go 1.23+, Node 20+, PostgreSQL 14+ (Postgres.app или brew),
Python 3.11+, goose (`brew install goose`).

```sh
git clone https://github.com/Kkisser/nutrition-plan.git
cd nutrition-plan

# 1. БД + миграции + загрузка smoke-каталога
make smoke

# 2. Поднять core (:8086) и web (:5173) в фоне
make dev

# 3. Открыть http://localhost:5173
```

Зарегистрируйтесь, заполните анкету, нажмите «Сформировать» — получите
семидневный план питания.

`make stop` — остановить фоновые процессы.

Полная инструкция: **[RUNBOOK.md](RUNBOOK.md)**.

---

## Тесты

```sh
make verify             # единая приёмочная команда: всё ниже сразу
make test-core          # Go: 12 пакетов, unit + integration (на live БД)
make test-web-unit      # vitest: 23 теста на critical-path
make test-e2e           # Playwright: 7 e2e-сценариев (нужен запущенный core)
make bench              # бенчмарк планировщика
```

**Покрытие:**

| Слой | Тесты |
|---|---|
| Go unit + integration | 12 пакетов: planner, repository/pg, targets, catalog, auth, mailer, api (+CORS, rate limit, request log), shopping, micronutrients, compliance, pricing, config |
| Pipeline через HTTP | Все 6 диет: register → verify → login → POST /plan → проверка ответа |
| Frontend unit (vitest) | 23 теста: kfa, strings, combineShopping |
| E2E (Playwright Chromium) | 7 сценариев: auth (3), план, замена блюда, корзина+категории+история, ручные КБЖУ |
| Бенчмарк планировщика | 13–32 µs/op для 20–200 рецептов на Apple M4 Pro |

---

## Архитектура

```
nutrition-plan/
├── core/              # Go 1.23, HTTP API (net/http + ServeMux pattern matching)
│   ├── cmd/server/    # entrypoint
│   ├── internal/
│   │   ├── activity/      # КФА из мини-анкеты
│   │   ├── api/           # HTTP-хендлеры, CORS, rate limit, logging
│   │   ├── auth/          # JWT, bcrypt, email verify-token
│   │   ├── catalog/       # фильтр по диете/аллергенам
│   │   ├── compliance/    # проверка попадания в коридор ±10%
│   │   ├── domain/        # доменные модели
│   │   ├── mailer/        # абстракция SMTP (LogMailer в dev)
│   │   ├── micronutrients/ # carryover недобора
│   │   ├── planner/       # двухфазный алгоритм (greedy + balance)
│   │   ├── pricing/       # клиент к price-service
│   │   ├── repository/pg/ # Postgres репо (pgx/v5)
│   │   ├── shopping/      # агрегация покупок
│   │   └── targets/       # расчёт DailyTargets
│   └── db/migrations/     # goose миграции (0001..0006)
├── web/               # React 18 + TypeScript + Vite + PWA
│   └── src/
│       ├── pages/         # Login, Survey, Plan, Replace, Shopping,
│       │                  # Pricing, History, Family
│       ├── api/           # клиенты к /api, IndexedDB persistence
│       ├── components/    # Toast, Spinner, InstallPrompt
│       └── lib/           # kfa-дерево, strings, shopping, printPlan
├── loader/            # Python 3.14: загрузка справочника Скурихин/Тутельян
│   └── data/smoke/    # CSV-набор для dev (products, recipes, нормы)
├── price-service/     # отдельный Go-сервис оценки стоимости (Edadil)
├── Makefile           # единые команды для всего стека
└── RUNBOOK.md         # инструкции по развёртыванию и тестам
```

---

## Документация проекта

Спецификации, по которым писался код:

- **[ФУНКЦИОНАЛ.md](ФУНКЦИОНАЛ.md)** — функциональные требования (16 разделов).
- **[МАТМОДЕЛЬ.txt](МАТМОДЕЛЬ.txt)** — формальная мат. модель (F-функция, штрафы,
  OptimalAlpha, двухфазный алгоритм).
- **[СХЕМА_БД.md](СХЕМА_БД.md)** — схема PostgreSQL.
- **[КОНТРАКТ_API.md](КОНТРАКТ_API.md)** — REST endpoints, форматы запросов/ответов.
- **[ИСТОЧНИКИ_ОБОСНОВАНИЯ.md](ИСТОЧНИКИ_ОБОСНОВАНИЯ.md)** — карта источников
  (МР 2.3.1.0253-21, Скурихин/Тутельян 2007, AND Position, и др.).
- **[АНКЕТА_активности.md](АНКЕТА_активности.md)** — обоснование вопросов
  Q1/Q3/Q4 и дерева решений КФА.
- **[КАТАЛОГ_ВСЕ_ДИЕТЫ_ИТОГ.md](КАТАЛОГ_ВСЕ_ДИЕТЫ_ИТОГ.md)** — итоговая
  сводка по каталогу блюд.
- **[RUNBOOK.md](RUNBOOK.md)** — пошаговая инструкция запуска и тестирования.
- **[DEPLOY.md](DEPLOY.md)** — развёртывание в проде (Docker).

---

## Переменные окружения core

| Переменная | Дефолт | Назначение |
|---|---|---|
| `DATABASE_DSN` | — | DSN PostgreSQL (обязательна). |
| `CORE_JWT_SECRET` | — | Секрет для JWT (HS256). |
| `CORE_HTTP_ADDR` | `:8080` | Адрес HTTP-сервера. |
| `PRICE_SERVICE_URL` | — | URL price-service. Не задан → `/pricing` отдаёт 503. |
| `CORE_MAIL_PROVIDER` | `log` | `log` — письма в slog; `smtp` (TODO). |
| `CORE_EXPOSE_AUTH_TOKEN` | `true` | dev: `confirm_token` в response; prod (`false`) — только через mailer. |
| `CORE_CORS_ORIGINS` | — | CSV разрешённых origin'ов. Пусто → выключен. |
| `CORE_AUTH_RATE_RPM` | `10` | Лимит запросов на `/auth/*` в минуту на IP. |
| `CORE_LOG_FORMAT` | `text` | `text` или `json`. |
| `CORE_LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error`. |

---

## Статус

Backend и frontend готовы к защите. Тесты зелёные (`make verify` ✓).

Открытые задачи (см. memory/backlog):

- Реальный SMTP — структурно готов (mailer-абстракция), нужен выбор провайдера.
- Расширение микронутриентов с 6 базовых до полного списка
  МР 2.3.1.1915-04 (~27 нутриентов) — требует Скурихин/Тутельян 2007.
- Реальный pricing — Edadil блокирует dev, нужна альтернатива
  (X5 retail / Перекрёсток / mock из Росстата).

---

## Автор

Краснов Кирилл, МИСиС, 2026.
