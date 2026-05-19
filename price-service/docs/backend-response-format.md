# Формат ответа price-service для backend

Base URL локального сервиса:

```text
http://localhost:8085
```

## Healthcheck

```http
GET /health
```

Ответ:

```json
{
  "status": "ok",
  "retailer": "Пятёрочка",
  "retailer_slug": "5ka"
}
```

Реальный пример сохранён в:

```text
docs/smoke-results/health.json
```

## Расчёт покупок

```http
POST /estimate
Content-Type: application/json
X-Request-ID: optional-client-request-id
```

Request body не менялся:

```json
{
  "selected_shop_uuid": "optional-edadeal-shop-uuid",
  "items": [
    {
      "ingredient_name": "молоко",
      "amount": 1000,
      "unit": "ml"
    }
  ]
}
```

Допустимые `unit` во входе:

- `g`
- `ml`
- `pcs`

`selected_shop_uuid` необязателен. Если пользователь выбрал конкретный магазин Пятёрочки, backend может передать его UUID из Edadil в `selected_shop_uuid` или коротким alias `shop_uuid`. Тогда сервис будет использовать только цены из `partner.nearest[]` с этим `shop_uuid`. Если магазин не передан, сервис считает вилку по ближайшим магазинам из Edadil.

Если backend передаёт `X-Request-ID`, сервис вернёт это же значение в `response.request_id`. Если header отсутствует, сервис сгенерирует UUID.

## Основной response

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
  "total_price": 273.98,
  "min_total_price": 273.98,
  "max_total_price": 309.98,
  "price_range": {
    "min_price": 273.98,
    "max_price": 309.98
  },
  "priced_items_count": 3,
  "unpriced_items_count": 0,
  "items": [
    {
      "ingredient_name": "молоко",
      "requested_amount": 1000,
      "requested_unit": "ml",
      "status": "priced",
      "error_message": null,
      "query": "молоко",
      "price_range": {
        "min_price": 110,
        "max_price": 130
      },
      "selected_option": {
        "total_price": 110,
        "min_total_price": 110,
        "max_total_price": 130,
        "price_range": {
          "min_price": 110,
          "max_price": 130
        },
        "covered_amount": 1000,
        "covered_unit": "ml",
        "packages": [
          {
            "product_title": "Молоко ЭкоНива ультрапастеризованное 3.2% 1 л",
            "item_uuid": "42fb8842-6aca-5ae7-b5eb-bebce44a9c9d",
            "base_offer_uuid": "98d1a508-e4f3-506b-9d41-78c756c4c51a",
            "package_amount": 1000,
            "package_unit": "ml",
            "unit_price": 110,
            "unit_price_min": 110,
            "unit_price_max": 130,
            "quantity": 1,
            "subtotal": 110,
            "subtotal_min": 110,
            "subtotal_max": 130,
            "image_url": "https://..."
          }
        ]
      }
    }
  ],
  "unpriced_items": []
}
```

Реальный полный ответ сохранён в:

```text
docs/smoke-results/estimate-response.json
```

## Как backend читать ответ

- `request_id` — correlation id для логов backend и price-service.
- `status` — `ok`, `partial` или `failed`.
- `calculated_at` — время завершения расчёта в RFC3339.
- `is_fully_priced` — `true`, если `unpriced_items_count == 0`.
- `price_type` — всегда `estimated_reference_price`.
- `pricing_scope` — `nearest_shops_range`, если магазин не выбран, или `selected_shop`, если backend передал конкретный магазин.
- `selected_shop_uuid` — UUID выбранного магазина, если он был передан.
- `total_price` — старое совместимое поле, равно `min_total_price`.
- `min_total_price` — нижняя граница корзины по выбранным целым упаковкам.
- `max_total_price` — верхняя граница корзины по тем же выбранным целым упаковкам.
- `price_range` — объект `{ "min_price": ..., "max_price": ... }` для всей корзины.
- `priced_items_count` — сколько ингредиентов удалось оценить.
- `unpriced_items_count` — сколько ингредиентов не удалось оценить.
- `items[].status` — статус конкретного ингредиента. Ошибка одной позиции не ломает весь ответ.
- `items[].error_message` — `null` для `priced`, короткое описание причины для остальных статусов.
- `items[].price_range` — вилка по конкретной позиции.
- `items[].selected_option.total_price` — старое совместимое поле, равно `selected_option.min_total_price`.
- `items[].selected_option.min_total_price` — нижняя цена выбранной комбинации упаковок.
- `items[].selected_option.max_total_price` — верхняя цена выбранной комбинации упаковок.
- `items[].selected_option.covered_amount` — сколько фактически покрыто целыми упаковками.
- `items[].selected_option.packages[]` — конкретные товары, которые нужно купить.
- `packages[].quantity` — количество целых упаковок.
- `packages[].unit_price` и `packages[].subtotal` — старые совместимые поля, равны нижней цене.
- `packages[].unit_price_min`, `packages[].unit_price_max` — нижняя и верхняя цена одной упаковки.
- `packages[].subtotal_min`, `packages[].subtotal_max` — нижняя и верхняя сумма по упаковке с учётом `quantity`.
- `packages[].shop_uuid` — заполнен, если расчёт сделан по конкретному выбранному магазину.
- `packages[].item_uuid` и `packages[].base_offer_uuid` — идентификаторы товара/оффера Edadil.

Важно: `total_price` считается по целым упаковкам. Например, если нужно 1000 мл, а товар 930 мл, сервис не считает пропорциональную цену за 1000 мл, а купит 2 упаковки, если это минимальная подходящая комбинация.

## Расчёт по выбранному магазину

Если пользователь в основном приложении выбрал конкретный магазин Пятёрочки, backend передаёт его Edadil UUID:

```json
{
  "selected_shop_uuid": "052761c1-6775-4ac4-8d9d-2f03c974932b",
  "items": [
    {
      "ingredient_name": "молоко",
      "amount": 1000,
      "unit": "ml"
    }
  ]
}
```

Можно использовать alias `shop_uuid`, если backend-у так удобнее. Также поддерживаются query params `?selected_shop_uuid=...` и `?shop_uuid=...`.

В этом режиме:

- `pricing_scope = "selected_shop"`;
- `selected_shop_uuid` возвращается в response;
- товары без цены в выбранном магазине отбрасываются;
- если по позиции товары есть, но цены именно в выбранном магазине нет, статус позиции будет `selected_shop_price_not_found`;
- `min_total_price` и `max_total_price` обычно равны, потому что расчёт идёт по одному магазину.

## Статусы response

- `ok` — все позиции получили `items[].status = "priced"`.
- `partial` — хотя бы одна позиция оценена и хотя бы одна не оценена.
- `failed` — ни одна позиция не оценена.

## Статусы позиции

Возможные `items[].status`:

- `priced`
- `unpriced`
- `invalid_unit`
- `api_error`
- `parse_error`
- `no_products_found`
- `no_compatible_products_found`
- `selected_shop_price_not_found`
- `package_unknown`
- `incompatible_unit`

Если позиция не оценена, она также попадает в `unpriced_items`:

```json
{
  "ingredient_name": "кинза",
  "requested_amount": 50,
  "requested_unit": "g",
  "reason": "no_products_found"
}
```

## Alternatives

По умолчанию `POST /estimate` не возвращает `alternatives`, чтобы основной ответ для backend был компактнее.

```http
POST /estimate?include_alternatives=true
```

Вернёт `alternatives`, но без `debug`.

```http
POST /estimate?debug=true
```

Вернёт и `alternatives`, и `debug`.

Реальный пример с alternatives сохранён в:

```text
docs/smoke-results/estimate-with-alternatives-response.json
```

## Debug response

```http
POST /estimate?debug=true
```

Добавляет в `items[].debug`:

- `detail_endpoint_urls`
- `nearest_shops`
- `rejected_candidates`
- `raw_field_summary`
- `price_sources`

Реальный debug-ответ сохранён в:

```text
docs/smoke-results/estimate-debug-response.json
```

Для обычной backend-интеграции лучше использовать `/estimate` без `debug=true`, чтобы не тянуть адреса магазинов и диагностические поля.

## HTTP-коды

- `200` — запрос обработан. Верхнеуровневый `status` может быть `ok`, `partial` или `failed`; часть позиций может быть не оценена.
- `400` — структурно некорректный запрос: невалидный JSON, пустой `items` или полностью невалидный request.
- `500` — внутренняя ошибка сервиса.
- `503` — Edadil недоступен полностью, например при `/reload` не удалось заново получить `retailer_info`.

Если ошибка касается только одной позиции в списке из нескольких позиций, сервис возвращает HTTP `200`, а причину кладёт в `items[].status`, `items[].error_message` и `unpriced_items[]`.

## Reload

```http
POST /reload
```

Ответ:

```json
{
  "status": "ok",
  "cache_cleared": true
}
```

Реальный пример сохранён в:

```text
docs/smoke-results/reload.json
```

## Smoke-check files

Результаты последней проверки:

```text
docs/smoke-results/check-summary.json
docs/smoke-results/health.json
docs/smoke-results/estimate-response.json
docs/smoke-results/estimate-selected-shop-response.json
docs/smoke-results/estimate-with-alternatives-response.json
docs/smoke-results/estimate-debug-response.json
docs/smoke-results/debug-last-requests.json
docs/smoke-results/reload.json
```
