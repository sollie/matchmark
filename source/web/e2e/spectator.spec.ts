import { expect, test, type Page } from "@playwright/test";
import {
  adminApi,
  addEntry,
  createCompetition,
  createShooter,
  newApi,
  row,
  setCounts,
  setStatus,
  setVisibility,
  uniqueName,
} from "./helpers";

const refresh = { timeout: 20_000 };

async function openCompetition(page: Page, name: string) {
  await page.goto("/");
  const option = page.locator("select option").filter({ hasText: name });
  await expect(option).toHaveCount(1, refresh);
  const value = await option.getAttribute("value");
  await page.getByLabel("Competition").selectOption(value!);
}

async function seedScored(finish = false) {
  const api = await adminApi(test.info().project.use.baseURL!);
  const competition = await createCompetition(api, { name: uniqueName("Spectator"), status: "active" });
  await setVisibility(api, competition.id, {});
  const club = uniqueName("Club");
  const entries = [] as { lane: number; last_name: string; first_name: string; club: string; country: string }[];
  const plans = [
    { lane: 1, counts: { x: 2, "10": 6, "9": 4 }, lock: true },
    { lane: 2, counts: { x: 6, "10": 6 }, lock: true },
    { lane: 3, counts: { "9": 12 }, lock: false },
    { lane: 4, counts: { x: 5, "10": 7 }, lock: false },
  ];
  const shooters = [];
  for (const plan of plans) {
    const shooter = await createShooter(api, { club: `${club} L${plan.lane}`, country: plan.lane % 2 ? "SWE" : "DEN" });
    shooters.push(shooter);
  }
  for (const [i, plan] of plans.entries()) {
    const entryId = await addEntry(api, competition.id, shooters[i].id, plan.lane);
    await setCounts(api, competition.id, entryId, 1, plan.counts, plan.lock);
    entries.push({ lane: plan.lane, ...shooters[i] });
  }
  if (finish) await setStatus(api, competition.id, "finished");
  return { api, competition, entries };
}

test.describe("spectator", () => {
  test("standings are ranked by total then X count with stage progress", async ({ page }) => {
    const { api, competition, entries } = await seedScored();
    await openCompetition(page, competition.name);

    const table = page.getByRole("table");
    const byLane = (lane: number) => row(page, entries[lane - 1].last_name);
    await expect(byLane(2)).toBeVisible(refresh);
    await expect(byLane(2).getByRole("cell").first()).toHaveText("1");
    await expect(byLane(4).getByRole("cell").first()).toHaveText("1");
    await expect(byLane(1).getByRole("cell").first()).toHaveText("3");
    await expect(byLane(3).getByRole("cell").first()).toHaveText("4");
    await expect(byLane(1).getByRole("cell").nth(1)).toHaveText("1");
    await expect(table.getByRole("columnheader", { name: "S1", exact: true })).toBeVisible();
    await expect(table.getByRole("columnheader", { name: "X", exact: true })).toBeVisible();

    const rows = table.locator("tbody tr");
    await expect(rows.first()).toContainText(entries[1].last_name);
    await expect(rows.last()).toContainText(entries[2].last_name);

    const progress = page.locator(".card").filter({ has: page.getByRole("heading", { name: "Stage progress" }) });
    await expect(progress.getByText("2/4 locked")).toBeVisible(refresh);
    await api.dispose();
  });

  test("draft competitions are hidden and finished ones keep results", async ({ page }) => {
    const { api, competition: active, entries } = await seedScored(true);
    const draft = await createCompetition(api, { name: uniqueName("Hidden draft") });

    await page.goto("/");
    await expect(page.locator("select option").filter({ hasText: active.name })).toHaveCount(1, refresh);
    await expect(page.locator("select option").filter({ hasText: draft.name })).toHaveCount(0);

    await openCompetition(page, active.name);
    await expect(page.locator("select option").filter({ hasText: active.name })).toContainText("finished");
    await expect(row(page, entries[0].last_name)).toBeVisible(refresh);
    await expect(row(page, entries[1].last_name)).toBeVisible();
    await api.dispose();
  });

  test("each visibility flag hides its fields", async ({ page }) => {
    const { api, competition, entries } = await seedScored();
    const shooter = entries[1];
    const table = page.getByRole("table");

    await openCompetition(page, competition.name);
    await expect(row(page, shooter.last_name)).toContainText(shooter.club, refresh);
    await expect(row(page, shooter.last_name)).toContainText(shooter.country);

    await setVisibility(api, competition.id, { show_clubs: false });
    await page.reload();
    await openCompetition(page, competition.name);
    await expect(row(page, shooter.last_name)).toBeVisible(refresh);
    await expect(table.getByRole("columnheader", { name: "Club", exact: true })).toHaveCount(0);
    await expect(table).not.toContainText(shooter.club);
    await expect(table.getByRole("columnheader", { name: "Country" })).toBeVisible();

    await setVisibility(api, competition.id, { show_countries: false });
    await openCompetition(page, competition.name);
    await expect(table.locator("tbody tr")).toHaveCount(4, refresh);
    await expect(table.getByRole("columnheader", { name: "Country" })).toHaveCount(0, refresh);

    await setVisibility(api, competition.id, { show_stage_scores: false });
    await openCompetition(page, competition.name);
    await expect(table.locator("tbody tr")).toHaveCount(4, refresh);
    await expect(table.getByRole("columnheader", { name: "S1", exact: true })).toHaveCount(0, refresh);
    await expect(table.getByRole("columnheader", { name: "Total" })).toBeVisible();

    await setVisibility(api, competition.id, { show_stage_scores: false, show_ring_counts: false });
    await openCompetition(page, competition.name);
    await expect(table.locator("tbody tr")).toHaveCount(4, refresh);
    await expect(table.getByRole("columnheader", { name: "X", exact: true })).toHaveCount(0, refresh);

    await setVisibility(api, competition.id, { show_names: false, show_stage_scores: false, show_ring_counts: false });
    await openCompetition(page, competition.name);
    await expect(table.locator("tbody tr")).toHaveCount(4, refresh);
    await expect(table.getByRole("columnheader", { name: "Name" })).toHaveCount(0, refresh);
    await expect(table).not.toContainText(shooter.last_name);
    await api.dispose();
  });

  test("a competition without results shows the empty state", async ({ page }) => {
    const api = await adminApi(test.info().project.use.baseURL!);
    const competition = await createCompetition(api, { name: uniqueName("Empty"), status: "active" });
    await openCompetition(page, competition.name);
    await expect(page.getByText("No results yet.")).toBeVisible(refresh);
    await expect(page.getByRole("table")).toHaveCount(0);
    await api.dispose();
  });

  test("the standings API returns 404 for a draft competition", async ({ baseURL }) => {
    const admin = await adminApi(baseURL!);
    const draft = await createCompetition(admin, { name: uniqueName("API draft") });
    const anonymous = await newApi(baseURL!);
    const response = await anonymous.get(`/api/spectator/competitions/${draft.id}/standings`);
    expect(response.status()).toBe(404);

    await setStatus(admin, draft.id, "active");
    expect((await anonymous.get(`/api/spectator/competitions/${draft.id}/standings`)).status()).toBe(200);
    await anonymous.dispose();
    await admin.dispose();
  });
});
