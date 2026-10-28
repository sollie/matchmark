import type { APIRequestContext, Page } from "@playwright/test";
import {
  addEntry,
  createMatch,
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

async function openMatch(page: Page, name: string) {
  await page.goto("/");
  const option = page.locator("select option").filter({ hasText: name });
  await expect(option).toHaveCount(1);
  const value = await option.getAttribute("value");
  await page.getByLabel("Match").selectOption(value!);
}

async function seedScored(api: APIRequestContext, finish = false) {
  const match = await createMatch(api, { name: uniqueName("Spectator") });
  await setVisibility(api, match.id, {});
  const club = uniqueName("Club");
  const entries = [] as {
    lane: number;
    last_name: string;
    first_name: string;
    club: string;
    country: string;
  }[];
  const plans = [
    { lane: 1, counts: { x: 2, "10": 6, "9": 4 }, lock: true },
    { lane: 2, counts: { x: 6, "10": 6 }, lock: true },
    { lane: 3, counts: { "9": 12 }, lock: false },
    { lane: 4, counts: { x: 5, "10": 7 }, lock: false },
  ];
  const shooters = [];
  for (const plan of plans) {
    const shooter = await createShooter(api, {
      club: `${club} L${plan.lane}`,
      country: plan.lane % 2 ? "SWE" : "DEN",
    });
    shooters.push(shooter);
  }
  const entryIds = [] as number[];
  for (const [i, plan] of plans.entries()) {
    entryIds.push(await addEntry(api, match.id, shooters[i].id, plan.lane));
    entries.push({ lane: plan.lane, ...shooters[i] });
  }
  await setStatus(api, match.id, "active");
  for (const [i, plan] of plans.entries()) {
    await setCounts(api, match.id, entryIds[i], 1, plan.counts, plan.lock);
  }
  if (finish) await setStatus(api, match.id, "finished");
  return { match, entries };
}

test.describe("spectator", () => {
  test("standings are ranked by total then X count with the stage status row", async ({
    page,
    admin,
  }) => {
    const { match, entries } = await seedScored(admin);
    await openMatch(page, match.name);

    const table = page.getByRole("table");
    const byLane = (lane: number) => row(page, entries[lane - 1].last_name);
    await expect(byLane(2)).toBeVisible();
    await expect(byLane(2).getByRole("cell").first()).toHaveText("1");
    await expect(byLane(4).getByRole("cell").first()).toHaveText("3");
    await expect(byLane(1).getByRole("cell").first()).toHaveText("2");
    await expect(byLane(3).getByRole("cell").first()).toHaveText("3");
    await expect(byLane(1).getByRole("cell").nth(1)).toHaveText("1");
    await expect(table.getByRole("columnheader", { name: "S1", exact: true })).toBeVisible();
    await expect(table.getByRole("columnheader", { name: "X", exact: true })).toBeVisible();

    const rows = table.locator("tbody tr");
    await expect(rows.first()).toContainText(entries[1].last_name);

    const progress = page.getByRole("status");
    await expect(progress.getByText("2/4 accepted")).toBeVisible();
    await expect(progress.getByText("Stage 1 of 10")).toBeVisible();
    await expect(page.locator("progress")).toHaveCount(0);
  });

  test("an accepted stage appears live without polling requests", async ({ page, admin }) => {
    const { match, entries } = await seedScored(admin);
    const polled: string[] = [];
    page.on("request", (request) => {
      const path = new URL(request.url()).pathname;
      if (/\/spectator\/matches\/\d+\/standings$/.test(path)) polled.push(path);
    });
    await openMatch(page, match.name);
    const progress = page.getByRole("status");
    await expect(progress.getByText("2/4 accepted")).toBeVisible();

    const entryId = (
      await (await admin.get(`/api/matches/${match.id}/entries`)).json()
    ).entries.find((e: { lane: number }) => e.lane === 3).id;
    await setCounts(admin, match.id, entryId, 1, { "9": 12 }, true);

    await expect(progress.getByText("3/4 accepted")).toBeVisible();
    await expect(row(page, entries[2].last_name)).toBeVisible();
    expect(polled).toEqual([]);
  });

  test("a hidden tab closes the stream and a visible one reconnects with a fresh snapshot", async ({
    page,
    admin,
  }) => {
    const { match, entries } = await seedScored(admin);
    let streams = 0;
    page.on("request", (request) => {
      if (/\/spectator\/matches\/\d+\/stream$/.test(new URL(request.url()).pathname)) streams++;
    });
    await openMatch(page, match.name);
    const progress = page.getByRole("status");
    await expect(progress.getByText("2/4 accepted")).toBeVisible();
    await expect.poll(() => streams).toBe(1);

    const setHidden = (hidden: boolean) =>
      page.evaluate((value) => {
        Object.defineProperty(document, "hidden", { configurable: true, get: () => value });
        document.dispatchEvent(new Event("visibilitychange"));
      }, hidden);

    await setHidden(true);
    const entryId = (
      await (await admin.get(`/api/matches/${match.id}/entries`)).json()
    ).entries.find((e: { lane: number }) => e.lane === 3).id;
    await setCounts(admin, match.id, entryId, 1, { "9": 12 }, true);
    await page.waitForTimeout(1000);
    await expect(progress.getByText("2/4 accepted")).toBeVisible();
    expect(streams).toBe(1);

    await setHidden(false);
    await expect(progress.getByText("3/4 accepted")).toBeVisible();
    await expect(row(page, entries[2].last_name)).toBeVisible();
    expect(streams).toBe(2);
  });

  test("draft matches are hidden and finished ones keep results", async ({ page, admin }) => {
    const { match: active, entries } = await seedScored(admin, true);
    const draft = await createMatch(admin, { name: uniqueName("Hidden draft") });

    await page.goto("/");
    await expect(page.locator("select option").filter({ hasText: active.name })).toHaveCount(1);
    await expect(page.locator("select option").filter({ hasText: draft.name })).toHaveCount(0);

    await openMatch(page, active.name);
    await expect(page.locator("select option").filter({ hasText: active.name })).toContainText(
      "finished",
    );
    await expect(page.getByRole("status").getByText("Finished", { exact: true })).toBeVisible();
    await expect(page.getByRole("status").getByText(/Stage \d+ of/)).toHaveCount(0);
    await expect(row(page, entries[0].last_name)).toBeVisible();
    await expect(row(page, entries[1].last_name)).toBeVisible();
  });

  test("stage cells show state as color and label with a legend", async ({ page, admin }) => {
    const { match, entries } = await seedScored(admin);
    await openMatch(page, match.name);

    const accepted = row(page, entries[1].last_name);
    await expect(accepted.getByLabel("Stage 1: accepted, 120", { exact: true })).toHaveText("120");
    const inProgress = row(page, entries[2].last_name);
    await expect(inProgress.getByLabel("Stage 1: in progress", { exact: true })).toHaveText("");
    await expect(inProgress.getByLabel("Stage 2: not started", { exact: true })).toHaveText("");
    await expect(inProgress.getByTitle("Stage 2: not started", { exact: true })).toBeVisible();

    const legend = page.getByRole("list", { name: "Stage state legend" });
    for (const state of ["Not started", "In progress", "Awaiting acceptance", "Accepted"]) {
      await expect(legend).toContainText(state);
    }
  });

  test("each visibility flag hides its fields and values", async ({ course, page, admin }) => {
    const { match, entries } = await seedScored(admin);
    const shooter = entries[1];
    const winnerCells = (shown: {
      names?: boolean;
      clubs?: boolean;
      countries?: boolean;
      stage?: boolean;
      x?: boolean;
    }) => [
      "1",
      "2",
      ...(shown.names ? [`${shooter.first_name} ${shooter.last_name}`] : []),
      ...(shown.clubs ? [shooter.club] : []),
      ...(shown.countries ? [shooter.country] : []),
      "Revolver",
      shown.stage ? "120" : "",
      ...Array<string>(course.stages - 1).fill(""),
      "120",
      ...(shown.x ? ["6"] : []),
    ];
    const table = page.getByRole("table");
    const winner = table.locator("tbody tr").first();

    await openMatch(page, match.name);
    await expect(table.locator("tbody tr")).toHaveCount(4);
    await expect(winner.getByRole("cell")).toHaveText(
      winnerCells({ names: true, clubs: true, countries: true, stage: true, x: true }),
    );

    await setVisibility(admin, match.id, { show_clubs: false });
    await openMatch(page, match.name);
    await expect(table.getByRole("columnheader", { name: "Club", exact: true })).toHaveCount(0);
    await expect(winner.getByRole("cell")).toHaveText(
      winnerCells({ names: true, countries: true, stage: true, x: true }),
    );
    await expect(table).not.toContainText(shooter.club);

    await setVisibility(admin, match.id, { show_clubs: false, show_countries: false });
    await openMatch(page, match.name);
    await expect(table.getByRole("columnheader", { name: "Country" })).toHaveCount(0);
    await expect(winner.getByRole("cell")).toHaveText(
      winnerCells({ names: true, stage: true, x: true }),
    );
    await expect(table).not.toContainText(shooter.country);

    await setVisibility(admin, match.id, {
      show_clubs: false,
      show_countries: false,
      show_stage_scores: false,
    });
    await openMatch(page, match.name);
    await expect(table.getByRole("columnheader", { name: "S1", exact: true })).toBeVisible();
    await expect(winner.getByRole("cell")).toHaveText(winnerCells({ names: true, x: true }));
    await expect(winner.getByLabel("Stage 1: accepted", { exact: true })).toBeVisible();

    await setVisibility(admin, match.id, {
      show_clubs: false,
      show_countries: false,
      show_stage_scores: false,
      show_ring_counts: false,
    });
    await openMatch(page, match.name);
    await expect(table.getByRole("columnheader", { name: "X", exact: true })).toHaveCount(0);
    await expect(winner.getByRole("cell")).toHaveText(winnerCells({ names: true }));

    await setVisibility(admin, match.id, {
      show_clubs: false,
      show_countries: false,
      show_names: false,
      show_stage_scores: false,
      show_ring_counts: false,
    });
    await openMatch(page, match.name);
    await expect(table.getByRole("columnheader", { name: "Name" })).toHaveCount(0);
    await expect(winner.getByRole("cell")).toHaveText(winnerCells({}));
    await expect(table).not.toContainText(shooter.last_name);
    await expect(table).not.toContainText(shooter.first_name);
  });

  test("a match without results shows the empty state", async ({ page, admin }) => {
    const match = await createMatch(admin, { name: uniqueName("Empty"), status: "active" });
    await openMatch(page, match.name);
    await expect(page.getByText("No results yet.")).toBeVisible();
    await expect(page.getByRole("table")).toHaveCount(0);
  });

  test("the standings API returns 404 for a draft match", async ({ admin, baseURL }) => {
    const draft = await createMatch(admin, { name: uniqueName("API draft") });
    const anonymous = await newApi(baseURL!);
    const response = await anonymous.get(`/api/spectator/matches/${draft.id}/standings`);
    expect(response.status()).toBe(404);

    await setStatus(admin, draft.id, "active");
    expect((await anonymous.get(`/api/spectator/matches/${draft.id}/standings`)).status()).toBe(
      200,
    );
    await anonymous.dispose();
  });
});

test("the footer links back to the repository on every page", async ({ page }) => {
  for (const path of ["/", "/score", "/admin"]) {
    await page.goto(path);
    const link = page.getByRole("link", { name: "Source code on GitHub" });
    await expect(link).toBeVisible();
    await expect(link).toHaveAttribute("href", "https://github.com/sollie/matchmark");
  }
});

test("the header sign-in icon leads to the admin area on public pages only", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("link", { name: "Admin", exact: true })).toHaveCount(0);
  await page.getByRole("link", { name: "Sign in" }).click();
  await expect(page).toHaveURL(/\/admin$/);
  await expect(page.getByRole("heading", { name: "Admin sign in" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Sign in" })).toHaveCount(0);

  await page.goto("/score");
  await expect(page.getByRole("link", { name: "Sign in" })).toHaveCount(0);
});

test("the header does not overflow at 320px", async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 640 });
  await page.goto("/");
  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth > document.documentElement.clientWidth,
  );
  expect(overflow).toBe(false);
  const box = await page.getByRole("link", { name: "Sign in" }).boundingBox();
  expect(box!.width).toBeGreaterThanOrEqual(44);
  expect(box!.height).toBeGreaterThanOrEqual(44);
});
