import type { Meal } from "./types";
import { authHeader } from "./auth";
import { loadExcludedDishes, loadProfile } from "./persist";

export interface CatalogItem {
  dish_id: string;
  name: string;
  meal: Meal;
  kcal: number;
  protein_g: number;
  fat_g: number;
  carb_g: number;
  cook_time_min: number;
}

export async function getCatalog(meal: Meal): Promise<CatalogItem[]> {
  const profile = await loadProfile();
  if (!profile) throw new Error("Профиль не заполнен.");
  const excluded = await loadExcludedDishes();

  const params = new URLSearchParams({
    meal,
    diet: profile.diet_type,
  });
  if (profile.allergens.length) params.set("allergens", profile.allergens.join(","));
  if (profile.excluded_products.length)
    params.set("excluded_products", profile.excluded_products.join(","));
  if (excluded.length) params.set("excluded_dishes", excluded.join(","));

  const r = await fetch(`/api/catalog?${params.toString()}`, {
    headers: await authHeader(),
  });
  if (r.status === 401) {
    const { clearAuth } = await import("./auth");
    await clearAuth();
    if (typeof window !== "undefined" && window.location.pathname !== "/login") {
      window.location.assign("/login");
    }
    throw new Error("Сессия истекла. Войдите снова.");
  }
  if (!r.ok) {
    const t = await r.text().catch(() => "");
    throw new Error(`GET /catalog ${r.status}: ${t || r.statusText}`);
  }
  const json = (await r.json()) as { items: CatalogItem[] };
  return json.items;
}
