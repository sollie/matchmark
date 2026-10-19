import { expect, test, type Browser, type Page } from "@playwright/test";
import {
  adminApi,
  createCompetition,
  createPairing,
  expectVisible,
  pairingPath,
  row,
  seedEntry,
  section,
  signInViaUi,
  uniqueClientIp,
} from "./helpers";

const poll = 20_000;

async function openDevice(browser: Browser, baseURL: string, token: string): Promise<Page> {
  const context = await browser.newContext({ baseURL });
  const page = await context.newPage();
  await page.goto(pairingPath(token));
  return page;
}

async function signInAdmin(page: Page, competitionId: number) {
  await page.setExtraHTTPHeaders({ "X-Forwarded-For": uniqueClientIp() });
  await signInViaUi(page);
  await expect(page.getByRole("heading", { name: "Competitions" })).toBeVisible();
  await page.goto(`/admin/competitions/${competitionId}`);
}

async function seedActive(lanes: number[]) {
  const api = await adminApi(test.info().project.use.baseURL!);
  const competition = await createCompetition(api, { status: "active" });
  const entries = [];
  for (const lane of lanes) entries.push(await seedEntry(api, competition.id, lane));
  return { api, competition, entries };
}

test.describe("pairing and scorer device", () => {
  test("the admin creates a pairing link and the device shows its lane and score lane", async ({
    page,
    browser,
    baseURL,
  }) => {
    const { competition, entries } = await seedActive([1, 2, 3]);
    await signInAdmin(page, competition.id);

    const pairing = section(page, "Scorer pairing");
    await pairing.getByRole("combobox").selectOption({ label: "Lane 1" });
    await pairing.getByRole("button", { name: "Create pairing link" }).click();
    await expect(pairing.getByRole("img", { name: "QR code pairing lane 1" })).toBeVisible();
    const link = await pairing.getByLabel("Pairing link").inputValue();
    expect(link).toContain("/score#token=");

    const device = await browser.newContext({ baseURL: baseURL! });
    const scorer = await device.newPage();
    await scorer.goto(link);
    await expect(
      scorer.getByRole("heading", { name: `Scoring lane 2: ${entries[1].shooter.first_name} ${entries[1].shooter.last_name}` }),
    ).toBeVisible();
    await expect(scorer.getByText("This device is paired to lane 1.")).toBeVisible();
    await device.close();
  });

  test("the score lane wraps from the last manned lane to the first", async ({ browser, baseURL }) => {
    const { api, competition, entries } = await seedActive([1, 4, 7]);
    const pairing = await createPairing(api, competition.id, 7);
    const scorer = await openDevice(browser, baseURL!, pairing.token);
    await expect(
      scorer.getByRole("heading", { name: `Scoring lane 1: ${entries[0].shooter.first_name} ${entries[0].shooter.last_name}` }),
    ).toBeVisible();
    await expect(scorer.getByText("This device is paired to lane 7.")).toBeVisible();
    await scorer.context().close();
  });

  test("the scorer sees a running total and a mismatch message, then saves and locks", async ({
    page,
    browser,
    baseURL,
  }) => {
    const { api, competition, entries } = await seedActive([1, 2]);
    const pairing = await createPairing(api, competition.id, 1);
    const scorer = await openDevice(browser, baseURL!, pairing.token);
    await expect(scorer.getByRole("heading", { name: /Scoring lane 2/ })).toBeVisible();

    await scorer.getByRole("button", { name: "Increase 10", exact: true }).click();
    await scorer.getByRole("button", { name: "Increase 10", exact: true }).click();
    await expect(scorer.getByText("/ 12")).toBeVisible();
    await expect(scorer.locator(".total strong")).toHaveText("2");
    await expect(scorer.getByText("10 to go")).toBeVisible();
    await scorer.getByRole("button", { name: "Decrease 10", exact: true }).click();
    await expect(scorer.locator(".total strong")).toHaveText("1");

    await scorer.getByRole("button", { name: "Save", exact: true }).click();
    await expect(scorer.getByRole("alert")).toHaveText("Ring counts must add up to 12 shots (entered 1).");

    await scorer.getByLabel("10 count", { exact: true }).fill("8");
    await scorer.getByLabel("9 count", { exact: true }).fill("4");
    await expect(scorer.locator(".total strong")).toHaveText("12");
    await expect(scorer.getByText("✓ Shots match")).toBeVisible();
    await scorer.getByRole("button", { name: "Save", exact: true }).click();
    await expect(scorer.getByRole("status")).toHaveText("Saved");
    await scorer.getByRole("button", { name: "Lock stage", exact: true }).click();
    await expect(scorer.getByRole("status")).toHaveText("Stage locked");

    await signInAdmin(page, competition.id);
    const stage = row(page, entries[1].shooter.last_name).locator("td.stage").first();
    await expect(stage.getByText("Locked", { exact: true })).toBeVisible({ timeout: poll });
    await scorer.context().close();
  });

  test("a locked stage is read-only for the scorer and the admin sees who locked it", async ({
    page,
    browser,
    baseURL,
  }) => {
    const { api, competition, entries } = await seedActive([1, 2]);
    const pairing = await createPairing(api, competition.id, 1);
    const scorer = await openDevice(browser, baseURL!, pairing.token);
    await scorer.getByLabel("10 count", { exact: true }).fill("12");
    await scorer.getByRole("button", { name: "Save", exact: true }).click();
    await expect(scorer.getByRole("status")).toHaveText("Saved");
    await scorer.getByRole("button", { name: "Lock stage", exact: true }).click();

    await expect(scorer.getByText("🔒 Stage 1 is locked.")).toBeVisible();
    await expect(scorer.getByRole("button", { name: "Save", exact: true })).toHaveCount(0);
    await expect(scorer.getByRole("button", { name: "Lock stage", exact: true })).toHaveCount(0);
    await expect(scorer.getByLabel("10 count", { exact: true })).toHaveCount(0);

    await signInAdmin(page, competition.id);
    const badge = row(page, entries[1].shooter.last_name).getByText("Locked", { exact: true }).first();
    await expectVisible(badge, poll);
    await expect(badge).toHaveAttribute(
      "title",
      `Scored by lane 1 ${entries[0].shooter.first_name} ${entries[0].shooter.last_name}`,
    );
    await scorer.context().close();
  });

  test("revoking a pairing logs the device out on its next load", async ({ page, browser, baseURL }) => {
    const { api, competition } = await seedActive([1, 2]);
    const pairing = await createPairing(api, competition.id, 1);
    const scorer = await openDevice(browser, baseURL!, pairing.token);
    await expect(scorer.getByRole("heading", { name: /Scoring lane 2/ })).toBeVisible();

    await signInAdmin(page, competition.id);
    const pairings = section(page, "Scorer pairing");
    await expectVisible(pairings.getByRole("row").filter({ hasText: "paired" }).first(), poll);
    await pairings.getByRole("row").filter({ hasText: "paired" }).getByRole("button", { name: "Revoke" }).click();
    await expect(pairings.getByText("revoked", { exact: true })).toBeVisible();

    await scorer.goto("/score");
    await expect(scorer.getByText("Open a pairing link from the admin to score a lane.")).toBeVisible();
    await scorer.context().close();
  });

  test("with one manned lane the device needs admin scoring and the admin can score it", async ({
    page,
    browser,
    baseURL,
  }) => {
    const { api, competition, entries } = await seedActive([1]);
    const pairing = await createPairing(api, competition.id, 1);
    const scorer = await openDevice(browser, baseURL!, pairing.token);
    await expect(scorer.getByRole("alert")).toHaveText("admin scoring required");

    await signInAdmin(page, competition.id);
    const entryRow = row(page, entries[0].shooter.last_name);
    await entryRow.locator("td.stage").first().getByRole("button", { name: "Edit" }).click();
    await page.getByLabel("10", { exact: true }).fill("12");
    await expect(page.getByText("✓ Shots match")).toBeVisible();
    await page.getByRole("button", { name: "Save and lock" }).click();
    await expect(entryRow.locator("td.stage").first().getByText("Locked", { exact: true })).toBeVisible({
      timeout: poll,
    });
    await scorer.context().close();
  });

  test("a draft competition cannot be scored from a device", async ({ browser, baseURL }) => {
    const api = await adminApi(baseURL!);
    const competition = await createCompetition(api, { status: "draft" });
    await seedEntry(api, competition.id, 1);
    await seedEntry(api, competition.id, 2);
    const pairing = await createPairing(api, competition.id, 1);
    const scorer = await openDevice(browser, baseURL!, pairing.token);
    await expect(scorer.getByRole("alert")).toHaveText("conflict");
    await expect(scorer.getByRole("button", { name: "Save", exact: true })).toHaveCount(0);
    await scorer.context().close();
  });
});
