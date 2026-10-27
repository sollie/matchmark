import type { APIRequestContext } from "@playwright/test";
import { createMatch, createPairing, expect, openMatch, row, section, seedEntry, setCounts, setStatus, test } from "./helpers";

const deviceTokenKey = "mm_device_token";

async function seedActive(api: APIRequestContext, lanes: number[]) {
  const match = await createMatch(api);
  const entries = [];
  for (const lane of lanes) entries.push(await seedEntry(api, match.id, lane));
  await setStatus(api, match.id, "active");
  return { match, entries };
}

test.describe("pairing and scorer device", () => {
  test("the admin creates a pairing link and the device shows its lane and score lane", async ({ adminPage: page, admin, openDevice }) => {
    const { match, entries } = await seedActive(admin, [1, 2, 3]);
    await openMatch(page, match.id);

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

  test("the total and Save stay visible on a small phone without scrolling", async ({ admin, openDevice }) => {
    const { match } = await seedActive(admin, [1, 2]);
    const pairing = await createPairing(admin, match.id, 1);
    const scorer = await openDevice(pairing.token);
    await scorer.setViewportSize({ width: 393, height: 659 });
    await expect(scorer.getByRole("heading", { name: /Scoring lane 2/ })).toBeVisible();
    await expect(scorer.locator(".summary .total")).toBeInViewport();
    await expect(scorer.getByRole("button", { name: "Save" })).toBeInViewport();
  });

  test("the score lane wraps from the last manned lane to the first", async ({ admin, openDevice }) => {
    const { match, entries } = await seedActive(admin, [1, 4, 7]);
    const pairing = await createPairing(admin, match.id, 7);
    const scorer = await openDevice(pairing.token);
    await expect(
      scorer.getByRole("heading", { name: `Scoring lane 1: ${entries[0].shooter.first_name} ${entries[0].shooter.last_name}` }),
    ).toBeVisible();
    await expect(scorer.getByText("This device is paired to lane 7.")).toBeVisible();
  });

  test("the scorer sees a running total and a mismatch message, then saves and locks", async ({ adminPage: page, admin, openDevice }) => {
    const { match, entries } = await seedActive(admin, [1, 2]);
    const pairing = await createPairing(admin, match.id, 1);
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
    await expect(scorer.getByRole("button", { name: "Lock stage", exact: true })).toBeDisabled();
    await expect(scorer.locator("#save-status")).toHaveText("Unsaved changes: press Save, then Lock stage.");
    await scorer.getByRole("button", { name: "Save", exact: true }).click();
    await expect(scorer.locator("#save-status")).toHaveText("Saved ✓");
    await scorer.getByRole("button", { name: "Lock stage", exact: true }).click();
    await expect(scorer.getByRole("status")).toHaveText("Stage locked");

    await openMatch(page, match.id);
    const stage = row(page, entries[1].shooter.last_name).locator("td.stage").first();
    await expect(stage.getByText("Awaiting acceptance", { exact: true })).toBeVisible();
  });

  test("a locked stage is read-only for the scorer and the admin sees who locked it", async ({ adminPage: page, admin, openDevice }) => {
    const { match, entries } = await seedActive(admin, [1, 2]);
    const pairing = await createPairing(admin, match.id, 1);
    const scorer = await openDevice(pairing.token);
    await scorer.getByLabel("10 count", { exact: true }).fill("12");
    await scorer.getByRole("button", { name: "Save", exact: true }).click();
    await expect(scorer.locator("#save-status")).toHaveText("Saved ✓");
    await scorer.getByRole("button", { name: "Lock stage", exact: true }).click();

    await expect(scorer.getByText("🔒 Stage 1 is locked.")).toBeVisible();
    await expect(scorer.getByRole("button", { name: "Save", exact: true })).toHaveCount(0);
    await expect(scorer.getByRole("button", { name: "Lock stage", exact: true })).toHaveCount(0);
    await expect(scorer.getByLabel("10 count", { exact: true })).toHaveCount(0);

    await openMatch(page, match.id);
    const badge = row(section(page, "Entries and progress"), entries[1].shooter.last_name).getByText("Awaiting acceptance", { exact: true }).first();
    await expect(badge).toBeVisible();
    await expect(badge).toHaveAttribute(
      "title",
      `Scored by lane 1 ${entries[0].shooter.first_name} ${entries[0].shooter.last_name}`,
    );

    await page.getByRole("button", { name: `Lane ${entries[1].lane} stage 1: Accept` }).click();
    await expect(row(page, entries[1].shooter.last_name).getByText("Accepted", { exact: true }).first()).toBeVisible();
    await expect(badge).toHaveCount(0);
  });

  test("the admin entries refresh when a scorer locks a stage without reloading", async ({ adminPage: page, admin, openDevice }) => {
    const { match, entries } = await seedActive(admin, [1, 2]);
    const pairing = await createPairing(admin, match.id, 1);
    await openMatch(page, match.id);
    const lockedBadge = row(page, entries[1].shooter.last_name).getByText("Awaiting acceptance", { exact: true });
    await expect(lockedBadge).toHaveCount(0);

    const scorer = await openDevice(pairing.token);
    await scorer.getByLabel("10 count", { exact: true }).fill("12");
    await scorer.getByRole("button", { name: "Save", exact: true }).click();
    await expect(scorer.locator("#save-status")).toHaveText("Saved ✓");
    await scorer.getByRole("button", { name: "Lock stage", exact: true }).click();

    await expect(lockedBadge.first()).toBeVisible();
  });

  test("revoking a pairing returns the device to the pairing screen without a reload", async ({ adminPage: page, admin, openDevice }) => {
    const { match } = await seedActive(admin, [1, 2]);
    const pairing = await createPairing(admin, match.id, 1);
    const scorer = await openDevice(pairing.token);
    await expect(scorer.getByRole("heading", { name: /Scoring lane 2/ })).toBeVisible();
    expect(await scorer.evaluate((key) => localStorage.getItem(key), deviceTokenKey)).toBe(pairing.token);

    await openMatch(page, match.id);
    const pairings = section(page, "Scorer pairing");
    await expect(pairings.getByRole("row").filter({ hasText: "paired" }).first()).toBeVisible();
    await pairings.getByRole("row").filter({ hasText: "paired" }).getByRole("button", { name: "Revoke" }).click();
    await expect(pairings.getByText("revoked", { exact: true })).toBeVisible();

    await expect(scorer.getByText("Open a pairing link from the admin to score a lane.")).toBeVisible();
    expect(await scorer.evaluate((key) => localStorage.getItem(key), deviceTokenKey)).toBeNull();
  });

  test("an admin lock and unlock show on the scorer screen without a reload", async ({ admin, openDevice }) => {
    const { match, entries } = await seedActive(admin, [1, 2]);
    const pairing = await createPairing(admin, match.id, 1);
    const scorer = await openDevice(pairing.token);
    await expect(scorer.getByRole("heading", { name: /Scoring lane 2/ })).toBeVisible();

    const base = `/api/matches/${match.id}/entries/${entries[1].entryId}/stages/1`;
    await setCounts(admin, match.id, entries[1].entryId, 1, { "10": 12 }, true);
    await expect(scorer.getByText("🔒 Stage 1 is locked.")).toBeVisible();
    await expect(scorer.getByRole("button", { name: "Save", exact: true })).toHaveCount(0);

    await admin.post(`${base}/unlock`);
    await expect(scorer.getByRole("button", { name: "Save", exact: true })).toBeVisible();
  });

  test("unpairing a device forgets the pairing and a reload stays unpaired", async ({ admin, openDevice }) => {
    const { match } = await seedActive(admin, [1, 2]);
    const pairing = await createPairing(admin, match.id, 1);
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
    const { match, entries } = await seedActive(admin, [1]);
    const pairing = await createPairing(admin, match.id, 1);
    const scorer = await openDevice(pairing.token);
    await expect(scorer.getByRole("alert")).toHaveText("admin scoring required");

    await openMatch(page, match.id);
    const entryRow = row(page, entries[0].shooter.last_name);
    await page.getByRole("button", { name: `Lane ${entries[0].lane} stage 1: Edit` }).click();
    await page.getByLabel("10", { exact: true }).fill("12");
    await expect(page.getByText("✓ Shots match")).toBeVisible();
    await page.getByRole("button", { name: "Save and lock" }).click();
    await expect(entryRow.locator("td.stage").first().getByText("Accepted", { exact: true })).toBeVisible();
  });

  test("a device paired to a draft match waits and cannot score", async ({ admin, openDevice }) => {
    const match = await createMatch(admin, { status: "draft" });
    await seedEntry(admin, match.id, 1);
    await seedEntry(admin, match.id, 2);
    const pairing = await createPairing(admin, match.id, 1);
    const scorer = await openDevice(pairing.token);
    await expect(scorer.getByRole("heading", { name: "Waiting for the match to start" })).toBeVisible();
    await expect(scorer.getByRole("button", { name: "Save", exact: true })).toHaveCount(0);
  });

  test("the shooter reviews a locked result, disputes it, and the scorer re-scores with values prefilled", async ({ admin, openDevice }) => {
    const { match } = await seedActive(admin, [1, 2]);
    const lane1 = await openDevice((await createPairing(admin, match.id, 1)).token);
    const lane2 = await openDevice((await createPairing(admin, match.id, 2)).token);
    await expect(lane1.getByRole("heading", { name: /Scoring lane 2/ })).toBeVisible();
    await expect(lane2.getByRole("heading", { name: /Scoring lane 1/ })).toBeVisible();

    await lane1.getByLabel("10 count", { exact: true }).fill("8");
    await lane1.getByLabel("9 count", { exact: true }).fill("4");
    await lane1.getByRole("button", { name: "Save", exact: true }).click();
    await expect(lane1.locator("#save-status")).toHaveText("Saved ✓");
    await lane1.getByRole("button", { name: "Lock stage", exact: true }).click();
    await expect(lane1.getByText("Waiting for", { exact: false })).toBeVisible();

    await expect(lane2.getByLabel("10 count", { exact: true })).toBeVisible();
    await expect(lane2.getByRole("button", { name: "Review result" })).toHaveCount(0);

    await lane2.getByLabel("10 count", { exact: true }).fill("12");
    await lane2.getByRole("button", { name: "Save", exact: true }).click();
    await expect(lane2.locator("#save-status")).toHaveText("Saved ✓");
    await lane2.getByRole("button", { name: "Lock stage", exact: true }).click();
    await expect(lane2.getByText(/Your result for stage 1 awaits your acceptance/)).toBeVisible();
    await lane2.getByRole("button", { name: "Review result" }).click();
    await expect(lane2.getByText("12 hits, 0 X")).toBeVisible();

    await lane2.reload();
    await expect(lane2.getByText(/Your result for stage 1 awaits your acceptance/)).toBeVisible();
    await lane2.getByRole("button", { name: "Review result" }).click();
    await lane2.getByRole("button", { name: "Dispute" }).click();
    await lane2.getByRole("button", { name: "Cancel" }).click();
    await expect(lane2.getByRole("button", { name: "Accept" })).toBeVisible();
    await lane2.getByRole("button", { name: "Dispute" }).click();
    await lane2.getByRole("button", { name: "Confirm dispute" }).click();

    await expect(lane1.getByText(/Disputed: the shooter disputed stage 1/)).toBeVisible();
    await expect(lane1.getByLabel("10 count", { exact: true })).toHaveValue("8");
    await expect(lane1.getByLabel("9 count", { exact: true })).toHaveValue("4");
    await lane1.reload();
    await expect(lane1.getByText(/Disputed: the shooter disputed stage 1/)).toBeVisible();
    await expect(lane1.getByLabel("9 count", { exact: true })).toHaveValue("4");

    await lane1.getByRole("button", { name: "Lock stage", exact: true }).click();
    await expect(lane1.getByText(/Disputed: the shooter disputed/)).toHaveCount(0);
    await expect(lane1.getByText(/Your result for stage 1 awaits your acceptance/)).toBeVisible();
    await lane1.getByRole("button", { name: "Review result" }).click();
    await expect(lane1.getByText("12 hits, 0 X")).toBeVisible();
    await lane1.getByRole("button", { name: "Accept" }).click();
    await expect(lane1.getByRole("status")).toHaveText("Result accepted");
  });
});
