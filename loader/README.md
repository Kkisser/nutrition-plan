# loader — Python CSV→PostgreSQL загрузчик справочников

Наполняет таблицы `products`, `product_micronutrients`, `micronutrients`,
`micronutrient_norms`, `energy_norms` из CSV-файлов.

## Источники данных

| CSV | Содержимое | Первоисточник |
|-----|-----------|----------------|
| `products.csv` | продукты + БЖУ-калорийность | Скурихин/Тутельян (изд. 2007) или ion.ru/food/ |
| `product_micronutrients.csv` | содержание микронутриентов в продуктах | тот же справочник |
| `micronutrients.csv` | перечень и единицы микронутриентов + UL | МР 2.3.1.0253-21 + МР 2.3.1.1915-04 / EFSA |
| `micronutrient_norms.csv` | нормы потребления по (пол, возраст) | МР 2.3.1.0253-21 |
| `energy_norms.csv` | нормы энергии и БЖУ по (пол, возраст, КФА) | МР 2.3.1.0253-21 |

Шаблоны и поля — `data/templates/`.

## Установка

```sh
cd loader
python3 -m venv .venv
. .venv/bin/activate
pip install -e .
```

## Запуск

```sh
export DATABASE_DSN="postgres://Kirill@localhost:5432/nutrition_dev"

loader load-all --data-dir data/smoke         # тестовый набор для smoke
loader load-all --data-dir data/full          # полная выгрузка (когда подготовлена)

loader load-products  --file data/full/products.csv
loader load-products  --file data/full/product_micronutrients.csv --kind micronutrients
loader load-norms     --file data/full/energy_norms.csv           --kind energy
loader load-norms     --file data/full/micronutrients.csv         --kind micronutrients
loader load-norms     --file data/full/micronutrient_norms.csv    --kind micronutrient_norms
```

Все команды идемпотентны: повторный запуск с тем же CSV не плодит дубли
(UPSERT по естественным ключам).

## Правовой режим

ВКР (учебно-научные цели). На основании ст. 1274 ГК РФ свободное использование
допустимо с указанием автора и источника. Каждая строка `products` хранит
`source_name` и `source_url` для верификации по первоисточнику.

Издание Скурихин-Тутельян 2007: ISBN 978-5-94343-122-7, ДеЛи принт.
Сайт ФИЦ питания: <http://web.ion.ru/food/FD_tree_grid.aspx>.

## Что не делает

- Парсинг ion.ru — не реализован (см. ТРЕКЕР). Обнуляющий риск через
  «План Б»: данные готовятся вручную в CSV. Парсер — возможный
  follow-up, не блокирует разработку ядра.
- OCR PDF — не реализован.
