-- +goose Up
-- Назначение: зафиксировать доли макронутриентов для каждого типа диеты.
-- Решение по открытому вопросу №2 (вариант C, с источниками).
--
-- Для CLASSIC доли НЕ хранятся в этой таблице (NULL) — они зависят от группы
-- КФА и берутся из energy_norms по МР 2.3.1.0253-21 (МАТМОДЕЛЬ.txt §1.3).
-- В коде ядра diet=classic → читать energy_norms.{protein_g_norm, fat_g_norm,
-- carb_g_norm} и пересчитывать в доли через коэффициенты Этуотера.
--
-- Для остальных пяти типов доли — общепринятые ориентиры из специальной
-- литературы. Источники указаны в комментариях. Сумма = 1.00.

-- KETO: классическая кетогенная диета.
-- Источник: Volek J.S., Phinney S.D. "The Art and Science of Low Carbohydrate
-- Living" (2011); EFSA Scientific Opinion on dietary reference values (2017).
-- Профиль: 25 % белок, 70 % жир, 5 % углеводы.
UPDATE diets SET protein_share = 0.250, fat_share = 0.700, carb_share = 0.050
 WHERE diet_id = 'keto';

-- VEGETARIAN (лакто-ово): согласуется с МР для классического питания.
-- Источник: Position of the Academy of Nutrition and Dietetics:
-- Vegetarian Diets (J Acad Nutr Diet 2016; 116:1970-1980).
-- Профиль: 14 % белок, 30 % жир, 56 % углеводы.
UPDATE diets SET protein_share = 0.140, fat_share = 0.300, carb_share = 0.560
 WHERE diet_id = 'vegetarian';

-- VEGAN: то же распределение, что и vegetarian; биодоступность растительного
-- белка ниже, но компенсируется объёмом без изменения долей.
-- Источник: Academy of Nutrition and Dietetics, там же.
-- Профиль: 14 % белок, 30 % жир, 56 % углеводы.
UPDATE diets SET protein_share = 0.140, fat_share = 0.300, carb_share = 0.560
 WHERE diet_id = 'vegan';

-- PALEO: высокобелковая модель с умеренным жиром.
-- Источник: Cordain L. "The Paleo Diet Revised" (2011); Frassetto L.A. et al.
-- Eur J Clin Nutr (2009) 63: 947-955.
-- Профиль: 28 % белок, 38 % жир, 34 % углеводы.
UPDATE diets SET protein_share = 0.280, fat_share = 0.380, carb_share = 0.340
 WHERE diet_id = 'paleo';

-- FASTING (православный пост): без молочного/мясного, акцент на крупы, овощи,
-- бобовые. Доля белка снижается, жира — умеренно, углеводов — растёт.
-- Источник: рекомендации НИИ питания РАМН по особенностям питания в период
-- религиозных постов (сводные обзоры в учебниках по нутрициологии).
-- Профиль: 12 % белок, 27 % жир, 61 % углеводы.
UPDATE diets SET protein_share = 0.120, fat_share = 0.270, carb_share = 0.610
 WHERE diet_id = 'fasting';

-- +goose Down

UPDATE diets SET protein_share = NULL, fat_share = NULL, carb_share = NULL
 WHERE diet_id IN ('keto', 'vegetarian', 'vegan', 'paleo', 'fasting');
