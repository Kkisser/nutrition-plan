import type { ShoppingItem } from "../api/types";

// Объединение двух shopping_list для §15 функционала (семейное меню).
// Ключ агрегации — name (lower+trim) + unit, как на серверной стороне в
// shopping.Aggregate. Если у одного из источников есть category, а у
// другого — нет, побеждает та, что попалась первой.
export function combineShopping(
  a: ShoppingItem[],
  b: ShoppingItem[],
): ShoppingItem[] {
  const acc = new Map<string, ShoppingItem>();
  for (const item of [...a, ...b]) {
    const k = `${item.ingredient_name.trim().toLowerCase()}::${item.unit}`;
    const ex = acc.get(k);
    if (ex) {
      ex.amount = +(ex.amount + item.amount).toFixed(2);
      if (!ex.category && item.category) ex.category = item.category;
    } else {
      acc.set(k, { ...item });
    }
  }
  return Array.from(acc.values()).sort((x, y) =>
    x.ingredient_name.localeCompare(y.ingredient_name, "ru"),
  );
}
