import { adminPassword, adminUsername, expect, newApi, openAccountLink, test } from "./helpers";

const changedPassword = "e2e-changed-password-5678";

test.describe("change password", () => {
  test("a wrong current password is rejected, a valid change ends other sessions and can be reverted", async ({
    adminPage: page,
    baseURL,
  }) => {
    const other = await newApi(baseURL!);
    expect(
      (
        await other.post("/api/admin/login", {
          data: { username: adminUsername, password: adminPassword },
        })
      ).status(),
    ).toBe(204);
    expect((await other.get("/api/shooters")).status()).toBe(200);

    await openAccountLink(page, "Change password");
    await page.getByLabel("Current password", { exact: true }).fill("not-the-current-password");
    await page.getByLabel("New password", { exact: true }).fill(changedPassword);
    await page.getByLabel("Confirm new password", { exact: true }).fill(changedPassword);
    await page.getByRole("button", { name: "Change password" }).click();
    await expect(page.getByRole("alert")).toHaveText("The current password is incorrect");
    expect((await other.get("/api/shooters")).status()).toBe(200);

    await page.getByLabel("Current password", { exact: true }).fill(adminPassword);
    await page.getByRole("button", { name: "Change password" }).click();
    await expect(page.getByRole("heading", { name: "Password changed" })).toBeVisible();
    await expect(page.getByText("Your other sessions were signed out")).toBeVisible();
    expect((await other.get("/api/shooters")).status()).toBe(401);
    await other.dispose();

    await page.getByRole("button", { name: "Continue" }).click();
    await openAccountLink(page, "Change password");
    await page.getByLabel("Current password", { exact: true }).fill(changedPassword);
    await page.getByLabel("New password", { exact: true }).fill(adminPassword);
    await page.getByLabel("Confirm new password", { exact: true }).fill(adminPassword);
    await page.getByRole("button", { name: "Change password" }).click();
    await expect(page.getByRole("heading", { name: "Password changed" })).toBeVisible();
  });

  test("the password fields can be revealed and show length and match feedback", async ({
    adminPage: page,
    browser,
    baseURL,
  }) => {
    const anon = await (await browser.newContext({ baseURL })).newPage();
    await anon.goto("/admin");
    const login = anon.getByLabel("Password", { exact: true });
    await expect(login).toHaveAttribute("type", "password");
    const loginToggle = anon.getByRole("button", { name: "Show password" });
    await expect(loginToggle).toHaveAttribute("aria-pressed", "false");
    await loginToggle.click();
    await expect(login).toHaveAttribute("type", "text");
    await anon.getByRole("button", { name: "Hide password" }).click();
    await expect(login).toHaveAttribute("type", "password");
    await anon.close();

    await openAccountLink(page, "Change password");
    const next = page.getByLabel("New password", { exact: true });
    const confirm = page.getByLabel("Confirm new password", { exact: true });
    await page.getByRole("button", { name: "Show password" }).nth(1).click();
    await expect(next).toHaveAttribute("type", "text");
    await expect(confirm).toHaveAttribute("type", "password");

    await next.fill("short-one");
    await expect(page.getByText("9 / 16 characters")).toBeVisible();
    await next.fill("a-long-enough-password");
    await expect(page.getByText("✓ 16 characters or more")).toBeVisible();

    await confirm.fill("a-long");
    await expect(page.getByText("Passwords do not match")).toBeVisible();
    await confirm.fill("a-long-enough-password");
    await expect(page.getByText("✓ Passwords match")).toBeVisible();
  });
});
