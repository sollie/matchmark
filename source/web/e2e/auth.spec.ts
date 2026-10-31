import {
  adminPassword,
  adminUsername,
  expect,
  newApi,
  signInViaUi,
  signOutViaMenu,
  test,
} from "./helpers";

test.describe("auth", () => {
  test("a wrong password shows an error and a correct one signs in", async ({ page }) => {
    await signInViaUi(page, "wrong-password-value");
    await expect(page.getByRole("alert")).toBeVisible();
    await expect(page.getByRole("heading", { name: "Admin sign in" })).toBeVisible();

    await page.getByLabel("Password", { exact: true }).fill(adminPassword);
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(page.getByRole("heading", { name: "Matches" })).toBeVisible();
  });

  test("an unknown username is rejected like a wrong password", async ({ page }) => {
    await signInViaUi(page, adminPassword, "nobody-here");
    await expect(page.getByRole("alert")).toHaveText(/unauthorized/i);
    await expect(page.getByRole("heading", { name: "Admin sign in" })).toBeVisible();
  });

  test("repeated failures lock out one client without affecting another", async ({
    page,
    baseURL,
  }) => {
    await page.goto("/admin");
    for (let attempt = 0; attempt < 5; attempt++) {
      await page.getByLabel("Username").fill(adminUsername);
      await page.getByLabel("Password", { exact: true }).fill(`wrong-password-${attempt}`);
      await page.getByRole("button", { name: "Sign in" }).click();
      await expect(page.getByRole("alert")).toHaveText(/unauthorized/i);
    }

    await page.getByLabel("Username").fill(adminUsername);
    await page.getByLabel("Password", { exact: true }).fill(adminPassword);
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(page.getByRole("alert")).toContainText("Too many attempts");
    await expect(page.getByRole("heading", { name: "Admin sign in" })).toBeVisible();

    const other = await newApi(baseURL!);
    const response = await other.post("/api/admin/login", {
      data: { username: adminUsername, password: adminPassword },
    });
    expect(response.status()).toBe(204);
    await other.dispose();
  });

  test("sign out ends the session and admin pages show the sign-in form", async ({
    adminPage: page,
  }) => {
    expect((await page.request.get("/api/shooters")).status()).toBe(200);

    await signOutViaMenu(page);
    await expect(page).toHaveURL(/\/$/);

    await page.goto("/admin/shooters");
    await expect(page.getByRole("heading", { name: "Admin sign in" })).toBeVisible();
    await page.goto("/admin");
    await expect(page.getByRole("heading", { name: "Admin sign in" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "Matches" })).toHaveCount(0);
    expect((await page.request.get("/api/shooters")).status()).toBe(401);
  });

  test("unauthenticated admin API calls return 401", async ({ baseURL }) => {
    const api = await newApi(baseURL!);
    for (const path of [
      "/api/admin/session",
      "/api/shooters",
      "/api/matches",
      "/api/admin/backup",
    ]) {
      const response = await api.get(path);
      expect(response.status(), path).toBe(401);
    }
    await api.dispose();
  });

  test("an unknown URL shows the not-found page with a link back", async ({ page }) => {
    await page.goto("/no/such/page");
    await expect(page.getByRole("heading", { name: "Page not found" })).toBeVisible();
    await page.getByRole("link", { name: "Back to start" }).click();
    await expect(page).toHaveURL(/\/$/);
    await expect(page.getByRole("heading", { name: "Page not found" })).toHaveCount(0);
  });

  test("the scorer page without a pairing explains how to pair", async ({ page }) => {
    await page.goto("/score");
    await expect(page.getByRole("heading", { name: "Scorer" })).toBeVisible();
    await expect(
      page.getByText("Open a pairing link from the admin to score a lane."),
    ).toBeVisible();
  });
});
