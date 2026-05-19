import { describe, it, expect } from "vitest";
import { deriveKfa, type Survey } from "./kfa";

// Дерево решений deriveKfa — критический путь анкеты. Если оно ошибётся,
// у пользователя поедут целевые КБЖУ. Источник логики: docs/АНКЕТА_активности.md.

describe("deriveKfa", () => {
  it("IV для тяжёлой физической работы вне зависимости от тренировок", () => {
    const s: Survey = { q1: "heavy_physical", q3: "none", q4: "" };
    expect(deriveKfa(s)).toBe("IV");
  });

  it("IV для 6+ интенсивных тренировок при сидячей работе", () => {
    const s: Survey = { q1: "sedentary", q3: "6_plus", q4: "intense" };
    expect(deriveKfa(s)).toBe("IV");
  });

  it("III для частого перемещения", () => {
    const s: Survey = { q1: "frequent_movement", q3: "none", q4: "" };
    expect(deriveKfa(s)).toBe("III");
  });

  it("III для 3-5 умеренных тренировок при сидячей работе", () => {
    const s: Survey = { q1: "sedentary", q3: "3_to_5", q4: "moderate" };
    expect(deriveKfa(s)).toBe("III");
  });

  it("II для standing_low даже без тренировок", () => {
    const s: Survey = { q1: "standing_low", q3: "none", q4: "" };
    expect(deriveKfa(s)).toBe("II");
  });

  it("II для 1-2 тренировок при сидячей работе", () => {
    const s: Survey = { q1: "sedentary", q3: "1_to_2", q4: "light" };
    expect(deriveKfa(s)).toBe("II");
  });

  it("II для частых, но лёгких тренировок", () => {
    const s: Survey = { q1: "sedentary", q3: "6_plus", q4: "light" };
    expect(deriveKfa(s)).toBe("II");
  });

  it("I по умолчанию для сидячей работы без тренировок", () => {
    const s: Survey = { q1: "sedentary", q3: "none", q4: "" };
    expect(deriveKfa(s)).toBe("I");
  });

  it("приоритет q1 над q3: heavy_physical всегда IV даже при light нагрузке", () => {
    const s: Survey = { q1: "heavy_physical", q3: "1_to_2", q4: "light" };
    expect(deriveKfa(s)).toBe("IV");
  });
});
