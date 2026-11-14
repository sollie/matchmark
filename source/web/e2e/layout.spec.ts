import {
  addEntry,
  createMatch,
  createPairing,
  createShooter,
  expect,
  openAccountLink,
  row,
  seedEntry,
  setCounts,
  setStatus,
  test,
  uniqueLastName,
  uniqueName,
} from "./helpers";

test.describe("layout", () => {
  test("the users table stacks without horizontal overflow at 375px", async ({
    adminPage: page,
  }) => {
    await page.setViewportSize({ width: 375, height: 800 });
    await openAccountLink(page, "Users");
    await expect(page.getByRole("heading", { name: "Users" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Reset password" }).first()).toBeVisible();
    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    );
    expect(overflow).toBeLessThanOrEqual(0);
  });

  test("the matches table stacks without horizontal overflow at 375px", async ({
    adminPage: page,
    admin,
    courses,
  }) => {
    const finished = await createMatch(admin, {
      name: uniqueName("Stacked finished"),
      date: "2026-03-03",
      course: courses[0].key,
      status: "finished",
    });
    await page.setViewportSize({ width: 375, height: 800 });
    await page.goto("/admin");
    await expect(page.getByRole("link", { name: finished.name, exact: true })).toBeVisible();
    for (const name of ["Results report (.xlsx)", "Detailed CSV", "Results PDF"]) {
      await expect(page.getByRole("link", { name: `${name} for ${finished.name}` })).toBeVisible();
    }
    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    );
    expect(overflow).toBeLessThanOrEqual(0);
  });

  test("the matches table is not scrolled sideways on a wide viewport", async ({
    adminPage: page,
    admin,
    courses,
  }) => {
    const finished = await createMatch(admin, {
      name: uniqueName("Wide finished"),
      date: "2026-03-03",
      course: courses[0].key,
      status: "finished",
    });
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto("/admin");
    const link = page.getByRole("link", { name: finished.name, exact: true });
    await expect(link).toBeVisible();
    const scroller = page.getByRole("region", { name: /^Matches in / }).filter({ has: link });
    const sideways = await scroller.evaluate((el) => el.scrollWidth - el.clientWidth);
    expect(sideways).toBeLessThanOrEqual(0);
  });

  test("the shooters table stacks without horizontal overflow at 375px", async ({
    adminPage: page,
    admin,
    courses,
  }) => {
    const shooter = await createShooter(admin, { last_name: uniqueLastName("Stacked") });
    const match = await createMatch(admin, {
      name: uniqueName("Stacked entry match"),
      date: "2026-03-03",
      course: courses[0].key,
    });
    await addEntry(admin, match.id, shooter.id, 1);
    await page.setViewportSize({ width: 375, height: 800 });
    await page.goto("/admin/shooters");
    const r = row(page, shooter.last_name);
    await r.getByRole("button", { name: /^Entered in 1 match/ }).click();
    await expect(r.getByRole("link", { name: match.name })).toBeVisible();
    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    );
    expect(overflow).toBeLessThanOrEqual(0);
  });

  test("the remaining admin tables stack without horizontal overflow at 375px", async ({
    adminPage: page,
    admin,
    courses,
  }) => {
    const course = courses[0];
    const match = await createMatch(admin, {
      name: uniqueName("Stacked audit"),
      course: course.key,
    });
    const { entryId } = await seedEntry(admin, match.id, 1);
    await seedEntry(admin, match.id, 2);
    await setStatus(admin, match.id, "active");
    await createPairing(admin, match.id, 1);
    await setCounts(admin, match.id, entryId, 1, { "10": course.segmentOneShots }, true);
    await page.setViewportSize({ width: 375, height: 800 });
    for (const path of [
      "/admin/audit",
      "/admin/screens",
      `/admin/matches/${match.id}`,
      `/admin/matches/${match.id}/control`,
      `/results/${match.id}`,
    ]) {
      await page.goto(path);
      await expect(page.getByRole("heading").first()).toBeVisible();
      const overflow = await page.evaluate(
        () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
      );
      expect(overflow, path).toBeLessThanOrEqual(0);
      const clipped = await page.evaluate(
        () =>
          [...document.querySelectorAll(".scroll")].filter((e) => e.scrollWidth > e.clientWidth + 1)
            .length,
      );
      expect(clipped, `${path} scroll containers`).toBe(0);
    }
  });

  test("tables keep their column layout on a wide monitor", async ({ adminPage: page }) => {
    await page.setViewportSize({ width: 1920, height: 1000 });
    await openAccountLink(page, "Users");
    await expect(page.getByRole("columnheader", { name: "Username" })).toBeVisible();
  });
});
