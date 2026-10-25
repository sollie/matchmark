import type { Page } from "@playwright/test";
import { adminUsername, type Category, createCompetition, createShooter, expect, openCompetition, row, section, seedEntry, setCounts, setStatus, test } from "./helpers";

function addForm(page: Page) {
  return page.getByRole("form", { name: "Add entry" });
}

test.describe("entries", () => {
  test("adding an entry uses the chosen category and the default lane", async ({ course, adminPage: page, admin }) => {
    const competition = await createCompetition(admin);
    await seedEntry(admin, competition.id, 1);
    const shooter = await createShooter(admin);
    await openCompetition(page, competition.id);

    const form = addForm(page);
    await expect(page.getByLabel("Entry lane")).toHaveValue("2");
    await form.getByRole("combobox", { name: "Shooter" }).fill(shooter.last_name);
    await page.getByRole("option", { name: new RegExp(shooter.last_name) }).click();
    await form.getByLabel("Entry category").selectOption("Pistol");
    await form.getByRole("button", { name: "Add entry" }).click();

    const added = row(page, shooter.last_name);
    await expect(added).toBeVisible();
    await expect(added.locator("td.lane")).toHaveText("2");
    await expect(added.locator("td.category")).toHaveText("Pistol");
  });

  test("a taken lane is rejected and an entered shooter is not offered again", async ({ course, adminPage: page, admin }) => {
    const competition = await createCompetition(admin);
    const taken = await seedEntry(admin, competition.id, 1);
    const other = await createShooter(admin);
    await openCompetition(page, competition.id);

    const form = addForm(page);
    const picker = form.getByRole("combobox", { name: "Shooter" });
    await picker.fill(taken.shooter.last_name);
    await expect(page.getByText("No match")).toBeVisible();

    await picker.fill(other.last_name);
    await page.getByRole("option", { name: new RegExp(other.last_name) }).click();
    await page.getByLabel("Entry lane").fill("1");
    await form.getByRole("button", { name: "Add entry" }).click();

    await expect(section(page, "Entries and progress").getByRole("alert")).toHaveText("conflict");
    await expect(row(page, other.last_name)).toHaveCount(0);
    await expect(picker).toHaveValue(`${other.first_name} ${other.last_name}`);
  });

  test("an entry can be moved, withdrawn and removed", async ({ course, adminPage: page, admin }) => {
    const competition = await createCompetition(admin);
    const entry = await seedEntry(admin, competition.id, 1);
    await openCompetition(page, competition.id);
    page.on("dialog", (dialog) => dialog.accept());

    const entryRow = row(page, entry.shooter.last_name);
    await entryRow.getByRole("button", { name: "Move", exact: true }).click();
    await entryRow.getByLabel("Move lane").fill("5");
    await entryRow.getByLabel("Move category").selectOption("Pistol OS");
    await entryRow.getByRole("button", { name: "Save", exact: true }).click();
    await expect(entryRow.locator("td.lane")).toHaveText("5");
    await expect(entryRow.locator("td.category")).toHaveText("Pistol OS");

    await entryRow.getByRole("button", { name: "Withdraw" }).click();
    await expect(entryRow.getByText("Withdrawn", { exact: true })).toBeVisible();
    await expect(entryRow.getByRole("button", { name: "Withdraw" })).toHaveCount(0);

    await entryRow.getByRole("button", { name: "Remove", exact: true }).click();
    await expect(row(page, entry.shooter.last_name)).toHaveCount(0);
  });
});

test.describe("entries layout", () => {
  test("every stage is reachable without horizontal scrolling", async ({ course, adminPage: page, admin }) => {
    const competition = await createCompetition(admin);
    const entry = await seedEntry(admin, competition.id, 1);
    await page.setViewportSize({ width: 1024, height: 768 });
    await openCompetition(page, competition.id);

    const stages = row(page, entry.shooter.last_name).locator("td.stage");
    await expect(stages).toHaveCount(course.stages);
    const scroll = page.locator(".scroll", { has: page.locator("table.entries") });
    expect(await scroll.evaluate((el) => el.scrollWidth <= el.clientWidth)).toBe(true);
    await expect(stages.last()).toBeInViewport();
  });
});

test.describe("admin ring counts", () => {
  test("a live shot total flags a wrong total and the save is rejected", async ({ course, adminPage: page, admin }) => {
    const competition = await createCompetition(admin);
    const entry = await seedEntry(admin, competition.id, 1);
    await setStatus(admin, competition.id, "active");
    await openCompetition(page, competition.id);

    await page.getByRole("button", { name: `Lane ${entry.lane} stage 1: Edit` }).click();
    const counts = page.locator(".counts");
    await expect(counts).toContainText(`must sum to ${course.stageOneShots} shots`);

    await counts.getByLabel("10", { exact: true }).fill("5");
    await counts.getByLabel("9", { exact: true }).fill("4");
    await expect(counts.locator(".total")).toContainText("3 to go");

    await counts.getByLabel("8", { exact: true }).fill("5");
    await expect(counts.locator(".total")).toContainText("2 too many");

    await counts.getByRole("button", { name: "Save counts" }).click();
    await expect(section(page, "Entries and progress").getByRole("alert")).toHaveText("invalid input");
    await expect(counts).toBeVisible();

    await counts.getByLabel("8", { exact: true }).fill("3");
    await expect(counts.locator(".total")).toContainText("Shots match");
  });

  test("save and lock records the stage as locked by the signed-in user and unlock reopens it", async ({ course, adminPage: page, admin }) => {
    const competition = await createCompetition(admin);
    const entry = await seedEntry(admin, competition.id, 1);
    await setStatus(admin, competition.id, "active");
    await openCompetition(page, competition.id);

    const entryRow = row(page, entry.shooter.last_name);
    const stage = entryRow.locator("td.stage").first();
    await page.getByRole("button", { name: `Lane ${entry.lane} stage 1: Edit` }).click();
    const counts = page.locator(".counts");
    await counts.getByLabel("X", { exact: true }).fill("2");
    await counts.getByLabel("10", { exact: true }).fill("4");
    await counts.getByLabel("9", { exact: true }).fill("3");
    await counts.getByLabel("0", { exact: true }).fill("1");
    await counts.getByLabel("miss", { exact: true }).fill("2");
    await expect(counts.locator(".total")).toContainText("Shots match");
    await counts.getByRole("button", { name: "Save and lock" }).click();

    await expect(counts).toHaveCount(0);
    const locked = stage.getByText("Accepted", { exact: true });
    await expect(locked).toBeVisible();
    await expect(locked).toHaveAttribute("title", new RegExp(`Locked by ${adminUsername}`));
    await expect(stage).toContainText("pts");

    await page.getByRole("button", { name: `Lane ${entry.lane} stage 1: Unlock` }).click();
    await expect(stage.getByText("Scored, not locked")).toBeVisible();
    await expect(page.getByRole("button", { name: `Lane ${entry.lane} stage 1: Lock` })).toBeVisible();
    await expect(stage.getByText("Accepted", { exact: true })).toHaveCount(0);
  });

  test("an admin overwrite replaces the counts of a stage that already has them", async ({ course, adminPage: page, admin }) => {
    const competition = await createCompetition(admin);
    const entry = await seedEntry(admin, competition.id, 1);
    await setStatus(admin, competition.id, "active");
    await setCounts(admin, competition.id, entry.entryId, 1, { "10": course.stageOneShots }, true);
    await openCompetition(page, competition.id);

    const stage = row(page, entry.shooter.last_name).locator("td.stage").first();
    await expect(stage.locator(".score strong")).toHaveText("120");
    await page.getByRole("button", { name: `Lane ${entry.lane} stage 1: Edit` }).click();
    const counts = page.locator(".counts");
    await counts.getByLabel("9", { exact: true }).fill(String(course.stageOneShots));
    await counts.getByRole("button", { name: "Save and lock" }).click();

    await expect(counts).toHaveCount(0);
    await expect(stage.locator(".score strong")).toHaveText("108");
    await expect(stage.getByText("Accepted", { exact: true })).toHaveAttribute("title", new RegExp(`Locked by ${adminUsername}`));
  });

  test("a shooter imported from WA1500 in the picker is selected for the entry", async ({ course, adminPage: page, admin }) => {
    const competition = await createCompetition(admin);
    const lastName = `Imported${Date.now()}`;
    const waid = `W${Date.now()}`;
    await page.route("**/api/wa1500/search*", (route) =>
      route.fulfill({ json: { results: [{ waid, first_name: "Ida", last_name: lastName, country: "SWE", club: "Remote Club" }] } }),
    );
    await openCompetition(page, competition.id);

    const form = addForm(page);
    await form.getByRole("combobox", { name: "Shooter" }).fill(lastName);
    await expect(page.getByText("No match")).toBeVisible();
    await page.getByRole("option", { name: "Search WA1500 / add new" }).click();
    await expect(page.getByRole("textbox", { name: "Search WA1500" })).toHaveValue(lastName);
    await page.getByRole("button", { name: "Search", exact: true }).click();
    await page.getByLabel(`Sex for Ida ${lastName}`).selectOption("F");
    await page.getByRole("button", { name: `Import Ida ${lastName}` }).click();

    await expect(form.getByRole("combobox", { name: "Shooter" })).toHaveValue(`Ida ${lastName}`);
    await form.getByRole("button", { name: "Add entry" }).click();
    await expect(row(page, lastName)).toBeVisible();
  });

  test("category dropdowns list only the categories the course allows", async ({ courses, adminPage: page, admin }) => {
    for (const key of ["40", "150"]) {
      const expected = courses.find((c) => c.key === key)?.categories;
      expect(expected).toBeDefined();
      const competition = await createCompetition(admin, { course: key });
      const entry = await seedEntry(admin, competition.id, 1, expected![0] as Category);
      await openCompetition(page, competition.id);

      await expect(addForm(page).getByLabel("Entry category").getByRole("option")).toHaveText(expected!);
      const entryRow = row(page, entry.shooter.last_name);
      await entryRow.getByRole("button", { name: "Move", exact: true }).click();
      await expect(entryRow.getByLabel("Move category").getByRole("option")).toHaveText(expected!);
    }
  });
});
