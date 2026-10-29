import {
  addEntry,
  createMatch,
  createShooter,
  expect,
  row,
  setCounts,
  setStatus,
  setVisibility,
  test,
  uniqueName,
} from "./helpers";

test.describe("results", () => {
  test("a finished match shows ranked results per category and links from the spectator page", async ({
    page,
    admin,
  }) => {
    const match = await createMatch(admin, { name: uniqueName("Results") });
    const known = await createShooter(admin, {
      waid: String(Math.floor(Math.random() * 9e6) + 1e6),
    });
    const fresh = await createShooter(admin);
    const first = await addEntry(admin, match.id, known.id, 1);
    const second = await addEntry(admin, match.id, fresh.id, 2);
    await setVisibility(admin, match.id, {});
    await setStatus(admin, match.id, "active");
    await setCounts(admin, match.id, first, 1, { x: 4, "10": 20 }, true);
    await setCounts(admin, match.id, second, 1, { x: 24 }, true);
    await setStatus(admin, match.id, "finished");

    await page.goto("/");
    await page.getByLabel("Match").selectOption(String(match.id));
    await page.getByRole("link", { name: "View results" }).click();
    await expect(page).toHaveURL(`/results/${match.id}`);

    await expect(page.getByRole("heading", { name: match.name })).toBeVisible();
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

  test("an unfinished match has no results", async ({ page, admin }) => {
    const match = await createMatch(admin, { name: uniqueName("Unfinished"), status: "active" });
    await page.goto(`/results/${match.id}`);
    await expect(page.getByText("Results are not available for this match.")).toBeVisible();
  });
});
