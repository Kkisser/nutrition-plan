import { defineConfig, devices } from "@playwright/test";

// E2E-конфигурация. Запуск:
//   npm run e2e             — headless
//   npm run e2e -- --ui     — интерактивный режим
//
// Поднимает vite dev (port 5173) и идёт по живому core на :8086.
// Core должен быть уже запущен (см. dev_environment).
export default defineConfig({
  testDir: "./e2e",
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: 1,
  reporter: process.env.CI ? "github" : "list",
  use: {
    baseURL: "http://localhost:5173",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  webServer: {
    command: "npm run dev",
    url: "http://localhost:5173",
    reuseExistingServer: !process.env.CI,
    timeout: 60_000,
  },
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
  ],
});
