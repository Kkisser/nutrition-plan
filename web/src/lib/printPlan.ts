import type { PlanResponse } from "../api/types";
import { MEAL_LABEL } from "./strings";

// Открывает новое окно, печатает план в виде A4-friendly HTML и вызывает
// window.print() — пользователь сохраняет PDF через системный диалог
// «Печать → Сохранить как PDF».
//
// Подход без зависимостей (jsPDF/react-pdf весят сотни КБ и плохо ладят с
// кириллицей без подключения шрифтов). Нативный print даёт качественный
// результат и работает офлайн.
export function printPlan(plan: PlanResponse): void {
  const w = window.open("", "_blank", "width=900,height=1200");
  if (!w) {
    alert("Браузер заблокировал всплывающее окно. Разрешите всплывающие окна для сайта.");
    return;
  }

  const html = renderPlanHTML(plan);
  w.document.open();
  w.document.write(html);
  w.document.close();
  // Дать браузеру времени отрисоваться перед печатью
  w.onload = () => {
    w.focus();
    w.print();
  };
}

function escape(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function renderPlanHTML(plan: PlanResponse): string {
  const days = plan.plan
    .map((d) => {
      const rows = d.meals
        .map(
          (m) => `
        <tr>
          <td class="meal">${escape(MEAL_LABEL[m.meal])}</td>
          <td class="dish">${escape(m.dish_title)}${m.pinned ? " 📌" : ""}</td>
          <td class="num">${m.portions.toFixed(2)}</td>
          <td class="num">${m.kcal.toFixed(0)}</td>
          <td class="num">${m.protein_g.toFixed(0)}</td>
          <td class="num">${m.fat_g.toFixed(0)}</td>
          <td class="num">${m.carb_g.toFixed(0)}</td>
        </tr>`,
        )
        .join("");
      const t = d.day_totals;
      return `
      <section class="day">
        <h2>День ${d.day}</h2>
        <table>
          <thead>
            <tr>
              <th>Приём</th><th>Блюдо</th>
              <th class="num">×</th>
              <th class="num">ккал</th><th class="num">Б</th><th class="num">Ж</th><th class="num">У</th>
            </tr>
          </thead>
          <tbody>${rows}</tbody>
          <tfoot>
            <tr>
              <td colspan="3" class="totals-label">Итого за день:</td>
              <td class="num"><b>${t.kcal.toFixed(0)}</b></td>
              <td class="num"><b>${t.protein_g.toFixed(0)}</b></td>
              <td class="num"><b>${t.fat_g.toFixed(0)}</b></td>
              <td class="num"><b>${t.carb_g.toFixed(0)}</b></td>
            </tr>
          </tfoot>
        </table>
      </section>`;
    })
    .join("");

  const shopping = plan.shopping_list
    .map(
      (it) =>
        `<li><b>${escape(it.ingredient_name)}</b>${it.category ? ` <span class="cat">(${escape(it.category)})</span>` : ""} — ${it.amount.toFixed(it.unit === "pcs" ? 0 : 1)} ${escape(it.unit)}</li>`,
    )
    .join("");

  return `<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8" />
<title>Недельный план питания · ${escape(plan.week_ref)}</title>
<style>
  @page { size: A4; margin: 14mm; }
  body { font-family: -apple-system, "Segoe UI", "Helvetica Neue", Arial, sans-serif; color: #222; font-size: 12pt; }
  h1 { font-size: 18pt; margin: 0 0 4pt; }
  h2 { font-size: 14pt; margin: 14pt 0 6pt; }
  .meta { color: #555; font-size: 10pt; margin-bottom: 14pt; }
  table { width: 100%; border-collapse: collapse; margin: 0; font-size: 10pt; }
  th, td { padding: 4pt 6pt; border-bottom: 1px solid #ddd; text-align: left; }
  th { background: #f3f3f0; font-weight: 600; }
  .num { text-align: right; white-space: nowrap; }
  .meal { font-weight: 600; width: 18%; }
  .dish { width: 52%; }
  .totals-label { text-align: right; color: #555; }
  tfoot td { border-top: 1px solid #999; border-bottom: none; }
  .day { page-break-inside: avoid; }
  ul.shopping { columns: 2; column-gap: 18pt; padding-left: 18pt; font-size: 10pt; }
  ul.shopping li { break-inside: avoid; margin-bottom: 2pt; }
  .cat { color: #777; font-size: 9pt; }
  @media print {
    button { display: none; }
  }
</style>
</head>
<body>
  <h1>Недельный план питания</h1>
  <div class="meta">
    Неделя ${escape(plan.week_ref)} ·
    ${plan.compliance.in_corridor ? "укладывается в коридор" : "вне коридора (информационно)"} ·
    позиций в списке покупок: ${plan.shopping_list.length}
  </div>
  ${days}
  <h2>Список покупок</h2>
  <ul class="shopping">${shopping}</ul>
</body>
</html>`;
}
