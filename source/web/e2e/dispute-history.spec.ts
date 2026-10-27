import {
  createMatch,
  createPairing,
  deviceAction,
  deviceLock,
  expect,
  openMatch,
  row,
  section,
  seedEntry,
  setStatus,
  test,
} from "./helpers";

test.describe("dispute history", () => {
  test("lists stage events with device actors resolved, counts disputes per scorer and filters", async ({
    adminPage: page,
    admin,
    course,
    baseURL,
  }) => {
    const match = await createMatch(admin);
    const first = await seedEntry(admin, match.id, 1);
    const second = await seedEntry(admin, match.id, 2);
    await setStatus(admin, match.id, "active");
    const name = (e: typeof first) => `${e.shooter.first_name} ${e.shooter.last_name}`;
    const lane1 = (await createPairing(admin, match.id, 1)).token;
    const lane2 = (await createPairing(admin, match.id, 2)).token;

    await deviceLock(baseURL!, lane1, 1, { "10": course.stageOneShots });
    await deviceAction(baseURL!, lane2, 1, "dispute");
    await deviceLock(baseURL!, lane1, 1, { "10": course.stageOneShots });
    await deviceLock(baseURL!, lane2, 1, { "10": course.stageOneShots });
    await openMatch(page, match.id);

    const history = section(page, "Dispute history");
    const events = history.locator("table.events");
    await expect(events.locator("tbody tr")).toHaveCount(4);
    await expect(row(events, "Disputed")).toHaveCount(1);
    await expect(row(events, "Disputed")).toContainText(name(second));
    await expect(row(events, "Disputed")).toContainText(name(first));

    const scorers = history.locator("table.scorers");
    await expect(row(scorers, name(first))).toContainText(/2\s*1$/);
    await expect(row(scorers, name(second))).toContainText(/1\s*0$/);

    await history.getByLabel("Filter by scorer").selectOption(name(second));
    await expect(events.locator("tbody tr")).toHaveCount(1);
    await expect(row(events, "Disputed")).toHaveCount(0);

    await history.getByLabel("Filter by scorer").selectOption("");
    await history.getByLabel("Filter by stage").selectOption("1");
    await expect(events.locator("tbody tr")).toHaveCount(4);
    await history.getByLabel("Filter by shooter").selectOption(name(first));
    await expect(events.locator("tbody tr")).toHaveCount(1);
  });
});
