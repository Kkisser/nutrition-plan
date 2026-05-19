import type { PlanRequest, PlanResponse } from "./types";
import { authHeader, clearAuth } from "./auth";

const BASE = "/api";

async function call<T>(path: string, init: RequestInit): Promise<T> {
  const r = await fetch(`${BASE}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(await authHeader()),
      ...(init.headers ?? {}),
    },
  });
  if (r.status === 401) {
    // токен невалиден — чистим и форсируем перерисовку шапки/маршрута.
    // Без редиректа пользователь оставался на /plan и видел кнопки,
    // которые молча проваливались по 401 (см. issue: «через раз не работает»).
    await clearAuth();
    if (typeof window !== "undefined" && window.location.pathname !== "/login") {
      window.location.assign("/login");
    }
    throw new Error("Сессия истекла. Войдите снова.");
  }
  if (!r.ok) {
    const text = await r.text().catch(() => "");
    throw new Error(`${path} ${r.status}: ${text || r.statusText}`);
  }
  return r.json() as Promise<T>;
}

export async function postPlan(req: PlanRequest): Promise<PlanResponse> {
  return call("/plan", { method: "POST", body: JSON.stringify(req) });
}

export async function health(): Promise<{ status: string }> {
  const r = await fetch(`${BASE}/health`);
  if (!r.ok) throw new Error("health failed");
  return r.json();
}
