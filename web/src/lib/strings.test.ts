import { describe, it, expect } from "vitest";
import {
  MEAL_LABEL,
  UNIT_LABEL,
  KFA_LABEL,
  CATEGORY_ORDER,
  categoryRank,
} from "./strings";

describe("strings dictionaries", () => {
  it("MEAL_LABEL покрывает все 4 приёма пищи", () => {
    expect(MEAL_LABEL.breakfast).toBe("Завтрак");
    expect(MEAL_LABEL.lunch).toBe("Обед");
    expect(MEAL_LABEL.dinner).toBe("Ужин");
    expect(MEAL_LABEL.snack).toBe("Перекус");
  });

  it("UNIT_LABEL соответствует enum Unit", () => {
    expect(UNIT_LABEL.g).toBe("г");
    expect(UNIT_LABEL.ml).toBe("мл");
    expect(UNIT_LABEL.pcs).toBe("шт");
  });

  it("KFA_LABEL покрывает все 4 группы", () => {
    expect(KFA_LABEL.I).toMatch(/I/);
    expect(KFA_LABEL.II).toMatch(/II/);
    expect(KFA_LABEL.III).toMatch(/III/);
    expect(KFA_LABEL.IV).toMatch(/IV/);
  });
});

describe("categoryRank", () => {
  it("ранжирует известные категории по фиксированному порядку", () => {
    expect(categoryRank("Молочные")).toBeLessThan(categoryRank("Мясо"));
    expect(categoryRank("Мясо")).toBeLessThan(categoryRank("Овощи"));
    expect(categoryRank("Овощи")).toBeLessThan(categoryRank("Крупы"));
    expect(categoryRank("Крупы")).toBeLessThan(categoryRank("Жиры"));
  });

  it("неизвестная категория ранжируется ниже известных", () => {
    const known = categoryRank("Молочные");
    expect(categoryRank("Какая-то новая категория")).toBeGreaterThan(known);
  });

  it("пустая категория опускается в конец", () => {
    expect(categoryRank(undefined)).toBe(999);
    expect(categoryRank("")).toBe(999);
  });

  it("все категории CATEGORY_ORDER имеют уникальный ранг и идут по порядку", () => {
    const ranks = CATEGORY_ORDER.map((c) => categoryRank(c));
    for (let i = 1; i < ranks.length; i++) {
      expect(ranks[i]).toBeGreaterThan(ranks[i - 1]);
    }
  });
});
