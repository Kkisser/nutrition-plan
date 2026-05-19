import { describe, it, expect } from "vitest";
import { combineShopping } from "./shopping";
import type { ShoppingItem } from "../api/types";

const I = (name: string, amount: number, unit: "g" | "ml" | "pcs", category?: string): ShoppingItem => ({
  ingredient_name: name,
  amount,
  unit,
  category,
});

describe("combineShopping", () => {
  it("суммирует одноимённые продукты в одинаковых единицах", () => {
    const a = [I("Гречка", 200, "g")];
    const b = [I("Гречка", 150, "g")];
    const r = combineShopping(a, b);
    expect(r).toHaveLength(1);
    expect(r[0].amount).toBe(350);
  });

  it("разные единицы не смешиваются", () => {
    const a = [I("Молоко", 500, "ml")];
    const b = [I("Молоко", 200, "g")];
    const r = combineShopping(a, b);
    expect(r).toHaveLength(2);
  });

  it("игнорирует регистр и пробелы в имени", () => {
    const a = [I("  Куриное филе  ", 200, "g")];
    const b = [I("куриное филе", 100, "g")];
    const r = combineShopping(a, b);
    expect(r).toHaveLength(1);
    expect(r[0].amount).toBe(300);
  });

  it("сохраняет категорию, если она есть хотя бы у одного источника", () => {
    const a = [I("Гречка", 100, "g")];
    const b = [I("Гречка", 100, "g", "Крупы")];
    const r = combineShopping(a, b);
    expect(r[0].category).toBe("Крупы");
  });

  it("сортирует результат по русскому алфавиту", () => {
    const a = [I("Яблоко", 100, "g"), I("Гречка", 100, "g")];
    const b = [I("Молоко", 200, "ml")];
    const r = combineShopping(a, b);
    expect(r.map((x) => x.ingredient_name)).toEqual([
      "Гречка",
      "Молоко",
      "Яблоко",
    ]);
  });

  it("результат округлён до 2 знаков (чтобы не было 0.30000004)", () => {
    const a = [I("Масло", 0.1, "ml")];
    const b = [I("Масло", 0.2, "ml")];
    const r = combineShopping(a, b);
    expect(r[0].amount).toBe(0.3);
  });

  it("пустые списки не падают", () => {
    expect(combineShopping([], [])).toEqual([]);
    expect(combineShopping([I("Яблоко", 50, "g")], [])).toHaveLength(1);
  });
});
