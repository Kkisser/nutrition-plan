"""CLI: loader load-all | load-products | load-norms."""
from __future__ import annotations

from pathlib import Path

import click

from loader.db import connect
from loader.sources.norms import (
    load_energy_norms,
    load_micronutrient_norms,
    load_micronutrients,
)
from loader.sources.products import load_product_micronutrients, load_products


@click.group(help="CSV → PostgreSQL loader.")
def main() -> None:
    pass


@main.command("load-all")
@click.option(
    "--data-dir",
    type=click.Path(exists=True, file_okay=False, path_type=Path),
    required=True,
    help="Каталог с CSV-файлами (стандартные имена).",
)
def load_all(data_dir: Path) -> None:
    """Загрузка всех справочников в порядке зависимостей."""
    files = {
        "micronutrients":         data_dir / "micronutrients.csv",
        "products":               data_dir / "products.csv",
        "product_micronutrients": data_dir / "product_micronutrients.csv",
        "micronutrient_norms":    data_dir / "micronutrient_norms.csv",
        "energy_norms":           data_dir / "energy_norms.csv",
    }
    missing = [k for k, p in files.items() if not p.exists()]
    if missing:
        raise click.ClickException(f"missing CSVs: {missing}")

    with connect() as conn:
        n1 = load_micronutrients(conn, files["micronutrients"])
        n2 = load_products(conn, files["products"])
        n3 = load_product_micronutrients(conn, files["product_micronutrients"])
        n4 = load_micronutrient_norms(conn, files["micronutrient_norms"])
        n5 = load_energy_norms(conn, files["energy_norms"])

    click.echo(f"micronutrients:         {n1}")
    click.echo(f"products:               {n2}")
    click.echo(f"product_micronutrients: {n3}")
    click.echo(f"micronutrient_norms:    {n4}")
    click.echo(f"energy_norms:           {n5}")


@main.command("load-products")
@click.option(
    "--file",
    "csv_file",
    type=click.Path(exists=True, dir_okay=False, path_type=Path),
    required=True,
)
@click.option(
    "--kind",
    type=click.Choice(["products", "micronutrients"]),
    default="products",
    help="products = таблица products; micronutrients = product_micronutrients.",
)
def cmd_load_products(csv_file: Path, kind: str) -> None:
    with connect() as conn:
        if kind == "products":
            n = load_products(conn, csv_file)
        else:
            n = load_product_micronutrients(conn, csv_file)
    click.echo(f"loaded: {n}")


@main.command("load-norms")
@click.option(
    "--file",
    "csv_file",
    type=click.Path(exists=True, dir_okay=False, path_type=Path),
    required=True,
)
@click.option(
    "--kind",
    type=click.Choice(["energy", "micronutrients", "micronutrient_norms"]),
    required=True,
)
def cmd_load_norms(csv_file: Path, kind: str) -> None:
    with connect() as conn:
        if kind == "energy":
            n = load_energy_norms(conn, csv_file)
        elif kind == "micronutrients":
            n = load_micronutrients(conn, csv_file)
        else:
            n = load_micronutrient_norms(conn, csv_file)
    click.echo(f"loaded: {n}")


if __name__ == "__main__":
    main()
