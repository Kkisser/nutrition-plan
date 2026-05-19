"""Загрузка нормативных справочников: micronutrients, micronutrient_norms, energy_norms."""
from __future__ import annotations

import csv
from pathlib import Path

import psycopg


_MICRONUTRIENT_UPSERT = """
INSERT INTO micronutrients (nutrient_id, name, norm_unit, ul_value)
VALUES (%(nutrient_id)s, %(name)s, %(norm_unit)s, %(ul_value)s)
ON CONFLICT (nutrient_id) DO UPDATE SET
    name      = EXCLUDED.name,
    norm_unit = EXCLUDED.norm_unit,
    ul_value  = EXCLUDED.ul_value;
"""

_MICRONUTRIENT_NORM_UPSERT = """
INSERT INTO micronutrient_norms (nutrient_id, sex, age_group, norm_value)
VALUES (%(nutrient_id)s, %(sex)s, %(age_group)s, %(norm_value)s)
ON CONFLICT (nutrient_id, sex, age_group) DO UPDATE SET
    norm_value = EXCLUDED.norm_value;
"""

_ENERGY_NORM_UPSERT = """
INSERT INTO energy_norms (
    sex, age_group, kfa_group,
    kcal_norm, protein_g_norm, fat_g_norm, carb_g_norm
) VALUES (
    %(sex)s, %(age_group)s, %(kfa_group)s,
    %(kcal_norm)s, %(protein_g_norm)s, %(fat_g_norm)s, %(carb_g_norm)s
)
ON CONFLICT (sex, age_group, kfa_group) DO UPDATE SET
    kcal_norm       = EXCLUDED.kcal_norm,
    protein_g_norm  = EXCLUDED.protein_g_norm,
    fat_g_norm      = EXCLUDED.fat_g_norm,
    carb_g_norm     = EXCLUDED.carb_g_norm;
"""


def _empty_to_none(value: str) -> str | None:
    return value if value else None


def load_micronutrients(conn: psycopg.Connection, csv_path: Path) -> int:
    count = 0
    with csv_path.open(encoding="utf-8") as f, conn.cursor() as cur:
        for row in csv.DictReader(f):
            cur.execute(_MICRONUTRIENT_UPSERT, {
                "nutrient_id": row["nutrient_id"],
                "name":        row["name"],
                "norm_unit":   row["norm_unit"],
                "ul_value":    _empty_to_none(row.get("ul_value", "")),
            })
            count += 1
    return count


def load_micronutrient_norms(conn: psycopg.Connection, csv_path: Path) -> int:
    count = 0
    with csv_path.open(encoding="utf-8") as f, conn.cursor() as cur:
        for row in csv.DictReader(f):
            cur.execute(_MICRONUTRIENT_NORM_UPSERT, {
                "nutrient_id": row["nutrient_id"],
                "sex":         row["sex"],
                "age_group":   row["age_group"],
                "norm_value":  row["norm_value"],
            })
            count += 1
    return count


def load_energy_norms(conn: psycopg.Connection, csv_path: Path) -> int:
    count = 0
    with csv_path.open(encoding="utf-8") as f, conn.cursor() as cur:
        for row in csv.DictReader(f):
            cur.execute(_ENERGY_NORM_UPSERT, {
                "sex":            row["sex"],
                "age_group":      row["age_group"],
                "kfa_group":      row["kfa_group"],
                "kcal_norm":      row["kcal_norm"],
                "protein_g_norm": row["protein_g_norm"],
                "fat_g_norm":     row["fat_g_norm"],
                "carb_g_norm":    row["carb_g_norm"],
            })
            count += 1
    return count
