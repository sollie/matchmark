import { randomBytes } from "node:crypto";
import {
  request,
  test as base,
  expect,
  type APIRequestContext,
  type Locator,
  type Page,
} from "@playwright/test";

export const adminUsername = process.env.E2E_ADMIN_USERNAME ?? "e2e-admin";
export const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? "e2e-admin-password-1234";

interface RingCounts {
  x: number;
  "10": number;
  "9": number;
  "8": number;
  "7": number;
  "0": number;
  miss: number;
}

interface Visibility {
  show_names: boolean;
  show_clubs: boolean;
  show_countries: boolean;
  show_segment_scores: boolean;
  show_ring_counts: boolean;
}

type MatchStatus = "draft" | "active" | "finished";

export type Category = "Revolver" | "Pistol" | "Revolver OS" | "Pistol OS" | "Carbine";

export interface Course {
  key: string;
  label: string;
  segments: number;
  segmentOneShots: number;
  segmentShots: number[];
  categories: string[];
}

interface RulebookResponse {
  courses: {
    key: string;
    label: string;
    categories: string[];
    segments: { shots: number }[];
  }[];
}

let rulebook: Promise<Course[]> | undefined;

async function fetchCourses(api: APIRequestContext): Promise<Course[]> {
  const body = await checked<RulebookResponse>(await api.get("/api/rulebook"), "get rulebook");
  return body.courses.map((c) => ({
    key: c.key,
    label: c.label,
    categories: c.categories,
    segments: c.segments.length,
    segmentOneShots: c.segments[0].shots,
    segmentShots: c.segments.map((st) => st.shots),
  }));
}

export async function loadCourses(api: APIRequestContext): Promise<Course[]> {
  rulebook ??= fetchCourses(api);
  return rulebook;
}

export async function defaultCourse(api: APIRequestContext): Promise<Course> {
  return (await loadCourses(api))[0];
}

export function uniqueName(prefix: string): string {
  return `${prefix} ${randomBytes(4).toString("hex")}`;
}

export function uniqueLastName(prefix: string): string {
  return uniqueName(prefix).toUpperCase();
}

export function uniqueClientIp(): string {
  const [a, b] = randomBytes(2);
  return `198.18.${a}.${b}`;
}

export function todayIso(): string {
  const now = new Date();
  const month = String(now.getMonth() + 1).padStart(2, "0");
  const day = String(now.getDate()).padStart(2, "0");
  return `${now.getFullYear()}-${month}-${day}`;
}

function ringCounts(partial: Partial<RingCounts> = {}): RingCounts {
  return { x: 0, "10": 0, "9": 0, "8": 0, "7": 0, "0": 0, miss: 0, ...partial };
}

async function checked<T>(
  response: Awaited<ReturnType<APIRequestContext["get"]>>,
  what: string,
): Promise<T> {
  if (!response.ok()) {
    throw new Error(`${what} failed: ${response.status()} ${await response.text()}`);
  }
  const text = await response.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

export async function newApi(
  baseURL: string,
  clientIp = uniqueClientIp(),
): Promise<APIRequestContext> {
  return request.newContext({ baseURL, extraHTTPHeaders: { "X-Forwarded-For": clientIp } });
}

export async function adminApi(baseURL: string): Promise<APIRequestContext> {
  const api = await newApi(baseURL);
  const response = await api.post("/api/admin/login", {
    data: { username: adminUsername, password: adminPassword },
  });
  await checked(response, "admin sign-in");
  return api;
}

interface ShooterSeed {
  last_name?: string;
  first_name?: string;
  country?: string;
  club?: string;
  waid?: string;
}

export async function createShooter(api: APIRequestContext, seed: ShooterSeed = {}) {
  const shooter = {
    last_name: uniqueLastName("Shooter"),
    first_name: "Test",
    country: "NOR",
    club: "E2E Club",
    waid: "",
    ...seed,
  };
  const { id } = await checked<{ id: number }>(
    await api.post("/api/shooters", { data: shooter }),
    "create shooter",
  );
  return { id, ...shooter };
}

export async function createMatch(
  api: APIRequestContext,
  seed: { name?: string; date?: string; course?: string; status?: MatchStatus } = {},
) {
  const { status, ...rest } = seed;
  const course = (await defaultCourse(api)).key;
  const match = { name: uniqueName("Match"), date: todayIso(), course, ...rest };
  const { id } = await checked<{ id: number }>(
    await api.post("/api/matches", { data: match }),
    "create match",
  );
  if (status && status !== "draft") {
    await setStatus(api, id, "active");
    if (status === "finished") await setStatus(api, id, "finished");
  }
  return { id, ...match };
}

export async function setStatus(api: APIRequestContext, matchId: number, status: MatchStatus) {
  await checked(
    await api.post(`/api/matches/${matchId}/status`, { data: { status } }),
    `set status ${status}`,
  );
}

export async function setVisibility(
  api: APIRequestContext,
  matchId: number,
  visibility: Partial<Visibility>,
) {
  const full: Visibility = {
    show_names: true,
    show_clubs: true,
    show_countries: true,
    show_segment_scores: true,
    show_ring_counts: true,
    ...visibility,
  };
  await checked(
    await api.put(`/api/matches/${matchId}/visibility`, { data: full }),
    "set visibility",
  );
}

export async function addEntry(
  api: APIRequestContext,
  matchId: number,
  shooterId: number,
  lane: number,
  category: Category = "Revolver",
) {
  const { id } = await checked<{ id: number }>(
    await api.post(`/api/matches/${matchId}/entries`, {
      data: { shooter_id: shooterId, category, lane },
    }),
    "add entry",
  );
  return id;
}

export async function seedEntry(
  api: APIRequestContext,
  matchId: number,
  lane: number,
  category: Category = "Revolver",
) {
  const shooter = await createShooter(api);
  const entryId = await addEntry(api, matchId, shooter.id, lane, category);
  return { shooter, entryId, lane };
}

export async function setCounts(
  api: APIRequestContext,
  matchId: number,
  entryId: number,
  segment: number,
  counts: Partial<RingCounts>,
  lock = false,
) {
  const base = `/api/matches/${matchId}/entries/${entryId}/segments/${segment}`;
  await checked(await api.put(base, { data: ringCounts(counts) }), "set ring counts");
  if (lock) await checked(await api.post(`${base}/lock`), "lock segment");
}

export async function createPairing(api: APIRequestContext, matchId: number, lane: number) {
  return checked<{ id: number; lane: number; token: string }>(
    await api.post(`/api/matches/${matchId}/pairings`, { data: { lane } }),
    "create pairing",
  );
}

export async function deviceLock(
  baseURL: string,
  token: string,
  segment: number,
  counts: Partial<RingCounts>,
) {
  const api = await request.newContext({
    baseURL,
    extraHTTPHeaders: { Authorization: `Bearer ${token}` },
  });
  try {
    await checked(
      await api.put(`/api/scorer/segments/${segment}`, { data: ringCounts(counts) }),
      "device set counts",
    );
    await checked(await api.post(`/api/scorer/segments/${segment}/lock`), "device lock");
  } finally {
    await api.dispose();
  }
}

export async function deviceAction(
  baseURL: string,
  token: string,
  segment: number,
  action: "accept" | "dispute" | "lock",
) {
  const api = await request.newContext({
    baseURL,
    extraHTTPHeaders: { Authorization: `Bearer ${token}` },
  });
  try {
    await checked(await api.post(`/api/scorer/segments/${segment}/${action}`), `device ${action}`);
  } finally {
    await api.dispose();
  }
}

function pairingPath(token: string): string {
  return `/score#token=${encodeURIComponent(token)}`;
}

export async function signInViaUi(page: Page, password = adminPassword, username = adminUsername) {
  await page.goto("/admin");
  await page.getByLabel("Username").fill(username);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Sign in" }).click();
}

export async function openAccountMenu(page: Page) {
  const trigger = page.getByRole("button", { name: "Account menu" });
  if ((await trigger.getAttribute("aria-expanded")) !== "true") await trigger.click();
  await expect(trigger).toHaveAttribute("aria-expanded", "true");
}

export async function openAccountLink(page: Page, name: string) {
  await openAccountMenu(page);
  await page.getByRole("link", { name, exact: true }).click();
}

export async function signOutViaMenu(page: Page) {
  await openAccountMenu(page);
  await page.getByRole("button", { name: "Sign out" }).click();
}

export function row(page: Page | Locator, text: string | RegExp): Locator {
  return page.getByRole("row").filter({ hasText: text });
}

export function section(page: Page, heading: string | RegExp): Locator {
  return page.locator("section").filter({ has: page.getByRole("heading", { name: heading }) });
}

export async function openMatch(page: Page, id: number) {
  await page.goto(`/admin/matches/${id}`);
  await expect(section(page, "Entries and progress")).toBeVisible();
}

interface Fixtures {
  courses: Course[];
  course: Course;
  admin: APIRequestContext;
  adminPage: Page;
  openDevice: (token: string) => Promise<Page>;
}

export const test = base.extend<Fixtures>({
  page: async ({ page }, use) => {
    await page.setExtraHTTPHeaders({ "X-Forwarded-For": uniqueClientIp() });
    await use(page);
  },
  courses: async ({ admin }, use) => {
    await use(await loadCourses(admin));
  },
  course: async ({ admin }, use) => {
    await use(await defaultCourse(admin));
  },
  admin: async ({ baseURL }, use) => {
    const api = await adminApi(baseURL!);
    await use(api);
    await api.dispose();
  },
  adminPage: async ({ page }, use) => {
    await signInViaUi(page);
    await expect(page.getByRole("heading", { name: "Matches" })).toBeVisible();
    await use(page);
  },
  openDevice: async ({ browser, baseURL }, use) => {
    const contexts: { close: () => Promise<void> }[] = [];
    await use(async (token) => {
      const context = await browser.newContext({ baseURL });
      contexts.push(context);
      const device = await context.newPage();
      await device.goto(pairingPath(token));
      return device;
    });
    for (const context of contexts) await context.close();
  },
});

export { expect };
