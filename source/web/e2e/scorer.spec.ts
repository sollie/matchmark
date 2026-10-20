import type { APIRequestContext } from "@playwright/test";
import { createCompetition, createPairing, expect, openCompetition, row, section, seedEntry, test } from "./helpers";

const deviceTokenKey = "mm_device_token";

async function seedActive(api: APIRequestContext, lanes: number[]) {
  const competition = await createCompetition(api, { status: "active" });
  const entries = [];
  for (const lane of lanes) entries.push(await seedEntry(api, competition.id, lane));
  return { competition, entries };
}

test.describe("pairing and scorer device", () => {
  test("the admin creates a pairing link and the device shows its lane and score lane", async ({ adminPage: page, admin, openDevice }) => {
    const { competition, entries } = await seedActive(admin, [1, 2, 3]);
    await openCompetition(page, competition.id);

    const pairing = section(page, "Scorer pairing");
    await page.getByLabel("Pairing lane").selectOption({ label: "Lane 1" });
    await pairing.getByRole("button", { name: "Create pairing link" }).click();
    await expect(pairing.getByRole("img", { name: "QR code pairing lane 1" })).toBeVisible();
    const link = await pairing.getByLabel("Pairing link").inputValue();
    expect(link).toContain("/score#token=");

    const scorer = await openDevice(decodeURIComponent(new URL(link).hash.slice("#token=".length)));
    await expect(
      scorer.getByRole("heading", { name: `Scoring lane 2: ${entries[1].shooter.first_name} ${entries[1].shooter.last_name}` }),
    ).toBeVisible();
    await expect(scorer.getByText("This device is paired to lane 1.")).toBeVisible();
  });

  test("the score lane wraps from the last manned lane to the first", async ({ admin, openDevice }) => {
    const { competition, entries } = await seedActive(admin, [1, 4, 7]);
    const pairing = await createPairing(admin, competition.id, 7);
    const scorer = await openDevice(pairing.token);
    await expect(
      scorer.getByRole("heading", { name: `Scoring lane 1: ${entries[0].shooter.first_name} ${entries[0].shooter.last_name}` }),
    ).toBeVisible();
    await expect(scorer.getByText("This device is paired to lane 7.")).toBeVisible();
  });

  test("the scorer sees a running total and a mismatch message, then saves and locks", async ({ adminPage: page, admin, openDevice }) => {
    const { competition, entries } = await seedActive(admin, [1, 2]);
    const pairing = await createPairing(admin, competition.id, 1);
    const scorer = await openDevice(pairing.token);
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

    await openCompetition(page, competition.id);
    const stage = row(page, entries[1].shooter.last_name).locator("td.stage").first();
    await expect(stage.getByText("Locked", { exact: true })).toBeVisible();
  });

  test("a locked stage is read-only for the scorer and the admin sees who locked it", async ({ adminPage: page, admin, openDevice }) => {
    const { competition, entries } = await seedActive(admin, [1, 2]);
    const pairing = await createPairing(admin, competition.id, 1);
    const scorer = await openDevice(pairing.token);
    await scorer.getByLabel("10 count", { exact: true }).fill("12");
    await scorer.getByRole("button", { name: "Save", exact: true }).click();
    await expect(scorer.getByRole("status")).toHaveText("Saved");
    await scorer.getByRole("button", { name: "Lock stage", exact: true }).click();

    await expect(scorer.getByText("🔒 Stage 1 is locked.")).toBeVisible();
    await expect(scorer.getByRole("button", { name: "Save", exact: true })).toHaveCount(0);
    await expect(scorer.getByRole("button", { name: "Lock stage", exact: true })).toHaveCount(0);
    await expect(scorer.getByLabel("10 count", { exact: true })).toHaveCount(0);

    await openCompetition(page, competition.id);
    const badge = row(page, entries[1].shooter.last_name).getByText("Locked", { exact: true }).first();
    await expect(badge).toBeVisible();
    await expect(badge).toHaveAttribute(
      "title",
      `Scored by lane 1 ${entries[0].shooter.first_name} ${entries[0].shooter.last_name}`,
    );
  });

  test("revoking a pairing logs the device out on its next load", async ({ adminPage: page, admin, openDevice }) => {
    const { competition } = await seedActive(admin, [1, 2]);
    const pairing = await createPairing(admin, competition.id, 1);
    const scorer = await openDevice(pairing.token);
    await expect(scorer.getByRole("heading", { name: /Scoring lane 2/ })).toBeVisible();
    expect(await scorer.evaluate((key) => localStorage.getItem(key), deviceTokenKey)).toBe(pairing.token);

    await openCompetition(page, competition.id);
    const pairings = section(page, "Scorer pairing");
    await expect(pairings.getByRole("row").filter({ hasText: "paired" }).first()).toBeVisible();
    await pairings.getByRole("row").filter({ hasText: "paired" }).getByRole("button", { name: "Revoke" }).click();
    await expect(pairings.getByText("revoked", { exact: true })).toBeVisible();

    await scorer.reload();
    await expect(scorer.getByText("Open a pairing link from the admin to score a lane.")).toBeVisible();
    expect(await scorer.evaluate((key) => localStorage.getItem(key), deviceTokenKey)).toBeNull();
  });

  test("unpairing a device forgets the pairing and a reload stays unpaired", async ({ admin, openDevice }) => {
    const { competition } = await seedActive(admin, [1, 2]);
    const pairing = await createPairing(admin, competition.id, 1);
    const scorer = await openDevice(pairing.token);
    await expect(scorer.getByRole("heading", { name: /Scoring lane 2/ })).toBeVisible();

    await scorer.getByRole("button", { name: "Unpair device" }).click();
    await expect(scorer.getByText("Open a pairing link from the admin to score a lane.")).toBeVisible();
    await expect(scorer.getByRole("heading", { name: /Scoring lane/ })).toHaveCount(0);

    await scorer.reload();
    await expect(scorer.getByText("Open a pairing link from the admin to score a lane.")).toBeVisible();
    await expect(scorer.getByRole("heading", { name: /Scoring lane/ })).toHaveCount(0);
  });

  test("with one manned lane the device needs admin scoring and the admin can score it", async ({ adminPage: page, admin, openDevice }) => {
    const { competition, entries } = await seedActive(admin, [1]);
    const pairing = await createPairing(admin, competition.id, 1);
    const scorer = await openDevice(pairing.token);
    await expect(scorer.getByRole("alert")).toHaveText("admin scoring required");

    await openCompetition(page, competition.id);
    const entryRow = row(page, entries[0].shooter.last_name);
    await entryRow.getByRole("button", { name: "Stage 1: Edit" }).click();
    await page.getByLabel("10", { exact: true }).fill("12");
    await expect(page.getByText("✓ Shots match")).toBeVisible();
    await page.getByRole("button", { name: "Save and lock" }).click();
    await expect(entryRow.locator("td.stage").first().getByText("Locked", { exact: true })).toBeVisible();
  });

  test("a draft competition cannot be scored from a device", async ({ admin, openDevice }) => {
    const competition = await createCompetition(admin, { status: "draft" });
    await seedEntry(admin, competition.id, 1);
    await seedEntry(admin, competition.id, 2);
    const pairing = await createPairing(admin, competition.id, 1);
    const scorer = await openDevice(pairing.token);
    await expect(scorer.getByRole("alert")).toHaveText("conflict");
    await expect(scorer.getByRole("button", { name: "Save", exact: true })).toHaveCount(0);
  });
});
