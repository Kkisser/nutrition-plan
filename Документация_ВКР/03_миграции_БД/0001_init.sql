-- +goose Up
-- Назначение: первичная схема БД ядра модуля формирования персонализированного плана питания.
-- Источник: docs/СХЕМА_БД.md (Задача Б3) + docs/МАТМОДЕЛЬ.txt + docs/ФУНКЦИОНАЛ.md.
-- Кластеры: 3.1 пользовательский, 3.2 рецептурно-продуктовый, 3.3 плановый.

-- ============================================================================
-- 1. ENUM-типы
-- ============================================================================

CREATE TYPE sex AS ENUM ('male', 'female');

CREATE TYPE kfa_group AS ENUM ('I', 'II', 'III', 'IV');

CREATE TYPE diet_type AS ENUM ('classic', 'keto', 'vegetarian', 'vegan', 'paleo', 'fasting');

CREATE TYPE goal_type AS ENUM ('deficit', 'maintain', 'surplus');

CREATE TYPE meal_type AS ENUM ('breakfast', 'lunch', 'dinner', 'snack');

CREATE TYPE unit_type AS ENUM ('g', 'ml', 'pcs');

CREATE TYPE allergen AS ENUM (
    'milk', 'eggs', 'fish', 'gluten', 'peanut',
    'sesame', 'shellfish', 'soy', 'nuts'
);

-- Возрастные группы по МР 2.3.1.0253-21 (МАТМОДЕЛЬ.txt §1.1).
CREATE TYPE age_group AS ENUM ('18-29', '30-44', '45-64', '65-74', '75+');

-- ============================================================================
-- 2. Пользовательский кластер
-- ============================================================================

CREATE TABLE diets (
    diet_id        diet_type PRIMARY KEY,
    protein_share  numeric(4, 3),
    fat_share      numeric(4, 3),
    carb_share     numeric(4, 3)
);

CREATE TABLE users (
    user_id              uuid        PRIMARY KEY,
    email                varchar(255) NOT NULL UNIQUE,
    password_hash        varchar(255) NOT NULL,
    email_confirmed      boolean     NOT NULL DEFAULT false,
    email_confirm_token  varchar(255),
    -- Поля профиля nullable до завершения опросника:
    sex                  sex,
    age                  int         CHECK (age IS NULL OR age >= 0),
    height_cm            int         CHECK (height_cm IS NULL OR height_cm > 0),
    weight_kg            numeric(5, 2) CHECK (weight_kg IS NULL OR weight_kg > 0),
    kfa_group            kfa_group,
    diet_type            diet_type   REFERENCES diets(diet_id),
    goal                 goal_type,
    persons              int         NOT NULL DEFAULT 1 CHECK (persons >= 1),
    created_at           timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE activity_survey (
    survey_id              uuid        PRIMARY KEY,
    user_id                uuid        NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    -- Enum-значения вопросов мини-анкеты будут уточнены отдельной миграцией
    -- после фиксации файла Анкета_с_активностью_FINAL.md (СХЕМА_БД.md §3 ACTIVITY_SURVEY).
    q1_daily_activity      varchar(64) NOT NULL,
    q3_exercise_freq       varchar(64) NOT NULL,
    q4_exercise_intensity  varchar(64),
    derived_kfa_group      kfa_group   NOT NULL,
    created_at             timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_activity_survey_user ON activity_survey(user_id);

CREATE TABLE user_goals (
    goal_id           uuid        PRIMARY KEY,
    user_id           uuid        NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    kcal_target       numeric(7, 2) NOT NULL CHECK (kcal_target > 0),
    protein_g_target  numeric(7, 2) NOT NULL CHECK (protein_g_target >= 0),
    fat_g_target      numeric(7, 2) NOT NULL CHECK (fat_g_target >= 0),
    carb_g_target     numeric(7, 2) NOT NULL CHECK (carb_g_target >= 0),
    manual_override   boolean     NOT NULL DEFAULT false,
    valid_from        timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_user_goals_user_valid ON user_goals(user_id, valid_from DESC);

CREATE TABLE user_allergies (
    user_id   uuid     NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    allergen  allergen NOT NULL,
    PRIMARY KEY (user_id, allergen)
);

CREATE TABLE user_excluded_products (
    user_id       uuid         NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    product_name  varchar(255) NOT NULL,
    PRIMARY KEY (user_id, product_name)
);

-- USER_DISH_EXCLUSIONS объявлена позже, после recipes.

-- ============================================================================
-- 3. Рецептурно-продуктовый кластер
-- ============================================================================

CREATE TABLE products (
    product_id    uuid         PRIMARY KEY,
    name          varchar(255) NOT NULL UNIQUE,
    category      varchar(128),
    kcal_100      numeric(7, 2) NOT NULL CHECK (kcal_100 >= 0),
    protein_100   numeric(7, 2) NOT NULL CHECK (protein_100 >= 0),
    fat_100       numeric(7, 2) NOT NULL CHECK (fat_100 >= 0),
    carb_100      numeric(7, 2) NOT NULL CHECK (carb_100 >= 0),
    default_unit  unit_type    NOT NULL
);

CREATE TABLE micronutrients (
    nutrient_id  varchar(32)  PRIMARY KEY,
    name         varchar(128) NOT NULL,
    norm_unit    varchar(16)  NOT NULL,
    ul_value     numeric(12, 4)
);

CREATE TABLE product_micronutrients (
    product_id   uuid           NOT NULL REFERENCES products(product_id) ON DELETE CASCADE,
    nutrient_id  varchar(32)    NOT NULL REFERENCES micronutrients(nutrient_id) ON DELETE RESTRICT,
    amount_100   numeric(12, 4) NOT NULL CHECK (amount_100 >= 0),
    PRIMARY KEY (product_id, nutrient_id)
);
CREATE INDEX idx_product_micronutrients_nutrient ON product_micronutrients(nutrient_id);

CREATE TABLE micronutrient_norms (
    nutrient_id  varchar(32)    NOT NULL REFERENCES micronutrients(nutrient_id) ON DELETE CASCADE,
    sex          sex            NOT NULL,
    age_group    age_group      NOT NULL,
    norm_value   numeric(12, 4) NOT NULL CHECK (norm_value > 0),
    PRIMARY KEY (nutrient_id, sex, age_group)
);

-- Ключевая таблица Решения №14 (СХЕМА_БД.md §3.4).
CREATE TABLE energy_norms (
    sex             sex           NOT NULL,
    age_group       age_group     NOT NULL,
    kfa_group       kfa_group     NOT NULL,
    kcal_norm       numeric(7, 2) NOT NULL CHECK (kcal_norm > 0),
    protein_g_norm  numeric(7, 2) NOT NULL CHECK (protein_g_norm > 0),
    fat_g_norm      numeric(7, 2) NOT NULL CHECK (fat_g_norm > 0),
    carb_g_norm     numeric(7, 2) NOT NULL CHECK (carb_g_norm > 0),
    PRIMARY KEY (sex, age_group, kfa_group)
);

CREATE TABLE recipes (
    recipe_id      uuid         PRIMARY KEY,
    name           varchar(255) NOT NULL,
    instruction    text         NOT NULL,
    cook_time_min  int          CHECK (cook_time_min IS NULL OR cook_time_min >= 0),
    base_portions  int          NOT NULL DEFAULT 1 CHECK (base_portions >= 1),
    meal_type      meal_type    NOT NULL,
    external_id    varchar(128) UNIQUE
);

CREATE TABLE recipe_ingredients (
    recipe_id   uuid          NOT NULL REFERENCES recipes(recipe_id) ON DELETE CASCADE,
    product_id  uuid          NOT NULL REFERENCES products(product_id) ON DELETE RESTRICT,
    amount      numeric(10, 3) NOT NULL CHECK (amount > 0),
    unit        unit_type     NOT NULL,
    PRIMARY KEY (recipe_id, product_id)
);
CREATE INDEX idx_recipe_ingredients_product ON recipe_ingredients(product_id);

CREATE TABLE recipe_diet_compat (
    recipe_id  uuid       NOT NULL REFERENCES recipes(recipe_id) ON DELETE CASCADE,
    diet_id    diet_type  NOT NULL REFERENCES diets(diet_id) ON DELETE RESTRICT,
    PRIMARY KEY (recipe_id, diet_id)
);
CREATE INDEX idx_recipe_diet_compat_diet ON recipe_diet_compat(diet_id);

CREATE TABLE recipe_allergens (
    recipe_id  uuid     NOT NULL REFERENCES recipes(recipe_id) ON DELETE CASCADE,
    allergen   allergen NOT NULL,
    PRIMARY KEY (recipe_id, allergen)
);
CREATE INDEX idx_recipe_allergens_allergen ON recipe_allergens(allergen);

CREATE TABLE user_dish_exclusions (
    user_id  uuid NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    dish_id  uuid NOT NULL REFERENCES recipes(recipe_id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, dish_id)
);
CREATE INDEX idx_user_dish_exclusions_dish ON user_dish_exclusions(dish_id);

-- ============================================================================
-- 4. Плановый кластер
-- ============================================================================

CREATE TABLE meal_plans (
    plan_id     uuid        PRIMARY KEY,
    user_id     uuid        NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    week_ref    varchar(8)  NOT NULL,
    date_start  date        NOT NULL,
    date_end    date        NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CHECK (date_end >= date_start)
);
CREATE INDEX idx_meal_plans_user_week ON meal_plans(user_id, week_ref);

CREATE TABLE meal_plan_slots (
    slot_id    uuid           PRIMARY KEY,
    plan_id    uuid           NOT NULL REFERENCES meal_plans(plan_id) ON DELETE CASCADE,
    day_no     int            NOT NULL CHECK (day_no BETWEEN 1 AND 7),
    meal_type  meal_type      NOT NULL,
    recipe_id  uuid           NOT NULL REFERENCES recipes(recipe_id) ON DELETE RESTRICT,
    portions   numeric(4, 2)  NOT NULL CHECK (portions > 0),
    is_pinned  boolean        NOT NULL DEFAULT false,
    -- Ровно один приём пищи каждого типа в день (МАТМОДЕЛЬ.txt §3.1).
    UNIQUE (plan_id, day_no, meal_type)
);
CREATE INDEX idx_meal_plan_slots_plan ON meal_plan_slots(plan_id);
CREATE INDEX idx_meal_plan_slots_recipe ON meal_plan_slots(recipe_id);

CREATE TABLE micronutrient_carryover (
    user_id          uuid           NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    week_ref         varchar(8)     NOT NULL,
    nutrient_id      varchar(32)    NOT NULL REFERENCES micronutrients(nutrient_id) ON DELETE CASCADE,
    deficit_per_day  numeric(12, 4) NOT NULL CHECK (deficit_per_day >= 0),
    PRIMARY KEY (user_id, week_ref, nutrient_id)
);

CREATE TABLE shopping_lists (
    list_id  uuid PRIMARY KEY,
    plan_id  uuid NOT NULL UNIQUE REFERENCES meal_plans(plan_id) ON DELETE CASCADE,
    user_id  uuid NOT NULL REFERENCES users(user_id) ON DELETE CASCADE
);
CREATE INDEX idx_shopping_lists_user ON shopping_lists(user_id);

CREATE TABLE shopping_list_items (
    list_id        uuid           NOT NULL REFERENCES shopping_lists(list_id) ON DELETE CASCADE,
    product_name   varchar(255)   NOT NULL,
    amount         numeric(10, 3) NOT NULL CHECK (amount >= 0),
    unit           unit_type      NOT NULL,
    is_purchased   boolean        NOT NULL DEFAULT false,
    is_manual      boolean        NOT NULL DEFAULT false,
    PRIMARY KEY (list_id, product_name)
);

CREATE TABLE plan_members (
    list_id         uuid        NOT NULL REFERENCES shopping_lists(list_id) ON DELETE CASCADE,
    member_user_id  uuid        NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    joined_at       timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (list_id, member_user_id)
);
CREATE INDEX idx_plan_members_user ON plan_members(member_user_id);

-- +goose Down

DROP TABLE IF EXISTS plan_members;
DROP TABLE IF EXISTS shopping_list_items;
DROP TABLE IF EXISTS shopping_lists;
DROP TABLE IF EXISTS micronutrient_carryover;
DROP TABLE IF EXISTS meal_plan_slots;
DROP TABLE IF EXISTS meal_plans;

DROP TABLE IF EXISTS user_dish_exclusions;
DROP TABLE IF EXISTS recipe_allergens;
DROP TABLE IF EXISTS recipe_diet_compat;
DROP TABLE IF EXISTS recipe_ingredients;
DROP TABLE IF EXISTS recipes;
DROP TABLE IF EXISTS energy_norms;
DROP TABLE IF EXISTS micronutrient_norms;
DROP TABLE IF EXISTS product_micronutrients;
DROP TABLE IF EXISTS micronutrients;
DROP TABLE IF EXISTS products;

DROP TABLE IF EXISTS user_excluded_products;
DROP TABLE IF EXISTS user_allergies;
DROP TABLE IF EXISTS user_goals;
DROP TABLE IF EXISTS activity_survey;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS diets;

DROP TYPE IF EXISTS age_group;
DROP TYPE IF EXISTS allergen;
DROP TYPE IF EXISTS unit_type;
DROP TYPE IF EXISTS meal_type;
DROP TYPE IF EXISTS goal_type;
DROP TYPE IF EXISTS diet_type;
DROP TYPE IF EXISTS kfa_group;
DROP TYPE IF EXISTS sex;
