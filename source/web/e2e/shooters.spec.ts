import { expect, test, type Page } from "@playwright/test";
import { adminApi, createShooter, row, signInViaUi, uniqueClientIp, uniqueName } from "./helpers";

async function openShooters(page: Page) {
  await signInViaUi(page);
  await expect(page.getByRole("heading", { name: "Competitions" })).toBeVisible();
  await page.goto("/admin/shooters");
  await expect(page.getByRole("heading", { name: "Shooters" })).toBeVisible();
}

test.describe("shooters", () => {
  test.beforeEach(async ({ page }) => {
    await page.setExtraHTTPHeaders({ "X-Forwarded-For": uniqueClientIp() });
  });

  test("adding a shooter lists it", async ({ page }) => {
    const lastName = uniqueName("Added");
    await openShooters(page);

    const form = page.locator("form");
    await form.getByLabel("Last name").fill(lastName);
    await form.getByLabel("First name").fill("Ada");
    await form.getByLabel("Sex").selectOption("F");
    await form.getByLabel("Country").fill("SWE");
    await form.getByLabel("Club").fill("Added Club");
    await form.getByRole("button", { name: "Add shooter" }).click();

    const added = row(page, lastName);
    await expect(added).toBeVisible();
    await expect(added.getByRole("cell", { name: `${lastName}, Ada` })).toBeVisible();
    await expect(added.getByRole("cell", { name: "F", exact: true })).toBeVisible();
    await expect(added.getByRole("cell", { name: "SWE" })).toBeVisible();
    await expect(added.getByRole("cell", { name: "Added Club" })).toBeVisible();
    await expect(form.getByLabel("Last name")).toHaveValue("");
  });

  test("editing a shooter saves the change and cancelling restores the add form", async ({ page, baseURL }) => {
    const api = await adminApi(baseURL!);
    const shooter = await createShooter(api, { last_name: uniqueName("Editable"), club: "Old Club" });
    const other = await createShooter(api, { last_name: uniqueName("Untouched") });
    await api.dispose();
    await openShooters(page);

    const form = page.locator("form");
    await row(page, shooter.last_name).getByRole("button", { name: "Edit" }).click();
    await expect(form.getByLabel("Last name")).toHaveValue(shooter.last_name);
    await expect(form.getByLabel("Club")).toHaveValue("Old Club");
    await expect(form.getByRole("button", { name: "Save" })).toBeVisible();
    await expect(form.getByRole("button", { name: "Add shooter" })).toHaveCount(0);

    await form.getByLabel("Club").fill("New Club");
    await form.getByRole("button", { name: "Save" }).click();

    await expect(row(page, shooter.last_name).getByRole("cell", { name: "New Club" })).toBeVisible();
    await expect(row(page, other.last_name).getByRole("cell", { name: "E2E Club" })).toBeVisible();
    await expect(form.getByRole("button", { name: "Add shooter" })).toBeVisible();

    await row(page, other.last_name).getByRole("button", { name: "Edit" }).click();
    await expect(form.getByLabel("Last name")).toHaveValue(other.last_name);
    await form.getByRole("button", { name: "Cancel" }).click();

    await expect(form.getByRole("button", { name: "Add shooter" })).toBeVisible();
    await expect(form.getByRole("button", { name: "Save" })).toHaveCount(0);
    await expect(form.getByRole("button", { name: "Cancel" })).toHaveCount(0);
    await expect(form.getByLabel("Last name")).toHaveValue("");
    await expect(form.getByLabel("Club")).toHaveValue("");
    await expect(row(page, other.last_name)).toBeVisible();
  });

  test("deleting asks for confirmation and keeps the shooter when declined", async ({ page, baseURL }) => {
    const api = await adminApi(baseURL!);
    const shooter = await createShooter(api, { last_name: uniqueName("Doomed") });
    await api.dispose();
    await openShooters(page);

    const dialogs: string[] = [];
    let accept = false;
    page.on("dialog", async (dialog) => {
      dialogs.push(dialog.message());
      if (accept) await dialog.accept();
      else await dialog.dismiss();
    });

    await row(page, shooter.last_name).getByRole("button", { name: "Delete" }).click();
    await expect.poll(() => dialogs.length).toBe(1);
    expect(dialogs[0]).toBe(`Delete ${shooter.first_name} ${shooter.last_name}?`);
    await expect(row(page, shooter.last_name)).toBeVisible();

    accept = true;
    await row(page, shooter.last_name).getByRole("button", { name: "Delete" }).click();
    await expect(row(page, shooter.last_name)).toHaveCount(0);
    await page.reload();
    await expect(page.getByRole("heading", { name: "Shooters" })).toBeVisible();
    await expect(row(page, shooter.last_name)).toHaveCount(0);
  });

  test("an incomplete form is blocked", async ({ page }) => {
    const lastName = uniqueName("Incomplete");
    await openShooters(page);

    const form = page.locator("form");
    const requests: string[] = [];
    page.on("request", (request) => {
      if (request.method() === "POST" && request.url().endsWith("/api/shooters")) requests.push(request.url());
    });

    await form.getByLabel("Last name").fill(lastName);
    await form.getByLabel("First name").fill("Partial");
    await form.getByRole("button", { name: "Add shooter" }).click();

    expect(await form.getByLabel("Club").evaluate((el: HTMLInputElement) => el.validity.valueMissing)).toBe(true);
    expect(requests).toHaveLength(0);
    await expect(row(page, lastName)).toHaveCount(0);
    await expect(form.getByLabel("Last name")).toHaveValue(lastName);
  });
});
