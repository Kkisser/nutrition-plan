import { test, expect } from "@playwright/test";

// Сценарий: «зарегистрировался → план → корзина с категориями».
// Проверяет, что shopping_list действительно сгруппирован.

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

test("план → покупки по категориям → история", async ({ page }) => {
  const email = `e2e_shop_${Date.now()}@example.com`;

  // 1. Регистрация + анкета по дефолтам.
  await page.goto("/login");
  await page.getByRole("button", { name: "Регистрация" }).click();
  await page.locator('input[type="email"]').fill(email);
  const passwords = page.locator('input[type="password"]');
  await passwords.nth(0).fill("TestPass123");
  await passwords.nth(1).fill("TestPass123");
  await page.getByRole("button", { name: "Зарегистрироваться" }).click();
  await expect(page).toHaveURL(/\/survey$/, { timeout: 10_000 });
  await page.getByRole("button", { name: "Сохранить и сформировать план" }).click();
  await expect(page).toHaveURL(/\/plan$/, { timeout: 10_000 });

  // 2. Формируем план.
  await page.getByRole("button", { name: /Сформировать|Перегенерировать/ }).click();
  await expect(page.getByRole("heading", { name: /День 1/ })).toBeVisible({
    timeout: 30_000,
  });

  // 3. Уходим в покупки и проверяем категории.
  await page.getByRole("link", { name: "Покупки" }).click();
  await expect(page).toHaveURL(/\/shopping$/);
  await expect(page.getByText(/позиций в .* категориях/)).toBeVisible();

  // Должно быть несколько типичных категорий из CATEGORY_ORDER.
  // На classic-диете с дефолтным профилем заведомо появляются Крупы и Овощи.
  await expect(page.getByRole("heading", { name: "Крупы" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Овощи" })).toBeVisible();

  // 4. История недель.
  await page.getByRole("link", { name: "История" }).click();
  await expect(page).toHaveURL(/\/history$/);
  // Должна быть хотя бы одна сохранённая неделя.
  await expect(page.locator(".card").first()).toBeVisible();
  await expect(page.getByRole("button", { name: "Открыть" }).first()).toBeVisible();
});
