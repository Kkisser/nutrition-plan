import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { useMutation } from "@tanstack/react-query";
import {
  PricingDisabledError,
  PricingUpstreamError,
  estimatePrice,
  type PricingResponse,
} from "../api/pricing";
import { loadLastPlan } from "../api/persist";
import type { PlanResponse, ShoppingItem } from "../api/types";
import {
  UNIT_LABEL,
  categoryRank,
  PRICING_SCOPE_LABEL as SCOPE_LABEL,
} from "../lib/strings";

function ShoppingFallback({ items }: { items: ShoppingItem[] }) {
  const grouped: Record<string, ShoppingItem[]> = {};
  for (const it of items) {
    (grouped[it.category || "Прочее"] ??= []).push(it);
  }
  const cats = Object.keys(grouped).sort(
    (a, b) => categoryRank(a) - categoryRank(b) || a.localeCompare(b, "ru"),
  );
  return (
    <div className="card" style={{ background: "#fafafa", marginTop: 12 }}>
      <h4 style={{ margin: "0 0 8px" }}>
        Список покупок (без цен)
      </h4>
      <p style={{ fontSize: 12, color: "#666", margin: "0 0 8px" }}>
        Цены сейчас недоступны, но сам список рассчитан из вашего плана и
        готов к использованию.
      </p>
      {cats.map((cat) => (
        <div key={cat} style={{ marginTop: 8 }}>
          <b style={{ display: "block", marginBottom: 4 }}>{cat}</b>
          {grouped[cat].map((it) => (
            <div key={`${it.ingredient_name}-${it.unit}`} style={{ fontSize: 13 }}>
              {it.ingredient_name} —{" "}
              {it.amount.toFixed(it.unit === "pcs" ? 0 : 1)}{" "}
              {UNIT_LABEL[it.unit] ?? it.unit}
            </div>
          ))}
        </div>
      ))}
    </div>
  );
}

export default function Pricing() {
  const [plan, setPlan] = useState<PlanResponse | null>(null);
  const [shopUUID, setShopUUID] = useState("");

  useEffect(() => {
    loadLastPlan().then((p) => p && setPlan(p));
  }, []);

  const m = useMutation({
    mutationFn: async (): Promise<PricingResponse> => {
      if (!plan) throw new Error("Нет плана");
      return estimatePrice(plan.shopping_list, shopUUID.trim() || undefined);
    },
  });

  if (!plan) {
    return (
      <>
        <h2>Оценка стоимости</h2>
        <p>
          План ещё не сформирован. Перейдите на{" "}
          <Link to="/plan">экран плана</Link> и нажмите «Сформировать».
        </p>
      </>
    );
  }

  return (
    <>
      <h2>Оценка стоимости · {plan.week_ref}</h2>
      <p style={{ color: "#666", fontSize: 13, margin: "4px 0 12px" }}>
        Ориентировочная вилка стоимости списка покупок через price-service.
        Цена — справочная по данным Edadil (Пятёрочка). Минимизация и
        распределение по нескольким магазинам не выполняются.
      </p>

      <div className="card">
        <p style={{ margin: "0 0 8px" }}>
          <b>{plan.shopping_list.length}</b> позиций в списке покупок.
        </p>

        <div className="field">
          <label>
            UUID магазина Пятёрочки (опционально)
            <br />
            <small style={{ color: "#888" }}>
              Если не указан — вилка по ближайшим магазинам. UUID берётся из
              Edadil; в полноценном UI выбор магазина будет встроен.
            </small>
          </label>
          <input
            type="text"
            value={shopUUID}
            onChange={(e) => setShopUUID(e.target.value)}
            placeholder="052761c1-6775-4ac4-8d9d-2f03c974932b"
          />
        </div>

        <button onClick={() => m.mutate()} disabled={m.isPending}>
          {m.isPending ? "Считаю…" : "Оценить стоимость"}
        </button>
      </div>

      {m.isError && (
        <>
          <div className="warn">
            {m.error instanceof PricingDisabledError && (
              <>
                Сервис цен не подключён в текущей среде. Backend поддерживает
                POST /pricing, но переменная <code>PRICE_SERVICE_URL</code>{" "}
                не задана. Это нормально для demo — основная функциональность
                (план + список покупок) от него не зависит.
              </>
            )}
            {m.error instanceof PricingUpstreamError && (
              <>
                {m.error.message} Возможные причины: price-service не запущен,
                Edadil недоступен, нет интернета.
              </>
            )}
            {!(m.error instanceof PricingDisabledError) &&
              !(m.error instanceof PricingUpstreamError) && (
                <>Ошибка: {(m.error as Error).message}</>
              )}
          </div>
          <ShoppingFallback items={plan.shopping_list} />
        </>
      )}

      {m.data && <ResultBlock data={m.data} list={plan.shopping_list.length} />}
    </>
  );
}

function ResultBlock({
  data,
  list,
}: {
  data: PricingResponse;
  list: number;
}) {
  const min = data.min_total_price.toFixed(2);
  const max = data.max_total_price.toFixed(2);
  const oneNumber = Math.abs(data.max_total_price - data.min_total_price) < 0.01;

  return (
    <>
      <div className="card" style={{ background: "#eef7ee", borderColor: "#a5d6a7" }}>
        <h3 style={{ margin: "0 0 8px" }}>
          {oneNumber ? (
            <>
              {min} {data.currency}
            </>
          ) : (
            <>
              от {min} до {max} {data.currency}
            </>
          )}
        </h3>
        <div style={{ fontSize: 13, color: "#444" }}>
          Учтено: {data.priced_items_count} из {list}.{" "}
          {data.pricing_scope && (
            <>
              Источник цен: {SCOPE_LABEL[data.pricing_scope] ?? data.pricing_scope}.
            </>
          )}
        </div>
      </div>

      {data.status === "partial" && data.unpriced_items?.length ? (
        <div className="card">
          <h4 style={{ margin: "0 0 8px" }}>
            Не удалось оценить ({data.unpriced_items_count})
          </h4>
          {data.unpriced_items.map((it, i) => (
            <div key={i} style={{ fontSize: 13, marginBottom: 4 }}>
              <b>{it.ingredient_name}</b> — {it.requested_amount}{" "}
              {UNIT_LABEL[it.requested_unit as keyof typeof UNIT_LABEL] ?? it.requested_unit}{" "}
              <small style={{ color: "#888" }}>({it.reason})</small>
            </div>
          ))}
        </div>
      ) : null}
    </>
  );
}
