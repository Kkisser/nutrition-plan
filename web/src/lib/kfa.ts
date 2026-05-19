// Дерево решений «ответы анкеты → группа КФА» по docs/АНКЕТА_активности.md
// (вариант B — без Q2/Q5; в источниках обосновано через WHO 2020 + EFSA 2013).
// Параллельная копия core/internal/activity/kfa.go — фронт должен уметь
// считать КФА офлайн (PWA), поэтому дублирование оправдано.

import type { Kfa } from "../api/types";

export type Q1 = "sedentary" | "standing_low" | "frequent_movement" | "heavy_physical";
export type Q3 = "none" | "1_to_2" | "3_to_5" | "6_plus";
export type Q4 = "" | "light" | "moderate" | "intense";

export interface Survey {
  q1: Q1;
  q3: Q3;
  q4: Q4; // пусто, если q3 == "none"
}

export function deriveKfa(s: Survey): Kfa {
  // IV
  if (s.q1 === "heavy_physical") return "IV";
  if (s.q3 === "6_plus" && s.q4 === "intense") return "IV";

  // III
  if (s.q1 === "frequent_movement") return "III";
  if (s.q3 === "3_to_5" && (s.q4 === "moderate" || s.q4 === "intense")) return "III";

  // II
  if (s.q1 === "standing_low") return "II";
  if (s.q3 === "1_to_2") return "II";
  if ((s.q3 === "3_to_5" || s.q3 === "6_plus") && s.q4 === "light") return "II";

  // I по умолчанию
  return "I";
}

// Тексты вариантов ответов из АНКЕТА_активности.md.
export const Q1_OPTIONS: { value: Q1; label: string; hint?: string }[] = [
  { value: "sedentary",         label: "В основном сижу",
    hint: "работа за компьютером, учёба, длительное пребывание дома без активного движения" },
  { value: "standing_low",      label: "В основном стою, но мало перемещаюсь",
    hint: "парикмахер, продавец за стойкой, преподаватель у доски" },
  { value: "frequent_movement", label: "Часто хожу или перемещаюсь",
    hint: "официант, продавец в торговом зале, медработник, курьер" },
  { value: "heavy_physical",    label: "Выполняю физически тяжёлую работу",
    hint: "склад, стройка, производство, грузовые работы" },
];

export const Q3_OPTIONS: { value: Q3; label: string }[] = [
  { value: "none",     label: "Нет таких нагрузок" },
  { value: "1_to_2",   label: "1–2 раза в неделю" },
  { value: "3_to_5",   label: "3–5 раз в неделю" },
  { value: "6_plus",   label: "6 и более раз в неделю" },
];

export const Q4_OPTIONS: { value: Exclude<Q4, "">; label: string; hint?: string }[] = [
  { value: "light",    label: "Лёгкая",
    hint: "спокойная зарядка, растяжка, неспешная йога, прогулка в умеренном темпе" },
  { value: "moderate", label: "Умеренная",
    hint: "фитнес, плавание, велосипед, быстрая ходьба" },
  { value: "intense",  label: "Интенсивная",
    hint: "силовая, бег, интервальные тренировки, единоборства" },
];
