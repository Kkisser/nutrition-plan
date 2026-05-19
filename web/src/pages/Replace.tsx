import { useEffect, useMemo, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import type { CatalogItem } from "../api/catalog";
import { getCatalog } from "../api/catalog";
import { loadExcludedDishes, saveExcludedDishes } from "../api/persist";
import type { Meal } from "../api/types";
import { Spinner, useToast } from "../components/Toast";
import { MEAL_LABEL } from "../lib/strings";

interface ReplaceState {
  day: number;
  meal: Meal;
  currentDishId: string;
  currentDishTitle: string;
}

export default function Replace() {
  const nav = useNavigate();
  const loc = useLocation();
  const toast = useToast();
  const state = loc.state as ReplaceState | null;

  const [items, setItems] = useState<CatalogItem[]>([]);
  const [filter, setFilter] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [excludedCount, setExcludedCount] = useState(0);

  const reload = () => {
    if (!state) return;
    setLoading(true);
    setError(null);
    Promise.all([
      getCatalog(state.meal),
      loadExcludedDishes(),
    ])
      .then(([xs, excl]) => {
        setItems(xs.filter((x) => x.dish_id !== state.currentDishId));
        setExcludedCount(excl.length);
      })
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    reload();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [state]);

  const onResetExcluded = async () => {
    await saveExcludedDishes([]);
    reload();
  };

  const filtered = useMemo(() => {
    const q = filter.trim().toLowerCase();
    if (!q) return items;
    return items.filter((x) => x.name.toLowerCase().includes(q));
  }, [items, filter]);

  if (!state) {
    return (
      <>
        <h2>Замена блюда</h2>
        <p>
          Перейдите с экрана плана: нажмите 🔄 рядом с блюдом, которое хотите заменить.
        </p>
      </>
    );
  }

  const onPick = (item: CatalogItem) => {
    toast.success(`«${state.currentDishTitle}» заменено на «${item.name}»`);
    nav("/plan", {
      state: {
        replace: {
          day: state.day,
          meal: state.meal,
          newDish: item,
        },
      },
    });
  };

  return (
    <>
      <h2>
        Замена · день {state.day} · {MEAL_LABEL[state.meal]}
      </h2>
      <p style={{ color: "#666", fontSize: 13 }}>
        Текущее блюдо: <b>{state.currentDishTitle}</b>. Выберите замену из
        отфильтрованного по вашему профилю каталога:
      </p>

      <div className="field">
        <input
          type="text"
          placeholder="Поиск по названию…"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
        />
      </div>

      {loading && (
        <p>
          <Spinner /> &nbsp; Загрузка каталога…
        </p>
      )}
      {error && <div className="warn">Ошибка: {error}</div>}
      {!loading && !error && filtered.length === 0 && (
        <div className="warn">
          <b>Подходящих блюд нет.</b>
          <div style={{ fontSize: 13, marginTop: 6 }}>
            В каталоге под текущую диету, аллергены и фильтры — нет других
            рецептов на этот приём пищи.
          </div>
          <div style={{ display: "flex", gap: 8, marginTop: 10, flexWrap: "wrap" }}>
            {excludedCount > 0 && (
              <button onClick={onResetExcluded}>
                Сбросить «не показывать» ({excludedCount})
              </button>
            )}
            <Link to="/survey">
              <button style={{ background: "transparent", color: "#2e7d32", border: "1px solid #2e7d32" }}>
                Изменить анкету
              </button>
            </Link>
          </div>
        </div>
      )}

      {filtered.map((item) => (
        <div
          key={item.dish_id}
          className="card"
          style={{ display: "flex", alignItems: "center", gap: 12, cursor: "pointer" }}
          onClick={() => onPick(item)}
        >
          <div style={{ flex: 1 }}>
            <b>{item.name}</b>
            <div className="totals">
              <span>{item.kcal.toFixed(0)} ккал</span>
              <span>Б {item.protein_g.toFixed(0)} г</span>
              <span>Ж {item.fat_g.toFixed(0)} г</span>
              <span>У {item.carb_g.toFixed(0)} г</span>
              {item.cook_time_min > 0 && <span>⏱ {item.cook_time_min} мин</span>}
            </div>
          </div>
          <button onClick={(e) => { e.stopPropagation(); onPick(item); }}>
            Выбрать
          </button>
        </div>
      ))}

      <button
        style={{ background: "transparent", color: "#666", border: "1px solid #c7c7c2" }}
        onClick={() => nav("/plan")}
      >
        Отмена
      </button>
    </>
  );
}
