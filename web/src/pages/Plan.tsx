import { useCallback, useEffect, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { useMutation } from "@tanstack/react-query";
import { postPlan } from "../api/client";
import type { CatalogItem } from "../api/catalog";
import { loadAuth } from "../api/auth";
import { Spinner, useToast } from "../components/Toast";
import { printPlan } from "../lib/printPlan";
import {
  loadExcludedDishes,
  loadLastPlan,
  loadManualTargets,
  loadPinnedDishes,
  loadProfile,
  saveExcludedDishes,
  saveLastPlan,
  savePinnedDishes,
} from "../api/persist";
import type { Day, Meal, MealSlot, PinnedDish, PlanResponse } from "../api/types";
import { MEAL_LABEL } from "../lib/strings";

function mealLabel(m: string): string {
  return MEAL_LABEL[m as Meal] ?? m;
}

interface ReplaceReturnState {
  replace?: { day: number; meal: Meal; newDish: CatalogItem };
}

export default function Plan() {
  const nav = useNavigate();
  const loc = useLocation();
  const toast = useToast();
  const [plan, setPlan] = useState<PlanResponse | null>(null);
  const [offline, setOffline] = useState(false);
  const [hasProfile, setHasProfile] = useState<boolean | null>(null);
  const [pinned, setPinned] = useState<PinnedDish[]>([]);
  const [excluded, setExcluded] = useState<string[]>([]);

  // Применяем замену слота ПЕРЕД асинхронной подгрузкой из IndexedDB,
  // и при подгрузке не перетираем уже применённую замену. Без этого был
  // race condition: иногда loadLastPlan завершался ПОСЛЕ setPlan(replace)
  // и откатывал замену (отсюда «через раз» в отчёте пользователя).
  const replacePayload = (loc.state as ReplaceReturnState | null)?.replace;

  useEffect(() => {
    (async () => {
      const [stored, prof, pins, excl] = await Promise.all([
        loadLastPlan(),
        loadProfile(),
        loadPinnedDishes(),
        loadExcludedDishes(),
      ]);
      setHasProfile(!!prof);
      setPinned(pins);
      setExcluded(excl);
      // Загруженный план применяем только если в state нет pending-замены —
      // иначе ниже сработает replace-effect и сам положит правильный план.
      if (stored && !replacePayload) setPlan(stored);
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Локальная замена слота после возврата из Replace.
  // По МАТМОДЕЛЬ.txt §3.2: при замене порция = OptimalAlpha(target_kcal / new_kcal),
  // здесь мы делаем упрощённый подход — оставляем portions=1.0 (без масштабирования
  // под цель), потому что прогноз цели приёма пищи на клиенте недоступен.
  // Полное масштабирование вернётся, если пользователь нажмёт «Перегенерировать».
  useEffect(() => {
    if (!replacePayload) return;
    const { day, meal, newDish } = replacePayload;
    // Если плана в памяти ещё нет (initial-load не успел) — берём его из idb
    // синхронно через promise, иначе setPlan(prev=>null) ничего не сделает.
    (async () => {
      const base = plan ?? (await loadLastPlan());
      if (!base) {
        nav(loc.pathname, { replace: true, state: null });
        return;
      }
      const updated: PlanResponse = {
        ...base,
        plan: base.plan.map((d) => {
          if (d.day !== day) return d;
          const newMeals = d.meals.map((mm) =>
            mm.meal === meal
              ? {
                  ...mm,
                  dish_id: newDish.dish_id,
                  dish_title: newDish.name,
                  portions: 1.0,
                  pinned: false,
                  kcal: newDish.kcal,
                  protein_g: newDish.protein_g,
                  fat_g: newDish.fat_g,
                  carb_g: newDish.carb_g,
                }
              : mm,
          );
          const totals = newMeals.reduce(
            (acc, mm) => ({
              kcal: acc.kcal + mm.kcal,
              protein_g: acc.protein_g + mm.protein_g,
              fat_g: acc.fat_g + mm.fat_g,
              carb_g: acc.carb_g + mm.carb_g,
            }),
            { kcal: 0, protein_g: 0, fat_g: 0, carb_g: 0 },
          );
          return { ...d, meals: newMeals, day_totals: totals };
        }),
      };
      await saveLastPlan(updated);
      setPlan(updated);
      nav(loc.pathname, { replace: true, state: null });
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [replacePayload]);

  const generate = useCallback(
    async (overridePinned?: PinnedDish[], overrideExcluded?: string[]) => {
      const profile = await loadProfile();
      if (!profile) throw new Error("Профиль не заполнен — заполните анкету.");
      const auth = await loadAuth();
      if (!auth) throw new Error("Не выполнен вход.");
      const pins = overridePinned ?? pinned;
      const excl = overrideExcluded ?? excluded;
      const manual = await loadManualTargets();
      return postPlan({
        user_id: auth.userId,
        profile,
        pinned_dishes: pins,
        excluded_dishes: excl,
        manual_targets_override: manual?.enabled ? manual : undefined,
      });
    },
    [pinned, excluded],
  );

  const m = useMutation({
    mutationFn: async (args: {
      pins?: PinnedDish[];
      excl?: string[];
    }): Promise<PlanResponse> => {
      const resp = await generate(args.pins, args.excl);
      await saveLastPlan(resp);
      setOffline(false);
      return resp;
    },
    onSuccess: (resp) => {
      setPlan(resp);
      toast.success(`План сформирован · неделя ${resp.week_ref}`);
    },
    onError: async (err: Error) => {
      const cached = await loadLastPlan();
      if (cached) {
        setPlan(cached);
        setOffline(true);
      }
      toast.error(err.message || "Не удалось сформировать план");
      console.error(err);
    },
  });

  const togglePin = useCallback(
    async (slot: MealSlot, day: number) => {
      const key = `${day}-${slot.meal}-${slot.dish_id}`;
      const isPinned = pinned.some(
        (p) => `${p.day}-${p.meal}-${p.dish_id}` === key,
      );
      const next = isPinned
        ? pinned.filter((p) => `${p.day}-${p.meal}-${p.dish_id}` !== key)
        : [...pinned, { day, meal: slot.meal, dish_id: slot.dish_id }];
      setPinned(next);
      await savePinnedDishes(next);
      setPlan((prev) => {
        if (!prev) return prev;
        return {
          ...prev,
          plan: prev.plan.map((d) =>
            d.day === day
              ? {
                  ...d,
                  meals: d.meals.map((mm) =>
                    mm.meal === slot.meal && mm.dish_id === slot.dish_id
                      ? { ...mm, pinned: !isPinned }
                      : mm,
                  ),
                }
              : d,
          ),
        };
      });
    },
    [pinned],
  );

  const exclude = useCallback(
    async (slot: MealSlot) => {
      if (!confirm(`Не показывать «${slot.dish_title}» больше?`)) return;
      const next = excluded.includes(slot.dish_id)
        ? excluded
        : [...excluded, slot.dish_id];
      setExcluded(next);
      await saveExcludedDishes(next);
      m.mutate({ excl: next });
    },
    [excluded, m],
  );

  const replace = useCallback(
    async (slot: MealSlot, day: number) => {
      // Если блюдо закреплено — предложим открепить и продолжить.
      // Раньше кнопка была disabled, и пользователь не понимал почему.
      if (slot.pinned) {
        const ok = confirm(
          `Блюдо «${slot.dish_title}» закреплено. Открепить и заменить?`,
        );
        if (!ok) return;
        const key = `${day}-${slot.meal}-${slot.dish_id}`;
        const next = pinned.filter(
          (p) => `${p.day}-${p.meal}-${p.dish_id}` !== key,
        );
        setPinned(next);
        await savePinnedDishes(next);
      }
      nav("/replace", {
        state: {
          day,
          meal: slot.meal,
          currentDishId: slot.dish_id,
          currentDishTitle: slot.dish_title,
        },
      });
    },
    [nav, pinned],
  );

  if (hasProfile === false) {
    return (
      <>
        <h2>Профиль не заполнен</h2>
        <p>
          Заполните <Link to="/survey">анкету</Link>, чтобы сформировать план.
        </p>
      </>
    );
  }

  return (
    <>
      <div style={{ display: "flex", gap: 8, alignItems: "center", marginBottom: 12, flexWrap: "wrap" }}>
        <h2 style={{ flex: 1, margin: 0 }}>Недельный план</h2>
        {plan && (
          <button
            type="button"
            onClick={() => printPlan(plan)}
            style={{ background: "transparent", color: "#2e7d32", border: "1px solid #2e7d32" }}
          >
            Скачать PDF
          </button>
        )}
        <button onClick={() => m.mutate({})} disabled={m.isPending}>
          {m.isPending ? (
            <>
              <Spinner /> &nbsp; Формирую…
            </>
          ) : plan ? (
            "Перегенерировать"
          ) : (
            "Сформировать"
          )}
        </button>
      </div>

      {offline && (
        <div className="warn offline">
          Нет связи с сервером — показан последний сохранённый план.
        </div>
      )}
      {m.isError && (
        <div className="warn">
          <b>Ошибка генерации:</b> {(m.error as Error).message}
          {plan && (
            <div style={{ fontSize: 12, marginTop: 4, color: "#555" }}>
              Показан предыдущий план — он может уже не соответствовать новым настройкам.
            </div>
          )}
        </div>
      )}

      {!plan && !m.isPending && (
        <p>Нажмите «Сформировать» чтобы получить недельный план.</p>
      )}

      {plan && (
        <>
          <p>
            Неделя <b>{plan.week_ref}</b> ·{" "}
            {plan.compliance.in_corridor ? (
              <span style={{ color: "#2e7d32" }}>укладывается в коридор</span>
            ) : (
              <span style={{ color: "#a05a00" }} title="План сформирован, но дневные КБЖУ выходят за допустимый коридор. Это предупреждение, действия по плану остаются доступными.">
                ⚠ выход за коридор (информационно)
              </span>
            )}
          </p>
          {excluded.length > 0 && (
            <p style={{ fontSize: 13, color: "#666" }}>
              Скрыто блюд: {excluded.length}.{" "}
              <button
                style={{
                  fontSize: 12,
                  padding: "2px 8px",
                  background: "transparent",
                  color: "#2e7d32",
                  border: "1px solid #2e7d32",
                }}
                onClick={async () => {
                  setExcluded([]);
                  await saveExcludedDishes([]);
                  m.mutate({ excl: [] });
                }}
              >
                сбросить
              </button>
            </p>
          )}
          {plan.plan.map((d) => (
            <DayBlock
              key={d.day}
              day={d}
              onPin={togglePin}
              onExclude={exclude}
              onReplace={replace}
              busy={m.isPending}
            />
          ))}
        </>
      )}
    </>
  );
}

interface DayBlockProps {
  day: Day;
  onPin: (slot: MealSlot, day: number) => void;
  onExclude: (slot: MealSlot) => void;
  onReplace: (slot: MealSlot, day: number) => void;
  busy: boolean;
}

function DayBlock({ day, onPin, onExclude, onReplace, busy }: DayBlockProps) {
  return (
    <div className="card">
      <h3 style={{ margin: "0 0 8px" }}>День {day.day}</h3>
      {day.meals.map((m) => (
        <div
          key={`${m.meal}-${m.dish_id}`}
          style={{
            marginBottom: 6,
            display: "flex",
            alignItems: "center",
            gap: 8,
          }}
        >
          <div style={{ flex: 1 }}>
            <b>{mealLabel(m.meal)}</b>: {m.dish_title}
            {m.pinned && " 📌"}{" "}
            <small>
              ×{m.portions.toFixed(2)} · {m.kcal.toFixed(0)} ккал
            </small>
          </div>
          <button
            title={m.pinned ? "Открепить" : "Закрепить"}
            onClick={() => onPin(m, day.day)}
            disabled={busy}
            style={iconButtonStyle(m.pinned)}
          >
            📌
          </button>
          <button
            title={m.pinned ? "Открепить и заменить" : "Заменить"}
            onClick={() => onReplace(m, day.day)}
            disabled={busy}
            style={iconButtonStyle(false)}
          >
            🔄
          </button>
          <button
            title="Не показывать"
            onClick={() => onExclude(m)}
            disabled={busy}
            style={iconButtonStyle(false)}
          >
            🚫
          </button>
        </div>
      ))}
      <div className="totals">
        <span>Итого: {day.day_totals.kcal.toFixed(0)} ккал</span>
        <span>Б: {day.day_totals.protein_g.toFixed(0)} г</span>
        <span>Ж: {day.day_totals.fat_g.toFixed(0)} г</span>
        <span>У: {day.day_totals.carb_g.toFixed(0)} г</span>
      </div>
    </div>
  );
}

function iconButtonStyle(active: boolean): React.CSSProperties {
  return {
    padding: "3px 8px",
    fontSize: 14,
    background: active ? "#2e7d32" : "transparent",
    color: active ? "white" : "#2e7d32",
    border: "1px solid #2e7d32",
    borderRadius: 4,
    cursor: "pointer",
  };
}
