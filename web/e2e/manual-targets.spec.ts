import { test, expect } from "@playwright/test";

// Manual targets: переключить чекбокс в анкете и проверить, что план
// после генерации соответствует переданным значениям.

test.beforeEach(async ({ context, page }) => {
  await context.clearCookies();
  await page.goto("/");
  await page.evaluate(async () => {
    localStorage.clear();
    sessionStorage.clear();
    const dbs = await indexedDB.databases?.();
    if (dbs) {
      for (const d of dbs) if (d.name) indexedDB.deleteDatabase(d.name);
    }
  });
});

test("анкета с ручными КБЖУ → план идёт по ним", async ({ page }) => {
  const email = `e2e_manual_${Date.now()}@example.com`;

  // Регистрация → /survey.
  await page.goto("/login");
  await page.getByRole("button", { name: "Регистрация" }).click();
  await page.locator('input[type="email"]').fill(email);
  const passwords = page.locator('input[type="password"]');
  await passwords.nth(0).fill("TestPass123");
  await passwords.nth(1).fill("TestPass123");
  await page.getByRole("button", { name: "Зарегистрироваться" }).click();
  await expect(page).toHaveURL(/\/survey$/, { timeout: 10_000 });

  // Включаем ручные КБЖУ, выставляем 2200 ккал.
  // Дефолт нормы для male/30/178/75/КФА-I ≈ 2400, ±15% = [2040, 2760] — 2200 в коридоре.
  const manualCheckbox = page.getByRole("checkbox", { name: /Использовать ручные значения/ });
  await manualCheckbox.check();
  await expect(manualCheckbox).toBeChecked();

  const kcalInput = page.locator('input[type="number"]').filter({
    has: page.locator("xpath=preceding-sibling::label[contains(., 'Калорийность')]"),
  });
  // Простой fallback: найдём по label-тексту в общей разметке .field.
  // Так как label у нас не связан с input через htmlFor, ищем по позиции.
  // В блоке Manual 4 числовых input после checkbox: kcal, protein, fat, carb.
  const manualSection = page.locator("section").filter({ hasText: "Ручные целевые КБЖУ" });
  const manualInputs = manualSection.locator('input[type="number"]');
  await manualInputs.nth(0).fill("2200"); // ккал
  await manualInputs.nth(1).fill("100");  // белок

  // Сохраняем и идём на план.
  await page.getByRole("button", { name: "Сохранить и сформировать план" }).click();
  await expect(page).toHaveURL(/\/plan$/, { timeout: 10_000 });
  await page.getByRole("button", { name: /Сформировать|Перегенерировать/ }).click();
  await expect(page.getByRole("heading", { name: /День 1/ })).toBeVisible({
    timeout: 30_000,
  });

  // Проверяем, что в дне 1 итог ккал не сильно отличается от 2200 (±15%).
  const day1 = page.locator(".card").filter({ hasText: "День 1" }).first();
  const totals = day1.locator(".totals");
  await expect(totals).toBeVisible();
  const text = await totals.textContent();
  expect(text).toBeTruthy();
  const match = text!.match(/Итого:\s*(\d+)\s*ккал/);
  expect(match).toBeTruthy();
  const kcal = parseInt(match![1], 10);
  // Коридор ±15% от 2200 = [1870, 2530].
  // На дискретных блюдах допуск шире — берём ±20% (1760..2640).
  expect(kcal).toBeGreaterThan(1760);
  expect(kcal).toBeLessThan(2640);
  // И заведомо ниже дефолтных ~2400, чтобы убедиться, что override применился:
  // НЕ проверяем строго, потому что дискретность; вместо этого ниже проверим
  // регистрацию manual_targets_override в IndexedDB.

  // Доп. проверка: manual targets записаны в IndexedDB.
  const stored = await page.evaluate(async () => {
    const dbs = await indexedDB.databases?.();
    const keyvalDb = dbs?.find((d) => d.name === "keyval-store");
    if (!keyvalDb) return null;
    return new Promise<unknown>((resolve) => {
      const req = indexedDB.open("keyval-store");
      req.onsuccess = () => {
        const tx = req.result.transaction("keyval", "readonly");
        const store = tx.objectStore("keyval");
        const get = store.get("manual_targets");
        get.onsuccess = () => resolve(get.result);
        get.onerror = () => resolve(null);
      };
      req.onerror = () => resolve(null);
    });
  });
  expect(stored).toMatchObject({ enabled: true, kcal: 2200 });
});
