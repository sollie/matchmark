import { addEntry, createCompetition, createShooter, expect, row, setCounts, setStatus, setVisibility, test, uniqueName } from "./helpers";

test.describe("results", () => {
  test("a finished competition shows ranked results per category and links from the spectator page", async ({ page, admin }) => {
    const competition = await createCompetition(admin, { name: uniqueName("Results") });
    const known = await createShooter(admin, { waid: String(Math.floor(Math.random() * 9e6) + 1e6) });
    const fresh = await createShooter(admin);
    const first = await addEntry(admin, competition.id, known.id, 1);
    const second = await addEntry(admin, competition.id, fresh.id, 2);
    await setVisibility(admin, competition.id, {});
    await setStatus(admin, competition.id, "active");
    await setCounts(admin, competition.id, first, 1, { x: 2, "10": 10 }, true);
    await setCounts(admin, competition.id, second, 1, { x: 12 }, true);
    await setStatus(admin, competition.id, "finished");

    await page.goto("/");
    await page.getByLabel("Competition").selectOption(String(competition.id));
    await page.getByRole("link", { name: "View results" }).click();
    await expect(page).toHaveURL(`/results/${competition.id}`);

    await expect(page.getByRole("heading", { name: competition.name })).toBeVisible();
    await expect(page.getByRole("heading", { name: "Revolver" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "New shooters" })).toBeVisible();
    await expect(row(page, known.last_name).first().getByRole("cell").first()).toHaveText("2");
    await expect(row(page, fresh.last_name).first().getByRole("cell").first()).toHaveText("1");
    await expect(page.getByRole("cell", { name: known.waid })).toBeVisible();

    const downloadPromise = page.waitForEvent("download");
    await page.getByRole("link", { name: "Download PDF" }).click();
    const download = await downloadPromise;
    expect(download.suggestedFilename()).toMatch(/-results\.pdf$/);
  });

  test("an unfinished competition has no results", async ({ page, admin }) => {
    const competition = await createCompetition(admin, { name: uniqueName("Unfinished"), status: "active" });
    await page.goto(`/results/${competition.id}`);
    await expect(page.getByText("Results are not available for this competition.")).toBeVisible();
  });
});
