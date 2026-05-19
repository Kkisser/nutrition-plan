-- +goose Up
-- Назначение: коррекция профиля KETO с 25/70/5 на 20/75/5.
--
-- Причина: 25 % белка — верхняя граница «модифицированной» кетогенной диеты.
-- Классическая «nutritional ketosis» по Volek/Phinney и обзорам клинической
-- практики — это 20 % белка / 75 % жира / 5 % углеводов. См. источники:
--   * Volek J.S., Phinney S.D. «The Art and Science of Low Carbohydrate
--     Living». — Beyond Obesity LLC, 2011. — ISBN 978-0983490708.
--   * Paoli A. «Ketogenic diet for obesity: friend or foe?». —
--     Int. J. Environ. Res. Public Health, 2014, 11(2): 2092-2107.
--     doi: 10.3390/ijerph110202092.
--   * Sharon F. Daley et al. «The Ketogenic Diet: Clinical Applications,
--     Evidence-based Indications, and Implementation». — StatPearls,
--     2026. NCBI Bookshelf NBK499830.
--
-- Корректировка обсуждалась с научным руководителем; подробности — в
-- ИСТОЧНИКИ_ОБОСНОВАНИЯ.md §3.1.

UPDATE diets SET protein_share = 0.200, fat_share = 0.750, carb_share = 0.050
 WHERE diet_id = 'keto';

-- +goose Down

UPDATE diets SET protein_share = 0.250, fat_share = 0.700, carb_share = 0.050
 WHERE diet_id = 'keto';
