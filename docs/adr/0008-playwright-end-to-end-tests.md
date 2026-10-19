---
status: accepted
---

# Playwright for browser end-to-end tests

A date-validation bug in the competition form shipped because Svelte rewrote the `pattern` attribute at compile time. Vitest with jsdom does not enforce HTML constraint validation and never ran the compiled bundle in a real engine, so no existing test could see it. The SPA now has a small Playwright suite in `source/web/e2e` that drives the real Go binary, with the embedded SPA, in Chromium, Firefox and WebKit.

Tests run against a freshly started server with a temporary database. Playwright's `webServer` launches `go run ./cmd/server` by default; setting `E2E_BASE_URL` targets an already running instance instead, which is how `docker-compose.e2e.yml` runs the suite from the official Playwright image against the built MatchMark image. CI runs the same command on every pull request.

## Considered Options

- Cypress or WebdriverIO: rejected, Playwright covers all three engines (including WebKit for Safari) with one dependency and ships an official Docker image.
- Vite dev server with a mocked API: rejected, it would not exercise the embedded SPA, cookie session or real validation.
- Keep jsdom only: rejected, it cannot catch browser-engine behaviour.

## Consequences

Supersedes the Playwright rejection in ADR 0006; Vitest remains the runner for fast component-level behaviour tests. `npm run e2e` joins the required checks for changes that affect user flows. The suite stays a small smoke set; planned coverage is listed in the README. CI installs three browsers, which adds a few minutes per run.
