import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import type { Allergen, Diet, Goal, ManualTargetsOverride, Meal, Profile, Sex } from "../api/types";
import {
  loadKfaSurvey,
  loadManualTargets,
  loadProfile,
  saveKfaSurvey,
  saveManualTargets,
  saveProfile,
} from "../api/persist";
import {
  Q1_OPTIONS,
  Q3_OPTIONS,
  Q4_OPTIONS,
  deriveKfa,
  type Q1,
  type Q3,
  type Q4,
  type Survey as KfaSurvey,
} from "../lib/kfa";

const ALLERGENS: Allergen[] = [
  "milk", "eggs", "fish", "gluten", "peanut",
  "sesame", "shellfish", "soy", "nuts",
];
const MEALS: Meal[] = ["breakfast", "lunch", "dinner", "snack"];

const DEFAULT_PROFILE: Profile = {
  sex: "male",
  age: 30,
  height_cm: 180,
  weight_kg: 75,
  kfa_group: "I",
  goal: "maintain",
  diet_type: "classic",
  allergens: [],
  excluded_products: [],
  meals: ["breakfast", "lunch", "dinner", "snack"],
  persons: 1,
};

const DEFAULT_SURVEY: KfaSurvey = { q1: "sedentary", q3: "none", q4: "" };

const DEFAULT_MANUAL: ManualTargetsOverride = { enabled: false };

import { MEAL_LABEL_LOWER as MEAL_LABEL, KFA_LABEL } from "../lib/strings";

export default function Survey() {
  const nav = useNavigate();
  const [p, setP] = useState<Profile>(DEFAULT_PROFILE);
  const [survey, setSurvey] = useState<KfaSurvey>(DEFAULT_SURVEY);
  const [manual, setManual] = useState<ManualTargetsOverride>(DEFAULT_MANUAL);

  useEffect(() => {
    Promise.all([loadProfile(), loadKfaSurvey(), loadManualTargets()]).then(
      ([prof, surv, man]) => {
        if (prof) setP(prof);
        if (surv) setSurvey(surv);
        if (man) setManual(man);
      },
    );
  }, []);

  const derivedKfa = deriveKfa(survey);

  const onSave = async () => {
    const final: Profile = { ...p, kfa_group: derivedKfa };
    await saveProfile(final);
    await saveKfaSurvey(survey);
    await saveManualTargets(manual);
    nav("/plan");
  };

  const setManualNumber = (field: keyof ManualTargetsOverride, raw: string) => {
    const num = raw === "" ? undefined : parseFloat(raw);
    setManual((m) => ({
      ...m,
      [field]: num !== undefined && Number.isFinite(num) ? num : undefined,
    }));
  };

  const setQ1 = (q1: Q1) => setSurvey((s) => ({ ...s, q1 }));
  const setQ3 = (q3: Q3) =>
    setSurvey((s) => ({ ...s, q3, q4: q3 === "none" ? "" : s.q4 || "moderate" }));
  const setQ4 = (q4: Q4) => setSurvey((s) => ({ ...s, q4 }));

  const toggleAllergen = (a: Allergen) =>
    setP((s) => ({
      ...s,
      allergens: s.allergens.includes(a)
        ? s.allergens.filter((x) => x !== a)
        : [...s.allergens, a],
    }));

  const toggleMeal = (m: Meal) =>
    setP((s) => ({
      ...s,
      meals: s.meals.includes(m)
        ? s.meals.filter((x) => x !== m)
        : [...s.meals, m],
    }));

  return (
    <>
      <h2>Анкета профиля</h2>

      <section>
        <h3>Физиологические данные</h3>

        <div className="field">
          <label>Пол</label>
          <select value={p.sex} onChange={(e) => setP({ ...p, sex: e.target.value as Sex })}>
            <option value="male">мужской</option>
            <option value="female">женский</option>
          </select>
        </div>

        <div className="field">
          <label>Возраст (лет)</label>
          <input
            type="number"
            value={p.age}
            onChange={(e) => setP({ ...p, age: parseInt(e.target.value || "0", 10) })}
          />
        </div>

        <div className="field">
          <label>Рост (см)</label>
          <input
            type="number"
            value={p.height_cm}
            onChange={(e) => setP({ ...p, height_cm: parseInt(e.target.value || "0", 10) })}
          />
        </div>

        <div className="field">
          <label>Масса тела (кг)</label>
          <input
            type="number"
            value={p.weight_kg}
            onChange={(e) => setP({ ...p, weight_kg: parseFloat(e.target.value || "0") })}
          />
        </div>
      </section>

      <section>
        <h3>Мини-анкета активности</h3>
        <p style={{ fontSize: 13, color: "#666", margin: "4px 0 12px" }}>
          По ответам определяется группа КФА (МР 2.3.1.0253-21).
          Прямой выбор не используется — снижает влияние самооценки.
        </p>

        <div className="field">
          <label>
            Вопрос 1. Какой вариант лучше всего описывает вашу обычную дневную активность без учёта отдельных тренировок?
          </label>
          {Q1_OPTIONS.map((o) => (
            <label key={o.value} style={{ display: "block", margin: "4px 0" }}>
              <input
                type="radio"
                name="q1"
                checked={survey.q1 === o.value}
                onChange={() => setQ1(o.value)}
              />{" "}
              {o.label}
              {o.hint && (
                <div style={{ fontSize: 12, color: "#777", marginLeft: 22 }}>{o.hint}</div>
              )}
            </label>
          ))}
        </div>

        <div className="field">
          <label>
            Вопрос 3. Сколько раз в неделю у вас есть отдельная физическая нагрузка длительностью не менее 30 минут?
          </label>
          {Q3_OPTIONS.map((o) => (
            <label key={o.value} style={{ display: "block", margin: "4px 0" }}>
              <input
                type="radio"
                name="q3"
                checked={survey.q3 === o.value}
                onChange={() => setQ3(o.value)}
              />{" "}
              {o.label}
            </label>
          ))}
        </div>

        {survey.q3 !== "none" && (
          <div className="field">
            <label>Вопрос 4. Как обычно проходит эта физическая нагрузка?</label>
            {Q4_OPTIONS.map((o) => (
              <label key={o.value} style={{ display: "block", margin: "4px 0" }}>
                <input
                  type="radio"
                  name="q4"
                  checked={survey.q4 === o.value}
                  onChange={() => setQ4(o.value)}
                />{" "}
                {o.label}
                {o.hint && (
                  <div style={{ fontSize: 12, color: "#777", marginLeft: 22 }}>{o.hint}</div>
                )}
              </label>
            ))}
          </div>
        )}

        <div className="card" style={{ background: "#eef7ee", borderColor: "#a5d6a7" }}>
          Группа КФА по ответам: <b>{KFA_LABEL[derivedKfa]}</b>
        </div>
      </section>

      <section>
        <h3>Цель и тип питания</h3>

        <div className="field">
          <label>Цель</label>
          <select value={p.goal} onChange={(e) => setP({ ...p, goal: e.target.value as Goal })}>
            <option value="deficit">снижение массы (-15%)</option>
            <option value="maintain">поддержание</option>
            <option value="surplus">набор массы (+15%)</option>
          </select>
        </div>

        <div className="field">
          <label>Тип питания</label>
          <select value={p.diet_type} onChange={(e) => setP({ ...p, diet_type: e.target.value as Diet })}>
            <option value="classic">классическое</option>
            <option value="keto">кето</option>
            <option value="vegetarian">вегетарианство</option>
            <option value="vegan">веганство</option>
            <option value="paleo">палео</option>
            <option value="fasting">пост</option>
          </select>
        </div>
      </section>

      <section>
        <h3>Структура приёмов пищи и ограничения</h3>

        <div className="field">
          <label>Активные приёмы пищи</label>
          <div>
            {MEALS.map((m) => (
              <label key={m} style={{ marginRight: 12 }}>
                <input
                  type="checkbox"
                  checked={p.meals.includes(m)}
                  onChange={() => toggleMeal(m)}
                />{" "}
                {MEAL_LABEL[m]}
              </label>
            ))}
          </div>
        </div>

        <div className="field">
          <label>Аллергены</label>
          <div>
            {ALLERGENS.map((a) => (
              <label key={a} style={{ marginRight: 12 }}>
                <input
                  type="checkbox"
                  checked={p.allergens.includes(a)}
                  onChange={() => toggleAllergen(a)}
                />{" "}
                {a}
              </label>
            ))}
          </div>
        </div>

        <div className="field">
          <label>Число персон для приготовления</label>
          <input
            type="number"
            value={p.persons}
            onChange={(e) =>
              setP({ ...p, persons: Math.max(1, parseInt(e.target.value || "1", 10)) })
            }
          />
        </div>
      </section>

      <section>
        <h3>Ручные целевые КБЖУ (опционально)</h3>
        <p style={{ fontSize: 13, color: "#666", margin: "4px 0 12px" }}>
          По умолчанию КБЖУ рассчитываются автоматически из норм
          МР 2.3.1.0253-21 (по полу, возрасту, КФА и цели). При желании
          можно скорректировать любое значение вручную — допустимый
          диапазон ограничен ±15 % от нормы, выходы за коридор
          зажимаются.
        </p>
        <label style={{ display: "block", marginBottom: 8 }}>
          <input
            type="checkbox"
            checked={manual.enabled}
            onChange={(e) => setManual({ ...manual, enabled: e.target.checked })}
          />{" "}
          Использовать ручные значения
        </label>
        {manual.enabled && (
          <>
            <div className="field">
              <label>Калорийность (ккал/день)</label>
              <input
                type="number"
                placeholder="—"
                value={manual.kcal ?? ""}
                onChange={(e) => setManualNumber("kcal", e.target.value)}
              />
            </div>
            <div className="field">
              <label>Белки (г/день)</label>
              <input
                type="number"
                placeholder="—"
                value={manual.protein_g ?? ""}
                onChange={(e) => setManualNumber("protein_g", e.target.value)}
              />
            </div>
            <div className="field">
              <label>Жиры (г/день)</label>
              <input
                type="number"
                placeholder="—"
                value={manual.fat_g ?? ""}
                onChange={(e) => setManualNumber("fat_g", e.target.value)}
              />
            </div>
            <div className="field">
              <label>Углеводы (г/день)</label>
              <input
                type="number"
                placeholder="—"
                value={manual.carb_g ?? ""}
                onChange={(e) => setManualNumber("carb_g", e.target.value)}
              />
            </div>
            <p style={{ fontSize: 12, color: "#777" }}>
              Пустое поле означает «оставить значение из нормы». Сохранённые
              значения, выходящие за ±15 %, будут автоматически прижаты к
              границе диапазона на стороне сервера.
            </p>
          </>
        )}
      </section>

      <button onClick={onSave}>Сохранить и сформировать план</button>
    </>
  );
}
