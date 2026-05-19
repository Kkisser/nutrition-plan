import { test, expect } from "@playwright/test";

// Сценарий «зарегистрировался → анкета → план». Самый длинный из e2e,
// проверяет двухфазный планировщик через реальный API.

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

test("регистрация → анкета → план", async ({ page }) => {
  const email = `e2e_plan_${Date.now()}@example.com`;

  // 1. Регистрация
  await page.goto("/login");
  await page.getByRole("button", { name: "Регистрация" }).click();
  await page.locator('input[type="email"]').fill(email);
  const passwords = page.locator('input[type="password"]');
  await passwords.nth(0).fill("TestPass123");
  await passwords.nth(1).fill("TestPass123");
  await page.getByRole("button", { name: "Зарегистрироваться" }).click();

  await expect(page).toHaveURL(/\/survey$/, { timeout: 10_000 });

  // 2. Survey: оставляем дефолты (male, 30, 180см, 75кг, classic).
  await page.getByRole("button", { name: "Сохранить и сформировать план" }).click();
  await expect(page).toHaveURL(/\/plan$/, { timeout: 10_000 });

  // 3. Plan: жмём «Сформировать», ждём 7 дней.
  await page.getByRole("button", { name: /Сформировать|Перегенерировать/ }).click();
  await expect(page.getByRole("heading", { name: /День 1/ })).toBeVisible({
    timeout: 30_000,
  });
  for (let d = 1; d <= 7; d++) {
    await expect(page.getByRole("heading", { name: `День ${d}` })).toBeVisible();
  }
});
