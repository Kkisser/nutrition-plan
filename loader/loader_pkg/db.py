"""PostgreSQL connection helper."""
from __future__ import annotations

import os
from contextlib import contextmanager
from typing import Iterator

import psycopg


def dsn_from_env() -> str:
    dsn = os.environ.get("DATABASE_DSN")
    if not dsn:
        raise RuntimeError(
            "DATABASE_DSN not set. Example: "
            "postgres://Kirill@localhost:5432/nutrition_dev"
        )
    return dsn


@contextmanager
def connect(dsn: str | None = None) -> Iterator[psycopg.Connection]:
    conn = psycopg.connect(dsn or dsn_from_env(), autocommit=False)
    try:
        yield conn
        conn.commit()
    except Exception:
        conn.rollback()
        raise
    finally:
        conn.close()
