import type { Page } from "@playwright/test";
import { createCompetition, createShooter, expect, openCompetition, row, section, seedEntry, setCounts, test } from "./helpers";

function addForm(page: Page) {
  return page.getByRole("form", { name: "Add entry" });
}

test.describe("entries", () => {
  test("adding an entry uses the chosen category and the default lane", async ({ courses, adminPage: page, admin }) => {
    const competition = await createCompetition(admin, { course: courses["60"].key });
    await seedEntry(admin, competition.id, 1);
    const shooter = await createShooter(admin);
    await openCompetition(page, competition.id);

    const form = addForm(page);
    await expect(page.getByLabel("Entry lane")).toHaveValue("2");
    await form.getByLabel("Shooter").selectOption({ label: `${shooter.last_name}, ${shooter.first_name}` });
    await form.getByLabel("Entry category").selectOption("Pistol");
    await form.getByRole("button", { name: "Add entry" }).click();

    const added = row(page, shooter.last_name);
    await expect(added).toBeVisible();
    await expect(added.locator("td.lane")).toHaveText("2");
    await expect(added.locator("td.category")).toHaveText("Pistol");
  });

  test("a taken lane is rejected and an entered shooter is not offered again", async ({ courses, adminPage: page, admin }) => {
    const competition = await createCompetition(admin, { course: courses["60"].key });
    const taken = await seedEntry(admin, competition.id, 1);
    const other = await createShooter(admin);
    await openCompetition(page, competition.id);

    const form = addForm(page);
    const shooterSelect = form.getByLabel("Shooter");
    await expect(shooterSelect.locator("option", { hasText: taken.shooter.last_name })).toHaveCount(0);

    await shooterSelect.selectOption({ label: `${other.last_name}, ${other.first_name}` });
    await page.getByLabel("Entry lane").fill("1");
    await form.getByRole("button", { name: "Add entry" }).click();

    await expect(section(page, "Entries and progress").getByRole("alert")).toHaveText("conflict");
    await expect(row(page, other.last_name)).toHaveCount(0);
    await expect(shooterSelect.locator("option", { hasText: other.last_name })).toHaveCount(1);
  });

  test("an entry can be moved, withdrawn and removed", async ({ courses, adminPage: page, admin }) => {
    const competition = await createCompetition(admin, { course: courses["60"].key });
    const entry = await seedEntry(admin, competition.id, 1);
    await openCompetition(page, competition.id);
    page.on("dialog", (dialog) => dialog.accept());

    const entryRow = row(page, entry.shooter.last_name);
    await entryRow.getByRole("button", { name: "Move", exact: true }).click();
    await entryRow.getByLabel("Move lane").fill("5");
    await entryRow.getByLabel("Move category").selectOption("Rifle");
    await entryRow.getByRole("button", { name: "Save", exact: true }).click();
    await expect(entryRow.locator("td.lane")).toHaveText("5");
    await expect(entryRow.locator("td.category")).toHaveText("Rifle");

    await entryRow.getByRole("button", { name: "Withdraw" }).click();
    await expect(entryRow.getByText("Withdrawn", { exact: true })).toBeVisible();
    await expect(entryRow.getByRole("button", { name: "Withdraw" })).toHaveCount(0);

    await entryRow.getByRole("button", { name: "Remove", exact: true }).click();
    await expect(row(page, entry.shooter.last_name)).toHaveCount(0);
  });
});

test.describe("admin ring counts", () => {
  test("a live shot total flags a wrong total and the save is rejected", async ({ courses, adminPage: page, admin }) => {
    const competition = await createCompetition(admin, { course: courses["60"].key, status: "active" });
    const entry = await seedEntry(admin, competition.id, 1);
    await openCompetition(page, competition.id);

    await row(page, entry.shooter.last_name).locator("td.stage").first().getByRole("button", { name: "Stage 1: Edit" }).click();
    const counts = page.locator(".counts");
    await expect(counts).toContainText(`must sum to ${courses["60"].stageOneShots} shots`);

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

  test("save and lock records the stage as locked by Admin and unlock reopens it", async ({ courses, adminPage: page, admin }) => {
    const competition = await createCompetition(admin, { course: courses["60"].key, status: "active" });
    const entry = await seedEntry(admin, competition.id, 1);
    await openCompetition(page, competition.id);

    const entryRow = row(page, entry.shooter.last_name);
    const stage = entryRow.locator("td.stage").first();
    await stage.getByRole("button", { name: "Stage 1: Edit" }).click();
    const counts = page.locator(".counts");
    await counts.getByLabel("X", { exact: true }).fill("2");
    await counts.getByLabel("10", { exact: true }).fill("4");
    await counts.getByLabel("9", { exact: true }).fill("3");
    await counts.getByLabel("0", { exact: true }).fill("1");
    await counts.getByLabel("miss", { exact: true }).fill("2");
    await expect(counts.locator(".total")).toContainText("Shots match");
    await counts.getByRole("button", { name: "Save and lock" }).click();

    await expect(counts).toHaveCount(0);
    const locked = stage.getByText("Locked", { exact: true });
    await expect(locked).toBeVisible();
    await expect(locked).toHaveAttribute("title", /Admin/);
    await expect(stage).toContainText("pts");

    await stage.getByRole("button", { name: "Stage 1: Unlock" }).click();
    await expect(stage.getByText("Scored, not locked")).toBeVisible();
    await expect(stage.getByRole("button", { name: "Stage 1: Lock" })).toBeVisible();
    await expect(stage.getByText("Locked", { exact: true })).toHaveCount(0);
  });

  test("an admin overwrite replaces the counts of a stage that already has them", async ({ courses, adminPage: page, admin }) => {
    const competition = await createCompetition(admin, { course: courses["60"].key, status: "active" });
    const entry = await seedEntry(admin, competition.id, 1);
    await setCounts(admin, competition.id, entry.entryId, 1, { "10": courses["60"].stageOneShots }, true);
    await openCompetition(page, competition.id);

    const stage = row(page, entry.shooter.last_name).locator("td.stage").first();
    await expect(stage.locator(".score strong")).toHaveText("120");
    await stage.getByRole("button", { name: "Stage 1: Edit" }).click();
    const counts = page.locator(".counts");
    await counts.getByLabel("9", { exact: true }).fill(String(courses["60"].stageOneShots));
    await counts.getByRole("button", { name: "Save and lock" }).click();

    await expect(counts).toHaveCount(0);
    await expect(stage.locator(".score strong")).toHaveText("108");
    await expect(stage.getByText("Locked", { exact: true })).toHaveAttribute("title", /Admin/);
  });
});
