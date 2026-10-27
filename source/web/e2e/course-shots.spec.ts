import {
  createMatch,
  expect,
  openMatch,
  row,
  section,
  seedEntry,
  setStatus,
  test,
} from "./helpers";

test.describe("course shot counts", () => {
  test("every stage of every course demands its expected shots", async ({
    courses,
    adminPage: page,
    admin,
  }) => {
    test.setTimeout(240_000);
    expect(courses).toHaveLength(4);
    for (const course of courses) {
      await test.step(`${course.key}-shot course`, async () => {
        const match = await createMatch(admin, { course: course.key });
        const entry = await seedEntry(admin, match.id, 1);
        await setStatus(admin, match.id, "active");
        await openMatch(page, match.id);
        const entryRow = row(page, entry.shooter.last_name);
        const alert = section(page, "Entries and progress").getByRole("alert");

        for (const [index, shots] of course.stageShots.entries()) {
          const stageNumber = index + 1;
          const stage = entryRow.locator("td.stage").nth(index);
          await page
            .getByRole("button", { name: `Lane ${entry.lane} stage ${stageNumber}: Edit` })
            .click();
          const counts = page.locator(".counts");
          await expect(counts).toContainText(`must sum to ${shots} shots`);

          await counts.getByLabel("10", { exact: true }).fill(String(shots - 1));
          await expect(counts.locator(".total")).toContainText("1 to go");
          await counts.getByRole("button", { name: "Save and lock" }).click();
          await expect(alert).toHaveText("invalid input");
          await expect(counts).toBeVisible();
          await expect(stage.getByText("Accepted", { exact: true })).toHaveCount(0);

          await counts.getByLabel("10", { exact: true }).fill(String(shots + 1));
          await expect(counts.locator(".total")).toContainText("1 too many");
          await counts.getByRole("button", { name: "Save and lock" }).click();
          await expect(alert).toHaveText("invalid input");
          await expect(stage.getByText("Accepted", { exact: true })).toHaveCount(0);

          await counts.getByLabel("10", { exact: true }).fill(String(shots));
          await expect(counts.locator(".total")).toContainText("Shots match");
          await counts.getByRole("button", { name: "Save and lock" }).click();
          await expect(counts).toHaveCount(0);
          await expect(stage.getByText("Accepted", { exact: true })).toBeVisible();
        }
      });
    }
  });
});
