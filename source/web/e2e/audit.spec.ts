import { createMatch, expect, row, test, uniqueName } from "./helpers";

test.describe("audit log", () => {
  test("an admin finds who did what, filtered by user and action, and pages through it", async ({
    adminPage: page,
    admin,
  }) => {
    const { name } = await createMatch(admin, { name: uniqueName("Audited") });

    await page.getByRole("link", { name: "Audit log" }).click();
    await expect(page.getByRole("heading", { name: "Audit log" })).toBeVisible();
    await expect(row(page, "match.create").filter({ hasText: name }).first()).toContainText("e2e-admin");

    await page.getByLabel("User").fill("unknown-audit-user");
    await page.getByRole("button", { name: "Filter" }).click();
    await expect(page.getByText("No entries match.")).toBeVisible();

    await page.getByLabel("User").fill("E2E-Admin");
    await page.getByLabel("Action").fill("login");
    await page.getByRole("button", { name: "Filter" }).click();
    await expect(row(page, "login").first()).toContainText("e2e-admin");
    await expect(page.getByRole("row").filter({ hasText: "match.create" })).toHaveCount(0);

    await page.getByLabel("Action").fill("");
    await page.getByLabel("User").fill("");
    await page.getByLabel("From").fill("2999-01-01");
    await page.getByRole("button", { name: "Filter" }).click();
    await expect(page.getByText("No entries match.")).toBeVisible();
    await page.getByLabel("From").fill("");
    await page.getByRole("button", { name: "Filter" }).click();
    await expect(page.getByRole("button", { name: "Previous" })).toBeDisabled();
    await expect(page.getByText(/^Page 1 of \d+/)).toBeVisible();
  });

  test("only admins see the audit log", async ({ adminPage: page, admin }) => {
    const username = uniqueName("u").replace(" ", "-");
    const temporary = "e2e-temporary-password-1";
    const chosen = "e2e-chosen-password-5678";
    await admin.post("/api/users", { data: { username, role: "user", password: temporary } });
    await page.getByRole("button", { name: "Sign out" }).click();
    await page.getByLabel("Username").fill(username);
    await page.getByLabel("Password").fill(temporary);
    await page.getByRole("button", { name: "Sign in" }).click();
    await page.getByLabel("Current password").fill(temporary);
    await page.getByLabel("New password", { exact: true }).fill(chosen);
    await page.getByLabel("Confirm new password").fill(chosen);
    await page.getByRole("button", { name: "Change password" }).click();
    await expect(page.getByRole("heading", { name: "Matches" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Audit log" })).toHaveCount(0);
    await page.goto("/admin/audit");
    await expect(page.getByText("Only admins can read the audit log.")).toBeVisible();
    expect((await page.request.get("/api/audit-log")).status()).toBe(403);
  });
});
