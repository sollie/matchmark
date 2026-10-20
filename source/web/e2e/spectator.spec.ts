import type { APIRequestContext, Page } from "@playwright/test";
import {
  addEntry,
  createCompetition,
  createShooter,
  expect,
  newApi,
  row,
  setCounts,
  setStatus,
  setVisibility,
  test,
  uniqueName,
} from "./helpers";

async function openCompetition(page: Page, name: string) {
  await page.goto("/");
  const option = page.locator("select option").filter({ hasText: name });
  await expect(option).toHaveCount(1);
  const value = await option.getAttribute("value");
  await page.getByLabel("Competition").selectOption(value!);
}

async function seedScored(api: APIRequestContext, finish = false) {
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
  return { competition, entries };
}

test.describe("spectator", () => {
  test("standings are ranked by total then X count with stage progress", async ({ page, admin }) => {
    const { competition, entries } = await seedScored(admin);
    await openCompetition(page, competition.name);

    const table = page.getByRole("table");
    const byLane = (lane: number) => row(page, entries[lane - 1].last_name);
    await expect(byLane(2)).toBeVisible();
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
    await expect(progress.getByText("2/4 locked")).toBeVisible();
  });

  test("a locked stage appears live without polling requests", async ({ page, admin }) => {
    const { competition, entries } = await seedScored(admin);
    const polled: string[] = [];
    page.on("request", (request) => {
      const path = new URL(request.url()).pathname;
      if (/\/spectator\/competitions\/\d+\/(standings|progress)$/.test(path)) polled.push(path);
    });
    await openCompetition(page, competition.name);
    const progress = page.locator(".card").filter({ has: page.getByRole("heading", { name: "Stage progress" }) });
    await expect(progress.getByText("2/4 locked")).toBeVisible();

    const entryId = (
      await (await admin.get(`/api/competitions/${competition.id}/entries`)).json()
    ).entries.find((e: { lane: number }) => e.lane === 3).id;
    await setCounts(admin, competition.id, entryId, 1, { "9": 12 }, true);

    await expect(progress.getByText("3/4 locked")).toBeVisible();
    await expect(row(page, entries[2].last_name)).toBeVisible();
    expect(polled).toEqual([]);
  });

  test("a hidden tab closes the stream and a visible one reconnects with a fresh snapshot", async ({ page, admin }) => {
    const { competition, entries } = await seedScored(admin);
    let streams = 0;
    page.on("request", (request) => {
      if (/\/spectator\/competitions\/\d+\/stream$/.test(new URL(request.url()).pathname)) streams++;
    });
    await openCompetition(page, competition.name);
    const progress = page.locator(".card").filter({ has: page.getByRole("heading", { name: "Stage progress" }) });
    await expect(progress.getByText("2/4 locked")).toBeVisible();
    await expect.poll(() => streams).toBe(1);

    const setHidden = (hidden: boolean) =>
      page.evaluate((value) => {
        Object.defineProperty(document, "hidden", { configurable: true, get: () => value });
        document.dispatchEvent(new Event("visibilitychange"));
      }, hidden);

    await setHidden(true);
    const entryId = (
      await (await admin.get(`/api/competitions/${competition.id}/entries`)).json()
    ).entries.find((e: { lane: number }) => e.lane === 3).id;
    await setCounts(admin, competition.id, entryId, 1, { "9": 12 }, true);
    await page.waitForTimeout(1000);
    await expect(progress.getByText("2/4 locked")).toBeVisible();
    expect(streams).toBe(1);

    await setHidden(false);
    await expect(progress.getByText("3/4 locked")).toBeVisible();
    await expect(row(page, entries[2].last_name)).toBeVisible();
    expect(streams).toBe(2);
  });

  test("draft competitions are hidden and finished ones keep results", async ({ page, admin }) => {
    const { competition: active, entries } = await seedScored(admin, true);
    const draft = await createCompetition(admin, { name: uniqueName("Hidden draft") });

    await page.goto("/");
    await expect(page.locator("select option").filter({ hasText: active.name })).toHaveCount(1);
    await expect(page.locator("select option").filter({ hasText: draft.name })).toHaveCount(0);

    await openCompetition(page, active.name);
    await expect(page.locator("select option").filter({ hasText: active.name })).toContainText("finished");
    await expect(row(page, entries[0].last_name)).toBeVisible();
    await expect(row(page, entries[1].last_name)).toBeVisible();
  });

  test("each visibility flag hides its fields and values", async ({ course, page, admin }) => {
    const { competition, entries } = await seedScored(admin);
    const shooter = entries[1];
    const winnerCells = (shown: { names?: boolean; clubs?: boolean; countries?: boolean; stage?: boolean; x?: boolean }) =>
      [
        "1",
        "2",
        ...(shown.names ? [`${shooter.first_name} ${shooter.last_name}`] : []),
        ...(shown.clubs ? [shooter.club] : []),
        ...(shown.countries ? [shooter.country] : []),
        "Revolver",
        ...(shown.stage ? ["120", ...Array<string>(course.stages - 1).fill("0")] : []),
        "120",
        ...(shown.x ? ["6"] : []),
      ];
    const table = page.getByRole("table");
    const winner = table.locator("tbody tr").first();

    await openCompetition(page, competition.name);
    await expect(table.locator("tbody tr")).toHaveCount(4);
    await expect(winner.getByRole("cell")).toHaveText(
      winnerCells({ names: true, clubs: true, countries: true, stage: true, x: true }),
    );

    await setVisibility(admin, competition.id, { show_clubs: false });
    await openCompetition(page, competition.name);
    await expect(table.getByRole("columnheader", { name: "Club", exact: true })).toHaveCount(0);
    await expect(winner.getByRole("cell")).toHaveText(
      winnerCells({ names: true, countries: true, stage: true, x: true }),
    );
    await expect(table).not.toContainText(shooter.club);

    await setVisibility(admin, competition.id, { show_clubs: false, show_countries: false });
    await openCompetition(page, competition.name);
    await expect(table.getByRole("columnheader", { name: "Country" })).toHaveCount(0);
    await expect(winner.getByRole("cell")).toHaveText(winnerCells({ names: true, stage: true, x: true }));
    await expect(table).not.toContainText(shooter.country);

    await setVisibility(admin, competition.id, { show_clubs: false, show_countries: false, show_stage_scores: false });
    await openCompetition(page, competition.name);
    await expect(table.getByRole("columnheader", { name: "S1", exact: true })).toHaveCount(0);
    await expect(winner.getByRole("cell")).toHaveText(winnerCells({ names: true, x: true }));

    await setVisibility(admin, competition.id, { show_clubs: false, show_countries: false, show_stage_scores: false, show_ring_counts: false });
    await openCompetition(page, competition.name);
    await expect(table.getByRole("columnheader", { name: "X", exact: true })).toHaveCount(0);
    await expect(winner.getByRole("cell")).toHaveText(winnerCells({ names: true }));

    await setVisibility(admin, competition.id, { show_clubs: false, show_countries: false, show_names: false, show_stage_scores: false, show_ring_counts: false });
    await openCompetition(page, competition.name);
    await expect(table.getByRole("columnheader", { name: "Name" })).toHaveCount(0);
    await expect(winner.getByRole("cell")).toHaveText(winnerCells({}));
    await expect(table).not.toContainText(shooter.last_name);
    await expect(table).not.toContainText(shooter.first_name);
  });

  test("a competition without results shows the empty state", async ({ page, admin }) => {
    const competition = await createCompetition(admin, { name: uniqueName("Empty"), status: "active" });
    await openCompetition(page, competition.name);
    await expect(page.getByText("No results yet.")).toBeVisible();
    await expect(page.getByRole("table")).toHaveCount(0);
  });

  test("the standings API returns 404 for a draft competition", async ({ admin, baseURL }) => {
    const draft = await createCompetition(admin, { name: uniqueName("API draft") });
    const anonymous = await newApi(baseURL!);
    const response = await anonymous.get(`/api/spectator/competitions/${draft.id}/standings`);
    expect(response.status()).toBe(404);

    await setStatus(admin, draft.id, "active");
    expect((await anonymous.get(`/api/spectator/competitions/${draft.id}/standings`)).status()).toBe(200);
    await anonymous.dispose();
  });
});
