"""Загрузка products и product_micronutrients из CSV."""
from __future__ import annotations

import csv
import uuid
from datetime import datetime
from pathlib import Path

import psycopg


_PRODUCT_UPSERT = """
INSERT INTO products (
    product_id, name, category, kcal_100, protein_100, fat_100, carb_100,
    default_unit, source_name, source_url, fetched_at
) VALUES (
    %(product_id)s, %(name)s, %(category)s, %(kcal_100)s, %(protein_100)s,
    %(fat_100)s, %(carb_100)s, %(default_unit)s,
    %(source_name)s, %(source_url)s, %(fetched_at)s
)
ON CONFLICT (name) DO UPDATE SET
    category     = EXCLUDED.category,
    kcal_100     = EXCLUDED.kcal_100,
    protein_100  = EXCLUDED.protein_100,
    fat_100      = EXCLUDED.fat_100,
    carb_100     = EXCLUDED.carb_100,
    default_unit = EXCLUDED.default_unit,
    source_name  = EXCLUDED.source_name,
    source_url   = EXCLUDED.source_url,
    fetched_at   = EXCLUDED.fetched_at;
"""


_PMN_UPSERT = """
INSERT INTO product_micronutrients (product_id, nutrient_id, amount_100)
VALUES (
    (SELECT product_id FROM products WHERE name = %(product_name)s),
    %(nutrient_id)s,
    %(amount_100)s
)
ON CONFLICT (product_id, nutrient_id) DO UPDATE SET
    amount_100 = EXCLUDED.amount_100;
"""


def _parse_dt(s: str | None) -> datetime | None:
    if not s:
        return None
    # Accept "2026-05-18T00:00:00Z" or with offset.
    return datetime.fromisoformat(s.replace("Z", "+00:00"))


def _empty_to_none(value: str) -> str | None:
    return value if value else None


def load_products(conn: psycopg.Connection, csv_path: Path) -> int:
    count = 0
    with csv_path.open(encoding="utf-8") as f, conn.cursor() as cur:
        reader = csv.DictReader(f)
        for row in reader:
            cur.execute(
                _PRODUCT_UPSERT,
                {
                    "product_id":   str(uuid.uuid4()),
                    "name":         row["name"],
                    "category":     _empty_to_none(row.get("category", "")),
                    "kcal_100":     row["kcal_100"],
                    "protein_100":  row["protein_100"],
                    "fat_100":      row["fat_100"],
                    "carb_100":     row["carb_100"],
                    "default_unit": row["default_unit"],
                    "source_name":  _empty_to_none(row.get("source_name", "")),
                    "source_url":   _empty_to_none(row.get("source_url", "")),
                    "fetched_at":   _parse_dt(row.get("fetched_at")),
                },
            )
            count += 1
    return count


def load_product_micronutrients(conn: psycopg.Connection, csv_path: Path) -> int:
    count = 0
    missing: list[str] = []
    with csv_path.open(encoding="utf-8") as f, conn.cursor() as cur:
        reader = csv.DictReader(f)
        for row in reader:
            cur.execute(
                "SELECT 1 FROM products WHERE name = %s",
                (row["product_name"],),
            )
            if cur.fetchone() is None:
                missing.append(row["product_name"])
                continue
            cur.execute(
                _PMN_UPSERT,
                {
                    "product_name": row["product_name"],
                    "nutrient_id":  row["nutrient_id"],
                    "amount_100":   row["amount_100"],
                },
            )
            count += 1
    if missing:
        unique = sorted(set(missing))
        raise RuntimeError(
            f"product_micronutrients references unknown products: {unique[:5]}"
            f"{' …' if len(unique) > 5 else ''}. Load products.csv first."
        )
    return count
