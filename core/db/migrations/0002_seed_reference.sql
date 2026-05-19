-- +goose Up
-- Назначение: справочные строки, известные на этапе создания схемы.
-- Полное наполнение нормативных справочников (energy_norms, micronutrients,
-- micronutrient_norms) выполняется Python-загрузчиком в рамках задачи П1.

-- DIETS: фиксируются 6 типов из СХЕМА_БД.md §3 DIETS / docs/ФУНКЦИОНАЛ.md §2.
-- Доли БЖУ для classic берутся из МР 2.3.1.0253-21 и зависят от группы КФА —
-- источник истины для classic это energy_norms (МАТМОДЕЛЬ.txt §1.3).
-- Для не-classic диет доли подлежат отдельной фиксации администратором каталога;
-- до их наполнения значения хранятся как NULL.

INSERT INTO diets (diet_id, protein_share, fat_share, carb_share) VALUES
    ('classic',    NULL, NULL, NULL),
    ('keto',       NULL, NULL, NULL),
    ('vegetarian', NULL, NULL, NULL),
    ('vegan',      NULL, NULL, NULL),
    ('paleo',      NULL, NULL, NULL),
    ('fasting',    NULL, NULL, NULL);

-- +goose Down

DELETE FROM diets WHERE diet_id IN
    ('classic', 'keto', 'vegetarian', 'vegan', 'paleo', 'fasting');
