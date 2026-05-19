-- +goose Up
-- Назначение: трекинг первоисточника данных каждого продукта.
-- Обоснование: ст. 1274 ГК РФ требует указания источника при свободном использовании
-- в учебных/научных целях. Для ВКР это также верификация по первоисточнику
-- (МАТМОДЕЛЬ.txt §1.3 указывает справочник Скурихина-Тутельяна как источник).
-- Колонки nullable, чтобы не ломать ранее загруженные тестовые данные.

ALTER TABLE products
    ADD COLUMN source_name varchar(64),
    ADD COLUMN source_url  varchar(512),
    ADD COLUMN fetched_at  timestamptz;

COMMENT ON COLUMN products.source_name IS 'Источник: skurikhin_2007 | ion_ru | mr_2_3_1_0253_21 | manual';
COMMENT ON COLUMN products.source_url  IS 'URL карточки продукта в первоисточнике (если применимо)';
COMMENT ON COLUMN products.fetched_at  IS 'Дата выгрузки данных из первоисточника';

-- +goose Down

ALTER TABLE products
    DROP COLUMN fetched_at,
    DROP COLUMN source_url,
    DROP COLUMN source_name;
