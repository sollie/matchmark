import { readFile } from "node:fs/promises";
import { expect, test } from "@playwright/test";
import {
  adminApi,
  createCompetition,
  expectVisible,
  newApi,
  seedEntry,
  setCounts,
  signInViaUi,
  uniqueClientIp,
} from "./helpers";

async function seedReportCompetition(baseURL: string) {
  const api = await adminApi(baseURL);
  const competition = await createCompetition(api);
  const { shooter, entryId } = await seedEntry(api, competition.id, 1);
  await setCounts(api, competition.id, entryId, 1, { x: 7, "10": 5 }, true);
  await api.dispose();
  return { competition, shooter };
}

test.describe("reports and backup", () => {
  test.beforeEach(async ({ page }) => {
    await page.setExtraHTTPHeaders({ "X-Forwarded-For": uniqueClientIp() });
  });

  test("the results report downloads as an xlsx workbook", async ({ page, baseURL }) => {
    const { competition } = await seedReportCompetition(baseURL!);
    await signInViaUi(page);
    await expect(page.getByRole("heading", { name: "Competitions" })).toBeVisible();
    await page.goto(`/admin/competitions/${competition.id}`);
    await expectVisible(page.getByRole("heading", { name: competition.name }));

    const downloadPromise = page.waitForEvent("download");
    await page.getByRole("link", { name: "Results report (.xlsx)" }).click();
    const download = await downloadPromise;

    expect(download.suggestedFilename()).toMatch(/\.xlsx$/);
    const bytes = await readFile(await download.path());
    expect(bytes.subarray(0, 2).toString("latin1")).toBe("PK");
  });

  test("the detailed CSV downloads and contains the shooter", async ({ page, baseURL }) => {
    const { competition, shooter } = await seedReportCompetition(baseURL!);
    await signInViaUi(page);
    await expect(page.getByRole("heading", { name: "Competitions" })).toBeVisible();
    await page.goto(`/admin/competitions/${competition.id}`);
    await expectVisible(page.getByRole("heading", { name: competition.name }));

    const downloadPromise = page.waitForEvent("download");
    await page.getByRole("link", { name: "Detailed CSV" }).click();
    const download = await downloadPromise;

    expect(download.suggestedFilename()).toMatch(/\.csv$/);
    const text = await readFile(await download.path(), "utf8");
    expect(text).toContain(shooter.last_name);
  });

  test("report and backup endpoints return 401 without a session", async ({ baseURL }) => {
    const { competition } = await seedReportCompetition(baseURL!);
    const api = await newApi(baseURL!);
    for (const path of [
      `/api/competitions/${competition.id}/results.xlsx`,
      `/api/competitions/${competition.id}/results.csv`,
      "/api/admin/backup",
    ]) {
      expect((await api.get(path)).status(), path).toBe(401);
    }
    await api.dispose();
  });

  test("a database backup downloads and is not empty", async ({ page }) => {
    await signInViaUi(page);
    await expect(page.getByRole("heading", { name: "Competitions" })).toBeVisible();

    const downloadPromise = page.waitForEvent("download");
    await page.getByRole("link", { name: "Backup" }).click();
    const download = await downloadPromise;

    const bytes = await readFile(await download.path());
    expect(bytes.length).toBeGreaterThan(0);
  });
});
