import type { Page } from "@playwright/test";
import {
  adminUsername,
  type Category,
  createMatch,
  createShooter,
  expect,
  openMatch,
  row,
  section,
  seedEntry,
  setCounts,
  setStatus,
  test,
} from "./helpers";

function addForm(page: Page) {
  return page.getByRole("form", { name: "Add entry" });
}

test.describe("entries", () => {
  test("adding an entry uses the chosen category and the default lane", async ({
    course,
    adminPage: page,
    admin,
  }) => {
    const match = await createMatch(admin);
    await seedEntry(admin, match.id, 1);
    const shooter = await createShooter(admin);
    await openMatch(page, match.id);

    const form = addForm(page);
    await expect(page.getByLabel("Entry lane")).toHaveValue("2");
    await form.getByRole("combobox", { name: "Shooter" }).fill(shooter.last_name);
    await page.getByRole("option", { name: new RegExp(shooter.last_name) }).click();
    await form.getByLabel("Entry category").selectOption("Pistol");
    await form.getByRole("button", { name: "Add entry" }).click();

    const added = row(page, shooter.last_name);
    await expect(added).toBeVisible();
    await expect(added.locator("td.lane")).toHaveText("2");
    await expect(added.locator("td.category")).toHaveText("Pistol");
  });

  test("a taken lane is rejected and an entered shooter is not offered again", async ({
    course,
    adminPage: page,
    admin,
  }) => {
    const match = await createMatch(admin);
    const taken = await seedEntry(admin, match.id, 1);
    const other = await createShooter(admin);
    await openMatch(page, match.id);

    const form = addForm(page);
    const picker = form.getByRole("combobox", { name: "Shooter" });
    await picker.fill(taken.shooter.last_name);
    await expect(page.getByText("No match")).toBeVisible();

    await picker.fill(other.last_name);
    await page.getByRole("option", { name: new RegExp(other.last_name) }).click();
    await page.getByLabel("Entry lane").fill("1");
    await form.getByRole("button", { name: "Add entry" }).click();

    await expect(section(page, "Entries and progress").getByRole("alert")).toHaveText("conflict");
    await expect(row(page, other.last_name)).toHaveCount(0);
    await expect(picker).toHaveValue(`${other.first_name} ${other.last_name}`);
  });

  test("an entry can be moved, withdrawn and removed", async ({
    course,
    adminPage: page,
    admin,
  }) => {
    const match = await createMatch(admin);
    const entry = await seedEntry(admin, match.id, 1);
    await openMatch(page, match.id);
    page.on("dialog", (dialog) => dialog.accept());

    const entryRow = row(page, entry.shooter.last_name);
    await entryRow.getByRole("button", { name: "Move", exact: true }).click();
    await entryRow.getByLabel("Move lane").fill("5");
    await entryRow.getByLabel("Move category").selectOption("Pistol OS");
    await entryRow.getByRole("button", { name: "Save", exact: true }).click();
    await expect(entryRow.locator("td.lane")).toHaveText("5");
    await expect(entryRow.locator("td.category")).toHaveText("Pistol OS");

    await entryRow.getByRole("button", { name: "Withdraw" }).click();
    await expect(entryRow.getByText("Withdrawn", { exact: true })).toBeVisible();
    await expect(entryRow.getByRole("button", { name: "Withdraw" })).toHaveCount(0);

    await entryRow.getByRole("button", { name: "Remove", exact: true }).click();
    await expect(row(page, entry.shooter.last_name)).toHaveCount(0);
  });
});

test.describe("entries layout", () => {
  test("every segment is reachable without horizontal scrolling", async ({
    course,
    adminPage: page,
    admin,
  }) => {
    const match = await createMatch(admin);
    const entry = await seedEntry(admin, match.id, 1);
    await page.setViewportSize({ width: 1024, height: 768 });
    await openMatch(page, match.id);

    const segments = row(page, entry.shooter.last_name).locator("td.segment");
    await expect(segments).toHaveCount(course.segments);
    const scroll = page.locator(".scroll", { has: page.locator("table.entries") });
    expect(await scroll.evaluate((el) => el.scrollWidth <= el.clientWidth)).toBe(true);
    await expect(segments.last()).toBeInViewport();
  });
});

test.describe("admin ring counts", () => {
  test("a live shot total flags a wrong total and the save is rejected", async ({
    course,
    adminPage: page,
    admin,
  }) => {
    const match = await createMatch(admin);
    const entry = await seedEntry(admin, match.id, 1);
    await setStatus(admin, match.id, "active");
    await openMatch(page, match.id);

    await page.getByRole("button", { name: `Lane ${entry.lane} segment 1: Edit` }).click();
    const counts = page.locator(".counts");
    await expect(counts).toContainText(`must sum to ${course.segmentOneShots} shots`);

    await counts.getByLabel("10", { exact: true }).fill("5");
    await counts.getByLabel("9", { exact: true }).fill("4");
    await expect(counts.locator(".total")).toContainText("3 to go");

    await counts.getByLabel("8", { exact: true }).fill("5");
    await expect(counts.locator(".total")).toContainText("2 too many");

    await counts.getByRole("button", { name: "Save counts" }).click();
    await expect(section(page, "Entries and progress").getByRole("alert")).toHaveText(
      "invalid input",
    );
    await expect(counts).toBeVisible();

    await counts.getByLabel("8", { exact: true }).fill("3");
    await expect(counts.locator(".total")).toContainText("Shots match");
  });

  test("save and lock records the segment as locked by the signed-in user and unlock reopens it", async ({
    course,
    adminPage: page,
    admin,
  }) => {
    const match = await createMatch(admin);
    const entry = await seedEntry(admin, match.id, 1);
    await setStatus(admin, match.id, "active");
    await openMatch(page, match.id);

    const entryRow = row(page, entry.shooter.last_name);
    const segment = entryRow.locator("td.segment").first();
    await page.getByRole("button", { name: `Lane ${entry.lane} segment 1: Edit` }).click();
    const counts = page.locator(".counts");
    await counts.getByLabel("X", { exact: true }).fill("2");
    await counts.getByLabel("10", { exact: true }).fill("4");
    await counts.getByLabel("9", { exact: true }).fill("3");
    await counts.getByLabel("0", { exact: true }).fill("1");
    await counts.getByLabel("miss", { exact: true }).fill("2");
    await expect(counts.locator(".total")).toContainText("Shots match");
    await counts.getByRole("button", { name: "Save and lock" }).click();

    await expect(counts).toHaveCount(0);
    const locked = segment.getByText("Accepted", { exact: true });
    await expect(locked).toBeVisible();
    await expect(locked).toHaveAttribute("title", new RegExp(`Locked by ${adminUsername}`));
    await expect(segment).toContainText("pts");

    await page.getByRole("button", { name: `Lane ${entry.lane} segment 1: Unlock` }).click();
    await expect(segment.getByText("Scored, not locked")).toBeVisible();
    await expect(
      page.getByRole("button", { name: `Lane ${entry.lane} segment 1: Lock` }),
    ).toBeVisible();
    await expect(segment.getByText("Accepted", { exact: true })).toHaveCount(0);
  });

  test("an admin overwrite replaces the counts of a segment that already has them", async ({
    course,
    adminPage: page,
    admin,
  }) => {
    const match = await createMatch(admin);
    const entry = await seedEntry(admin, match.id, 1);
    await setStatus(admin, match.id, "active");
    await setCounts(admin, match.id, entry.entryId, 1, { "10": course.segmentOneShots }, true);
    await openMatch(page, match.id);

    const segment = row(page, entry.shooter.last_name).locator("td.segment").first();
    await expect(segment.locator(".score strong")).toHaveText("120");
    await page.getByRole("button", { name: `Lane ${entry.lane} segment 1: Edit` }).click();
    const counts = page.locator(".counts");
    await counts.getByLabel("9", { exact: true }).fill(String(course.segmentOneShots));
    await counts.getByRole("button", { name: "Save and lock" }).click();

    await expect(counts).toHaveCount(0);
    await expect(segment.locator(".score strong")).toHaveText("108");
    await expect(segment.getByText("Accepted", { exact: true })).toHaveAttribute(
      "title",
      new RegExp(`Locked by ${adminUsername}`),
    );
  });

  test("a shooter imported from WA1500 in the picker is selected for the entry", async ({
    course,
    adminPage: page,
    admin,
  }) => {
    const match = await createMatch(admin);
    const lastName = `IMPORTED${Date.now()}`;
    const waid = `W${Date.now()}`;
    await page.route("**/api/wa1500/search*", (route) =>
      route.fulfill({
        json: {
          results: [
            { waid, first_name: "Ida", last_name: lastName, country: "SWE", club: "Remote Club" },
          ],
        },
      }),
    );
    await openMatch(page, match.id);

    const form = addForm(page);
    await form.getByRole("combobox", { name: "Shooter" }).fill(lastName);
    await expect(page.getByText("No match")).toBeVisible();
    await page.getByRole("option", { name: "Search WA1500 / add new" }).click();
    await expect(page.getByRole("textbox", { name: "Search WA1500" })).toHaveValue(lastName);
    await page.getByRole("button", { name: `Import Ida ${lastName}` }).click();

    await expect(form.getByRole("combobox", { name: "Shooter" })).toHaveValue(`Ida ${lastName}`);
    await form.getByRole("button", { name: "Add entry" }).click();
    await expect(row(page, lastName)).toBeVisible();
  });

  test("category dropdowns list only the categories the course allows", async ({
    courses,
    adminPage: page,
    admin,
  }) => {
    for (const key of ["40", "150"]) {
      const categories = courses.find((c) => c.key === key)?.categories;
      expect(categories).toBeDefined();
      const expected = [...categories!].sort((a, b) => a.localeCompare(b));
      const match = await createMatch(admin, { course: key });
      const entry = await seedEntry(admin, match.id, 1, expected[0] as Category);
      await openMatch(page, match.id);

      await expect(addForm(page).getByLabel("Entry category").getByRole("option")).toHaveText(
        expected,
      );
      const entryRow = row(page, entry.shooter.last_name);
      await entryRow.getByRole("button", { name: "Move", exact: true }).click();
      await expect(entryRow.getByLabel("Move category").getByRole("option")).toHaveText(expected);
    }
  });

  test("a Carbine entry can be added on the 150-shot course", async ({
    adminPage: page,
    admin,
  }) => {
    const match = await createMatch(admin, { course: "150" });
    const entry = await seedEntry(admin, match.id, 1, "Carbine");
    await openMatch(page, match.id);
    await expect(row(page, entry.shooter.last_name).locator("td.category")).toHaveText("Carbine");
  });

  test("an archived shooter is refused with an inline unarchive", async ({
    course,
    adminPage: page,
    admin,
  }) => {
    const match = await createMatch(admin);
    const shooter = await createShooter(admin);
    await openMatch(page, match.id);

    const form = addForm(page);
    await form.getByRole("combobox", { name: "Shooter" }).fill(shooter.last_name);
    await page.getByRole("option", { name: new RegExp(shooter.last_name) }).click();
    await admin.post(`/api/shooters/${shooter.id}/archive`);
    await form.getByRole("button", { name: "Add entry" }).click();

    await expect(page.getByText("This shooter is archived")).toBeVisible();
    await page.getByRole("button", { name: "Unarchive" }).click();
    await expect(page.getByText("This shooter is archived")).toHaveCount(0);
    await form.getByRole("button", { name: "Add entry" }).click();
    await expect(row(page, shooter.last_name)).toBeVisible();
  });
});
