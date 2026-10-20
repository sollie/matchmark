import type { Page } from "@playwright/test";
import {
  allCourses,
  courses,
  createCompetition,
  createPairing,
  createShooter,
  expect,
  openCompetition,
  row,
  section,
  seedEntry,
  test,
  todayIso,
  uniqueName,
} from "./helpers";

test.describe("competitions", () => {
  function stageHeaders(page: Page) {
    return section(page, "Entries and progress").getByRole("columnheader", { name: /^S\d+$/ });
  }

  for (const course of allCourses) {
    test(`creating a ${course.key}-shot competition shows ${course.stages} stage columns`, async ({ adminPage: page, admin }) => {
      const name = uniqueName(`Course ${course.key}`);

      await page.getByLabel("Name").fill(name);
      await page.getByLabel("Course of fire").selectOption({ label: course.label });
      await page.getByRole("button", { name: "Create competition" }).click();

      await expect(page.getByRole("heading", { name })).toBeVisible();
      await expect(page.getByText(course.label, { exact: false }).first()).toBeVisible();
      await expect(page.getByText("draft", { exact: true })).toBeVisible();

      const id = Number(page.url().split("/").pop());
      await seedEntry(admin, id, 1);
      await openCompetition(page, id);
      await expect(stageHeaders(page)).toHaveCount(course.stages);
      await expect(stageHeaders(page).first()).toHaveText("S1");
      await expect(stageHeaders(page).last()).toHaveText(`S${course.stages}`);
    });
  }

  test("the date is prefilled in ISO 8601 and an invalid date is blocked", async ({ adminPage: page }) => {
    const name = uniqueName("Bad date");

    await expect(page.getByLabel("Date")).toHaveValue(todayIso());
    await page.getByLabel("Name").fill(name);
    await page.getByLabel("Date").fill("19/10/2026");
    await page.getByRole("button", { name: "Create competition" }).click();

    await expect(page).toHaveURL(/\/admin$/);
    await expect(page.getByLabel("Date")).toHaveJSProperty("validity.valid", false);
    await expect(row(page, name)).toHaveCount(0);

    await page.getByLabel("Date").fill("2026-10-19");
    await page.getByRole("button", { name: "Create competition" }).click();
    await expect(page.getByRole("heading", { name })).toBeVisible();
    await expect(page.getByText("2026-10-19")).toBeVisible();
  });

  test("the list shows each competition with its status", async ({ adminPage: page, admin }) => {
    const draft = await createCompetition(admin, { name: uniqueName("Listed draft"), date: "2026-03-01" });
    const active = await createCompetition(admin, {
      name: uniqueName("Listed active"),
      date: "2026-03-02",
      course: courses.sixty.key,
      status: "active",
    });
    const finished = await createCompetition(admin, {
      name: uniqueName("Listed finished"),
      date: "2026-03-03",
      course: courses.fortyEight.key,
      status: "finished",
    });

    await page.goto("/admin");
    const table = page.getByRole("table");
    const expectations = [
      { c: draft, status: "draft", course: courses.wa1500.label },
      { c: active, status: "active", course: courses.sixty.label },
      { c: finished, status: "finished", course: courses.fortyEight.label },
    ];
    for (const { c, status, course } of expectations) {
      const r = row(page, c.name);
      await expect(r).toHaveCount(1);
      await expect(r.getByRole("link", { name: c.name })).toBeVisible();
      await expect(r).toContainText(c.date);
      await expect(r).toContainText(course);
      await expect(r.getByText(status, { exact: true })).toBeVisible();
    }
    await expect(table).toBeVisible();

    await row(page, active.name).getByRole("link").click();
    await expect(page.getByRole("heading", { name: active.name })).toBeVisible();
  });

  test("start and finish ask for confirmation and declining changes nothing", async ({ adminPage: page, admin }) => {
    const competition = await createCompetition(admin);
    await openCompetition(page, competition.id);
    await expect(page.getByRole("heading", { name: competition.name })).toBeVisible();

    const messages: string[] = [];
    page.on("dialog", (dialog) => {
      messages.push(dialog.message());
      void (accept ? dialog.accept() : dialog.dismiss());
    });

    let accept = false;
    await page.getByRole("button", { name: "Start competition" }).click();
    await expect(page.getByText("draft", { exact: true })).toBeVisible();
    await expect(page.getByRole("button", { name: "Start competition" })).toBeVisible();

    accept = true;
    await page.getByRole("button", { name: "Start competition" }).click();
    await expect(page.getByText("active", { exact: true })).toBeVisible();

    accept = false;
    await page.getByRole("button", { name: "Finish competition" }).click();
    await expect(page.getByText("active", { exact: true })).toBeVisible();

    accept = true;
    await page.getByRole("button", { name: "Finish competition" }).click();
    await expect(page.getByText("finished", { exact: true })).toBeVisible();
    await expect(page.getByRole("button", { name: /Start competition|Finish competition/ })).toHaveCount(0);

    expect(messages).toEqual([
      expect.stringContaining("Start this competition?"),
      expect.stringContaining("Start this competition?"),
      expect.stringContaining("Finish this competition?"),
      expect.stringContaining("Finish this competition?"),
    ]);
  });

  test("a finished competition shows revoked pairings and refuses new entries", async ({ adminPage: page, admin }) => {
    const competition = await createCompetition(admin, { status: "active" });
    await seedEntry(admin, competition.id, 1);
    await seedEntry(admin, competition.id, 2);
    await createPairing(admin, competition.id, 1);
    const extra = await createShooter(admin);

    await openCompetition(page, competition.id);
    const pairings = section(page, "Scorer pairing");
    await expect(pairings.getByRole("row", { name: /paired/ })).toHaveCount(1);

    page.once("dialog", (dialog) => dialog.accept());
    await page.getByRole("button", { name: "Finish competition" }).click();
    await expect(page.getByText("finished", { exact: true })).toBeVisible();

    await expect(pairings.getByText("Competition finished. Pairings are revoked.")).toBeVisible();
    await page.reload();
    await expect(pairings.getByRole("row", { name: /revoked/ })).toHaveCount(1);
    await expect(pairings.getByRole("button", { name: "Revoke" })).toHaveCount(0);

    const entries = page.getByRole("form", { name: "Add entry" });
    await entries.getByLabel("Shooter").selectOption({ label: `${extra.last_name}, ${extra.first_name}` });
    await entries.getByLabel("Entry lane").fill("3");
    await entries.getByRole("button", { name: "Add entry" }).click();
    await expect(section(page, "Entries and progress").getByRole("alert")).toHaveText("conflict");
    await expect(row(page, extra.last_name)).toHaveCount(0);
  });

  test("a draft competition can be edited", async ({ adminPage: page, admin }) => {
    const competition = await createCompetition(admin, { date: "2026-03-01" });
    const name = uniqueName("Renamed");
    await openCompetition(page, competition.id);

    await page.getByRole("button", { name: "Edit competition" }).click();
    const form = page.getByRole("form", { name: "Edit competition" });
    await form.getByLabel("Name").fill(name);
    await form.getByLabel("Date").fill("2026-10-19");
    await form.getByLabel("Course of fire").selectOption({ label: courses.sixty.label });
    await form.getByRole("button", { name: "Save" }).click();

    await expect(page.getByRole("heading", { name })).toBeVisible();
    await expect(page.getByText("2026-10-19")).toBeVisible();
    await expect(page.getByText(courses.sixty.label, { exact: false }).first()).toBeVisible();

    await seedEntry(admin, competition.id, 1);
    await openCompetition(page, competition.id);
    await expect(stageHeaders(page)).toHaveCount(courses.sixty.stages);

    await page.goto("/admin");
    const listed = row(page, name);
    await expect(listed).toContainText("2026-10-19");
    await expect(listed).toContainText(courses.sixty.label);
  });

  test("an invalid date is blocked when editing", async ({ adminPage: page, admin }) => {
    const competition = await createCompetition(admin);
    await openCompetition(page, competition.id);

    await page.getByRole("button", { name: "Edit competition" }).click();
    const form = page.getByRole("form", { name: "Edit competition" });
    await form.getByLabel("Date").fill("19/10/2026");
    await form.getByRole("button", { name: "Save" }).click();
    await expect(form.getByLabel("Date")).toHaveJSProperty("validity.valid", false);
    await expect(page.getByRole("heading", { name: competition.name })).toBeVisible();
  });

  test("the course is locked once an entry exists", async ({ adminPage: page, admin }) => {
    const competition = await createCompetition(admin);
    await seedEntry(admin, competition.id, 1);
    const name = uniqueName("Entered");
    await openCompetition(page, competition.id);

    await page.getByRole("button", { name: "Edit competition" }).click();
    const form = page.getByRole("form", { name: "Edit competition" });
    await expect(form.getByLabel("Course of fire")).toBeDisabled();
    await expect(form.getByText("The course cannot change once the competition has entries.")).toBeVisible();
    await form.getByLabel("Name").fill(name);
    await form.getByRole("button", { name: "Save" }).click();
    await expect(page.getByRole("heading", { name })).toBeVisible();
    await expect(page.getByText(courses.wa1500.label, { exact: false }).first()).toBeVisible();
  });

  test("an active or finished competition has no edit action", async ({ adminPage: page, admin }) => {
    const active = await createCompetition(admin, { status: "active" });
    await openCompetition(page, active.id);
    await expect(page.getByRole("button", { name: "Delete competition" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Edit competition" })).toHaveCount(0);

    const finished = await createCompetition(admin, { status: "finished" });
    await openCompetition(page, finished.id);
    await expect(page.getByRole("button", { name: "Delete competition" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Edit competition" })).toHaveCount(0);
  });

  test("deleting a competition requires confirmation", async ({ adminPage: page, admin }) => {
    const competition = await createCompetition(admin);
    await openCompetition(page, competition.id);

    page.once("dialog", (dialog) => dialog.dismiss());
    await page.getByRole("button", { name: "Delete competition" }).click();
    await expect(page.getByRole("heading", { name: competition.name })).toBeVisible();

    page.once("dialog", (dialog) => {
      expect(dialog.message()).toContain(competition.name);
      void dialog.accept();
    });
    await page.getByRole("button", { name: "Delete competition" }).click();
    await expect(page).toHaveURL(/\/admin$/);
    await expect(page.getByRole("heading", { name: "Competitions" })).toBeVisible();
    await expect(row(page, competition.name)).toHaveCount(0);
  });

  test("visibility flags persist after saving and reloading", async ({ adminPage: page, admin }) => {
    const competition = await createCompetition(admin, { status: "active" });
    await openCompetition(page, competition.id);

    const visibility = section(page, "Spectator visibility");
    await expect(visibility.getByLabel("Clubs")).toBeChecked();
    await expect(visibility.getByLabel("Ring counts")).not.toBeChecked();
    await visibility.getByLabel("Clubs").uncheck();
    await visibility.getByLabel("Ring counts").check();
    await visibility.getByRole("button", { name: "Save" }).click();
    await expect(visibility.getByRole("status")).toHaveText("Saved");

    await page.reload();
    const reloaded = section(page, "Spectator visibility");
    await expect(reloaded.getByLabel("Names")).toBeChecked();
    await expect(reloaded.getByLabel("Clubs")).not.toBeChecked();
    await expect(reloaded.getByLabel("Countries")).toBeChecked();
    await expect(reloaded.getByLabel("Stage scores")).toBeChecked();
    await expect(reloaded.getByLabel("Ring counts")).toBeChecked();
  });
});
