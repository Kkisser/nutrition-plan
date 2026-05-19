import { test, expect } from "@playwright/test";

// Сценарий замены блюда: открыть план, нажать 🔄 у первого блюда,
// выбрать в каталоге другое, вернуться на /plan и убедиться, что блюдо
// поменялось. Это критический путь — race condition pinning vs replace
// чинился ранее, регрессионный тест нужен.

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

test("план → замена первого блюда", async ({ page }) => {
  const email = `e2e_replace_${Date.now()}@example.com`;

  // Регистрация → /survey → /plan → сформировать.
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
  await page.getByRole("button", { name: /Сформировать|Перегенерировать/ }).click();
  await expect(page.getByRole("heading", { name: /День 1/ })).toBeVisible({
    timeout: 30_000,
  });

  // Запомним название первого блюда дня 1.
  const day1 = page.locator(".card").filter({ hasText: "День 1" }).first();
  const firstSlot = day1.locator("> div").first();
  const originalText = (await firstSlot.innerText()).trim();

  // Жмём 🔄 у первого блюда.
  await firstSlot.getByRole("button", { name: "🔄" }).click();
  await expect(page).toHaveURL(/\/replace$/, { timeout: 5_000 });

  // На /replace выбираем первое доступное блюдо в каталоге.
  // Кнопка «Выбрать» внутри карточки с альтернативами.
  await page.getByRole("button", { name: "Выбрать" }).first().click();

  // Вернулись на /plan, первое блюдо дня 1 изменилось.
  await expect(page).toHaveURL(/\/plan$/, { timeout: 5_000 });
  const newFirstSlot = page.locator(".card").filter({ hasText: "День 1" }).first().locator("> div").first();
  const newText = (await newFirstSlot.innerText()).trim();
  expect(newText).not.toBe(originalText);
});
