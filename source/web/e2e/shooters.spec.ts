import type { Page } from "@playwright/test";
import {
  addEntry,
  createMatch,
  createShooter,
  expect,
  row,
  setStatus,
  test,
  uniqueLastName,
} from "./helpers";

async function openShooters(page: Page) {
  await page.goto("/admin/shooters");
  await expect(page.getByRole("heading", { name: "Shooters" })).toBeVisible();
}

test.describe("shooters", () => {
  test("adding a shooter lists it", async ({ adminPage: page }) => {
    const lastName = uniqueLastName("Added");
    await openShooters(page);

    const form = page.locator("form");
    await form.getByLabel("Last name").fill(lastName);
    await form.getByLabel("First name").fill("Ada");
    await form.getByLabel("Country").fill("SWE");
    await form.getByLabel("Club").fill("Added Club");
    await form.getByRole("button", { name: "Add shooter" }).click();

    const added = row(page, lastName);
    await expect(added).toBeVisible();
    await expect(added.getByRole("cell", { name: `Ada ${lastName}` })).toBeVisible();
    await expect(added.getByRole("cell", { name: "SWE" })).toBeVisible();
    await expect(added.getByRole("cell", { name: "Added Club" })).toBeVisible();
    await expect(form.getByLabel("Last name")).toHaveValue("");
  });

  test("first name comes before last name and typing shows the normalized form", async ({
    adminPage: page,
  }) => {
    await openShooters(page);

    const form = page
      .locator("form")
      .filter({ has: page.getByRole("button", { name: "Add shooter" }) });
    const labels = await form.locator("label").allTextContents();
    expect(labels.findIndex((l) => l.startsWith("First name"))).toBeLessThan(
      labels.findIndex((l) => l.startsWith("Last name")),
    );

    const first = form.getByLabel("First name");
    const last = form.getByLabel("Last name");
    await first.pressSequentially("anna-lise o'neil");
    await last.pressSequentially("van der berg");
    await expect(first).toHaveValue("Anna-Lise O'Neil");
    await expect(last).toHaveValue("VAN DER BERG");

    await first.fill("mARY   ann ");
    await first.blur();
    await expect(first).toHaveValue("Mary Ann");
  });

  test("a lowercase name typed in the manual entry form is saved normalized", async ({
    adminPage: page,
  }) => {
    const suffix = uniqueLastName("Typed").replace(" ", "").toLowerCase();
    await page.route("**/api/wa1500/search*", (route) =>
      route.fulfill({ status: 503, json: { error: "wa1500 unavailable" } }),
    );
    await openShooters(page);
    await page.getByRole("button", { name: "Import from WA1500" }).click();
    await page.getByRole("textbox", { name: "Search WA1500" }).fill("anything");
    await page.getByRole("button", { name: "Search", exact: true }).click();

    const manual = page.getByRole("form", { name: "Manual shooter entry" });
    await expect(manual).toBeVisible();
    const labels = await manual.locator("label").allTextContents();
    expect(labels.findIndex((l) => l.startsWith("First name"))).toBeLessThan(
      labels.findIndex((l) => l.startsWith("Last name")),
    );
    await manual.getByLabel("First name").pressSequentially("lars-erik");
    await manual.getByLabel("Last name").pressSequentially(suffix);
    await expect(manual.getByLabel("First name")).toHaveValue("Lars-Erik");
    await expect(manual.getByLabel("Last name")).toHaveValue(suffix.toUpperCase());
    await manual.getByLabel("Country").fill("NOR");
    await manual.getByLabel("Club").fill("Typed Club");
    await manual.getByRole("button", { name: "Save shooter" }).click();

    await expect(
      row(page, suffix.toUpperCase()).getByRole("cell", {
        name: `Lars-Erik ${suffix.toUpperCase()}`,
      }),
    ).toBeVisible();
  });

  test("editing a shooter saves the change and cancelling restores the add form", async ({
    adminPage: page,
    admin,
  }) => {
    const shooter = await createShooter(admin, {
      last_name: uniqueLastName("Editable"),
      club: "Old Club",
    });
    const other = await createShooter(admin, { last_name: uniqueLastName("Untouched") });
    await openShooters(page);

    const form = page.locator("form");
    await row(page, shooter.last_name).getByRole("button", { name: "Edit" }).click();
    await expect(form.getByLabel("Last name")).toHaveValue(shooter.last_name);
    await expect(form.getByLabel("Club")).toHaveValue("Old Club");
    await expect(form.getByRole("button", { name: "Save" })).toBeVisible();
    await expect(form.getByRole("button", { name: "Add shooter" })).toHaveCount(0);

    await form.getByLabel("Club").fill("New Club");
    await form.getByRole("button", { name: "Save" }).click();

    await expect(
      row(page, shooter.last_name).getByRole("cell", { name: "New Club" }),
    ).toBeVisible();
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

  test("deleting asks for confirmation and keeps the shooter when declined", async ({
    adminPage: page,
    admin,
  }) => {
    const shooter = await createShooter(admin, { last_name: uniqueLastName("Doomed") });
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

  test("a shooter with entries cannot be deleted and links to their matches", async ({
    adminPage: page,
    admin,
  }) => {
    const shooter = await createShooter(admin, { last_name: uniqueLastName("Entered") });
    const draft = await createMatch(admin);
    const active = await createMatch(admin);
    await addEntry(admin, draft.id, shooter.id, 1);
    await addEntry(admin, active.id, shooter.id, 1);
    await setStatus(admin, active.id, "active");
    await openShooters(page);

    const entered = row(page, shooter.last_name);
    await expect(entered.getByRole("button", { name: "Delete" })).toBeDisabled();
    await entered.getByRole("button", { name: "Entered in 2 matches" }).click();
    await expect(entered.getByRole("link", { name: draft.name })).toHaveAttribute(
      "href",
      `/admin/matches/${draft.id}`,
    );
    await expect(entered.getByRole("link", { name: active.name })).toHaveAttribute(
      "href",
      `/admin/matches/${active.id}`,
    );
    await entered.getByRole("link", { name: draft.name }).click();
    await expect(page).toHaveURL(`/admin/matches/${draft.id}`);
  });

  test("a delete refused by the server shows the conflict", async ({ adminPage: page, admin }) => {
    const shooter = await createShooter(admin, { last_name: uniqueLastName("Raced") });
    await openShooters(page);
    await expect(
      row(page, shooter.last_name).getByRole("button", { name: "Delete" }),
    ).toBeEnabled();

    const match = await createMatch(admin);
    await addEntry(admin, match.id, shooter.id, 1);
    page.once("dialog", (dialog) => dialog.accept());
    await row(page, shooter.last_name).getByRole("button", { name: "Delete" }).click();

    await expect(page.getByRole("alert")).toContainText("cannot be deleted");
    await expect(
      row(page, shooter.last_name).getByRole("button", { name: "Entered in 1 match" }),
    ).toBeVisible();
    await expect(
      row(page, shooter.last_name).getByRole("button", { name: "Delete" }),
    ).toBeDisabled();
  });

  test("an incomplete form is blocked", async ({ adminPage: page }) => {
    const lastName = uniqueLastName("Incomplete");
    await openShooters(page);

    const form = page.locator("form");
    const requests: string[] = [];
    page.on("request", (request) => {
      if (request.method() === "POST" && request.url().endsWith("/api/shooters"))
        requests.push(request.url());
    });

    await form.getByLabel("Last name").fill(lastName);
    await form.getByLabel("First name").fill("Partial");
    await form.getByRole("button", { name: "Add shooter" }).click();

    expect(
      await form.getByLabel("Club").evaluate((el: HTMLInputElement) => el.validity.valueMissing),
    ).toBe(true);
    expect(requests).toHaveLength(0);
    await expect(row(page, lastName)).toHaveCount(0);
    await expect(form.getByLabel("Last name")).toHaveValue(lastName);
  });

  test("local search narrows the list by name or WAID", async ({ adminPage: page, admin }) => {
    const found = await createShooter(admin, {
      last_name: uniqueLastName("Findable"),
      waid: `E${Date.now()}`,
    });
    const hidden = await createShooter(admin, { last_name: uniqueLastName("Hidden") });
    await openShooters(page);

    await page.getByLabel("Search shooters").fill(found.waid);
    await expect(row(page, found.last_name)).toBeVisible();
    await expect(row(page, hidden.last_name)).toHaveCount(0);

    await page.getByLabel("Search shooters").fill("no-such-shooter-zzz");
    await expect(page.getByText("No matching shooters.")).toBeVisible();
  });

  test("importing a WA1500 hit lists the shooter", async ({ adminPage: page }) => {
    const lastName = uniqueLastName("Imported").replace(" ", "");
    const waid = `W${Date.now()}`;
    await page.route("**/api/wa1500/search*", (route) =>
      route.fulfill({
        json: {
          results: [
            { waid, first_name: "Ida", last_name: lastName, country: "SWE", club: "Remote Club" },
          ],
        },
      }),
    );
    await openShooters(page);
    await page.getByRole("button", { name: "Import from WA1500" }).click();
    await page.getByRole("textbox", { name: "Search WA1500" }).fill(lastName);
    await page.getByRole("button", { name: "Search", exact: true }).click();

    const importButton = page.getByRole("button", { name: `Import Ida ${lastName}` });
    await importButton.click();

    await expect(page.getByRole("status")).toContainText(`Saved Ida ${lastName}`);
    const added = row(page, lastName);
    await expect(added.getByRole("cell", { name: "Remote Club" })).toBeVisible();
  });

  test("a name conflict asks whether to update or create new", async ({
    adminPage: page,
    admin,
  }) => {
    const existing = await createShooter(admin, {
      last_name: uniqueLastName("Twin").replace(" ", ""),
      first_name: "Tim",
      club: "Local Club",
    });
    const waid = `W${Date.now()}`;
    await page.route("**/api/wa1500/search*", (route) =>
      route.fulfill({
        json: {
          results: [
            {
              waid,
              first_name: "Tim",
              last_name: existing.last_name,
              country: "DEN",
              club: "Remote Club",
            },
          ],
        },
      }),
    );
    await openShooters(page);
    await page.getByRole("button", { name: "Import from WA1500" }).click();
    await page.getByRole("textbox", { name: "Search WA1500" }).fill(existing.last_name);
    await page.getByRole("button", { name: "Search", exact: true }).click();
    await page.getByRole("button", { name: `Import Tim ${existing.last_name}` }).click();

    const dialog = page.getByRole("dialog", { name: "Name already exists" });
    await dialog.getByRole("button", { name: /Update Tim .* Local Club/ }).click();

    const updated = row(page, existing.last_name);
    await expect(updated.getByRole("cell", { name: "Remote Club" })).toBeVisible();
    await expect(updated.getByRole("cell", { name: waid })).toBeVisible();
    await expect(updated).toHaveCount(1);
  });

  test("manual entry is offered when WA1500 is unavailable", async ({ adminPage: page }) => {
    const lastName = uniqueLastName("Manual").replace(" ", "");
    await page.route("**/api/wa1500/search*", (route) =>
      route.fulfill({ status: 503, json: { error: "wa1500 unavailable" } }),
    );
    await openShooters(page);
    await page.getByRole("button", { name: "Import from WA1500" }).click();
    await page.getByRole("textbox", { name: "Search WA1500" }).fill("anything");
    await page.getByRole("button", { name: "Search", exact: true }).click();
    await expect(page.getByRole("alert")).toContainText("wa1500 unavailable");

    const manual = page.getByRole("form", { name: "Manual shooter entry" });
    await manual.getByLabel("Last name").fill(lastName);
    await manual.getByLabel("First name").fill("Max");
    await manual.getByLabel("Country").fill("NOR");
    await manual.getByLabel("Club").fill("Manual Club");
    await manual.getByRole("button", { name: "Save shooter" }).click();

    await expect(row(page, lastName).getByRole("cell", { name: "Manual Club" })).toBeVisible();
  });
});
