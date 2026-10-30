import { readFile } from "node:fs/promises";
import type { APIRequestContext } from "@playwright/test";
import {
  createMatch,
  expect,
  newApi,
  openAccountLink,
  openMatch,
  seedEntry,
  setCounts,
  setStatus,
  test,
} from "./helpers";

async function seedReportMatch(api: APIRequestContext) {
  const match = await createMatch(api);
  const { shooter, entryId } = await seedEntry(api, match.id, 1);
  await setCounts(api, match.id, entryId, 1, { x: 14, "10": 10 }, true);
  await setStatus(api, match.id, "active");
  await setStatus(api, match.id, "finished");
  return { match, shooter };
}

test.describe("reports and backup", () => {
  test("the results report downloads as an xlsx workbook", async ({ adminPage: page, admin }) => {
    const { match } = await seedReportMatch(admin);
    await page.goto("/admin");

    const downloadPromise = page.waitForEvent("download");
    await page.getByRole("link", { name: "Results report (.xlsx) for " + match.name }).click();
    const download = await downloadPromise;

    expect(download.suggestedFilename()).toMatch(/\.xlsx$/);
    const bytes = await readFile(await download.path());
    expect(bytes.subarray(0, 2).toString("latin1")).toBe("PK");
  });

  test("the detailed CSV downloads and contains the shooter", async ({
    adminPage: page,
    admin,
  }) => {
    const { match, shooter } = await seedReportMatch(admin);
    await page.goto("/admin");

    const downloadPromise = page.waitForEvent("download");
    await page.getByRole("link", { name: "Detailed CSV for " + match.name }).click();
    const download = await downloadPromise;

    expect(download.suggestedFilename()).toMatch(/\.csv$/);
    const text = await readFile(await download.path(), "utf8");
    expect(text).toContain(shooter.last_name);
  });

  test("the admin results PDF downloads", async ({ adminPage: page, admin }) => {
    const { match } = await seedReportMatch(admin);
    await page.goto("/admin");

    const downloadPromise = page.waitForEvent("download");
    await page.getByRole("link", { name: "Results PDF for " + match.name }).click();
    const download = await downloadPromise;

    expect(download.suggestedFilename()).toMatch(/-results\.pdf$/);
    const bytes = await readFile(await download.path());
    expect(bytes.subarray(0, 5).toString("latin1")).toBe("%PDF-");
  });

  test("the competition downloads a zip of every finished match", async ({
    adminPage: page,
    admin,
  }) => {
    const { match } = await seedReportMatch(admin);
    await page.goto("/admin");
    const group = page.getByRole("region", { name: `Competition ${match.name}` });

    for (const [name, entry] of [
      ["Results reports (.xlsx) zip", `${match.name}`],
      ["All formats zip", ""],
    ] as const) {
      const downloadPromise = page.waitForEvent("download");
      await group.getByRole("link", { name: `${name} for ${match.name}` }).click();
      const download = await downloadPromise;
      expect(download.suggestedFilename()).toMatch(/\.zip$/);
      const bytes = await readFile(await download.path());
      expect(bytes.subarray(0, 2).toString("latin1")).toBe("PK");
      if (entry) expect(bytes.toString("latin1")).toContain(".xlsx");
      else expect(bytes.toString("latin1")).toMatch(/\.pdf/);
    }
  });

  test("the combined 1500 workbook button warns about unfinished 1500 matches", async ({
    adminPage: page,
    admin,
  }) => {
    const { match } = await seedReportMatch(admin);
    const group = page.getByRole("region", { name: `Competition ${match.name}` });
    const link = group.getByRole("link", {
      name: `Combined 1500 workbook (.xlsx) for ${match.name}`,
    });

    const { matches } = (await (await admin.get("/api/matches")).json()) as {
      matches: { id: number; competition_id: number }[];
    };
    const competitionId = matches.find((m) => m.id === match.id)!.competition_id;
    const openName = `${match.name} open`;
    const created = await admin.post("/api/matches", {
      data: { name: openName, date: match.date, course: "150", competition_id: competitionId },
    });
    expect(created.ok()).toBe(true);

    await page.goto("/admin");
    await expect(group.getByRole("note")).toContainText(`unfinished 1500 matches: ${openName}`);
    await expect(
      group.getByRole("link", { name: `Results reports (.xlsx) zip for ${match.name}` }),
    ).toBeVisible();

    const downloadPromise = page.waitForEvent("download");
    await link.click();
    const download = await downloadPromise;
    expect(download.suggestedFilename()).toMatch(/-1500\.xlsx$/);
    const bytes = await readFile(await download.path());
    expect(bytes.subarray(0, 2).toString("latin1")).toBe("PK");
  });

  test("the combined 1500 workbook button is hidden without a finished 1500 match", async ({
    adminPage: page,
    admin,
  }) => {
    const match = await createMatch(admin);
    await page.goto("/admin");
    const group = page.getByRole("region", { name: `Competition ${match.name}` });
    await expect(group).toBeVisible();
    await expect(group.getByRole("link", { name: /^Combined 1500 workbook/ })).toHaveCount(0);
  });

  test("the match page offers no downloads", async ({ adminPage: page, admin }) => {
    const { match } = await seedReportMatch(admin);
    await openMatch(page, match.id);
    await expect(page.getByRole("heading", { name: match.name })).toBeVisible();
    await expect(
      page.getByRole("link", { name: /Results report|Detailed CSV|Results PDF/ }),
    ).toHaveCount(0);
  });

  test("report and backup endpoints return 401 without a session", async ({ admin, baseURL }) => {
    const { match } = await seedReportMatch(admin);
    const api = await newApi(baseURL!);
    for (const path of [
      `/api/matches/${match.id}/results.xlsx`,
      `/api/matches/${match.id}/results.csv`,
      `/api/competitions/1/results.zip?format=csv`,
      `/api/admin/matches/${match.id}/results`,
      "/api/admin/backup",
    ]) {
      expect((await api.get(path)).status(), path).toBe(401);
    }
    await api.dispose();
  });

  test("a database backup downloads and is not empty", async ({ adminPage: page }) => {
    const downloadPromise = page.waitForEvent("download");
    await openAccountLink(page, "Backup");
    const download = await downloadPromise;

    const bytes = await readFile(await download.path());
    expect(bytes.length).toBeGreaterThan(0);
  });
});
