# core/db — миграции схемы ядра

Миграции PostgreSQL под [goose](https://github.com/pressly/goose).
Источник схемы — `../../СХЕМА_БД.md` (Задача Б3).

## Требования

- PostgreSQL ≥ 14
- `goose` CLI ≥ v3 (`go install github.com/pressly/goose/v3/cmd/goose@latest`)

## Окружение

```sh
export DATABASE_DSN="postgres://app:app@localhost:5432/nutrition?sslmode=disable"
```

## Накат

```sh
goose -dir core/db/migrations postgres "$DATABASE_DSN" up
```

## Откат (последняя миграция)

```sh
goose -dir core/db/migrations postgres "$DATABASE_DSN" down
```

## Статус

```sh
goose -dir core/db/migrations postgres "$DATABASE_DSN" status
```

## Текущие миграции

- `0001_init.sql` — ENUMы и 22 таблицы трёх кластеров (пользовательский,
  рецептурно-продуктовый, плановый). Без данных.
- `0002_seed_reference.sql` — 6 строк `diets`. Доли БЖУ оставлены `NULL`
  до наполнения нормативного справочника (см. ниже).

## Что не заполняется этими миграциями

Нормативные справочники наполняются отдельно:

- `energy_norms` — таблицы МР 2.3.1.0253-21 по (пол, возраст, группа КФА).
- `micronutrients`, `micronutrient_norms` — перечень и нормы микронутриентов
  по МР 2.3.1.0253-21; верхние допустимые уровни — МР 2.3.1.1915-04 / EFSA.
- `products`, `product_micronutrients` — справочник Скурихина-Тутельяна.

Источник наполнения — Python-загрузчик (задача П1 трекера).

## Открытые места, требующие отдельной миграции позже

- `activity_survey.q1_daily_activity`, `q3_exercise_freq`,
  `q4_exercise_intensity` пока `varchar(64)`. После фиксации перечня
  ответов мини-анкеты (файл `Анкета_с_активностью_FINAL.md`, упоминается
  в `СХЕМА_БД.md §3`) — отдельной миграцией заменить на enum + CHECK.
- `diets.protein_share` / `fat_share` / `carb_share` для не-classic
  диет — фиксируются при наполнении каталога блюд (задача П4).
