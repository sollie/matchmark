import { randomBytes } from "node:crypto";
import { expect, request, type APIRequestContext, type Locator, type Page } from "@playwright/test";

export const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? "e2e-admin-password-1234";

export interface RingCounts {
  x: number;
  "10": number;
  "9": number;
  "8": number;
  "7": number;
  "0": number;
  miss: number;
}

export interface Visibility {
  show_names: boolean;
  show_clubs: boolean;
  show_countries: boolean;
  show_stage_scores: boolean;
  show_ring_counts: boolean;
}

export type CompetitionStatus = "draft" | "active" | "finished";

export function uniqueName(prefix: string): string {
  return `${prefix} ${randomBytes(4).toString("hex")}`;
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

export function ringCounts(partial: Partial<RingCounts> = {}): RingCounts {
  return { x: 0, "10": 0, "9": 0, "8": 0, "7": 0, "0": 0, miss: 0, ...partial };
}

async function checked<T>(response: Awaited<ReturnType<APIRequestContext["get"]>>, what: string): Promise<T> {
  if (!response.ok()) {
    throw new Error(`${what} failed: ${response.status()} ${await response.text()}`);
  }
  const text = await response.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

export async function newApi(baseURL: string, clientIp = uniqueClientIp()): Promise<APIRequestContext> {
  return request.newContext({ baseURL, extraHTTPHeaders: { "X-Forwarded-For": clientIp } });
}

export async function adminApi(baseURL: string): Promise<APIRequestContext> {
  const api = await newApi(baseURL);
  const response = await api.post("/api/admin/login", { data: { password: adminPassword } });
  await checked(response, "admin sign-in");
  return api;
}

export interface ShooterSeed {
  last_name?: string;
  first_name?: string;
  sex?: "M" | "F";
  country?: string;
  club?: string;
  waid?: string;
}

export async function createShooter(api: APIRequestContext, seed: ShooterSeed = {}) {
  const shooter = {
    last_name: uniqueName("Shooter"),
    first_name: "Test",
    sex: "M" as const,
    country: "NOR",
    club: "E2E Club",
    waid: "",
    ...seed,
  };
  const { id } = await checked<{ id: number }>(await api.post("/api/shooters", { data: shooter }), "create shooter");
  return { id, ...shooter };
}

export async function createCompetition(
  api: APIRequestContext,
  seed: { name?: string; date?: string; course?: string; status?: CompetitionStatus } = {},
) {
  const { status, ...rest } = seed;
  const competition = { name: uniqueName("Match"), date: todayIso(), course: "150", ...rest };
  const { id } = await checked<{ id: number }>(
    await api.post("/api/competitions", { data: competition }),
    "create competition",
  );
  if (status && status !== "draft") {
    await setStatus(api, id, "active");
    if (status === "finished") await setStatus(api, id, "finished");
  }
  return { id, ...competition };
}

export async function setStatus(api: APIRequestContext, competitionId: number, status: CompetitionStatus) {
  await checked(
    await api.post(`/api/competitions/${competitionId}/status`, { data: { status } }),
    `set status ${status}`,
  );
}

export async function setVisibility(api: APIRequestContext, competitionId: number, visibility: Partial<Visibility>) {
  const full: Visibility = {
    show_names: true,
    show_clubs: true,
    show_countries: true,
    show_stage_scores: true,
    show_ring_counts: true,
    ...visibility,
  };
  await checked(
    await api.put(`/api/competitions/${competitionId}/visibility`, { data: full }),
    "set visibility",
  );
}

export async function addEntry(
  api: APIRequestContext,
  competitionId: number,
  shooterId: number,
  lane: number,
  category = "Revolver",
) {
  const { id } = await checked<{ id: number }>(
    await api.post(`/api/competitions/${competitionId}/entries`, {
      data: { shooter_id: shooterId, category, lane },
    }),
    "add entry",
  );
  return id;
}

export async function seedEntry(api: APIRequestContext, competitionId: number, lane: number, category = "Revolver") {
  const shooter = await createShooter(api);
  const entryId = await addEntry(api, competitionId, shooter.id, lane, category);
  return { shooter, entryId, lane };
}

export async function setCounts(
  api: APIRequestContext,
  competitionId: number,
  entryId: number,
  stage: number,
  counts: Partial<RingCounts>,
  lock = false,
) {
  const base = `/api/competitions/${competitionId}/entries/${entryId}/stages/${stage}`;
  await checked(await api.put(base, { data: ringCounts(counts) }), "set ring counts");
  if (lock) await checked(await api.post(`${base}/lock`), "lock stage");
}

export async function unlockStage(api: APIRequestContext, competitionId: number, entryId: number, stage: number) {
  await checked(
    await api.post(`/api/competitions/${competitionId}/entries/${entryId}/stages/${stage}/unlock`),
    "unlock stage",
  );
}

export async function createPairing(api: APIRequestContext, competitionId: number, lane: number) {
  return checked<{ id: number; lane: number; token: string }>(
    await api.post(`/api/competitions/${competitionId}/pairings`, { data: { lane } }),
    "create pairing",
  );
}

export function pairingPath(token: string): string {
  return `/score#token=${encodeURIComponent(token)}`;
}

export async function signInViaUi(page: Page, password = adminPassword) {
  await page.goto("/admin");
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Sign in" }).click();
}

export function row(page: Page, text: string | RegExp): Locator {
  return page.getByRole("row").filter({ hasText: text });
}

export function section(page: Page, heading: string | RegExp): Locator {
  return page
    .locator("section")
    .filter({ has: page.getByRole("heading", { name: heading }) });
}

export async function expectVisible(locator: Locator, timeout = 15_000) {
  await expect(locator).toBeVisible({ timeout });
}
