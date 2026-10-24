import { readFile } from "node:fs/promises";
import type { APIRequestContext } from "@playwright/test";
import { createCompetition, expect, newApi, openCompetition, seedEntry, setCounts, setStatus, test } from "./helpers";

async function seedReportCompetition(api: APIRequestContext) {
  const competition = await createCompetition(api);
  const { shooter, entryId } = await seedEntry(api, competition.id, 1);
  await setCounts(api, competition.id, entryId, 1, { x: 7, "10": 5 }, true);
  await setStatus(api, competition.id, "active");
  await setStatus(api, competition.id, "finished");
  return { competition, shooter };
}

test.describe("reports and backup", () => {
  test("the results report downloads as an xlsx workbook", async ({ adminPage: page, admin }) => {
    const { competition } = await seedReportCompetition(admin);
    await page.goto("/admin");

    const downloadPromise = page.waitForEvent("download");
    await page.getByRole("link", { name: "Results report (.xlsx) for " + competition.name }).click();
    const download = await downloadPromise;

    expect(download.suggestedFilename()).toMatch(/\.xlsx$/);
    const bytes = await readFile(await download.path());
    expect(bytes.subarray(0, 2).toString("latin1")).toBe("PK");
  });

  test("the detailed CSV downloads and contains the shooter", async ({ adminPage: page, admin }) => {
    const { competition, shooter } = await seedReportCompetition(admin);
    await page.goto("/admin");

    const downloadPromise = page.waitForEvent("download");
    await page.getByRole("link", { name: "Detailed CSV for " + competition.name }).click();
    const download = await downloadPromise;

    expect(download.suggestedFilename()).toMatch(/\.csv$/);
    const text = await readFile(await download.path(), "utf8");
    expect(text).toContain(shooter.last_name);
  });

  test("the admin results PDF downloads", async ({ adminPage: page, admin }) => {
    const { competition } = await seedReportCompetition(admin);
    await page.goto("/admin");

    const downloadPromise = page.waitForEvent("download");
    await page.getByRole("link", { name: "Results PDF for " + competition.name }).click();
    const download = await downloadPromise;

    expect(download.suggestedFilename()).toMatch(/-results\.pdf$/);
    const bytes = await readFile(await download.path());
    expect(bytes.subarray(0, 5).toString("latin1")).toBe("%PDF-");
  });

  test("the competition page offers no downloads", async ({ adminPage: page, admin }) => {
    const { competition } = await seedReportCompetition(admin);
    await openCompetition(page, competition.id);
    await expect(page.getByRole("heading", { name: competition.name })).toBeVisible();
    await expect(page.getByRole("link", { name: /Results report|Detailed CSV|Results PDF/ })).toHaveCount(0);
  });

  test("report and backup endpoints return 401 without a session", async ({ admin, baseURL }) => {
    const { competition } = await seedReportCompetition(admin);
    const api = await newApi(baseURL!);
    for (const path of [
      `/api/competitions/${competition.id}/results.xlsx`,
      `/api/competitions/${competition.id}/results.csv`,
      `/api/admin/competitions/${competition.id}/results`,
      "/api/admin/backup",
    ]) {
      expect((await api.get(path)).status(), path).toBe(401);
    }
    await api.dispose();
  });

  test("a database backup downloads and is not empty", async ({ adminPage: page }) => {
    const downloadPromise = page.waitForEvent("download");
    await page.getByRole("link", { name: "Backup" }).click();
    const download = await downloadPromise;

    const bytes = await readFile(await download.path());
    expect(bytes.length).toBeGreaterThan(0);
  });
});
