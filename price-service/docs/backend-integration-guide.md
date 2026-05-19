# Инструкция для backend-интеграции с price-service

Этот файл предназначен для разработчика или другой нейронки, которая будет писать backend проекта питания и подключать к нему `price-service`.

## Что такое price-service

`price-service` — локальный stateless Go-сервис для получения ориентировочных реальных цен на продукты через Edadil API.

Сервис:

- принимает уже агрегированный список покупок;
- ищет товары только в сети `Пятёрочка`;
- получает реальные карточки товаров через Edadil API;
- извлекает цену, фасовку, изображение и идентификаторы Edadil;
- выбирает минимальную по стоимости комбинацию целых упаковок;
- возвращает вилку `min_total_price` / `max_total_price` по корзине;
- умеет принимать `selected_shop_uuid`, чтобы считать только по выбранному магазину Пятёрочки;
- возвращает JSON для backend;
- не хранит бизнес-данные;
- не использует БД;
- не использует Docker.

Важно: `price-service` не формирует план питания. Backend должен передавать в него уже готовый агрегированный список покупок.

## Что нельзя добавлять в backend-интеграцию

Не нужно расширять `price-service` и не нужно ожидать от него:

- PostgreSQL;
- Docker;
- пользователей;
- бюджет;
- UI/логику выбора магазина пользователем;
- историю цен;
- рецепты;
- КБЖУ;
- диеты;
- аллергии;
- генерацию плана питания;
- распределение покупок по разным магазинам;
- гарантированную проверку наличия товара.

Эта часть системы отвечает только за оценку стоимости списка покупок по найденным товарам Пятёрочки. Если пользователь уже выбрал магазин в основном приложении, backend может передать его Edadil `shop_uuid` в `price-service`.

## Как запустить price-service

Рабочая директория:

```text
price-service
```

Запуск HTTP-сервера:

```powershell
go run ./cmd/price-service serve
```

Локальный base URL:

```text
http://localhost:8085
```

Проверка:

```powershell
curl http://localhost:8085/health
```

Ожидаемый ответ:

```json
{
  "status": "ok",
  "retailer": "Пятёрочка",
  "retailer_slug": "5ka"
}
```

## Конфигурация сервиса

Конфигурация берётся из переменных окружения или `.env` в корне `price-service`. Для локального запуска backend-разработчику достаточно скопировать `.env.example` в `.env`:

```powershell
Copy-Item .env.example .env
```

Полный набор переменных:

| Переменная | Пример | Зачем нужна backend-разработчику |
| --- | --- | --- |
| `HTTP_PORT` | `8085` | Порт локального HTTP API. Backend вызывает `http://localhost:{HTTP_PORT}`. |
| `DEBUG` | `true` | Включает подробное логирование сервиса. |
| `DEBUG_MASK_SECRETS` | `true` | Маскирует чувствительные debug-значения. |
| `EDADEAL_BASE_URL` | `https://search.edadeal.io` | Base URL внешнего Edadil API. |
| `EDADEAL_RETAILER_SLUG` | `5ka` | Сеть, по которой работает MVP. Сейчас только Пятёрочка. |
| `EDADEAL_RETAILER_NAME` | `Пятёрочка` | Человекочитаемое имя сети в ответах. |
| `EDADEAL_APP_ID` | `edadeal` | Header `x-app-id` для Edadil. |
| `EDADEAL_APP_VERSION` | `1.92.0` | Header `x-app-version` для Edadil. |
| `EDADEAL_PLATFORM` | `desktop` | Header `x-platform` для Edadil. |
| `EDADEAL_OS_VERSION` | `1.0.0` | Header `x-os-version` для Edadil. |
| `EDADEAL_ORIGIN` | `https://edadeal.ru` | Header `Origin` для Edadil. |
| `EDADEAL_REFERER` | `https://edadeal.ru/` | Header `Referer` для Edadil. |
| `EDADEAL_DUID` | пусто | Необязательный stable client id. Не хранить секреты в репозитории. |
| `EDADEAL_PRESET` | `moscow` | Городской preset географии. |
| `EDADEAL_COUNTRY_GEO_ID` | `225` | Country geo id для headers Edadil. |
| `EDADEAL_GEO_ID` | `213` | Geo id города. По умолчанию Москва. |
| `EDADEAL_GEO_PATH` | `225,3,1,213` | Geo path для `x-edadeal-chercher-area`. |
| `EDADEAL_LATITUDE` | `55.6965` | Широта центра поиска. |
| `EDADEAL_LONGITUDE` | `37.5` | Долгота центра поиска. |
| `EDADEAL_CHERCHER_AREA` | пусто | Fallback: готовое значение `x-edadeal-chercher-area`. |
| `EDADEAL_USE_GO_AREA_GENERATOR` | `true` | `true` — генерировать geo header в Go; `false` — брать `EDADEAL_CHERCHER_AREA`. |
| `SEARCH_LIMIT` | `20` | Сколько товаров брать из Edadil search. |
| `DETAIL_CANDIDATES_LIMIT` | `8` | По скольким кандидатам запрашивать detail. |
| `MAX_SHOPS` | `10` | Сколько ближайших магазинов просить у Edadil в detail. |
| `REQUEST_TIMEOUT_SECONDS` | `15` | Timeout внешних запросов к Edadil. |
| `CACHE_TTL_SECONDS` | `600` | TTL in-memory cache результатов поиска. |
| `MAX_CONCURRENT_ITEM_DETAILS` | `4` | Ограничение параллельных detail-запросов. |

Для backend-интеграции обычно менять нужно только `HTTP_PORT`. Географию менять можно, но важно: `EDADEAL_LATITUDE`, `EDADEAL_LONGITUDE`, `EDADEAL_GEO_ID`, `EDADEAL_GEO_PATH` и `x-edadeal-chercher-area` должны соответствовать друг другу.

## Как сервис работает внутри

Startup flow при запуске `go run ./cmd/price-service serve`:

1. Сервис загружает `.env` и переменные окружения.
2. Собирает географические headers Edadil:
   - preferred mode: Go-generator H3 area;
   - fallback mode: `EDADEAL_CHERCHER_AREA` из env.
3. Делает `GET /api/v4/retailer_info?retailerSlug=5ka`.
4. Получает и сохраняет в памяти `retailer_uuid`, `retailer_slug`, `retailer_name`.
5. Создаёт in-memory cache.
6. Запускает HTTP API на `HTTP_PORT`.

Runtime flow для `POST /estimate`:

1. Backend отправляет агрегированный список покупок.
2. Сервис валидирует структуру запроса.
3. Для каждой позиции нормализует query: lower-case, trim, `ё -> е`, схлопывание пробелов.
4. Проверяет in-memory cache по ключу сети, query и географии.
5. При cache miss делает `GET /api/v4/search` в Edadil.
6. Берёт top `DETAIL_CANDIDATES_LIMIT` кандидатов и делает `GET /api/v4/item/{itemUuid}`.
7. Из detail достаёт:
   - цену;
   - вилку цен по `partner.nearest[]`;
   - фасовку;
   - `image_url`;
   - `item_uuid`;
   - `base_offer_uuid`;
   - ближайшие магазины для debug.
8. Фильтрует товары по совместимой единице: `g`, `ml`, `pcs`.
9. Если backend передал `selected_shop_uuid`, оставляет только товары с ценой в этом магазине.
10. DP-алгоритмом выбирает минимальную комбинацию целых упаковок, покрывающую нужное количество.
11. Возвращает JSON с нижней и верхней оценкой корзины.

Важное правило: сервис никогда не считает дробную цену за часть упаковки. Если нужно 1000 мл, а упаковка 930 мл, покупаются 2 целые упаковки.

## Внешние Edadil endpoint'ы

В runtime используются только эти endpoints:

```http
GET https://search.edadeal.io/api/v4/retailer_info
GET https://search.edadeal.io/api/v4/search
GET https://search.edadeal.io/api/v4/item/{itemUuid}
```

Опциональный debug endpoint `GET /api/v4/retailer/{retailerUuid}/items` в основном runtime flow не используется.

## Как backend должен подключаться

Минимальная схема подключения:

1. Backend формирует `shopping_list_id` и агрегированный список покупок после подбора блюд.
2. Backend вызывает `POST http://localhost:8085/estimate`.
3. Backend передаёт `X-Request-ID`, чтобы связать свои логи с логами `price-service`.
4. Если пользователь выбрал магазин, backend передаёт `selected_shop_uuid`.
5. Backend получает `status`, `min_total_price`, `max_total_price`, `items[]`, `unpriced_items[]`.
6. Backend сохраняет raw JSON ответа и нужные поля в своей БД.
7. Backend показывает пользователю вилку корзины и предупреждения по `unpriced_items[]`.

## Главный endpoint для backend

```http
POST /estimate
Content-Type: application/json
X-Request-ID: optional-backend-request-id
```

Полный URL:

```text
http://localhost:8085/estimate
```

`X-Request-ID` необязателен, но backend лучше должен передавать свой request/correlation id. Тогда `price-service` вернёт его же в поле `request_id`.

Если header не передан, `price-service` сам сгенерирует UUID.

## Входной формат

Базовый входной контракт по `items[]` не менялся. Дополнительно можно передать выбранный магазин.

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

Поля:

- `ingredient_name` — название ингредиента или продукта для поиска в Edadil.
- `amount` — требуемое количество.
- `unit` — единица измерения.

Допустимые `unit`:

- `g` — граммы;
- `ml` — миллилитры;
- `pcs` — штуки.

Backend должен отправлять уже агрегированные позиции. Например, если в плане питания молоко нужно в нескольких рецептах, backend должен сам сложить количество и отправить одну позицию.

`selected_shop_uuid` необязателен. Если его передать, сервис будет считать цены только по этому магазину из `partner.nearest[]`. Поддерживается alias `shop_uuid` и query params `?selected_shop_uuid=...` / `?shop_uuid=...`.

## Основной ответ `/estimate`

Обычный `POST /estimate` возвращает compact response без `alternatives` и без `debug`.

Пример реального ответа:

```json
{
  "request_id": "smoke-estimate-request",
  "status": "ok",
  "calculated_at": "2026-05-10T19:20:22Z",
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
            "image_url": "https://leonardo.edadeal.io/..."
          }
        ]
      }
    }
  ],
  "unpriced_items": []
}
```

Полный реальный пример лежит здесь:

```text
docs/smoke-results/estimate-response.json
```

## Как backend должен интерпретировать ответ

Верхний уровень:

- `request_id` — correlation id. Логировать вместе с backend request id.
- `status` — общий статус расчёта.
- `calculated_at` — время завершения расчёта в RFC3339.
- `is_fully_priced` — `true`, если все позиции оценены.
- `price_type` — всегда `estimated_reference_price`.
- `pricing_scope` — `nearest_shops_range`, если магазин не выбран, или `selected_shop`, если backend передал магазин.
- `selected_shop_uuid` — UUID выбранного магазина, если он был передан.
- `retailer` — человекочитаемое имя сети.
- `retailer_slug` — `5ka`.
- `currency` — `RUB`.
- `total_price` — старое совместимое поле, равно `min_total_price`.
- `min_total_price` — нижняя граница стоимости корзины по выбранным целым упаковкам.
- `max_total_price` — верхняя граница стоимости корзины по тем же выбранным целым упаковкам.
- `price_range` — объект с `min_price` и `max_price` для всей корзины.
- `priced_items_count` — количество оценённых позиций.
- `unpriced_items_count` — количество неоценённых позиций.
- `items` — результат по каждой входной позиции.
- `unpriced_items` — короткий список неоценённых позиций.

Общий `status`:

- `ok` — все позиции оценены.
- `partial` — часть позиций оценена, часть нет.
- `failed` — ни одна позиция не оценена.

Backend не должен считать `partial` транспортной ошибкой. Это нормальный бизнес-результат: часть товаров не нашлась или не подошла по фасовке.

## Как читать `items[]`

Каждый элемент `items[]` соответствует одной входной позиции.

Важные поля:

- `ingredient_name` — исходное название из request.
- `requested_amount` — исходное требуемое количество.
- `requested_unit` — исходная единица.
- `status` — статус оценки конкретной позиции.
- `error_message` — `null`, если позиция оценена; текст ошибки, если не оценена.
- `query` — нормализованная строка поиска.
- `price_range` — вилка по конкретной позиции.
- `selected_option` — выбранная минимальная комбинация упаковок, если позиция оценена.

Если `status = "priced"`, backend может брать:

```text
items[].selected_option.packages[]
```

и показывать пользователю выбранные товары.

Если `status != "priced"`, `selected_option` отсутствует, а причина лежит в:

```text
items[].status
items[].error_message
unpriced_items[]
```

## Как читать `selected_option`

```json
{
  "total_price": 107.98,
  "min_total_price": 107.98,
  "max_total_price": 119.98,
  "price_range": {
    "min_price": 107.98,
    "max_price": 119.98
  },
  "covered_amount": 12,
  "covered_unit": "pcs",
  "packages": [
    {
      "product_title": "Яйца куриные Выручай С1 6шт., 6 шт",
      "item_uuid": "fcd2679c-99ba-5942-a2bf-6f9a832d330e",
      "base_offer_uuid": "a5e36cce-7bb4-5425-8565-5f9e9ab2607e",
      "package_amount": 6,
      "package_unit": "pcs",
      "unit_price": 53.99,
      "unit_price_min": 53.99,
      "unit_price_max": 59.99,
      "quantity": 2,
      "subtotal": 107.98,
      "subtotal_min": 107.98,
      "subtotal_max": 119.98,
      "image_url": "https://leonardo.edadeal.io/..."
    }
  ]
}
```

Смысл:

- `total_price` — стоимость выбранной комбинации упаковок для одной позиции.
- `min_total_price` — нижняя стоимость выбранной комбинации.
- `max_total_price` — верхняя стоимость выбранной комбинации.
- `price_range` — дубль в объектном виде для удобного чтения.
- `covered_amount` — сколько реально покрыто целыми упаковками.
- `covered_unit` — единица покрытия.
- `packages[]` — какие товары и в каком количестве купить.

Важно: сервис всегда считает целые упаковки.

Пример:

- нужно `10 pcs` яиц;
- найдена упаковка `6 pcs` за `53.99`;
- сервис может выбрать `2` упаковки;
- `covered_amount = 12`;
- `subtotal = 107.98`.

Нельзя интерпретировать цену пропорционально. Цена всегда за целую упаковку.

## Вилка цен и выбранный магазин

По умолчанию `price-service` считает вилку по ближайшим магазинам, которые пришли в `partner.nearest[]` от Edadil:

- `min_total_price` берётся из минимальных найденных цен;
- `max_total_price` считается по верхним ценам для тех же выбранных упаковок;
- `total_price` оставлен для совместимости и равен `min_total_price`.

Если пользователь выбрал конкретный магазин Пятёрочки, backend передаёт:

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

Можно передавать alias `shop_uuid`, а также query params `?selected_shop_uuid=...` или `?shop_uuid=...`.

В режиме выбранного магазина:

- `pricing_scope = "selected_shop"`;
- response содержит `selected_shop_uuid`;
- для товаров используется только цена из этого магазина;
- `packages[].shop_uuid` показывает магазин, по которому взята цена;
- если у найденного товара нет цены в этом магазине, он не участвует в расчёте;
- если по позиции нет ни одного совместимого товара с ценой в выбранном магазине, статус будет `selected_shop_price_not_found`;
- `min_total_price` и `max_total_price` обычно равны, потому что выбран один конкретный магазин.

Откуда backend берёт `selected_shop_uuid`:

- это Edadil `shopUuid` из `partner.nearest[]`;
- в текущем MVP отдельного endpoint-а выбора магазинов нет;
- для разработки `shopUuid` можно увидеть в `POST /estimate?debug=true` внутри `items[].debug.nearest_shops`;
- в полноценном приложении backend должен передать сюда уже выбранный пользователем Edadil `shopUuid`, если такой выбор реализован в другом модуле.

## Что такое `item_uuid` и `base_offer_uuid`

В каждом `packages[]` есть:

- `item_uuid` — UUID карточки товара Edadil;
- `base_offer_uuid` — UUID оффера Edadil.

Backend должен сохранять/пробрасывать эти поля в UI или логи, если нужно показать источник товара.

Текущее фактическое поведение Edadil:

- `/api/v4/search` сейчас возвращает компактный item с `uuid` и `itemType`, но без `baseOfferUuid`;
- `/api/v4/item/{itemUuid}` без `baseOfferUuid` возвращает полную карточку и `offerUuids`;
- сервис берёт `base_offer_uuid` из detail-ответа и возвращает его backend.

Был проверен detail-запрос с `baseOfferUuid` и без него. Ключевые поля совпали, поэтому в runtime flow `baseOfferUuid` в detail-запрос не добавлен как необязательный.

Результат проверки:

```text
docs/base-offer-check/summary.json
```

## `unpriced_items[]`

Если позиция не оценена, она попадает в `unpriced_items`.

Формат:

```json
{
  "ingredient_name": "кинза",
  "requested_amount": 50,
  "requested_unit": "g",
  "reason": "no_products_found"
}
```

Backend может использовать `unpriced_items[]`, чтобы показать пользователю предупреждение: часть списка не удалось оценить.

## Возможные статусы позиции

`items[].status` может быть:

- `priced` — позиция оценена.
- `unpriced` — позиция не оценена.
- `invalid_unit` — недопустимая единица измерения.
- `api_error` — ошибка при обращении к Edadil по конкретной позиции.
- `parse_error` — не удалось разобрать позицию.
- `no_products_found` — товары не найдены.
- `no_compatible_products_found` — товары найдены, но фасовка/единица не подходят.
- `selected_shop_price_not_found` — товары найдены, но по выбранному магазину нет цены.
- `package_unknown` — не удалось определить фасовку.
- `incompatible_unit` — единица товара несовместима с запрошенной.

Если ошибка только по одной позиции в списке, backend получит HTTP `200`, а ошибка будет внутри `items[]`.

## Alternatives

По умолчанию:

```http
POST /estimate
```

`alternatives` не возвращаются.

Чтобы получить альтернативные фасовки без debug:

```http
POST /estimate?include_alternatives=true
```

Реальный пример:

```text
docs/smoke-results/estimate-with-alternatives-response.json
```

Чтобы получить alternatives и debug:

```http
POST /estimate?debug=true
```

Debug-ответ большой. Его не стоит использовать в обычной backend-интеграции.

## Debug

```http
POST /estimate?debug=true
```

Добавляет в `items[].debug`:

- `detail_endpoint_urls`;
- `nearest_shops`;
- `rejected_candidates`;
- `raw_field_summary`;
- `price_sources`.

Реальный debug-ответ:

```text
docs/smoke-results/estimate-debug-response.json
```

Нужен для отладки и доказательства, что сервис реально ходит в Edadil.

## Последние запросы к Edadil

```http
GET /debug/last-requests
```

Возвращает последние обращения к Edadil:

```json
{
  "requests": [
    {
      "operation": "search_products",
      "url": "https://search.edadeal.io/api/v4/search?...",
      "status_code": 200,
      "duration_ms": 120,
      "items_count": 20,
      "error": ""
    }
  ]
}
```

Реальный пример:

```text
docs/smoke-results/debug-last-requests.json
```

Backend обычно не должен вызывать этот endpoint. Он нужен для локальной диагностики.

## Reload

```http
POST /reload
```

Перечитывает конфиг и очищает in-memory cache.

Ответ:

```json
{
  "status": "ok",
  "cache_cleared": true
}
```

Реальный пример:

```text
docs/smoke-results/reload.json
```

Backend обычно не должен вызывать `/reload` в пользовательском сценарии.

## HTTP-коды

`200`:

- запрос обработан;
- `response.status` может быть `ok`, `partial` или `failed`;
- часть позиций может быть не оценена.

`400`:

- структурно некорректный запрос;
- невалидный JSON;
- пустой `items`;
- полностью невалидный request, который нельзя интерпретировать как список покупок.

`500`:

- внутренняя ошибка сервиса.

`503`:

- Edadil недоступен полностью;
- например, `/reload` не смог заново получить `retailer_info`.

Важно: если ошибка относится только к одной позиции внутри списка, сервис возвращает HTTP `200`, а ошибку фиксирует в `items[].status`, `items[].error_message` и `unpriced_items[]`.

## Примеры curl

Health:

```powershell
curl http://localhost:8085/health
```

Estimate:

```powershell
curl -X POST "http://localhost:8085/estimate" `
  -H "Content-Type: application/json" `
  -H "X-Request-ID: backend-request-123" `
  -d "@data/sample_request.json"
```

Estimate for selected shop:

```powershell
curl -X POST "http://localhost:8085/estimate" `
  -H "Content-Type: application/json" `
  -d "{\"selected_shop_uuid\":\"052761c1-6775-4ac4-8d9d-2f03c974932b\",\"items\":[{\"ingredient_name\":\"молоко\",\"amount\":1000,\"unit\":\"ml\"}]}"
```

Estimate with alternatives:

```powershell
curl -X POST "http://localhost:8085/estimate?include_alternatives=true" `
  -H "Content-Type: application/json" `
  -d "@data/sample_request.json"
```

Estimate with debug:

```powershell
curl -X POST "http://localhost:8085/estimate?debug=true" `
  -H "Content-Type: application/json" `
  -d "@data/sample_request.json"
```

Reload:

```powershell
curl -X POST "http://localhost:8085/reload"
```

Debug last requests:

```powershell
curl http://localhost:8085/debug/last-requests
```

## Рекомендации для backend

Backend-интеграция должна:

- отправлять в `price-service` только агрегированный список покупок;
- использовать `POST /estimate` без `debug=true` для обычного сценария;
- передавать `X-Request-ID`;
- передавать `selected_shop_uuid`, если пользователь выбрал конкретный магазин;
- читать `response.status`;
- не падать, если `response.status = "partial"`;
- показывать пользователю `unpriced_items`, если часть товаров не оценена;
- брать выбранные товары из `items[].selected_option.packages[]`;
- использовать `min_total_price` и `max_total_price` как вилку корзины;
- использовать `total_price` только как совместимое поле нижней цены;
- помнить, что `price_type = "estimated_reference_price"`, то есть это справочная оценка, а не гарантия наличия или финальной цены на кассе.

Backend-интеграция не должна:

- ожидать, что `price-service` сам выберет магазин за пользователя;
- ожидать гарантированного наличия товара;
- пересчитывать цену пропорционально граммам/миллилитрам;
- заменять `total_price` своей суммой по дробным упаковкам;
- требовать от `price-service` данные о пользователях, рецептах, КБЖУ, диетах или аллергиях.

## Что backend рекомендуется сохранять в своей БД

`price-service` сам не хранит бизнес-данные, поэтому если backend-у нужна история расчётов или связь с пользовательским списком покупок, сохранять это нужно на стороне backend.

Рекомендуемый минимум для сохранения результата расчёта:

- `shopping_list_id` — id списка покупок в backend.
- `request_id` — correlation id из ответа `price-service`.
- `calculated_at` — время расчёта из ответа.
- `retailer_slug` — сейчас всегда `5ka`.
- `currency` — сейчас `RUB`.
- `price_type` — сейчас `estimated_reference_price`.
- `pricing_scope` — режим расчёта цен.
- `selected_shop_uuid` — если расчёт был по конкретному магазину.
- `total_price` — совместимое поле нижней цены.
- `min_total_price` — нижняя граница корзины.
- `max_total_price` — верхняя граница корзины.
- `priced_items_count` — количество оценённых позиций.
- `unpriced_items_count` — количество неоценённых позиций.
- `is_fully_priced` — удалось ли оценить весь список.
- `raw response JSON` — полный JSON-ответ `price-service` для отладки и воспроизводимости.
- `items[].selected_option.packages[]` — выбранные упаковки для оценённых позиций.
- `unpriced_items[]` — предупреждения для пользователя по неоценённым позициям.

Не нужно сохранять это в `price-service`: он остаётся stateless-интеграционным сервисом.

## Проверенные smoke-файлы

Последний smoke-test сохранён здесь:

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

Краткий результат последней проверки:

```json
{
  "health_ok": true,
  "estimate_ok": true,
  "estimate_status": "ok",
  "request_id_present": true,
  "calculated_at_present": true,
  "is_fully_priced_present": true,
  "price_type": "estimated_reference_price",
  "pricing_scope": "nearest_shops_range",
  "min_total_price_present": true,
  "max_total_price_present": true,
  "price_range_present": true,
  "total_price_equals_min_total_price": true,
  "alternatives_hidden_by_default": true,
  "alternatives_present_with_include_alternatives": true,
  "debug_present_with_debug_true": true,
  "selected_shop_uuid_used": true,
  "selected_shop_range_collapsed": true,
  "reload_cache_cleared": true,
  "real_edadeal_requests_recorded": true,
  "base_offer_uuid_detail_behavior": "not_needed"
}
```

## Минимальный алгоритм backend-вызова

1. Сформировать агрегированный список покупок.
2. Отправить `POST http://localhost:8085/estimate`.
3. Передать `X-Request-ID`.
4. Если HTTP `200`, прочитать JSON.
5. Если `status = "ok"`, использовать `min_total_price`, `max_total_price` и `items[].selected_option`.
6. Если `status = "partial"`, использовать оценённые позиции и показать предупреждение по `unpriced_items`.
7. Если `status = "failed"`, показать, что оценить список не удалось.
8. Если HTTP `400`, исправить request backend-а.
9. Если HTTP `503`, считать сервис цен временно недоступным.
