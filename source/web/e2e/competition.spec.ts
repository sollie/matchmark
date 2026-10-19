import { expect, test } from "@playwright/test";

const password = process.env.E2E_ADMIN_PASSWORD ?? "e2e-admin-password-1234";

test("admin signs in and creates a competition with the prefilled date", async ({ page }) => {
  await page.goto("/admin");
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Sign in" }).click();

  await expect(page.getByRole("heading", { name: "Competitions" })).toBeVisible();
  await expect(page.getByLabel("Date")).toHaveValue(/^\d{4}-\d{2}-\d{2}$/);

  await page.getByLabel("Name").fill("E2E club match");
  await page.getByRole("button", { name: "Create competition" }).click();

  await expect(page).toHaveURL(/\/admin\/competitions\/.+/);
  await expect(page.getByRole("heading", { name: /E2E club match/ })).toBeVisible();
});
