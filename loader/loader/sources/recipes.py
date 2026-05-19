"""Загрузка каталога блюд: recipes + recipe_ingredients + diet_compat + allergens."""
from __future__ import annotations

import csv
import uuid
from pathlib import Path

import psycopg


_RECIPE_UPSERT = """
INSERT INTO recipes (
    recipe_id, external_id, name, instruction,
    cook_time_min, base_portions, meal_type
) VALUES (
    %(recipe_id)s, %(external_id)s, %(name)s, %(instruction)s,
    %(cook_time_min)s, %(base_portions)s, %(meal_type)s
)
ON CONFLICT (external_id) DO UPDATE SET
    name          = EXCLUDED.name,
    instruction   = EXCLUDED.instruction,
    cook_time_min = EXCLUDED.cook_time_min,
    base_portions = EXCLUDED.base_portions,
    meal_type     = EXCLUDED.meal_type;
"""

_INGREDIENT_UPSERT = """
INSERT INTO recipe_ingredients (recipe_id, product_id, amount, unit)
VALUES (
    (SELECT recipe_id  FROM recipes  WHERE external_id = %(recipe_external_id)s),
    (SELECT product_id FROM products WHERE name        = %(product_name)s),
    %(amount)s,
    %(unit)s
)
ON CONFLICT (recipe_id, product_id) DO UPDATE SET
    amount = EXCLUDED.amount,
    unit   = EXCLUDED.unit;
"""

_DIET_COMPAT_INSERT = """
INSERT INTO recipe_diet_compat (recipe_id, diet_id)
VALUES (
    (SELECT recipe_id FROM recipes WHERE external_id = %(recipe_external_id)s),
    %(diet_id)s
)
ON CONFLICT (recipe_id, diet_id) DO NOTHING;
"""

_ALLERGEN_INSERT = """
INSERT INTO recipe_allergens (recipe_id, allergen)
VALUES (
    (SELECT recipe_id FROM recipes WHERE external_id = %(recipe_external_id)s),
    %(allergen)s
)
ON CONFLICT (recipe_id, allergen) DO NOTHING;
"""


def _empty_to_none(v: str) -> str | None:
    return v if v else None


def load_recipes(conn: psycopg.Connection, csv_path: Path) -> int:
    count = 0
    with csv_path.open(encoding="utf-8") as f, conn.cursor() as cur:
        for row in csv.DictReader(f):
            cur.execute(_RECIPE_UPSERT, {
                "recipe_id":     str(uuid.uuid4()),
                "external_id":   row["external_id"],
                "name":          row["name"],
                "instruction":   row["instruction"],
                "cook_time_min": _empty_to_none(row.get("cook_time_min", "")),
                "base_portions": row.get("base_portions") or 1,
                "meal_type":     row["meal_type"],
            })
            count += 1
    return count


def _check_refs(
    cur: psycopg.Cursor, csv_path: Path, refs: dict[str, tuple[str, str]]
) -> None:
    """Validate FK references exist before inserting.

    refs: column → (sql to test existence, human description)
    """
    pass  # placeholder — каждая функция делает собственную проверку.


def load_recipe_ingredients(conn: psycopg.Connection, csv_path: Path) -> int:
    count = 0
    missing_recipes: set[str] = set()
    missing_products: set[str] = set()
    with csv_path.open(encoding="utf-8") as f, conn.cursor() as cur:
        for row in csv.DictReader(f):
            cur.execute("SELECT 1 FROM recipes WHERE external_id = %s",
                        (row["recipe_external_id"],))
            if cur.fetchone() is None:
                missing_recipes.add(row["recipe_external_id"])
                continue
            cur.execute("SELECT 1 FROM products WHERE name = %s",
                        (row["product_name"],))
            if cur.fetchone() is None:
                missing_products.add(row["product_name"])
                continue
            cur.execute(_INGREDIENT_UPSERT, {
                "recipe_external_id": row["recipe_external_id"],
                "product_name":       row["product_name"],
                "amount":             row["amount"],
                "unit":               row["unit"],
            })
            count += 1
    problems = []
    if missing_recipes:
        problems.append(f"unknown recipes: {sorted(missing_recipes)}")
    if missing_products:
        problems.append(f"unknown products: {sorted(missing_products)}")
    if problems:
        raise RuntimeError("recipe_ingredients FK errors — " + "; ".join(problems))
    return count


def load_recipe_diet_compat(conn: psycopg.Connection, csv_path: Path) -> int:
    count = 0
    with csv_path.open(encoding="utf-8") as f, conn.cursor() as cur:
        for row in csv.DictReader(f):
            cur.execute(_DIET_COMPAT_INSERT, {
                "recipe_external_id": row["recipe_external_id"],
                "diet_id":            row["diet_id"],
            })
            count += 1
    return count


def load_recipe_allergens(conn: psycopg.Connection, csv_path: Path) -> int:
    count = 0
    with csv_path.open(encoding="utf-8") as f, conn.cursor() as cur:
        for row in csv.DictReader(f):
            cur.execute(_ALLERGEN_INSERT, {
                "recipe_external_id": row["recipe_external_id"],
                "allergen":           row["allergen"],
            })
            count += 1
    return count
