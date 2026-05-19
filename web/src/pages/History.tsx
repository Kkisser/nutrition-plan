import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  type PlanHistoryEntry,
  loadPlanHistory,
  saveLastPlan,
} from "../api/persist";

function formatSaved(iso: string): string {
  try {
    const d = new Date(iso);
    return d.toLocaleString("ru-RU", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return iso;
  }
}

export default function History() {
  const nav = useNavigate();
  const [entries, setEntries] = useState<PlanHistoryEntry[] | null>(null);

  useEffect(() => {
    loadPlanHistory().then(setEntries);
  }, []);

  const openWeek = async (e: PlanHistoryEntry) => {
    await saveLastPlan(e.plan);
    nav("/plan");
  };

  if (entries === null) return <p>Загружаю историю…</p>;

  if (entries.length === 0) {
    return (
      <>
        <h2>История планов</h2>
        <p>Сохранённых недель пока нет. Сгенерируйте план — он попадёт сюда.</p>
      </>
    );
  }

  return (
    <>
      <h2>История планов</h2>
      <p style={{ fontSize: 13, color: "#666" }}>
        Сохранены последние сгенерированные планы по неделям. Открытие
        делает выбранный план «текущим» на экране Плана.
      </p>
      {entries.map((e) => {
        const totalKcal =
          e.plan.plan.reduce((s, d) => s + d.day_totals.kcal, 0) /
          Math.max(1, e.plan.plan.length);
        return (
          <div className="card" key={e.plan.week_ref}>
            <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
              <div style={{ flex: 1 }}>
                <b>Неделя {e.plan.week_ref}</b>{" "}
                <small style={{ color: "#666" }}>
                  · сохранено {formatSaved(e.savedAt)}
                </small>
                <div style={{ fontSize: 13, color: "#444", marginTop: 4 }}>
                  Дней: {e.plan.plan.length} · средняя калорийность:{" "}
                  {totalKcal.toFixed(0)} ккал/день ·{" "}
                  {e.plan.compliance.in_corridor
                    ? "в коридоре"
                    : "вне коридора"}
                </div>
              </div>
              <button onClick={() => openWeek(e)}>Открыть</button>
            </div>
          </div>
        );
      })}
    </>
  );
}
