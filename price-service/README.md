# price-service

Локальный Go-сервис для MVP проекта «Разработка программного модуля по формированию персонализированного плана питания».

Сервис принимает уже агрегированный список покупок от backend, ищет товары через Edadil API только по сети «Пятёрочка», получает цены и фасовки, выбирает минимальную по стоимости комбинацию целых упаковок и возвращает JSON с вилкой `min_total_price` / `max_total_price`.

Ключевая цель MVP: доказать, что цены и карточки товаров реально достаются через API, а не читаются из локального mock-файла.

Подробный документ для backend-разработчика: `docs/backend-integration-guide.md`.

## Ограничения MVP

- Работает только с Пятёрочкой (`retailerSlug=5ka`).
- Принимает уже агрегированный список покупок.
- Не формирует план питания.
- Не работает с КБЖУ.
- Не проверяет диеты и аллергии.
- Не проверяет гарантированное актуальное наличие товара.
- Не реализует UI выбора магазина, но может принять `selected_shop_uuid` от backend и считать только по нему.
- Не оптимизирует бюджет пользователя.
- Не распределяет список покупок по магазинам.
- Не работает с рецептами.
- Не интегрируется с 2GIS.
- Не использует PostgreSQL и Docker.
- Не использует старые protobuf endpoint'ы Edadil.
- Выбирает минимальную комбинацию упаковок по найденным ценам.

## Edadil endpoints

Base URL:

```text
https://search.edadeal.io
```

Используются:

- `GET /api/v4/retailer_info` — получение `retailer_uuid` для `retailerSlug=5ka`.
- `GET /api/v4/search` — поиск товаров по названию ингредиента внутри Пятёрочки.
- `GET /api/v4/item/{itemUuid}` — детальная карточка товара, цены, фасовка, изображение и ближайшие магазины.

Опционально для ручной диагностики структуры каталога:

- `GET /api/v4/retailer/{retailerUuid}/items`

В основном runtime flow debug endpoint не используется.

### Фактическое поведение `baseOfferUuid` в detail-запросах

Проверено на реальной карточке товара:

- без `baseOfferUuid`: `GET /api/v4/item/{itemUuid}?disablePlatformSourceExclusion=true&maxShops=10&type=meta_offer`
- с `baseOfferUuid`: `GET /api/v4/item/{itemUuid}?baseOfferUuid={baseOfferUuid}&disablePlatformSourceExclusion=true&maxShops=10&type=meta_offer`

Для проверенного товара Edadil вернул одинаковые ключевые поля: HTTP status, `title`, `priceData`, `quantity`, `quantityUnit`, `imageUrl`, `partner.nearest`, parsed price и parsed package. Текущий `/api/v4/search` отдаёт компактные элементы с `uuid/itemType` и не даёт `baseOfferUuid` до detail-запроса; detail-запрос без `baseOfferUuid` возвращает `offerUuids`, из которых сервис сохраняет `base_offer_uuid` в итоговый ответ backend. Поэтому `baseOfferUuid` в detail-запросе для текущего runtime flow не нужен. Результаты сравнения лежат в `docs/base-offer-check/summary.json`.

## Конфигурация

Скопируйте пример:

```powershell
Copy-Item .env.example .env
```

Основные переменные:

```text
HTTP_PORT=8085
DEBUG=true
DEBUG_MASK_SECRETS=true

EDADEAL_BASE_URL=https://search.edadeal.io
EDADEAL_RETAILER_SLUG=5ka
EDADEAL_RETAILER_NAME=Пятёрочка

EDADEAL_APP_ID=edadeal
EDADEAL_APP_VERSION=1.92.0
EDADEAL_PLATFORM=desktop
EDADEAL_OS_VERSION=1.0.0
EDADEAL_ORIGIN=https://edadeal.ru
EDADEAL_REFERER=https://edadeal.ru/
EDADEAL_DUID=

EDADEAL_PRESET=moscow
EDADEAL_COUNTRY_GEO_ID=225
EDADEAL_GEO_ID=213
EDADEAL_GEO_PATH=225,3,1,213
EDADEAL_LATITUDE=55.6965
EDADEAL_LONGITUDE=37.5

EDADEAL_CHERCHER_AREA=
EDADEAL_USE_GO_AREA_GENERATOR=true

SEARCH_LIMIT=20
DETAIL_CANDIDATES_LIMIT=8
MAX_SHOPS=10
REQUEST_TIMEOUT_SECONDS=15
CACHE_TTL_SECONDS=600
MAX_CONCURRENT_ITEM_DETAILS=4
```

Если `EDADEAL_USE_GO_AREA_GENERATOR=true`, сервис генерирует `x-edadeal-chercher-area` сам.

Если генератор нужно временно отключить, задайте:

```text
EDADEAL_USE_GO_AREA_GENERATOR=false
EDADEAL_CHERCHER_AREA=<готовое base64 значение>
```

Архитектурно генерация вынесена за интерфейс `AreaGenerator`. В этой реализации используется чистый Go H3-пакет, потому что официальный `github.com/uber/h3-go/v4` требует CGO, а локальная Windows-среда проекта запускается с `CGO_ENABLED=0`.

## Запуск HTTP server

```powershell
go run ./cmd/price-service serve
```

Healthcheck:

```powershell
curl http://localhost:8085/health
```

Ответ:

```json
{
  "status": "ok",
  "retailer": "Пятёрочка",
  "retailer_slug": "5ka"
}
```

## Расчёт цен

```powershell
curl -X POST "http://localhost:8085/estimate?debug=true" `
  -H "Content-Type: application/json" `
  -d "@data/sample_request.json"
```

Пример request:

```json
{
  "selected_shop_uuid": "optional-edadeal-shop-uuid",
  "items": [
    {
      "ingredient_name": "молоко",
      "amount": 1000,
      "unit": "ml"
    },
    {
      "ingredient_name": "рис",
      "amount": 500,
      "unit": "g"
    },
    {
      "ingredient_name": "яйца",
      "amount": 10,
      "unit": "pcs"
    }
  ]
}
```

`selected_shop_uuid` необязателен. Если backend передаёт UUID конкретного магазина Пятёрочки из Edadil, сервис использует только цены из этого магазина. Также поддерживаются alias `shop_uuid` и query params `?selected_shop_uuid=...` / `?shop_uuid=...`.

Пример формы response:

```json
{
  "request_id": "client-or-generated-uuid",
  "status": "ok",
  "calculated_at": "2026-05-10T18:01:29Z",
  "is_fully_priced": true,
  "price_type": "estimated_reference_price",
  "pricing_scope": "nearest_shops_range",
  "retailer": "Пятёрочка",
  "retailer_slug": "5ka",
  "currency": "RUB",
  "total_price": 249.98,
  "min_total_price": 249.98,
  "max_total_price": 289.98,
  "price_range": {
    "min_price": 249.98,
    "max_price": 289.98
  },
  "priced_items_count": 2,
  "unpriced_items_count": 1,
  "items": [
    {
      "ingredient_name": "молоко",
      "requested_amount": 1000,
      "requested_unit": "ml",
      "status": "priced",
      "error_message": null,
      "query": "молоко",
      "price_range": {
        "min_price": 89.99,
        "max_price": 99.99
      },
      "selected_option": {
        "total_price": 89.99,
        "min_total_price": 89.99,
        "max_total_price": 99.99,
        "price_range": {
          "min_price": 89.99,
          "max_price": 99.99
        },
        "covered_amount": 1000,
        "covered_unit": "ml",
        "packages": [
          {
            "product_title": "Молоко 1 л",
            "item_uuid": "...",
            "base_offer_uuid": "...",
            "package_amount": 1000,
            "package_unit": "ml",
            "unit_price": 89.99,
            "unit_price_min": 89.99,
            "unit_price_max": 99.99,
            "quantity": 1,
            "subtotal": 89.99,
            "subtotal_min": 89.99,
            "subtotal_max": 99.99,
            "image_url": "..."
          }
        ]
      }
    }
  ],
  "unpriced_items": [
    {
      "ingredient_name": "кинза",
      "requested_amount": 50,
      "requested_unit": "g",
      "reason": "no_products_found"
    }
  ]
}
```

`POST /estimate` по умолчанию не возвращает `alternatives`. Чтобы получить их без debug:

```powershell
curl -X POST "http://localhost:8085/estimate?include_alternatives=true" `
  -H "Content-Type: application/json" `
  -d "@data/sample_request.json"
```

Расчёт по выбранному магазину:

```powershell
curl -X POST "http://localhost:8085/estimate" `
  -H "Content-Type: application/json" `
  -d "{\"selected_shop_uuid\":\"052761c1-6775-4ac4-8d9d-2f03c974932b\",\"items\":[{\"ingredient_name\":\"молоко\",\"amount\":1000,\"unit\":\"ml\"}]}"
```

В этом режиме `pricing_scope = "selected_shop"`, а `min_total_price` и `max_total_price` считаются только по ценам выбранного магазина.

`debug=true` добавляет в каждую позицию:

- detail endpoint URL без secret headers;
- nearest shops;
- rejected candidates;
- raw field summary;
- price source.

При `debug=false` адреса конкретных магазинов не возвращаются.

HTTP-коды:

- `200` — запрос обработан; верхнеуровневый `status` может быть `ok`, `partial` или `failed`.
- `400` — структурно некорректный запрос: невалидный JSON, пустой `items` или полностью невалидный request.
- `500` — внутренняя ошибка сервиса.
- `503` — Edadil недоступен полностью, например при `/reload` не удалось получить `retailer_info`.

Если ошибка касается только одной позиции в списке из нескольких позиций, сервис возвращает HTTP `200`, а причину кладёт в `items[].status`, `items[].error_message` и `unpriced_items[]`.

## CLI estimate

```powershell
go run ./cmd/price-service estimate --input ./data/sample_request.json
```

CLI печатает JSON-ответ в stdout.

## Header debug

```powershell
go run ./cmd/price-service header --preset moscow --lat 55.6965 --lon 37.5 --json
```

Команда выводит:

- generated headers;
- декодированный `x-edadeal-chercher-area` JSON при флаге `--json`.

## Tests

```powershell
go test ./...
```

Покрыто:

- нормализация единиц и парсинг фасовки из title;
- DP-оптимизатор упаковок;
- парсинг search/detail ответов Edadil;
- `/estimate` и `/reload` через mocked Edadil client.
