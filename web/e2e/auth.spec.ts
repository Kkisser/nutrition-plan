import { test, expect } from "@playwright/test";

// Базовый critical-path: регистрация → авто-верификация → переход на анкету.
//
// Селекторы по типу input + placeholder используются вместо getByLabel,
// потому что разметка проекта не связывает <label> с <input> через htmlFor
// (label/input соседствуют внутри .field, но не связаны явно). Менять
// разметку всего фронта только ради e2e — не оправдано; используем
// устойчивые селекторы по DOM-структуре.

function uniqueEmail(): string {
  return `e2e_${Date.now()}_${Math.random().toString(36).slice(2, 8)}@example.com`;
}

// Перед каждым тестом — чистая БД браузера. Иначе auth-токен и idb-keyval
// плана/анкеты «протекают» из предыдущего теста, ломая шаги вроде
// «после регистрации мы на /login → должны прыгнуть на /survey».
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

test.describe("auth flow", () => {
  test("регистрация → анкета", async ({ page }) => {
    await page.goto("/login");

    // На /login две кнопки: «Войти» (submit) и «Регистрация» (переключатель).
    // Переключаемся на режим регистрации.
    await page.getByRole("button", { name: "Регистрация" }).click();
    await expect(page.getByRole("heading", { name: "Регистрация" })).toBeVisible();

    const email = uniqueEmail();
    await page.locator('input[type="email"]').fill(email);
    // У страницы 2 password-инпута в режиме регистрации (пароль + повтор).
    const passwords = page.locator('input[type="password"]');
    await passwords.nth(0).fill("TestPass123");
    await passwords.nth(1).fill("TestPass123");

    await page.getByRole("button", { name: "Зарегистрироваться" }).click();

    await expect(page).toHaveURL(/\/survey$/, { timeout: 10_000 });
    await expect(page.getByRole("heading", { name: "Анкета профиля" })).toBeVisible();
  });

  test("вход с неверным паролем показывает ошибку", async ({ page }) => {
    await page.goto("/login");
    await page.locator('input[type="email"]').fill("nonexistent_user@example.com");
    await page.locator('input[type="password"]').first().fill("WrongPass123");
    await page.getByRole("button", { name: "Войти" }).click();

    await expect(page.locator(".warn")).toBeVisible({ timeout: 5_000 });
  });

  test("валидация пароля на стороне фронта", async ({ page }) => {
    await page.goto("/login");
    await page.getByRole("button", { name: "Регистрация" }).click();

    await page.locator('input[type="email"]').fill(uniqueEmail());
    const passwords = page.locator('input[type="password"]');
    await passwords.nth(0).fill("short");
    await passwords.nth(1).fill("short");

    await page.getByRole("button", { name: "Зарегистрироваться" }).click();
    await expect(page.locator(".warn")).toBeVisible({ timeout: 2_000 });
    await expect(page).not.toHaveURL(/\/survey$/);
  });
});
