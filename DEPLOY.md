# DEPLOY — варианты развёртывания

Три способа поднять проект в проде/демо. Выбирайте по обстоятельствам.

---

## Вариант 1. Локальный Docker Compose (рекомендуется для защиты ВКР)

Требует только Docker Desktop.

```sh
docker compose up --build
```

Что произойдёт:
1. Поднимется Postgres 16 на встроенной сети.
2. Контейнер `migrate` дождётся БД и накатит все 5 миграций через goose.
3. Соберётся и стартует core на :8086.
4. Соберётся web (Vite production build → nginx) на :8080.

Открыть: <http://localhost:8080>.

Для cмены JWT-секрета:
```sh
CORE_JWT_SECRET="$(openssl rand -hex 32)" docker compose up --build
```

Чтобы загрузить тестовые данные после первого подъёма (БД в томе):
```sh
docker compose exec postgres psql -U nutrition -d nutrition < loader/data/smoke/*.csv  # не годится для CSV
# проще запустить loader отдельно:
cd loader && python3 -m venv .venv && . .venv/bin/activate && pip install -e .
DATABASE_DSN="postgres://nutrition:nutrition@localhost:5432/nutrition" \
  python -m loader.cli load-all --data-dir data/smoke
```

Остановить и очистить:
```sh
docker compose down -v   # -v уничтожит том с БД
```

---

## Вариант 2. Облако: Vercel (web) + Fly.io / Render (core+postgres)

### 2.1. core + postgres → Fly.io

```sh
cd core
flyctl launch --no-deploy
# отредактировать fly.toml: указать app, region, internal_port=8086
flyctl postgres create
flyctl postgres attach <pg-app-name>
flyctl secrets set CORE_JWT_SECRET="$(openssl rand -hex 32)"
flyctl deploy
```

Fly прицепит DATABASE_URL автоматически. Накатить миграции:
```sh
flyctl ssh console -C "goose -dir db/migrations postgres \"$DATABASE_URL\" up"
```

URL: `https://<app>.fly.dev`.

### 2.2. web → Vercel

В `web/`:
```sh
npm i -g vercel
vercel link
vercel env add VITE_API_BASE   # ввести https://<core-app>.fly.dev
vercel deploy --prod
```

`vite.config.ts` уже настроен на относительный `/api/*`. Чтобы фронт ходил
в облачный core, добавить в `client.ts` базу из `VITE_API_BASE`:

```ts
const BASE = import.meta.env.VITE_API_BASE ?? "/api";
```

(сейчас этот ENV не подцеплен в коде — добавьте при необходимости).

URL: `https://<project>.vercel.app`. PWA сразу можно установить на iOS/Android.

---

## Вариант 3. Один сервер (VPS): docker compose на удалённой машине

```sh
ssh user@server
git clone <repo> && cd <project>
CORE_JWT_SECRET="$(openssl rand -hex 32)" docker compose up -d --build
```

Чтобы был HTTPS (обязательно для PWA-установки), поставить Caddy перед
nginx-ом или дописать `caddy` в docker-compose.yml с автоматическими
Let's Encrypt-сертификатами.

Пример `Caddyfile`:
```
example.com {
    reverse_proxy web:80
}
```

---

## price-service

Опциональный сервис цен. Требует доступ к `search.edadeal.io` (HTTPS).
Поднимается отдельно — мы НЕ кладём его в основной compose, чтобы не
блокировать стек при недоступности Edadil.

```sh
cd price-service
docker build -t price-service .   # (Dockerfile в этом сервисе уже есть)
docker run -p 8085:8085 --env-file .env price-service
```

Затем перезапустить core с `PRICE_SERVICE_URL=http://price-service:8085`.

---

## SMTP для верификации email

Сейчас токен подтверждения возвращается в ответе на `/auth/register`.
Это **dev-режим**. Для прода:

1. Подключить провайдер: SendGrid / Mailgun / Resend / Postmark.
2. Реализовать в `core/internal/auth/mailer.go` функцию
   `SendVerificationEmail(email, token, link)`.
3. Изменить `PostAuthRegister`: после `users.Create` вызвать
   `mailer.SendVerificationEmail` вместо возврата токена в response.

Шаблон email — простой HTML с кнопкой/ссылкой:
```
https://example.com/verify?token=<confirm_token>
```

Фронт страница `/verify` извлекает `?token=` из URL и зовёт `POST /auth/verify`.

---

## Чек-лист безопасности для прод

- [ ] `CORE_JWT_SECRET` — стабильный 32+ байта (иначе токены ломаются при рестарте)
- [ ] Postgres за приватной сетью (Fly internal, VPC)
- [ ] `sslmode=require` в DATABASE_DSN
- [ ] HTTPS обязателен (PWA не работает без него)
- [ ] CORS — добавить в core middleware с явным списком origin
- [ ] Rate-limit на /auth/* — не реализован, добавить (например, github.com/ulule/limiter)
- [ ] Логирование без PII (не писать пароли, токены в логи)
- [ ] Бэкап Postgres
