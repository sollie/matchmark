import { expect, newApi, row, signInViaUi, test, uniqueLastName, uniqueName } from "./helpers";

const temporaryPassword = "e2e-temporary-password-1";
const chosenPassword = "e2e-chosen-password-5678";

function newUsername(): string {
  return uniqueName("u").replace(" ", "-");
}

test.describe("user management", () => {
  test("an admin creates a user who must change the temporary password before using the app", async ({
    adminPage: page,
  }) => {
    const username = newUsername();
    await page.getByRole("link", { name: "Users" }).click();
    await expect(page.getByRole("heading", { name: "Users" })).toBeVisible();

    await page.getByLabel("New username").fill(username);
    await page.getByLabel("Temporary password").fill(temporaryPassword);
    await page.getByRole("button", { name: "Create user" }).click();
    await expect(page.getByRole("status")).toContainText(username);
    await expect(row(page, username)).toContainText("User");
    await expect(row(page, username)).toContainText("Must change password");

    await page.getByLabel("New username").fill(username.toUpperCase());
    await page.getByLabel("Temporary password").fill(temporaryPassword);
    await page.getByRole("button", { name: "Create user" }).click();
    await expect(page.getByRole("alert")).toHaveText("conflict");

    await page.getByLabel("New username").fill("bad name");
    await page.getByLabel("Temporary password").fill(temporaryPassword);
    await page.getByRole("button", { name: "Create user" }).click();
    await expect(row(page, "bad name")).toHaveCount(0);

    await page.getByRole("button", { name: "Sign out" }).click();
    await signInViaUi(page, temporaryPassword, username);

    await expect(page.getByRole("heading", { name: "Change password" })).toBeVisible();
    await expect(page.getByText("You must choose a new password before you can continue.")).toBeVisible();
    await expect(page.getByRole("link", { name: "Shooters" })).toHaveCount(0);
    expect((await page.request.get("/api/shooters")).status()).toBe(403);

    await page.getByLabel("Current password").fill(temporaryPassword);
    await page.getByLabel("New password", { exact: true }).fill(chosenPassword);
    await page.getByLabel("Confirm new password").fill(chosenPassword);
    await page.getByRole("button", { name: "Change password" }).click();

    await expect(page.getByRole("heading", { name: "Matches" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Shooters" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Users" })).toHaveCount(0);
    await expect(page.getByRole("link", { name: "Backup" })).toHaveCount(0);
    expect((await page.request.get("/api/shooters")).status()).toBe(200);
    expect((await page.request.get("/api/users")).status()).toBe(403);

    await page.goto("/admin/users");
    await expect(page.getByText("Only admins can manage users.")).toBeVisible();
  });

  test("an admin grants and revokes the backup permission and the Backup link follows", async ({ adminPage: page, baseURL }) => {
    const username = newUsername();
    await page.getByRole("link", { name: "Users" }).click();
    await page.getByLabel("New username").fill(username);
    await page.getByLabel("Role").selectOption("admin");
    await page.getByLabel("Temporary password").fill(temporaryPassword);
    await page.getByRole("button", { name: "Create user" }).click();
    await expect(row(page, username)).toContainText("Not allowed");

    await page.getByRole("button", { name: "Sign out" }).click();
    await signInViaUi(page, temporaryPassword, username);
    await page.getByLabel("Current password").fill(temporaryPassword);
    await page.getByLabel("New password", { exact: true }).fill(chosenPassword);
    await page.getByLabel("Confirm new password").fill(chosenPassword);
    await page.getByRole("button", { name: "Change password" }).click();
    await expect(page.getByRole("heading", { name: "Matches" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Backup" })).toHaveCount(0);
    expect((await page.request.get("/api/admin/backup")).status()).toBe(403);

    await page.getByRole("button", { name: "Sign out" }).click();
    await signInViaUi(page);
    await page.getByRole("link", { name: "Users" }).click();
    await row(page, username).getByRole("button", { name: "Allow backup" }).click();
    await expect(row(page, username)).toContainText("Allowed");
    await expect(row(page, username).getByRole("button", { name: "Forbid backup" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Backup" })).toBeVisible();
    await row(page, username).getByRole("button", { name: "Forbid backup" }).click();
    await expect(row(page, username)).toContainText("Not allowed");
  });

  test("a regular user can create a shooter", async ({ admin, baseURL }) => {
    const username = newUsername();
    const created = await admin.post("/api/users", { data: { username, role: "user", password: temporaryPassword } });
    expect(created.status()).toBe(201);

    const api = await newApi(baseURL!);
    expect((await api.post("/api/admin/login", { data: { username, password: temporaryPassword } })).status()).toBe(204);
    expect(
      (await api.post("/api/admin/password", { data: { current_password: temporaryPassword, new_password: chosenPassword } })).status(),
    ).toBe(204);
    const shooter = await api.post("/api/shooters", {
      data: { last_name: uniqueLastName("User"), first_name: "Test", country: "NOR", club: "E2E Club", waid: "" },
    });
    expect(shooter.status()).toBe(201);
    expect((await api.get("/api/users")).status()).toBe(403);
    await api.dispose();
  });
});

test.describe("user lifecycle", () => {
  test("an admin resets, demotes, deactivates and reactivates a user", async ({ adminPage: page, baseURL }) => {
    const username = newUsername();
    await page.getByRole("link", { name: "Users" }).click();
    await page.getByLabel("New username").fill(username);
    await page.getByLabel("Role").selectOption("admin");
    await page.getByLabel("Temporary password").fill(temporaryPassword);
    await page.getByRole("button", { name: "Create user" }).click();
    await expect(row(page, username)).toContainText("Admin");

    const api = await newApi(baseURL!);
    expect((await api.post("/api/admin/login", { data: { username, password: temporaryPassword } })).status()).toBe(204);
    expect(
      (await api.post("/api/admin/password", { data: { current_password: temporaryPassword, new_password: chosenPassword } })).status(),
    ).toBe(204);
    expect((await api.get("/api/users")).status()).toBe(200);

    await row(page, username).getByRole("button", { name: "Make user" }).click();
    await expect(row(page, username)).toContainText("User");
    expect((await api.get("/api/admin/session")).status()).toBe(401);

    await row(page, username).getByRole("button", { name: "Reset password" }).click();
    await page.getByLabel(`Temporary password for ${username}`).fill(temporaryPassword);
    await page.getByRole("button", { name: "Confirm reset" }).click();
    await expect(row(page, username)).toContainText("Must change password");

    await row(page, username).getByRole("button", { name: "Deactivate" }).click();
    await expect(row(page, username)).toContainText("Deactivated");
    expect((await api.post("/api/admin/login", { data: { username, password: temporaryPassword } })).status()).toBe(401);

    await row(page, username).getByRole("button", { name: "Reactivate" }).click();
    await expect(row(page, username)).toContainText("Must change password");
    expect((await api.post("/api/admin/login", { data: { username, password: temporaryPassword } })).status()).toBe(204);
    await api.dispose();
  });
});
