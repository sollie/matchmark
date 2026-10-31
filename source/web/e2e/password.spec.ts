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
    await expect(page.getByRole("status")).toHaveText("Password changed.");
    expect((await other.get("/api/shooters")).status()).toBe(401);
    await other.dispose();

    await page.getByLabel("Current password", { exact: true }).fill(changedPassword);
    await page.getByLabel("New password", { exact: true }).fill(adminPassword);
    await page.getByLabel("Confirm new password", { exact: true }).fill(adminPassword);
    await page.getByRole("button", { name: "Change password" }).click();
    await expect(page.getByRole("status")).toHaveText("Password changed.");
  });
});
