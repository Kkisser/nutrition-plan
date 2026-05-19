import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { loadLastPlan } from "../api/persist";
import type { PlanResponse, ShoppingItem } from "../api/types";
import { UNIT_LABEL, categoryRank } from "../lib/strings";

function groupByCategory(items: ShoppingItem[]): Record<string, ShoppingItem[]> {
  const out: Record<string, ShoppingItem[]> = {};
  for (const it of items) {
    const key = it.category || "Прочее";
    (out[key] ??= []).push(it);
  }
  for (const k of Object.keys(out)) {
    out[k].sort((a, b) => a.ingredient_name.localeCompare(b.ingredient_name, "ru"));
  }
  return out;
}

export default function Shopping() {
  const [plan, setPlan] = useState<PlanResponse | null>(null);

  useEffect(() => {
    loadLastPlan().then((p) => p && setPlan(p));
  }, []);

  if (!plan) {
    return (
      <>
        <h2>Список покупок</h2>
        <p>
          План ещё не сформирован. Перейдите на <Link to="/plan">экран плана</Link> и нажмите «Сформировать».
        </p>
      </>
    );
  }

  const grouped = groupByCategory(plan.shopping_list);
  const sortedCats = Object.keys(grouped).sort(
    (a, b) => categoryRank(a) - categoryRank(b) || a.localeCompare(b, "ru"),
  );

  return (
    <>
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 12 }}>
        <h2 style={{ flex: 1, margin: 0 }}>
          Список покупок · {plan.week_ref}
        </h2>
        <Link to="/pricing">
          <button type="button">Оценить стоимость</button>
        </Link>
      </div>
      <p>{plan.shopping_list.length} позиций в {sortedCats.length} категориях</p>
      {sortedCats.map((cat) => (
        <section key={cat} style={{ marginBottom: 16 }}>
          <h3 style={{ margin: "8px 0 6px" }}>{cat}</h3>
          {grouped[cat].map((it) => (
            <div className="card" key={`${it.ingredient_name}-${it.unit}`} style={{ padding: "6px 10px" }}>
              <b>{it.ingredient_name}</b>{" "}
              — {it.amount.toFixed(it.unit === "pcs" ? 0 : 1)} {UNIT_LABEL[it.unit] ?? it.unit}
            </div>
          ))}
        </section>
      ))}
    </>
  );
}
