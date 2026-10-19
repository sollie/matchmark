---
status: accepted
---

# Vitest and Testing Library for frontend behaviour tests

The frontend originally had no test runner; changes were verified by type checking, building and manual use. As the admin and scorer screens grew, layout and behaviour refactors needed a safety net, so the SPA now uses Vitest with a jsdom environment and `@testing-library/svelte`. Tests render whole screens with `fetch` stubbed at the HTTP boundary and assert what the user sees and which requests are sent. All packages are dev-only and are not part of the built bundle or the embedded binary.

## Considered Options

- Keep manual verification only: rejected, UI refactors had no regression protection.
- Browser end-to-end tests (Playwright): rejected for now, heavy for a small single-club app. Layout is still checked by hand at phone, tablet and monitor widths.

## Consequences

`npm test` joins `npm run check`, `npm run build` and `npm audit` as a required post-change check and runs in CI. The suite stays small and behaviour-focused; it does not assert CSS or internal state.
