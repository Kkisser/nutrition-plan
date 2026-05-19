import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { get, set, del } from "idb-keyval";
import { loadLastPlan } from "../api/persist";
import type { PlanResponse, ShoppingItem } from "../api/types";
import { UNIT_LABEL, categoryRank } from "../lib/strings";
import { combineShopping } from "../lib/shopping";

const KEY_PARTNER = "partner_plan";

function downloadJSON(filename: string, payload: unknown) {
  const blob = new Blob([JSON.stringify(payload, null, 2)], {
    type: "application/json",
  });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

export default function Family() {
  const [myPlan, setMyPlan] = useState<PlanResponse | null>(null);
  const [partner, setPartner] = useState<PlanResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      const mine = await loadLastPlan();
      if (mine) setMyPlan(mine);
      const p = await get<PlanResponse>(KEY_PARTNER);
      if (p) setPartner(p);
    })();
  }, []);

  const exportMine = () => {
    if (!myPlan) return;
    downloadJSON(`plan-${myPlan.week_ref}.json`, myPlan);
  };

  const importPartner = async (file: File) => {
    setError(null);
    try {
      const text = await file.text();
      const parsed = JSON.parse(text) as PlanResponse;
      if (!parsed.shopping_list || !Array.isArray(parsed.shopping_list)) {
        throw new Error("Файл не содержит shopping_list — это не план питания.");
      }
      await set(KEY_PARTNER, parsed);
      setPartner(parsed);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Не удалось разобрать файл.");
    }
  };

  const clearPartner = async () => {
    await del(KEY_PARTNER);
    setPartner(null);
  };

  if (!myPlan) {
    return (
      <>
        <h2>Совместное меню</h2>
        <p>
          Нет собственного плана. Сначала сформируйте свой план на{" "}
          <Link to="/plan">экране Плана</Link>.
        </p>
      </>
    );
  }

  const combined = partner
    ? combineShopping(myPlan.shopping_list, partner.shopping_list)
    : myPlan.shopping_list;
  const groupedKeys = new Set<string>();
  const grouped: Record<string, ShoppingItem[]> = {};
  for (const it of combined) {
    const k = it.category || "Прочее";
    groupedKeys.add(k);
    (grouped[k] ??= []).push(it);
  }
  const cats = Array.from(groupedKeys).sort(
    (a, b) => categoryRank(a) - categoryRank(b) || a.localeCompare(b, "ru"),
  );

  return (
    <>
      <h2>Совместное меню</h2>
      <p style={{ fontSize: 13, color: "#666" }}>
        Согласно §15 функционала: каждый член семьи работает в своей
        учётной записи. Для совместной готовки экспортируйте свой план,
        отправьте его другому человеку, а его план импортируйте здесь —
        получите общий список покупок (одноимённые продукты суммируются).
      </p>

      <section style={{ marginTop: 16 }}>
        <h3>Мой план</h3>
        <p>
          Неделя <b>{myPlan.week_ref}</b> · {myPlan.shopping_list.length} позиций
        </p>
        <button onClick={exportMine}>Экспортировать мой план (JSON)</button>
      </section>

      <section style={{ marginTop: 16 }}>
        <h3>План партнёра</h3>
        {partner ? (
          <>
            <p>
              Импортирован план недели <b>{partner.week_ref}</b> ·{" "}
              {partner.shopping_list.length} позиций
            </p>
            <button onClick={clearPartner} style={{ background: "#aa3a3a" }}>
              Убрать план партнёра
            </button>
          </>
        ) : (
          <>
            <p>План партнёра не загружен.</p>
            <input
              type="file"
              accept="application/json,.json"
              onChange={(e) => {
                const f = e.target.files?.[0];
                if (f) importPartner(f);
              }}
            />
          </>
        )}
        {error && <p style={{ color: "#a05a00", marginTop: 8 }}>{error}</p>}
      </section>

      <section style={{ marginTop: 24 }}>
        <h3>
          {partner ? "Объединённый список покупок" : "Список покупок (только мой)"}
        </h3>
        <p>{combined.length} позиций</p>
        {cats.map((cat) => (
          <div key={cat} style={{ marginBottom: 12 }}>
            <h4 style={{ margin: "6px 0" }}>{cat}</h4>
            {grouped[cat].map((it) => (
              <div className="card" key={`${it.ingredient_name}-${it.unit}`} style={{ padding: "6px 10px" }}>
                <b>{it.ingredient_name}</b> —{" "}
                {it.amount.toFixed(it.unit === "pcs" ? 0 : 1)}{" "}
                {UNIT_LABEL[it.unit] ?? it.unit}
              </div>
            ))}
          </div>
        ))}
      </section>
    </>
  );
}
