import { addEntry, createCompetition, createShooter, expect, setCounts, setStatus, setVisibility, test, uniqueName } from "./helpers";

test.describe("scorecard", () => {
  test("the spectator standings link to a lane's scorecard once finished", async ({ page, admin }) => {
    const competition = await createCompetition(admin, { name: uniqueName("Scorecard") });
    const shooter = await createShooter(admin);
    const entry = await addEntry(admin, competition.id, shooter.id, 1);
    await setVisibility(admin, competition.id, {});
    await setStatus(admin, competition.id, "active");
    await setCounts(admin, competition.id, entry, 1, { x: 2, "10": 10 }, true);

    await page.goto("/");
    await page.getByLabel("Competition").selectOption(String(competition.id));
    await expect(page.getByRole("table")).toBeVisible();
    await expect(page.getByRole("link", { name: "Scorecard for lane 1" })).toHaveCount(0);

    await setStatus(admin, competition.id, "finished");
    await page.getByRole("link", { name: "Scorecard for lane 1" }).click();
    await expect(page).toHaveURL(`/scorecard/${competition.id}/1`);
    await expect(page.getByText(`${shooter.first_name} ${shooter.last_name}`, { exact: true })).toBeVisible();
    const image = page.getByRole("img", { name: /^Scorecard for .+, lane 1$/ });
    await expect(image).toBeVisible();
    await expect.poll(() => image.evaluate((el: HTMLImageElement) => el.naturalWidth)).toBeGreaterThan(0);
    await expect(page.getByRole("link", { name: "Download PNG" })).toHaveAttribute(
      "href",
      `/api/scorer/scorecard/${competition.id}/1?format=png&download=1`,
    );
  });

  test("the scorecard fits a phone screen", async ({ page, admin }) => {
    const competition = await createCompetition(admin, { name: uniqueName("Scorecard phone") });
    const shooter = await createShooter(admin);
    await addEntry(admin, competition.id, shooter.id, 1);
    await setStatus(admin, competition.id, "active");
    await setStatus(admin, competition.id, "finished");
    await page.setViewportSize({ width: 393, height: 659 });
    await page.goto(`/scorecard/${competition.id}/1`);
    await expect(page.getByRole("img", { name: /^Scorecard for .+, lane 1$/ })).toBeVisible();
    await expect(page.getByRole("link", { name: "Download PNG" })).toBeInViewport();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  });

  test("an unfinished competition explains that the card is not available", async ({ page, admin }) => {
    const competition = await createCompetition(admin, { name: uniqueName("Scorecard unfinished"), status: "active" });
    await page.goto(`/scorecard/${competition.id}/1`);
    await expect(page.getByText("This scorecard is not available.")).toBeVisible();
    await expect(page.getByRole("link", { name: "Download PNG" })).toHaveCount(0);
  });
});
