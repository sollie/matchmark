import { expect, test, type Page } from "@playwright/test";
import {
  adminApi,
  createCompetition,
  createPairing,
  createShooter,
  row,
  section,
  seedEntry,
  signInViaUi,
  todayIso,
  uniqueClientIp,
  uniqueName,
} from "./helpers";

const courses = [
  { label: "150 shots (WA1500)", key: "150", stages: 10 },
  { label: "60 shots", key: "60", stages: 4 },
  { label: "48 shots", key: "48", stages: 4 },
  { label: "40 shots (5-shot revolver)", key: "40", stages: 4 },
];

test.describe("competitions", () => {
  test.beforeEach(async ({ page }) => {
    await page.setExtraHTTPHeaders({ "X-Forwarded-For": uniqueClientIp() });
  });

  async function signedIn(page: Page, baseURL: string) {
    await signInViaUi(page);
    await expect(page.getByRole("heading", { name: "Competitions" })).toBeVisible();
    return adminApi(baseURL);
  }

  async function openCompetition(page: Page, id: number) {
    await page.goto(`/admin/competitions/${id}`);
  }

  function stageHeaders(page: Page) {
    return section(page, "Entries and progress").getByRole("columnheader", { name: /^S\d+$/ });
  }

  for (const course of courses) {
    test(`creating a ${course.key}-shot competition shows ${course.stages} stage columns`, async ({ page, baseURL }) => {
      const api = await signedIn(page, baseURL!);
      const name = uniqueName(`Course ${course.key}`);

      await page.getByLabel("Name").fill(name);
      await page.getByLabel("Course of fire").selectOption({ label: course.label });
      await page.getByRole("button", { name: "Create competition" }).click();

      await expect(page.getByRole("heading", { name })).toBeVisible();
      await expect(page.getByText(course.label, { exact: false }).first()).toBeVisible();
      await expect(page.getByText("draft", { exact: true })).toBeVisible();

      const id = Number(page.url().split("/").pop());
      await seedEntry(api, id, 1);
      await openCompetition(page, id);
      await expect(stageHeaders(page)).toHaveCount(course.stages);
      await expect(stageHeaders(page).first()).toHaveText("S1");
      await expect(stageHeaders(page).last()).toHaveText(`S${course.stages}`);
      await api.dispose();
    });
  }

  test("the date is prefilled in ISO 8601 and an invalid date is blocked", async ({ page, baseURL }) => {
    const api = await signedIn(page, baseURL!);
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
    await api.dispose();
  });

  test("the list shows each competition with its status", async ({ page, baseURL }) => {
    const api = await adminApi(baseURL!);
    const draft = await createCompetition(api, { name: uniqueName("Listed draft"), date: "2026-03-01" });
    const active = await createCompetition(api, {
      name: uniqueName("Listed active"),
      date: "2026-03-02",
      course: "60",
      status: "active",
    });
    const finished = await createCompetition(api, {
      name: uniqueName("Listed finished"),
      date: "2026-03-03",
      course: "48",
      status: "finished",
    });

    await signInViaUi(page);
    const table = page.getByRole("table");
    const expectations = [
      { c: draft, status: "draft", course: "150 shots (WA1500)" },
      { c: active, status: "active", course: "60 shots" },
      { c: finished, status: "finished", course: "48 shots" },
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
    await api.dispose();
  });

  test("start and finish ask for confirmation and declining changes nothing", async ({ page, baseURL }) => {
    const api = await adminApi(baseURL!);
    const competition = await createCompetition(api);
    await signInViaUi(page);
    await expect(page.getByRole("heading", { name: "Competitions" })).toBeVisible();
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
    await api.dispose();
  });

  test("a finished competition shows revoked pairings and refuses new entries", async ({ page, baseURL }) => {
    const api = await adminApi(baseURL!);
    const competition = await createCompetition(api, { status: "active" });
    await seedEntry(api, competition.id, 1);
    await seedEntry(api, competition.id, 2);
    await createPairing(api, competition.id, 1);
    const extra = await createShooter(api);

    await signInViaUi(page);
    await expect(page.getByRole("heading", { name: "Competitions" })).toBeVisible();
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

    const entries = section(page, "Entries and progress");
    await entries.getByLabel("Shooter").selectOption({ label: `${extra.last_name}, ${extra.first_name}` });
    await entries.getByLabel("Lane").fill("3");
    await entries.getByRole("button", { name: "Add entry" }).click();
    await expect(entries.getByRole("alert")).toBeVisible();
    await expect(row(page, extra.last_name)).toHaveCount(0);
    await api.dispose();
  });

  test("deleting a competition requires confirmation", async ({ page, baseURL }) => {
    const api = await adminApi(baseURL!);
    const competition = await createCompetition(api);
    await signInViaUi(page);
    await expect(page.getByRole("heading", { name: "Competitions" })).toBeVisible();
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
    await api.dispose();
  });

  test("visibility flags persist after saving and reloading", async ({ page, baseURL }) => {
    const api = await adminApi(baseURL!);
    const competition = await createCompetition(api, { status: "active" });
    await signInViaUi(page);
    await expect(page.getByRole("heading", { name: "Competitions" })).toBeVisible();
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
    await api.dispose();
  });
});
