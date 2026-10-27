import type { Page } from "@playwright/test";
import {
  createMatch,
  createPairing,
  createShooter,
  expect,
  openMatch,
  row,
  section,
  seedEntry,
  setStatus,
  test,
  todayIso,
  uniqueName,
} from "./helpers";

test.describe("matches", () => {
  function stageHeaders(page: Page) {
    return section(page, "Entries and progress").getByRole("columnheader", { name: /^S\d+$/ });
  }

  test("every course shows its label and one column per stage", async ({
    adminPage: page,
    admin,
    courses,
  }) => {
    expect(courses).toHaveLength(4);
    for (const course of courses) {
      await test.step(`${course.key}-shot course`, async () => {
        const name = uniqueName(`Course ${course.key}`);

        await page.goto("/admin");
        await page.getByLabel("Name").fill(name);
        await page.getByLabel("Course of fire").selectOption({ label: course.label });
        await page.getByRole("button", { name: "Create match" }).click();

        await expect(page.getByRole("heading", { name })).toBeVisible();
        await expect(page.getByText(course.label, { exact: false }).first()).toBeVisible();
        await expect(page.getByText("draft", { exact: true })).toBeVisible();

        const id = Number(page.url().split("/").pop());
        await seedEntry(admin, id, 1);
        await openMatch(page, id);
        await expect(stageHeaders(page)).toHaveCount(course.stages);
        await expect(stageHeaders(page).first()).toHaveText("S1");
        await expect(stageHeaders(page).last()).toHaveText(`S${course.stages}`);

        await page.goto("/admin");
        await expect(row(page, name)).toContainText(course.label);
      });
    }
  });

  test("the date is prefilled in ISO 8601 and an invalid date is blocked", async ({
    adminPage: page,
  }) => {
    const name = uniqueName("Bad date");
    const form = page.locator("form").filter({ has: page.getByRole("button", { name: "Create match" }) });

    await expect(form.getByLabel("Date", { exact: true })).toHaveValue(todayIso());
    await form.getByLabel("Name", { exact: true }).fill(name);
    await form.getByLabel("Date", { exact: true }).fill("19/10/2026");
    await page.getByRole("button", { name: "Create match" }).click();

    await expect(page).toHaveURL(/\/admin$/);
    await expect(form.getByLabel("Date", { exact: true })).toHaveJSProperty("validity.valid", false);
    await expect(row(page, name)).toHaveCount(0);

    await form.getByLabel("Date", { exact: true }).fill("2026-10-19");
    await page.getByRole("button", { name: "Create match" }).click();
    await expect(page.getByRole("heading", { name })).toBeVisible();
    await expect(page.getByText("2026-10-19")).toBeVisible();
  });

  test("the list shows each match with its status", async ({ courses, adminPage: page, admin }) => {
    const [first, second, third] = courses;
    const draft = await createMatch(admin, {
      name: uniqueName("Listed draft"),
      date: "2026-03-01",
      course: first.key,
    });
    const active = await createMatch(admin, {
      name: uniqueName("Listed active"),
      date: "2026-03-02",
      course: second.key,
      status: "active",
    });
    const finished = await createMatch(admin, {
      name: uniqueName("Listed finished"),
      date: "2026-03-03",
      course: third.key,
      status: "finished",
    });

    await page.goto("/admin");
    const table = page.getByRole("table").first();
    const expectations = [
      { c: draft, status: "draft", course: first.label },
      { c: active, status: "active", course: second.label },
      { c: finished, status: "finished", course: third.label },
    ];
    for (const { c, status, course } of expectations) {
      const r = row(page, c.name);
      await expect(r).toHaveCount(1);
      await expect(r.getByRole("link", { name: c.name, exact: true })).toBeVisible();
      await expect(r).toContainText(c.date);
      await expect(r).toContainText(course);
      await expect(r.getByText(status, { exact: true })).toBeVisible();
      const downloads = r.getByRole("link", {
        name: /^(Results report \(\.xlsx\)|Detailed CSV|Results PDF) for /,
      });
      await expect(downloads).toHaveCount(status === "finished" ? 3 : 0);
    }
    await expect(table).toBeVisible();

    await row(page, active.name).getByRole("link", { name: active.name, exact: true }).click();
    await expect(page.getByRole("heading", { name: active.name })).toBeVisible();
  });

  test("start and finish ask for confirmation and declining changes nothing", async ({
    adminPage: page,
    admin,
  }) => {
    const match = await createMatch(admin);
    await openMatch(page, match.id);
    await expect(page.getByRole("heading", { name: match.name })).toBeVisible();

    const messages: string[] = [];
    page.on("dialog", (dialog) => {
      messages.push(dialog.message());
      void (accept ? dialog.accept() : dialog.dismiss());
    });

    let accept = false;
    await page.getByRole("button", { name: "Start match" }).click();
    await expect(page.getByText("draft", { exact: true })).toBeVisible();
    await expect(page.getByRole("button", { name: "Start match" })).toBeVisible();

    accept = true;
    await page.getByRole("button", { name: "Start match" }).click();
    await expect(page.getByText("active", { exact: true })).toBeVisible();

    accept = false;
    await page.getByRole("button", { name: "Finish match" }).click();
    await expect(page.getByText("active", { exact: true })).toBeVisible();

    accept = true;
    await page.getByRole("button", { name: "Finish match" }).click();
    await expect(page.getByText("finished", { exact: true })).toBeVisible();
    await expect(page.getByRole("button", { name: /Start match|Finish match/ })).toHaveCount(0);

    expect(messages).toEqual([
      expect.stringContaining("Start this match?"),
      expect.stringContaining("Start this match?"),
      expect.stringContaining("Finish this match?"),
      expect.stringContaining("Finish this match?"),
    ]);
  });

  test("a finished match shows revoked pairings and refuses new entries", async ({
    adminPage: page,
    admin,
  }) => {
    const match = await createMatch(admin);
    await seedEntry(admin, match.id, 1);
    await seedEntry(admin, match.id, 2);
    await setStatus(admin, match.id, "active");
    await createPairing(admin, match.id, 1);
    const extra = await createShooter(admin);

    await openMatch(page, match.id);
    const pairings = section(page, "Scorer pairing");
    await expect(pairings.getByRole("row", { name: /paired/ })).toHaveCount(1);

    page.once("dialog", (dialog) => dialog.accept());
    await page.getByRole("button", { name: "Finish match" }).click();
    await expect(page.getByText("finished", { exact: true })).toBeVisible();

    await expect(pairings.getByText("Match finished. Pairings are revoked.")).toBeVisible();
    await page.reload();
    await expect(pairings.getByRole("row", { name: /revoked/ })).toHaveCount(1);
    await expect(pairings.getByRole("button", { name: "Revoke" })).toHaveCount(0);

    await expect(page.getByRole("form", { name: "Add entry" })).toHaveCount(0);
    const refused = await admin.post(`/api/matches/${match.id}/entries`, {
      data: { shooter_id: extra.id, category: "Revolver", lane: 3 },
    });
    expect(refused.status()).toBe(409);
    await expect(row(page, extra.last_name)).toHaveCount(0);
  });

  test("a draft match can be edited", async ({ courses, adminPage: page, admin }) => {
    const [original, target] = courses;
    const match = await createMatch(admin, { date: "2026-03-01", course: original.key });
    const name = uniqueName("Renamed");
    await openMatch(page, match.id);

    await page.getByRole("button", { name: "Edit match" }).click();
    const form = page.getByRole("form", { name: "Edit match" });
    await form.getByLabel("Name").fill(name);
    await form.getByLabel("Date").fill("2026-10-19");
    await form.getByLabel("Course of fire").selectOption({ label: target.label });
    await form.getByRole("button", { name: "Save" }).click();

    await expect(page.getByRole("heading", { name })).toBeVisible();
    await expect(page.getByText("2026-10-19")).toBeVisible();
    await expect(page.getByText(target.label, { exact: false }).first()).toBeVisible();

    await seedEntry(admin, match.id, 1);
    await openMatch(page, match.id);
    await expect(stageHeaders(page)).toHaveCount(target.stages);

    await page.goto("/admin");
    const listed = row(page, name);
    await expect(listed).toContainText("2026-10-19");
    await expect(listed).toContainText(target.label);
  });

  test("an invalid date is blocked when editing", async ({ adminPage: page, admin }) => {
    const match = await createMatch(admin);
    await openMatch(page, match.id);

    await page.getByRole("button", { name: "Edit match" }).click();
    const form = page.getByRole("form", { name: "Edit match" });
    await form.getByLabel("Date").fill("19/10/2026");
    await form.getByRole("button", { name: "Save" }).click();
    await expect(form.getByLabel("Date")).toHaveJSProperty("validity.valid", false);
    await expect(page.getByRole("heading", { name: match.name })).toBeVisible();
  });

  test("the course is locked once an entry exists", async ({ courses, adminPage: page, admin }) => {
    const [locked] = courses;
    const match = await createMatch(admin, { course: locked.key });
    await seedEntry(admin, match.id, 1);
    const name = uniqueName("Entered");
    await openMatch(page, match.id);

    await page.getByRole("button", { name: "Edit match" }).click();
    const form = page.getByRole("form", { name: "Edit match" });
    await expect(form.getByLabel("Course of fire")).toBeDisabled();
    await expect(
      form.getByText("The course cannot change once the match has entries."),
    ).toBeVisible();
    await form.getByLabel("Name").fill(name);
    await form.getByRole("button", { name: "Save" }).click();
    await expect(page.getByRole("heading", { name })).toBeVisible();
    await expect(page.getByText(locked.label, { exact: false }).first()).toBeVisible();
  });

  test("an active or finished match has no edit action", async ({ adminPage: page, admin }) => {
    const active = await createMatch(admin, { status: "active" });
    await openMatch(page, active.id);
    await expect(page.getByRole("button", { name: "Delete match" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Edit match" })).toHaveCount(0);

    const finished = await createMatch(admin, { status: "finished" });
    await openMatch(page, finished.id);
    await expect(page.getByRole("button", { name: "Delete match" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Edit match" })).toHaveCount(0);
  });

  test("deleting a match requires confirmation", async ({ adminPage: page, admin }) => {
    const match = await createMatch(admin);
    await openMatch(page, match.id);

    page.once("dialog", (dialog) => dialog.dismiss());
    await page.getByRole("button", { name: "Delete match" }).click();
    await expect(page.getByRole("heading", { name: match.name })).toBeVisible();

    page.once("dialog", (dialog) => {
      expect(dialog.message()).toContain(match.name);
      void dialog.accept();
    });
    await page.getByRole("button", { name: "Delete match" }).click();
    await expect(page).toHaveURL(/\/admin$/);
    await expect(page.getByRole("heading", { name: "Matches" })).toBeVisible();
    await expect(row(page, match.name)).toHaveCount(0);
  });

  test("visibility flags persist after saving and reloading", async ({
    adminPage: page,
    admin,
  }) => {
    const match = await createMatch(admin, { status: "active" });
    await openMatch(page, match.id);

    const visibility = section(page, "Spectator visibility");
    await expect(visibility.getByLabel("Clubs")).toBeChecked();
    await expect(visibility.getByLabel("X count")).not.toBeChecked();
    await visibility.getByLabel("Clubs").uncheck();
    await visibility.getByLabel("X count").check();
    await visibility.getByRole("button", { name: "Save" }).click();
    await expect(visibility.getByRole("status")).toHaveText("Saved");

    await page.reload();
    const reloaded = section(page, "Spectator visibility");
    await expect(reloaded.getByLabel("Names")).toBeChecked();
    await expect(reloaded.getByLabel("Clubs")).not.toBeChecked();
    await expect(reloaded.getByLabel("Countries")).toBeChecked();
    await expect(reloaded.getByLabel("Stage scores")).toBeChecked();
    await expect(reloaded.getByLabel("X count")).toBeChecked();
  });
});
