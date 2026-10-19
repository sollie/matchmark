import { mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { defineConfig, devices } from "@playwright/test";

const external = process.env.E2E_BASE_URL;
const port = process.env.E2E_PORT ?? "18080";
const baseURL = external ?? `http://127.0.0.1:${port}`;
const dbDir = mkdtempSync(join(tmpdir(), "matchmark-e2e-"));

export default defineConfig({
  testDir: "e2e",
  fullyParallel: false,
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["github"], ["html", { open: "never" }]] : "list",
  use: { baseURL, trace: "retain-on-failure" },
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "firefox", use: { ...devices["Desktop Firefox"] } },
    { name: "webkit", use: { ...devices["Desktop Safari"] } },
  ],
  webServer: external
    ? undefined
    : {
        command: "go run ./cmd/server",
        cwd: "..",
        url: `${baseURL}/`,
        reuseExistingServer: false,
        timeout: 180_000,
        env: {
          CGO_ENABLED: "0",
          ADMIN_PASSWORD: process.env.E2E_ADMIN_PASSWORD ?? "e2e-admin-password-1234",
          DB_PATH: join(dbDir, "matchmark.db"),
          LISTEN_ADDR: `127.0.0.1:${port}`,
        },
      },
});
