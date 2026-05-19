-- +goose Up
-- Назначение: фиксация enum-значений мини-анкеты активности по
-- АНКЕТА_активности.md (FINAL). Заменяет varchar(64) на типизированные enum.
-- Источник правил: docs/АНКЕТА_активности.md, docs/ФУНКЦИОНАЛ.md §3, docs/МАТМОДЕЛЬ.txt §1.4.

CREATE TYPE q1_daily_activity AS ENUM (
    'sedentary',          -- В основном сижу
    'standing_low',       -- В основном стою, но мало перемещаюсь
    'frequent_movement',  -- Часто хожу или перемещаюсь
    'heavy_physical'      -- Выполняю физически тяжёлую работу
);

CREATE TYPE q3_exercise_freq AS ENUM (
    'none',    -- Нет таких нагрузок
    '1_to_2',  -- 1–2 раза в неделю
    '3_to_5',  -- 3–5 раз в неделю
    '6_plus'   -- 6 и более раз в неделю
);

CREATE TYPE q4_exercise_intensity AS ENUM (
    'light',     -- Лёгкая
    'moderate',  -- Умеренная
    'intense'    -- Интенсивная
);

-- Сначала очищаем пробные данные, затем меняем тип.
TRUNCATE TABLE activity_survey;

ALTER TABLE activity_survey
    ALTER COLUMN q1_daily_activity     TYPE q1_daily_activity
        USING q1_daily_activity::q1_daily_activity,
    ALTER COLUMN q3_exercise_freq      TYPE q3_exercise_freq
        USING q3_exercise_freq::q3_exercise_freq,
    ALTER COLUMN q4_exercise_intensity TYPE q4_exercise_intensity
        USING q4_exercise_intensity::q4_exercise_intensity;

-- +goose Down

ALTER TABLE activity_survey
    ALTER COLUMN q1_daily_activity     TYPE varchar(64) USING q1_daily_activity::text,
    ALTER COLUMN q3_exercise_freq      TYPE varchar(64) USING q3_exercise_freq::text,
    ALTER COLUMN q4_exercise_intensity TYPE varchar(64) USING q4_exercise_intensity::text;

DROP TYPE IF EXISTS q4_exercise_intensity;
DROP TYPE IF EXISTS q3_exercise_freq;
DROP TYPE IF EXISTS q1_daily_activity;
