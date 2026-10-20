import { expect, test, type Page } from "@playwright/test";
import {
  adminApi,
  createCompetition,
  createShooter,
  row,
  section,
  seedEntry,
  signInViaUi,
  uniqueClientIp,
} from "./helpers";

const longTimeout = 15_000;

async function openCompetition(page: Page, id: number) {
  await page.setExtraHTTPHeaders({ "X-Forwarded-For": uniqueClientIp() });
  await signInViaUi(page);
  await expect(page.getByRole("heading", { name: "Competitions" })).toBeVisible();
  await page.goto(`/admin/competitions/${id}`);
  await expect(section(page, "Entries and progress")).toBeVisible({ timeout: longTimeout });
}

function addForm(page: Page) {
  return section(page, "Entries and progress").locator("form.add-form");
}

test.describe("entries", () => {
  test("adding an entry uses the chosen category and the default lane", async ({ page, baseURL }) => {
    const api = await adminApi(baseURL!);
    const competition = await createCompetition(api, { course: "60" });
    await seedEntry(api, competition.id, 1);
    const shooter = await createShooter(api);
    await openCompetition(page, competition.id);

    const form = addForm(page);
    await expect(form.getByLabel("Lane")).toHaveValue("2");
    await form.getByLabel("Shooter").selectOption({ label: `${shooter.last_name}, ${shooter.first_name}` });
    await form.getByLabel("Category").selectOption("Pistol");
    await form.getByRole("button", { name: "Add entry" }).click();

    const added = row(page, shooter.last_name);
    await expect(added).toBeVisible({ timeout: longTimeout });
    await expect(added.locator("td.lane")).toHaveText("2");
    await expect(added.locator("td.category")).toHaveText("Pistol");
    await api.dispose();
  });

  test("a taken lane is rejected and an entered shooter is not offered again", async ({ page, baseURL }) => {
    const api = await adminApi(baseURL!);
    const competition = await createCompetition(api, { course: "60" });
    const taken = await seedEntry(api, competition.id, 1);
    const other = await createShooter(api);
    await openCompetition(page, competition.id);

    const form = addForm(page);
    const shooterSelect = form.getByLabel("Shooter");
    await expect(shooterSelect.locator("option", { hasText: taken.shooter.last_name })).toHaveCount(0);

    await shooterSelect.selectOption({ label: `${other.last_name}, ${other.first_name}` });
    await form.getByLabel("Lane").fill("1");
    await form.getByRole("button", { name: "Add entry" }).click();

    await expect(section(page, "Entries and progress").getByRole("alert")).toBeVisible();
    await expect(row(page, other.last_name)).toHaveCount(0);
    await expect(shooterSelect.locator("option", { hasText: other.last_name })).toHaveCount(1);
    await api.dispose();
  });

  test("an entry can be moved, withdrawn and removed", async ({ page, baseURL }) => {
    const api = await adminApi(baseURL!);
    const competition = await createCompetition(api, { course: "60" });
    const entry = await seedEntry(api, competition.id, 1);
    await openCompetition(page, competition.id);
    page.on("dialog", (dialog) => dialog.accept());

    const entryRow = row(page, entry.shooter.last_name);
    await entryRow.getByRole("button", { name: "Move", exact: true }).click();
    await entryRow.getByLabel("Lane").fill("5");
    await entryRow.getByLabel("Category").selectOption("Rifle");
    await entryRow.getByRole("button", { name: "Save", exact: true }).click();
    await expect(entryRow.locator("td.lane")).toHaveText("5", { timeout: longTimeout });
    await expect(entryRow.locator("td.category")).toHaveText("Rifle");

    await entryRow.getByRole("button", { name: "Withdraw" }).click();
    await expect(entryRow.getByText("Withdrawn", { exact: true })).toBeVisible({ timeout: longTimeout });
    await expect(entryRow.getByRole("button", { name: "Withdraw" })).toHaveCount(0);

    await entryRow.getByRole("button", { name: "Remove", exact: true }).click();
    await expect(row(page, entry.shooter.last_name)).toHaveCount(0, { timeout: longTimeout });
    await api.dispose();
  });
});

test.describe("admin ring counts", () => {
  test("a live shot total flags a wrong total and the save is rejected", async ({ page, baseURL }) => {
    const api = await adminApi(baseURL!);
    const competition = await createCompetition(api, { course: "60", status: "active" });
    const entry = await seedEntry(api, competition.id, 1);
    await openCompetition(page, competition.id);

    await row(page, entry.shooter.last_name).locator("td.stage").first().getByRole("button", { name: "Edit" }).click();
    const counts = page.locator(".counts");
    await expect(counts).toContainText("must sum to 12 shots");

    await counts.getByLabel("10", { exact: true }).fill("5");
    await counts.getByLabel("9", { exact: true }).fill("4");
    await expect(counts.locator(".total")).toContainText("3 to go");

    await counts.getByLabel("8", { exact: true }).fill("5");
    await expect(counts.locator(".total")).toContainText("2 too many");

    await counts.getByRole("button", { name: "Save counts" }).click();
    await expect(section(page, "Entries and progress").getByRole("alert")).toBeVisible();
    await expect(counts).toBeVisible();

    await counts.getByLabel("8", { exact: true }).fill("3");
    await expect(counts.locator(".total")).toContainText("Shots match");
    await api.dispose();
  });

  test("save and lock records the stage as locked by Admin and unlock reopens it", async ({ page, baseURL }) => {
    const api = await adminApi(baseURL!);
    const competition = await createCompetition(api, { course: "60", status: "active" });
    const entry = await seedEntry(api, competition.id, 1);
    await openCompetition(page, competition.id);

    const entryRow = row(page, entry.shooter.last_name);
    const stage = entryRow.locator("td.stage").first();
    await stage.getByRole("button", { name: "Edit" }).click();
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
    await expect(locked).toBeVisible({ timeout: longTimeout });
    await expect(locked).toHaveAttribute("title", /Admin/);
    await expect(stage).toContainText("pts");

    await stage.getByRole("button", { name: "Unlock" }).click();
    await expect(stage.getByText("Scored, not locked")).toBeVisible({ timeout: longTimeout });
    await expect(stage.getByRole("button", { name: "Lock", exact: true })).toBeVisible();
    await expect(stage.getByText("Locked", { exact: true })).toHaveCount(0);
    await api.dispose();
  });
});
