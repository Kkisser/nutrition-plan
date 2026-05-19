import type { ShoppingItem } from "./types";
import { authHeader } from "./auth";

export interface UnpricedItem {
  ingredient_name: string;
  requested_amount: number;
  requested_unit: string;
  reason: string;
}

export interface PricingResponse {
  status: "ok" | "partial" | "failed";
  pricing_scope: "nearest_shops_range" | "selected_shop" | "";
  currency: string;
  min_total_price: number;
  max_total_price: number;
  priced_items_count: number;
  unpriced_items_count: number;
  unpriced_items?: UnpricedItem[];
}

export class PricingDisabledError extends Error {
  constructor() {
    super("Сервис цен не подключён в этой среде.");
    this.name = "PricingDisabledError";
  }
}

export class PricingUpstreamError extends Error {
  constructor(msg: string) {
    super(msg);
    this.name = "PricingUpstreamError";
  }
}

export async function estimatePrice(
  shoppingList: ShoppingItem[],
  shopUUID?: string,
): Promise<PricingResponse> {
  const r = await fetch("/api/pricing", {
    method: "POST",
    headers: { "Content-Type": "application/json", ...(await authHeader()) },
    body: JSON.stringify({
      shopping_list: shoppingList,
      shop_uuid: shopUUID || undefined,
    }),
  });

  if (r.status === 401) {
    const { clearAuth } = await import("./auth");
    await clearAuth();
    if (typeof window !== "undefined" && window.location.pathname !== "/login") {
      window.location.assign("/login");
    }
    throw new Error("Сессия истекла. Войдите снова.");
  }
  if (r.status === 503) {
    throw new PricingDisabledError();
  }
  if (r.status === 502) {
    throw new PricingUpstreamError("Сервис цен сейчас недоступен.");
  }
  if (!r.ok) {
    const text = await r.text().catch(() => "");
    throw new Error(`POST /pricing ${r.status}: ${text || r.statusText}`);
  }
  return r.json();
}
