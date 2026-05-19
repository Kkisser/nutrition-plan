# web — PWA-фронт для модуля формирования плана питания

React 18 + TypeScript + Vite + vite-plugin-pwa. Спецификация: `../ФРОНТЕНД.md`.

## Требования

- Node 20+ (рекомендовано 22)
- Запущенный core-сервер (по умолчанию `:8086`)

## Установка

```sh
cd web
npm install
```

## Разработка

```sh
# В соседнем терминале — запустить ядро:
cd ../core
DATABASE_DSN="postgres://Kirill@localhost:5432/nutrition_dev?sslmode=disable" \
  CORE_HTTP_ADDR=":8086" \
  go run ./cmd/server

# В web/:
npm run dev
# → http://localhost:5173
```

Vite-прокси перенаправляет `/api/*` на `http://localhost:8086`,
поэтому фронт обращается к относительному `/api/plan` и т.д.

## Сборка для прод/демо

```sh
npm run build      # → dist/
npm run preview    # локальный просмотр dist
```

При `npm run build` `vite-plugin-pwa` генерирует Service Worker и
прекеш всех ассетов; результат можно открыть на HTTPS и установить
как приложение.

## Структура

```
web/
├── public/
│   └── icons/                 # 192, 512, maskable (PNG)
├── src/
│   ├── main.tsx               # точка входа
│   ├── App.tsx                # роуты
│   ├── api/
│   │   ├── types.ts           # типы по КОНТРАКТ_API.md §2
│   │   ├── client.ts          # fetch
│   │   └── persist.ts         # IndexedDB (idb-keyval)
│   └── pages/
│       ├── Survey.tsx         # анкета профиля
│       ├── Plan.tsx           # недельный план
│       └── Shopping.tsx       # список покупок
├── index.html
├── vite.config.ts
├── package.json
└── tsconfig.json
```

## Текущий объём

MVP: 3 экрана (Анкета, План, Покупки). Без аутентификации (пока
`user_id` — фиксированный uuid), без оценки стоимости, без замены
блюд. Эти страницы добавляются отдельными итерациями.

## Иконки

В `public/icons/` лежат **плейсхолдеры**. Для защиты заменить на
дизайнерские (192, 512, maskable-512 PNG).
