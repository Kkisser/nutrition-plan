// Каталог пользовательских строк и маппингов «значение enum → label».
// Точка входа для будущей i18n: при подключении локалей вместо строк
// можно положить функцию t(key) с lookup по выбранной локали.
//
// Сейчас одна локаль — ru-RU; ключи выбраны так, чтобы при появлении
// английского варианта переменные на местах вызова не менялись.

import type { Meal, Unit, Kfa, Allergen } from "../api/types";

export const MEAL_LABEL: Record<Meal, string> = {
  breakfast: "Завтрак",
  lunch: "Обед",
  dinner: "Ужин",
  snack: "Перекус",
};

// Использование в Survey (нижний регистр для контекста «выберите __»).
export const MEAL_LABEL_LOWER: Record<Meal, string> = {
  breakfast: "завтрак",
  lunch: "обед",
  dinner: "ужин",
  snack: "перекус",
};

export const UNIT_LABEL: Record<Unit, string> = {
  g: "г",
  ml: "мл",
  pcs: "шт",
};

export const KFA_LABEL: Record<Kfa, string> = {
  I: "I — низкая активность",
  II: "II — умеренная активность",
  III: "III — высокая активность",
  IV: "IV — очень высокая активность",
};

export const ALLERGEN_LABEL: Record<Allergen, string> = {
  milk: "молоко",
  eggs: "яйца",
  fish: "рыба",
  gluten: "глютен",
  peanut: "арахис",
  sesame: "кунжут",
  shellfish: "морепродукты",
  soy: "соя",
  nuts: "орехи",
};

// Фиксированный порядок категорий в shopping/family/pricing.
// При выводе общего списка покупок этим порядком определяется верхний-вниз
// маршрут по магазину (молочка → мясо → овощи → ...).
export const CATEGORY_ORDER: ReadonlyArray<string> = [
  "Молочные",
  "Мясо",
  "Рыба",
  "Яйца",
  "Овощи",
  "Фрукты",
  "Крупы",
  "Хлеб",
  "Бобовые",
  "Орехи",
  "Жиры",
];

export function categoryRank(cat?: string): number {
  if (!cat) return 999;
  const i = CATEGORY_ORDER.indexOf(cat);
  return i >= 0 ? i : 998;
}

export const PRICING_SCOPE_LABEL: Record<string, string> = {
  nearest_shops_range: "ближайшие магазины Пятёрочки",
  selected_shop: "выбранный магазин",
};
