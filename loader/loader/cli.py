"""CLI: loader load-all | load-products | load-norms | load-recipes."""
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
from loader.sources.recipes import (
    load_recipe_allergens,
    load_recipe_diet_compat,
    load_recipe_ingredients,
    load_recipes,
)


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
    """Загрузка всех справочников и каталога блюд в порядке зависимостей."""
    required = {
        "micronutrients":         data_dir / "micronutrients.csv",
        "products":               data_dir / "products.csv",
        "product_micronutrients": data_dir / "product_micronutrients.csv",
        "micronutrient_norms":    data_dir / "micronutrient_norms.csv",
        "energy_norms":           data_dir / "energy_norms.csv",
    }
    missing = [k for k, p in required.items() if not p.exists()]
    if missing:
        raise click.ClickException(f"missing CSVs: {missing}")

    optional = {
        "recipes":             data_dir / "recipes.csv",
        "recipe_ingredients":  data_dir / "recipe_ingredients.csv",
        "recipe_diet_compat":  data_dir / "recipe_diet_compat.csv",
        "recipe_allergens":    data_dir / "recipe_allergens.csv",
    }

    with connect() as conn:
        n1 = load_micronutrients(conn, required["micronutrients"])
        n2 = load_products(conn, required["products"])
        n3 = load_product_micronutrients(conn, required["product_micronutrients"])
        n4 = load_micronutrient_norms(conn, required["micronutrient_norms"])
        n5 = load_energy_norms(conn, required["energy_norms"])

        nr = ni = nd = na = 0
        if optional["recipes"].exists():
            nr = load_recipes(conn, optional["recipes"])
        if optional["recipe_ingredients"].exists():
            ni = load_recipe_ingredients(conn, optional["recipe_ingredients"])
        if optional["recipe_diet_compat"].exists():
            nd = load_recipe_diet_compat(conn, optional["recipe_diet_compat"])
        if optional["recipe_allergens"].exists():
            na = load_recipe_allergens(conn, optional["recipe_allergens"])

    click.echo(f"micronutrients:         {n1}")
    click.echo(f"products:               {n2}")
    click.echo(f"product_micronutrients: {n3}")
    click.echo(f"micronutrient_norms:    {n4}")
    click.echo(f"energy_norms:           {n5}")
    click.echo(f"recipes:                {nr}")
    click.echo(f"recipe_ingredients:     {ni}")
    click.echo(f"recipe_diet_compat:     {nd}")
    click.echo(f"recipe_allergens:       {na}")


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


@main.command("load-recipes")
@click.option(
    "--file",
    "csv_file",
    type=click.Path(exists=True, dir_okay=False, path_type=Path),
    required=True,
)
@click.option(
    "--kind",
    type=click.Choice(["recipes", "ingredients", "diet_compat", "allergens"]),
    required=True,
)
def cmd_load_recipes(csv_file: Path, kind: str) -> None:
    with connect() as conn:
        if kind == "recipes":
            n = load_recipes(conn, csv_file)
        elif kind == "ingredients":
            n = load_recipe_ingredients(conn, csv_file)
        elif kind == "diet_compat":
            n = load_recipe_diet_compat(conn, csv_file)
        else:
            n = load_recipe_allergens(conn, csv_file)
    click.echo(f"loaded: {n}")


if __name__ == "__main__":
    main()
