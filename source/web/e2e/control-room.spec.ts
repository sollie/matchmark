import type { Locator } from "@playwright/test";
import {
  createMatch,
  createPairing,
  deviceLock,
  expect,
  openMatch,
  section,
  seedEntry,
  setCounts,
  setStatus,
  test,
} from "./helpers";

function laneRow(room: Locator, lane: number): Locator {
  return room
    .locator("tbody tr")
    .filter({ has: room.page().locator("td:first-child", { hasText: new RegExp(`^${lane}$`) }) });
}

test.describe("control room", () => {
  test("shows one row per manned lane with scorer, status, pairing and best-effort connection", async ({
    adminPage: page,
    admin,
    course,
    baseURL,
    openDevice,
  }) => {
    const match = await createMatch(admin);
    const first = await seedEntry(admin, match.id, 1);
    const second = await seedEntry(admin, match.id, 2);
    await setStatus(admin, match.id, "active");
    const token = (await createPairing(admin, match.id, 1)).token;
    await openMatch(page, match.id);

    const room = section(page, "Control room");
    await expect(room.getByText("Current stage:")).toContainText("1");
    const lane1 = laneRow(room, 1);
    const lane2 = laneRow(room, 2);
    await expect(lane1.getByText("Paired", { exact: true })).toBeVisible();
    await expect(lane1.getByText("Not connected")).toBeVisible();
    await expect(lane2.getByText("Not paired")).toBeVisible();
    await expect(lane2.getByText("Open", { exact: true })).toBeVisible();
    await expect(room.getByText(/best-effort/)).toBeVisible();

    await openDevice(token);
    await expect(lane1.getByText("Connected", { exact: true })).toBeVisible();

    await deviceLock(baseURL!, token, 1, { "10": course.stageOneShots });
    await expect(lane2.getByText("Awaiting acceptance")).toBeVisible();
    await expect(lane2.getByText("Open", { exact: true })).toHaveCount(0);
  });

  test("accept all is enabled only when no lane is open and accepts every awaiting stage, then the next stage can start", async ({
    adminPage: page,
    admin,
    course,
    baseURL,
  }) => {
    const match = await createMatch(admin);
    await seedEntry(admin, match.id, 1);
    await seedEntry(admin, match.id, 2);
    await setStatus(admin, match.id, "active");
    const lane1 = (await createPairing(admin, match.id, 1)).token;
    const lane2 = (await createPairing(admin, match.id, 2)).token;
    await openMatch(page, match.id);
    const room = section(page, "Control room");
    const acceptAll = room.getByRole("button", { name: "Accept all" });
    const startNext = room.getByRole("button", { name: "Start next stage" });

    await expect(acceptAll).toBeDisabled();
    await expect(startNext).toBeDisabled();
    await deviceLock(baseURL!, lane1, 1, { "10": course.stageOneShots });
    await expect(room.getByText("Awaiting acceptance")).toBeVisible();
    await expect(acceptAll).toBeDisabled();
    await deviceLock(baseURL!, lane2, 1, { "10": course.stageOneShots });
    await expect(acceptAll).toBeEnabled();
    await expect(startNext).toBeDisabled();

    await acceptAll.click();
    await expect(room.getByText("Accepted", { exact: true })).toHaveCount(2);
    await expect(acceptAll).toBeDisabled();
    await expect(startNext).toBeEnabled();
    await startNext.click();
    await expect(room.getByText("Current stage:")).toContainText("2");
    await expect(startNext).toBeDisabled();
  });

  test("the admin accepts and unlocks single lanes without a device", async ({
    adminPage: page,
    admin,
    course,
    baseURL,
  }) => {
    const match = await createMatch(admin);
    const first = await seedEntry(admin, match.id, 1);
    const second = await seedEntry(admin, match.id, 2);
    await setStatus(admin, match.id, "active");
    const lane1 = (await createPairing(admin, match.id, 1)).token;
    await deviceLock(baseURL!, lane1, 1, { "10": course.stageOneShots });
    await setCounts(admin, match.id, first.entryId, 1, { "10": course.stageOneShots }, true);
    await openMatch(page, match.id);
    const room = section(page, "Control room");
    const awaiting = laneRow(room, 2);
    await expect(awaiting.getByText("Awaiting acceptance")).toBeVisible();

    await awaiting.getByRole("button", { name: "Control room lane 2: Accept" }).click();
    await expect(awaiting.getByText("Accepted", { exact: true })).toBeVisible();
    await expect(room.getByRole("button", { name: "Start next stage" })).toBeEnabled();

    await laneRow(room, 1).getByRole("button", { name: "Control room lane 1: Unlock" }).click();
    await expect(laneRow(room, 1).getByText("Open", { exact: true })).toBeVisible();
    await expect(room.getByRole("button", { name: "Start next stage" })).toBeDisabled();
  });
});
