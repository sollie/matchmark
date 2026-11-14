import {
  addEntry,
  createMatch,
  createShooter,
  defaultCourse,
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
    const course = await defaultCourse(admin);
    const shots = course.segmentOneShots;
    const match = await createMatch(admin, { name: uniqueName("Results") });
    const known = await createShooter(admin, {
      waid: String(Math.floor(Math.random() * 9e6) + 1e6),
    });
    const fresh = await createShooter(admin);
    const first = await addEntry(admin, match.id, known.id, 1);
    const second = await addEntry(admin, match.id, fresh.id, 2);
    await setVisibility(admin, match.id, {});
    await setStatus(admin, match.id, "active");
    await setCounts(admin, match.id, first, 1, { x: 2, "10": shots - 2 }, true);
    await setCounts(admin, match.id, second, 1, { x: shots }, true);
    await setStatus(admin, match.id, "finished");

    await page.goto("/");
    await page.getByLabel("Match").selectOption(String(match.id));
    await page.getByRole("link", { name: "View results" }).click();
    await expect(page).toHaveURL(`/results/${match.id}`);

    await expect(page.getByRole("heading", { name: match.name })).toBeVisible();
    await expect(page.getByRole("heading", { name: "Revolver 1500" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "New shooters" })).toBeVisible();
    await expect(row(page, known.last_name).first().getByRole("cell").first()).toHaveText("2");
    await expect(row(page, fresh.last_name).first().getByRole("cell").first()).toHaveText("1");
    await expect(page.getByRole("cell", { name: known.waid })).toBeVisible();
    await expect(
      page.getByRole("table", { name: `Score card for ${fresh.first_name} ${fresh.last_name}` }),
    ).toBeVisible();

    const downloadPromise = page.waitForEvent("download");
    await page.getByRole("link", { name: "Download PDF" }).click();
    const download = await downloadPromise;
    expect(download.suggestedFilename()).toMatch(/-results\.pdf$/);
  });

  test("the Classification column shows the Classification and hides Unclassified", async ({
    page,
    admin,
  }) => {
    const course = await defaultCourse(admin);
    const match = await createMatch(admin, { name: uniqueName("Class") });
    const unclassify = await admin.put(`/api/matches/${match.id}`, {
      data: { name: match.name, date: match.date, course: match.course, classified: false },
    });
    expect(unclassify.ok()).toBe(true);
    const classed = await createShooter(admin);
    const unclassed = await createShooter(admin);
    const first = await addEntry(admin, match.id, classed.id, 1);
    const second = await addEntry(admin, match.id, unclassed.id, 2);
    await setVisibility(admin, match.id, {});
    await setStatus(admin, match.id, "active");
    await setCounts(admin, match.id, first, 1, { x: course.segmentOneShots }, true);
    await setCounts(admin, match.id, second, 1, { "10": course.segmentOneShots }, true);
    await setStatus(admin, match.id, "finished");

    await page.route("**/api/spectator/matches/*/results", async (route) => {
      const response = await route.fetch();
      const body = await response.json();
      const rows = body.categories[0].rows;
      rows[0].classification = { value: 4, name: "Master" };
      rows[1].classification = { value: 0, name: "Unclassified" };
      await route.fulfill({ response, json: body });
    });
    await page.goto(`/results/${match.id}`);

    await expect(page.getByRole("columnheader", { name: "Classification" })).toBeVisible();
    await expect(
      row(page, classed.last_name).first().getByRole("cell", { name: "Master" }),
    ).toBeVisible();
    await expect(row(page, unclassed.last_name).first()).not.toContainText("Unclassified");
  });

  test("a large viewport shows categories side by side without sideways scrolling", async ({
    page,
    admin,
  }) => {
    const course = await defaultCourse(admin);
    const match = await createMatch(admin, { name: uniqueName("Wide") });
    const shooter = await createShooter(admin);
    const entry = await addEntry(admin, match.id, shooter.id, 1);
    await setVisibility(admin, match.id, {});
    await setStatus(admin, match.id, "active");
    await setCounts(admin, match.id, entry, 1, { x: course.segmentOneShots }, true);
    await setStatus(admin, match.id, "finished");

    await page.route("**/api/spectator/matches/*/results", async (route) => {
      const response = await route.fetch();
      const body = await response.json();
      body.categories = [
        { ...body.categories[0], category: "Alpha class" },
        { ...body.categories[0], category: "Bravo class" },
      ];
      await route.fulfill({ response, json: body });
    });
    await page.setViewportSize({ width: 1920, height: 1080 });
    await page.goto(`/results/${match.id}`);

    const alpha = await page.getByRole("heading", { name: "Alpha class" }).boundingBox();
    const bravo = await page.getByRole("heading", { name: "Bravo class" }).boundingBox();
    expect(alpha && bravo && Math.abs(alpha.y - bravo.y) < 4 && bravo.x > alpha.x + 200).toBe(true);
    expect(
      await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
    ).toBe(true);
  });

  test("a phone shows the categories stacked", async ({ page, admin }) => {
    const match = await createMatch(admin, { name: uniqueName("Narrow") });
    const shooter = await createShooter(admin);
    await addEntry(admin, match.id, shooter.id, 1);
    await setStatus(admin, match.id, "active");
    await setStatus(admin, match.id, "finished");
    await page.setViewportSize({ width: 390, height: 800 });
    await page.goto(`/results/${match.id}`);
    await expect(page.getByRole("heading", { name: match.name })).toBeVisible();
    expect(
      await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
    ).toBe(true);
  });

  test("an unfinished match has no results", async ({ page, admin }) => {
    const match = await createMatch(admin, { name: uniqueName("Unfinished"), status: "active" });
    await page.goto(`/results/${match.id}`);
    await expect(page.getByText("Results are not available for this match.")).toBeVisible();
  });
});
